package filesystem

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// Error constants for better error handling
var (
	ErrFileNotFound      = fmt.Errorf("filesystem: file not found")
	ErrDirectoryNotFound = fmt.Errorf("filesystem: directory not found")
	ErrFileAlreadyExists = fmt.Errorf("filesystem: file already exists")
	ErrInvalidPath       = fmt.Errorf("filesystem: invalid path")
)

type Filesystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, content []byte) error
	AppendFile(path string, content []byte) error

	CreateFile(path string) error
	DeleteFile(path string) error
	MoveFile(source, destination string) error
	CopyFile(source, destination string) error

	FileExists(path string) (bool, error)
	FileSize(path string) (int64, error)
	FileMetaData(path string) (os.FileInfo, error)

	DirectoryExists(path string) (bool, error)
	DeleteDirectory(path string) error
	CreateDirectory(path string) error
	ListDirectory(path string) ([]os.FileInfo, error)

	// Utility methods
	IsFile(path string) (bool, error)
	IsDirectory(path string) (bool, error)
	GetAbsolutePath(path string) (string, error)
}

type localFileSystem struct {
}

// CopyFile implements Filesystem.
func (filesystem *localFileSystem) CopyFile(source string, destination string) error {
	// Validate paths
	if source == "" || destination == "" {
		return ErrInvalidPath
	}

	// Check if source and destination are the same
	absSource, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	absDestination, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	if absSource == absDestination {
		return fmt.Errorf("filesystem: source and destination are the same file")
	}

	sourceFileExists, err := filesystem.FileExists(source)
	if err != nil {
		return err
	}
	if !sourceFileExists {
		return fmt.Errorf("filesystem: source file %s does not exist", source)
	}

	destinationFileExists, err := filesystem.FileExists(destination)
	if err != nil {
		return err
	}
	if destinationFileExists {
		return fmt.Errorf("filesystem: destination file %s already exists", destination)
	}

	// Create destination directory if it doesn't exist
	dir := filepath.Dir(destination)
	if err := filesystem.CreateDirectory(dir); err != nil {
		return err
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := sourceFile.Close(); closeErr != nil {
			slog.Error("closing source file error", "error", closeErr)
		}
	}()

	destinationFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := destinationFile.Close(); closeErr != nil {
			slog.Error("closing file error", "error", closeErr)
		}
	}()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	if err := destinationFile.Sync(); err != nil {
		return err
	}

	return nil
}

// AppendFile implements Filesystem.
func (filesystem *localFileSystem) AppendFile(path string, content []byte) error {
	exists, err := filesystem.FileExists(path)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("filesystem: file %s does not exist", path)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			slog.Error("closing file error", "error", closeErr)
		}
	}()

	_, err = file.Write(content)
	return err
}

func (filesystem *localFileSystem) CreateDirectory(path string) error {
	exists, err := filesystem.DirectoryExists(path)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if err := os.MkdirAll(path, 0770); err != nil {
		return err
	}

	return nil
}

func (filesystem *localFileSystem) CreateFile(path string) error {
	exists, err := filesystem.FileExists(path)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := filesystem.CreateDirectory(dir); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			slog.Error("closing created file error", "error", closeErr)
		}
	}()

	return nil
}

func (filesystem *localFileSystem) DeleteDirectory(path string) error {
	exists, err := filesystem.DirectoryExists(path)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	// Use RemoveAll to remove directory and all its contents
	if err := os.RemoveAll(path); err != nil {
		return err
	}

	return nil
}

func (filesystem *localFileSystem) DeleteFile(path string) error {
	exists, err := filesystem.FileExists(path)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	if err := os.Remove(path); err != nil {
		return err
	}

	return nil
}

func (filesystem *localFileSystem) DirectoryExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (filesystem *localFileSystem) FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (filesystem *localFileSystem) FileMetaData(path string) (os.FileInfo, error) {
	exists, err := filesystem.FileExists(path)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("filesystem: file %s does not exist", path)
	}

	return os.Stat(path)
}

// FileSize implements Filesystem.
func (filesystem *localFileSystem) FileSize(path string) (int64, error) {
	info, err := filesystem.FileMetaData(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// ListDirectory implements Filesystem.
func (filesystem *localFileSystem) ListDirectory(path string) ([]os.FileInfo, error) {
	exists, err := filesystem.DirectoryExists(path)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("filesystem: directory %s does not exist", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	infos := make([]os.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}

	return infos, nil
}

// IsFile implements Filesystem.
func (filesystem *localFileSystem) IsFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}

// IsDirectory implements Filesystem.
func (filesystem *localFileSystem) IsDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

// GetAbsolutePath implements Filesystem.
func (filesystem *localFileSystem) GetAbsolutePath(path string) (string, error) {
	return filepath.Abs(path)
}

func (filesystem *localFileSystem) MoveFile(source string, destination string) error {
	if err := filesystem.CopyFile(source, destination); err != nil {
		return err
	}

	if err := os.Remove(source); err != nil {
		return err
	}

	return nil
}

func (filesystem *localFileSystem) ReadFile(path string) ([]byte, error) {
	exists, err := filesystem.FileExists(path)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("filesystem: file %s does not exist", path)
	}

	return os.ReadFile(path)
}

func (filesystem *localFileSystem) WriteFile(path string, content []byte) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := filesystem.CreateDirectory(dir); err != nil {
		return err
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		return err
	}

	return nil
}

func NewLocalFileSystem() Filesystem {
	return &localFileSystem{}
}

// Convenience functions for common operations

// EnsureDirectory creates a directory and all parent directories if they don't exist
func EnsureDirectory(filesystem Filesystem, path string) error {
	return filesystem.CreateDirectory(path)
}

// CopyDirectory recursively copies a directory and all its contents
func CopyDirectory(filesystem Filesystem, source, destination string) error {
	// Check if source directory exists
	exists, err := filesystem.DirectoryExists(source)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("filesystem: source directory %s does not exist", source)
	}

	// Create destination directory
	if err := filesystem.CreateDirectory(destination); err != nil {
		return err
	}

	// List source directory contents
	entries, err := filesystem.ListDirectory(source)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(source, entry.Name())
		dstPath := filepath.Join(destination, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := CopyDirectory(filesystem, srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := filesystem.CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetFileExtension returns the file extension
func GetFileExtension(path string) string {
	return filepath.Ext(path)
}

// GetFileName returns the filename without path
func GetFileName(path string) string {
	return filepath.Base(path)
}

// GetDirectoryName returns the directory path
func GetDirectoryName(path string) string {
	return filepath.Dir(path)
}
