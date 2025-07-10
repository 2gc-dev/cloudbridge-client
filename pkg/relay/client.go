package relay

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/2gc-dev/cloudbridge-client/pkg/config"
	"github.com/2gc-dev/cloudbridge-client/pkg/protocol"
)

// Message types
const (
	MessageTypeHello         = "hello"
	MessageTypeAuth          = "auth"
	MessageTypeAuthResponse  = "auth_response"
	MessageTypeTunnelInfo    = "tunnel_info"
	MessageTypeTunnelResponse = "tunnel_response"
	MessageTypeHeartbeat     = "heartbeat"
	MessageTypeHeartbeatResponse = "heartbeat_response"
	MessageTypeError         = "error"

	MaxMessageSize      = 1024 * 1024 // 1MB
	ConnectTimeout      = 10 * time.Second
	ReadWriteTimeout    = 30 * time.Second
	HeartbeatInterval   = 30 * time.Second
	HeartbeatTimeout    = 5 * time.Second
	MaxMissedHeartbeats = 3
)

// Client represents a CloudBridge Relay client
type Client struct {
	conn    net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
	useTLS  bool
	config  *tls.Config
	cfg     *config.Config

	missedHeartbeats int32
	stopHeartbeat    chan struct{}
	tunnels          map[string]*Tunnel
	tunnelMutex      sync.RWMutex
	
	// New fields for v2.0
	protocolEngine *protocol.ProtocolEngine
	tenantID       string
	version        string
	features       []string
}

// Tunnel represents a managed tunnel connection
type Tunnel struct {
	ID         string
	LocalPort  int
	RemoteHost string
	RemotePort int
	Protocol   string
	Options    map[string]interface{}
	stopChan   chan struct{}
	proxyCmd   *exec.Cmd
}

// NewClient creates a new CloudBridge Relay client
func NewClient(useTLS bool, tlsConfig *tls.Config) *Client {
	return &Client{
		useTLS:        useTLS,
		config:        tlsConfig,
		stopHeartbeat: make(chan struct{}),
		tunnels:       make(map[string]*Tunnel),
		protocolEngine: protocol.NewProtocolEngine(),
		version:       protocol.ProtocolVersionV2,
		features:      []string{
			protocol.FeatureTLS, protocol.FeatureHeartbeat, protocol.FeatureTunnelInfo,
			protocol.FeatureMultiTenant, protocol.FeatureProxy, protocol.FeatureQUIC, protocol.FeatureMetrics,
		},
	}
}

// NewClientV1 creates a new CloudBridge Relay client for v1.0.0 (backward compatibility)
func NewClientV1(useTLS bool, tlsConfig *tls.Config) *Client {
	return &Client{
		useTLS:        useTLS,
		config:        tlsConfig,
		stopHeartbeat: make(chan struct{}),
		tunnels:       make(map[string]*Tunnel),
		protocolEngine: protocol.NewProtocolEngineV1(),
		version:       protocol.ProtocolVersionV1,
		features:      []string{
			protocol.FeatureTLS, protocol.FeatureJWT, protocol.FeatureTunneling, protocol.FeatureQUIC, protocol.FeatureHTTP2,
		},
	}
}

