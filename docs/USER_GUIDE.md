# CloudBridge Client - Руководство пользователя

## Содержание

1. [Быстрый старт](#быстрый-старт)
2. [Установка](#установка)
3. [Конфигурация](#конфигурация)
4. [Использование](#использование)
5. [Примеры](#примеры)
6. [Мониторинг](#мониторинг)
7. [Устранение неполадок](#устранение-неполадок)
8. [FAQ](#faq)

## Быстрый старт

### 1. Скачивание и установка

```bash
# Клонирование репозитория
git clone https://github.com/2gc-dev/cloudbridge-client.git
cd cloudbridge-client

# Сборка
make build

# Проверка версии
./cloudbridge-client --version
```

### 2. Базовая конфигурация

Создайте файл `config.yaml`:

```yaml
server:
  host: "relay.example.com"
  port: 8082
  jwt_token: "your-jwt-token-here"

tls:
  enabled: false  # Включите для production

logging:
  level: "info"
  file: "/var/log/cloudbridge-client/client.log"
```

### 3. Первый запуск

```bash
# Запуск клиента
./cloudbridge-client --config config.yaml

# Или с параметрами командной строки
./cloudbridge-client \
  --config config.yaml \
  --token "your-jwt-token" \
  --local-port 3389 \
  --remote-host "192.168.1.100" \
  --remote-port 3389
```

## Установка

### Docker

```bash
# Сборка образа
docker build -t cloudbridge-client .

# Запуск контейнера
docker run -d \
  --name cloudbridge-client \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/logs:/var/log/cloudbridge-client \
  -p 9090:9090 \
  cloudbridge-client
```

### Systemd

```bash
# Копирование файлов
sudo cp cloudbridge-client /usr/local/bin/
sudo mkdir -p /etc/cloudbridge-client
sudo cp config.yaml /etc/cloudbridge-client/
sudo cp deploy/cloudbridge-client.service /etc/systemd/system/

# Включение и запуск сервиса
sudo systemctl daemon-reload
sudo systemctl enable cloudbridge-client
sudo systemctl start cloudbridge-client

# Проверка статуса
sudo systemctl status cloudbridge-client
```

### Из исходного кода

```bash
# Требования
# - Go 1.23+
# - Make
# - git

# Клонирование
git clone https://github.com/2gc-dev/cloudbridge-client.git
cd cloudbridge-client

# Зависимости
go mod download
go mod tidy

# Сборка
make build-all

# Тесты
make test
```

## Конфигурация

### Основные параметры

#### server
```yaml
server:
  host: "relay.example.com"      # Хост relay сервера
  port: 8082                     # Порт relay сервера
  jwt_token: "your-jwt-token"    # JWT токен для аутентификации
  timeout: 30s                   # Таймаут подключения
```

#### tls
```yaml
tls:
  enabled: true                  # Включить TLS
  cert_file: "/path/to/cert.pem" # Сертификат клиента
  key_file: "/path/to/key.pem"   # Приватный ключ
  ca_file: "/path/to/ca.pem"     # CA сертификат
  min_version: "1.3"             # Минимальная версия TLS
```

#### tunnel
```yaml
tunnel:
  local_port: 3389               # Локальный порт
  reconnect_delay: 5             # Задержка переподключения (сек)
  max_retries: 3                 # Максимум попыток
  heartbeat_interval: 30         # Интервал heartbeat (сек)
```

#### logging
```yaml
logging:
  level: "info"                  # Уровень логирования
  format: "json"                 # Формат логов
  file: "/var/log/cloudbridge-client/client.log"
  max_size: 100                 # Максимальный размер файла (MB)
  max_backups: 3                # Количество резервных копий
  max_age: 28                   # Максимальный возраст (дни)
  compress: true                # Сжатие старых логов
```

#### metrics
```yaml
metrics:
  enabled: true                  # Включить метрики
  port: 9090                    # Порт для метрик
  path: "/metrics"              # Путь к метрикам
```

### Переменные окружения

```bash
# Основные
export RELAY_JWT_SECRET="your-secret"
export RELAY_SERVER_HOST="relay.example.com"
export RELAY_SERVER_PORT="8082"

# Keycloak (если используется)
export KEYCLOAK_SERVER_URL="https://edge.2gc.ru"
export KEYCLOAK_REALM="cloudbridge"
export KEYCLOAK_CLIENT_ID="relay-client"
export KEYCLOAK_CLIENT_SECRET="your-secret"

# Django (если используется)
export DJANGO_API_ENDPOINT="http://localhost:8000/api/"
export DJANGO_JWT_SECRET="your-secret"
```

### Параметры командной строки

```bash
./cloudbridge-client --help

# Основные параметры
-c, --config string        # Путь к конфигурационному файлу
-t, --token string         # JWT токен для аутентификации
-l, --local-port int       # Локальный порт (по умолчанию 3389)
-r, --remote-host string   # Удаленный хост (по умолчанию 192.168.1.100)
-p, --remote-port int      # Удаленный порт (по умолчанию 3389)
-i, --tunnel-id string     # ID туннеля (по умолчанию tunnel_001)
-v, --verbose              # Подробный вывод
```

## Использование

### Основные команды

#### Подключение к relay серверу
```bash
# Простое подключение
./cloudbridge-client --config config.yaml

# С параметрами
./cloudbridge-client \
  --config config.yaml \
  --token "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9..." \
  --local-port 3389 \
  --remote-host "192.168.1.100" \
  --remote-port 3389
```

#### Создание туннеля
```bash
# TCP туннель для RDP
./cloudbridge-client \
  --config config.yaml \
  --local-port 3389 \
  --remote-host "192.168.1.100" \
  --remote-port 3389

# SSH туннель
./cloudbridge-client \
  --config config.yaml \
  --local-port 2222 \
  --remote-host "192.168.1.101" \
  --remote-port 22
```

#### Мониторинг
```bash
# Просмотр метрик
curl http://localhost:9090/metrics

# Проверка статуса
curl http://localhost:9090/health
```

### Программное использование

```go
package main

import (
    "context"
    "log"
    
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
    
    // Подключение
    ctx := context.Background()
    err = client.Connect(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Создание туннеля
    tunnelID, err := client.CreateTunnel(3389, "192.168.1.100", 3389)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Туннель создан: %s", tunnelID)
    
    // Ожидание
    select {}
}
```

## Примеры

### Пример 1: RDP туннель

```yaml
# config-rdp.yaml
server:
  host: "relay.example.com"
  port: 8082
  jwt_token: "your-jwt-token"

tunnel:
  local_port: 3389
  reconnect_delay: 5
  max_retries: 3

logging:
  level: "info"
  file: "/var/log/cloudbridge-client/rdp.log"
```

```bash
# Запуск
./cloudbridge-client --config config-rdp.yaml

# Подключение через RDP клиент
# localhost:3389 -> 192.168.1.100:3389
```

### Пример 2: SSH туннель

```yaml
# config-ssh.yaml
server:
  host: "relay.example.com"
  port: 8082
  jwt_token: "your-jwt-token"

tunnel:
  local_port: 2222
  reconnect_delay: 3
  max_retries: 5

logging:
  level: "debug"
  file: "/var/log/cloudbridge-client/ssh.log"
```

```bash
# Запуск
./cloudbridge-client --config config-ssh.yaml

# SSH подключение
ssh -p 2222 user@localhost
```

### Пример 3: Веб-сервер

```yaml
# config-web.yaml
server:
  host: "relay.example.com"
  port: 8082
  jwt_token: "your-jwt-token"

tunnel:
  local_port: 8080
  reconnect_delay: 2
  max_retries: 10

logging:
  level: "info"
  file: "/var/log/cloudbridge-client/web.log"
```

```bash
# Запуск
./cloudbridge-client --config config-web.yaml

# Доступ к веб-серверу
curl http://localhost:8080
```

### Пример 4: База данных

```yaml
# config-db.yaml
server:
  host: "relay.example.com"
  port: 8082
  jwt_token: "your-jwt-token"

tunnel:
  local_port: 5432
  reconnect_delay: 1
  max_retries: 20

logging:
  level: "warn"
  file: "/var/log/cloudbridge-client/db.log"
```

```bash
# Запуск
./cloudbridge-client --config config-db.yaml

# Подключение к PostgreSQL
psql -h localhost -p 5432 -U username -d database
```

## Мониторинг

### Метрики Prometheus

#### Основные метрики
```bash
# Подключения
curl http://localhost:9090/metrics | grep relay_connections_total

# Активные подключения
curl http://localhost:9090/metrics | grep relay_active_connections

# Аутентификация
curl http://localhost:9090/metrics | grep relay_auth_attempts_total

# Туннели
curl http://localhost:9090/metrics | grep relay_tunnels_created_total
```

#### Grafana дашборд
```json
{
  "dashboard": {
    "title": "CloudBridge Client",
    "panels": [
      {
        "title": "Подключения",
        "type": "graph",
        "targets": [
          {
            "expr": "relay_connections_total",
            "legendFormat": "{{status}}"
          }
        ]
      },
      {
        "title": "Активные туннели",
        "type": "stat",
        "targets": [
          {
            "expr": "relay_active_connections"
          }
        ]
      }
    ]
  }
}
```

### Логирование

#### Просмотр логов
```bash
# Последние записи
tail -f /var/log/cloudbridge-client/client.log

# Ошибки
grep "ERROR" /var/log/cloudbridge-client/client.log

# Аутентификация
grep "auth" /var/log/cloudbridge-client/client.log

# Туннели
grep "tunnel" /var/log/cloudbridge-client/client.log
```

#### Ротация логов
```bash
# Ручная ротация
logrotate -f /etc/logrotate.d/cloudbridge-client

# Проверка конфигурации
logrotate -d /etc/logrotate.d/cloudbridge-client
```

### Системный мониторинг

#### Systemd
```bash
# Статус сервиса
systemctl status cloudbridge-client

# Логи сервиса
journalctl -u cloudbridge-client -f

# Перезапуск
systemctl restart cloudbridge-client
```

#### Docker
```bash
# Статус контейнера
docker ps | grep cloudbridge-client

# Логи контейнера
docker logs -f cloudbridge-client

# Статистика
docker stats cloudbridge-client
```

## Устранение неполадок

### Частые проблемы

#### 1. Ошибка подключения
```
Error: connection refused
```

**Решение:**
```bash
# Проверка доступности сервера
telnet relay.example.com 8082

# Проверка DNS
nslookup relay.example.com

# Проверка файрвола
sudo ufw status
```

#### 2. Ошибка аутентификации
```
Error: invalid token
```

**Решение:**
```bash
# Проверка токена
echo "your-jwt-token" | cut -d'.' -f2 | base64 -d | jq .

# Проверка срока действия
jwt decode your-jwt-token

# Получение нового токена
curl -X POST https://edge.2gc.ru/realms/cloudbridge/protocol/openid-connect/token \
  -d "grant_type=client_credentials" \
  -d "client_id=relay-client" \
  -d "client_secret=your-secret"
```

#### 3. Ошибка создания туннеля
```
Error: tunnel creation failed
```

**Решение:**
```bash
# Проверка удаленного хоста
ping 192.168.1.100

# Проверка порта
telnet 192.168.1.100 3389

# Проверка локального порта
netstat -tlnp | grep 3389
```

#### 4. Высокое потребление ресурсов
```
CPU: 100%
Memory: 1GB+
```

**Решение:**
```bash
# Проверка процессов
ps aux | grep cloudbridge-client

# Проверка соединений
netstat -an | grep 8082

# Ограничение ресурсов (Docker)
docker run --cpus=1 --memory=512m cloudbridge-client
```

### Диагностика

#### Сбор информации
```bash
# Системная информация
uname -a
cat /etc/os-release

# Сетевая информация
ip addr show
ip route show

# Процессы
ps aux | grep cloudbridge-client

# Порты
netstat -tlnp | grep cloudbridge

# Логи
tail -100 /var/log/cloudbridge-client/client.log
```

#### Отладочный режим
```bash
# Включение debug логирования
./cloudbridge-client \
  --config config.yaml \
  --verbose \
  --log-level debug

# Проверка конфигурации
./cloudbridge-client --config config.yaml --dry-run
```

#### Тестирование подключения
```bash
# Тест TCP подключения
nc -zv relay.example.com 8082

# Тест TLS подключения
openssl s_client -connect relay.example.com:8082

# Тест HTTP (если поддерживается)
curl -v http://relay.example.com:8082/health
```

## FAQ

### Q: Как изменить порт туннеля?
**A:** Используйте параметр `--local-port`:
```bash
./cloudbridge-client --local-port 8080 --remote-host 192.168.1.100 --remote-port 80
```

### Q: Как использовать несколько туннелей?
**A:** Запустите несколько экземпляров клиента с разными портами:
```bash
# Туннель 1: RDP
./cloudbridge-client --local-port 3389 --remote-host 192.168.1.100 --remote-port 3389 &

# Туннель 2: SSH
./cloudbridge-client --local-port 2222 --remote-host 192.168.1.101 --remote-port 22 &
```

### Q: Как настроить автопереподключение?
**A:** Настройте параметры в конфигурации:
```yaml
tunnel:
  reconnect_delay: 5    # Задержка между попытками (сек)
  max_retries: 10       # Максимум попыток
  heartbeat_interval: 30 # Интервал проверки соединения
```

### Q: Как включить TLS?
**A:** Настройте TLS в конфигурации:
```yaml
tls:
  enabled: true
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
  ca_file: "/path/to/ca.pem"
```

### Q: Как получить JWT токен?
**A:** Зависит от провайдера аутентификации:

**Keycloak:**
```bash
curl -X POST https://edge.2gc.ru/realms/cloudbridge/protocol/openid-connect/token \
  -d "grant_type=client_credentials" \
  -d "client_id=relay-client" \
  -d "client_secret=your-secret"
```

**Django:**
```bash
curl -X POST http://localhost:8000/api/token/ \
  -H "Content-Type: application/json" \
  -d '{"username": "user", "password": "pass"}'
```

### Q: Как мониторить производительность?
**A:** Используйте Prometheus метрики:
```bash
# Основные метрики
curl http://localhost:9090/metrics | grep relay_

# Создание дашборда в Grafana
# Импортируйте готовый дашборд или создайте свой
```

### Q: Как обновить клиент?
**A:** Зависит от способа установки:

**Из исходного кода:**
```bash
git pull
make build
sudo systemctl restart cloudbridge-client
```

**Docker:**
```bash
docker pull cloudbridge-client:latest
docker stop cloudbridge-client
docker rm cloudbridge-client
docker run ... # с новыми параметрами
```

### Q: Как настроить логирование?
**A:** Настройте в конфигурации:
```yaml
logging:
  level: "info"        # debug, info, warn, error
  format: "json"       # json, text
  file: "/var/log/cloudbridge-client/client.log"
  max_size: 100        # MB
  max_backups: 3
  max_age: 28          # дни
  compress: true
```

### Q: Как отладить проблемы с сетью?
**A:** Используйте сетевые инструменты:
```bash
# Проверка маршрутизации
traceroute relay.example.com

# Проверка DNS
dig relay.example.com

# Проверка портов
nmap -p 8082 relay.example.com

# Мониторинг трафика
tcpdump -i any port 8082
```

### Q: Как настроить автозапуск?
**A:** Используйте systemd:
```bash
sudo systemctl enable cloudbridge-client
sudo systemctl start cloudbridge-client
```

Или добавьте в crontab:
```bash
@reboot /usr/local/bin/cloudbridge-client --config /etc/cloudbridge-client/config.yaml
``` 