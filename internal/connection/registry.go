package connection

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/steveyegge/gastown/internal/filelock"
)

// Machine represents a managed machine in the federation.
type Machine struct {
	Name     string `json:"name"`
	Type     string `json:"type"`      // "local", "ssh"
	Host     string `json:"host"`      // for ssh: user@host
	KeyPath  string `json:"key_path"`  // SSH private key path
	TownPath string `json:"town_path"` // Path to town root on remote
}

// registryData is the JSON file structure.
type registryData struct {
	Version  int                 `json:"version"`
	Machines map[string]*Machine `json:"machines"`
}

// MachineRegistry manages machine configurations and provides Connection instances.
// Uses file-level locking for multi-process safety.
type MachineRegistry struct {
	path     string
	machines map[string]*Machine
	mu       sync.RWMutex // Protects in-memory state
}

// NewMachineRegistry creates a registry from the given config file path.
// If the file doesn't exist, an empty registry is created.
func NewMachineRegistry(configPath string) (*MachineRegistry, error) {
	r := &MachineRegistry{
		path:     configPath,
		machines: make(map[string]*Machine),
	}

	// Load existing config if present
	if err := r.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading registry: %w", err)
	}

	// Ensure "local" machine always exists
	if _, ok := r.machines["local"]; !ok {
		r.machines["local"] = &Machine{
			Name: "local",
			Type: "local",
		}
	}

	return r, nil
}

// load reads the registry from disk with file locking.
func (r *MachineRegistry) load() error {
	return filelock.WithReadLock(r.path, func() error {
		return r.loadUnsafe()
	})
}

// loadUnsafe reads the registry from disk without file locking.
// Must only be called when file lock is already held.
// Still uses in-memory mutex to protect r.machines from concurrent goroutine access.
func (r *MachineRegistry) loadUnsafe() error {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}

	var rd registryData
	if err := json.Unmarshal(data, &rd); err != nil {
		return fmt.Errorf("parsing registry: %w", err)
	}

	if rd.Machines == nil {
		rd.Machines = make(map[string]*Machine)
	}

	// Populate machine names from keys
	for name, m := range rd.Machines {
		m.Name = name
	}

	// Update in-memory state under mutex
	r.mu.Lock()
	r.machines = rd.Machines
	r.mu.Unlock()

	return nil
}

// save writes the registry to disk atomically with file locking.
func (r *MachineRegistry) save() error {
	// Ensure parent directory exists before acquiring lock
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	return filelock.WithWriteLock(r.path, func() error {
		return r.saveUnsafe()
	})
}

// saveUnsafe writes the registry to disk without locking.
// Must only be called when file lock is already held.
func (r *MachineRegistry) saveUnsafe() error {
	rd := registryData{
		Version:  1,
		Machines: r.machines,
	}

	data, err := json.MarshalIndent(rd, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling registry: %w", err)
	}

	// Atomic write via unique temp file (multi-process safe)
	dir := filepath.Dir(r.path)
	base := filepath.Base(r.path)
	tmpFile, err := os.CreateTemp(dir, base+".*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up on error

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, r.path); err != nil {
		return fmt.Errorf("writing registry: %w", err)
	}

	return nil
}

// Get returns a machine by name.
func (r *MachineRegistry) Get(name string) (*Machine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.machines[name]
	if !ok {
		return nil, fmt.Errorf("machine not found: %s", name)
	}
	return m, nil
}

// Add adds or updates a machine in the registry.
// Uses read-modify-write pattern with file locking for multi-process safety.
func (r *MachineRegistry) Add(m *Machine) error {
	if m.Name == "" {
		return fmt.Errorf("machine name is required")
	}
	if m.Type == "" {
		return fmt.Errorf("machine type is required")
	}
	if m.Type == "ssh" && m.Host == "" {
		return fmt.Errorf("ssh machine requires host")
	}

	// Ensure parent directory exists before acquiring lock
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	return filelock.WithWriteLock(r.path, func() error {
		// Reload from disk to get latest state (multi-process safety)
		if err := r.loadUnsafe(); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("reloading registry: %w", err)
		}

		// Modify in-memory state (already protected by loadUnsafe's mutex)
		r.mu.Lock()
		r.machines[m.Name] = m
		r.mu.Unlock()

		// Save back to disk
		return r.saveUnsafe()
	})
}

// Remove removes a machine from the registry.
// Uses read-modify-write pattern with file locking for multi-process safety.
func (r *MachineRegistry) Remove(name string) error {
	if name == "local" {
		return fmt.Errorf("cannot remove local machine")
	}

	return filelock.WithWriteLock(r.path, func() error {
		// Reload from disk to get latest state (multi-process safety)
		if err := r.loadUnsafe(); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("reloading registry: %w", err)
		}

		// Check if machine exists
		r.mu.RLock()
		_, ok := r.machines[name]
		r.mu.RUnlock()

		if !ok {
			return fmt.Errorf("machine not found: %s", name)
		}

		// Modify in-memory state
		r.mu.Lock()
		delete(r.machines, name)
		r.mu.Unlock()

		// Save back to disk
		return r.saveUnsafe()
	})
}

// List returns all machines in the registry.
func (r *MachineRegistry) List() []*Machine {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Machine, 0, len(r.machines))
	for _, m := range r.machines {
		result = append(result, m)
	}
	return result
}

// Connection returns a Connection for the named machine.
func (r *MachineRegistry) Connection(name string) (Connection, error) {
	m, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	switch m.Type {
	case "local":
		return NewLocalConnection(), nil
	case "ssh":
		// SSH connection not yet implemented
		return nil, fmt.Errorf("ssh connections not yet implemented")
	default:
		return nil, fmt.Errorf("unknown machine type: %s", m.Type)
	}
}

// LocalConnection returns the local connection.
// This is a convenience method for the common case.
func (r *MachineRegistry) LocalConnection() *LocalConnection {
	return NewLocalConnection()
}