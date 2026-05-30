package gotmp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// ErrSandboxClosed indicates that the sandbox has been cleaned up and no further operations can be performed
var ErrSandboxClosed = errors.New("sandbox is already cleaned up")

// Sandbox represents a managed temporary directory sandbox
type Sandbox struct {
	rootDir string     // Root directory path of the sandbox
	mu      sync.Mutex // Mutex for concurrent safety
	closed  bool       // Indicates whether the sandbox has been cleaned up
}

// NewSandbox creates a new temporary directory sandbox
// pattern follows os.MkdirTemp conventions, e.g. "myapp_*"
func NewSandbox(pattern string) (*Sandbox, error) {
	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}

	return &Sandbox{
		rootDir: dir,
		closed:  false,
	}, nil
}

// Root returns the root directory path of the sandbox
func (s *Sandbox) Root() string {
	return s.rootDir
}

// Join joins the sandbox root directory with sub-paths and returns the full path
// Note: this method does not check if the sandbox is closed or if the path exists
func (s *Sandbox) Join(elem ...string) string {
	return filepath.Join(append([]string{s.rootDir}, elem...)...)
}

// Exists checks whether the specified path exists within the sandbox
func (s *Sandbox) Exists(name string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return false, fmt.Errorf("%w", ErrSandboxClosed)
	}

	path := filepath.Join(s.rootDir, name)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check existence: %w", err)
}

// Mdir creates a subdirectory within the sandbox
// Parent directories are created automatically if they do not exist
func (s *Sandbox) Mdir(name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return "", fmt.Errorf("%w", ErrSandboxClosed)
	}

	path := filepath.Join(s.rootDir, name)
	// 0755: owner read/write/execute, group and others read/execute
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory in sandbox: %w", err)
	}
	return path, nil
}

// CreateFile creates and writes a file within the sandbox
// Parent directories are created automatically if they do not exist
func (s *Sandbox) CreateFile(name string, content []byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return "", fmt.Errorf("%w", ErrSandboxClosed)
	}

	path := filepath.Join(s.rootDir, name)

	// Ensure the parent directory of the file exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("failed to create parent directory: %w", err)
	}

	// 0644: owner read/write, group and others read-only
	if err := os.WriteFile(path, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write file in sandbox: %w", err)
	}
	return path, nil
}

// ReadFile reads the content of the specified file within the sandbox.
// This operation is thread-safe and will fail if the sandbox has been closed.
//
// Parameters:
//   - name: the relative path of the file to read within the sandbox
//
// Returns:
//   - []byte: the content of the file if successfully read
//   - error: an error if the sandbox is closed, or if the file cannot be read
func (s *Sandbox) ReadFile(name string) ([]byte, error) {
	// Ensure thread-safe access to the sandbox
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the sandbox has been closed
	if s.closed {
		return nil, fmt.Errorf("%w", ErrSandboxClosed)
	}

	// Construct the full file path and read its content
	path := filepath.Join(s.rootDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file in sandbox: %w", err)
	}
	return data, nil
}

// Remove removes a file or directory within the sandbox by name.
// This operation is thread-safe and will fail if the sandbox has been closed.
//
// Parameters:
//   - name: the relative path of the file or directory to remove within the sandbox
//
// Returns:
//   - error: nil on success, or an error if the sandbox is closed or the removal fails
//
// Possible errors:
//   - ErrSandboxClosed: if the sandbox has already been cleaned up
//   - os.RemoveAll error: wrapped with additional context if the filesystem operation fails
func (s *Sandbox) Remove(name string) error {
	// Acquire lock to ensure thread-safe access to sandbox state
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if sandbox has been closed
	if s.closed {
		return fmt.Errorf("%w", ErrSandboxClosed)
	}

	// Construct the full path by joining sandbox root with the provided name
	path := filepath.Join(s.rootDir, name)
	
	// Remove the file or directory recursively
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove path in sandbox: %w", err)
	}
	return nil
}

// Close implements the io.Closer interface, equivalent to Cleanup
// Allows Sandbox to be used with defer: defer sb.Close()
func (s *Sandbox) Close() error {
	return s.Cleanup()
}

// Cleanup recursively removes the sandbox and all files and directories within it
// This operation is idempotent; multiple calls will not return an error
func (s *Sandbox) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil // Idempotent: multiple calls will not error
	}

	if err := os.RemoveAll(s.rootDir); err != nil {
		return fmt.Errorf("failed to cleanup sandbox: %w", err)
	}

	s.closed = true
	return nil
}

// IsClosed returns whether the sandbox has been cleaned up
func (s *Sandbox) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Ensure Sandbox implements the io.Closer interface
var _ io.Closer = (*Sandbox)(nil)
