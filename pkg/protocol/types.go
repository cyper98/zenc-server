package protocol

import "time"

// RoomRole represents the role of a participant in a room
type RoomRole string

const (
	RolePublisher  RoomRole = "publisher"
	RoleSubscriber RoomRole = "subscriber"
	RoleModerator  RoomRole = "moderator"
)

// TrackKind represents the type of media track
type TrackKind string

const (
	TrackKindAudio TrackKind = "audio"
	TrackKindVideo TrackKind = "video"
)

// TrackSource represents the source of a media track
type TrackSource string

const (
	TrackSourceCamera     TrackSource = "camera"
	TrackSourceScreen     TrackSource = "screen"
	TrackSourceMicrophone TrackSource = "microphone"
)

// TrackStatus represents the current status of a track
type TrackStatus string

const (
	TrackStatusActive  TrackStatus = "active"
	TrackStatusPaused  TrackStatus = "paused"
	TrackStatusStopped TrackStatus = "stopped"
)

// ParticipantState represents the state of a participant
type ParticipantState string

const (
	StateActive    ParticipantState = "active"
	StateOnHold    ParticipantState = "on_hold"
	StateInTransit ParticipantState = "in_transit"
)

// Room represents a streaming session
type Room struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	MaxUsers  int                    `json:"max_users"`
	CreatedAt time.Time              `json:"created_at"`
	Zone      string                 `json:"zone"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Participant represents a user in a room
type Participant struct {
	ID           string           `json:"id"`
	RoomID       string           `json:"room_id"`
	UserID       string           `json:"user_id"`
	Role         RoomRole         `json:"role"`
	Tracks       []Track          `json:"tracks"`
	ConnectionID string           `json:"connection_id"`
	JoinedAt     time.Time        `json:"joined_at"`
	State        ParticipantState `json:"state"`
	HoldRooms    []string         `json:"hold_rooms,omitempty"`
	MaxHoldSlots int              `json:"max_hold_slots"`
}

// Track represents an audio or video track
type Track struct {
	ID     string      `json:"id"`
	Kind   TrackKind   `json:"kind"`
	Source TrackSource `json:"source"`
	Status TrackStatus `json:"status"`
}

// QualityLayer represents a quality level for adaptive bitrate
type QualityLayer struct {
	Quality   string `json:"quality"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Bitrate   int    `json:"bitrate"`
	FrameRate int    `json:"frame_rate"`
}

// CreateRoomRequest represents a request to create a room
type CreateRoomRequest struct {
	Name     string                 `json:"name" binding:"required"`
	MaxUsers int                    `json:"max_users"`
	Zone     string                 `json:"zone"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// JoinRoomRequest represents a request to join a room
type JoinRoomRequest struct {
	UserID   string                 `json:"user_id" binding:"required"`
	Role     RoomRole               `json:"role"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// PublishTrackRequest represents a request to publish a track
type PublishTrackRequest struct {
	ParticipantID string      `json:"participant_id" binding:"required"`
	Kind          TrackKind   `json:"kind" binding:"required"`
	Source        TrackSource `json:"source" binding:"required"`
	SDPOffer      string      `json:"sdp_offer" binding:"required"`
}

// SubscribeTrackRequest represents a request to subscribe to a track
type SubscribeTrackRequest struct {
	ParticipantID string `json:"participant_id" binding:"required"`
	TrackID       string `json:"track_id" binding:"required"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
