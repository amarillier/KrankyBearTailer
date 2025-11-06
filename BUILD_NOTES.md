# Build Notes

## Cross-Compilation Considerations

Fyne applications use CGO and have native dependencies (OpenGL, GLFW), which limits cross-compilation.

### What Works

✅ **Windows → Linux**: Works
- Building from Windows to Linux generally works
- Binary: `tailer-linux`

✅ **Linux → Windows**: Works  
- Building from Linux to Windows generally works
- Binary: `tailer-windows.exe`

✅ **Native builds**: Always work
- Build on macOS for macOS
- Build on Windows for Windows
- Build on Linux for Linux

### What Doesn't Work Well

❌ **macOS → Windows/Linux**: Limited/Fails
- OpenGL dependencies cause issues
- GLFW bindings are complex
- Error: "build constraints exclude all Go files"

❌ **Cross-compile macOS from non-macOS**: Not recommended
- Requires macOS SDK and tools
- Docker is needed for reliable results

## Recommended Build Strategy

### Option 1: Build on Target Platform (Best)

Each developer builds for their platform:
```bash
# On macOS
go build -ldflags="-s -w" -trimpath -o bin/tailer-macos-amd64

# On Windows
go build -ldflags="-s -w -H windowsgui" -trimpath -o bin/tailer-windows.exe

# On Linux
go build -ldflags="-s -w" -trimpath -o bin/tailer-linux
```

### Option 2: Use Sync Scripts

Use the provided sync scripts to build on remote platforms:
```bash
# From macOS to Windows
./sync2windows.sh

# From macOS to Ubuntu 18
./sync2ubuntu18.sh
```

These scripts:
1. Sync your source code to the target machine
2. Build the binary there
3. Sync it back

### Option 3: Use fyne-cross with Docker

Install fyne-cross:
```bash
go install github.com/fyne-io/fyne-cross@latest
```

Build for different platforms:
```bash
fyne-cross windows
fyne-cross linux
fyne-cross darwin  # (from macOS)
```

This uses Docker containers to handle dependencies correctly.

## Current Build Behavior

Running `./compile-mac.sh` on macOS will:
- ✅ Successfully build Windows binary
- ✅ Successfully build Linux binary  
- ⚠️  Fail or produce incomplete macOS binaries for cross-compile

**Solution**: For macOS builds, build natively on macOS:
```bash
# On macOS
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/tailer-macos-amd64
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/tailer-macos-arm64
```

## CI/CD Recommendation

For automated builds, use a CI service that supports multiple platforms:
- GitHub Actions with matrix strategy
- Multiple Docker containers (one per platform)
- Build each platform in its native environment

Example GitHub Actions matrix:
```yaml
strategy:
  matrix:
    os: [windows, linux, macos]
    arch: [amd64, arm64]
```

