package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/2gc-dev/cloudbridge-client/pkg/config"
	"github.com/2gc-dev/cloudbridge-client/pkg/health"
	"github.com/2gc-dev/cloudbridge-client/pkg/relay"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

var (
	version = "1.0.0"
	configFile string
	token      string
	tunnelID   string
	localPort  int
	remoteHost string
	remotePort int
	verbose    bool
	
	// Global variables for health checks
	healthChecker *health.HealthChecker
	relayClient   *relay.Client
	appConfig     *config.Config
)

const (
	maxRetries      = 5
	initialDelaySec = 1
	maxDelaySec     = 30
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                    `json:"status"`
	Timestamp time.Time                 `json:"timestamp"`
	Version   string                    `json:"version"`
	Uptime    time.Duration             `json:"uptime"`
	Checks    map[string]*health.HealthCheck `json:"checks"`
	Metadata  map[string]interface{}    `json:"metadata"`
}

var startTime = time.Now()

// healthHandler handles health check requests
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	response := HealthResponse{
		Status:    string(healthChecker.GetStatus()),
		Timestamp: time.Now(),
		Version:   version,
		Uptime:    time.Since(startTime),
		Checks:    healthChecker.GetResults(),
		Metadata: map[string]interface{}{
			"go_version": runtime.Version(),
			"platform":   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			"goroutines": runtime.NumGoroutine(),
		},
	}
	
	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if response.Status == string(health.Unhealthy) {
		statusCode = http.StatusServiceUnavailable
	} else if response.Status == string(health.Degraded) {
		statusCode = http.StatusOK // Degraded is still OK for HTTP
	}
	
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding health response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// readyHandler handles readiness check
func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Check if client is connected and tunnel is active
	isReady := relayClient != nil && relayClient.IsConnected()
	
	response := map[string]interface{}{
		"ready":     isReady,
		"timestamp": time.Now(),
		"status":    "ready",
	}
	
	if !isReady {
		response["status"] = "not_ready"
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding ready response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// liveHandler handles liveness check
func liveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	response := map[string]interface{}{
		"alive":    true,
		"timestamp": time.Now(),
		"status":    "alive",
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding live response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// setupHealthChecks initializes health checks
func setupHealthChecks(cfg *config.Config) {
	healthConfig := &health.Config{
		Interval: 30 * time.Second,
		Timeout:  10 * time.Second,
	}
	
	healthChecker = health.NewHealthChecker(healthConfig)
	
	// Add health checks
	healthChecker.AddCheck("relay_connection", func(ctx context.Context) (*health.HealthCheck, error) {
		if relayClient == nil {
			return &health.HealthCheck{
				Name:        "relay_connection",
				Description: "Connection to relay server",
				Status:      health.Unhealthy,
				LastCheck:   time.Now(),
				LastError:   fmt.Errorf("client not initialized"),
			}, nil
		}
		
		if !relayClient.IsConnected() {
			return &health.HealthCheck{
				Name:        "relay_connection",
				Description: "Connection to relay server",
				Status:      health.Unhealthy,
				LastCheck:   time.Now(),
				LastError:   fmt.Errorf("not connected to relay server"),
			}, nil
		}
		
		return &health.HealthCheck{
			Name:        "relay_connection",
			Description: "Connection to relay server",
			Status:      health.Healthy,
			LastCheck:   time.Now(),
		}, nil
	})
	
	// Add tunnel health check
	healthChecker.AddCheck("tunnel_status", func(ctx context.Context) (*health.HealthCheck, error) {
		if relayClient == nil {
			return &health.HealthCheck{
				Name:        "tunnel_status",
				Description: "Tunnel status",
				Status:      health.Unhealthy,
				LastCheck:   time.Now(),
				LastError:   fmt.Errorf("client not initialized"),
			}, nil
		}
		
		// This would need to be implemented in the relay client
		// For now, we'll assume it's healthy if connected
		return &health.HealthCheck{
			Name:        "tunnel_status",
			Description: "Tunnel status",
			Status:      health.Healthy,
			LastCheck:   time.Now(),
		}, nil
	})
	
	// Add metrics health check
	healthChecker.AddCheck("metrics_endpoint", func(ctx context.Context) (*health.HealthCheck, error) {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get("http://localhost:9090/metrics")
		if err != nil {
			return &health.HealthCheck{
				Name:        "metrics_endpoint",
				Description: "Metrics endpoint availability",
				Status:      health.Unhealthy,
				LastCheck:   time.Now(),
				LastError:   err,
			}, nil
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return &health.HealthCheck{
				Name:        "metrics_endpoint",
				Description: "Metrics endpoint availability",
				Status:      health.Unhealthy,
				LastCheck:   time.Now(),
				LastError:   fmt.Errorf("metrics endpoint returned status %d", resp.StatusCode),
			}, nil
		}
		
		return &health.HealthCheck{
			Name:        "metrics_endpoint",
			Description: "Metrics endpoint availability",
			Status:      health.Healthy,
			LastCheck:   time.Now(),
		}, nil
	})
	
	// Start health checker
	healthChecker.Start()
}

func main() {
	// Если есть аргументы командной строки, обрабатываем их как команды
	if len(os.Args) > 1 {
		if err := parseCommand(); err != nil {
			log.Fatalf("Command error: %v", err)
		}
		return
	}

	// Оригинальные флаги
	configPath := flag.String("config", "", "Path to config file")
	logFilePath := flag.String("logfile", "/var/log/cloudbridge-client/client.log", "Path to log file")
	metricsAddr := flag.String("metrics-addr", ":9090", "Address to serve metrics on")
	flag.Parse()

	// Логирование в файл и консоль
	logFile, err := os.OpenFile(*logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			log.Printf("Error closing log file: %v", err)
		}
	}()
	log.SetOutput(os.Stdout) // Упростим логирование

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	appConfig = cfg

	// Setup health checks
	setupHealthChecks(cfg)

	// Запуск метрик и health check
	metricsServer := &http.Server{
		Addr:         *metricsAddr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.Handle("/health", http.HandlerFunc(healthHandler))
		http.Handle("/ready", http.HandlerFunc(readyHandler))
		http.Handle("/live", http.HandlerFunc(liveHandler))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Failed to start metrics server: %v", err)
		}
	}()

	log.Printf("Running on %s/%s", runtime.GOOS, runtime.GOARCH)

	var tlsConfig *tls.Config
	if cfg.TLS.Enabled {
		tlsConfig, err = relay.NewTLSConfig(cfg.TLS.CertFile, cfg.TLS.KeyFile, cfg.TLS.CAFile)
		if err != nil {
			log.Fatalf("Failed to create TLS config: %v", err)
		}
	}

	sigChan := make(chan os.Signal, 1)
	if runtime.GOOS == "windows" {
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	} else {
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	}

	go func() {
		retries := 0
		delay := initialDelaySec
		for {
			start := time.Now()
			client := relay.NewClient(cfg.TLS.Enabled, tlsConfig)
			relayClient = client // Set global variable for health checks
			
			if err := client.Connect(cfg.Server.Host, cfg.Server.Port); err != nil {
				log.Printf("Failed to connect to relay server: %v", err)
				retries++
				if retries > maxRetries {
					log.Fatalf("Max reconnect attempts reached. Exiting.")
				}
				log.Printf("Retrying in %d seconds...", delay)
				time.Sleep(time.Duration(delay) * time.Second)
				delay = min(delay*2, maxDelaySec)
				continue
			}
			retries = 0
			delay = initialDelaySec
			defer func() {
				if err := client.Close(); err != nil {
					log.Printf("Error closing client: %v", err)
				}
			}()

			if err := client.Handshake(cfg.Server.JWTToken); err != nil {
											log.Printf("Handshake failed: %v", err)
							if err := client.Close(); err != nil {
								log.Printf("Error closing client: %v", err)
							}
							retries++
				if retries > maxRetries {
					log.Fatalf("Max reconnect attempts reached. Exiting.")
				}
				log.Printf("Retrying in %d seconds...", delay)
				time.Sleep(time.Duration(delay) * time.Second)
				delay = min(delay*2, maxDelaySec)
				continue
			}

			log.Printf("Connected successfully in %v", time.Since(start))

			// Создание туннеля
			tunnelID, err := client.CreateTunnel(localPort, remoteHost, remotePort)
			if err != nil {
											log.Printf("Failed to create tunnel: %v", err)
							if err := client.Close(); err != nil {
								log.Printf("Error closing client: %v", err)
							}
							retries++
				if retries > maxRetries {
					log.Fatalf("Max reconnect attempts reached. Exiting.")
				}
				log.Printf("Retrying in %d seconds...", delay)
				time.Sleep(time.Duration(delay) * time.Second)
				delay = min(delay*2, maxDelaySec)
				continue
			}

			log.Printf("Tunnel created: %s -> %s:%d", tunnelID, remoteHost, remotePort)

			// Ожидание сигнала завершения
			<-sigChan
									log.Println("Shutting down...")
						if err := client.Close(); err != nil {
							log.Printf("Error closing client: %v", err)
						}
						return
		}
	}()

	// Ожидание сигнала завершения
	<-sigChan
	log.Println("Shutting down...")
	
	// Stop health checker
	if healthChecker != nil {
		healthChecker.Stop()
	}
}

