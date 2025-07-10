# 🏗️ Руководство по архитектурной миграции: CloudBridge Client → Relay Server v2.0

## 📋 Обзор

Данное руководство содержит детальный анализ архитектурных различий между CloudBridge Client и CloudBridge Relay Server v2.0, а также пошаговые инструкции по миграции.

## 🔍 Анализ архитектурных различий

### 1. Протокольный стек

#### 1.1 Relay Server v2.0 протокольный стек
```
┌─────────────────────────────────────┐
│           Application Layer         │
│  ┌─────────────────────────────────┐ │
│  │      Multi-tenant Logic         │ │
│  │  ┌─────────────────────────────┐ │ │
│  │  │      Tunnel Manager         │ │ │
│  │  │  ┌─────────────────────────┐ │ │ │
│  │  │  │    Protocol Engine      │ │ │ │
│  │  │  │  ┌─────────────────────┐ │ │ │ │
│  │  │  │  │   QUIC Protocol     │ │ │ │ │
│  │  │  │  │  ┌─────────────────┐ │ │ │ │ │
│  │  │  │  │  │  HTTP/2 Proto   │ │ │ │ │ │
│  │  │  │  │  │ ┌───────────────┐│ │ │ │ │ │
│  │  │  │  │  │ │ HTTP/1.1 Proto││ │ │ │ │ │
│  │  │  │  │  │ └───────────────┘│ │ │ │ │ │
│  │  │  │  │  └─────────────────┘ │ │ │ │ │
│  │  │  │  └─────────────────────┘ │ │ │ │
│  │  │  └─────────────────────────┘ │ │ │
│  │  └─────────────────────────────┘ │ │
│  └─────────────────────────────────┘ │
└─────────────────────────────────────┘
```

#### 1.2 Client текущий протокольный стек
```
┌─────────────────────────────────────┐
│           Application Layer         │
│  ┌─────────────────────────────────┐ │
│  │      Basic Client Logic         │ │
│  │  ┌─────────────────────────────┐ │ │
│  │  │    Simple Protocol Engine   │ │ │
│  │  │  ┌─────────────────────────┐ │ │ │
│  │  │  │   Basic QUIC Support    │ │ │ │
│  │  │  │  ┌─────────────────────┐ │ │ │ │
│  │  │  │  │   HTTP/2 Support    │ │ │ │ │
│  │  │  │  │ ┌───────────────────┐│ │ │ │ │
│  │  │  │  │ │ HTTP/1.1 Support ││ │ │ │ │
│  │  │  │  │ └───────────────────┘│ │ │ │ │
│  │  │  │  └─────────────────────┘ │ │ │ │
│  │  │  └─────────────────────────┘ │ │ │
│  │  └─────────────────────────────┘ │ │
│  └─────────────────────────────────┘ │
└─────────────────────────────────────┘
```

**Ключевые различия:**
- ❌ Отсутствие multi-tenant logic
- ❌ Упрощенный tunnel manager
- ❌ Базовая поддержка QUIC
- ❌ Отсутствие advanced protocol features

### 2. Система метрик

#### 2.1 Relay Server v2.0 метрики
```go
// Enhanced Metrics Structure
type EnhancedMetrics struct {
    // Protocol Metrics
    protocolLatency    *prometheus.HistogramVec
    protocolErrors     *prometheus.CounterVec
    protocolSwitches   *prometheus.CounterVec
    
    // Connection Metrics
    connectionHealth   *prometheus.GaugeVec
    connectionPool     *prometheus.GaugeVec
    
    // Tunnel Metrics
    tunnelMetrics      *prometheus.CounterVec
    tunnelLatency      *prometheus.HistogramVec
    
    // Tenant Metrics
    tenantMetrics      *prometheus.CounterVec
    tenantLimits       *prometheus.GaugeVec
    
    // Health Check Metrics
    healthCheckStatus  *prometheus.GaugeVec
    healthCheckLatency *prometheus.HistogramVec
    
    // Build Info
    buildInfo          *prometheus.GaugeVec
}
```

#### 2.2 Client текущие метрики
```go
// Basic Metrics Structure
type Metrics struct {
    ActiveConnections int64
    TotalConnections  int64
    Errors           int64
    ProtocolStats    map[string]interface{}
}
```

**Ключевые различия:**
- ❌ Отсутствие Prometheus integration
- ❌ Отсутствие tenant-specific метрик
- ❌ Отсутствие health check метрик
- ❌ Отсутствие build info метрик

