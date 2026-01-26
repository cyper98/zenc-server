package media

import (
	"github.com/google/uuid"

	"github.com/aminofox/zenc-server/pkg/protocol"
)

// Router handles media packet routing (SFU core)
// This is a placeholder implementation for the MVP
type Router struct {
	// Will be implemented with Pion WebRTC in future iterations
}

// NewRouter creates a new media router
func NewRouter() *Router {
	return &Router{}
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
