package beads

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestConcurrentRoutes verifies that concurrent route operations are safe.
func TestConcurrentRoutes(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Number of concurrent goroutines
	const numGoroutines = 10
	const numOpsPerGoroutine = 20

	// WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Error channel to collect errors
	errCh := make(chan error, numGoroutines*numOpsPerGoroutine)

	// Barrier to start all goroutines at once
	startCh := make(chan struct{})

	// Launch concurrent workers
	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer wg.Done()

			// Wait for start signal
			<-startCh

			// Perform multiple operations
			for j := 0; j < numOpsPerGoroutine; j++ {
				// Use unique prefix for each operation to avoid collisions
				prefix := fmt.Sprintf("w%d-op%d-", workerID, j)
				path := fmt.Sprintf("test/%d/%d", workerID, j)

				// Append route
				route := Route{
					Prefix: prefix,
					Path:   path,
				}
				if err := AppendRouteToDir(beadsDir, route); err != nil {
					errCh <- fmt.Errorf("worker %d append: %w", workerID, err)
					return
				}

				// Read routes - may return empty slice if file doesn't exist yet
				_, err := LoadRoutes(beadsDir)
				if err != nil {
					errCh <- fmt.Errorf("worker %d load: %w", workerID, err)
					return
				}

				// Just verify we can read without error
				// The actual routes may vary due to concurrent writes
			}
		}(i)
	}

	// Start all goroutines
	close(startCh)

	// Wait for completion
	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		t.Error(err)
	}

	// Verify final routes file is valid
	routes, err := LoadRoutes(beadsDir)
	if err != nil {
		t.Fatalf("loading final routes: %v", err)
	}

	// We should have at least some routes (exact count may vary due to concurrent updates)
	if len(routes) == 0 {
		t.Error("expected some routes, got none")
	}

	// Verify all routes are valid
	for _, r := range routes {
		if r.Prefix == "" || r.Path == "" {
			t.Errorf("invalid route: %+v", r)
		}
	}

	t.Logf("Successfully created %d routes with concurrent writes", len(routes))
}

// TestConcurrentCatalog verifies that concurrent catalog operations are safe.
func TestConcurrentCatalog(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "molecules.jsonl")

	// Number of concurrent readers and writers
	const numReaders = 5
	const numWriters = 2
	const numOps = 10

	// Create initial catalog
	catalog := NewMoleculeCatalog()
	for i := 0; i < 10; i++ {
		catalog.Add(&CatalogMolecule{
			ID:          fmt.Sprintf("mol-%d", i),
			Title:       fmt.Sprintf("Molecule %d", i),
			Description: fmt.Sprintf("Description %d", i),
		})
	}
	if err := catalog.SaveToFile(catalogPath); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, (numReaders+numWriters)*numOps)
	startCh := make(chan struct{})

	// Launch readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			<-startCh

			for j := 0; j < numOps; j++ {
				cat := NewMoleculeCatalog()
				if err := cat.LoadFromFile(catalogPath, "test"); err != nil {
					errCh <- fmt.Errorf("reader %d: %w", readerID, err)
					return
				}

				// Note: catalog may be empty if a writer is in the middle of writing
				// Just verify we can read without error

				// Small delay to increase concurrency
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	// Launch writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			<-startCh

			for j := 0; j < numOps; j++ {
				// Create new catalog with writer-specific molecules
				cat := NewMoleculeCatalog()
				for k := 0; k < 5; k++ {
					cat.Add(&CatalogMolecule{
						ID:          fmt.Sprintf("w%d-mol-%d", writerID, k),
						Title:       fmt.Sprintf("Writer %d Molecule %d", writerID, k),
						Description: fmt.Sprintf("Description from writer %d", writerID),
					})
				}

				if err := cat.SaveToFile(catalogPath); err != nil {
					errCh <- fmt.Errorf("writer %d: %w", writerID, err)
					return
				}

				// Small delay
				time.Sleep(2 * time.Millisecond)
			}
		}(i)
	}

	// Start all goroutines
	close(startCh)

	// Wait for completion
	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		t.Error(err)
	}

	// Verify final catalog is valid (not corrupted)
	finalCat := NewMoleculeCatalog()
	if err := finalCat.LoadFromFile(catalogPath, "test"); err != nil {
		t.Fatalf("loading final catalog: %v", err)
	}

	// Should have at least some molecules
	if finalCat.Count() == 0 {
		t.Error("final catalog is empty")
	}
}

