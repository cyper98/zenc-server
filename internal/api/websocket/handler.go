package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cyper98/zenc-server/internal/room"
	"github.com/cyper98/zenc-server/internal/webrtc"
	"github.com/cyper98/zenc-server/pkg/logger"
	"github.com/cyper98/zenc-server/pkg/protocol"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	pionwebrtc "github.com/pion/webrtc/v3"
)

// OriginChecker is a function type for checking WebSocket origins
type OriginChecker func(*http.Request) bool

var originChecker OriginChecker

// SetOriginChecker sets the origin checker function
func SetOriginChecker(checker OriginChecker) {
	originChecker = checker
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		if originChecker != nil {
			return originChecker(r)
		}
		// Default: allow all origins (user should configure this in production)
		return true
	},
}

// Connection represents a WebSocket connection
type Connection struct {
	conn            *websocket.Conn
	send            chan []byte
	roomManager     *room.Manager
	webrtcManager   *webrtc.Manager
	logger          *logger.Logger
	mu              sync.Mutex
	participantID   string
	roomID          string
	disconnectTime  *time.Time
	reconnectConfig ReconnectConfig
}

// ReconnectConfig holds reconnection settings
type ReconnectConfig struct {
	Timeout     time.Duration
	MaxAttempts int
}

// HandleConnection handles WebSocket connections
func HandleConnection(w http.ResponseWriter, r *http.Request, roomManager *room.Manager, webrtcManager *webrtc.Manager, logger *logger.Logger) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Errorw("Failed to upgrade WebSocket connection", "error", err)
		return
	}

	c := &Connection{
		conn:          conn,
		send:          make(chan []byte, 256),
		roomManager:   roomManager,
		webrtcManager: webrtcManager,
		logger:        logger,
	}

	// Start goroutines for reading and writing
	go c.readPump()
	go c.writePump()

	logger.Infow("WebSocket connection established",
		"remote_addr", r.RemoteAddr,
	)
}

// readPump handles incoming WebSocket messages
func (c *Connection) readPump() {
	defer func() {
		c.cleanup()
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Errorw("WebSocket read error", "error", err)
			}
			break
		}

		// Parse message
		var msg protocol.WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.logger.Warnw("Failed to parse WebSocket message", "error", err)
			continue
		}

		c.handleMessage(&msg)
	}
}

// cleanup handles connection cleanup and auto-hold
func (c *Connection) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.roomID != "" && c.participantID != "" {
		// Set participant to in-transit state (waiting for reconnect)
		if err := c.roomManager.SetParticipantState(c.roomID, c.participantID, protocol.StateInTransit); err != nil {
			c.logger.Errorw("Failed to set in-transit state", "error", err)
		} else {
			now := time.Now()
			c.disconnectTime = &now
			c.logger.Infow("Participant disconnected, waiting for reconnect",
				"participant_id", c.participantID,
				"room_id", c.roomID,
				"timeout", c.reconnectConfig.Timeout,
			)

			// Start auto-hold timer
			go c.autoHoldAfterTimeout()
		}
	}
}

// autoHoldAfterTimeout automatically holds participant after timeout
func (c *Connection) autoHoldAfterTimeout() {
	time.Sleep(c.reconnectConfig.Timeout)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if still disconnected
	if c.disconnectTime == nil {
		return // Reconnected already
	}

	state, err := c.roomManager.GetParticipantState(c.roomID, c.participantID)
	if err != nil {
		return
	}

	// If still in-transit, auto-hold
	if state == protocol.StateInTransit {
		if err := c.roomManager.SetParticipantState(c.roomID, c.participantID, protocol.StateOnHold); err != nil {
			c.logger.Errorw("Failed to auto-hold participant", "error", err)
		} else {
			c.logger.Infow("Participant auto-held after disconnect timeout",
				"participant_id", c.participantID,
				"room_id", c.roomID,
			)
		}
	}
}

// writePump handles outgoing WebSocket messages
func (c *Connection) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *Connection) handleMessage(msg *protocol.WebSocketMessage) {
	c.logger.Debugw("Received WebSocket message", "type", msg.Type)

	switch msg.Type {
	case "join":
		c.handleJoin(msg.Payload)
	case "leave":
		c.handleLeave(msg.Payload)
	case "offer":
		c.handleOffer(msg.Payload)
	case "answer":
		c.handleAnswer(msg.Payload)
	case "ice_candidate":
		c.handleICECandidate(msg.Payload)
	default:
		c.logger.Warnw("Unknown WebSocket message type", "type", msg.Type)
	}
}

// handleJoin handles room join requests
func (c *Connection) handleJoin(payload map[string]interface{}) {
	roomID, _ := payload["room_id"].(string)
	userID, _ := payload["user_id"].(string)
	roleStr, _ := payload["role"].(string)

	c.logger.Infow("Handling join request",
		"room_id", roomID,
		"user_id", userID,
	)

	// Verify room exists
	room, err := c.roomManager.GetRoom(roomID)
	if err != nil {
		c.sendError("Room not found", err)
		return
	}

	// Create participant
	participant := &protocol.Participant{
		ID:           uuid.New().String(),
		RoomID:       roomID,
		UserID:       userID,
		Role:         protocol.RoomRole(roleStr),
		Tracks:       []protocol.Track{},
		ConnectionID: c.participantID,
		JoinedAt:     time.Now(),
		State:        protocol.StateActive,
		MaxHoldSlots: 3,
	}

	if participant.Role == "" {
		participant.Role = protocol.RolePublisher
	}

	// Add participant to room
	if err := c.roomManager.AddParticipant(roomID, participant); err != nil {
		c.sendError("Failed to join room", err)
		return
	}

	// Store participant and room IDs
	c.participantID = participant.ID
	c.roomID = roomID

	// Send success response
	response := protocol.WebSocketMessage{
		Type: "joined",
		Payload: map[string]interface{}{
			"room_id":        roomID,
			"participant_id": participant.ID,
			"room_name":      room.Name,
			"status":         "success",
		},
	}

	c.sendMessage(&response)

	c.logger.Infow("Participant joined room",
		"room_id", roomID,
		"participant_id", participant.ID,
		"user_id", userID,
	)
}