### 3. QUIC протокол

#### 3.1 Relay Server v2.0 QUIC
```go
type QUICConnection struct {
    conn           *quic.Conn
    streams        map[quic.StreamID]*quic.Stream
    connectionID   uuid.UUID
    activeStreams  int32
    totalStreams   int64
    config         *QUICConfig
    metrics        *metrics.EnhancedMetrics
}

type QUICConfig struct {
    MaxStreams        int
    KeepAliveInterval time.Duration
    IdleTimeout       time.Duration
    MaxIdleTime       time.Duration
    EnableMetrics     bool
    EnableTracing     bool
}
```

#### 3.2 Client QUIC
```go
type QUICClient struct {
    connection *quic.Connection
    config     *QUICConfig
}

type QUICConfig struct {
    TLSConfig        *tls.Config
    KeepAlive        bool
    KeepAlivePeriod  time.Duration
    IdleTimeout      time.Duration
}
```

**Ключевые различия:**
- ❌ Отсутствие stream multiplexing
- ❌ Отсутствие connection pooling
- ❌ Отсутствие enhanced metrics
- ❌ Отсутствие graceful shutdown

### 4. Multi-tenancy

#### 4.1 Relay Server v2.0 Multi-tenancy
```go
type Tenant struct {
    ID          string
    Name        string
    Limits      *TenantLimits
    Permissions []string
    Metrics     *TenantMetrics
}

type TenantLimits struct {
    MaxTunnels      int
    MaxConnections  int
    MaxBandwidth    int64
    MaxStorage      int64
}

type TenantMetrics struct {
    ActiveTunnels   int64
    ActiveConnections int64
    BandwidthUsed   int64
    StorageUsed     int64
}
```

#### 4.2 Client Multi-tenancy
```go
// Отсутствует поддержка multi-tenancy
```

**Ключевые различия:**
- ❌ Полное отсутствие multi-tenancy
- ❌ Отсутствие tenant isolation
- ❌ Отсутствие resource limits
- ❌ Отсутствие tenant-specific metrics

## 🚀 План миграции

### Этап 1: Подготовка инфраструктуры (2-3 дня)

#### 1.1 Обновление зависимостей
```go
// go.mod обновления
require (
    github.com/prometheus/client_golang v1.17.0
    github.com/quic-go/quic-go v0.40.0
    github.com/google/uuid v1.4.0
    go.uber.org/zap v1.26.0
)
```

#### 1.2 Создание новых пакетов
```
pkg/
├── metrics/
│   ├── enhanced_metrics.go
│   ├── prometheus.go
│   └── health_metrics.go
├── tenant/
│   ├── tenant.go
│   ├── limits.go
│   └── isolation.go
├── health/
│   ├── health_checker.go
│   └── checks.go
└── connection/
    ├── pool.go
    └── quic_pool.go
```

### Этап 2: Миграция протокольного стека (4-5 дней)

#### 2.1 Обновление Protocol Engine
```go
// pkg/protocol/enhanced_engine.go
type EnhancedProtocolEngine struct {
    currentProtocol Protocol
    preferredOrder  []Protocol
    metrics         map[Protocol]*ProtocolMetrics
    fallbackEnabled bool
    switchThreshold float64
    mu              sync.RWMutex
    logger          *zap.Logger
    prometheusMetrics *PrometheusMetrics
}

type PrometheusMetrics struct {
    protocolLatency    *prometheus.HistogramVec
    protocolErrors     *prometheus.CounterVec
    protocolSwitches   *prometheus.CounterVec
    protocolSuccess    *prometheus.CounterVec
}
```

#### 2.2 Обновление QUIC протокола
```go
// pkg/protocol/enhanced_quic.go
type EnhancedQUICClient struct {
    connection     *quic.Connection
    streams        map[quic.StreamID]*quic.Stream
    connectionID   uuid.UUID
    activeStreams  int32
    totalStreams   int64
    config         *EnhancedQUICConfig
    metrics        *metrics.EnhancedMetrics
    logger         *zap.Logger
}

type EnhancedQUICConfig struct {
    MaxStreams        int
    KeepAliveInterval time.Duration
    IdleTimeout       time.Duration
    MaxIdleTime       time.Duration
    EnableMetrics     bool
    EnableTracing     bool
    TLSConfig         *tls.Config
}
```