func parseCommand() error {
	rootCmd := &cobra.Command{
		Use:   "cloudbridge-client",
		Short: "CloudBridge Relay Client",
		Long:  "A cross-platform client for CloudBridge Relay with TLS 1.3 support and JWT authentication",
		RunE:  run,
	}

	// Add flags
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
	rootCmd.Flags().StringVarP(&token, "token", "t", "", "JWT token for authentication")
	rootCmd.Flags().StringVarP(&tunnelID, "tunnel-id", "i", "tunnel_001", "Tunnel ID")
	rootCmd.Flags().IntVarP(&localPort, "local-port", "l", 3389, "Local port to bind")
	rootCmd.Flags().StringVarP(&remoteHost, "remote-host", "r", "192.168.1.100", "Remote host")
	rootCmd.Flags().IntVarP(&remotePort, "remote-port", "p", 3389, "Remote port")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	// Mark required flags
	if err := rootCmd.MarkFlagRequired("token"); err != nil {
		return fmt.Errorf("failed to mark token flag as required: %w", err)
	}

	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) error {
	// Log platform information
	log.Printf("Running on %s/%s", runtime.GOOS, runtime.GOARCH)

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override config with command line flags if provided
	if token != "" {
		cfg.Auth.Secret = token // For JWT auth, secret is the token
	}

	// Setup health checks
	setupHealthChecks(cfg)

	// Start HTTP server for metrics and health checks
	if cfg.Metrics.Enabled {
		metricsAddr := fmt.Sprintf(":%d", cfg.Metrics.Port)
		metricsServer := &http.Server{
			Addr:         metricsAddr,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		
		go func() {
			http.Handle(cfg.Metrics.Path, promhttp.Handler())
			http.Handle(cfg.Health.Path, http.HandlerFunc(healthHandler))
			http.Handle("/ready", http.HandlerFunc(readyHandler))
			http.Handle("/live", http.HandlerFunc(liveHandler))
			
			log.Printf("Starting metrics server on %s", metricsAddr)
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Failed to start metrics server: %v", err)
			}
		}()
	}

	// Create client
	client, err := relay.NewClientFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	relayClient = client // Set global variable for health checks
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Error closing client: %v", err)
		}
	}()

	// Set up signal handling for graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	if runtime.GOOS == "windows" {
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	} else {
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	}

	go func() {
		retries := 0
		delay := initialDelaySec
		for {
			start := time.Now()
			if err := client.Connect(cfg.Server.Host, cfg.Server.Port); err != nil {
				log.Printf("Failed to connect to relay server: %v", err)
				retries++
				if retries > maxRetries {
					log.Fatalf("Max reconnect attempts reached. Exiting.")
				}
				log.Printf("Retrying in %d seconds...", delay)
				time.Sleep(time.Duration(delay) * time.Second)
				delay = min(delay*2, maxDelaySec)
				continue
			}
			retries = 0
			delay = initialDelaySec

			if err := client.Handshake(cfg.Server.JWTToken); err != nil {
				log.Printf("Handshake failed: %v", err)
				if closeErr := client.Close(); closeErr != nil {
					log.Printf("Error closing client after handshake failure: %v", closeErr)
				}
				retries++
				if retries > maxRetries {
					log.Fatalf("Max reconnect attempts reached. Exiting.")
				}
				log.Printf("Retrying in %d seconds...", delay)
				time.Sleep(time.Duration(delay) * time.Second)
				delay = min(delay*2, maxDelaySec)
				continue
			}

			log.Printf("Connected successfully in %v", time.Since(start))

			// Создание туннеля
			tunnelID, err := client.CreateTunnel(localPort, remoteHost, remotePort)
			if err != nil {
				log.Printf("Failed to create tunnel: %v", err)
				if closeErr := client.Close(); closeErr != nil {
					log.Printf("Error closing client after tunnel creation failure: %v", closeErr)
				}
				retries++
				if retries > maxRetries {
					log.Fatalf("Max reconnect attempts reached. Exiting.")
				}
				log.Printf("Retrying in %d seconds...", delay)
				time.Sleep(time.Duration(delay) * time.Second)
				delay = min(delay*2, maxDelaySec)
				continue
			}

			log.Printf("Tunnel created: %s -> %s:%d", tunnelID, remoteHost, remotePort)

			// Ожидание сигнала завершения
			<-sigChan
									log.Println("Shutting down...")
						if err := client.Close(); err != nil {
							log.Printf("Error closing client: %v", err)
						}
						return
		}
	}()

	// Ожидание сигнала завершения
	<-sigChan
	log.Println("Shutting down...")
	
	// Stop health checker
	if healthChecker != nil {
		healthChecker.Stop()
	}
	
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
} 