# CloudBridge Client

[![Go Report Card](https://goreportcard.com/badge/github.com/2gc-dev/cloudbridge-client)](https://goreportcard.com/report/github.com/2gc-dev/cloudbridge-client)
[![Go Version](https://img.shields.io/github/go-mod/go-version/2gc-dev/cloudbridge-client)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)](https://github.com/2gc-dev/cloudbridge-client/actions)

Go-клиент для подключения к CloudBridge Relay серверу с поддержкой множественных протоколов, аутентификации и создания туннелей.

## 🚀 Возможности

- **Множественные протоколы**: QUIC, HTTP/2, HTTP/1.1
- **Аутентификация**: JWT токены с поддержкой Keycloak и Django
- **Туннелирование**: Создание и управление TCP туннелями
- **Отказоустойчивость**: Circuit breaker, rate limiting, переподключение
- **Мониторинг**: Prometheus метрики и структурированное логирование
- **Тестирование**: Unit, интеграционные тесты и бенчмарки
- **Документация**: Полная техническая документация и руководства

## 📋 Требования

- Go 1.23+
- Docker (опционально)
- Make
- golangci-lint (для разработки)

## 🛠 Установка

### Из исходного кода

```bash
# Клонирование репозитория
git clone https://github.com/2gc-dev/cloudbridge-client.git
cd cloudbridge-client

# Зависимости
go mod download
go mod tidy

# Сборка
make build-all

# Проверка версии
./cloudbridge-client --version
```

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
```

## ⚙️ Конфигурация

Создайте файл `config.yaml`:

```yaml
server:
  host: "relay.example.com"
  port: 8082
  jwt_token: "your-jwt-token-here"

tls:
  enabled: false  # Включите для production

tunnel:
  local_port: 3389
  reconnect_delay: 5
  max_retries: 3

logging:
  level: "info"
  file: "/var/log/cloudbridge-client/client.log"

metrics:
  enabled: true
  port: 9090
  path: "/metrics"
```

## 🚀 Использование

### Базовый запуск

```bash
# Запуск клиента
./cloudbridge-client --config config.yaml

# С параметрами командной строки
./cloudbridge-client \
  --config config.yaml \
  --token "your-jwt-token" \
  --local-port 3389 \
  --remote-host "192.168.1.100" \
  --remote-port 3389
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

## 🧪 Тестирование

### Запуск тестов

```bash
# Все тесты
make test

# Только unit тесты
make test-unit

# Интеграционные тесты
make test-integration

# Тесты с покрытием
make test-coverage

# Бенчмарки
make test-benchmark
```

### Mock Relay сервер

```bash
# Сборка mock relay
make build-mock

# Запуск mock relay на порту 8084
make mock-relay

# Тестирование с mock relay
go test -v ./test/ -tags=integration
```

### Примеры тестов

```bash
# Тест handshake протокола
go test -v -run TestHandshakeProtocol

# Тест создания туннелей
go test -v -run TestTunnelCreation

# Тест обработки ошибок
go test -v -run TestErrorHandling

# Бенчмарк подключений
go test -v -bench=BenchmarkHandshake -benchmem
```

## 📊 Мониторинг

### Prometheus метрики

```bash
# Просмотр метрик
curl http://localhost:9090/metrics

# Основные метрики
curl http://localhost:9090/metrics | grep relay_connections_total
curl http://localhost:9090/metrics | grep relay_active_connections
curl http://localhost:9090/metrics | grep relay_tunnels_created_total
```

### Логирование

```bash
# Просмотр логов
tail -f /var/log/cloudbridge-client/client.log

# Фильтрация по уровню
grep "ERROR" /var/log/cloudbridge-client/client.log
grep "auth" /var/log/cloudbridge-client/client.log
```

## 📚 Документация

### Основная документация

- [Техническая спецификация](docs/TECHNICAL_SPECIFICATION.md) - Подробное описание архитектуры и протоколов
- [Руководство пользователя](docs/USER_GUIDE.md) - Пошаговые инструкции по использованию
- [API Reference](docs/API_REFERENCE.md) - Документация по API интерфейсам

### Примеры конфигурации

- [RDP туннель](config/config-rdp.yaml) - Настройка для Remote Desktop
- [SSH туннель](config/config-ssh.yaml) - Настройка для SSH
- [Веб-сервер](config/config-web.yaml) - Настройка для веб-приложений
- [База данных](config/config-db.yaml) - Настройка для баз данных

### Развертывание

- [Docker](docs/DEPLOYMENT.md#docker) - Развертывание в контейнерах
- [Systemd](docs/DEPLOYMENT.md#systemd) - Развертывание как системный сервис
- [Kubernetes](docs/DEPLOYMENT.md#kubernetes) - Развертывание в Kubernetes

## 🔧 Разработка

### Настройка среды разработки

```bash
# Клонирование
git clone https://github.com/2gc-dev/cloudbridge-client.git
cd cloudbridge-client

# Зависимости
go mod download
go mod tidy

# Линтер
make lint

# Форматирование кода
make format

# Тесты
make test
```

### Структура проекта

```
cloudbridge-client/
├── cmd/                    # Исполняемые файлы
│   └── cloudbridge-client/
├── pkg/                    # Основные пакеты
│   ├── auth/              # Аутентификация
│   ├── client/            # Основной клиент
│   ├── config/            # Конфигурация
│   ├── errors/            # Обработка ошибок
│   ├── protocol/          # Протоколы связи
│   ├── relay/             # Relay клиент
│   ├── tunnel/            # Управление туннелями
│   └── types/             # Типы данных
├── test/                  # Тесты
│   ├── integration_test.go
│   └── mock_relay/        # Mock relay сервер
├── config/                # Конфигурационные файлы
├── docs/                  # Документация
├── deploy/                # Файлы развертывания
└── scripts/               # Скрипты
```

### Команды Make

```bash
# Сборка
make build          # Основной клиент
make build-mock     # Mock relay сервер
make build-all      # Все компоненты

# Тестирование
make test           # Все тесты
make test-unit      # Unit тесты
make test-integration # Интеграционные тесты
make test-coverage  # Тесты с покрытием
make test-benchmark # Бенчмарки

# Качество кода
make lint           # Линтер
make lint-fix       # Автоисправление
make security-check # Проверка безопасности
make format         # Форматирование

# Документация
make docs           # Запуск godoc сервера
make api-docs       # Генерация API документации

# Разработка
make clean          # Очистка
make deps           # Зависимости
make mock-relay     # Запуск mock relay
make run-client     # Запуск клиента

# Docker
make docker-build   # Сборка образа
make docker-test    # Тесты в Docker

# CI/CD
make ci-test        # Полный набор тестов CI
make ci-build       # Сборка для CI
```

## 🐛 Устранение неполадок

### Частые проблемы

#### Ошибка подключения
```bash
# Проверка доступности сервера
telnet relay.example.com 8082

# Проверка DNS
nslookup relay.example.com
```

#### Ошибка аутентификации
```bash
# Проверка токена
echo "your-jwt-token" | cut -d'.' -f2 | base64 -d | jq .

# Получение нового токена
curl -X POST https://edge.2gc.ru/realms/cloudbridge/protocol/openid-connect/token \
  -d "grant_type=client_credentials" \
  -d "client_id=relay-client" \
  -d "client_secret=your-secret"
```

#### Ошибка создания туннеля
```bash
# Проверка удаленного хоста
ping 192.168.1.100

# Проверка порта
telnet 192.168.1.100 3389
```

### Диагностика

```bash
# Отладочный режим
./cloudbridge-client --config config.yaml --verbose --log-level debug

# Проверка конфигурации
./cloudbridge-client --config config.yaml --dry-run

# Сбор информации для отладки
make debug-info
```

## 🤝 Вклад в проект

### Отчеты об ошибках

1. Проверьте [существующие issues](https://github.com/2gc-dev/cloudbridge-client/issues)
2. Создайте новый issue с подробным описанием проблемы
3. Приложите логи и конфигурацию

### Pull Requests

1. Форкните репозиторий
2. Создайте ветку для новой функции
3. Внесите изменения
4. Добавьте тесты
5. Обновите документацию
6. Создайте Pull Request

### Стандарты кода

- Следуйте [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Используйте `gofmt` для форматирования
- Добавляйте комментарии к экспортируемым функциям
- Покрывайте код тестами

## 📄 Лицензия

Проект распространяется под лицензией MIT. См. файл [LICENSE](LICENSE) для подробностей.

## 📞 Контакты

- **Репозиторий**: https://github.com/2gc-dev/cloudbridge-client
- **Документация**: https://docs.2gc.ru/cloudbridge-client
- **Issues**: https://github.com/2gc-dev/cloudbridge-client/issues
- **Discussions**: https://github.com/2gc-dev/cloudbridge-client/discussions

## 🙏 Благодарности

- [Go Team](https://golang.org/) за отличный язык программирования
- [Prometheus](https://prometheus.io/) за систему мониторинга
- [Keycloak](https://www.keycloak.org/) за систему аутентификации
- Всем контрибьюторам проекта

---

**CloudBridge Client** - Надежное и безопасное туннелирование для ваших приложений.
