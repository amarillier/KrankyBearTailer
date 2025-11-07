# Quick Start Guide

## Running the Application

### For Current Platform (macOS)

```bash
go run main.go
```

### Building Binary

```bash
go build -ldflags="-s -w" -trimpath -o bin/tailer
./bin/tailer
```

## Cross-Compilation

### Build All Platforms

Run the platform compile script:

```bash
./compile-mac.sh (on macOS)
```

This will create binaries for:
- Windows: `bin/tailer-windows.exe`
- Linux: `bin/tailer-linux`
- macOS (Intel): `bin/tailer-macos-amd64`
- macOS (Apple Silicon): `bin/tailer-macos-arm64`

### Individual Platform Builds

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H windowsgui" -trimpath -o bin/tailer-windows.exe

# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o bin/tailer-linux

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o bin/tailer-macos-amd64

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o bin/tailer-macos-arm64
```

## Testing the Application

1. Create a test log file:
```bash
echo "Starting application..." > test.log
```

2. Run the tailer:
```bash
go run main.go
```

3. In File menu, click "Open File..." and select `test.log`

4. In another terminal, add lines to test.log:
```bash
echo "INFO: Application running" >> test.log
echo "ERROR: Something went wrong!" >> test.log
echo "WARNING: This is a warning" >> test.log
echo "INFO: Application stopped" >> test.log
```

5. Add keywords like "ERROR" or "WARNING" in the keyword field and click "Add" to highlight matching lines

## Features

- **Real-time tailing**: New log lines appear automatically
- **Multiple tabs**: Open multiple log files simultaneously
- **Keyword highlighting**: Highlight lines containing specific keywords
- **Case-insensitive**: Keyword matching is not case-sensitive
- **File rotation support**: Automatically handles log file rotation

## Tips

- Use the Add button to highlight keywords
- Use the Remove button to stop highlighting a keyword
- Use Clear All to remove all keywords
- Each tab manages its own set of keywords
- Scroll to bottom automatically follows new lines

## "Now this is not even the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
