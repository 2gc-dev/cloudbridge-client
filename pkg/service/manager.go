package service

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ServiceManager handles system service management
type ServiceManager struct {
	serviceName string
	execPath    string
	configPath  string
	user        string
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	ExecPath    string `yaml:"exec_path"`
	ConfigPath  string `yaml:"config_path"`
	User        string `yaml:"user"`
	WorkingDir  string `yaml:"working_dir"`
}

// NewServiceManager creates a new service manager
func NewServiceManager(config *ServiceConfig) *ServiceManager {
	if config == nil {
		config = &ServiceConfig{
			Name:        "cloudbridge-client",
			Description: "CloudBridge Relay Client",
			User:        "root",
		}
	}

	// Determine executable path
	execPath := config.ExecPath
	if execPath == "" {
		execPath, _ = os.Executable()
	}

	// Determine config path
	configPath := config.ConfigPath
	if configPath == "" {
		configPath = "/etc/cloudbridge-client/config.yaml"
	}

	return &ServiceManager{
		serviceName: config.Name,
		execPath:    execPath,
		configPath:  configPath,
		user:        config.User,
	}
}

// Install installs the service
func (sm *ServiceManager) Install(token string) error {
	switch runtime.GOOS {
	case "linux":
		return sm.installSystemd(token)
	case "windows":
		return sm.installWindows(token)
	case "darwin":
		return sm.installLaunchd(token)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Uninstall removes the service
func (sm *ServiceManager) Uninstall() error {
	switch runtime.GOOS {
	case "linux":
		return sm.uninstallSystemd()
	case "windows":
		return sm.uninstallWindows()
	case "darwin":
		return sm.uninstallLaunchd()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Start starts the service
func (sm *ServiceManager) Start() error {
	switch runtime.GOOS {
	case "linux":
		return sm.startSystemd()
	case "windows":
		return sm.startWindows()
	case "darwin":
		return sm.startLaunchd()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Stop stops the service
func (sm *ServiceManager) Stop() error {
	switch runtime.GOOS {
	case "linux":
		return sm.stopSystemd()
	case "windows":
		return sm.stopWindows()
	case "darwin":
		return sm.stopLaunchd()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Status returns the service status
func (sm *ServiceManager) Status() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return sm.statusSystemd()
	case "windows":
		return sm.statusWindows()
	case "darwin":
		return sm.statusLaunchd()
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Restart restarts the service
func (sm *ServiceManager) Restart() error {
	if err := sm.Stop(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	return sm.Start()
}

// installSystemd installs systemd service on Linux
func (sm *ServiceManager) installSystemd(token string) error {
	// Create service file content
	serviceContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
User=%s
ExecStart=%s --config %s --token %s
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`, sm.serviceName, sm.user, sm.execPath, sm.configPath, token)

	// Write service file
	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", sm.serviceName)
	        if err := os.WriteFile(servicePath, []byte(serviceContent), 0600); err != nil {
                return fmt.Errorf("failed to write service file: %w", err)
        }

	// Reload systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable service
	if err := exec.Command("systemctl", "enable", sm.serviceName).Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	return nil
}

// uninstallSystemd removes systemd service
func (sm *ServiceManager) uninstallSystemd() error {
	// Stop and disable service
	if err := exec.Command("systemctl", "stop", sm.serviceName).Run(); err != nil {
		log.Printf("Error stopping service: %v", err)
	}
	if err := exec.Command("systemctl", "disable", sm.serviceName).Run(); err != nil {
		log.Printf("Error disabling service: %v", err)
	}

	// Remove service file
	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", sm.serviceName)
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Reload systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		log.Printf("Error reloading systemd: %v", err)
	}

	return nil
}

// startSystemd starts systemd service
func (sm *ServiceManager) startSystemd() error {
	return exec.Command("systemctl", "start", sm.serviceName).Run()
}

// stopSystemd stops systemd service
func (sm *ServiceManager) stopSystemd() error {
	return exec.Command("systemctl", "stop", sm.serviceName).Run()
}

// statusSystemd returns systemd service status
func (sm *ServiceManager) statusSystemd() (string, error) {
	output, err := exec.Command("systemctl", "is-active", sm.serviceName).Output()
	if err != nil {
		return "inactive", nil
	}
	return strings.TrimSpace(string(output)), nil
}

// installWindows installs Windows service
func (sm *ServiceManager) installWindows(token string) error {
	// Create service using sc.exe
	cmd := exec.Command("sc", "create", sm.serviceName,
		"binPath=", fmt.Sprintf("\"%s --config %s --token %s\"", sm.execPath, sm.configPath, token),
		"start=", "auto",
		"DisplayName=", sm.serviceName)
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create Windows service: %w", err)
	}

	return nil
}

// uninstallWindows removes Windows service
func (sm *ServiceManager) uninstallWindows() error {
	// Stop service first
	if err := exec.Command("sc", "stop", sm.serviceName).Run(); err != nil {
		log.Printf("Error stopping Windows service: %v", err)
	}
	
	// Delete service
	return exec.Command("sc", "delete", sm.serviceName).Run()
}

// startWindows starts Windows service
func (sm *ServiceManager) startWindows() error {
	return exec.Command("sc", "start", sm.serviceName).Run()
}

// stopWindows stops Windows service
func (sm *ServiceManager) stopWindows() error {
	return exec.Command("sc", "stop", sm.serviceName).Run()
}

// statusWindows returns Windows service status
func (sm *ServiceManager) statusWindows() (string, error) {
	output, err := exec.Command("sc", "query", sm.serviceName).Output()
	if err != nil {
		return "unknown", nil
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "STATE") {
			if strings.Contains(line, "RUNNING") {
				return "active", nil
			}
			return "inactive", nil
		}
	}
	
	return "unknown", nil
}

// installLaunchd installs launchd service on macOS
func (sm *ServiceManager) installLaunchd(token string) error {
	// Create plist content
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>--config</string>
        <string>%s</string>
        <string>--token</string>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/%s.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/%s.log</string>
</dict>
</plist>
`, sm.serviceName, sm.execPath, sm.configPath, token, sm.serviceName, sm.serviceName)

	// Write plist file
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", sm.serviceName)
	        if err := os.WriteFile(plistPath, []byte(plistContent), 0600); err != nil {
                return fmt.Errorf("failed to write plist file: %w", err)
        }

	// Load service
	if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
		return fmt.Errorf("failed to load service: %w", err)
	}

	return nil
}

// uninstallLaunchd removes launchd service
func (sm *ServiceManager) uninstallLaunchd() error {
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", sm.serviceName)
	
	// Unload service
	if err := exec.Command("launchctl", "unload", plistPath).Run(); err != nil {
		log.Printf("Error unloading service: %v", err)
	}
	
	// Remove plist file
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	return nil
}

// startLaunchd starts launchd service
func (sm *ServiceManager) startLaunchd() error {
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", sm.serviceName)
	return exec.Command("launchctl", "load", plistPath).Run()
}

// stopLaunchd stops launchd service
func (sm *ServiceManager) stopLaunchd() error {
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", sm.serviceName)
	return exec.Command("launchctl", "unload", plistPath).Run()
}

// statusLaunchd returns launchd service status
func (sm *ServiceManager) statusLaunchd() (string, error) {
	output, err := exec.Command("launchctl", "list", sm.serviceName).Output()
	if err != nil {
		return "inactive", nil
	}
	
	if strings.Contains(string(output), sm.serviceName) {
		return "active", nil
	}
	return "inactive", nil
} 