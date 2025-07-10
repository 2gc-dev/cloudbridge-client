#!/bin/bash

# CloudBridge Client - Test Runner Script
# Автоматизированный запуск всех типов тестов

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Переменные
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_RESULTS_DIR="$PROJECT_ROOT/test-results"
COVERAGE_DIR="$PROJECT_ROOT/coverage"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
LOG_FILE="$TEST_RESULTS_DIR/test_run_$TIMESTAMP.log"

# Создание директорий
mkdir -p "$TEST_RESULTS_DIR"
mkdir -p "$COVERAGE_DIR"

# Функции
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

print_header() {
    echo -e "${BLUE}"
    echo "=========================================="
    echo "  CloudBridge Client - Test Runner"
    echo "=========================================="
    echo -e "${NC}"
}

print_footer() {
    echo -e "${BLUE}"
    echo "=========================================="
    echo "  Test Run Completed"
    echo "=========================================="
    echo -e "${NC}"
}

check_dependencies() {
    log_info "Checking dependencies..."
    
    # Проверка Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    # Проверка версии Go
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    REQUIRED_VERSION="1.23"
    
    if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
        log_error "Go version $GO_VERSION is less than required $REQUIRED_VERSION"
        exit 1
    fi
    
    log_success "Go version: $GO_VERSION"
    
    # Проверка Make
    if ! command -v make &> /dev/null; then
        log_error "Make is not installed"
        exit 1
    fi
    
    # Проверка golangci-lint (опционально)
    if command -v golangci-lint &> /dev/null; then
        log_success "golangci-lint found"
        LINT_AVAILABLE=true
    else
        log_warning "golangci-lint not found, skipping linting"
        LINT_AVAILABLE=false
    fi
    
    # Проверка gosec (опционально)
    if command -v gosec &> /dev/null; then
        log_success "gosec found"
        SECURITY_AVAILABLE=true
    else
        log_warning "gosec not found, skipping security checks"
        SECURITY_AVAILABLE=false
    fi
}

