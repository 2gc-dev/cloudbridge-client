# CloudBridge Client Architecture

## Обзор

CloudBridge Client - это Go-приложение для создания защищенных туннелей через CloudBridge Relay Server. Клиент подключается к relay серверу и создает туннели для безопасного доступа к удаленным ресурсам.

## Архитектура подключения

### Основной домен: edge.2gc.ru

Клиент подключается к следующим сервисам на домене `edge.2gc.ru`:

#### 1. Relay Server (основной) - 3456/tcp
- **Назначение**: Основной сервис для туннелирования
- **Протокол**: TCP
- **Использование**: Создание и управление туннелями
- **Аутентификация**: JWT токен

#### 2. Relay API - 8082/tcp
- **Назначение**: REST API для управления
- **Протокол**: HTTP/HTTPS
- **Использование**: Управление туннелями, мониторинг, конфигурация
- **Аутентификация**: JWT токен

#### 3. Keycloak - 8080/tcp
- **Назначение**: Аутентификация и авторизация
- **Протокол**: HTTP/HTTPS
- **Использование**: Получение JWT токенов
- **Аутентификация**: Client credentials

## Схема подключения

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CloudBridge   │    │   Relay Server  │    │   Remote Host   │
│     Client      │    │   (edge.2gc.ru) │    │   (192.168.x.x) │
│                 │    │                 │    │                 │
│ Local Port      │◄──►│ Port 3456       │◄──►│ Remote Port     │
│ (3389)          │    │ (Tunneling)     │    │ (3389)          │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Keycloak      │    │   Relay API     │    │   Health Check  │
│   (8080/tcp)    │    │   (8082/tcp)    │    │   (9090/tcp)    │
│   Auth          │    │   Management    │    │   Metrics       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Процесс подключения

### 1. Аутентификация
```bash
# Получение JWT токена от Keycloak
curl -X POST "https://edge.2gc.ru/realms/cloudbridge/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=relay-client" \
  -d "client_secret=<YOUR_CLIENT_SECRET>"
```

### 2. Подключение к Relay Server
```yaml
server:
  host: edge.2gc.ru
  port: 3456  # Основной порт для туннелирования
  jwt_token: "<JWT_TOKEN_FROM_KEYCLOAK>"
```

### 3. Создание туннеля
```go
// Клиент подключается к порту 3456
client.Connect("edge.2gc.ru", 3456)

// Выполняет handshake с JWT токеном
client.Handshake(jwtToken)

// Создает туннель
tunnelID, err := client.CreateTunnel(localPort, remoteHost, remotePort)
```

## Конфигурация клиента

### Основная конфигурация
```yaml
# config.yaml
server:
  host: edge.2gc.ru
  port: 3456  # Relay Server (основной)
  jwt_token: "<JWT_TOKEN>"

tls:
  enabled: false  # Для production включить TLS

tunnel:
  local_port: 3389
  reconnect_delay: 5
  max_retries: 3

metrics:
  enabled: true
  port: 9090
  path: "/metrics"
```

### Переменные окружения
```bash
export CLOUDBRIDGE_RELAY_HOST="edge.2gc.ru"
export CLOUDBRIDGE_RELAY_PORT="3456"
export CLOUDBRIDGE_JWT_TOKEN="<JWT_TOKEN>"
```

## Безопасность

### TLS/SSL
- Поддержка TLS 1.3
- Сертификаты клиента и сервера
- Проверка CA сертификатов

### Аутентификация
- JWT токены от Keycloak
- HMAC-SHA256 подпись
- Проверка срока действия токенов

### Сетевая безопасность
- Шифрование трафика
- Защита от MITM атак
- Rate limiting

## Мониторинг

### Метрики Prometheus
```bash
# Метрики доступны на порту 9090
curl http://localhost:9090/metrics

# Основные метрики:
# - relay_connections_total
# - relay_tunnels_active
# - relay_handshake_duration_seconds
# - relay_errors_total
```

### Health Checks
```bash
# Health check endpoint
curl http://localhost:9090/health

# Readiness check
curl http://localhost:9090/ready

# Liveness check
curl http://localhost:9090/live
```

## Устранение неполадок

### Проверка подключения
```bash
# Проверка доступности relay сервера
telnet edge.2gc.ru 3456

# Проверка TLS подключения
openssl s_client -connect edge.2gc.ru:3456

# Проверка Keycloak
curl -k https://edge.2gc.ru/realms/cloudbridge/.well-known/openid_configuration
```

### Логирование
```bash
# Просмотр логов клиента
tail -f /var/log/cloudbridge-client/client.log

# Фильтрация по уровню
grep "ERROR" /var/log/cloudbridge-client/client.log
```

### Частые проблемы

1. **Connection refused на порту 3456**
   - Проверить доступность edge.2gc.ru
   - Проверить настройки файрвола
   - Проверить DNS резолвинг

2. **Invalid JWT token**
   - Проверить срок действия токена
   - Получить новый токен от Keycloak
   - Проверить права доступа

3. **Tunnel creation failed**
   - Проверить доступность удаленного хоста
   - Проверить настройки туннеля
   - Проверить логи relay сервера

## Развертывание

### Docker
```bash
docker run -d \
  --name cloudbridge-client \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -p 3389:3389 \
  -p 9090:9090 \
  cloudbridge-client
```

### Systemd
```bash
sudo systemctl start cloudbridge-client
sudo systemctl status cloudbridge-client
sudo journalctl -u cloudbridge-client -f
```

## Разработка

### Локальная разработка
```bash
# Запуск с тестовой конфигурацией
./cloudbridge-client --config testdata/config-test.yaml

# Запуск с отладкой
./cloudbridge-client --config config.yaml --verbose --log-level debug
```

### Тестирование
```bash
# Unit тесты
go test -v ./...

# Интеграционные тесты
go test -v -tags=integration ./test/

# Бенчмарки
go test -v -bench=. -benchmem ./test/
``` 