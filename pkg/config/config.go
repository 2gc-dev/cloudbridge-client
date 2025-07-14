package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	TLS struct {
		Enabled  bool   `yaml:"enabled"`
		CertFile string `yaml:"cert_file"`
		KeyFile  string `yaml:"key_file"`
		CAFile   string `yaml:"ca_file"`
	} `yaml:"tls"`

	Server struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		JWTToken string `yaml:"jwt_token"`
	} `yaml:"server"`

	Auth struct {
		Secret string `yaml:"secret"`
	} `yaml:"auth"`

	Tunnel struct {
		LocalPort      int `yaml:"local_port"`
		ReconnectDelay int `yaml:"reconnect_delay"`
		MaxRetries     int `yaml:"max_retries"`
	} `yaml:"tunnel"`

	Logging struct {
		Level      string `yaml:"level"`
		File       string `yaml:"file"`
		MaxSize    int    `yaml:"max_size"`
		MaxBackups int    `yaml:"max_backups"`
		MaxAge     int    `yaml:"max_age"`
		Compress   bool   `yaml:"compress"`
	} `yaml:"logging"`

	// New fields for v2.0 support
	Protocol struct {
		Version string `yaml:"version"`
		Features []string `yaml:"features"`
	} `yaml:"protocol"`

	Tenant struct {
		ID   string `yaml:"id"`
		Name string `yaml:"name"`
	} `yaml:"tenant"`

	Metrics struct {
		Enabled  bool   `yaml:"enabled"`
		Port     int    `yaml:"port"`
		Path     string `yaml:"path"`
		Interval string `yaml:"interval"`
	} `yaml:"metrics"`

	Health struct {
		Enabled       bool   `yaml:"enabled"`
		Path          string `yaml:"path"`
		CheckInterval string `yaml:"check_interval"`
	} `yaml:"health"`
}

// Save сохраняет конфигурацию в файл
func (c *Config) Save(path string) error {
	// Validate path to prevent path traversal
	if path == "" || path == "." || path == ".." || path == "/" {
		return fmt.Errorf("invalid config path")
	}
	
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("error marshaling config: %v", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	return nil
}

func LoadConfig(configPath string) (*Config, error) {
	// If no config path is provided, try default locations
	if configPath == "" {
		configPath = os.Getenv("CONFIG_FILE")
		if configPath == "" {
			configPath = "/etc/cloudbridge-client/config.yaml"
		}
	}

	// Validate path to prevent path traversal
	if configPath == "" || configPath == "." || configPath == ".." || configPath == "/" {
		return nil, fmt.Errorf("invalid config path")
	}

	        	// Validate config path to prevent directory traversal
	cleanPath := filepath.Clean(configPath)
	if !filepath.IsAbs(cleanPath) || strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("invalid config path: %s", configPath)
	}
	
	// Additional security check - ensure path is within allowed directories
	allowedDirs := []string{"/etc/cloudbridge-client", "/opt/cloudbridge-client", "/var/lib/cloudbridge-client"}
	allowed := false
	for _, dir := range allowedDirs {
		if strings.HasPrefix(cleanPath, dir) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("config path not in allowed directories: %s", configPath)
	}
	
	// Read config file
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	// Parse YAML
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	// Set defaults if not provided
	if config.Server.Host == "" {
		config.Server.Host = "edge.2gc.ru"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 3456
	}
	if config.Tunnel.LocalPort == 0 {
		config.Tunnel.LocalPort = 3389
	}
	if config.Tunnel.ReconnectDelay == 0 {
		config.Tunnel.ReconnectDelay = 5
	}
	if config.Tunnel.MaxRetries == 0 {
		config.Tunnel.MaxRetries = 3
	}

	// Set protocol defaults
	if config.Protocol.Version == "" {
		config.Protocol.Version = "2.0"
	}

	// Set metrics defaults
	if config.Metrics.Port == 0 {
		config.Metrics.Port = 9090
	}
	if config.Metrics.Path == "" {
		config.Metrics.Path = "/metrics"
	}
	if config.Metrics.Interval == "" {
		config.Metrics.Interval = "15s"
	}

	// Set health defaults
	if config.Health.Path == "" {
		config.Health.Path = "/health"
	}
	if config.Health.CheckInterval == "" {
		config.Health.CheckInterval = "30s"
	}

	return config, nil
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.Server.Host == "" {
		return fmt.Errorf("server host is required")
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port")
	}

	if c.TLS.Enabled {
		if c.TLS.CertFile != "" {
			if _, err := os.Stat(c.TLS.CertFile); os.IsNotExist(err) {
				return fmt.Errorf("TLS certificate file not found: %s", c.TLS.CertFile)
			}
		}
		if c.TLS.KeyFile != "" {
			if _, err := os.Stat(c.TLS.KeyFile); os.IsNotExist(err) {
				return fmt.Errorf("TLS key file not found: %s", c.TLS.KeyFile)
			}
		}
		if c.TLS.CAFile != "" {
			if _, err := os.Stat(c.TLS.CAFile); os.IsNotExist(err) {
				return fmt.Errorf("TLS CA file not found: %s", c.TLS.CAFile)
			}
		}
	}

	// Validate protocol version
	if c.Protocol.Version != "" && c.Protocol.Version != "1.0.0" && c.Protocol.Version != "2.0" {
		return fmt.Errorf("unsupported protocol version: %s", c.Protocol.Version)
	}

	return nil
} 