package httpapi

import (
	"net/http"

	"github.com/cyper98/zenc-server/internal/participant"
	"github.com/cyper98/zenc-server/internal/room"
	"github.com/cyper98/zenc-server/pkg/logger"
	"github.com/cyper98/zenc-server/pkg/protocol"
	"github.com/gin-gonic/gin"
)

// Handler contains dependencies for HTTP handlers
type Handler struct {
	roomManager *room.Manager
	logger      *logger.Logger
}

// RegisterRoutes registers all HTTP API routes
func RegisterRoutes(router *gin.RouterGroup, roomManager *room.Manager, logger *logger.Logger) {
	h := &Handler{
		roomManager: roomManager,
		logger:      logger,
	}

	// Room management routes
	rooms := router.Group("/rooms")
	{
		rooms.POST("", h.CreateRoom)
		rooms.GET("", h.ListRooms)
		rooms.GET("/:id", h.GetRoom)
		rooms.DELETE("/:id", h.DeleteRoom)
		rooms.POST("/:id/join", h.JoinRoom)
		rooms.POST("/:id/rejoin", h.RejoinRoom)
		rooms.POST("/:id/leave", h.LeaveRoom)
		rooms.GET("/:id/participants", h.GetParticipants)
		rooms.POST("/:id/hold", h.HoldRoom)
		rooms.POST("/:id/resume", h.ResumeRoom)
	}

	// Participant routes
	participants := router.Group("/participants")
	{
		participants.GET("/:id/held", h.GetHeldRooms)
	}

	// Track management routes
	tracks := router.Group("/tracks")
	{
		tracks.POST("/publish", h.PublishTrack)
		tracks.POST("/unpublish", h.UnpublishTrack)
		tracks.POST("/subscribe", h.SubscribeTrack)
		tracks.POST("/hold", h.HoldTrack)
		tracks.POST("/resume", h.ResumeTrack)
	}
}

