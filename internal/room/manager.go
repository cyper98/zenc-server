package room

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aminofox/zenc-server/internal/store"
	"github.com/aminofox/zenc-server/pkg/logger"
	"github.com/aminofox/zenc-server/pkg/protocol"
	"github.com/google/uuid"
)

// Manager handles room operations
type Manager struct {
	store  *store.RedisStore
	logger *logger.Logger
	rooms  sync.Map // In-memory cache of active rooms
	mu     sync.RWMutex
}

// NewManager creates a new room manager
func NewManager(store *store.RedisStore, logger *logger.Logger) *Manager {
	return &Manager{
		store:  store,
		logger: logger,
	}
}

// CreateRoom creates a new room
func (m *Manager) CreateRoom(req protocol.CreateRoomRequest) (*protocol.Room, error) {
	room := &protocol.Room{
		ID:        uuid.New().String(),
		Name:      req.Name,
		MaxUsers:  req.MaxUsers,
		CreatedAt: time.Now(),
		Zone:      req.Zone,
		Metadata:  req.Metadata,
	}

	if room.MaxUsers == 0 {
		room.MaxUsers = 100 // Default max users
	}

	if room.Zone == "" {
		room.Zone = "default"
	}

	// Store room in Redis
	roomKey := fmt.Sprintf("room:%s", room.ID)
	roomData, err := json.Marshal(room)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal room: %w", err)
	}

	if err := m.store.Set(roomKey, roomData, 0); err != nil {
		return nil, fmt.Errorf("failed to store room: %w", err)
	}

	// Add to active rooms set
	if err := m.store.SAdd("rooms:active", room.ID); err != nil {
		return nil, fmt.Errorf("failed to add room to active set: %w", err)
	}

	// Cache in memory
	m.rooms.Store(room.ID, room)

	m.logger.Infow("Room created",
		"room_id", room.ID,
		"name", room.Name,
		"zone", room.Zone,
	)

	return room, nil
}

// GetRoom retrieves a room by ID
func (m *Manager) GetRoom(roomID string) (*protocol.Room, error) {
	// Check memory cache first
	if cached, ok := m.rooms.Load(roomID); ok {
		return cached.(*protocol.Room), nil
	}

	// Fetch from Redis
	roomKey := fmt.Sprintf("room:%s", roomID)
	roomData, err := m.store.Get(roomKey)
	if err != nil {
		return nil, fmt.Errorf("room not found: %w", err)
	}

	var room protocol.Room
	if err := json.Unmarshal([]byte(roomData), &room); err != nil {
		return nil, fmt.Errorf("failed to unmarshal room: %w", err)
	}

	// Cache in memory
	m.rooms.Store(roomID, &room)

	return &room, nil
}

// DeleteRoom deletes a room
func (m *Manager) DeleteRoom(roomID string) error {
	// Remove from Redis
	roomKey := fmt.Sprintf("room:%s", roomID)
	if err := m.store.Delete(roomKey); err != nil {
		return fmt.Errorf("failed to delete room: %w", err)
	}

	// Remove from active rooms set
	if err := m.store.SRem("rooms:active", roomID); err != nil {
		return fmt.Errorf("failed to remove room from active set: %w", err)
	}

	// Remove participants
	participantsKey := fmt.Sprintf("room:%s:participants", roomID)
	if err := m.store.Delete(participantsKey); err != nil {
		m.logger.Warnw("Failed to delete room participants", "error", err)
	}

	// Remove from memory cache
	m.rooms.Delete(roomID)

	m.logger.Infow("Room deleted", "room_id", roomID)

	return nil
}

