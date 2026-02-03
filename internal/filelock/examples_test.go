package filelock_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/steveyegge/gastown/internal/filelock"
)

// Example of protecting a simple state file
func ExampleWithWriteLock() {
	// Protect state file writes
	err := filelock.WithWriteLock("/tmp/state.json", func() error {
		data := map[string]interface{}{
			"enabled": true,
			"version": "1.0.0",
		}
		bytes, _ := json.MarshalIndent(data, "", "  ")
		return os.WriteFile("/tmp/state.json", bytes, 0644)
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

// Example of protecting a JSONL file (like beads issues.jsonl)
func ExampleFileLock_jsonlAppend() {
	path := "/tmp/issues.jsonl"

	// Append a new issue with write lock
	appendIssue := func(issue map[string]string) error {
		return filelock.WithWriteLock(path, func() error {
			f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer f.Close()
			return json.NewEncoder(f).Encode(issue)
		})
	}

	// Append some issues
	_ = appendIssue(map[string]string{"id": "gt-001", "title": "First issue"})
	_ = appendIssue(map[string]string{"id": "gt-002", "title": "Second issue"})

	// Read all issues with read lock
	var issues []map[string]string
	err := filelock.WithReadLock(path, func() error {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var issue map[string]string
			if err := json.Unmarshal(scanner.Bytes(), &issue); err != nil {
				return err
			}
			issues = append(issues, issue)
		}
		return scanner.Err()
	})

	if err != nil {
		fmt.Printf("Error reading: %v\n", err)
		return
	}

	_ = issues // Use the variable
	// Output example (actual count may vary in test runs)
}

// Example of protecting a registry with read-modify-write
func ExampleFileLock_readModifyWrite() {
	path := "/tmp/registry.json"

	// Read-modify-write pattern
	updateRegistry := func(agentID string, status string) error {
		return filelock.WithWriteLock(path, func() error {
			// Read current state
			var registry map[string]string
			data, err := os.ReadFile(path)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			if len(data) > 0 {
				if err := json.Unmarshal(data, &registry); err != nil {
					return err
				}
			} else {
				registry = make(map[string]string)
			}

			// Modify
			registry[agentID] = status

			// Write back atomically
			newData, _ := json.MarshalIndent(registry, "", "  ")
			tmp := path + ".tmp"
			if err := os.WriteFile(tmp, newData, 0644); err != nil {
				return err
			}
			return os.Rename(tmp, path)
		})
	}

	_ = updateRegistry("agent-1", "active")
	_ = updateRegistry("agent-2", "idle")
	_ = updateRegistry("agent-1", "busy")

	// Read registry
	var registry map[string]string
	_ = filelock.WithReadLock(path, func() error {
		data, _ := os.ReadFile(path)
		return json.Unmarshal(data, &registry)
	})

	fmt.Printf("agent-1: %s, agent-2: %s\n", registry["agent-1"], registry["agent-2"])
	// Output: agent-1: busy, agent-2: idle
}

// Example of using manual lock management for complex operations
func ExampleFileLock_manualLock() {
	path := "/tmp/data.json"

	// Create lock with custom options
	lock := filelock.NewWithOptions(path, filelock.Options{
		Timeout:  5000000000, // 5 seconds
		LockType: filelock.Exclusive,
	})

	// Acquire lock
	if err := lock.Lock(); err != nil {
		fmt.Printf("Failed to acquire lock: %v\n", err)
		return
	}

	// Always unlock (even if panic occurs)
	defer lock.Unlock()

	// Perform multiple operations under single lock
	data1 := map[string]string{"step": "1"}
	bytes1, _ := json.Marshal(data1)
	_ = os.WriteFile(path, bytes1, 0644)

	data2 := map[string]string{"step": "2"}
	bytes2, _ := json.Marshal(data2)
	_ = os.WriteFile(path, bytes2, 0644)

	fmt.Println("Operations completed")
	// Output: Operations completed
}

// Example of protecting beads database operations
func ExampleFileLock_beadsDatabase() {
	beadsDir := filepath.Join("/tmp", ".beads")
	issuesPath := filepath.Join(beadsDir, "issues.jsonl")
	_ = os.MkdirAll(beadsDir, 0755)

	type Issue struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}

	// Append new issue
	createIssue := func(issue Issue) error {
		return filelock.WithWriteLock(issuesPath, func() error {
			f, err := os.OpenFile(issuesPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer f.Close()
			return json.NewEncoder(f).Encode(issue)
		})
	}

	// List all issues
	listIssues := func() ([]Issue, error) {
		var issues []Issue
		err := filelock.WithReadLock(issuesPath, func() error {
			f, err := os.Open(issuesPath)
			if err != nil {
				if os.IsNotExist(err) {
					return nil // Empty list for non-existent file
				}
				return err
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				var issue Issue
				if err := json.Unmarshal(scanner.Bytes(), &issue); err != nil {
					return err
				}
				issues = append(issues, issue)
			}
			return scanner.Err()
		})
		return issues, err
	}

	// Create some issues
	_ = createIssue(Issue{ID: "gt-100", Title: "Add file locking", Status: "done"})
	_ = createIssue(Issue{ID: "gt-101", Title: "Test file locking", Status: "ready"})

	// List them
	issues, _ := listIssues()
	_ = issues // Use the variable
	// Output example (actual count may vary in test runs)
}

// Example of protecting queue operations
func ExampleFileLock_queue() {
	queuePath := "/tmp/queue.jsonl"

	type Task struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}

	// Enqueue task
	enqueue := func(task Task) error {
		return filelock.WithWriteLock(queuePath, func() error {
			f, err := os.OpenFile(queuePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer f.Close()
			return json.NewEncoder(f).Encode(task)
		})
	}

	// Dequeue task (read and remove first line)
	dequeue := func() (*Task, error) {
		var task *Task
		err := filelock.WithWriteLock(queuePath, func() error {
			// Read all tasks
			data, err := os.ReadFile(queuePath)
			if err != nil {
				if os.IsNotExist(err) {
					return nil // Empty queue
				}
				return err
			}

			// Parse tasks
			var tasks []Task
			scanner := bufio.NewScanner(bufio.NewReader(os.NewFile(0, "")))
			scanner.Split(bufio.ScanLines)
			for _, line := range splitLines(string(data)) {
				if line == "" {
					continue
				}
				var t Task
				if err := json.Unmarshal([]byte(line), &t); err != nil {
					return err
				}
				tasks = append(tasks, t)
			}

			if len(tasks) == 0 {
				return nil // Empty queue
			}

			// Take first task
			task = &tasks[0]

			// Write remaining tasks back
			f, err := os.Create(queuePath)
			if err != nil {
				return err
			}
			defer f.Close()

			for i := 1; i < len(tasks); i++ {
				if err := json.NewEncoder(f).Encode(tasks[i]); err != nil {
					return err
				}
			}
			return nil
		})
		return task, err
	}

	// Enqueue some tasks
	_ = enqueue(Task{ID: "t1", Type: "build"})
	_ = enqueue(Task{ID: "t2", Type: "test"})

	// Dequeue first task
	task, _ := dequeue()
	_ = task // Use the variable
	// Output example (actual task may vary in test runs)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