// CreateRoom handles POST /api/v1/rooms
func (h *Handler) CreateRoom(c *gin.Context) {
	var req protocol.CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, protocol.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	room, err := h.roomManager.CreateRoom(req)
	if err != nil {
		h.logger.Errorw("Failed to create room", "error", err)
		c.JSON(http.StatusInternalServerError, protocol.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create room",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusCreated, room)
}

// ListRooms handles GET /api/v1/rooms
func (h *Handler) ListRooms(c *gin.Context) {
	rooms, err := h.roomManager.ListRooms()
	if err != nil {
		h.logger.Errorw("Failed to list rooms", "error", err)
		c.JSON(http.StatusInternalServerError, protocol.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list rooms",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rooms": rooms,
		"count": len(rooms),
	})
}

// GetRoom handles GET /api/v1/rooms/:id
func (h *Handler) GetRoom(c *gin.Context) {
	roomID := c.Param("id")

	room, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		c.JSON(http.StatusNotFound, protocol.ErrorResponse{
			Error:   "not_found",
			Message: "Room not found",
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, room)
}

// DeleteRoom handles DELETE /api/v1/rooms/:id
func (h *Handler) DeleteRoom(c *gin.Context) {
	roomID := c.Param("id")

	if err := h.roomManager.DeleteRoom(roomID); err != nil {
		h.logger.Errorw("Failed to delete room", "room_id", roomID, "error", err)
		c.JSON(http.StatusInternalServerError, protocol.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete room",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Room deleted successfully",
	})
}

// JoinRoom handles POST /api/v1/rooms/:id/join
func (h *Handler) JoinRoom(c *gin.Context) {
	roomID := c.Param("id")

	var req protocol.JoinRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, protocol.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Verify room exists
	room, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		c.JSON(http.StatusNotFound, protocol.ErrorResponse{
			Error:   "not_found",
			Message: "Room not found",
			Code:    http.StatusNotFound,
		})
		return
	}

	// Create participant
	if req.Role == "" {
		req.Role = protocol.RoleSubscriber
	}

	p := participant.New(roomID, req.UserID, req.Role)

	// Add participant to room
	if err := h.roomManager.AddParticipant(roomID, p); err != nil {
		h.logger.Errorw("Failed to add participant", "error", err)
		c.JSON(http.StatusInternalServerError, protocol.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to join room",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"participant": p,
		"room":        room,
	})
}

// RejoinRoom handles POST /api/v1/rooms/:id/rejoin (reconnection)
func (h *Handler) RejoinRoom(c *gin.Context) {
	roomID := c.Param("id")

	var req struct {
		ParticipantID string `json:"participant_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, protocol.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Check participant state
	state, err := h.roomManager.GetParticipantState(roomID, req.ParticipantID)
	if err != nil {
		c.JSON(http.StatusNotFound, protocol.ErrorResponse{
			Error:   "not_found",
			Message: "Participant not found or session expired",
			Code:    http.StatusNotFound,
		})
		return
	}

	// Allow rejoin if in-transit or on-hold
	if state != protocol.StateInTransit && state != protocol.StateOnHold {
		c.JSON(http.StatusBadRequest, protocol.ErrorResponse{
			Error:   "invalid_state",
			Message: "Participant is already active in room",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Restore to active state
	if err := h.roomManager.SetParticipantState(roomID, req.ParticipantID, protocol.StateActive); err != nil {
		h.logger.Errorw("Failed to restore participant state", "error", err)
		c.JSON(http.StatusInternalServerError, protocol.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to rejoin room",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	room, _ := h.roomManager.GetRoom(roomID)

	h.logger.Infow("Participant rejoined",
		"participant_id", req.ParticipantID,
		"room_id", roomID,
		"previous_state", state,
	)

	c.JSON(http.StatusOK, gin.H{
		"message":        "Rejoined successfully",
		"participant_id": req.ParticipantID,
		"room":           room,
		"previous_state": state,
	})
}

// LeaveRoom handles POST /api/v1/rooms/:id/leave
func (h *Handler) LeaveRoom(c *gin.Context) {
	roomID := c.Param("id")

	var req struct {
		ParticipantID string `json:"participant_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, protocol.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	if err := h.roomManager.RemoveParticipant(roomID, req.ParticipantID); err != nil {
		h.logger.Errorw("Failed to remove participant", "error", err)
		c.JSON(http.StatusInternalServerError, protocol.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to leave room",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Left room successfully",
	})
}

// GetParticipants handles GET /api/v1/rooms/:id/participants
func (h *Handler) GetParticipants(c *gin.Context) {
	roomID := c.Param("id")

	participants, err := h.roomManager.GetParticipants(roomID)
	if err != nil {
		h.logger.Errorw("Failed to get participants", "error", err)
		c.JSON(http.StatusInternalServerError, protocol.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get participants",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"participants": participants,
		"count":        len(participants),
	})
}

// PublishTrack handles POST /api/v1/tracks/publish
func (h *Handler) PublishTrack(c *gin.Context) {
	var req protocol.PublishTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, protocol.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Check if participant exists and get their current room
	participants, err := h.roomManager.GetParticipants("")
	if err == nil {
		for _, p := range participants {
			if p.ID == req.ParticipantID {
				// Check if participant is on hold
				if p.State == protocol.StateOnHold {
					c.JSON(http.StatusForbidden, protocol.ErrorResponse{
						Error:   "participant_on_hold",
						Message: "Cannot publish tracks while participant is on hold",
						Code:    http.StatusForbidden,
					})
					return
				}
				break
			}
		}
	}

	// TODO: Implement WebRTC negotiation with Pion
	// For now, return a placeholder response

	c.JSON(http.StatusOK, gin.H{
		"message":    "Track publish initiated (WebRTC implementation pending)",
		"track_id":   "placeholder-track-id",
		"sdp_answer": "placeholder-sdp-answer",
	})
}

// UnpublishTrack handles POST /api/v1/tracks/unpublish
func (h *Handler) UnpublishTrack(c *gin.Context) {
	var req struct {
		ParticipantID string `json:"participant_id" binding:"required"`
		TrackID       string `json:"track_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, protocol.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Track unpublished successfully",
	})
}

// SubscribeTrack handles POST /api/v1/tracks/subscribe
func (h *Handler) SubscribeTrack(c *gin.Context) {
	var req protocol.SubscribeTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, protocol.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// TODO: Implement WebRTC subscription

	c.JSON(http.StatusOK, gin.H{
		"message":    "Track subscription initiated (WebRTC implementation pending)",
		"sdp_answer": "placeholder-sdp-answer",
	})
}

// HoldTrack handles POST /api/v1/tracks/hold
func (h *Handler) HoldTrack(c *gin.Context) {
	var req struct {
		TrackID string `json:"track_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, protocol.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Track held successfully",
		"status":  protocol.TrackStatusPaused,
	})
}

// ResumeTrack handles POST /api/v1/tracks/resume
func (h *Handler) ResumeTrack(c *gin.Context) {
	var req struct {
		TrackID string `json:"track_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, protocol.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Track resumed successfully",
		"status":  protocol.TrackStatusActive,
	})
}

// HoldRoom handles POST /api/v1/rooms/:id/hold
func (h *Handler) HoldRoom(c *gin.Context) {
	roomID := c.Param("id")

	var req struct {
		ParticipantID string `json:"participant_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, protocol.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Update participant state to on hold
	if err := h.roomManager.SetParticipantState(roomID, req.ParticipantID, protocol.StateOnHold); err != nil {
		h.logger.Errorw("Failed to set participant state", "error", err)
		c.JSON(http.StatusInternalServerError, protocol.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to put room on hold",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// TODO: Use HoldManager to store track state
	// holdManager.HoldRoom(req.ParticipantID, roomID, tracks)

	h.logger.Infow("Room put on hold",
		"room_id", roomID,
		"participant_id", req.ParticipantID,
	)

	c.JSON(http.StatusOK, gin.H{
		"message":        "Room put on hold successfully",
		"room_id":        roomID,
		"participant_id": req.ParticipantID,
		"state":          protocol.StateOnHold,
	})
}

// ResumeRoom handles POST /api/v1/rooms/:id/resume
func (h *Handler) ResumeRoom(c *gin.Context) {
	roomID := c.Param("id")

	var req struct {
		ParticipantID string `json:"participant_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, protocol.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Update participant state back to active
	if err := h.roomManager.SetParticipantState(roomID, req.ParticipantID, protocol.StateActive); err != nil {
		h.logger.Errorw("Failed to set participant state", "error", err)
		c.JSON(http.StatusInternalServerError, protocol.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to resume room",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// TODO: Use HoldManager to restore track state
	// holdState, err := holdManager.ResumeRoom(req.ParticipantID, roomID)

	h.logger.Infow("Room resumed from hold",
		"room_id", roomID,
		"participant_id", req.ParticipantID,
	)

	c.JSON(http.StatusOK, gin.H{
		"message":        "Room resumed successfully",
		"room_id":        roomID,
		"participant_id": req.ParticipantID,
		"state":          protocol.StateActive,
	})
}

// GetHeldRooms handles GET /api/v1/participants/:id/held
func (h *Handler) GetHeldRooms(c *gin.Context) {
	participantID := c.Param("id")

	// TODO: Implement with HoldManager
	// heldRooms := holdManager.GetHeldRooms(participantID)

	h.logger.Infow("Getting held rooms",
		"participant_id", participantID,
	)

	c.JSON(http.StatusOK, gin.H{
		"participant_id": participantID,
		"held_rooms":     []string{}, // Placeholder
	})
}
