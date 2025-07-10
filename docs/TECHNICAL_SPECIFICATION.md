# CloudBridge Client - Техническая спецификация

## 1. Обзор системы

CloudBridge Client - это Go-клиент для подключения к CloudBridge Relay серверу с поддержкой множественных протоколов, аутентификации и создания туннелей.

### 1.1 Основные возможности

- **Множественные протоколы**: QUIC, HTTP/2, HTTP/1.1
- **Аутентификация**: JWT токены с поддержкой Keycloak и Django
- **Туннелирование**: Создание и управление TCP туннелями
- **Отказоустойчивость**: Circuit breaker, rate limiting, переподключение
- **Мониторинг**: Prometheus метрики и структурированное логирование

### 1.2 Архитектура

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CloudBridge   │    │   CloudBridge   │    │   CloudBridge   │
│     Client      │◄──►│     Relay       │◄──►│   Backend       │
│                 │    │                 │    │   (Keycloak/    │
│  - Handshake    │    │  - Auth         │    │    Django)      │
│  - Tunneling    │    │  - Routing      │    │                 │
│  - Monitoring   │    │  - Metrics      │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 2. Протокол обмена сообщениями

### 2.1 Общие принципы

- **Формат**: JSON с разделителем `\n`
- **Кодировка**: UTF-8
- **Максимальный размер сообщения**: 1MB
- **Таймауты**: 30 секунд на операцию

### 2.2 Последовательность handshake

#### 2.2.1 Hello (сервер → клиент)
```json
{
  "type": "hello",
  "version": "1.0.0",
  "features": ["tls", "jwt", "tunneling", "quic", "http2"]
}
```

#### 2.2.2 Auth (клиент → сервер)
```json
{
  "type": "auth",
  "token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9...",
  "version": "1.0.0",
  "client_info": {
    "os": "linux",
    "arch": "amd64",
    "version": "1.0.0",
    "capabilities": ["tls", "quic"]
  }
}
```

#### 2.2.3 Auth Response (сервер → клиент)
```json
{
  "type": "auth_response",
  "status": "ok",
  "client_id": "client-001",
  "session_id": "session-123",
  "permissions": ["tunnel:create", "tunnel:read"]
}
```

#### 2.2.4 Tunnel Info (клиент → сервер)
```json
{
  "type": "tunnel_info",
  "local_port": 3389,
  "remote_host": "192.168.1.100",
  "remote_port": 3389,
  "protocol": "tcp",
  "options": {
    "compression": false,
    "encryption": true
  }
}
```

#### 2.2.5 Tunnel Response (сервер → клиент)
```json
{
  "type": "tunnel_response",
  "status": "ok",
  "tunnel_id": "tunnel_001",
  "public_port": 12345,
  "endpoint": "relay.example.com:12345"
}
```

### 2.3 Обработка ошибок

```json
{
  "type": "error",
  "code": "AUTH_FAILED",
  "message": "Invalid JWT token",
  "details": {
    "reason": "token_expired",
    "expires_at": "2025-01-01T00:00:00Z"
  }
}
```

## 3. Аутентификация и авторизация

### 3.1 JWT токены

#### 3.1.1 Структура токена
```json
{
  "sub": "client-001",
  "aud": "relay-client",
  "iss": "https://edge.2gc.ru/realms/cloudbridge",
  "exp": 1752021832,
  "iat": 1752018232,
  "nbf": 1752018232,
  "client_id": "relay-client",
  "scope": "tunnel:create tunnel:read"
}
```

#### 3.1.2 Валидация
- **Алгоритм**: HS256
- **Секрет**: `Aewy5jf8Omfg70VKxDCkh3F2FpH4fDIgbcrmcgVfYvE=`
- **Время жизни**: 1 час
- **Clock skew**: 30 секунд

### 3.2 Провайдеры аутентификации

#### 3.2.1 Keycloak
```yaml
keycloak:
  enabled: true
  server_url: "https://edge.2gc.ru"
  realm: "cloudbridge"
  client_id: "relay-client"
  client_secret: "${KEYCLOAK_CLIENT_SECRET}"
  public_key: "${KEYCLOAK_PUBLIC_KEY}"
```

