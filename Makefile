# Makefile for WFRP Bot Docker
.PHONY: help build up down logs restart test clean env-check

# Default target
.DEFAULT_GOAL := help

# Colors
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m

help: ## Показать все доступные команды
	@echo "$(BLUE)WFRP Bot Docker Makefile$(NC)"
	@echo ""
	@echo "$(GREEN)Доступные команды:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'

build: ## Собрать Docker образ
	@echo "$(BLUE)Сборка Docker образа...$(NC)"
	docker-compose build

up: ## Запустить контейнер
	@echo "$(BLUE)Запуск контейнера...$(NC)"
	docker-compose up -d

down: ## Остановить контейнер
	@echo "$(BLUE)Остановка контейнера...$(NC)"
	docker-compose down

restart: ## Перезапустить контейнер
	@echo "$(BLUE)Перезапуск контейнера...$(NC)"
	docker-compose restart

logs: ## Показать логи
	docker-compose logs -f

logs-tail: ## Показать последние логи
	docker-compose logs --tail=100

status: ## Статус контейнера
	docker-compose ps

exec: ## Войти в контейнер (введите команду для выполнения)
	docker-compose exec wfrp-bot sh

exec-shell: ## Открыть shell в контейнере
	docker-compose exec wfrp-bot sh

test: ## Запустить тестирование контейнера
	./test-docker.sh

quick-start: ## Быстрый старт (создание .env и запуск)
	./quick-start.sh

clean: ## Очистить все (контейнер и данные)
	@echo "$(RED)Очистка...$(NC)"
	docker-compose down -v
	docker-compose down
	@echo "$(GREEN)Очистка завершена$(NC)"

clean-images: ## Удалить неиспользуемые образы
	@echo "$(BLUE)Удаление неиспользуемых образов...$(NC)"
	docker image prune -a

clean-all: clean clean-images ## Полная очистка
	@echo "$(GREEN)Полная очистка завершена$(NC)"

env-check: ## Проверить .env файл
	@if [ ! -f .env ]; then \
		echo "$(RED)Файл .env не найден$(NC)"; \
		echo "$(YELLOW)Создайте .env из .env.minimax.example$(NC)"; \
		exit 1; \
	else \
		echo "$(GREEN)Файл .env найден$(NC)"; \
		echo "" ; \
		echo "$(BLUE)Переменные окружения:$(NC)"; \
		docker-compose exec wfrp-bot env | grep TELEGRAM; \
	fi

check: env-check build test ## Проверить все (env + build + test)

migrate-data: ## Миграция данных (если нужно)
	@echo "$(BLUE)Миграция данных...$(NC)"
	# Добавьте здесь логику миграции данных
	@echo "$(GREEN)Миграция данных завершена$(NC)"

backup: ## Создать бэкап данных
	@echo "$(BLUE)Создание бэкапа данных...$(NC)"
	tar -czf backup-$(shell date +%Y%m%d-%H%M%S).tar.gz history/ characters/
	@echo "$(GREEN)Бэкап создан: backup-$(shell date +%Y%m%d-%H%M%S).tar.gz$(NC)"

backup-logs: ## Создать бэкап логов
	@echo "$(BLUE)Создание бэкапа логов...$(NC)"
	mkdir -p logs
	docker-compose logs > logs/logs-$(shell date +%Y%m%d-%H%M%S).txt
	@echo "$(GREEN)Логи сохранены в logs/$(NC)"

dev: ## Запуск в development режиме
	@echo "$(BLUE)Запуск в development режиме...$(NC)"
	docker-compose up --build

prod: up logs ## Запуск в production режиме

stats: ## Показать статистику ресурсов
	docker stats wfrp-bot --no-stream

health: ## Проверить здоровье контейнера
	@echo "$(BLUE)Проверка здоровья...$(NC)"
	docker-compose exec wfrp-bot pgrep -f wfrp-bot
