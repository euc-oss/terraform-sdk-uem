package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// TestRecords keeps track of resources created during tests
// This ensures we only delete resources we created, not existing ones.
type TestRecords struct {
	mu           sync.RWMutex
	Profiles     []int    `json:"profiles"`
	Scripts      []string `json:"scripts"` // Scripts use UUID
	Sensors      []string `json:"sensors"` // Sensors use UUID
	Baselines    []int    `json:"baselines"`
	Applications []int    `json:"applications"`
}

var (
	testRecords     *TestRecords
	testRecordsOnce sync.Once
	recordsFile     = filepath.Join("tests", "test_records.json")
)

// GetTestRecords returns the singleton test records instance.
func GetTestRecords() *TestRecords {
	testRecordsOnce.Do(func() {
		testRecords = &TestRecords{
			Profiles:     make([]int, 0),
			Scripts:      make([]string, 0),
			Sensors:      make([]string, 0),
			Baselines:    make([]int, 0),
			Applications: make([]int, 0),
		}
		// Try to load existing records
		// Check the error returned by Load() but don't fail if it's just a missing file
		if err := testRecords.Load(); err != nil && !os.IsNotExist(err) {
			panic(err)
		}
	})
	return testRecords
}

// AddProfile records a profile ID that was created during testing.
func (tr *TestRecords) AddProfile(profileID int) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.Profiles = append(tr.Profiles, profileID)
	return tr.save()
}

// RemoveProfile removes a profile ID from the records.
func (tr *TestRecords) RemoveProfile(profileID int) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	for i, id := range tr.Profiles {
		if id == profileID {
			tr.Profiles = append(tr.Profiles[:i], tr.Profiles[i+1:]...)
			break
		}
	}
	return tr.save()
}

// IsProfileTracked checks if a profile ID is tracked.
func (tr *TestRecords) IsProfileTracked(profileID int) bool {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	for _, id := range tr.Profiles {
		if id == profileID {
			return true
		}
	}
	return false
}

// AddScript records a script UUID that was created during testing.
func (tr *TestRecords) AddScript(scriptUUID string) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.Scripts = append(tr.Scripts, scriptUUID)
	return tr.save()
}

// RemoveScript removes a script UUID from the records.
func (tr *TestRecords) RemoveScript(scriptUUID string) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	for i, uuid := range tr.Scripts {
		if uuid == scriptUUID {
			tr.Scripts = append(tr.Scripts[:i], tr.Scripts[i+1:]...)
			break
		}
	}
	return tr.save()
}

// Load reads test records from disk.
func (tr *TestRecords) Load() error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	// Clean path to prevent directory traversal (gosec G304)
	cleanPath := filepath.Clean(recordsFile)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's OK
		}
		return err
	}

	return json.Unmarshal(data, tr)
}

// save writes test records to disk (internal, assumes lock is held).
func (tr *TestRecords) save() error {
	data, err := json.MarshalIndent(tr, "", "  ")
	if err != nil {
		return err
	}

	// Clean path and use restrictive permissions (gosec G304, G306)
	cleanPath := filepath.Clean(recordsFile)
	return os.WriteFile(cleanPath, data, 0600)
}

// Clear removes all tracked records and deletes the file.
func (tr *TestRecords) Clear() error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.Profiles = make([]int, 0)
	tr.Scripts = make([]string, 0)
	tr.Sensors = make([]string, 0)
	tr.Baselines = make([]int, 0)
	tr.Applications = make([]int, 0)

	// Remove the file
	if err := os.Remove(recordsFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