#### 3.2.2 Django
```yaml
django:
  api_endpoint: "http://localhost:8000/api/"
  jwt_secret: "${RELAY_JWT_SECRET}"
  jwt_expiry: "24h"
```

## 4. Туннелирование

### 4.1 Типы туннелей

#### 4.1.1 TCP туннель
- **Протокол**: TCP
- **Направление**: Bidirectional
- **Сжатие**: Опционально
- **Шифрование**: TLS 1.3

#### 4.1.2 UDP туннель (планируется)
- **Протокол**: UDP
- **Направление**: Bidirectional
- **Надежность**: Retransmission

### 4.2 Управление туннелями

#### 4.2.1 Создание туннеля
```go
tunnelID, err := client.CreateTunnel(localPort, remoteHost, remotePort)
```

#### 4.2.2 Мониторинг туннеля
```go
status, err := client.GetTunnelStatus(tunnelID)
```

#### 4.2.3 Закрытие туннеля
```go
err := client.CloseTunnel(tunnelID)
```

## 5. Конфигурация

### 5.1 Основная конфигурация

```yaml
server:
  host: "localhost"
  port: 8082
  jwt_token: "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9..."

tls:
  enabled: true
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
  ca_file: "/path/to/ca.pem"
  min_version: "1.3"

auth:
  secret: "Aewy5jf8Omfg70VKxDCkh3F2FpH4fDIgbcrmcgVfYvE="
  provider: "keycloak"  # keycloak, django

tunnel:
  local_port: 3389
  reconnect_delay: 5
  max_retries: 3
  heartbeat_interval: 30

logging:
  level: "info"
  format: "json"
  file: "/var/log/cloudbridge-client/client.log"
  max_size: 100
  max_backups: 3
  max_age: 28
  compress: true

metrics:
  enabled: true
  port: 9090
  path: "/metrics"
```

### 5.2 Переменные окружения

| Переменная | Описание | Пример |
|------------|----------|--------|
| `RELAY_JWT_SECRET` | Секрет для JWT | `Aewy5jf8Omfg70VKxDCkh3F2FpH4fDIgbcrmcgVfYvE=` |
| `KEYCLOAK_SERVER_URL` | URL Keycloak сервера | `https://edge.2gc.ru` |
| `KEYCLOAK_REALM` | Realm в Keycloak | `cloudbridge` |
| `KEYCLOAK_CLIENT_ID` | ID клиента | `relay-client` |
| `KEYCLOAK_CLIENT_SECRET` | Секрет клиента | `secret` |

## 6. API интерфейсы

### 6.1 Основные интерфейсы

#### 6.1.1 IntegratedClient
```go
type IntegratedClient interface {
    Connect(ctx context.Context) error
    Close() error
    CreateTunnel(localPort int, remoteHost string, remotePort int) (string, error)
    GetTunnelStatus(tunnelID string) (*TunnelStatus, error)
    CloseTunnel(tunnelID string) error
    GetMetrics() (*Metrics, error)
}
```

#### 6.1.2 RelayClient
```go
type RelayClient interface {
    Connect(host string, port int) error
    Handshake(token, version string) error
    SendMessage(msg interface{}) error
    ReadMessage() (map[string]interface{}, error)
    Close() error
}
```

### 6.2 Структуры данных

#### 6.2.1 TunnelStatus
```go
type TunnelStatus struct {
    ID          string    `json:"id"`
    LocalPort   int       `json:"local_port"`
    RemoteHost  string    `json:"remote_host"`
    RemotePort  int       `json:"remote_port"`
    Status      string    `json:"status"`
    CreatedAt   time.Time `json:"created_at"`
    LastActivity time.Time `json:"last_activity"`
    BytesSent   int64     `json:"bytes_sent"`
    BytesReceived int64   `json:"bytes_received"`
}
```

#### 6.2.2 Metrics
```go
type Metrics struct {
    ConnectionsTotal    int64 `json:"connections_total"`
    ActiveConnections   int64 `json:"active_connections"`
    TunnelsCreated      int64 `json:"tunnels_created"`
    AuthAttempts        int64 `json:"auth_attempts"`
    ErrorsTotal         int64 `json:"errors_total"`
    Uptime              time.Duration `json:"uptime"`
}
```

## 7. Мониторинг и метрики

### 7.1 Prometheus метрики

