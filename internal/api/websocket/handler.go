package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/aminofox/zenc-server/internal/room"
	"github.com/aminofox/zenc-server/pkg/logger"
	"github.com/aminofox/zenc-server/pkg/protocol"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking
		return true
	},
}

// Connection represents a WebSocket connection
type Connection struct {
	conn        *websocket.Conn
	send        chan []byte
	roomManager *room.Manager
	logger      *logger.Logger
	mu          sync.Mutex
}

// HandleConnection handles WebSocket connections
func HandleConnection(w http.ResponseWriter, r *http.Request, roomManager *room.Manager, logger *logger.Logger) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Errorw("Failed to upgrade WebSocket connection", "error", err)
		return
	}

	c := &Connection{
		conn:        conn,
		send:        make(chan []byte, 256),
		roomManager: roomManager,
		logger:      logger,
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

		// Handle message
		c.handleMessage(&msg)
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

	c.logger.Infow("Handling join request",
		"room_id", roomID,
		"user_id", userID,
	)

	// TODO: Implement join logic
	response := protocol.WebSocketMessage{
		Type: "joined",
		Payload: map[string]interface{}{
			"room_id": roomID,
			"status":  "success",
		},
	}

	c.sendMessage(&response)
}

// handleLeave handles room leave requests
func (c *Connection) handleLeave(payload map[string]interface{}) {
	c.logger.Infow("Handling leave request")
	// TODO: Implement leave logic
}

// handleOffer handles WebRTC offer
func (c *Connection) handleOffer(payload map[string]interface{}) {
	c.logger.Infow("Handling WebRTC offer")
	// TODO: Implement WebRTC offer handling with Pion
}

// handleAnswer handles WebRTC answer
func (c *Connection) handleAnswer(payload map[string]interface{}) {
	c.logger.Infow("Handling WebRTC answer")
	// TODO: Implement WebRTC answer handling with Pion
}

// handleICECandidate handles ICE candidate
func (c *Connection) handleICECandidate(payload map[string]interface{}) {
	c.logger.Debugw("Handling ICE candidate")
	// TODO: Implement ICE candidate handling with Pion
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
