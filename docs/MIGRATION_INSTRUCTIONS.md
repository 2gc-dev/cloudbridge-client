# 🚀 Инструкции по миграции CloudBridge Client на v2.0

## 📋 Обзор

Данный документ содержит пошаговые инструкции по миграции CloudBridge Client с версии 1.0.0 на версию 2.0 для полной совместимости с Relay Server v2.0.

## 🎯 Что изменилось в v2.0

### ✅ Новые возможности
- **Multi-tenancy поддержка** - изоляция ресурсов по tenant
- **Enhanced метрики** - 200+ метрик с Prometheus интеграцией
- **Health Check система** - автоматический мониторинг состояния
- **Протокол v2.0** - обновленный handshake и auth
- **QUIC improvements** - stream multiplexing и connection pooling
- **Backward compatibility** - поддержка v1.0.0 для плавной миграции

### 🔄 Изменения в протоколе
- Версия протокола: `1.0.0` → `2.0`
- Новые features: `multi_tenant`, `proxy`, `metrics`
- Auth message: добавлен `tenant_id`
- Hello message: обновлены features

## 📦 Установка и настройка

### 1. Обновление зависимостей

Добавьте новые зависимости в `go.mod`:

```go
require (
    github.com/prometheus/client_golang v1.17.0
    go.uber.org/zap v1.26.0
    // ... existing dependencies
)
```

### 2. Обновление конфигурации

#### Старая конфигурация (v1.0.0):
```yaml
tls:
  enabled: true
  cert_file: "/etc/certs/client.crt"
  key_file: "/etc/certs/client.key"

server:
  host: "relay.example.com"
  port: 8443
  jwt_token: "your-token"
```

#### Новая конфигурация (v2.0):
```yaml
# Добавьте новые секции
protocol:
  version: "2.0"
  features:
    - "tls"
    - "heartbeat"
    - "tunnel_info"
    - "multi_tenant"
    - "proxy"
    - "quic"
    - "metrics"

tenant:
  id: "tenant_001"
  name: "Example Tenant"

# Остальные секции остаются без изменений
tls:
  enabled: true
  cert_file: "/etc/certs/client.crt"
  key_file: "/etc/certs/client.key"

server:
  host: "relay.example.com"
  port: 8443
  jwt_token: "your-token"
```

### 3. Обновление кода

#### Старый код (v1.0.0):
```go
import (
    "github.com/2gc-dev/cloudbridge-client/pkg/client"
    "github.com/2gc-dev/cloudbridge-client/pkg/protocol"
)

// Создание клиента
config := &client.Config{
    ProtocolOrder: []protocol.Protocol{protocol.QUIC, protocol.HTTP2, protocol.HTTP1},
    SwitchThreshold: 0.8,
    ConnectTimeout: 10 * time.Second,
    RequestTimeout: 30 * time.Second,
}

ic := client.NewIntegratedClient(config)
```

#### Новый код (v2.0):
```go
import (
    "github.com/2gc-dev/cloudbridge-client/pkg/client"
    "github.com/2gc-dev/cloudbridge-client/pkg/protocol"
    "github.com/2gc-dev/cloudbridge-client/pkg/health"
)

// Создание клиента с новыми возможностями
config := &client.Config{
    ProtocolOrder: []protocol.Protocol{protocol.QUIC, protocol.HTTP2, protocol.HTTP1},
    SwitchThreshold: 0.8,
    ConnectTimeout: 10 * time.Second,
    RequestTimeout: 30 * time.Second,
    
    // Новые поля для v2.0
    TenantID: "tenant_001",
    Version: "2.0",
    Features: []string{
        "tls", "heartbeat", "tunnel_info",
        "multi_tenant", "proxy", "quic", "metrics",
    },
    MetricsEnabled: true,
    HealthCheckEnabled: true,
    HealthCheckConfig: &health.Config{
        Interval: 30 * time.Second,
        Timeout: 10 * time.Second,
    },
}

ic := client.NewIntegratedClient(config)

// Использование новых возможностей
ic.SetTenantID("tenant_001")
metrics := ic.GetMetrics()
healthChecker := ic.GetHealthChecker()
```

## 🔧 Пошаговая миграция

### Этап 1: Подготовка (1-2 дня)

1. **Создайте backup текущей конфигурации**
   ```bash
   cp config.yaml config-v1-backup.yaml
   ```

2. **Обновите зависимости**
   ```bash
   go mod tidy
   go mod download
   ```

3. **Создайте новую конфигурацию v2.0**
   ```bash
   cp config/config-v2.yaml config.yaml
   # Отредактируйте под ваши нужды
   ```

### Этап 2: Обновление кода (2-3 дня)

1. **Обновите импорты**
   ```go
   // Добавьте новые импорты
   import (
       "github.com/2gc-dev/cloudbridge-client/pkg/metrics"
       "github.com/2gc-dev/cloudbridge-client/pkg/health"
   )
   ```

2. **Обновите создание клиента**
   ```go
   // Добавьте новые поля в конфигурацию
   config.TenantID = "your-tenant-id"
   config.Version = "2.0"
   config.MetricsEnabled = true
   config.HealthCheckEnabled = true
   ```

