package filesystem

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
)

type Filesystem interface {
	ReadFile(path string) ([]byte, error)
	// ReadStream(path string) []byte
	WriteFile(path string, content []byte) error
	// WriteStream(path string, content []byte) error

	CreateFile(path string) error
	DeleteFile(path string) error
	MoveFile(source, destination string) error
	CopyFile(source, destination string) error

	FileExists(path string) (bool, error)
	FileMetaData(path string) error

	DirectoryExists(path string) (bool, error)
	DeleteDirectory(path string) error
	CreateDirectory(path string) error
}

type localFileSystem struct {
}

// CopyFile implements Filesystem.
func (filesystem *localFileSystem) CopyFile(source string, destination string) error {
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
		return fmt.Errorf("filesystem: destination file %s already exists", source)
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := sourceFile.Close(); closeErr != nil {
			log.Printf("closing file error: %v", err)
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

	if _, err := os.Create(path); err != nil {
		return err
	}

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

	// todo check if empty

	if err := os.Remove(path); err != nil {
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

func (filesystem *localFileSystem) FileMetaData(path string) error {
	panic("unimplemented")
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
	exists, err := filesystem.FileExists(path)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("filesystem: file %s does not exist", path)
	}

	if err := os.WriteFile(path, content, os.ModeAppend); err != nil {
		return err
	}

	return nil
}

func NewLocalFileSystem() Filesystem {
	return &localFileSystem{}
}
