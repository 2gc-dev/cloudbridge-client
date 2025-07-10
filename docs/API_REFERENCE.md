# CloudBridge Client - API Reference

## Содержание

1. [Обзор](#обзор)
2. [Основные интерфейсы](#основные-интерфейсы)
3. [Структуры данных](#структуры-данных)
4. [Конфигурация](#конфигурация)
5. [Ошибки](#ошибки)
6. [Примеры использования](#примеры-использования)

## Обзор

CloudBridge Client предоставляет набор Go интерфейсов и структур для подключения к CloudBridge Relay серверу, аутентификации и создания туннелей.

### Основные пакеты

- `pkg/client` - Основной клиент для работы с relay сервером
- `pkg/config` - Конфигурация и настройки
- `pkg/auth` - Аутентификация и авторизация
- `pkg/tunnel` - Управление туннелями
- `pkg/metrics` - Метрики и мониторинг
- `pkg/errors` - Обработка ошибок

## Основные интерфейсы

### IntegratedClient

Основной интерфейс для работы с CloudBridge Relay сервером.

```go
type IntegratedClient interface {
    // Connect устанавливает соединение с relay сервером
    Connect(ctx context.Context) error
    
    // Close закрывает соединение
    Close() error
    
    // CreateTunnel создает новый туннель
    CreateTunnel(localPort int, remoteHost string, remotePort int) (string, error)
    
    // GetTunnelStatus возвращает статус туннеля
    GetTunnelStatus(tunnelID string) (*TunnelStatus, error)
    
    // CloseTunnel закрывает туннель
    CloseTunnel(tunnelID string) error
    
    // GetMetrics возвращает метрики клиента
    GetMetrics() (*Metrics, error)
    
    // IsConnected проверяет статус подключения
    IsConnected() bool
    
    // GetConnectionInfo возвращает информацию о соединении
    GetConnectionInfo() *ConnectionInfo
}
```

#### Методы

##### Connect(ctx context.Context) error

Устанавливает соединение с relay сервером и выполняет handshake.

**Параметры:**
- `ctx context.Context` - Контекст с таймаутом

**Возвращает:**
- `error` - Ошибка подключения или nil при успехе

**Пример:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := client.Connect(ctx)
if err != nil {
    log.Fatal("Failed to connect:", err)
}
```

##### Close() error

Закрывает соединение с relay сервером.

**Возвращает:**
- `error` - Ошибка закрытия или nil при успехе

**Пример:**
```go
defer client.Close()
```

##### CreateTunnel(localPort int, remoteHost string, remotePort int) (string, error)

Создает новый TCP туннель.

**Параметры:**
- `localPort int` - Локальный порт для туннеля
- `remoteHost string` - Удаленный хост
- `remotePort int` - Удаленный порт

**Возвращает:**
- `string` - ID созданного туннеля
- `error` - Ошибка создания или nil при успехе

**Пример:**
```go
tunnelID, err := client.CreateTunnel(3389, "192.168.1.100", 3389)
if err != nil {
    log.Fatal("Failed to create tunnel:", err)
}
log.Printf("Tunnel created: %s", tunnelID)
```

##### GetTunnelStatus(tunnelID string) (*TunnelStatus, error)

Возвращает статус туннеля.

**Параметры:**
- `tunnelID string` - ID туннеля

**Возвращает:**
- `*TunnelStatus` - Статус туннеля
- `error` - Ошибка получения или nil при успехе

**Пример:**
```go
status, err := client.GetTunnelStatus("tunnel_001")
if err != nil {
    log.Fatal("Failed to get tunnel status:", err)
}
log.Printf("Tunnel status: %s", status.Status)
```

##### CloseTunnel(tunnelID string) error

Закрывает туннель.

**Параметры:**
- `tunnelID string` - ID туннеля

**Возвращает:**
- `error` - Ошибка закрытия или nil при успехе

**Пример:**
```go
err := client.CloseTunnel("tunnel_001")
if err != nil {
    log.Fatal("Failed to close tunnel:", err)
}
```

##### GetMetrics() (*Metrics, error)

Возвращает метрики клиента.

**Возвращает:**
- `*Metrics` - Метрики клиента
- `error` - Ошибка получения или nil при успехе

**Пример:**
```go
metrics, err := client.GetMetrics()
if err != nil {
    log.Fatal("Failed to get metrics:", err)
}
log.Printf("Active connections: %d", metrics.ActiveConnections)
```

### RelayClient

Интерфейс для низкоуровневой работы с relay сервером.

```go
type RelayClient interface {
    // Connect устанавливает TCP соединение
    Connect(host string, port int) error
    
    // Handshake выполняет протокол handshake
    Handshake(token, version string) error
    
    // SendMessage отправляет JSON сообщение
    SendMessage(msg interface{}) error
    
    // ReadMessage читает JSON сообщение
    ReadMessage() (map[string]interface{}, error)
    
    // Close закрывает соединение
    Close() error
    
    // IsConnected проверяет статус подключения
    IsConnected() bool
}
```

#### Методы

##### Connect(host string, port int) error

Устанавливает TCP соединение с relay сервером.

**Параметры:**
- `host string` - Хост сервера
- `port int` - Порт сервера

**Возвращает:**
- `error` - Ошибка подключения или nil при успехе

##### Handshake(token, version string) error

Выполняет протокол handshake с сервером.

**Параметры:**
- `token string` - JWT токен для аутентификации
- `version string` - Версия протокола

**Возвращает:**
- `error` - Ошибка handshake или nil при успехе

##### SendMessage(msg interface{}) error

Отправляет JSON сообщение на сервер.

**Параметры:**
- `msg interface{}` - Сообщение для отправки

**Возвращает:**
- `error` - Ошибка отправки или nil при успехе

##### ReadMessage() (map[string]interface{}, error)

Читает JSON сообщение от сервера.

**Возвращает:**
- `map[string]interface{}` - Полученное сообщение
- `error` - Ошибка чтения или nil при успехе

## Структуры данных

### TunnelStatus

Статус туннеля.

```go
type TunnelStatus struct {
    ID            string    `json:"id"`              // ID туннеля
    LocalPort     int       `json:"local_port"`      // Локальный порт
    RemoteHost    string    `json:"remote_host"`     // Удаленный хост
    RemotePort    int       `json:"remote_port"`     // Удаленный порт
    Status        string    `json:"status"`          // Статус: active, inactive, error
    CreatedAt     time.Time `json:"created_at"`      // Время создания
    LastActivity  time.Time `json:"last_activity"`   // Последняя активность
    BytesSent     int64     `json:"bytes_sent"`      // Отправлено байт
    BytesReceived int64     `json:"bytes_received"`  // Получено байт
    Error         string    `json:"error,omitempty"` // Ошибка (если есть)
}
```

**Пример:**
```go
status := &TunnelStatus{
    ID:           "tunnel_001",
    LocalPort:    3389,
    RemoteHost:   "192.168.1.100",
    RemotePort:   3389,
    Status:       "active",
    CreatedAt:    time.Now(),
    LastActivity: time.Now(),
    BytesSent:    1024,
    BytesReceived: 2048,
}
```

### Metrics

Метрики клиента.

```go
type Metrics struct {
    ConnectionsTotal    int64         `json:"connections_total"`    // Всего подключений
    ActiveConnections   int64         `json:"active_connections"`   // Активные подключения
    TunnelsCreated      int64         `json:"tunnels_created"`      // Создано туннелей
    AuthAttempts        int64         `json:"auth_attempts"`        // Попыток аутентификации
    ErrorsTotal         int64         `json:"errors_total"`         // Всего ошибок
    Uptime              time.Duration `json:"uptime"`               // Время работы
    LastError           string        `json:"last_error,omitempty"` // Последняя ошибка
}
```

**Пример:**
```go
metrics := &Metrics{
    ConnectionsTotal:  100,
    ActiveConnections: 5,
    TunnelsCreated:    10,
    AuthAttempts:      15,
    ErrorsTotal:       2,
    Uptime:            3600 * time.Second,
    LastError:         "connection timeout",
}
```

### ConnectionInfo

Информация о соединении.

```go
type ConnectionInfo struct {
    Host            string    `json:"host"`             // Хост сервера
    Port            int       `json:"port"`             // Порт сервера
    ConnectedAt     time.Time `json:"connected_at"`     // Время подключения
    LastHeartbeat   time.Time `json:"last_heartbeat"`   // Последний heartbeat
    Protocol        string    `json:"protocol"`         // Протокол: tcp, tls, quic
    ClientID        string    `json:"client_id"`        // ID клиента
    SessionID       string    `json:"session_id"`       // ID сессии
}
```

**Пример:**
```go
info := &ConnectionInfo{
    Host:          "relay.example.com",
    Port:          8082,
    ConnectedAt:   time.Now(),
    LastHeartbeat: time.Now(),
    Protocol:      "tcp",
    ClientID:      "client-001",
    SessionID:     "session-123",
}
```

### Config

Конфигурация клиента.

```go
type Config struct {
    Server  ServerConfig  `yaml:"server"`  // Настройки сервера
    TLS     TLSConfig     `yaml:"tls"`     // Настройки TLS
    Auth    AuthConfig    `yaml:"auth"`    // Настройки аутентификации
    Tunnel  TunnelConfig  `yaml:"tunnel"`  // Настройки туннелей
    Logging LoggingConfig `yaml:"logging"` // Настройки логирования
    Metrics MetricsConfig `yaml:"metrics"` // Настройки метрик
}
```

### ServerConfig

Настройки сервера.

```go
type ServerConfig struct {
    Host      string        `yaml:"host"`       // Хост сервера
    Port      int           `yaml:"port"`       // Порт сервера
    JWTToken  string        `yaml:"jwt_token"`  // JWT токен
    Timeout   time.Duration `yaml:"timeout"`    // Таймаут подключения
}
```

### TLSConfig

Настройки TLS.

```go
type TLSConfig struct {
    Enabled    bool   `yaml:"enabled"`     // Включить TLS
    CertFile   string `yaml:"cert_file"`   // Файл сертификата
    KeyFile    string `yaml:"key_file"`    // Файл приватного ключа
    CAFile     string `yaml:"ca_file"`     // Файл CA сертификата
    MinVersion string `yaml:"min_version"` // Минимальная версия TLS
}
```

### AuthConfig

Настройки аутентификации.

```go
type AuthConfig struct {
    Secret   string `yaml:"secret"`    // Секрет для JWT
    Provider string `yaml:"provider"`  // Провайдер: keycloak, django
}
```

### TunnelConfig

Настройки туннелей.

```go
type TunnelConfig struct {
    LocalPort         int           `yaml:"local_port"`          // Локальный порт
    ReconnectDelay    time.Duration `yaml:"reconnect_delay"`     // Задержка переподключения
    MaxRetries        int           `yaml:"max_retries"`         // Максимум попыток
    HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`  // Интервал heartbeat
}
```

### LoggingConfig

Настройки логирования.

```go
type LoggingConfig struct {
    Level      string `yaml:"level"`       // Уровень: debug, info, warn, error
    Format     string `yaml:"format"`      // Формат: json, text
    File       string `yaml:"file"`        // Файл логов
    MaxSize    int    `yaml:"max_size"`    // Максимальный размер (MB)
    MaxBackups int    `yaml:"max_backups"` // Количество резервных копий
    MaxAge     int    `yaml:"max_age"`     // Максимальный возраст (дни)
    Compress   bool   `yaml:"compress"`    // Сжатие старых логов
}
```

### MetricsConfig

Настройки метрик.

```go
type MetricsConfig struct {
    Enabled bool   `yaml:"enabled"` // Включить метрики
    Port    int    `yaml:"port"`    // Порт для метрик
    Path    string `yaml:"path"`    // Путь к метрикам
}
```

## Конфигурация

### Загрузка конфигурации

```go
// Загрузка из файла
cfg, err := config.Load("config.yaml")
if err != nil {
    log.Fatal(err)
}

// Создание конфигурации программно
cfg := &config.Config{
    Server: config.ServerConfig{
        Host:     "relay.example.com",
        Port:     8082,
        JWTToken: "your-jwt-token",
        Timeout:  30 * time.Second,
    },
    TLS: config.TLSConfig{
        Enabled: false,
    },
    Logging: config.LoggingConfig{
        Level:  "info",
        Format: "json",
        File:   "/var/log/cloudbridge-client/client.log",
    },
}
```

### Валидация конфигурации

```go
// Валидация конфигурации
err := cfg.Validate()
if err != nil {
    log.Fatal("Invalid configuration:", err)
}
```

## Ошибки

### Типы ошибок

```go
// Ошибки подключения
var (
    ErrConnectionFailed    = errors.New("connection failed")
    ErrConnectionTimeout   = errors.New("connection timeout")
    ErrConnectionRefused   = errors.New("connection refused")
)

// Ошибки аутентификации
var (
    ErrAuthFailed         = errors.New("authentication failed")
    ErrInvalidToken       = errors.New("invalid token")
    ErrTokenExpired       = errors.New("token expired")
)

// Ошибки туннелей
var (
    ErrTunnelCreationFailed = errors.New("tunnel creation failed")
    ErrTunnelNotFound       = errors.New("tunnel not found")
    ErrTunnelClosed         = errors.New("tunnel closed")
)
```

### Обработка ошибок

```go
// Проверка типа ошибки
if errors.Is(err, ErrConnectionFailed) {
    log.Println("Connection failed, retrying...")
    // Логика повторных попыток
}

// Обертывание ошибок
if err != nil {
    return fmt.Errorf("failed to create tunnel: %w", err)
}

// Логирование ошибок
if err != nil {
    log.Printf("Error: %v", err)
    log.Printf("Error type: %T", err)
}
```

## Примеры использования

### Базовый пример

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/2gc-dev/cloudbridge-client/pkg/client"
    "github.com/2gc-dev/cloudbridge-client/pkg/config"
)

func main() {
    // Загрузка конфигурации
    cfg, err := config.Load("config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Создание клиента
    client := client.NewIntegratedClient(cfg)
    
    // Подключение с таймаутом
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    err = client.Connect(ctx)
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    defer client.Close()
    
    // Создание туннеля
    tunnelID, err := client.CreateTunnel(3389, "192.168.1.100", 3389)
    if err != nil {
        log.Fatal("Failed to create tunnel:", err)
    }
    
    log.Printf("Tunnel created: %s", tunnelID)
    
    // Мониторинг
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            // Получение метрик
            metrics, err := client.GetMetrics()
            if err != nil {
                log.Printf("Failed to get metrics: %v", err)
                continue
            }
            
            log.Printf("Active connections: %d", metrics.ActiveConnections)
            log.Printf("Tunnels created: %d", metrics.TunnelsCreated)
            
            // Проверка статуса туннеля
            status, err := client.GetTunnelStatus(tunnelID)
            if err != nil {
                log.Printf("Failed to get tunnel status: %v", err)
                continue
            }
            
            log.Printf("Tunnel status: %s", status.Status)
            log.Printf("Bytes sent: %d, received: %d", status.BytesSent, status.BytesReceived)
        }
    }
}
```

### Пример с обработкой ошибок

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/2gc-dev/cloudbridge-client/pkg/client"
    "github.com/2gc-dev/cloudbridge-client/pkg/config"
    "github.com/2gc-dev/cloudbridge-client/pkg/errors"
)

func main() {
    cfg, err := config.Load("config.yaml")
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    client := client.NewIntegratedClient(cfg)
    
    // Подключение с повторными попытками
    for i := 0; i < 3; i++ {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        
        err = client.Connect(ctx)
        if err == nil {
            break
        }
        
        cancel()
        
        if errors.Is(err, errors.ErrConnectionFailed) {
            log.Printf("Connection failed, retrying in 5 seconds... (attempt %d/3)", i+1)
            time.Sleep(5 * time.Second)
            continue
        }
        
        log.Fatal("Fatal connection error:", err)
    }
    
    defer client.Close()
    
    // Создание туннеля с обработкой ошибок
    tunnelID, err := client.CreateTunnel(3389, "192.168.1.100", 3389)
    if err != nil {
        if errors.Is(err, errors.ErrTunnelCreationFailed) {
            log.Fatal("Tunnel creation failed, check remote host availability")
        }
        log.Fatal("Unexpected error:", err)
    }
    
    log.Printf("Tunnel created successfully: %s", tunnelID)
    
    // Ожидание
    select {}
}
```

### Пример с метриками

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"
    
    "github.com/2gc-dev/cloudbridge-client/pkg/client"
    "github.com/2gc-dev/cloudbridge-client/pkg/config"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
    cfg, err := config.Load("config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    client := client.NewIntegratedClient(cfg)
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    err = client.Connect(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Запуск HTTP сервера для метрик
    go func() {
        http.Handle("/metrics", promhttp.Handler())
        http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
            if client.IsConnected() {
                w.WriteHeader(http.StatusOK)
                w.Write([]byte("OK"))
            } else {
                w.WriteHeader(http.StatusServiceUnavailable)
                w.Write([]byte("Disconnected"))
            }
        })
        
        log.Fatal(http.ListenAndServe(":9090", nil))
    }()
    
    // Создание туннеля
    tunnelID, err := client.CreateTunnel(3389, "192.168.1.100", 3389)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Tunnel created: %s", tunnelID)
    
    // Мониторинг
    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            metrics, err := client.GetMetrics()
            if err != nil {
                log.Printf("Failed to get metrics: %v", err)
                continue
            }
            
            log.Printf("=== Metrics ===")
            log.Printf("Connections total: %d", metrics.ConnectionsTotal)
            log.Printf("Active connections: %d", metrics.ActiveConnections)
            log.Printf("Tunnels created: %d", metrics.TunnelsCreated)
            log.Printf("Auth attempts: %d", metrics.AuthAttempts)
            log.Printf("Errors total: %d", metrics.ErrorsTotal)
            log.Printf("Uptime: %v", metrics.Uptime)
            
            if metrics.LastError != "" {
                log.Printf("Last error: %s", metrics.LastError)
            }
        }
    }
}
```

### Пример с несколькими туннелями

```go
package main

import (
    "context"
    "log"
    "sync"
    "time"
    
    "github.com/2gc-dev/cloudbridge-client/pkg/client"
    "github.com/2gc-dev/cloudbridge-client/pkg/config"
)

type TunnelConfig struct {
    LocalPort  int
    RemoteHost string
    RemotePort int
    Name       string
}

func main() {
    cfg, err := config.Load("config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    client := client.NewIntegratedClient(cfg)
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    err = client.Connect(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Конфигурация туннелей
    tunnels := []TunnelConfig{
        {LocalPort: 3389, RemoteHost: "192.168.1.100", RemotePort: 3389, Name: "RDP"},
        {LocalPort: 2222, RemoteHost: "192.168.1.101", RemotePort: 22, Name: "SSH"},
        {LocalPort: 8080, RemoteHost: "192.168.1.102", RemotePort: 80, Name: "Web"},
        {LocalPort: 5432, RemoteHost: "192.168.1.103", RemotePort: 5432, Name: "PostgreSQL"},
    }
    
    var wg sync.WaitGroup
    tunnelIDs := make([]string, len(tunnels))
    
    // Создание туннелей
    for i, tunnel := range tunnels {
        wg.Add(1)
        go func(i int, tunnel TunnelConfig) {
            defer wg.Done()
            
            tunnelID, err := client.CreateTunnel(tunnel.LocalPort, tunnel.RemoteHost, tunnel.RemotePort)
            if err != nil {
                log.Printf("Failed to create %s tunnel: %v", tunnel.Name, err)
                return
            }
            
            tunnelIDs[i] = tunnelID
            log.Printf("%s tunnel created: %s (localhost:%d -> %s:%d)", 
                tunnel.Name, tunnelID, tunnel.LocalPort, tunnel.RemoteHost, tunnel.RemotePort)
        }(i, tunnel)
    }
    
    wg.Wait()
    
    // Мониторинг туннелей
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            log.Printf("=== Tunnel Status ===")
            for i, tunnelID := range tunnelIDs {
                if tunnelID == "" {
                    continue
                }
                
                status, err := client.GetTunnelStatus(tunnelID)
                if err != nil {
                    log.Printf("Failed to get status for tunnel %s: %v", tunnelID, err)
                    continue
                }
                
                log.Printf("%s: %s (sent: %d, received: %d)", 
                    tunnels[i].Name, status.Status, status.BytesSent, status.BytesReceived)
            }
        }
    }
}
```

## Дополнительные ресурсы

- [Техническая спецификация](TECHNICAL_SPECIFICATION.md)
- [Руководство пользователя](USER_GUIDE.md)
- [Примеры конфигурации](../config/)
- [Тесты](../test/) 