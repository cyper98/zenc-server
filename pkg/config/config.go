package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the complete server configuration
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	WebRTC  WebRTCConfig  `yaml:"webrtc"`
	Redis   RedisConfig   `yaml:"redis"`
	Zones   []ZoneConfig  `yaml:"zones"`
	Logging LoggingConfig `yaml:"logging"`
	Metrics MetricsConfig `yaml:"metrics"`
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

	return nil
}
