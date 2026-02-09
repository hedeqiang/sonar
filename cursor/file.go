package cursor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// File is a file-based Cursor implementation that persists progress as JSON.
type File struct {
	mu   sync.Mutex
	path string
}

// NewFile creates a file-backed cursor. The directory containing path
// will be created if it does not exist.
func NewFile(path string) *File {
	return &File{path: path}
}

// Load reads the last saved block number for the chain from the file.
func (f *File) Load(chainID string) (uint64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.readAll()
	if err != nil {
		return 0, nil // file doesn't exist yet, start from 0
	}
	return data[chainID], nil
}

// Save writes the block number for the chain to the file.
func (f *File) Save(chainID string, block uint64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, _ := f.readAll()
	if data == nil {
		data = make(map[string]uint64)
	}
	data[chainID] = block

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(f.path, b, 0o644)
}

func (f *File) readAll() (map[string]uint64, error) {
	b, err := os.ReadFile(f.path)
	if err != nil {
		return nil, err
	}
	var data map[string]uint64
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return data, nil
}
