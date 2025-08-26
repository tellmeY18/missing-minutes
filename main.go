// main.go
// A simple, single-file iCal server in Go.
//
// This server follows the KISS (Keep It Simple, Stupid) philosophy.
//
// ## Endpoints:
// - GET  /{username}/{calendar}.ics : Public, read-only access to a calendar.
// - PUT  /{username}/{calendar}.ics : Authenticated endpoint to create or update a calendar.
//
// ## Authentication:
// - Uses HTTP Basic Authentication for PUT requests.
// - Users and passwords are now loaded from a `users.json` file.
//
// ## Storage:
// - iCal files are stored directly on the filesystem in a 'calendars' directory.
// - The structure is: ./calendars/{username}/{calendar}.ics
//
// ## How to Run:
// 1. Save this code as `main.go`.
// 2. Create a file named `users.json` in the same directory.
//    Example `users.json` content:
//    {
//      "john": "password123",
//      "jane": "anotherpassword"
//    }
// 3. Create a directory named `calendars`.
// 4. Run the server: `go run main.go`
// 5. The server will start on `http://localhost:8080`.
//
// ## Example Usage (with curl):
//
// 1. Create/Update a calendar for user 'john':
//    curl -X PUT --user "john:password123" --header "Content-Type: text/calendar" --data "@path/to/your/local/event.ics" http://localhost:8080/john/work.ics
//
// 2. Read the calendar (no authentication needed):
//    curl http://localhost:8080/john/work.ics
//
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// users will be populated from the users.json file.
var users map[string]string

const (
	// dataDir is the directory where all calendar files will be stored.
	dataDir = "calendars"
	// serverPort is the port the web server will listen on.
	serverPort = "8080"
	// userFile is the name of the file containing user credentials.
	userFile = "users.json"
)

// loadUsers reads the specified file and unmarshals the JSON content
// into the global 'users' map.
func loadUsers(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("could not read user file '%s': %w", file, err)
	}

	// Unmarshal the JSON data into the users map.
	err = json.Unmarshal(data, &users)
	if err != nil {
		return fmt.Errorf("could not parse user file '%s' as JSON: %w", file, err)
	}

	log.Printf("Successfully loaded %d users from %s", len(users), file)
	return nil
}

func main() {
	// Attempt to load users from the JSON file.
	if err := loadUsers(userFile); err != nil {
		// If the file doesn't exist, create a default one and exit with instructions.
		if os.IsNotExist(err) {
			log.Printf("User file '%s' not found.", userFile)
			defaultUsers := map[string]string{
				"user1": "changeme",
				"user2": "pleasereset",
			}
			defaultData, _ := json.MarshalIndent(defaultUsers, "", "  ")
			if writeErr := os.WriteFile(userFile, defaultData, 0644); writeErr != nil {
				log.Fatalf("Could not write default user file: %v", writeErr)
			}
			log.Fatalf("A default '%s' has been created. Please edit it with real credentials and restart the server.", userFile)
		}
		// For any other error (e.g., bad JSON), exit fatally.
		log.Fatalf("Failed to load users: %v", err)
	}

	// Ensure the main data directory exists before starting.
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory '%s': %v", dataDir, err)
	}

	// Register the root handler to serve index.html
	http.HandleFunc("/", rootHandler)

	// Start the server.
	fmt.Printf(" KIS iCal Server starting on http://localhost:%s\n", serverPort)
	fmt.Printf(" Storing calendar files in ./%s/\n", dataDir)
	log.Fatal(http.ListenAndServe(":"+serverPort, nil))
}

// rootHandler serves the index.html file for the root path, otherwise delegates
// to calendarHandler for calendar-related requests.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	// If the request is for the root path, serve index.html
	if r.URL.Path == "/" {
		http.ServeFile(w, r, "index.html")
		return
	}

	// Otherwise, delegate to the calendar handler
	calendarHandler(w, r)
}

// calendarHandler is the main router. It inspects the request method and URL
// and delegates to the appropriate handler function.
func calendarHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetCalendar(w, r)
	case http.MethodPut:
		// Wrap the PUT handler with our authentication middleware.
		authMiddleware(handlePutCalendar)(w, r)
	default:
		// If the method is not GET or PUT, it's not allowed.
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetCalendar serves a calendar file to the client.
// This is a public endpoint and requires no authentication.
func handleGetCalendar(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Basic validation: ensure the path looks like a calendar file request.
	if !strings.HasSuffix(path, ".ics") {
		http.NotFound(w, r)
		return
	}

	// Construct the full file path and clean it to prevent directory traversal attacks.
	filePath := filepath.Join(dataDir, filepath.Clean(path))

	// Check if the file exists.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// Set the correct Content-Type header for iCalendar files.
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	http.ServeFile(w, r, filePath)
}

// handlePutCalendar creates or updates a calendar file.
// This function assumes authentication has already been handled by middleware.
func handlePutCalendar(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Basic validation: ensure the path looks like a calendar file request.
	if !strings.HasSuffix(path, ".ics") {
		http.Error(w, "Invalid path. Must end with .ics", http.StatusBadRequest)
		return
	}

	// Extract username from the path to verify ownership.
	// Path format: /{username}/{calendar}.ics
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid path format. Expected /{username}/{calendar}.ics", http.StatusBadRequest)
		return
	}

	// The username from the URL must match the authenticated user.
	// The authenticated user's name is passed via the request context from the middleware.
	authUser, ok := r.Context().Value("user").(string)
	if !ok || authUser != parts[0] {
		http.Error(w, "Forbidden. You can only edit your own calendars.", http.StatusForbidden)
		return
	}

	// Construct the full file path.
	filePath := filepath.Join(dataDir, filepath.Clean(path))
	dir := filepath.Dir(filePath)

	// Create the user's directory if it doesn't exist.
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Error creating directory %s: %v", dir, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Read the iCal data from the request body.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Write the data to the file, creating it or overwriting it.
	err = os.WriteFile(filePath, body, 0644)
	if err != nil {
		log.Printf("Error writing file %s: %v", filePath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("Updated calendar: %s", filePath)
	w.WriteHeader(http.StatusNoContent) // Success, no content to return.
}

// authMiddleware is a simple middleware to handle HTTP Basic Authentication.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the username and password from the Authorization header.
		username, password, ok := r.BasicAuth()

		// If credentials are not provided or are malformed, request them.
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if the user exists and the password is correct.
		expectedPassword, userExists := users[username]
		if !userExists || expectedPassword != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Authentication successful.
		// Add the username to the request context so the next handler knows who is logged in.
		ctx := r.Context()
		ctx = context.WithValue(ctx, "user", username)

		// Call the next handler in the chain with the modified request.
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