// handleLeave handles room leave requests
func (c *Connection) handleLeave(payload map[string]interface{}) {
	c.logger.Infow("Handling leave request",
		"participant_id", c.participantID,
		"room_id", c.roomID,
	)

	if c.roomID == "" || c.participantID == "" {
		c.sendError("Not in a room", nil)
		return
	}

	// Remove WebRTC peer connection if active
	if c.webrtcManager != nil && c.participantID != "" {
		if err := c.webrtcManager.RemovePeerConnection(c.participantID); err != nil {
			c.logger.Warnw("Failed to remove peer connection on leave", "error", err)
		}
	}

	// Remove participant from room
	if err := c.roomManager.RemoveParticipant(c.roomID, c.participantID); err != nil {
		c.logger.Warnw("Failed to remove participant", "error", err)
	}

	// Send success response
	response := protocol.WebSocketMessage{
		Type: "left",
		Payload: map[string]interface{}{
			"room_id":        c.roomID,
			"participant_id": c.participantID,
			"status":         "success",
		},
	}

	c.sendMessage(&response)

	// Clear connection state
	c.roomID = ""
	c.participantID = ""
}

// handleOffer handles WebRTC offer
func (c *Connection) handleOffer(payload map[string]interface{}) {
	sdp, _ := payload["sdp"].(string)

	c.logger.Infow("Handling WebRTC offer", "participant_id", c.participantID)

	if c.participantID == "" {
		c.sendError("Not joined to a room", nil)
		return
	}

	if c.webrtcManager == nil {
		c.sendError("WebRTC manager not configured", nil)
		return
	}

	answerSDP, err := c.webrtcManager.HandleOffer(c.participantID, sdp)
	if err != nil {
		c.logger.Errorw("Failed to process WebRTC offer", "error", err, "participant_id", c.participantID)
		c.sendError("Failed to process WebRTC offer", err)
		return
	}

	response := protocol.WebSocketMessage{
		Type: "answer",
		Payload: map[string]interface{}{
			"sdp":            answerSDP,
			"participant_id": c.participantID,
		},
	}

	c.sendMessage(&response)
}

// handleAnswer handles WebRTC answer
func (c *Connection) handleAnswer(payload map[string]interface{}) {
	sdp, _ := payload["sdp"].(string)

	c.logger.Infow("Handling WebRTC answer",
		"participant_id", c.participantID,
		"sdp_length", len(sdp),
	)

	if c.participantID == "" {
		c.sendError("Not joined to a room", nil)
		return
	}

	if c.webrtcManager == nil {
		c.sendError("WebRTC manager not configured", nil)
		return
	}

	if err := c.webrtcManager.HandleAnswer(c.participantID, sdp); err != nil {
		c.logger.Errorw("Failed to process WebRTC answer", "error", err, "participant_id", c.participantID)
		c.sendError("Failed to process WebRTC answer", err)
		return
	}

	response := protocol.WebSocketMessage{
		Type: "answer_processed",
		Payload: map[string]interface{}{
			"status":         "success",
			"participant_id": c.participantID,
		},
	}

	c.sendMessage(&response)
}

// handleICECandidate handles ICE candidate
func (c *Connection) handleICECandidate(payload map[string]interface{}) {
	c.logger.Debugw("Handling ICE candidate", "participant_id", c.participantID)

	if c.participantID == "" || c.webrtcManager == nil {
		return
	}

	// Extract candidate data
	candidateData, ok := payload["candidate"].(map[string]interface{})
	if !ok {
		c.logger.Warnw("Invalid ICE candidate format")
		return
	}

	// Parse candidate
	candidateStr, _ := candidateData["candidate"].(string)
	candidate := pionwebrtc.ICECandidateInit{
		Candidate: candidateStr,
	}

	if sdpMLineIndex, ok := candidateData["sdpMLineIndex"].(float64); ok {
		idx := uint16(sdpMLineIndex)
		candidate.SDPMLineIndex = &idx
	}

	if sdpMid, ok := candidateData["sdpMid"].(string); ok {
		candidate.SDPMid = &sdpMid
	}

	if err := c.webrtcManager.AddICECandidate(c.participantID, candidate); err != nil {
		c.logger.Warnw("Failed to add ICE candidate", "error", err, "participant_id", c.participantID)
		return
	}

	c.logger.Debugw("ICE candidate processed successfully",
		"participant_id", c.participantID,
	)
}

// sendMessage sends a message through the WebSocket connection
func (c *Connection) sendMessage(msg *protocol.WebSocketMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		c.logger.Errorw("Failed to marshal message", "error", err)
		return
	}

	select {
	case c.send <- data:
	default:
		c.logger.Warnw("Send channel full, dropping message")
	}
}

// sendError sends an error message through the WebSocket connection
func (c *Connection) sendError(message string, err error) {
	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}

	response := protocol.WebSocketMessage{
		Type: "error",
		Payload: map[string]interface{}{
			"error":   errorMsg,
			"message": message,
		},
	}

	c.sendMessage(&response)

	c.logger.Warnw("Sent error to client",
		"message", message,
		"error", err,
	)
}
