# go-tmp

A Go library for managing temporary directory sandboxes with concurrent safety, automatic cleanup, and error wrapping.

## Features

- **Sandbox Management** — Create isolated temporary directories with a simple API
- **Concurrent Safety** — All operations are protected by a mutex for safe goroutine access
- **Idempotent Cleanup** — Multiple `Cleanup()` calls never error
- **io.Closer Interface** — Use `defer sb.Close()` for clean resource management
- **Error Wrapping** — Errors are wrapped with `%w` for `errors.Is` / `errors.As` chaining
- **Sentinel Error** — `ErrSandboxClosed` for detecting post-cleanup operations

## Installation

```bash
go get github.com/yourusername/go-tmp
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    gotmp "github.com/yourusername/go-tmp"
)

func main() {
    // Create a new sandbox
    sb, err := gotmp.NewSandbox("myapp_*")
    if err != nil {
        log.Fatal(err)
    }
    defer sb.Close() // Auto-cleanup when done

    // Create subdirectories
    dir, err := sb.Mdir("data/output")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Created dir:", dir)

    // Create files (parent directories are auto-created)
    path, err := sb.CreateFile("config/settings.json", []byte(`{"key": "value"}`))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Created file:", path)

    // Read files
    content, err := sb.ReadFile("config/settings.json")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Content:", string(content))

    // Check existence
    exists, err := sb.Exists("config/settings.json")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Exists:", exists)

    // Remove files or directories
    err = sb.Remove("config")
    if err != nil {
        log.Fatal(err)
    }

    // Join paths
    fullPath := sb.Join("data", "output", "result.txt")
    fmt.Println("Full path:", fullPath)
}
```

## API Reference

### `func NewSandbox(pattern string) (*Sandbox, error)`

Creates a new temporary directory sandbox. The `pattern` follows `os.MkdirTemp` conventions (e.g. `"myapp_*"`).

### `func (s *Sandbox) Root() string`

Returns the root directory path of the sandbox.

### `func (s *Sandbox) Join(elem ...string) string`

Joins the sandbox root with sub-paths. Does not check if the sandbox is closed or if the path exists.

### `func (s *Sandbox) Exists(name string) (bool, error)`

Checks whether the specified path exists within the sandbox.

### `func (s *Sandbox) Mdir(name string) (string, error)`

Creates a subdirectory within the sandbox. Parent directories are created automatically.

### `func (s *Sandbox) CreateFile(name string, content []byte) (string, error)`

Creates and writes a file within the sandbox. Parent directories are created automatically.

### `func (s *Sandbox) ReadFile(name string) ([]byte, error)`

Reads the content of the specified file within the sandbox.

### `func (s *Sandbox) Remove(name string) error`

Deletes the specified file or directory within the sandbox.

### `func (s *Sandbox) Close() error`

Implements `io.Closer`. Equivalent to `Cleanup()`.

### `func (s *Sandbox) Cleanup() error`

Recursively removes the sandbox and all its contents. Idempotent — multiple calls never error.

### `func (s *Sandbox) IsClosed() bool`

Returns whether the sandbox has been cleaned up.

### `var ErrSandboxClosed`

Sentinel error returned when operations are attempted on a cleaned-up sandbox. Use `errors.Is(err, ErrSandboxClosed)` to check.

## Error Handling

All errors are wrapped with `fmt.Errorf("%w", ...)` for proper error chaining:

```go
_, err := sb.ReadFile("nonexistent.txt")
if errors.Is(err, os.ErrNotExist) {
    fmt.Println("File not found")
}

sb.Cleanup()
_, err = sb.Mdir("test")
if errors.Is(err, gotmp.ErrSandboxClosed) {
    fmt.Println("Sandbox is closed")
}
```

## License

MIT