#### 7.1.1 Основные метрики
```
# Подключения
relay_connections_total{status="accepted"}
relay_connections_total{status="completed"}
relay_connections_total{status="failed"}

# Активные подключения
relay_active_connections

# Аутентификация
relay_auth_attempts_total{status="success",provider="keycloak"}
relay_auth_attempts_total{status="failed",provider="keycloak"}

# Туннели
relay_tunnels_created_total{status="success"}
relay_tunnels_created_total{status="failed"}

# Ошибки
relay_errors_total{type="connection"}
relay_errors_total{type="authentication"}
relay_errors_total{type="tunnel"}
```

#### 7.1.2 Гистограммы
```
# Время подключения
relay_connection_duration_seconds_bucket{le="0.1"}
relay_connection_duration_seconds_bucket{le="0.5"}
relay_connection_duration_seconds_bucket{le="1.0"}

# Размер сообщений
relay_message_size_bytes_bucket{le="1024"}
relay_message_size_bytes_bucket{le="10240"}
relay_message_size_bytes_bucket{le="102400"}
```

### 7.2 Логирование

#### 7.2.1 Уровни логирования
- **DEBUG**: Детальная отладочная информация
- **INFO**: Общая информация о работе
- **WARN**: Предупреждения
- **ERROR**: Ошибки
- **FATAL**: Критические ошибки

#### 7.2.2 Структура логов
```json
{
  "level": "INFO",
  "timestamp": "2025-07-09T03:15:24.123Z",
  "caller": "client/integrated_client.go:123",
  "msg": "Tunnel created successfully",
  "tunnel_id": "tunnel_001",
  "local_port": 3389,
  "remote_host": "192.168.1.100",
  "remote_port": 3389,
  "duration_ms": 45
}
```

## 8. Безопасность

### 8.1 TLS/SSL

#### 8.1.1 Минимальные требования
- **TLS версия**: 1.3 (минимально 1.2)
- **Шифрование**: AES-256-GCM
- **Аутентификация**: Сертификаты X.509
- **Perfect Forward Secrecy**: Обязательно

#### 8.1.2 Сертификаты
```bash
# Генерация самоподписанного сертификата
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

### 8.2 Аутентификация

#### 8.2.1 JWT безопасность
- **Алгоритм**: HS256 (рекомендуется RS256 для production)
- **Время жизни**: 1 час
- **Refresh токены**: Поддерживаются
- **Blacklisting**: Поддерживается

#### 8.2.2 Rate limiting
- **Подключения**: 100 в минуту на IP
- **Аутентификация**: 10 попыток в минуту на IP
- **Создание туннелей**: 50 в минуту на пользователя

## 9. Производительность

### 9.1 Бенчмарки

#### 9.1.1 Подключения
```
BenchmarkHandshake-8         1000           1234567 ns/op
BenchmarkTunnelCreation-8     500           2345678 ns/op
BenchmarkMessageSend-8       2000            567890 ns/op
```

#### 9.1.2 Пропускная способность
- **TCP туннель**: 100 Mbps
- **HTTP/2**: 50 Mbps
- **QUIC**: 75 Mbps

### 9.2 Ограничения

#### 9.2.1 Системные
- **Максимум подключений**: 1000
- **Максимум туннелей**: 100 на клиента
- **Размер сообщения**: 1MB
- **Таймаут**: 30 секунд

#### 9.2.2 Сетевые
- **MTU**: 1500 байт
- **Keep-alive**: 60 секунд
- **Retry**: 3 попытки

## 10. Развертывание

### 10.1 Docker

#### 10.1.1 Dockerfile
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o cloudbridge-client cmd/cloudbridge-client/main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/cloudbridge-client .
COPY config.yaml .
EXPOSE 9090
CMD ["./cloudbridge-client"]
```

#### 10.1.2 Docker Compose
```yaml
version: '3.8'
services:
  cloudbridge-client:
    build: .
    environment:
      - RELAY_JWT_SECRET=${RELAY_JWT_SECRET}
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./logs:/var/log/cloudbridge-client
    ports:
      - "9090:9090"
```

### 10.2 Systemd