### Этап 3: Интеграция Multi-tenancy (3-4 дня)

#### 3.1 Создание Tenant Manager
```go
// pkg/tenant/manager.go
type TenantManager struct {
    tenants map[string]*Tenant
    mu      sync.RWMutex
    logger  *zap.Logger
    metrics *metrics.EnhancedMetrics
}

type Tenant struct {
    ID          string
    Name        string
    Limits      *TenantLimits
    Permissions []string
    Metrics     *TenantMetrics
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

func (tm *TenantManager) CreateTenant(id, name string, limits *TenantLimits) (*Tenant, error) {
    tm.mu.Lock()
    defer tm.mu.Unlock()
    
    tenant := &Tenant{
        ID:        id,
        Name:      name,
        Limits:    limits,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        Metrics:   &TenantMetrics{},
    }
    
    tm.tenants[id] = tenant
    return tenant, nil
}
```

#### 3.2 Обновление аутентификации
```go
// pkg/auth/enhanced_auth.go
type EnhancedAuthMessage struct {
    Type      string `json:"type"`
    Token     string `json:"token"`
    TenantID  string `json:"tenant_id"`
    Version   string `json:"version"`
}

type EnhancedAuthResponse struct {
    Type        string                 `json:"type"`
    Status      string                 `json:"status"`
    ClientID    string                 `json:"client_id"`
    SessionID   string                 `json:"session_id"`
    Permissions []string               `json:"permissions"`
    Limits      map[string]interface{} `json:"limits"`
    TenantInfo  *TenantInfo            `json:"tenant_info"`
}

type TenantInfo struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Limits      map[string]interface{} `json:"limits"`
    Permissions []string               `json:"permissions"`
}
```

### Этап 4: Интеграция Enhanced Metrics (3-4 дня)

#### 4.1 Создание Enhanced Metrics
```go
// pkg/metrics/enhanced_metrics.go
type EnhancedMetrics struct {
    // Protocol Metrics
    protocolLatency    *prometheus.HistogramVec
    protocolErrors     *prometheus.CounterVec
    protocolSwitches   *prometheus.CounterVec
    protocolSuccess    *prometheus.CounterVec
    
    // Connection Metrics
    connectionHealth   *prometheus.GaugeVec
    connectionPool     *prometheus.GaugeVec
    connectionLatency  *prometheus.HistogramVec
    
    // Tunnel Metrics
    tunnelMetrics      *prometheus.CounterVec
    tunnelLatency      *prometheus.HistogramVec
    tunnelBandwidth    *prometheus.CounterVec
    
    // Tenant Metrics
    tenantMetrics      *prometheus.CounterVec
    tenantLimits       *prometheus.GaugeVec
    tenantUsage        *prometheus.GaugeVec
    
    // Health Check Metrics
    healthCheckStatus  *prometheus.GaugeVec
    healthCheckLatency *prometheus.HistogramVec
    
    // Build Info
    buildInfo          *prometheus.GaugeVec
}

func NewEnhancedMetrics() *EnhancedMetrics {
    return &EnhancedMetrics{
        protocolLatency: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "cloudbridge_protocol_latency_seconds",
                Help: "Protocol latency in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"protocol", "operation"},
        ),
        // ... другие метрики
    }
}
```

#### 4.2 Создание Health Check системы
```go
// pkg/health/health_checker.go
type HealthCheck interface {
    Name() string
    Check(ctx context.Context) error
    IsCritical() bool
    GetMetrics() map[string]interface{}
}

type HealthChecker struct {
    checks []HealthCheck
    logger *zap.Logger
    metrics *metrics.EnhancedMetrics
}

func (hc *HealthChecker) AddCheck(check HealthCheck) {
    hc.checks = append(hc.checks, check)
}

func (hc *HealthChecker) RunChecks(ctx context.Context) map[string]error {
    results := make(map[string]error)
    
    for _, check := range hc.checks {
        start := time.Now()
        if err := check.Check(ctx); err != nil {
            results[check.Name()] = err
            hc.metrics.healthCheckStatus.WithLabelValues(check.Name(), "failed").Set(0)
        } else {
            hc.metrics.healthCheckStatus.WithLabelValues(check.Name(), "healthy").Set(1)
        }
        
        duration := time.Since(start)
        hc.metrics.healthCheckLatency.WithLabelValues(check.Name()).Observe(duration.Seconds())
    }
    
    return results
}
```

