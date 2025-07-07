# Руководство по развёртыванию: CloudBridge Relay Client

## Необходимые условия
- Go 1.20+
- Доступ к серверу relay (требуется TLS 1.3)
- Валидный JWT-токен или учётные данные Keycloak

## Сборка из исходников
```bash
git clone https://github.com/2gc-dev/cloudbridge-client.git
cd cloudbridge-client
go build -o cloudbridge-client ./cmd/cloudbridge-client
```

## Готовые бинарные файлы
- Скачайте с [страницы релизов](https://github.com/2gc-dev/cloudbridge-client/releases)
- Сделайте исполняемым: `chmod +x cloudbridge-client`

## Запуск клиента
```bash
./cloudbridge-client --token "ваш-jwt-токен"
```

## Использование конфигурационного файла
```bash
./cloudbridge-client --config config.yaml --token "ваш-jwt-токен"
```

## Переменные окружения
Все параметры можно задать через префикс `CLOUDBRIDGE_`, например:
```bash
export CLOUDBRIDGE_RELAY_HOST="relay.example.com"
export CLOUDBRIDGE_AUTH_SECRET="jwt-секрет"
```

## Пример systemd-сервиса
Создайте файл `/etc/systemd/system/cloudbridge-client.service`:
```ini
[Unit]
Description=CloudBridge Relay Client
After=network.target

[Service]
ExecStart=/path/to/cloudbridge-client --config /path/to/config.yaml --token "ваш-jwt-токен"
Restart=on-failure
User=ubuntu

[Install]
WantedBy=multi-user.target
```

## Обновление
- Получите последние изменения: `git pull`
- Пересоберите: `go build -o cloudbridge-client ./cmd/cloudbridge-client`

## Логи
- По умолчанию логи выводятся в stdout. Настраивается через `config.yaml`.

## Диагностика
- См. `docs/TROUBLESHOOTING.md` для типовых проблем. 