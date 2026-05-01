package mockserver

import (
	"fmt"
	"sync"
	"time"
)

// ServerState manages the stateful behavior of the mock server.
// It tracks created, updated, and deleted resources to provide realistic CRUD behavior.
type ServerState struct {
	mu              sync.RWMutex
	profiles        map[int]*ProfileState // Profile ID -> Profile data
	deletedProfiles map[int]bool          // Track deleted profile IDs
}

// NewServerState creates a new server state manager.
func NewServerState() *ServerState {
	return &ServerState{
		profiles:        make(map[int]*ProfileState),
		deletedProfiles: make(map[int]bool),
	}
}

// CreateProfile stores a newly created profile in state.
func (s *ServerState) CreateProfile(id int, name, description, platform string, general, payloads map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	s.profiles[id] = &ProfileState{
		ID:          id,
		Name:        name,
		Description: description,
		Platform:    platform,
		Status:      "Active",
		General:     general,
		Payloads:    payloads,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// GetProfile retrieves a profile from state.
// Returns nil if the profile doesn't exist or has been deleted.
func (s *ServerState) GetProfile(id int) *ProfileState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if profile was deleted
	if s.deletedProfiles[id] {
		return nil
	}

	return s.profiles[id]
}

// UpdateProfile updates an existing profile in state.
// Returns error if the profile doesn't exist.
func (s *ServerState) UpdateProfile(id int, name, description string, general, payloads map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, exists := s.profiles[id]
	if !exists {
		return fmt.Errorf("profile %d not found", id)
	}

	// Update fields
	if name != "" {
		profile.Name = name
	}
	if description != "" {
		profile.Description = description
	}
	if general != nil {
		profile.General = general
	}
	if payloads != nil {
		profile.Payloads = payloads
	}
	profile.UpdatedAt = time.Now().Format(time.RFC3339)

	return nil
}

// DeleteProfile removes a profile from state and marks it as deleted.
func (s *ServerState) DeleteProfile(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.profiles, id)
	s.deletedProfiles[id] = true
}

// ProfileExists checks if a profile exists in state.
func (s *ServerState) ProfileExists(id int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.profiles[id]
	return exists
}

// Clear removes all state (useful for testing).
func (s *ServerState) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.profiles = make(map[int]*ProfileState)
	s.deletedProfiles = make(map[int]bool)
}