// NewClientFromConfig creates a new client from config
func NewClientFromConfig(cfg *config.Config) (*Client, error) {
	var tlsConfig *tls.Config
	var err error

	if cfg.TLS.Enabled {
		tlsConfig, err = NewTLSConfig(cfg.TLS.CertFile, cfg.TLS.KeyFile, cfg.TLS.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
	}

	// Determine version from config or default to v2.0
	version := protocol.ProtocolVersionV2
	if cfg.Protocol.Version != "" {
		version = cfg.Protocol.Version
	}

	var protocolEngine *protocol.ProtocolEngine
	if version == protocol.ProtocolVersionV1 {
		protocolEngine = protocol.NewProtocolEngineV1()
	} else {
		protocolEngine = protocol.NewProtocolEngine()
	}

	client := &Client{
		useTLS:        cfg.TLS.Enabled,
		config:        tlsConfig,
		cfg:           cfg,
		stopHeartbeat: make(chan struct{}),
		tunnels:       make(map[string]*Tunnel),
		protocolEngine: protocolEngine,
		version:       version,
		tenantID:      cfg.Tenant.ID,
		features:      protocolEngine.GetFeatures(),
	}

	return client, nil
}

// SetTenantID sets the tenant ID for multi-tenancy support
func (c *Client) SetTenantID(tenantID string) {
	c.tenantID = tenantID
}

// GetTenantID returns the current tenant ID
func (c *Client) GetTenantID() string {
	return c.tenantID
}

// GetVersion returns the protocol version
func (c *Client) GetVersion() string {
	return c.version
}

// GetFeatures returns the supported features
func (c *Client) GetFeatures() []string {
	return c.features
}

// Connect establishes a connection to the relay server
func (c *Client) Connect(host string, port int) error {
	var err error
	var conn net.Conn
	dialer := &net.Dialer{Timeout: ConnectTimeout}
	address := fmt.Sprintf("%s:%d", host, port)

	if c.useTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", address, c.config)
	} else {
		conn, err = dialer.Dial("tcp", address)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}

	c.conn = conn
	c.reader = bufio.NewReaderSize(conn, MaxMessageSize)
	c.writer = bufio.NewWriter(conn)
	return nil
}

// Close closes the connection to the relay server
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendMessage отправляет JSON-сообщение с \n
func (c *Client) SendMessage(msg interface{}) error {
	if c.conn == nil {
		return fmt.Errorf("not connected to server")
	}
	
	if err := c.conn.SetWriteDeadline(time.Now().Add(ReadWriteTimeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if len(data) > MaxMessageSize {
		return fmt.Errorf("message too large")
	}
	if _, err := c.writer.Write(append(data, '\n')); err != nil {
		return err
	}
	return c.writer.Flush()
}

// ReadMessage читает строку, парсит JSON, ограничивает размер
func (c *Client) ReadMessage() (map[string]interface{}, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(ReadWriteTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if len(line) > MaxMessageSize {
		return nil, fmt.Errorf("message too large")
	}
	line = strings.TrimSpace(line)
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// Handshake: ждет hello, отправляет auth, ждет auth_response
func (c *Client) Handshake(token string) error {
	// 1. Ждем hello
	hello, err := c.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read hello: %w", err)
	}

	if hello["type"] != MessageTypeHello {
		return fmt.Errorf("expected hello message, got: %s", hello["type"])
	}

	// 2. Отправляем auth based on version
	var authMsg interface{}
	if c.version == protocol.ProtocolVersionV2 {
		authMsg = protocol.NewAuthMessage(token, c.tenantID)
	} else {
		// v1.0.0 backward compatibility
		clientInfo := map[string]interface{}{
			"os":   runtime.GOOS,
			"arch": runtime.GOARCH,
		}
		authMsg = protocol.NewAuthMessageV1(token, clientInfo)
	}

	if err := c.SendMessage(authMsg); err != nil {
		return fmt.Errorf("failed to send auth: %w", err)
	}

	// 3. Ждем auth_response
	authResp, err := c.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	if authResp["type"] != MessageTypeAuthResponse {
		return fmt.Errorf("expected auth_response message, got: %s", authResp["type"])
	}

	if status, ok := authResp["status"].(string); !ok || status != "success" {
		errorMsg := "authentication failed"
		if msg, ok := authResp["message"].(string); ok {
			errorMsg = msg
		}
		return fmt.Errorf("authentication failed: %s", errorMsg)
	}

	return nil
}

// CreateTunnel creates a new tunnel
func (c *Client) CreateTunnel(localPort int, remoteHost string, remotePort int) (string, error) {
	tunnelID := fmt.Sprintf("tunnel_%d_%s_%d", localPort, remoteHost, remotePort)
	
	tunnel := &Tunnel{
		ID:         tunnelID,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		Protocol:   "tcp",
		Options:    make(map[string]interface{}),
		stopChan:   make(chan struct{}),
	}

	c.tunnelMutex.Lock()
	c.tunnels[tunnelID] = tunnel
	c.tunnelMutex.Unlock()

	return tunnelID, nil
}

// NewTLSConfig creates a new TLS configuration
func NewTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	// Implementation for TLS config creation
	return &tls.Config{
		InsecureSkipVerify: true, // For development, should be false in production
	}, nil
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	return c.conn != nil
} 