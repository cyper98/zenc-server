package media

import (
	"sync"

	"github.com/google/uuid"

	"github.com/cyper98/zenc-server/pkg/protocol"
)

// Router handles media packet routing and track lifecycle across rooms (SFU core).
type Router struct {
	mu           sync.RWMutex
	activeTracks map[string]*protocol.Track            // trackID -> Track
	roomTracks   map[string]map[string]*protocol.Track // roomID -> trackID -> Track
}

// NewRouter creates a new media router
func NewRouter() *Router {
	return &Router{
		activeTracks: make(map[string]*protocol.Track),
		roomTracks:   make(map[string]map[string]*protocol.Track),
	}
}

// RegisterTrack registers an active track for routing in a room
func (r *Router) RegisterTrack(roomID string, track *protocol.Track) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.activeTracks[track.ID] = track
	if _, ok := r.roomTracks[roomID]; !ok {
		r.roomTracks[roomID] = make(map[string]*protocol.Track)
	}
	r.roomTracks[roomID][track.ID] = track
}

// UnregisterTrack removes a track from routing
func (r *Router) UnregisterTrack(roomID, trackID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.activeTracks, trackID)
	if tracks, ok := r.roomTracks[roomID]; ok {
		delete(tracks, trackID)
		if len(tracks) == 0 {
			delete(r.roomTracks, roomID)
		}
	}
}

// GetRoomTracks returns all active tracks in a room
func (r *Router) GetRoomTracks(roomID string) []*protocol.Track {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tracks, ok := r.roomTracks[roomID]
	if !ok {
		return nil
	}

	result := make([]*protocol.Track, 0, len(tracks))
	for _, t := range tracks {
		result = append(result, t)
	}
	return result
}

// NewTrack creates a new media track
func NewTrack(kind protocol.TrackKind, source protocol.TrackSource) *protocol.Track {
	return &protocol.Track{
		ID:     uuid.New().String(),
		Kind:   kind,
		Source: source,
		Status: protocol.TrackStatusActive,
	}
}
