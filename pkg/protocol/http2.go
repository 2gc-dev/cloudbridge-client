package protocol

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/http2"
)

// HTTP2Client represents an HTTP/2 connection client
type HTTP2Client struct {
	client  *http.Client
	config  *HTTP2Config
	baseURL string
}

// HTTP2Config holds HTTP/2-specific configuration
type HTTP2Config struct {
	TLSConfig        *tls.Config
	Timeout          time.Duration
	KeepAlive        bool
	KeepAlivePeriod  time.Duration
	MaxIdleConns     int
	IdleConnTimeout  time.Duration
}

// DefaultHTTP2Config returns default HTTP/2 configuration
func DefaultHTTP2Config() *HTTP2Config {
	return &HTTP2Config{
		Timeout:          30 * time.Second,
		KeepAlive:        true,
		KeepAlivePeriod:  30 * time.Second,
		MaxIdleConns:     100,
		IdleConnTimeout:  90 * time.Second,
	}
}

// NewHTTP2Client creates a new HTTP/2 client
func NewHTTP2Client(config *HTTP2Config) *HTTP2Client {
	if config == nil {
		config = DefaultHTTP2Config()
	}
	
	// Create HTTP/2 transport
	transport := &http2.Transport{
		TLSClientConfig: config.TLSConfig,
		AllowHTTP:       false, // Require TLS for HTTP/2
	}
	
	// Create HTTP client
	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}
	
	return &HTTP2Client{
		client: client,
		config: config,
	}
}

// Connect establishes an HTTP/2 connection (validates connectivity)
func (hc *HTTP2Client) Connect(ctx context.Context, address string) error {
	hc.baseURL = fmt.Sprintf("https://%s", address)
	
	// Test connection with a simple request
	req, err := http.NewRequestWithContext(ctx, "GET", hc.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := hc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()
	
	// Check if HTTP/2 is being used
	if resp.ProtoMajor != 2 {
		return fmt.Errorf("server does not support HTTP/2, got HTTP/%d", resp.ProtoMajor)
	}
	
	return nil
}

// Send sends data via HTTP/2 POST request
func (hc *HTTP2Client) Send(data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "POST", hc.baseURL+"/data", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Body = io.NopCloser(strings.NewReader(string(data)))
	req.ContentLength = int64(len(data))
	req.Header.Set("Content-Type", "application/octet-stream")
	
	resp, err := hc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return nil
}

// Receive receives data via HTTP/2 GET request
func (hc *HTTP2Client) Receive(buffer []byte) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", hc.baseURL+"/data", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := hc.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to receive request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return io.ReadFull(resp.Body, buffer)
}

// Close closes the HTTP/2 client
func (hc *HTTP2Client) Close() error {
	// HTTP client doesn't need explicit closing
	// Transport will handle connection cleanup
	return nil
}

// IsConnected returns true if the client can make requests
func (hc *HTTP2Client) IsConnected() bool {
	// Test connectivity with a simple request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", hc.baseURL+"/health", nil)
	if err != nil {
		return false
	}
	
	resp, err := hc.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

// GetConnectionState returns the TLS connection state
func (hc *HTTP2Client) GetConnectionState() tls.ConnectionState {
	// For HTTP/2, we need to make a request to get the connection state
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", hc.baseURL+"/health", nil)
	if err != nil {
		return tls.ConnectionState{}
	}
	
	resp, err := hc.client.Do(req)
	if err != nil {
		return tls.ConnectionState{}
	}
	defer resp.Body.Close()
	
	// Try to get TLS state from response
	if tlsConn, ok := resp.Body.(interface{ ConnectionState() tls.ConnectionState }); ok {
		return tlsConn.ConnectionState()
	}
	
	return tls.ConnectionState{}
}

// GetStats returns HTTP/2 connection statistics
func (hc *HTTP2Client) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	stats["connected"] = hc.IsConnected()
	stats["base_url"] = hc.baseURL
	stats["timeout"] = hc.config.Timeout.String()
	stats["keep_alive"] = hc.config.KeepAlive
	
	// Get transport stats if available
	if _, ok := hc.client.Transport.(*http2.Transport); ok {
		stats["transport_type"] = "http2"
		// Note: http2.Transport doesn't expose detailed stats
	}
	
	return stats
}

// Ping sends a ping request to test connectivity
func (hc *HTTP2Client) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", hc.baseURL+"/ping", nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}
	
	resp, err := hc.client.Do(req)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping returned status: %d", resp.StatusCode)
	}
	
	return nil
}

// SetKeepAlive enables or disables keep-alive
func (hc *HTTP2Client) SetKeepAlive(enabled bool, period time.Duration) {
	hc.config.KeepAlive = enabled
	hc.config.KeepAlivePeriod = period
} 