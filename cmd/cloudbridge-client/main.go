package main

import (
	"context"
	"crypto/tls"
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
)

const (
	maxRetries      = 5
	initialDelaySec = 1
	maxDelaySec     = 30
)

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

	// Запуск метрик и health check
	metricsServer := &http.Server{
		Addr:         *metricsAddr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Failed to start metrics server: %v", err)
		}
	}()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

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

	// Create client
	client, err := relay.NewClientFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
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
				client.Close()
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
				client.Close()
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
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
} 