#!/bin/bash

# Quick Start Script for WFRP Bot Docker
# Быстрый запуск бота в Docker контейнере

set -e

# Цвета для вывода
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}WFRP Bot Quick Start${NC}"
echo -e "${BLUE}================================${NC}"
echo ""

# Шаг 1: Проверка .env файла
if [ ! -f .env ]; then
    echo -e "${YELLOW}Шаг 1/3: Настройка конфигурации${NC}"
    echo "Файл .env не найден. Создаем из .env.minimax.example..."
    cp .env.minimax.example .env
    echo -e "${GREEN}✓${NC} .env файл создан"
    echo ""
    echo "Пожалуйста, отредактируйте .env файл и заполните:"
    echo "  1. TELEGRAM_BOT_TOKEN"
    echo "  2. TELEGRAM_GROUP_ID"
    echo "  3. MINIMAX_API_KEY"
    echo ""
    read -p "Нажмите Enter после редактирования .env файла..."
    echo ""
fi

# Шаг 2: Проверка значений в .env
if grep -q "your_bot_token_here" .env; then
    echo -e "${RED}Ошибка: TELEGRAM_BOT_TOKEN не заполнен${NC}"
    echo "Пожалуйста, отредактируйте .env файл"
    exit 1
fi

if grep -q "your_minimax_api_key_here" .env; then
    echo -e "${RED}Ошибка: MINIMAX_API_KEY не заполнен${NC}"
    echo "Пожалуйста, отредактируйте .env файл"
    exit 1
fi

echo -e "${GREEN}Шаг 2/3: Проверка конфигурации${NC}"
echo -e "${GREEN}✓${NC} .env файл настроен корректно"
echo ""

# Шаг 3: Билд и запуск
echo -e "${GREEN}Шаг 3/3: Запуск контейнера${NC}"
echo "Билд контейнера..."
docker-compose up -d --build

echo ""
sleep 5

# Проверка статуса
if docker-compose ps | grep -q "Up"; then
    echo -e "${GREEN}✓${NC} Контейнер успешно запущен"
    echo ""
    echo "========================================"
    echo -e "${BLUE}Всё готово!${NC}"
    echo "========================================"
    echo ""
    echo -e "${GREEN}Доступные команды:${NC}"
    echo "  docker-compose logs -f      # Просмотр логов"
    echo "  docker-compose ps           # Статус контейнера"
    echo "  docker-compose stop         # Остановка"
    echo "  docker-compose restart      # Перезапуск"
    echo ""
    echo -e "${YELLOW}Следующие шаги:${NC}"
    echo "  1. Проверьте логи: docker-compose logs -f"
    echo "  2. Проверьте статус: docker-compose ps"
    echo "  3. Откройте Telegram и проверьте бота"
    echo ""
    echo "Для подробного тестирования запустите: ./test-docker.sh"
else
    echo -e "${RED}Ошибка: Контейнер не запустился${NC}"
    echo "Проверьте логи: docker-compose logs"
    exit 1
fi
