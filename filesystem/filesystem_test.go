package filesystem

import (
	"path/filepath"
	"testing"
)

func TestLocalFileSystem(t *testing.T) {
	fs := NewLocalFileSystem()
	tempDir := t.TempDir()

	// Test CreateDirectory
	testDir := filepath.Join(tempDir, "testdir")
	if err := fs.CreateDirectory(testDir); err != nil {
		t.Errorf("CreateDirectory failed: %v", err)
	}

	// Test DirectoryExists
	exists, err := fs.DirectoryExists(testDir)
	if err != nil {
		t.Errorf("DirectoryExists failed: %v", err)
	}
	if !exists {
		t.Error("Directory should exist")
	}

	// Test CreateFile
	testFile := filepath.Join(testDir, "test.txt")
	if err := fs.CreateFile(testFile); err != nil {
		t.Errorf("CreateFile failed: %v", err)
	}

	// Test FileExists
	exists, err = fs.FileExists(testFile)
	if err != nil {
		t.Errorf("FileExists failed: %v", err)
	}
	if !exists {
		t.Error("File should exist")
	}

	// Test WriteFile
	content := []byte("Hello, World!")
	if err := fs.WriteFile(testFile, content); err != nil {
		t.Errorf("WriteFile failed: %v", err)
	}

	// Test ReadFile
	readContent, err := fs.ReadFile(testFile)
	if err != nil {
		t.Errorf("ReadFile failed: %v", err)
	}
	if string(readContent) != string(content) {
		t.Errorf("Expected %s, got %s", content, readContent)
	}

	// Test AppendFile
	appendContent := []byte(" Appended text")
	if err := fs.AppendFile(testFile, appendContent); err != nil {
		t.Errorf("AppendFile failed: %v", err)
	}

	// Test FileSize
	size, err := fs.FileSize(testFile)
	if err != nil {
		t.Errorf("FileSize failed: %v", err)
	}
	expectedSize := int64(len(content) + len(appendContent))
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}

	// Test CopyFile
	copyFile := filepath.Join(testDir, "copy.txt")
	if err := fs.CopyFile(testFile, copyFile); err != nil {
		t.Errorf("CopyFile failed: %v", err)
	}

	// Test IsFile
	isFile, err := fs.IsFile(copyFile)
	if err != nil {
		t.Errorf("IsFile failed: %v", err)
	}
	if !isFile {
		t.Error("Copy should be a file")
	}

	// Test IsDirectory
	isDir, err := fs.IsDirectory(testDir)
	if err != nil {
		t.Errorf("IsDirectory failed: %v", err)
	}
	if !isDir {
		t.Error("testDir should be a directory")
	}

	// Test ListDirectory
	files, err := fs.ListDirectory(testDir)
	if err != nil {
		t.Errorf("ListDirectory failed: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// Test MoveFile
	moveFile := filepath.Join(testDir, "moved.txt")
	if err := fs.MoveFile(copyFile, moveFile); err != nil {
		t.Errorf("MoveFile failed: %v", err)
	}

	// Verify original file doesn't exist
	exists, err = fs.FileExists(copyFile)
	if err != nil {
		t.Errorf("FileExists failed: %v", err)
	}
	if exists {
		t.Error("Original file should not exist after move")
	}

	// Verify moved file exists
	exists, err = fs.FileExists(moveFile)
	if err != nil {
		t.Errorf("FileExists failed: %v", err)
	}
	if !exists {
		t.Error("Moved file should exist")
	}
}

func TestUtilityFunctions(t *testing.T) {
	path := "/path/to/file.txt"

	ext := GetFileExtension(path)
	if ext != ".txt" {
		t.Errorf("Expected .txt, got %s", ext)
	}

	name := GetFileName(path)
	if name != "file.txt" {
		t.Errorf("Expected file.txt, got %s", name)
	}

	dir := GetDirectoryName(path)
	if dir != "/path/to" {
		t.Errorf("Expected /path/to, got %s", dir)
	}
}
