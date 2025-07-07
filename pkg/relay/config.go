package relay

import (
	"fmt"
)

// Config represents the client configuration
type Config struct {
	UseTLS          bool
	TLSCertFile     string
	TLSKeyFile      string
	TLSCAFile       string
	ServerHost      string
	ServerPort      int
	JWTToken        string
	LocalPort       int
	ReconnectDelay  int
	MaxRetries      int
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ServerHost == "" {
		return fmt.Errorf("server host is required")
	}

	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid server port")
	}

	if c.LocalPort <= 0 || c.LocalPort > 65535 {
		return fmt.Errorf("invalid local port")
	}

	if c.UseTLS {
		if c.TLSCertFile != "" && c.TLSKeyFile == "" {
			return fmt.Errorf("TLS key file is required when certificate file is provided")
		}
		if c.TLSKeyFile != "" && c.TLSCertFile == "" {
			return fmt.Errorf("TLS certificate file is required when key file is provided")
		}
	}

	return nil
} 