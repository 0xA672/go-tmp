package gotmp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// ==================== Basic Tests ====================

func TestNewSandbox(t *testing.T) {
	sandbox, err := NewSandbox("gotmp_test_*")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sandbox.Cleanup()

	if _, err := os.Stat(sandbox.Root()); os.IsNotExist(err) {
		t.Errorf("Sandbox root directory does not exist: %s", sandbox.Root())
	}
	if sandbox.Root() == "" {
		t.Error("Sandbox root directory should not be empty")
	}
	if sandbox.IsClosed() {
		t.Error("New sandbox should not be closed")
	}
}

func TestMdir(t *testing.T) {
	sandbox, err := NewSandbox("gotmp_mdir_*")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sandbox.Cleanup()

	subDir, err := sandbox.Mdir("subdir")
	if err != nil {
		t.Fatalf("Failed to create sub directory: %v", err)
	}
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Errorf("Sub directory does not exist: %s", subDir)
	}

	nestedDir, err := sandbox.Mdir("deep/nested/dir")
	if err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Errorf("Nested directory does not exist: %s", nestedDir)
	}

	sameDir, err := sandbox.Mdir("subdir")
	if err != nil {
		t.Fatalf("Failed to create existing directory again: %v", err)
	}
	if sameDir != subDir {
		t.Errorf("Same directory path mismatch: got %s, want %s", sameDir, subDir)
	}
}

func TestCreateFile(t *testing.T) {
	sandbox, err := NewSandbox("gotmp_file_*")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sandbox.Cleanup()

	filePath, err := sandbox.CreateFile("test.txt", []byte("hello go-tmp"))
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != "hello go-tmp" {
		t.Errorf("File content mismatch, got: %s, want: hello go-tmp", string(content))
	}

	nestedPath, err := sandbox.CreateFile("nested/dir/file.txt", []byte("nested content"))
	if err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}
	if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
		t.Errorf("Nested file does not exist: %s", nestedPath)
	}

	emptyPath, err := sandbox.CreateFile("empty.txt", []byte{})
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}
	emptyContent, err := os.ReadFile(emptyPath)
	if err != nil {
		t.Fatalf("Failed to read empty file: %v", err)
	}
	if len(emptyContent) != 0 {
		t.Errorf("Empty file should have no content, got: %d bytes", len(emptyContent))
	}

	overwritePath, err := sandbox.CreateFile("test.txt", []byte("overwritten"))
	if err != nil {
		t.Fatalf("Failed to overwrite file: %v", err)
	}
	overwrittenContent, err := os.ReadFile(overwritePath)
	if err != nil {
		t.Fatalf("Failed to read overwritten file: %v", err)
	}
	if string(overwrittenContent) != "overwritten" {
		t.Errorf("Overwritten content mismatch, got: %s", string(overwrittenContent))
	}
}

// ==================== New Method Tests ====================

func TestReadFile(t *testing.T) {
	sandbox, err := NewSandbox("gotmp_read_*")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sandbox.Cleanup()

	_, err = sandbox.CreateFile("read_test.txt", []byte("read me"))
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	content, err := sandbox.ReadFile("read_test.txt")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != "read me" {
		t.Errorf("ReadFile content mismatch, got: %s, want: read me", string(content))
	}

	_, err = sandbox.ReadFile("nonexistent.txt")
	if err == nil {
		t.Error("Should fail when reading nonexistent file")
	}
}

func TestRemove(t *testing.T) {
	sandbox, err := NewSandbox("gotmp_remove_*")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sandbox.Cleanup()

	_, err = sandbox.CreateFile("to_remove.txt", []byte("remove me"))
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	err = sandbox.Remove("to_remove.txt")
	if err != nil {
		t.Fatalf("Failed to remove file: %v", err)
	}
	exists, err := sandbox.Exists("to_remove.txt")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("File should not exist after removal")
	}

	_, err = sandbox.Mdir("dir_to_remove")
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	err = sandbox.Remove("dir_to_remove")
	if err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	exists, err = sandbox.Exists("dir_to_remove")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Directory should not exist after removal")
	}
}

func TestExists(t *testing.T) {
	sandbox, err := NewSandbox("gotmp_exists_*")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sandbox.Cleanup()

	exists, err := sandbox.Exists("nonexistent.txt")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Nonexistent path should not exist")
	}

	_, err = sandbox.CreateFile("exists.txt", []byte("I exist"))
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	exists, err = sandbox.Exists("exists.txt")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Created file should exist")
	}

	_, err = sandbox.Mdir("exists_dir")
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	exists, err = sandbox.Exists("exists_dir")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Created directory should exist")
	}
}

func TestJoin(t *testing.T) {
	sandbox, err := NewSandbox("gotmp_join_*")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sandbox.Cleanup()

	joined := sandbox.Join("file.txt")
	expected := filepath.Join(sandbox.Root(), "file.txt")
	if joined != expected {
		t.Errorf("Join result mismatch, got: %s, want: %s", joined, expected)
	}

	joined = sandbox.Join("deep", "nested", "file.txt")
	expected = filepath.Join(sandbox.Root(), "deep", "nested", "file.txt")
	if joined != expected {
		t.Errorf("Join result mismatch, got: %s, want: %s", joined, expected)
	}
}