// ListRooms returns all active rooms
func (m *Manager) ListRooms() ([]*protocol.Room, error) {
	roomIDs, err := m.store.SMembers("rooms:active")
	if err != nil {
		return nil, fmt.Errorf("failed to list rooms: %w", err)
	}

	rooms := make([]*protocol.Room, 0, len(roomIDs))
	for _, roomID := range roomIDs {
		room, err := m.GetRoom(roomID)
		if err != nil {
			m.logger.Warnw("Failed to get room", "room_id", roomID, "error", err)
			continue
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// AddParticipant adds a participant to a room
func (m *Manager) AddParticipant(roomID string, participant *protocol.Participant) error {
	// Verify room exists
	room, err := m.GetRoom(roomID)
	if err != nil {
		return fmt.Errorf("room not found: %w", err)
	}

	// Check room capacity
	currentCount, err := m.GetParticipantCount(roomID)
	if err != nil {
		return fmt.Errorf("failed to get participant count: %w", err)
	}

	if currentCount >= room.MaxUsers {
		return errors.New("room is full")
	}

	// Store participant
	participantKey := fmt.Sprintf("participant:%s", participant.ID)
	participantData, err := json.Marshal(participant)
	if err != nil {
		return fmt.Errorf("failed to marshal participant: %w", err)
	}

	if err := m.store.Set(participantKey, participantData, 24*time.Hour); err != nil {
		return fmt.Errorf("failed to store participant: %w", err)
	}

	// Add to room's participant set
	participantsKey := fmt.Sprintf("room:%s:participants", roomID)
	if err := m.store.SAdd(participantsKey, participant.ID); err != nil {
		return fmt.Errorf("failed to add participant to room: %w", err)
	}

	m.logger.Infow("Participant added to room",
		"room_id", roomID,
		"participant_id", participant.ID,
		"user_id", participant.UserID,
	)

	return nil
}

// RemoveParticipant removes a participant from a room
func (m *Manager) RemoveParticipant(roomID, participantID string) error {
	// Remove from room's participant set
	participantsKey := fmt.Sprintf("room:%s:participants", roomID)
	if err := m.store.SRem(participantsKey, participantID); err != nil {
		return fmt.Errorf("failed to remove participant from room: %w", err)
	}

	// Delete participant data
	participantKey := fmt.Sprintf("participant:%s", participantID)
	if err := m.store.Delete(participantKey); err != nil {
		m.logger.Warnw("Failed to delete participant data", "error", err)
	}

	m.logger.Infow("Participant removed from room",
		"room_id", roomID,
		"participant_id", participantID,
	)

	return nil
}

// GetParticipants returns all participants in a room
func (m *Manager) GetParticipants(roomID string) ([]*protocol.Participant, error) {
	participantsKey := fmt.Sprintf("room:%s:participants", roomID)
	participantIDs, err := m.store.SMembers(participantsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	participants := make([]*protocol.Participant, 0, len(participantIDs))
	for _, participantID := range participantIDs {
		participantKey := fmt.Sprintf("participant:%s", participantID)
		participantData, err := m.store.Get(participantKey)
		if err != nil {
			m.logger.Warnw("Failed to get participant", "participant_id", participantID, "error", err)
			continue
		}

		var participant protocol.Participant
		if err := json.Unmarshal([]byte(participantData), &participant); err != nil {
			m.logger.Warnw("Failed to unmarshal participant", "participant_id", participantID, "error", err)
			continue
		}

		participants = append(participants, &participant)
	}

	return participants, nil
}

// GetParticipantCount returns the number of participants in a room
func (m *Manager) GetParticipantCount(roomID string) (int, error) {
	participants, err := m.GetParticipants(roomID)
	if err != nil {
		return 0, err
	}
	return len(participants), nil
}

// RoomTrackStats contains statistics about tracks in a room
type RoomTrackStats struct {
	ActiveScreenShares int
	MaxScreenShares    int
}

// CanPublishScreenShare checks if a screen share can be published in a room
func (m *Manager) CanPublishScreenShare(roomID string) error {
	stats, err := m.GetRoomTrackStats(roomID)
	if err != nil {
		return err
	}

	if stats.ActiveScreenShares >= stats.MaxScreenShares {
		return errors.New("maximum screen shares reached for this room")
	}

	return nil
}

// GetRoomTrackStats returns track statistics for a room
func (m *Manager) GetRoomTrackStats(roomID string) (*RoomTrackStats, error) {
	participants, err := m.GetParticipants(roomID)
	if err != nil {
		return nil, err
	}

	screenShares := 0
	for _, p := range participants {
		for _, track := range p.Tracks {
			if track.Source == protocol.TrackSourceScreen &&
				track.Status == protocol.TrackStatusActive {
				screenShares++
			}
		}
	}

	room, _ := m.GetRoom(roomID)
	maxScreenShares := 5 // Default

	// Check metadata for custom limit
	if room.Metadata != nil {
		if val, ok := room.Metadata["max_screen_shares"]; ok {
			if max, ok := val.(float64); ok {
				maxScreenShares = int(max)
			}
		}
	}

	return &RoomTrackStats{
		ActiveScreenShares: screenShares,
		MaxScreenShares:    maxScreenShares,
	}, nil
}

// SetParticipantState updates the state of a participant
func (m *Manager) SetParticipantState(roomID, participantID string, state protocol.ParticipantState) error {
	participantKey := fmt.Sprintf("participant:%s", participantID)

	// Get current participant data
	participantData, err := m.store.Get(participantKey)
	if err != nil {
		return fmt.Errorf("participant not found: %w", err)
	}

	var participant protocol.Participant
	if err := json.Unmarshal([]byte(participantData), &participant); err != nil {
		return fmt.Errorf("failed to unmarshal participant: %w", err)
	}

	// Update state
	participant.State = state

	// Save back to Redis
	updatedData, err := json.Marshal(participant)
	if err != nil {
		return fmt.Errorf("failed to marshal participant: %w", err)
	}

	if err := m.store.Set(participantKey, string(updatedData), 0); err != nil {
		return fmt.Errorf("failed to save participant state: %w", err)
	}

	m.logger.Infow("Participant state updated",
		"participant_id", participantID,
		"room_id", roomID,
		"state", state,
	)

	return nil
}

// GetParticipantState gets the current state of a participant
func (m *Manager) GetParticipantState(roomID, participantID string) (protocol.ParticipantState, error) {
	participantKey := fmt.Sprintf("participant:%s", participantID)

	participantData, err := m.store.Get(participantKey)
	if err != nil {
		return "", fmt.Errorf("participant not found: %w", err)
	}

	var participant protocol.Participant
	if err := json.Unmarshal([]byte(participantData), &participant); err != nil {
		return "", fmt.Errorf("failed to unmarshal participant: %w", err)
	}

	return participant.State, nil
}