// TestConcurrentAuditLog verifies that concurrent audit log writes don't corrupt the file.
func TestConcurrentAuditLog(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create .beads directory
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a Beads instance
	b := NewIsolated(tmpDir)

	const numWorkers = 10
	const numLogsPerWorker = 20

	var wg sync.WaitGroup
	errCh := make(chan error, numWorkers*numLogsPerWorker)
	startCh := make(chan struct{})

	// Launch concurrent audit loggers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startCh

			for j := 0; j < numLogsPerWorker; j++ {
				entry := DetachAuditEntry{
					Timestamp:        time.Now().Format(time.RFC3339),
					Operation:        "test",
					PinnedBeadID:     fmt.Sprintf("bead-%d-%d", workerID, j),
					DetachedMolecule: fmt.Sprintf("mol-%d-%d", workerID, j),
					DetachedBy:       fmt.Sprintf("worker-%d", workerID),
					Reason:           fmt.Sprintf("Test operation %d", j),
				}

				if err := b.LogDetachAudit(entry); err != nil {
					errCh <- fmt.Errorf("worker %d: %w", workerID, err)
					return
				}
			}
		}(i)
	}

	// Start all goroutines
	close(startCh)

	// Wait for completion
	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		t.Error(err)
	}

	// Verify audit log is valid JSONL
	auditPath := filepath.Join(tmpDir, ".beads", "audit.log")
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("reading audit log: %v", err)
	}

	// Count lines
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}

	expectedLines := numWorkers * numLogsPerWorker
	if lines != expectedLines {
		t.Errorf("expected %d log lines, got %d", expectedLines, lines)
	}
}

// TestConcurrentProvisionPrimeMD verifies that concurrent provisioning is safe.
func TestConcurrentProvisionPrimeMD(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}

	const numWorkers = 10

	var wg sync.WaitGroup
	errCh := make(chan error, numWorkers)
	startCh := make(chan struct{})

	// Launch concurrent provisioners
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startCh

			if err := ProvisionPrimeMD(beadsDir); err != nil {
				errCh <- fmt.Errorf("worker %d: %w", workerID, err)
			}
		}(i)
	}

	// Start all goroutines
	close(startCh)

	// Wait for completion
	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		t.Error(err)
	}

	// Verify PRIME.md exists and is valid
	primePath := filepath.Join(beadsDir, "PRIME.md")
	data, err := os.ReadFile(primePath)
	if err != nil {
		t.Fatalf("reading PRIME.md: %v", err)
	}

	if len(data) == 0 {
		t.Error("PRIME.md is empty")
	}

	// Should contain expected content
	content := string(data)
	if content != primeContent {
		t.Error("PRIME.md content mismatch")
	}
}

// TestRedirectWithConcurrency verifies redirect operations are safe under concurrent access.
func TestRedirectWithConcurrency(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	townRoot := tmpDir
	rigPath := filepath.Join(townRoot, "testrig")
	worktree1 := filepath.Join(rigPath, "crew", "worker1")
	worktree2 := filepath.Join(rigPath, "crew", "worker2")

	// Create directories
	for _, dir := range []string{rigPath, worktree1, worktree2} {
		if err := os.MkdirAll(filepath.Join(dir, ".beads"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	startCh := make(chan struct{})

	// Concurrently setup redirects
	for _, wt := range []string{worktree1, worktree2} {
		wg.Add(1)
		go func(worktree string) {
			defer wg.Done()
			<-startCh

			if err := SetupRedirect(townRoot, worktree); err != nil {
				errCh <- fmt.Errorf("setup redirect: %w", err)
			}
		}(wt)
	}

	// Start all goroutines
	close(startCh)

	// Wait for completion
	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		t.Error(err)
	}

	// Verify redirects were created
	for _, wt := range []string{worktree1, worktree2} {
		redirectPath := filepath.Join(wt, ".beads", "redirect")
		if _, err := os.Stat(redirectPath); err != nil {
			t.Errorf("redirect not created for %s: %v", wt, err)
		}
	}

	// Concurrently resolve redirects
	wg.Add(2)
	errCh = make(chan error, 2)
	startCh = make(chan struct{})

	for _, wt := range []string{worktree1, worktree2} {
		go func(worktree string) {
			defer wg.Done()
			<-startCh

			// Resolve redirect multiple times
			for i := 0; i < 10; i++ {
				resolved := ResolveBeadsDir(worktree)
				if resolved == "" {
					errCh <- fmt.Errorf("resolved path is empty")
					return
				}
			}
		}(wt)
	}

	// Start all goroutines
	close(startCh)

	// Wait for completion
	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		t.Error(err)
	}
}
