# Filesystem Package

A high-performance, feature-rich filesystem abstraction layer for Go applications.

## Features

### Core Operations
- **File Operations**: Create, read, write, append, copy, move, and delete files
- **Directory Operations**: Create, list, and delete directories (including recursive deletion)
- **Metadata Access**: Get file size, metadata, and check file/directory existence
- **Path Utilities**: Absolute path resolution, file extension extraction, filename parsing

### Key Improvements

#### 1. **Enhanced Interface**
- Added `AppendFile()` for efficient file appending
- Added `FileSize()` for quick size queries
- Added `ListDirectory()` for directory traversal
- Added utility methods `IsFile()`, `IsDirectory()`, `GetAbsolutePath()`

#### 2. **Better Error Handling**
- Defined error constants for common error types
- Comprehensive path validation
- Prevention of same-file copy operations
- Graceful handling of non-existent files/directories

#### 3. **Automatic Directory Creation**
- `WriteFile()` now creates parent directories automatically
- `CreateFile()` creates parent directories if needed
- `CopyFile()` creates destination directories

#### 4. **Performance Optimizations**
- Uses `os.ReadDir()` for better directory listing performance
- Proper file handle management with deferred cleanup
- Efficient buffer management in copy operations

#### 5. **Safety Features**
- Path validation prevents empty paths
- Same-file copy detection
- Proper resource cleanup with deferred functions
- Thread-safe operations

## Usage Examples

### Basic File Operations

```go
fs := NewLocalFileSystem()

// Write content to a file (creates directories if needed)
content := []byte("Hello, World!")
err := fs.WriteFile("/path/to/file.txt", content)

// Read file content
data, err := fs.ReadFile("/path/to/file.txt")

// Append to existing file
err = fs.AppendFile("/path/to/file.txt", []byte(" More content"))

// Get file size
size, err := fs.FileSize("/path/to/file.txt")
```

### Directory Operations

```go
// Create directory with parents
err := fs.CreateDirectory("/path/to/nested/directory")

// List directory contents
files, err := fs.ListDirectory("/path/to/directory")
for _, file := range files {
    fmt.Printf("File: %s, Size: %d\n", file.Name(), file.Size())
}

// Check if path is file or directory
isFile, err := fs.IsFile("/path/to/item")
isDir, err := fs.IsDirectory("/path/to/item")
```

### Advanced Operations

```go
// Copy files with automatic directory creation
err := fs.CopyFile("/source/file.txt", "/destination/file.txt")

// Move files
err := fs.MoveFile("/old/location.txt", "/new/location.txt")

// Recursive directory copying
err := CopyDirectory(fs, "/source/dir", "/destination/dir")
```

### Utility Functions

```go
// Path utilities
ext := GetFileExtension("/path/to/file.txt")      // ".txt"
name := GetFileName("/path/to/file.txt")          // "file.txt"
dir := GetDirectoryName("/path/to/file.txt")      // "/path/to"

// Get absolute path
absPath, err := fs.GetAbsolutePath("./relative/path")
```

## Error Handling

The package provides predefined error constants for better error handling:

```go
var (
    ErrFileNotFound      = fmt.Errorf("filesystem: file not found")
    ErrDirectoryNotFound = fmt.Errorf("filesystem: directory not found")
    ErrFileAlreadyExists = fmt.Errorf("filesystem: file already exists")
    ErrInvalidPath       = fmt.Errorf("filesystem: invalid path")
)
```

## Thread Safety

All operations are thread-safe and can be used concurrently. The implementation uses the underlying OS filesystem operations which handle concurrent access appropriately.

## Performance Characteristics

- **Zero-allocation path operations** where possible
- **Efficient directory traversal** using `os.ReadDir()`
- **Proper resource management** with deferred cleanup
- **Minimal system calls** through existence checks before operations
- **Bulk operations support** for directory copying

## Testing

The package includes comprehensive tests covering all functionality. Run tests with:

```bash
go test ./filesystem/ -v
```

## Integration

This filesystem package is designed to integrate seamlessly with the high-performance gravel HTTP server framework, providing efficient file operations for static file serving, template loading, and configuration management.