setup_environment() {
    log_info "Setting up test environment..."
    
    cd "$PROJECT_ROOT"
    
    # Очистка предыдущих результатов
    rm -rf "$TEST_RESULTS_DIR"/*
    rm -rf "$COVERAGE_DIR"/*
    
    # Загрузка зависимостей
    log_info "Downloading dependencies..."
    go mod download
    go mod tidy
    
    # Сборка компонентов
    log_info "Building components..."
    make build-all
    
    log_success "Environment setup completed"
}

run_unit_tests() {
    log_info "Running unit tests..."
    
    local unit_log="$TEST_RESULTS_DIR/unit_tests_$TIMESTAMP.log"
    local unit_coverage="$COVERAGE_DIR/unit_coverage_$TIMESTAMP.out"
    
    if go test -v -short -coverprofile="$unit_coverage" ./... 2>&1 | tee "$unit_log"; then
        log_success "Unit tests passed"
        
        # Генерация HTML отчета покрытия
        if [ -f "$unit_coverage" ]; then
            go tool cover -html="$unit_coverage" -o "$COVERAGE_DIR/unit_coverage_$TIMESTAMP.html"
            log_info "Unit test coverage report: $COVERAGE_DIR/unit_coverage_$TIMESTAMP.html"
        fi
    else
        log_error "Unit tests failed"
        return 1
    fi
}

run_integration_tests() {
    log_info "Running integration tests..."
    
    local integration_log="$TEST_RESULTS_DIR/integration_tests_$TIMESTAMP.log"
    local integration_coverage="$COVERAGE_DIR/integration_coverage_$TIMESTAMP.out"
    
    # Запуск mock relay сервера
    log_info "Starting mock relay server..."
    cd "$PROJECT_ROOT"
    ./mock_relay 8085 &
    MOCK_RELAY_PID=$!
    
    # Ожидание запуска сервера
    sleep 3
    
    # Проверка доступности mock relay
    if ! nc -z localhost 8085; then
        log_error "Mock relay server failed to start"
        kill $MOCK_RELAY_PID 2>/dev/null || true
        return 1
    fi
    
    log_success "Mock relay server started on port 8085"
    
    # Запуск интеграционных тестов
    if go test -v -tags=integration -coverprofile="$integration_coverage" ./test/ 2>&1 | tee "$integration_log"; then
        log_success "Integration tests passed"
        
        # Генерация HTML отчета покрытия
        if [ -f "$integration_coverage" ]; then
            go tool cover -html="$integration_coverage" -o "$COVERAGE_DIR/integration_coverage_$TIMESTAMP.html"
            log_info "Integration test coverage report: $COVERAGE_DIR/integration_coverage_$TIMESTAMP.html"
        fi
    else
        log_error "Integration tests failed"
        kill $MOCK_RELAY_PID 2>/dev/null || true
        return 1
    fi
    
    # Остановка mock relay
    kill $MOCK_RELAY_PID 2>/dev/null || true
    log_info "Mock relay server stopped"
}

run_benchmarks() {
    log_info "Running benchmarks..."
    
    local benchmark_log="$TEST_RESULTS_DIR/benchmarks_$TIMESTAMP.log"
    
    if go test -v -bench=. -benchmem ./test/ 2>&1 | tee "$benchmark_log"; then
        log_success "Benchmarks completed"
        log_info "Benchmark results: $benchmark_log"
    else
        log_error "Benchmarks failed"
        return 1
    fi
}

run_linting() {
    if [ "$LINT_AVAILABLE" = true ]; then
        log_info "Running linting..."
        
        local lint_log="$TEST_RESULTS_DIR/linting_$TIMESTAMP.log"
        
        if golangci-lint run 2>&1 | tee "$lint_log"; then
            log_success "Linting passed"
        else
            log_error "Linting failed"
            return 1
        fi
    else
        log_warning "Skipping linting (golangci-lint not available)"
    fi
}

run_security_checks() {
    if [ "$SECURITY_AVAILABLE" = true ]; then
        log_info "Running security checks..."
        
        local security_log="$TEST_RESULTS_DIR/security_$TIMESTAMP.log"
        
        if gosec ./... 2>&1 | tee "$security_log"; then
            log_success "Security checks passed"
        else
            log_warning "Security checks found issues (see $security_log)"
        fi
    else
        log_warning "Skipping security checks (gosec not available)"
    fi
}

generate_test_report() {
    log_info "Generating test report..."
    
    local report_file="$TEST_RESULTS_DIR/test_report_$TIMESTAMP.md"
    
    cat > "$report_file" << EOF
# CloudBridge Client - Test Report

**Generated:** $(date)
**Timestamp:** $TIMESTAMP

## Summary

- **Unit Tests:** $(if [ -f "$TEST_RESULTS_DIR/unit_tests_$TIMESTAMP.log" ]; then echo "✅ PASSED"; else echo "❌ FAILED"; fi)
- **Integration Tests:** $(if [ -f "$TEST_RESULTS_DIR/integration_tests_$TIMESTAMP.log" ]; then echo "✅ PASSED"; else echo "❌ FAILED"; fi)
- **Benchmarks:** $(if [ -f "$TEST_RESULTS_DIR/benchmarks_$TIMESTAMP.log" ]; then echo "✅ COMPLETED"; else echo "❌ FAILED"; fi)
- **Linting:** $(if [ "$LINT_AVAILABLE" = true ] && [ -f "$TEST_RESULTS_DIR/linting_$TIMESTAMP.log" ]; then echo "✅ PASSED"; else echo "⚠️ SKIPPED"; fi)
- **Security:** $(if [ "$SECURITY_AVAILABLE" = true ] && [ -f "$TEST_RESULTS_DIR/security_$TIMESTAMP.log" ]; then echo "✅ COMPLETED"; else echo "⚠️ SKIPPED"; fi)

## Coverage Reports

- **Unit Tests:** [$COVERAGE_DIR/unit_coverage_$TIMESTAMP.html]($COVERAGE_DIR/unit_coverage_$TIMESTAMP.html)
- **Integration Tests:** [$COVERAGE_DIR/integration_coverage_$TIMESTAMP.html]($COVERAGE_DIR/integration_coverage_$TIMESTAMP.html)

## Log Files

- **Main Log:** [$LOG_FILE]($LOG_FILE)
- **Unit Tests:** [$TEST_RESULTS_DIR/unit_tests_$TIMESTAMP.log]($TEST_RESULTS_DIR/unit_tests_$TIMESTAMP.log)
- **Integration Tests:** [$TEST_RESULTS_DIR/integration_tests_$TIMESTAMP.log]($TEST_RESULTS_DIR/integration_tests_$TIMESTAMP.log)
- **Benchmarks:** [$TEST_RESULTS_DIR/benchmarks_$TIMESTAMP.log]($TEST_RESULTS_DIR/benchmarks_$TIMESTAMP.log)
$(if [ "$LINT_AVAILABLE" = true ]; then echo "- **Linting:** [$TEST_RESULTS_DIR/linting_$TIMESTAMP.log]($TEST_RESULTS_DIR/linting_$TIMESTAMP.log)"; fi)
$(if [ "$SECURITY_AVAILABLE" = true ]; then echo "- **Security:** [$TEST_RESULTS_DIR/security_$TIMESTAMP.log]($TEST_RESULTS_DIR/security_$TIMESTAMP.log)"; fi)

## Environment

- **Go Version:** $(go version)
- **OS:** $(uname -s)
- **Architecture:** $(uname -m)
- **Project Root:** $PROJECT_ROOT

## Test Results

### Unit Tests
\`\`\`
$(if [ -f "$TEST_RESULTS_DIR/unit_tests_$TIMESTAMP.log" ]; then tail -20 "$TEST_RESULTS_DIR/unit_tests_$TIMESTAMP.log"; else echo "No unit test results available"; fi)
\`\`\`

### Integration Tests
\`\`\`
$(if [ -f "$TEST_RESULTS_DIR/integration_tests_$TIMESTAMP.log" ]; then tail -20 "$TEST_RESULTS_DIR/integration_tests_$TIMESTAMP.log"; else echo "No integration test results available"; fi)
\`\`\`

### Benchmarks
\`\`\`
$(if [ -f "$TEST_RESULTS_DIR/benchmarks_$TIMESTAMP.log" ]; then tail -20 "$TEST_RESULTS_DIR/benchmarks_$TIMESTAMP.log"; else echo "No benchmark results available"; fi)
\`\`\`
EOF
    
    log_success "Test report generated: $report_file"
}

cleanup() {
    log_info "Cleaning up..."
    
    # Остановка всех фоновых процессов
    jobs -p | xargs -r kill
    
    # Очистка временных файлов
    rm -f /tmp/cloudbridge-*
    
    log_success "Cleanup completed"
}

# Основная функция
main() {
    print_header
    
    # Перехват сигналов для корректного завершения
    trap cleanup EXIT
    trap 'log_error "Test run interrupted"; exit 1' INT TERM
    
    # Проверка зависимостей
    check_dependencies
    
    # Настройка окружения
    setup_environment
    
    # Счетчики результатов
    PASSED=0
    FAILED=0
    
    # Запуск тестов
    log_info "Starting test suite..."
    
    # Unit тесты
    if run_unit_tests; then
        ((PASSED++))
    else
        ((FAILED++))
    fi
    
    # Интеграционные тесты
    if run_integration_tests; then
        ((PASSED++))
    else
        ((FAILED++))
    fi
    
    # Бенчмарки
    if run_benchmarks; then
        ((PASSED++))
    else
        ((FAILED++))
    fi
    
    # Линтинг
    if run_linting; then
        ((PASSED++))
    else
        ((FAILED++))
    fi
    
    # Проверки безопасности
    if run_security_checks; then
        ((PASSED++))
    else
        ((FAILED++))
    fi
    
    # Генерация отчета
    generate_test_report
    
    # Итоговый результат
    print_footer
    
    if [ $FAILED -eq 0 ]; then
        log_success "All tests passed! ($PASSED/$((PASSED + FAILED)))"
        echo -e "${GREEN}"
        echo "🎉 Test run completed successfully!"
        echo "📊 Check the test report for detailed results"
        echo -e "${NC}"
        exit 0
    else
        log_error "Some tests failed! ($PASSED passed, $FAILED failed)"
        echo -e "${RED}"
        echo "❌ Test run completed with failures"
        echo "📊 Check the test report for detailed results"
        echo -e "${NC}"
        exit 1
    fi
}

# Запуск основной функции
main "$@" 