# CloudBridge Client - Инструкции по сборке

## Требования

- Go 1.23 или выше
- Git
- Make (опционально)
- Docker (для сборки контейнера)

## Локальная сборка

### 1. Клонирование репозитория

```bash
git clone https://github.com/2gc-dev/cloudbridge-client.git
cd cloudbridge-client
```

### 2. Установка зависимостей

```bash
go mod download
go mod tidy
```

### 3. Сборка для текущей платформы

```bash
# Простая сборка
go build -o cloudbridge-client ./cmd/cloudbridge-client

# Сборка с версией
VERSION=$(git describe --tags --always --dirty)
go build -ldflags "-X main.Version=${VERSION}" -o cloudbridge-client ./cmd/cloudbridge-client
```

### 4. Сборка для всех платформ

```bash
# Используя Makefile
make build-all

# Или вручную
./build.sh
```

## Сборка с помощью Make

### Доступные команды

```bash
# Основные команды
make build          # Сборка основного клиента
make build-mock     # Сборка mock relay сервера
make build-all      # Сборка всех компонентов
make test           # Запуск тестов
make test-coverage  # Тесты с покрытием
make lint           # Линтинг кода
make clean          # Очистка артефактов

# Сборка для разных платформ
make build-linux    # Linux AMD64
make build-windows  # Windows AMD64
make build-darwin   # macOS AMD64
```

### Сборка для конкретной платформы

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o cloudbridge-client-linux-amd64 ./cmd/cloudbridge-client

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o cloudbridge-client-linux-arm64 ./cmd/cloudbridge-client

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o cloudbridge-client-windows-amd64.exe ./cmd/cloudbridge-client

# Windows ARM64
GOOS=windows GOARCH=arm64 go build -o cloudbridge-client-windows-arm64.exe ./cmd/cloudbridge-client

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o cloudbridge-client-darwin-amd64 ./cmd/cloudbridge-client

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o cloudbridge-client-darwin-arm64 ./cmd/cloudbridge-client
```

## Docker сборка

### 1. Сборка образа

```bash
# Сборка локального образа
docker build -t cloudbridge-client .

# Сборка с тегом версии
docker build -t cloudbridge-client:v1.0.0 .
```

### 2. Запуск контейнера

```bash
# Запуск с конфигурацией по умолчанию
docker run -d \
  --name cloudbridge-client \
  -p 3389:3389 \
  -p 9090:9090 \
  cloudbridge-client

# Запуск с кастомной конфигурацией
docker run -d \
  --name cloudbridge-client \
  -v $(pwd)/config.yaml:/etc/cloudbridge-client/config.yaml \
  -v $(pwd)/logs:/var/log/cloudbridge-client \
  -p 3389:3389 \
  -p 9090:9090 \
  cloudbridge-client
```

### 3. Multi-arch сборка

```bash
# Создание multi-arch образа
docker buildx create --use
docker buildx build --platform linux/amd64,linux/arm64 -t cloudbridge-client:latest .
```

## GitHub Actions

### Автоматическая сборка

При каждом push в main ветку или создании pull request автоматически запускается:

1. **Тестирование** на Ubuntu, Windows и macOS
2. **Линтинг** и проверка безопасности
3. **Сборка** для всех поддерживаемых платформ
4. **Создание артефактов** для каждой платформы

### Ручной запуск

```bash
# Запуск через GitHub CLI
gh workflow run build.yml

# Или через веб-интерфейс GitHub
# Actions -> Build and Test -> Run workflow
```

### Получение артефактов

После успешной сборки артефакты доступны в разделе Actions:

1. Перейдите в Actions -> Build and Test
2. Выберите успешный workflow
3. Скачайте артефакты для нужной платформы

## Конфигурация сборки

### Переменные окружения

```bash
# Версия Go
export GO_VERSION=1.23

# Отключение CGO для статической сборки
export CGO_ENABLED=0

# Флаги сборки
export LDFLAGS="-w -s -X main.Version=${VERSION}"
```

### Флаги сборки

```bash
# Минимальный размер бинарного файла
go build -ldflags "-w -s" -o cloudbridge-client ./cmd/cloudbridge-client

# С информацией о версии
go build -ldflags "-X main.Version=v1.0.0 -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" -o cloudbridge-client ./cmd/cloudbridge-client

# Статическая сборка
CGO_ENABLED=0 go build -a -installsuffix cgo -o cloudbridge-client ./cmd/cloudbridge-client
```

## Проверка сборки

### 1. Проверка бинарного файла

```bash
# Информация о файле
file cloudbridge-client

# Размер файла
ls -lh cloudbridge-client

# Зависимости (для Linux)
ldd cloudbridge-client

# Проверка версии
./cloudbridge-client --version
```

### 2. Тестирование

```bash
# Unit тесты
go test -v ./...

# Интеграционные тесты
go test -v -tags=integration ./test/

# Бенчмарки
go test -v -bench=. -benchmem ./test/
```

### 3. Линтинг

```bash
# golangci-lint
golangci-lint run

# go vet
go vet ./...

# Проверка форматирования
go fmt ./...
```

## Устранение неполадок

### Ошибки сборки

1. **"go: module lookup disabled by GOPROXY=off"**
   ```bash
   export GOPROXY=https://proxy.golang.org,direct
   ```

2. **"CGO_ENABLED=1 but no C compiler"**
   ```bash
   export CGO_ENABLED=0
   ```

3. **"undefined: main.Version"**
   ```bash
   # Убедитесь, что переменная Version определена в main.go
   go build -ldflags "-X main.Version=v1.0.0" -o cloudbridge-client ./cmd/cloudbridge-client
   ```

### Проблемы с Docker

1. **"failed to compute cache key"**
   ```bash
   docker build --no-cache -t cloudbridge-client .
   ```

2. **"permission denied"**
   ```bash
   # Проверьте права доступа к файлам
   chmod +x cloudbridge-client
   ```

### Проблемы с GitHub Actions

1. **Timeout при сборке**
   - Увеличьте timeout в workflow
   - Оптимизируйте Dockerfile

2. **Ошибки загрузки артефактов**
   - Проверьте размер артефактов
   - Убедитесь в корректности путей

## Оптимизация

### Размер бинарного файла

```bash
# Сжатие символов
go build -ldflags "-w -s" -o cloudbridge-client ./cmd/cloudbridge-client

# UPX сжатие (требует установки UPX)
upx --best cloudbridge-client
```

### Время сборки

```bash
# Кэширование модулей
go mod download

# Параллельная сборка
go build -p 4 -o cloudbridge-client ./cmd/cloudbridge-client
```

### Docker оптимизация

```bash
# Multi-stage build
# Используйте .dockerignore
# Кэширование слоев
docker build --build-arg BUILDKIT_INLINE_CACHE=1 -t cloudbridge-client .
``` 