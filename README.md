# Kranky Bear Tailer

A cross-platform GUI log tail application similar to BareTail, written in Go using the Fyne GUI library.

## Features

- 🚀 **Real-time File Tailing**: Follow log files with live updates
- 📑 **Multiple Tabs**: Open and monitor multiple log files simultaneously
- 🎨 **Keyword Highlighting**: Highlight lines containing user-specified keywords
- 🔍 **Case-Insensitive Search**: Flexible keyword matching
- 💻 **Cross-Platform**: Compile for Windows, Linux, and macOS
- 🎯 **File Rotation Support**: Automatically handles log file rotation

## Installation

### Prerequisites

- Go 1.21 or later
- System dependencies for Fyne (see below)

#### macOS
```bash
# Install via Homebrew
brew install go
```

#### Linux (Ubuntu/Debian)
```bash
# System dependencies for Fyne on Linux
sudo apt-get install libgl1-mesa-dev libx11-dev libxcursor-dev libxinerama-dev libxi-dev libxrandr-dev libxss-dev libxxf86vm-dev

# Install Go
sudo apt-get install golang-go
```

#### Windows
- Download and install Go from https://golang.org/dl/
- System dependencies are included with Go on Windows

## Building

### Local Build

Build for your current platform:
```bash
go build -ldflags="-s -w" -trimpath -o bin/tailer
```

### Cross-Compilation

**Note:** Due to Fyne's CGO requirements and native dependencies (OpenGL, GLFW), cross-compilation is limited:
- ✅ Windows → Linux: Works
- ✅ Linux → Windows: Works
- ❌ macOS → Windows/Linux: Limited (some dependencies fail)
- ✅ macOS: Works (build on macOS for macOS)
- ⚠️  Cross-compile macOS binary from Linux/Windows: Not recommended

For best results, build each platform on that platform, or use Docker/CI.

Build for all platforms:
```bash
# Build for Windows (best on Windows)
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o bin/tailer-windows.exe

# Build for Linux (best on Linux)
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o bin/tailer-linux

# Build for macOS (Intel) - must be on macOS
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o bin/tailer-macos-amd64

# Build for macOS (Apple Silicon) - must be on macOS
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o bin/tailer-macos-arm64
```

### Build All Platforms at Once

You can create a simple compile script:

**compile-mac.sh** (for macOS):
```bash
#!/bin/bash

mkdir -p bin

echo "Building for macOS (Intel)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o bin/tailer-macos-amd64

echo "Building for macOS (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o bin/tailer-macos-arm64

echo "Build complete!"
```

**compile-linux.sh** (for Linux):
```bash
#!/bin/bash

mkdir -p bin

echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o bin/tailer-linux

echo "Build complete!"
```

**compile-windows.ps1** (for Windows PowerShell):
```powershell
Write-Host "Building for Windows..."
$env:GOOS="windows"
$env:GOARCH="amd64"
go build -ldflags="-s -w" -trimpath -o bin/tailer-windows.exe

# Optional: package installer with Inno Setup
# .\compile-windows.ps1 -Windows -Package
```

## Usage

1. **Launch the application**: Run the compiled binary for your platform
2. **Open a log file**: Click `File` → `Open File...` or use `Cmd+O` (Mac) / `Ctrl+O` (Windows/Linux)
3. **Add keywords**: Enter a keyword in the input field at the bottom of the tab and click "Add"
4. **Remove keywords**: Enter the keyword and click "Remove"
5. **Clear all keywords**: Click "Clear All"
6. **Open multiple files**: Use tabs to switch between different log files

### Keyboard Shortcuts

- `Cmd+O` (Mac) / `Ctrl+O` (Windows/Linux): Open file
- `Cmd+Q` (Mac) / `Ctrl+Q` (Windows/Linux): Quit application

## Features in Detail

### File Tailing
The application automatically tails log files and displays new lines as they appear. It starts reading from the end of the file, so you only see new log entries.

### Keyword Highlighting
- Enter keywords to highlight lines containing those keywords
- Keywords are case-insensitive
- Lines matching keywords are highlighted in the log view
- Add/remove keywords without stopping the tail process

### Tab Management
- Each opened file has its own tab
- Tabs are named after the file basename
- Click tabs to switch between files
- Multiple files can be tailed simultaneously

### File Rotation Support
The application automatically detects log file rotation and reopens the file to continue tailing from the new file.

## Development

### Project Structure
```
KrankyBearTailer/
├── main.go              # Main application code
├── go.mod               # Go module file
├── go.sum               # Dependency checksums
├── README.md            # This file
└── bin/                 # Compiled binaries (created on build)
```

### Dependencies

- **Fyne v2.4.5**: Cross-platform GUI library for Go
  - Handles windowing, widgets, and UI rendering
  - Abstracts platform-specific functionality

### Running from Source

```bash
# Get dependencies
go mod download

# Run the application
go run main.go
```

### Testing

Test the application with a sample log file:

```bash
# Create a test log file
echo "Starting application..." > test.log

# Run the tailer
go run main.go

# In another terminal, add lines to test.log
echo "Application running..." >> test.log
echo "ERROR: Something went wrong" >> test.log
echo "Application stopped." >> test.log
```

## Troubleshooting

### Issue: Binary is too large
Use the `-ldflags="-s -w"` flags to strip debug symbols and reduce binary size.

### Issue: Missing system libraries on Linux
Make sure you have installed all required system dependencies:
```bash
sudo apt-get install libgl1-mesa-dev libx11-dev libxcursor-dev libxinerama-dev libxi-dev libxrandr-dev libxss-dev libxxf86vm-dev
```

### Issue: Can't open file
- Make sure the file path is correct
- Check file permissions
- On some systems, you may need to run with appropriate permissions

## License

This project is provided as-is for educational and personal use.

## Contributing

Feel free to fork, modify, and use this project as you see fit.

