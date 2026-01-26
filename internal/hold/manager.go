package hold

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aminofox/zenc-server/pkg/logger"
	"github.com/aminofox/zenc-server/pkg/protocol"
)

// HoldState represents the state of a held room
type HoldState struct {
	ParticipantID string
	OriginalRoom  string
	HoldTime      time.Time
	Tracks        []protocol.Track
	State         protocol.ParticipantState
}

// Config contains configuration for hold manager
type Config struct {
	MaxConcurrentHolds int
	HoldTimeout        time.Duration
}

// DefaultConfig returns default hold configuration
func DefaultConfig() Config {
	return Config{
		MaxConcurrentHolds: 3,
		HoldTimeout:        3600 * time.Second, // 1 hour
	}
}

// Manager manages held rooms and participant states
type Manager struct {
	config           Config
	heldParticipants map[string]map[string]*HoldState // participantID -> roomID -> HoldState
	logger           *logger.Logger
	mu               sync.RWMutex
}

// NewManager creates a new hold manager
func NewManager(config Config, logger *logger.Logger) *Manager {
	m := &Manager{
		config:           config,
		heldParticipants: make(map[string]map[string]*HoldState),
		logger:           logger,
	}

	// Start background cleanup of expired holds
	go m.cleanupExpiredHolds()

	return m
}

// HoldRoom puts a room on hold for a participant
func (m *Manager) HoldRoom(participantID, roomID string, tracks []protocol.Track) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize participant's hold map if doesn't exist
	if _, exists := m.heldParticipants[participantID]; !exists {
		m.heldParticipants[participantID] = make(map[string]*HoldState)
	}

	// Check if participant can hold more rooms
	if len(m.heldParticipants[participantID]) >= m.config.MaxConcurrentHolds {
		return fmt.Errorf("max concurrent holds reached (%d)", m.config.MaxConcurrentHolds)
	}

	// Create hold state
	holdState := &HoldState{
		ParticipantID: participantID,
		OriginalRoom:  roomID,
		HoldTime:      time.Now(),
		Tracks:        tracks,
	}

	m.heldParticipants[participantID][roomID] = holdState

	m.logger.Infow("Room put on hold",
		"participant_id", participantID,
		"room_id", roomID,
		"tracks_count", len(tracks),
	)

	return nil
}

// ResumeRoom resumes a held room for a participant
func (m *Manager) ResumeRoom(participantID, roomID string) (*HoldState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	holds, exists := m.heldParticipants[participantID]
	if !exists {
		return nil, errors.New("no held rooms for participant")
	}

	holdState, exists := holds[roomID]
	if !exists {
		return nil, fmt.Errorf("room not on hold: %s", roomID)
	}

	// Remove from holds
	delete(m.heldParticipants[participantID], roomID)

	// Clean up empty participant map
	if len(m.heldParticipants[participantID]) == 0 {
		delete(m.heldParticipants, participantID)
	}

	m.logger.Infow("Room resumed from hold",
		"participant_id", participantID,
		"room_id", roomID,
	)

	return holdState, nil
}

// CanHoldMore checks if a participant can hold more rooms
func (m *Manager) CanHoldMore(participantID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	holds, exists := m.heldParticipants[participantID]
	if !exists {
		return true
	}

	return len(holds) < m.config.MaxConcurrentHolds
}

// GetHeldRooms returns all rooms on hold for a participant
func (m *Manager) GetHeldRooms(participantID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	holds, exists := m.heldParticipants[participantID]
	if !exists {
		return []string{}
	}

	roomIDs := make([]string, 0, len(holds))
	for roomID := range holds {
		roomIDs = append(roomIDs, roomID)
	}

	return roomIDs
}

// GetHoldState retrieves the hold state for a specific room
func (m *Manager) GetHoldState(participantID, roomID string) (*HoldState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	holds, exists := m.heldParticipants[participantID]
	if !exists {
		return nil, errors.New("no held rooms for participant")
	}

	holdState, exists := holds[roomID]
	if !exists {
		return nil, fmt.Errorf("room not on hold: %s", roomID)
	}

	return holdState, nil
}

// RemoveAllHolds removes all held rooms for a participant
func (m *Manager) RemoveAllHolds(participantID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.heldParticipants, participantID)

	m.logger.Infow("All held rooms removed", "participant_id", participantID)
}

// cleanupExpiredHolds periodically removes expired holds
func (m *Manager) cleanupExpiredHolds() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()

		now := time.Now()
		for participantID, holds := range m.heldParticipants {
			for roomID, holdState := range holds {
				if now.Sub(holdState.HoldTime) > m.config.HoldTimeout {
					m.logger.Infow("Removing expired hold",
						"participant_id", participantID,
						"room_id", roomID,
						"hold_duration", now.Sub(holdState.HoldTime),
					)

					delete(holds, roomID)
				}
			}

			// Clean up empty participant map
			if len(holds) == 0 {
				delete(m.heldParticipants, participantID)
			}
		}

		m.mu.Unlock()
	}
}

// GetStats returns statistics about held rooms
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalHolds := 0
	for _, holds := range m.heldParticipants {
		totalHolds += len(holds)
	}

	return map[string]interface{}{
		"participants_with_holds": len(m.heldParticipants),
		"total_held_rooms":        totalHolds,
		"max_concurrent_holds":    m.config.MaxConcurrentHolds,
	}
}