func TestIsClosed(t *testing.T) {
	sandbox, err := NewSandbox("gotmp_closed_*")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	if sandbox.IsClosed() {
		t.Error("New sandbox should not be closed")
	}

	if err := sandbox.Cleanup(); err != nil {
		t.Fatalf("Failed to cleanup sandbox: %v", err)
	}
	if !sandbox.IsClosed() {
		t.Error("Sandbox should be closed after cleanup")
	}
}

func TestClose(t *testing.T) {
	sandbox, err := NewSandbox("gotmp_close_*")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	if err := sandbox.Close(); err != nil {
		t.Fatalf("Failed to close sandbox: %v", err)
	}
	if !sandbox.IsClosed() {
		t.Error("Sandbox should be closed after Close()")
	}

	if _, err := os.Stat(sandbox.Root()); !os.IsNotExist(err) {
		t.Errorf("Sandbox root directory still exists after close: %s", sandbox.Root())
	}

	if err := sandbox.Close(); err != nil {
		t.Fatalf("Second close failed: %v", err)
	}
}

// ==================== After Cleanup Tests ====================

func TestOperationsAfterCleanup(t *testing.T) {
	sandbox, err := NewSandbox("gotmp_after_*")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	if err := sandbox.Cleanup(); err != nil {
		t.Fatalf("Failed to cleanup sandbox: %v", err)
	}

	_, err = sandbox.Mdir("should_fail")
	if err == nil {
		t.Error("Should not be able to create directory after cleanup")
	}

	_, err = sandbox.CreateFile("should_fail.txt", []byte("fail"))
	if err == nil {
		t.Error("Should not be able to create file after cleanup")
	}

	_, err = sandbox.ReadFile("should_fail.txt")
	if err == nil {
		t.Error("Should not be able to read file after cleanup")
	}

	err = sandbox.Remove("should_fail.txt")
	if err == nil {
		t.Error("Should not be able to remove after cleanup")
	}

	_, err = sandbox.Exists("should_fail.txt")
	if err == nil {
		t.Error("Should not be able to check existence after cleanup")
	}
}

func TestCleanupIdempotent(t *testing.T) {
	sandbox, err := NewSandbox("gotmp_idempotent_*")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	for i := 0; i < 5; i++ {
		if err := sandbox.Cleanup(); err != nil {
			t.Fatalf("Cleanup %d failed: %v", i+1, err)
		}
	}
}

// ==================== Concurrency Tests ====================

func TestConcurrentAccess(t *testing.T) {
	sandbox, err := NewSandbox("gotmp_concurrent_*")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sandbox.Cleanup()

	var wg sync.WaitGroup
	numGoroutines := 20

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			_, err := sandbox.Mdir(filepath.Join("concurrent", "dir", fmt.Sprintf("%d", idx)))
			if err != nil {
				t.Errorf("Concurrent Mdir failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			name := filepath.Join("concurrent_files", fmt.Sprintf("file_%d.txt", idx))
			_, err := sandbox.CreateFile(name, []byte(fmt.Sprintf("content_%d", idx)))
			if err != nil {
				t.Errorf("Concurrent CreateFile failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			name := filepath.Join("concurrent_files", fmt.Sprintf("file_%d.txt", idx))
			content, err := sandbox.ReadFile(name)
			if err != nil {
				t.Errorf("Concurrent ReadFile failed: %v", err)
				return
			}
			expected := fmt.Sprintf("content_%d", idx)
			if string(content) != expected {
				t.Errorf("Concurrent read mismatch, got: %s, want: %s", string(content), expected)
			}
		}(i)
	}
	wg.Wait()

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			name := filepath.Join("concurrent_files", fmt.Sprintf("file_%d.txt", idx))
			exists, err := sandbox.Exists(name)
			if err != nil {
				t.Errorf("Concurrent Exists failed: %v", err)
				return
			}
			if !exists {
				t.Errorf("Concurrent file should exist: %s", name)
			}
		}(i)
	}
	wg.Wait()
}

// ==================== Error Wrapping Tests ====================

func TestErrorWrapping(t *testing.T) {
	sandbox, err := NewSandbox("gotmp_error_*")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sandbox.Cleanup()

	_, err = sandbox.ReadFile("nonexistent.txt")
	if err == nil {
		t.Fatal("Expected error for reading nonexistent file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Error should wrap os.ErrNotExist, got: %v", err)
	}

	if err := sandbox.Cleanup(); err != nil {
		t.Fatalf("Failed to cleanup: %v", err)
	}

	_, err = sandbox.Mdir("after_cleanup")
	if err == nil {
		t.Fatal("Expected error for Mdir after cleanup")
	}
	if !errors.Is(err, ErrSandboxClosed) {
		t.Errorf("Error should be ErrSandboxClosed, got: %v", err)
	}

	_, err = sandbox.CreateFile("after_cleanup.txt", nil)
	if err == nil {
		t.Fatal("Expected error for CreateFile after cleanup")
	}
	if !errors.Is(err, ErrSandboxClosed) {
		t.Errorf("Error should be ErrSandboxClosed, got: %v", err)
	}
}
