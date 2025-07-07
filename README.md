# CloudBridge Relay Client

**CloudBridge Relay Client** — это безопасный кроссплатформенный клиент на Go для работы с сервисом CloudBridge Relay. Клиент поддерживает протокол с TLS 1.3, JWT-аутентификацию, обработку ошибок, управление туннелями и системным сервисом.

## Возможности
- Поддержка TLS 1.3 и безопасных шифров
- JWT-аутентификация (HMAC и RSA)
- Интеграция с Keycloak (OpenID Connect)
- Кроссплатформенность: Windows, Linux, macOS (x86_64, ARM64)
- Ограничение скорости с экспоненциальным backoff
- Мониторинг соединения (heartbeat)
- Управление туннелями
- Гибкая конфигурация через YAML и переменные окружения
- Установка как системный сервис
- Метрики Prometheus и health-check

## Установка

### Через Go
```bash
go install github.com/2gc-dev/cloudbridge-client/cmd/cloudbridge-client@latest
```

### Сборка из исходников
```bash
git clone https://github.com/2gc-dev/cloudbridge-client.git
cd cloudbridge-client
go build -o cloudbridge-client ./cmd/cloudbridge-client
```

## Быстрый старт

### Простой запуск
```bash
cloudbridge-client --token "ваш-jwt-токен"
```

### С использованием конфигурационного файла
```bash
cloudbridge-client --config config.yaml --token "ваш-jwt-токен"
```

### Кастомный туннель
```bash
cloudbridge-client \
  --token "ваш-jwt-токен" \
  --tunnel-id "мой-туннель" \
  --local-port 3389 \
  --remote-host "192.168.1.100" \
  --remote-port 3389
```

## Конфигурация

Клиент поддерживает настройку через YAML-файл и переменные окружения.

### Пример config.yaml
```yaml
relay:
  host: "edge.2gc.ru"
  port: 8080
  timeout: "30s"
  tls:
    enabled: true
    min_version: "1.3"
    verify_cert: true
    ca_cert: "/path/to/ca.pem"
    client_cert: "/path/to/client.crt"
    client_key: "/path/to/client.key"

auth:
  type: "jwt"
  secret: "jwt-секрет"
  keycloak:
    enabled: false
    server_url: "https://keycloak.example.com"
    realm: "cloudbridge"
    client_id: "relay-client"

rate_limiting:
  enabled: true
  max_retries: 3
  backoff_multiplier: 2.0
  max_backoff: "30s"
  window_size: "1m"
  max_requests: 100

logging:
  level: "info"
  format: "json"
  output: "stdout"
```

### Переменные окружения
Любую опцию можно задать через переменные с префиксом `CLOUDBRIDGE_`:
```bash
export CLOUDBRIDGE_RELAY_HOST="edge.2gc.ru"
export CLOUDBRIDGE_RELAY_PORT="8080"
export CLOUDBRIDGE_AUTH_SECRET="jwt-секрет"
```

### Ключевые параметры командной строки
- `--config, -c`: путь к конфигу
- `--token, -t`: JWT-токен (обязателен)
- `--tunnel-id, -i`: ID туннеля (по умолчанию: tunnel_001)
- `--local-port, -l`: локальный порт (по умолчанию: 3389)
- `--remote-host, -r`: удалённый хост (по умолчанию: 192.168.1.100)
- `--remote-port, -p`: удалённый порт (по умолчанию: 3389)
- `--verbose, -v`: подробное логирование

## Управление сервисом

### Установка как системный сервис
```bash
# Linux/macOS
sudo cloudbridge-client service install <jwt-token>

# Windows (от имени администратора)
cloudbridge-client.exe service install <jwt-token>
```

### Команды сервиса
```bash
cloudbridge-client service status      # Проверить статус
cloudbridge-client service start       # Запустить сервис
cloudbridge-client service stop        # Остановить сервис
cloudbridge-client service restart     # Перезапустить сервис
cloudbridge-client service uninstall   # Удалить сервис
```

## Безопасность

### TLS 1.3
- Минимальная версия TLS 1.3
- Только безопасные шифры:
  - TLS_AES_256_GCM_SHA384
  - TLS_CHACHA20_POLY1305_SHA256
  - TLS_AES_128_GCM_SHA256
- Проверка сертификатов
- Поддержка SNI

### JWT-аутентификация
- Поддержка HMAC-SHA256 и RSA
- Проверка срока действия токена
- Извлечение subject для rate limiting

### Keycloak
- OpenID Connect
- Автоматическая загрузка JWKS
- Проверка ролей

## Ограничение скорости
- Ограничение по пользователю (по subject JWT)
- Экспоненциальный backoff
- Настраиваемое максимальное число попыток
- Sliding window

## Мониторинг

### Метрики Prometheus
Доступны по адресу: http://localhost:9090/metrics
- relay_connections_total — всего соединений
- relay_connection_duration_seconds — длительность соединений
- relay_errors_total — количество ошибок по типам
- relay_active_tunnels — число активных туннелей
- relay_heartbeat_latency_seconds — задержка heartbeat
- relay_missed_heartbeats_total — пропущенные heartbeat

### Health-check
http://localhost:9090/health
```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime": "2h30m15s",
  "connections_total": 42,
  "active_tunnels": 3,
  "errors_total": 0,
  "missed_heartbeats": 0
}
```

## Поддерживаемые платформы
- Windows: x86_64, ARM64
- Linux: x86_64, ARM64
- macOS: x86_64, ARM64

## Разработка

### Сборка под разные платформы
```bash
GOOS=windows GOARCH=amd64 go build -o cloudbridge-client.exe ./cmd/cloudbridge-client
GOOS=linux GOARCH=amd64 go build -o cloudbridge-client ./cmd/cloudbridge-client
GOOS=darwin GOARCH=amd64 go build -o cloudbridge-client ./cmd/cloudbridge-client
```

### Запуск тестов
```bash
go test ./...
# или по пакетам
go test ./pkg/auth
go test ./pkg/rate_limiting
go test ./pkg/relay
```

## Структура кода
```
pkg/
├── auth/          # Аутентификация
├── config/        # Конфигурация
├── errors/        # Обработка ошибок
├── heartbeat/     # Мониторинг соединения
├── rate_limiting/ # Ограничение скорости
├── relay/         # Основной клиент
├── service/       # Управление сервисом
└── tunnel/        # Управление туннелями

cmd/
└── cloudbridge-client/  # Главный исполняемый файл

docs/             # Документация
├── API.md        # Описание протокола
├── ARCHITECTURE.md # Архитектура
├── DEPLOYMENT.md # Развёртывание
├── PERFORMANCE.md # Производительность
├── SECURITY.md   # Безопасность
├── TESTING.md    # Тестирование
└── TROUBLESHOOTING.md # Диагностика
```

## Вклад
1. Сделайте fork репозитория
2. Создайте ветку (`git checkout -b feature/ваша-фича`)
3. Зафиксируйте изменения (`git commit -m 'Добавить новую фичу'`)
4. Отправьте ветку (`git push origin feature/ваша-фича`)
5. Откройте Pull Request

## Лицензия
Проект распространяется под лицензией MIT. Подробнее — в файле LICENSE.

## Поддержка
- Создайте issue на GitHub
- Изучите документацию в папке `docs/`
- Посмотрите примеры конфигурации

## История изменений
### v1.0.0
- Первый релиз
- Поддержка TLS 1.3
- JWT-аутентификация
- Кроссплатформенность
- Обработка ошибок
- Ограничение скорости и retry
- Heartbeat
- Управление туннелями
- Управление сервисом
- Метрики Prometheus
- Health-check
