package webrtc

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aminofox/zenc-server/pkg/logger"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
)

// Config contains WebRTC configuration
type Config struct {
	ICEServers []webrtc.ICEServer
	PortMin    uint16
	PortMax    uint16
}

// Manager handles WebRTC connections and media tracks
type Manager struct {
	config          Config
	logger          *logger.Logger
	peerConnections map[string]*webrtc.PeerConnection
	tracks          map[string]*webrtc.TrackLocalStaticRTP
	api             *webrtc.API
	mu              sync.RWMutex
}

// NewManager creates a new WebRTC manager
func NewManager(config Config, logger *logger.Logger) (*Manager, error) {
	// Create a MediaEngine
	mediaEngine := &webrtc.MediaEngine{}

	// Register default codecs
	if err := mediaEngine.RegisterDefaultCodecs(); err != nil {
		return nil, fmt.Errorf("failed to register codecs: %w", err)
	}

	// Create an Interceptor Registry
	interceptorRegistry := &interceptor.Registry{}

	// Register default interceptors (RTCP, etc.)
	if err := webrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry); err != nil {
		return nil, fmt.Errorf("failed to register interceptors: %w", err)
	}

	// Create a SettingEngine to configure port range
	settingEngine := webrtc.SettingEngine{}
	if config.PortMin > 0 && config.PortMax > 0 {
		settingEngine.SetEphemeralUDPPortRange(config.PortMin, config.PortMax)
	}

	// Create WebRTC API with MediaEngine and SettingEngine
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithInterceptorRegistry(interceptorRegistry),
		webrtc.WithSettingEngine(settingEngine),
	)

	return &Manager{
		config:          config,
		logger:          logger,
		peerConnections: make(map[string]*webrtc.PeerConnection),
		tracks:          make(map[string]*webrtc.TrackLocalStaticRTP),
		api:             api,
	}, nil
}

// CreatePeerConnection creates a new peer connection for a participant
func (m *Manager) CreatePeerConnection(participantID string) (*webrtc.PeerConnection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if connection already exists
	if pc, exists := m.peerConnections[participantID]; exists {
		return pc, nil
	}

	// Create PeerConnection configuration
	peerConnectionConfig := webrtc.Configuration{
		ICEServers: m.config.ICEServers,
	}

	// Create new PeerConnection
	peerConnection, err := m.api.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	// Handle ICE connection state changes
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		m.logger.Infow("ICE connection state changed",
			"participant_id", participantID,
			"state", state.String(),
		)

		// Clean up on disconnect
		if state == webrtc.ICEConnectionStateDisconnected ||
			state == webrtc.ICEConnectionStateFailed ||
			state == webrtc.ICEConnectionStateClosed {
			m.RemovePeerConnection(participantID)
		}
	})

	// Handle incoming tracks
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		m.logger.Infow("Received remote track",
			"participant_id", participantID,
			"kind", track.Kind().String(),
			"id", track.ID(),
		)

		// Create local track to relay to other participants
		codec := track.Codec()
		localTrack, err := m.CreateLocalTrack(codec.RTPCodecCapability, participantID, track.Kind())
		if err != nil {
			m.logger.Errorw("Failed to create local track", "error", err)
			return
		}

		// Read RTP packets from remote track and write to local track
		go func() {
			rtpBuf := make([]byte, 1500)
			for {
				i, _, readErr := track.Read(rtpBuf)
				if readErr != nil {
					return
				}

				if _, writeErr := localTrack.Write(rtpBuf[:i]); writeErr != nil {
					return
				}
			}
		}()
	})

	m.peerConnections[participantID] = peerConnection

	m.logger.Infow("Created peer connection", "participant_id", participantID)

	return peerConnection, nil
}

// HandleOffer processes a WebRTC offer and creates an answer
func (m *Manager) HandleOffer(participantID, sdp string) (string, error) {
	pc, err := m.CreatePeerConnection(participantID)
	if err != nil {
		return "", err
	}

	// Set remote description (offer)
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}

	if err := pc.SetRemoteDescription(offer); err != nil {
		return "", fmt.Errorf("failed to set remote description: %w", err)
	}

	// Create answer
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create answer: %w", err)
	}

	// Set local description (answer)
	if err := pc.SetLocalDescription(answer); err != nil {
		return "", fmt.Errorf("failed to set local description: %w", err)
	}

	m.logger.Infow("Created answer for offer", "participant_id", participantID)

	return answer.SDP, nil
}