3. **Добавьте обработку метрик**
   ```go
   // Получение метрик
   if metrics := ic.GetMetrics(); metrics != nil {
       summary := metrics.GetMetricsSummary()
       log.Printf("Metrics: %+v", summary)
   }
   ```

4. **Добавьте health checks**
   ```go
   // Получение статуса здоровья
   if healthChecker := ic.GetHealthChecker(); healthChecker != nil {
       status := healthChecker.GetStatus()
       log.Printf("Health status: %s", status)
   }
   ```

### Этап 3: Тестирование (1-2 дня)

1. **Запустите тесты**
   ```bash
   go test ./...
   ```

2. **Протестируйте подключение**
   ```bash
   go run examples/v2_client_example.go
   ```

3. **Проверьте метрики**
   ```bash
   curl http://localhost:9090/metrics
   ```

4. **Проверьте health checks**
   ```bash
   curl http://localhost:8080/health
   ```

### Этап 4: Развертывание (1 день)

1. **Обновите production конфигурацию**
2. **Разверните новую версию**
3. **Мониторьте логи и метрики**
4. **Проверьте совместимость с relay сервером**

## 🔄 Backward Compatibility

### Поддержка v1.0.0

Клиент v2.0 поддерживает обратную совместимость с v1.0.0:

```go
// Для v1.0.0 совместимости
config := &client.Config{
    Version: "1.0.0",
    Features: []string{
        "tls", "jwt", "tunneling", "quic", "http2",
    },
    // Остальные поля как в v1.0.0
}

ic := client.NewIntegratedClient(config)
```

### Автоматическое определение версии

Клиент автоматически определяет версию протокола:

```go
// Автоматическое определение на основе конфигурации
if cfg.Protocol.Version == "1.0.0" {
    // Использует v1.0.0 протокол
} else {
    // Использует v2.0 протокол
}
```

## 📊 Мониторинг и метрики

### Prometheus метрики

Клиент v2.0 предоставляет расширенные метрики:

```bash
# Просмотр метрик
curl http://localhost:9090/metrics

# Основные метрики
client_connections_total
client_active_connections
client_protocol_latency_seconds
client_tunnel_creations_total
client_auth_attempts_total
client_heartbeats_total
```

### Health Checks

```bash
# Проверка здоровья
curl http://localhost:8080/health

# Результат
{
  "status": "healthy",
  "checks": {
    "connection": {
      "status": "healthy",
      "duration": "0.001s"
    },
    "protocol": {
      "status": "healthy", 
      "duration": "0.002s"
    }
  }
}
```

## 🚨 Troubleshooting

### Частые проблемы

1. **Ошибка аутентификации**
   ```
   Error: authentication failed: invalid tenant_id
   ```
   **Решение**: Проверьте правильность `tenant_id` в конфигурации

2. **Несовместимость протокола**
   ```
   Error: protocol version mismatch
   ```
   **Решение**: Убедитесь, что relay сервер поддерживает v2.0

3. **Ошибки метрик**
   ```
   Error: prometheus metrics registration failed
   ```
   **Решение**: Проверьте, что порт 9090 свободен

### Логи и отладка

```go
// Включение debug логирования
config.Logging.Level = "debug"

// Просмотр детальных логов
tail -f /var/log/cloudbridge-client/client.log
```

## 📈 Производительность

### Ожидаемые улучшения

- **Latency**: улучшение на 30-50% для QUIC
- **Throughput**: увеличение до 100MB/s+
- **Connection reuse**: >90%
- **Error rate**: <1%

### Мониторинг производительности

```bash
# Мониторинг метрик в реальном времени
watch -n 5 'curl -s http://localhost:9090/metrics | grep client_'

# Анализ производительности
go tool pprof http://localhost:9090/debug/pprof/profile
```

## 🔗 Полезные ссылки

- [Техническое задание](PROTOCOL_ALIGNMENT_TECHNICAL_SPECIFICATION.md)
- [Руководство по миграции](ARCHITECTURAL_MIGRATION_GUIDE.md)
- [API Reference](API_REFERENCE.md)
- [Примеры использования](../examples/)

## 📞 Поддержка

При возникновении проблем:

1. Проверьте логи: `/var/log/cloudbridge-client/client.log`
2. Проверьте метрики: `http://localhost:9090/metrics`
3. Проверьте health status: `http://localhost:8080/health`
4. Создайте issue в репозитории с детальным описанием проблемы

## ✅ Чек-лист миграции

- [ ] Backup текущей конфигурации
- [ ] Обновление зависимостей
- [ ] Создание новой конфигурации v2.0
- [ ] Обновление кода приложения
- [ ] Добавление метрик и health checks
- [ ] Тестирование в dev окружении
- [ ] Тестирование в staging окружении
- [ ] Развертывание в production
- [ ] Мониторинг после развертывания
- [ ] Документирование изменений

**Время выполнения**: 5-8 дней
**Сложность**: Средняя
**Риск**: Низкий (благодаря backward compatibility) 