package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/2gc-dev/cloudbridge-client/pkg/client"
	"github.com/2gc-dev/cloudbridge-client/pkg/config"
	"github.com/2gc-dev/cloudbridge-client/pkg/health"
	"github.com/2gc-dev/cloudbridge-client/pkg/protocol"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config/config-v2.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create client configuration
	clientConfig := &client.Config{
		TLSConfig:        nil, // Will be created from config
		CircuitBreaker:   nil, // Will use defaults
		ProtocolOrder:    []protocol.Protocol{protocol.QUIC, protocol.HTTP2, protocol.HTTP1},
		SwitchThreshold:  0.8,
		ConnectTimeout:   10 * time.Second,
		RequestTimeout:   30 * time.Second,
		TenantID:         cfg.Tenant.ID,
		Version:          cfg.Protocol.Version,
		Features:         cfg.Protocol.Features,
		MetricsEnabled:   true,
		HealthCheckEnabled: true,
		HealthCheckConfig: &health.Config{
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
		},
	}

	// Create integrated client
	ic := client.NewIntegratedClient(clientConfig)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics update goroutine
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				if metrics := ic.GetMetrics(); metrics != nil {
					metrics.UpdateClientUptime()
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Connect to relay server
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Connecting to relay server at %s", serverAddr)
	
	if err := ic.Connect(ctx, serverAddr); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	
	log.Printf("Connected successfully using protocol: %s", ic.GetCurrentProtocol())

	// Perform handshake
	if err := performHandshake(ic, cfg.Server.JWTToken); err != nil {
		log.Fatalf("Handshake failed: %v", err)
	}
	log.Println("Handshake completed successfully")

	// Start health monitoring
	go monitorHealth(ic)

	// Start metrics monitoring
	go monitorMetrics(ic)

	// Main loop - keep connection alive
	log.Println("Client running. Press Ctrl+C to stop.")
	
	for {
		select {
		case <-sigChan:
			log.Println("Received shutdown signal")
			return
		case <-time.After(30 * time.Second):
			// Send periodic ping
			if err := ic.Ping(); err != nil {
				log.Printf("Ping failed: %v", err)
				// Try to reconnect
				if err := ic.Connect(ctx, serverAddr); err != nil {
					log.Printf("Reconnection failed: %v", err)
				} else {
					log.Println("Reconnected successfully")
				}
			}
		}
	}
}

func performHandshake(ic *client.IntegratedClient, token string) error {
	// This would typically be done through the relay client
	// For this example, we'll just log the attempt
	log.Printf("Performing handshake with token: %s", token[:10]+"...")
	log.Printf("Tenant ID: %s", ic.GetTenantID())
	log.Printf("Protocol version: %s", ic.GetVersion())
	log.Printf("Features: %v", ic.GetFeatures())
	return nil
}

func monitorHealth(ic *client.IntegratedClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if healthChecker := ic.GetHealthChecker(); healthChecker != nil {
				status := healthChecker.GetStatus()
				results := healthChecker.GetResults()
				
				log.Printf("Health status: %s", status)
				for name, result := range results {
					log.Printf("  %s: %s (%.2fs)", name, result.Status, result.Duration.Seconds())
					if result.LastError != nil {
						log.Printf("    Error: %v", result.LastError)
					}
				}
			}
		}
	}
}

func monitorMetrics(ic *client.IntegratedClient) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if metrics := ic.GetMetrics(); metrics != nil {
				summary := metrics.GetMetricsSummary()
				log.Printf("Metrics summary: %+v", summary)
			}
			
			stats := ic.GetStats()
			log.Printf("Client stats: %+v", stats)
		}
	}
}

// Example of creating a tunnel
func createTunnel(ic *client.IntegratedClient, localPort int, remoteHost string, remotePort int) error {
	// This would be implemented based on the specific tunnel creation logic
	log.Printf("Creating tunnel: local:%d -> %s:%d", localPort, remoteHost, remotePort)
	
	// For this example, we'll just simulate tunnel creation
	time.Sleep(1 * time.Second)
	
	if metrics := ic.GetMetrics(); metrics != nil {
		metrics.IncTunnelCreations()
		metrics.SetTunnelStatus("example_tunnel", true)
	}
	
	log.Println("Tunnel created successfully")
	return nil
}

// Example of sending data through tunnel
func sendData(ic *client.IntegratedClient, data []byte) error {
	log.Printf("Sending %d bytes of data", len(data))
	
	if err := ic.Send(data); err != nil {
		log.Printf("Failed to send data: %v", err)
		return err
	}
	
	log.Println("Data sent successfully")
	return nil
}

// Example of receiving data from tunnel
func receiveData(ic *client.IntegratedClient, buffer []byte) (int, error) {
	n, err := ic.Receive(buffer)
	if err != nil {
		log.Printf("Failed to receive data: %v", err)
		return 0, err
	}
	
	log.Printf("Received %d bytes of data", n)
	return n, nil
} 