### Этап 5: Обновление Integrated Client (2-3 дня)

#### 5.1 Обновление конфигурации
```go
// pkg/client/enhanced_config.go
type EnhancedConfig struct {
    TLSConfig        *tls.Config
    CircuitBreaker   *circuitbreaker.Config
    ProtocolOrder    []protocol.Protocol
    SwitchThreshold  float64
    ConnectTimeout   time.Duration
    RequestTimeout   time.Duration
    
    // Новые поля
    TenantID         string
    TenantLimits     *tenant.TenantLimits
    EnableMetrics    bool
    EnableHealthCheck bool
    MetricsPort      int
    HealthCheckPort  int
}
```

#### 5.2 Обновление Integrated Client
```go
// pkg/client/enhanced_integrated_client.go
type EnhancedIntegratedClient struct {
    protocolEngine *protocol.EnhancedProtocolEngine
    circuitBreaker *circuitbreaker.CircuitBreaker
    currentProtocol protocol.Protocol
    clients        map[protocol.Protocol]interface{}
    mu             sync.RWMutex
    config         *EnhancedConfig
    
    // Новые поля
    tenantManager   *tenant.TenantManager
    metrics         *metrics.EnhancedMetrics
    healthChecker   *health.HealthChecker
    logger          *zap.Logger
}

func (eic *EnhancedIntegratedClient) Connect(ctx context.Context, address string) error {
    // Инициализация метрик
    if eic.config.EnableMetrics {
        eic.metrics = metrics.NewEnhancedMetrics()
    }
    
    // Инициализация health checker
    if eic.config.EnableHealthCheck {
        eic.healthChecker = health.NewHealthChecker(eic.logger, eic.metrics)
        eic.setupHealthChecks()
    }
    
    // Подключение с multi-tenancy
    return eic.connectWithTenancy(ctx, address)
}

func (eic *EnhancedIntegratedClient) connectWithTenancy(ctx context.Context, address string) error {
    // Проверка лимитов тенанта
    if err := eic.checkTenantLimits(); err != nil {
        return fmt.Errorf("tenant limits exceeded: %w", err)
    }
    
    // Подключение с обновленным протоколом
    return eic.connectWithEnhancedProtocol(ctx, address)
}
```

## 📊 Критерии успешной миграции

### 5.1 Функциональные критерии
- [ ] 100% совместимость с Relay Server v2.0
- [ ] Поддержка multi-tenancy
- [ ] Enhanced metrics с Prometheus
- [ ] Health check система
- [ ] Улучшенный QUIC протокол

### 5.2 Производительность
- [ ] Latency < 50ms для QUIC
- [ ] Throughput > 100MB/s
- [ ] Connection reuse > 90%
- [ ] Error rate < 1%

### 5.3 Надежность
- [ ] 99.9% uptime
- [ ] Automatic failover
- [ ] Graceful degradation
- [ ] Circuit breaker protection

## 🔧 Инструменты для миграции

### 6.1 Тестирование
```bash
# Unit тесты
go test ./pkg/...

# Integration тесты
go test ./test/integration/...

# Performance тесты
go test -bench=. ./test/performance/...

# Coverage
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out
```

### 6.2 Мониторинг
```bash
# Prometheus метрики
curl http://localhost:9090/metrics

# Health check
curl http://localhost:8080/health

# Tenant метрики
curl http://localhost:9090/metrics | grep tenant
```

### 6.3 Логирование
```go
// Настройка структурированного логирования
logger, _ := zap.NewProduction()
defer logger.Sync()

logger.Info("Enhanced client started",
    zap.String("version", "2.0"),
    zap.String("tenant_id", config.TenantID),
    zap.Bool("metrics_enabled", config.EnableMetrics),
)
```

## 📝 Заключение

Данное руководство по архитектурной миграции предоставляет детальный план по приведению CloudBridge Client в соответствие с CloudBridge Relay Server v2.0. Успешная реализация обеспечит полную совместимость, улучшенную производительность и расширенные возможности мониторинга.

Ключевые преимущества после миграции:
- ✅ Полная совместимость с Relay Server v2.0
- ✅ Поддержка multi-tenancy
- ✅ Enhanced metrics и мониторинг
- ✅ Улучшенная производительность QUIC
- ✅ Health check система
- ✅ Graceful degradation и failover 