package participant

import (
	"time"

	"github.com/google/uuid"

	"github.com/cyper98/zenc-server/pkg/protocol"
)

// New creates a new participant
func New(roomID, userID string, role protocol.RoomRole) *protocol.Participant {
	return &protocol.Participant{
		ID:           uuid.New().String(),
		RoomID:       roomID,
		UserID:       userID,
		Role:         role,
		Tracks:       make([]protocol.Track, 0),
		ConnectionID: "",
		JoinedAt:     time.Now(),
		State:        protocol.StateActive,
		HoldRooms:    make([]string, 0),
		MaxHoldSlots: 3,
	}
}
