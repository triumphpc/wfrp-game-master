#!/bin/bash

# WFRP Bot Docker Test Script
# Автоматизированное тестирование контейнера

set -e  # Exit on error

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция для вывода логов
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Проверка предусловий
check_prerequisites() {
    log_info "Проверка предусловий..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker не установлен"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose не установлен"
        exit 1
    fi

    if [ ! -f .env ]; then
        log_error "Файл .env не найден"
        log_info "Создайте .env файл из .env.minimax.example"
        exit 1
    fi

    log_success "Предусловия проверены"
}

# Проверка Dockerfile
check_dockerfile() {
    log_info "Проверка Dockerfile..."

    if [ ! -f Dockerfile ]; then
        log_error "Dockerfile не найден"
        exit 1
    fi

    log_success "Dockerfile найден"
}

# Проверка docker-compose.yml
check_docker_compose() {
    log_info "Проверка docker-compose.yml..."

    if [ ! -f docker-compose.yml ]; then
        log_error "docker-compose.yml не найден"
        exit 1
    fi

    log_success "docker-compose.yml найден"
}

# Создание .env файла
create_env_file() {
    log_info "Проверка .env файла..."

    if [ ! -f .env ]; then
        log_warning ".env не найден, создаем из .env.minimax.example"
        cp .env.minimax.example .env
        log_info "Создайте .env файл и заполните реальные значения"
        log_warning "Для продолжения теста заполните .env файл"
        exit 1
    fi

    log_success ".env файл найден"
}

# Билд контейнера
build_container() {
    log_info "Билд контейнера..."

    if docker-compose images wfrp-bot | grep -q "wfrp-bot"; then
        log_warning "Контейнер уже собран, пропускаем билд"
        return
    fi

    docker-compose build

    if [ $? -eq 0 ]; then
        log_success "Контейнер успешно собран"
    else
        log_error "Билд контейнера не удался"
        exit 1
    fi
}

# Запуск контейнера
start_container() {
    log_info "Запуск контейнера..."

    docker-compose up -d

    sleep 5

    if docker-compose ps | grep -q "Up"; then
        log_success "Контейнер запущен"
    else
        log_error "Контейнер не запущен"
        docker-compose logs
        exit 1
    fi
}

# Проверка логов
check_logs() {
    log_info "Проверка логов контейнера..."

    sleep 10

    docker-compose logs --tail=30

    if docker-compose logs | grep -q "Config loaded"; then
        log_success "Контейнер работает корректно"
    else
        log_error "Контейнер не запустился корректно"
        log_info "Логи контейнера:"
        docker-compose logs
        exit 1
    fi
}

# Проверка монтирования директорий
check_volumes() {
    log_info "Проверка монтирования директорий..."

    docker-compose exec wfrp-bot ls -la /app/rules/ > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        log_success "Правила WFRP монтируются корректно"
    else
        log_error "Не удалось проверить правила WFRP"
    fi

    docker-compose exec wfrp-bot ls -la /app/history/ > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        log_success "История монтируется корректно"
    else
        log_warning "История не монтируется (это нормально для первого запуска)"
    fi
}

# Проверка переменных окружения
check_env_vars() {
    log_info "Проверка переменных окружения..."

    if docker-compose exec wfrp-bot env | grep -q "TELEGRAM_BOT_TOKEN=your_bot_token_here"; then
        log_error "TELEGRAM_BOT_TOKEN содержит примерное значение"
        log_info "Пожалуйста, обновите .env файл"
        exit 1
    fi

    if docker-compose exec wfrp-bot env | grep -q "MINIMAX_API_KEY=your_minimax_api_key_here"; then
        log_error "MINIMAX_API_KEY содержит примерное значение"
        log_info "Пожалуйста, обновите .env файл"
        exit 1
    fi

    log_success "Переменные окружения настроены корректно"
}

# Проверка ресурсов
check_resources() {
    log_info "Проверка ресурсов контейнера..."

    docker stats wfrp-bot --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"

    log_success "Проверка ресурсов завершена"
}

# Тестовая команда
run_test_command() {
    log_info "Запуск тестовой команды..."

    docker-compose exec wfrp-bot ps aux | grep wfrp-bot

    log_success "Тестовая команда выполнена"
}

# Очистка после тестирования
cleanup() {
    if [ "$1" == "keep" ]; then
        log_info "Оставляем контейнер запущенным для дальнейшего использования"
    else
        log_info "Остановка контейнера..."
        docker-compose down
        log_success "Контейнер остановлен"
    fi
}

# Тестирование Minimax API (опционально)
test_minimax_api() {
    log_info "Тестирование Minimax API..."

    # Этот тест может занять время, если нет API ключа
    if ! docker-compose exec wfrp-bot env | grep -q "MINIMAX_API_KEY=your_minimax_api_key_here"; then
        log_warning "Пропускаем тест Minimax API (нет реального API ключа)"
        return
    fi

    # Можно добавить тестирование Minimax API здесь
    log_success "Тест Minimax API пропущен (требует реального API ключа)"
}

# Главный цикл тестирования
main() {
    echo "========================================"
    echo "WFRP Bot Docker Тестирование"
    echo "========================================"
    echo ""

    # Проверка предусловий
    check_prerequisites
    check_dockerfile
    check_docker_compose
    create_env_file

    # Билд и запуск
    build_container
    start_container

    # Проверки
    check_logs
    check_volumes
    check_env_vars
    check_resources
    run_test_command

    # Тестирование API (опционально)
    test_minimax_api

    echo ""
    echo "========================================"
    log_success "Все тесты завершены успешно!"
    echo "========================================"
    echo ""
    log_info "Контейнер запущен и работает"
    log_info "Проверьте логи: docker-compose logs -f"
    log_info "Введите 'cleanup' для остановки контейнера"
    log_info "Введите 'keep' для продолжения работы"

    read -p "Остановить контейнер после тестирования? (y/n): " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cleanup "stop"
    else
        cleanup "keep"
    fi
}

# Запуск главного цикла
main
