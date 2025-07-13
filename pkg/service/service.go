package service

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	serviceName = "cloudbridge-client"
	configDir   = "/etc/cloudbridge-client"
	logDir      = "/var/log/cloudbridge-client"
)

// Install устанавливает службу в зависимости от ОС
func Install(binaryPath string) error {
	switch runtime.GOOS {
	case "linux":
		return installLinux(binaryPath)
	case "windows":
		return installWindows(binaryPath)
	case "darwin":
		return installMacOS(binaryPath)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Uninstall удаляет службу
func Uninstall() error {
	switch runtime.GOOS {
	case "linux":
		return uninstallLinux()
	case "windows":
		return uninstallWindows()
	case "darwin":
		return uninstallMacOS()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Start запускает службу
func Start() error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("systemctl", "start", serviceName).Run()
	case "windows":
		return exec.Command("nssm", "start", serviceName).Run()
	case "darwin":
		return exec.Command("launchctl", "start", serviceName).Run()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Stop останавливает службу
func Stop() error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("systemctl", "stop", serviceName).Run()
	case "windows":
		return exec.Command("nssm", "stop", serviceName).Run()
	case "darwin":
		return exec.Command("launchctl", "stop", serviceName).Run()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Restart перезапускает службу
func Restart() error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("systemctl", "restart", serviceName).Run()
	case "windows":
		return exec.Command("nssm", "restart", serviceName).Run()
	case "darwin":
		if err := Stop(); err != nil {
			return err
		}
		return Start()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Status возвращает статус службы
func Status() (string, error) {
	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("systemctl", "is-active", serviceName)
		output, err := cmd.Output()
		if err != nil {
			return "inactive", nil
		}
		return string(output), nil
	case "windows":
		cmd := exec.Command("nssm", "status", serviceName)
		output, err := cmd.Output()
		if err != nil {
			return "inactive", nil
		}
		return string(output), nil
	case "darwin":
		cmd := exec.Command("launchctl", "list", serviceName)
		output, err := cmd.Output()
		if err != nil {
			return "inactive", nil
		}
		return string(output), nil
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Вспомогательные функции для установки на разных ОС
func installLinux(binaryPath string) error {
	// Копируем бинарный файл
	if err := os.MkdirAll("/usr/local/bin", 0750); err != nil {
		return err
	}
	if err := copyFile(binaryPath, "/usr/local/bin/"+serviceName); err != nil {
		return err
	}

	// Копируем файл службы
	if err := os.MkdirAll("/etc/systemd/system", 0750); err != nil {
		return err
	}
	serviceContent := `[Unit]
Description=CloudBridge Client Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/cloudbridge-client
Restart=always
RestartSec=5
Environment=CONFIG_FILE=/etc/cloudbridge-client/config.yaml

[Install]
WantedBy=multi-user.target`

	        if err := os.WriteFile("/etc/systemd/system/"+serviceName+".service", []byte(serviceContent), 0600); err != nil {
                return err
        }

	// Перезагружаем systemd и включаем службу
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return err
	}
	if err := exec.Command("systemctl", "enable", serviceName).Run(); err != nil {
		return err
	}
	return nil
}

func installWindows(binaryPath string) error {
	// Проверяем наличие NSSM
	if _, err := exec.LookPath("nssm"); err != nil {
		return fmt.Errorf("nssm not found. Please install it first")
	}

	// Устанавливаем службу через NSSM
	cmd := exec.Command("nssm", "install", serviceName, binaryPath)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Настраиваем параметры службы
	configPath := filepath.Join(os.Getenv("ProgramData"), "cloudbridge-client", "config.yaml")
	if err := exec.Command("nssm", "set", serviceName, "AppParameters", "--config", configPath).Run(); err != nil {
		log.Printf("Error setting app parameters: %v", err)
	}
	if err := exec.Command("nssm", "set", serviceName, "DisplayName", "CloudBridge Client").Run(); err != nil {
		log.Printf("Error setting display name: %v", err)
	}
	if err := exec.Command("nssm", "set", serviceName, "Description", "CloudBridge Client Service").Run(); err != nil {
		log.Printf("Error setting description: %v", err)
	}
	if err := exec.Command("nssm", "set", serviceName, "Start", "SERVICE_AUTO_START").Run(); err != nil {
		log.Printf("Error setting start mode: %v", err)
	}

	return nil
}

func installMacOS(binaryPath string) error {
	// Создаем необходимые директории
	dirs := []string{"/usr/local/bin", "/Library/LaunchDaemons", logDir}
	        for _, dir := range dirs {
                if err := os.MkdirAll(dir, 0750); err != nil {
                        return err
                }
        }

	// Копируем бинарный файл
	if err := copyFile(binaryPath, "/usr/local/bin/"+serviceName); err != nil {
		return err
	}

	// Создаем plist файл
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.cloudbridge.client</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/cloudbridge-client</string>
        <string>--config</string>
        <string>/etc/cloudbridge-client/config.yaml</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardErrorPath</key>
    <string>/var/log/cloudbridge-client/error.log</string>
    <key>StandardOutPath</key>
    <string>/var/log/cloudbridge-client/output.log</string>
    <key>WorkingDirectory</key>
    <string>/usr/local/bin</string>
</dict>
</plist>`

	        if err := os.WriteFile("/Library/LaunchDaemons/com.cloudbridge.client.plist", []byte(plistContent), 0600); err != nil {
                return err
        }

	// Загружаем службу
	return exec.Command("launchctl", "load", "/Library/LaunchDaemons/com.cloudbridge.client.plist").Run()
}

// Вспомогательные функции для удаления службы
func uninstallLinux() error {
	if err := exec.Command("systemctl", "stop", serviceName).Run(); err != nil {
		log.Printf("Error stopping service: %v", err)
	}
	if err := exec.Command("systemctl", "disable", serviceName).Run(); err != nil {
		log.Printf("Error disabling service: %v", err)
	}
	if err := os.Remove("/etc/systemd/system/" + serviceName + ".service"); err != nil {
		log.Printf("Error removing service file: %v", err)
	}
	if err := os.Remove("/usr/local/bin/" + serviceName); err != nil {
		log.Printf("Error removing binary: %v", err)
	}
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return err
	}
	return nil
}

func uninstallWindows() error {
	if err := exec.Command("nssm", "stop", serviceName).Run(); err != nil {
		log.Printf("Error stopping service: %v", err)
	}
	return exec.Command("nssm", "remove", serviceName, "confirm").Run()
}

func uninstallMacOS() error {
	if err := exec.Command("launchctl", "unload", "/Library/LaunchDaemons/com.cloudbridge.client.plist").Run(); err != nil {
		log.Printf("Error unloading service: %v", err)
	}
	if err := os.Remove("/Library/LaunchDaemons/com.cloudbridge.client.plist"); err != nil {
		log.Printf("Error removing plist file: %v", err)
	}
	if err := os.Remove("/usr/local/bin/" + serviceName); err != nil {
		log.Printf("Error removing binary: %v", err)
	}
	return nil
}

// Вспомогательная функция для копирования файлов
func copyFile(src, dst string) error {
	// Validate source path to prevent directory traversal
	cleanSrc := filepath.Clean(src)
	if !filepath.IsAbs(cleanSrc) || strings.Contains(cleanSrc, "..") {
		return fmt.Errorf("invalid source path: %s", src)
	}
	
	// Additional security check - ensure source is within allowed directories
	allowedDirs := []string{"/tmp", "/var/tmp", "/usr/local/bin", "/opt"}
	allowed := false
	for _, dir := range allowedDirs {
		if strings.HasPrefix(cleanSrc, dir) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("source path not in allowed directories: %s", src)
	}
	
	input, err := os.ReadFile(cleanSrc)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0600)
} 