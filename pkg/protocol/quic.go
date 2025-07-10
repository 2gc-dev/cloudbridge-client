package protocol

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

// QUICClient represents a QUIC connection client
type QUICClient struct {
	conn     quic.Connection
	stream   quic.Stream
	config   *QUICConfig
	address  string
}

// QUICConfig holds QUIC-specific configuration
type QUICConfig struct {
	TLSConfig        *tls.Config
	KeepAlive        bool
	KeepAlivePeriod  time.Duration
	IdleTimeout      time.Duration
	HandshakeTimeout time.Duration
	MaxStreams       int
}

// DefaultQUICConfig returns default QUIC configuration
func DefaultQUICConfig() *QUICConfig {
	return &QUICConfig{
		KeepAlive:        true,
		KeepAlivePeriod:  30 * time.Second,
		IdleTimeout:      60 * time.Second,
		HandshakeTimeout: 10 * time.Second,
		MaxStreams:       100,
	}
}

// NewQUICClient creates a new QUIC client
func NewQUICClient(config *QUICConfig) *QUICClient {
	if config == nil {
		config = DefaultQUICConfig()
	}
	
	return &QUICClient{
		config: config,
	}
}

// Connect establishes a QUIC connection
func (qc *QUICClient) Connect(ctx context.Context, address string) error {
	qc.address = address
	
	// Create UDP connection
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}
	
	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return fmt.Errorf("failed to create UDP connection: %w", err)
	}
	
	// Create QUIC config
	quicConfig := &quic.Config{
		MaxIdleTimeout:  qc.config.IdleTimeout,
		MaxIncomingStreams: int64(qc.config.MaxStreams),
	}
	
	// Establish QUIC connection
	conn, err := quic.Dial(ctx, udpConn, udpAddr, qc.config.TLSConfig, quicConfig)
	if err != nil {
		return fmt.Errorf("failed to establish QUIC connection: %w", err)
	}
	
	qc.conn = conn
	
	// Open a stream for data transfer
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return fmt.Errorf("failed to open QUIC stream: %w", err)
	}
	
	qc.stream = stream
	
	return nil
}

// Send sends data over QUIC stream
func (qc *QUICClient) Send(data []byte) error {
	if qc.stream == nil {
		return fmt.Errorf("QUIC stream not established")
	}
	
	_, err := qc.stream.Write(data)
	return err
}

// Receive receives data from QUIC stream
func (qc *QUICClient) Receive(buffer []byte) (int, error) {
	if qc.stream == nil {
		return 0, fmt.Errorf("QUIC stream not established")
	}
	
	return qc.stream.Read(buffer)
}

// Close closes the QUIC connection
func (qc *QUICClient) Close() error {
	var errs []error
	
	if qc.stream != nil {
		if err := qc.stream.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close stream: %w", err))
		}
	}
	
	if qc.conn != nil {
		if err := qc.conn.CloseWithError(0, "client closing"); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection: %w", err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("errors closing QUIC client: %v", errs)
	}
	
	return nil
}

// IsConnected returns true if the client is connected
func (qc *QUICClient) IsConnected() bool {
	return qc.conn != nil && qc.stream != nil
}

// GetConnectionState returns the connection state
func (qc *QUICClient) GetConnectionState() tls.ConnectionState {
	if qc.conn != nil {
		// QUIC connection state is different from TLS connection state
		// Return empty TLS state for compatibility
		return tls.ConnectionState{}
	}
	return tls.ConnectionState{}
}

// GetStats returns QUIC connection statistics
func (qc *QUICClient) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	if qc.conn != nil {
		stats["connected"] = true
		stats["address"] = qc.address
		stats["connection_id"] = qc.conn.RemoteAddr().String()
		
		// QUIC connection doesn't expose stats directly
		// Could implement custom stats tracking if needed
	} else {
		stats["connected"] = false
	}
	
	return stats
}

// Ping sends a ping to test connectivity
func (qc *QUICClient) Ping() error {
	if !qc.IsConnected() {
		return fmt.Errorf("not connected")
	}
	
	// Send a simple ping message
	pingData := []byte("ping")
	return qc.Send(pingData)
}

// SetKeepAlive enables or disables keep-alive
func (qc *QUICClient) SetKeepAlive(enabled bool, period time.Duration) {
	qc.config.KeepAlive = enabled
	qc.config.KeepAlivePeriod = period
} 