package track

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cyper98/zenc-server/pkg/logger"
	"github.com/cyper98/zenc-server/pkg/protocol"
)

// TrackLimits defines limits for different track types
type TrackLimits struct {
	MaxCameraTracks int
	MaxScreenTracks int
	MaxAudioTracks  int
}

// DefaultTrackLimits returns default track limits
func DefaultTrackLimits() TrackLimits {
	return TrackLimits{
		MaxCameraTracks: 1,
		MaxScreenTracks: 1,
		MaxAudioTracks:  2,
	}
}

// Manager handles track management and enforcement of limits
type Manager struct {
	participantTracks map[string]map[protocol.TrackSource][]protocol.Track
	limits            TrackLimits
	logger            *logger.Logger
	mu                sync.RWMutex
}

// NewManager creates a new track manager
func NewManager(limits TrackLimits, logger *logger.Logger) *Manager {
	return &Manager{
		participantTracks: make(map[string]map[protocol.TrackSource][]protocol.Track),
		limits:            limits,
		logger:            logger,
	}
}

// CanPublishTrack checks if a participant can publish a new track
func (tm *Manager) CanPublishTrack(participantID string, source protocol.TrackSource) error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tracks, exists := tm.participantTracks[participantID]
	if !exists {
		return nil // No tracks yet, can publish
	}

	existing := tracks[source]

	switch source {
	case protocol.TrackSourceCamera:
		if len(existing) >= tm.limits.MaxCameraTracks {
			return errors.New("max camera tracks reached")
		}
	case protocol.TrackSourceScreen:
		if len(existing) >= tm.limits.MaxScreenTracks {
			return errors.New("max screen shares reached")
		}
	case protocol.TrackSourceMicrophone:
		if len(existing) >= tm.limits.MaxAudioTracks {
			return errors.New("max audio tracks reached")
		}
	}

	return nil
}

// AddTrack adds a track for a participant
func (tm *Manager) AddTrack(participantID string, track protocol.Track) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check limits before adding
	if err := tm.canPublishTrackUnsafe(participantID, track.Source); err != nil {
		return err
	}

	if _, exists := tm.participantTracks[participantID]; !exists {
		tm.participantTracks[participantID] = make(map[protocol.TrackSource][]protocol.Track)
	}

	tm.participantTracks[participantID][track.Source] = append(
		tm.participantTracks[participantID][track.Source],
		track,
	)

	tm.logger.Infow("Track added",
		"participant_id", participantID,
		"track_id", track.ID,
		"source", track.Source,
		"kind", track.Kind,
	)

	return nil
}

// canPublishTrackUnsafe checks limits without locking (internal use)
func (tm *Manager) canPublishTrackUnsafe(participantID string, source protocol.TrackSource) error {
	tracks, exists := tm.participantTracks[participantID]
	if !exists {
		return nil
	}

	existing := tracks[source]

	switch source {
	case protocol.TrackSourceCamera:
		if len(existing) >= tm.limits.MaxCameraTracks {
			return errors.New("max camera tracks reached")
		}
	case protocol.TrackSourceScreen:
		if len(existing) >= tm.limits.MaxScreenTracks {
			return errors.New("max screen shares reached")
		}
	case protocol.TrackSourceMicrophone:
		if len(existing) >= tm.limits.MaxAudioTracks {
			return errors.New("max audio tracks reached")
		}
	}

	return nil
}

// RemoveTrack removes a specific track for a participant
func (tm *Manager) RemoveTrack(participantID, trackID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tracks, exists := tm.participantTracks[participantID]
	if !exists {
		return fmt.Errorf("participant not found: %s", participantID)
	}

	// Search through all track sources
	for source, trackList := range tracks {
		for i, track := range trackList {
			if track.ID == trackID {
				// Remove track from slice
				tm.participantTracks[participantID][source] = append(
					trackList[:i],
					trackList[i+1:]...,
				)

				tm.logger.Infow("Track removed",
					"participant_id", participantID,
					"track_id", trackID,
					"source", source,
				)

				return nil
			}
		}
	}

	return fmt.Errorf("track not found: %s", trackID)
}

// RemoveAllTracks removes all tracks for a participant
func (tm *Manager) RemoveAllTracks(participantID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.participantTracks, participantID)

	tm.logger.Infow("All tracks removed", "participant_id", participantID)
}

// GetTracks returns all tracks for a participant
func (tm *Manager) GetTracks(participantID string) []protocol.Track {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tracks, exists := tm.participantTracks[participantID]
	if !exists {
		return []protocol.Track{}
	}

	// Flatten all tracks
	var allTracks []protocol.Track
	for _, trackList := range tracks {
		allTracks = append(allTracks, trackList...)
	}

	return allTracks
}

// GetTracksBySource returns tracks of a specific source for a participant
func (tm *Manager) GetTracksBySource(participantID string, source protocol.TrackSource) []protocol.Track {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tracks, exists := tm.participantTracks[participantID]
	if !exists {
		return []protocol.Track{}
	}

	return tracks[source]
}

// UpdateTrackStatus updates the status of a track
func (tm *Manager) UpdateTrackStatus(participantID, trackID string, status protocol.TrackStatus) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tracks, exists := tm.participantTracks[participantID]
	if !exists {
		return fmt.Errorf("participant not found: %s", participantID)
	}

	// Search through all track sources
	for source, trackList := range tracks {
		for i, track := range trackList {
			if track.ID == trackID {
				tm.participantTracks[participantID][source][i].Status = status

				tm.logger.Infow("Track status updated",
					"participant_id", participantID,
					"track_id", trackID,
					"status", status,
				)

				return nil
			}
		}
	}

	return fmt.Errorf("track not found: %s", trackID)
}

// GetStats returns statistics about tracked media
func (tm *Manager) GetStats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	totalTracks := 0
	activeTracks := 0

	for _, tracks := range tm.participantTracks {
		for _, trackList := range tracks {
			for _, track := range trackList {
				totalTracks++
				if track.Status == protocol.TrackStatusActive {
					activeTracks++
				}
			}
		}
	}

	return map[string]interface{}{
		"total_participants": len(tm.participantTracks),
		"total_tracks":       totalTracks,
		"active_tracks":      activeTracks,
	}
}

// RoomTrackStats contains statistics about tracks in a room
type RoomTrackStats struct {
	ActiveScreenShares int
	MaxScreenShares    int
}
