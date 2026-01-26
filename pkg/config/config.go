package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the complete server configuration
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	WebRTC    WebRTCConfig    `yaml:"webrtc"`
	Redis     RedisConfig     `yaml:"redis"`
	Limits    LimitsConfig    `yaml:"limits"`
	WebSocket WebSocketConfig `yaml:"websocket"`
	Zones     []ZoneConfig    `yaml:"zones"`
	Logging   LoggingConfig   `yaml:"logging"`
	Metrics   MetricsConfig   `yaml:"metrics"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Host string    `yaml:"host"`
	Port int       `yaml:"port"`
	Name string    `yaml:"name"`
	TLS  TLSConfig `yaml:"tls"`
}

// TLSConfig represents TLS/SSL configuration
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// WebRTCConfig represents WebRTC configuration
type WebRTCConfig struct {
	ICEServers []ICEServerConfig `yaml:"ice_servers"`
	PortRange  PortRangeConfig   `yaml:"port_range"`
}

// ICEServerConfig represents a single ICE server
type ICEServerConfig struct {
	URLs       []string `yaml:"urls"`
	Username   string   `yaml:"username,omitempty"`
	Credential string   `yaml:"credential,omitempty"`
}

// PortRangeConfig represents UDP port range for WebRTC
type PortRangeConfig struct {
	Min uint16 `yaml:"min"`
	Max uint16 `yaml:"max"`
}

// RedisConfig represents Redis connection configuration
type RedisConfig struct {
	Mode      string   `yaml:"mode"` // single, sentinel, cluster
	Addresses []string `yaml:"addresses"`
	Password  string   `yaml:"password"`
	DB        int      `yaml:"db"`
}

// LimitsConfig represents track and participant limits
type LimitsConfig struct {
	MaxCameraTracksPerUser int `yaml:"max_camera_tracks_per_user"`
	MaxScreenTracksPerUser int `yaml:"max_screen_tracks_per_user"`
	MaxScreenTracksPerRoom int `yaml:"max_screen_tracks_per_room"`
	MaxAudioTracksPerUser  int `yaml:"max_audio_tracks_per_user"`
	MaxConcurrentHolds     int `yaml:"max_concurrent_holds"`
	HoldTimeoutSeconds     int `yaml:"hold_timeout_seconds"`
}

// WebSocketConfig represents WebSocket security configuration
type WebSocketConfig struct {
	AllowedOrigins       []string `yaml:"allowed_origins"`
	ReconnectTimeout     int      `yaml:"reconnect_timeout"` // Seconds before auto-hold
	MaxReconnectAttempts int      `yaml:"max_reconnect_attempts"`
}

// ZoneConfig represents a stream zone configuration
type ZoneConfig struct {
	ID      string `yaml:"id"`
	Region  string `yaml:"region"`
	Primary bool   `yaml:"primary"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, console
}

// MetricsConfig represents metrics/monitoring configuration
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		// Set defaults for limits if not specified
		if c.Limits.MaxCameraTracksPerUser == 0 {
			c.Limits.MaxCameraTracksPerUser = 1
		}
		if c.Limits.MaxScreenTracksPerUser == 0 {
			c.Limits.MaxScreenTracksPerUser = 1
		}
		if c.Limits.MaxScreenTracksPerRoom == 0 {
			c.Limits.MaxScreenTracksPerRoom = 5
		}
		if c.Limits.MaxAudioTracksPerUser == 0 {
			c.Limits.MaxAudioTracksPerUser = 2
		}
		if c.Limits.MaxConcurrentHolds == 0 {
			c.Limits.MaxConcurrentHolds = 3
		}
		if c.Limits.HoldTimeoutSeconds == 0 {
			c.Limits.HoldTimeoutSeconds = 3600
		}

		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.WebRTC.PortRange.Min >= c.WebRTC.PortRange.Max {
		return fmt.Errorf("invalid WebRTC port range")
	}

	if len(c.Redis.Addresses) == 0 {
		return fmt.Errorf("at least one Redis address is required")
	}

	if len(c.Zones) == 0 {
		return fmt.Errorf("at least one zone is required")
	}

	// WebSocket defaults
	if c.WebSocket.ReconnectTimeout == 0 {
		c.WebSocket.ReconnectTimeout = 30 // 30 seconds
	}
	if c.WebSocket.MaxReconnectAttempts == 0 {
		c.WebSocket.MaxReconnectAttempts = 3
	}

	return nil
}
