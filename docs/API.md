# API и протокол CloudBridge Relay Client

Все сообщения — JSON, кодировка UTF-8, без сжатия.

## Типы сообщений

### 1. Hello
- **Клиент → Сервер**
```json
{
  "type": "hello",
  "version": "1.0",
  "features": ["tls", "heartbeat", "tunnel_info"]
}
```
- **Сервер → Клиент**
```json
{
  "type": "hello_response",
  "version": "1.0",
  "features": ["tls", "heartbeat", "tunnel_info"]
}
```

### 2. Аутентификация
- **Клиент → Сервер**
```json
{
  "type": "auth",
  "token": "<jwt>"
}
```
- **Сервер → Клиент**
```json
{
  "type": "auth_response",
  "status": "ok",
  "client_id": "user123"
}
```

### 3. Управление туннелем
- **Клиент → Сервер**
```json
{
  "type": "tunnel_info",
  "tunnel_id": "tunnel_001",
  "local_port": 3389,
  "remote_host": "192.168.1.100",
  "remote_port": 3389
}
```
- **Сервер → Клиент**
```json
{
  "type": "tunnel_response",
  "status": "ok",
  "tunnel_id": "tunnel_001"
}
```

### 4. Heartbeat
- **Клиент → Сервер**
```json
{
  "type": "heartbeat"
}
```
- **Сервер → Клиент**
```json
{
  "type": "heartbeat_response"
}
```

### 5. Ошибка
- **Сервер → Клиент**
```json
{
  "type": "error",
  "code": "rate_limit_exceeded",
  "message": "Превышен лимит запросов для пользователя"
}
```

## Коды ошибок
- `invalid_token` — неверный или истёкший JWT-токен
- `rate_limit_exceeded` — превышен лимит запросов
- `connection_limit_reached` — превышено число соединений
- `server_unavailable` — сервер недоступен
- `invalid_tunnel_info` — некорректные параметры туннеля
- `unknown_message_type` — неизвестный тип сообщения

## Примечания
- Все поля обязательны, если не указано иное.
- Все сообщения должны быть валидным JSON в UTF-8.
- Сжатие сообщений не используется.
- Все соединения должны использовать TLS 1.3. 