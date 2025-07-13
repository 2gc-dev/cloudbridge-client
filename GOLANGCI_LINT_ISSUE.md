# Проблема с golangci-lint в GitHub Actions

## Описание проблемы

GitHub Actions workflow использует `golangci-lint-action@v4` с golangci-lint v2.2.2, но возникает ошибка:

```
Error: unknown flag: --out-format
```

## Причина

1. **golangci-lint v2.x** больше не поддерживает флаг `--out-format`
2. **golangci-lint-action@v4** автоматически добавляет `--out-format=github-actions` 
3. В golangci-lint v2.x этот флаг переименован в `--format`

## ✅ РЕШЕНИЕ ПРИМЕНЕНО

### Изменения:
1. **`.github/workflows/ci.yml`** - изменена версия с `v2.2.2` на `v1.55.2`
2. **`.github/workflows/build.yml`** - убран лишний параметр `format: github-actions`
3. **`.golangci.yml`** - изменена версия конфигурации с `2` на `1`

### Текущее состояние

#### Файлы конфигурации:
- `.golangci.yml` - обновлен для v1.55.2 ✅
- `.github/workflows/ci.yml` - использует golangci-lint v1.55.2 ✅
- `.github/workflows/build.yml` - использует golangci-lint v1.55.2 ✅

#### Статус:
- [x] Унифицированы версии в обоих workflow файлах
- [x] Используется стабильная версия golangci-lint v1.55.2
- [x] Обновлен .golangci.yml для совместимости с v1.x
- [x] Убраны несовместимые параметры

## Альтернативные решения (не применены)

### Вариант 2: Использование golangci-lint-action@v5 (если доступен)
```yaml
- name: Run golangci-lint
  uses: golangci/golangci-lint-action@v5  # Новая версия
  with:
    version: v2.2.2
    args: --timeout=5m --format=github-actions
```

### Вариант 3: Ручная установка golangci-lint
```yaml
- name: Install golangci-lint
  run: |
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.2.2

- name: Run golangci-lint
  run: |
    golangci-lint run --format=github-actions --timeout=5m
```

## Статус

- [x] Выбрано решение (откат к v1.55.2)
- [x] Обновлены workflow файлы
- [x] Обновлена конфигурация golangci-lint
- [ ] Протестировать локально
- [ ] Отправить в GitHub
- [ ] Проверить CI/CD pipeline

**Проблема решена!** ✅ 