#### 10.2.1 Сервис
```ini
[Unit]
Description=CloudBridge Client
After=network.target

[Service]
Type=simple
User=cloudbridge
ExecStart=/usr/local/bin/cloudbridge-client --config /etc/cloudbridge-client/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## 11. Тестирование

### 11.1 Типы тестов

#### 11.1.1 Unit тесты
```bash
go test -v -short ./...
```

#### 11.1.2 Интеграционные тесты
```bash
go test -v -tags=integration ./test/
```

#### 11.1.3 Бенчмарки
```bash
go test -v -bench=. -benchmem ./test/
```

### 11.2 Покрытие кода
```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## 12. Устранение неполадок

### 12.1 Частые проблемы

#### 12.1.1 Ошибки подключения
```
Error: connection refused
Solution: Проверить доступность relay сервера и порты
```

#### 12.1.2 Ошибки аутентификации
```
Error: invalid token
Solution: Проверить JWT токен и его срок действия
```

#### 12.1.3 Ошибки туннелирования
```
Error: tunnel creation failed
Solution: Проверить доступность удаленного хоста и порта
```

### 12.2 Диагностика

#### 12.2.1 Логи
```bash
# Просмотр логов
tail -f /var/log/cloudbridge-client/client.log

# Фильтрация по уровню
grep "ERROR" /var/log/cloudbridge-client/client.log
```

#### 12.2.2 Метрики
```bash
# Prometheus метрики
curl http://localhost:9090/metrics

# Конкретная метрика
curl http://localhost:9090/metrics | grep relay_connections_total
```

#### 12.2.3 Сетевая диагностика
```bash
# Проверка подключения
telnet relay.example.com 8082

# Проверка TLS
openssl s_client -connect relay.example.com:8082
```

### 12.1 Production тестирование и запуск

#### 12.1.1 Пример production-конфига клиента
```yaml
tls:
  enabled: false
  cert_file: ""
  key_file: ""
  ca_file: ""

server:
  host: localhost
  port: 8082
  jwt_token: "<ВАШ_ВАЛИДНЫЙ_JWT_ТОКЕН_ОТ_KEYCLOAK>"
```

#### 12.1.2 Получение токена Keycloak
```bash
curl -k -X POST "https://edge.2gc.ru/realms/cloudbridge/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=relay-client" \
  -d "client_secret=<ВАШ_CLIENT_SECRET>"
```

#### 12.1.3 Запуск клиента
```bash
./cloudbridge-client --config /path/to/config.yaml --token "<ВАШ_ВАЛИДНЫЙ_JWT_ТОКЕН>"
```

#### 12.1.4 Проверка соединения
- Проверить логи клиента (`logs/client.log` или `/var/log/cloudbridge-client/client.log`).
- Проверить метрики на `http://localhost:8081/metrics` (или другой порт, если настроено иначе).
- Проверить логи relay сервера через `docker logs relay`.

#### 12.1.5 Troubleshooting
- Если соединение сбрасывается (EOF) — проверить версию клиента и сервера, формат handshake, circuit breaker.
- Если relay ожидает TLS — включить TLS в конфиге клиента и указать корректные сертификаты.
- Если relay работает без TLS — отключить TLS в конфиге клиента.
- Проверить проброс портов Docker: порт 8082 на хосте должен быть проброшен в контейнер relay.

#### 12.1.6 Пример Docker и портов
- relay: 8082 (host) → 3456 (container)
- keycloak: 8080 (host) → 8080 (container), 443 (host) → 8443 (container)

## 13. Разработка

### 13.1 Среда разработки

#### 13.1.1 Требования
- Go 1.23+
- Docker
- Make
- golangci-lint

#### 13.1.2 Настройка
```bash
# Клонирование
git clone https://github.com/2gc-dev/cloudbridge-client.git
cd cloudbridge-client

# Зависимости
go mod download
go mod tidy

# Линтер
golangci-lint run

# Тесты
make test
```

### 13.2 CI/CD

#### 13.2.1 GitHub Actions
```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - run: make ci-test
      - run: make ci-build
```

## 14. Лицензия

Проект распространяется под лицензией MIT. См. файл [LICENSE](LICENSE) для подробностей.

## 15. Контакты

- **Репозиторий**: https://github.com/2gc-dev/cloudbridge-client
- **Документация**: https://docs.2gc.ru/cloudbridge-client
- **Issues**: https://github.com/2gc-dev/cloudbridge-client/issues 