// HandleAnswer processes a WebRTC answer
func (m *Manager) HandleAnswer(participantID, sdp string) error {
	m.mu.RLock()
	pc, exists := m.peerConnections[participantID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("peer connection not found for participant: %s", participantID)
	}

	// Set remote description (answer)
	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}

	if err := pc.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("failed to set remote description: %w", err)
	}

	m.logger.Infow("Processed answer", "participant_id", participantID)

	return nil
}

// AddICECandidate adds an ICE candidate to a peer connection
func (m *Manager) AddICECandidate(participantID string, candidateInit webrtc.ICECandidateInit) error {
	m.mu.RLock()
	pc, exists := m.peerConnections[participantID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("peer connection not found for participant: %s", participantID)
	}

	if err := pc.AddICECandidate(candidateInit); err != nil {
		return fmt.Errorf("failed to add ICE candidate: %w", err)
	}

	m.logger.Debugw("Added ICE candidate", "participant_id", participantID)

	return nil
}

// CreateLocalTrack creates a local track for relaying media
func (m *Manager) CreateLocalTrack(codec webrtc.RTPCodecCapability, trackID string, kind webrtc.RTPCodecType) (*webrtc.TrackLocalStaticRTP, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	track, err := webrtc.NewTrackLocalStaticRTP(codec, trackID, trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to create local track: %w", err)
	}

	m.tracks[trackID] = track

	m.logger.Infow("Created local track",
		"track_id", trackID,
		"kind", kind.String(),
	)

	return track, nil
}

// AddTrackToPeer adds a local track to a participant's peer connection
func (m *Manager) AddTrackToPeer(participantID, trackID string) error {
	m.mu.RLock()
	pc, pcExists := m.peerConnections[participantID]
	track, trackExists := m.tracks[trackID]
	m.mu.RUnlock()

	if !pcExists {
		return fmt.Errorf("peer connection not found for participant: %s", participantID)
	}

	if !trackExists {
		return fmt.Errorf("track not found: %s", trackID)
	}

	if _, err := pc.AddTrack(track); err != nil {
		return fmt.Errorf("failed to add track to peer connection: %w", err)
	}

	m.logger.Infow("Added track to peer connection",
		"participant_id", participantID,
		"track_id", trackID,
	)

	return nil
}

// RemovePeerConnection closes and removes a peer connection
func (m *Manager) RemovePeerConnection(participantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pc, exists := m.peerConnections[participantID]
	if !exists {
		return nil
	}

	if err := pc.Close(); err != nil {
		m.logger.Warnw("Error closing peer connection", "error", err)
	}

	delete(m.peerConnections, participantID)

	m.logger.Infow("Removed peer connection", "participant_id", participantID)

	return nil
}

// GetPeerConnection retrieves a peer connection by participant ID
func (m *Manager) GetPeerConnection(participantID string) (*webrtc.PeerConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pc, exists := m.peerConnections[participantID]
	return pc, exists
}

// Close closes all peer connections and cleans up
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for participantID, pc := range m.peerConnections {
		if err := pc.Close(); err != nil {
			m.logger.Warnw("Error closing peer connection",
				"participant_id", participantID,
				"error", err,
			)
		}
	}

	m.peerConnections = make(map[string]*webrtc.PeerConnection)
	m.tracks = make(map[string]*webrtc.TrackLocalStaticRTP)

	m.logger.Info("WebRTC manager closed")

	return nil
}

// GetStats returns statistics about active connections
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"peer_connections": len(m.peerConnections),
		"tracks":           len(m.tracks),
		"timestamp":        time.Now().Unix(),
	}
}

// MarshalICECandidate converts ICE candidate init to JSON
func MarshalICECandidate(candidate webrtc.ICECandidateInit) (string, error) {
	data, err := json.Marshal(candidate)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalICECandidate converts JSON to ICE candidate init
func UnmarshalICECandidate(data string) (*webrtc.ICECandidateInit, error) {
	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal([]byte(data), &candidate); err != nil {
		return nil, err
	}
	return &candidate, nil
}
