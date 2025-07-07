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
        if !filepath.IsAbs(configPath) || strings.Contains(configPath, "..") {
                return nil, fmt.Errorf("invalid config path: %s", configPath)
        }
        
        // Read config file
        data, err := os.ReadFile(configPath)
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
		config.Server.Host = "localhost"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
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

	return nil
} 