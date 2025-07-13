#!/bin/bash

# Скрипт для автоматического создания релиза
# Использование: ./scripts/create_release.sh [major|minor|patch]

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция для вывода с цветом
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Проверяем, что мы в git репозитории
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "Не найден git репозиторий"
    exit 1
fi

# Получаем текущую версию
CURRENT_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
print_info "Текущая версия: $CURRENT_TAG"

# Убираем префикс 'v' для парсинга
CURRENT_VERSION=${CURRENT_TAG#v}

# Парсим версию
IFS='.' read -ra VERSION_PARTS <<< "$CURRENT_VERSION"
MAJOR=${VERSION_PARTS[0]:-0}
MINOR=${VERSION_PARTS[1]:-0}
PATCH=${VERSION_PARTS[2]:-0}

# Определяем тип релиза
RELEASE_TYPE=${1:-patch}

case $RELEASE_TYPE in
    major)
        NEW_MAJOR=$((MAJOR + 1))
        NEW_MINOR=0
        NEW_PATCH=0
        print_info "Создаем major релиз: $MAJOR.$MINOR.$PATCH → $NEW_MAJOR.$NEW_MINOR.$NEW_PATCH"
        ;;
    minor)
        NEW_MAJOR=$MAJOR
        NEW_MINOR=$((MINOR + 1))
        NEW_PATCH=0
        print_info "Создаем minor релиз: $MAJOR.$MINOR.$PATCH → $NEW_MAJOR.$NEW_MINOR.$NEW_PATCH"
        ;;
    patch)
        NEW_MAJOR=$MAJOR
        NEW_MINOR=$MINOR
        NEW_PATCH=$((PATCH + 1))
        print_info "Создаем patch релиз: $MAJOR.$MINOR.$PATCH → $NEW_MAJOR.$NEW_MINOR.$NEW_PATCH"
        ;;
    *)
        print_error "Неизвестный тип релиза: $RELEASE_TYPE"
        print_info "Используйте: major, minor или patch"
        exit 1
        ;;
esac

# Формируем новую версию
NEW_VERSION="$NEW_MAJOR.$NEW_MINOR.$NEW_PATCH"
NEW_TAG="v$NEW_VERSION"

# Проверяем, что нет незакоммиченных изменений
if ! git diff-index --quiet HEAD --; then
    print_warning "Есть незакоммиченные изменения. Коммитите их перед созданием релиза."
    git status --short
    read -p "Продолжить? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Отменено"
        exit 1
    fi
fi

# Проверяем, что мы на main ветке
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ]; then
    print_warning "Вы не на main ветке (текущая: $CURRENT_BRANCH)"
    read -p "Продолжить? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Отменено"
        exit 1
    fi
fi

# Создаем тег
print_info "Создаем тег: $NEW_TAG"
git tag -a "$NEW_TAG" -m "Release $NEW_TAG"

# Отправляем изменения
print_info "Отправляем изменения в репозиторий..."
git push origin main
git push origin "$NEW_TAG"

print_success "Релиз $NEW_TAG создан и отправлен!"
print_info "GitHub Actions автоматически создаст релиз с бинарниками"
print_info "Ссылка на релиз: https://github.com/$(git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^/]*\).*/\1/')/releases/tag/$NEW_TAG" 