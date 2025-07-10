# CloudBridge Client Protocol Negotiation Guide

## Обзор

CloudBridge Client использует интеллектуальную систему согласования протокола, которая **динамически выбирает оптимальный протокол для каждого соединения**. Всё это происходит незаметно в фоновом режиме — никакого вмешательства пользователя не требуется.

## Поддерживаемые протоколы

### 1. QUIC (UDP, RFC 9000) - По умолчанию
- **Характеристики:** Быстрый, поддерживает 0-RTT и мультиплексирование
- **Преимущества:** 
  - Низкая задержка (0-RTT)
  - Мультиплексирование потоков
  - Встроенная шифрование
  - Устойчивость к потере пакетов
- **Когда используется:** По умолчанию для всех новых соединений

### 2. HTTP/2 - Резервный вариант
- **Характеристики:** Мультиплексирование, надежный fallback
- **Преимущества:**
  - Мультиплексирование через TCP
  - Высокая совместимость
  - Надежная доставка
- **Когда используется:** Если UDP заблокирован или QUIC недоступен

### 3. HTTP/1.1 - Окончательный резерв
- **Характеристики:** Legacy compatibility
- **Преимущества:**
  - Максимальная совместимость
  - Работает везде
- **Когда используется:** Для совместимости с legacy-системами

## Как работает согласование

### Алгоритм выбора протокола

```go
// 1. Получить оптимальный протокол для соединения
optimalProtocol := protocolEngine.GetOptimalProtocolForConnection(ctx, address)

// 2. Попробовать подключиться с оптимальным протоколом
if err := tryConnect(optimalProtocol); err == nil {
    return optimalProtocol // Успех!
}

// 3. Если не удалось, попробовать fallback протоколы
fallbackProtocols := getFallbackProtocols(optimalProtocol)
for _, protocol := range fallbackProtocols {
    if err := tryConnect(protocol); err == nil {
        return protocol // Успех с fallback!
    }
}
```

### Порядок приоритетов

1. **QUIC** - самый быстрый и современный
2. **HTTP/2** - надежный fallback с мультиплексированием  
3. **HTTP/1.1** - максимальная совместимость

### Автоматическое переключение

Система постоянно мониторит производительность и автоматически переключается на лучший протокол:

```go
// Проверка необходимости переключения
if protocolEngine.ShouldSwitchProtocol(currentProtocol) {
    nextProtocol := protocolEngine.GetNextProtocol(currentProtocol)
    client.SwitchProtocol(nextProtocol)
}
```

## Конфигурация

### Базовая настройка

```go
config := client.DefaultConfig()
config.ProtocolOrder = []protocol.Protocol{
    protocol.QUIC,  // Попробовать QUIC первым
    protocol.HTTP2, // Fallback к HTTP/2
    protocol.HTTP1, // Финальный fallback к HTTP/1.1
}
config.SwitchThreshold = 0.8  // Порог для переключения (80% неудач)
config.ConnectTimeout = 10 * time.Second
config.RequestTimeout = 30 * time.Second
```

### Включение автоматического переключения

```go
client := client.NewIntegratedClient(config)
client.EnableAutoProtocolSwitching() // Включить автоматическое переключение
```

## Мониторинг и статистика

### Получение статистики протоколов

```go
stats := client.GetStats()
for protocolName, info := range stats {
    infoMap := info.(map[string]interface{})
    fmt.Printf("Protocol: %s\n", protocolName)
    fmt.Printf("  Success: %v, Failures: %v\n", 
        infoMap["success_count"], infoMap["failure_count"])
    fmt.Printf("  Average Latency: %v\n", infoMap["average_latency"])
    fmt.Printf("  Failure Rate: %.2f%%\n", 
        infoMap["failure_rate"].(float64)*100)
    fmt.Printf("  Available: %v\n", infoMap["is_available"])
}
```

### Рекомендации по протоколам

```go
recommendation := client.GetProtocolRecommendation()
for protocolName, info := range recommendation {
    infoMap := info.(map[string]interface{})
    fmt.Printf("Protocol: %s\n", protocolName)
    fmt.Printf("  Description: %s\n", infoMap["description"])
    fmt.Printf("  Priority: %d\n", infoMap["priority"])
    fmt.Printf("  Recommended: %v\n", infoMap["recommended"])
    fmt.Printf("  Available: %v\n", infoMap["is_available"])
}
```

## Примеры использования

### Простой пример

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/2gc-dev/cloudbridge-client/pkg/client"
    "github.com/2gc-dev/cloudbridge-client/pkg/protocol"
)

