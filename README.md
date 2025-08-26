# Miss Minutes V2 iCal Server

A simple, single-file iCal server written in Go that follows the KISS (Keep It Simple, Stupid) philosophy.

## Features

- 📅 **Simple iCal server** - Store and serve calendar files
- 🔒 **Basic authentication** - HTTP Basic Auth for write operations
- 🌐 **Web interface** - Clean HTML interface for easy interaction
- 📁 **File-based storage** - No database required, just files on disk
- 🚀 **Single binary** - Everything in one Go file

## Quick Start

1. **Clone and navigate to the directory**
   ```bash
   git clone https://github.com/tellmeY18/missing-minutes.git
   cd missing-minutes
   ```

2. **Run the server**
   ```bash
   go run main.go
   ```

3. **Open your browser**
   Navigate to `http://localhost:8080` to use the web interface.

## First Run Setup

When you first run the server, it will create a default `users.json` file:

```json
{
  "user1": "changeme",
  "user2": "pleasereset"
}
```

**⚠️ Important**: Edit this file with real usernames and passwords before using the server!

## API Endpoints

| Method | Endpoint | Description | Authentication |
|--------|----------|-------------|----------------|
| `GET` | `/` | Web interface | None |
| `GET` | `/{username}/{calendar}.ics` | Read calendar | None |
| `PUT` | `/{username}/{calendar}.ics` | Create/update calendar | HTTP Basic Auth |

## Directory Structure

```
GO/
├── main.go           # The entire server
├── index.html        # Web interface
├── users.json        # User credentials (created on first run)
└── calendars/        # Calendar storage (created automatically)
    └── {username}/
        └── {calendar}.ics
```

## Usage Examples

### Web Interface (Recommended)
1. Go to `http://localhost:8080`
2. Use the forms to fetch or create calendars
3. Calendar names automatically get `.ics` extension if not provided

### Command Line (curl)

**Create/Update a calendar:**
```bash
curl -X PUT \
  --user "john:password123" \
  --header "Content-Type: text/calendar" \
  --data "@my-calendar.ics" \
  http://localhost:8080/john/work.ics
```

**Read a calendar:**
```bash
curl http://localhost:8080/john/work.ics
```

## Security Notes

- 🔓 **Read operations** are public (no authentication)
- 🔒 **Write operations** require HTTP Basic Authentication
- 👤 **Users can only edit their own calendars**
- 📁 **Files are stored with safe permissions** (0644)

## Configuration

Edit these constants in `main.go` if needed:

```go
const (
    dataDir    = "calendars"  // Where calendar files are stored
    serverPort = "8080"       // Server port
    userFile   = "users.json" // User credentials file
)
```

## Sample iCal Data

Here's a minimal example of iCal data you can use for testing:

```
BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Example Corp//Example Calendar//EN
BEGIN:VEVENT
UID:example-event-1@example.com
DTSTAMP:20230826T120000Z
DTSTART:20230827T090000Z
DTEND:20230827T100000Z
SUMMARY:Team Meeting
DESCRIPTION:Weekly team sync-up meeting
END:VEVENT
END:VCALENDAR
```

## Development

The entire server is contained in a single `main.go` file for maximum simplicity. Key components:

- **User management**: JSON-based user storage
- **Authentication**: HTTP Basic Auth middleware
- **File handling**: Direct filesystem operations
- **Web server**: Standard Go `net/http` package

## Troubleshooting

**Server won't start:**
- Check if port 8080 is already in use
- Ensure you have write permissions in the current directory

**Authentication errors:**
- Verify credentials in `users.json`
- Check that username in URL matches authenticated user

**File not found errors:**
- Calendar files are case-sensitive
- Ensure the calendar was created first using PUT

## License

This project follows the KISS philosophy - it's meant to be simple, understandable, and easily modifiable for your needs.