func main() {
    // Создать клиент с автоматическим согласованием протокола
    config := client.DefaultConfig()
    client := client.NewIntegratedClient(config)
    
    // Включить автоматическое переключение
    client.EnableAutoProtocolSwitching()
    
    // Подключиться (система автоматически выберет лучший протокол)
    err := client.Connect(context.Background(), "relay.example.com:443")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Connected using %s protocol\n", client.GetCurrentProtocol())
    
    // Отправить данные
    data := []byte("Hello, CloudBridge!")
    err = client.Send(data)
    if err != nil {
        log.Fatal(err)
    }
    
    // Получить ответ
    buffer := make([]byte, 1024)
    n, err := client.Receive(buffer)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Received %d bytes\n", n)
    
    // Закрыть соединение
    client.Close()
}
```

### Продвинутый пример с мониторингом

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/2gc-dev/cloudbridge-client/pkg/client"
    "github.com/2gc-dev/cloudbridge-client/pkg/protocol"
)

func main() {
    // Расширенная конфигурация
    config := client.DefaultConfig()
    config.ProtocolOrder = []protocol.Protocol{
        protocol.QUIC,  // QUIC первым (быстрый, 0-RTT)
        protocol.HTTP2, // HTTP/2 как fallback
        protocol.HTTP1, // HTTP/1.1 как последний резерв
    }
    config.SwitchThreshold = 0.8
    config.ConnectTimeout = 10 * time.Second
    config.RequestTimeout = 30 * time.Second
    config.MetricsEnabled = true
    config.HealthCheckEnabled = true
    
    client := client.NewIntegratedClient(config)
    client.EnableAutoProtocolSwitching()
    
    // Показать начальные рекомендации
    fmt.Println("Initial Protocol Recommendations:")
    recommendation := client.GetProtocolRecommendation()
    for protocolName, info := range recommendation {
        infoMap := info.(map[string]interface{})
        fmt.Printf("  %s: %s\n", protocolName, infoMap["description"])
        fmt.Printf("    Priority: %d, Recommended: %v\n",
            infoMap["priority"], infoMap["recommended"])
    }
    
    // Подключиться
    err := client.Connect(context.Background(), "relay.example.com:443")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Connected using %s protocol\n", client.GetCurrentProtocol())
    
    // Симуляция работы с автоматическим переключением
    for i := 0; i < 10; i++ {
        // Отправить данные
        data := []byte(fmt.Sprintf("Message %d", i))
        err := client.Send(data)
        if err != nil {
            log.Printf("Send failed: %v", err)
            break
        }
        
        // Получить ответ
        buffer := make([]byte, 1024)
        n, err := client.Receive(buffer)
        if err != nil {
            log.Printf("Receive failed: %v", err)
            break
        }
        
        fmt.Printf("Sent %d bytes, received %d bytes\n", len(data), n)
        
        // Проверить необходимость переключения протокола
        if i%3 == 0 {
            err = client.AutoSwitchProtocol()
            if err == nil {
                fmt.Printf("Switched to %s protocol\n", client.GetCurrentProtocol())
            }
        }
        
        time.Sleep(100 * time.Millisecond)
    }
    
    // Показать финальную статистику
    fmt.Println("\nFinal Statistics:")
    stats := client.GetStats()
    for protocolName, info := range stats {
        if protocolName == "client" || protocolName == "metrics" {
            continue
        }
        infoMap := info.(map[string]interface{})
        fmt.Printf("  %s:\n", protocolName)
        fmt.Printf("    Success: %v, Failures: %v\n", 
            infoMap["success_count"], infoMap["failure_count"])
        fmt.Printf("    Average Latency: %v\n", infoMap["average_latency"])
        fmt.Printf("    Failure Rate: %.2f%%\n", 
            infoMap["failure_rate"].(float64)*100)
    }
    
    client.Close()
}
```

## Особенности реализации

### Потокобезопасность

Все операции с протоколами защищены мьютексами для обеспечения потокобезопасности:

```go
type ProtocolEngine struct {
    mu sync.RWMutex
    // ... другие поля
}
```

### Метрики производительности

Система отслеживает:
- Количество успешных/неудачных соединений
- Среднюю задержку
- Причины неудач
- Время последнего использования

### Автоматическое восстановление

Если протокол становится недоступным, система:
1. Помечает его как недоступный
2. Автоматически переключается на следующий
3. Периодически проверяет возможность восстановления

## Лучшие практики

### 1. Всегда включайте автоматическое переключение

```go
client.EnableAutoProtocolSwitching()
```

### 2. Мониторьте статистику

```go
// Периодически проверяйте статистику
stats := client.GetStats()
// Анализируйте производительность протоколов
```

### 3. Настройте подходящие таймауты

```go
config.ConnectTimeout = 10 * time.Second  // Для быстрого fallback
config.RequestTimeout = 30 * time.Second  // Для стабильности
```

### 4. Используйте метрики

```go
config.MetricsEnabled = true
// Интегрируйте с системами мониторинга
```

## Заключение

Система согласования протокола CloudBridge Client обеспечивает:

✅ **Автоматический выбор** оптимального протокола  
✅ **Прозрачный fallback** при проблемах  
✅ **Мониторинг производительности** в реальном времени  
✅ **Автоматическое переключение** на лучший протокол  
✅ **Максимальную совместимость** с legacy-системами  

**Всё это происходит без вмешательства пользователя!** 