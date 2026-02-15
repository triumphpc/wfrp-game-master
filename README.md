# WFRP Game Master Bot

Telegram бот для ведения игр Warhammer Fantasy Roleplay 4th Edition с использованием LLM.

## Возможности

- **Интеграция с Telegram Bot API**: Обработка команд и сообщений в группе
- **Мульти-провайдер LLM**: Поддержка OpenAI, z.ai (Claude), Minimax и custom провайдеров
- **Горячая перезагрузка конфигурации**: Изменение настроек без перезапуска бота
- **Управление кампаниями**: Создание, загрузка и управление кампаниями
- **Система карточек персонажей**: Загрузка, сохранение и валидация по правилам WFRP
- **История сессий**: Автоматическое создание и хранение логов сессий
- **Проверка правил**: Интеграция с RAG-MCP-Server для проверки правил WFRP

## Требования

- Go 1.21 или выше
- Telegram Bot API токен
- API ключ для выбранного LLM провайдера
- Доступ к RAG-MCP-Server (опционально)

## Установка

### 1. Клонирование репозитория

```bash
git clone <repository-url>
cd wfrp-game-master
```

### 2. Установка зависимостей

```bash
go mod download
go mod tidy
```

### 3. Настройка конфигурации

Скопируйте пример конфигурационного файла:

```bash
cp .env.example .env
```

Отредактируйте `.env` с вашими значениями:

```env
# Токен бота из @BotFather
TELEGRAM_BOT_TOKEN=your_bot_token_here

# ID вашей группы
TELEGRAM_GROUP_ID=-1001234567890

# Провайдер LLM (openai, zai, minimax, custom)
DEFAULT_PROVIDER=openai

# API ключ выбранного провайдера
OPENAI_API_KEY=your_openai_api_key_here
```

### 4. Сборка

```bash
go build -o wfrp-bot
```

### 5. Запуск

```bash
./wfrp-bot
```

Для запуска в фоновом режиме (Linux):

```bash
nohup ./wfrp-bot > bot.log 2>&1 &
```

Для systemd:

```ini
[Unit]
Description=WFRP Game Master Bot
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/wfrp-game-master
ExecStart=/path/to/wfrp-game-master/wfrp-bot
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## Команды бота

| Команда | Описание |
|---------|------------|
| `/start` | Начать новую игру или сессию |
| `/help` | Показать справку по командам |
| `/campaign [name]` | Выбрать кампанию |
| `/character [name]` | Показать карточку персонажа |
| `/status` | Показать статус текущей сессии |
| `/reload` | Перезагрузить конфигурацию |
| `/stop` | Остановить текущую сессию |

## Структура проекта

```
.
├── main.go              # Точка входа
├── config/               # Конфигурация
│   └── config.go
├── llm/                  # LLM провайдеры
│   ├── provider.go       # Интерфейс провайдера
│   ├── openai.go        # OpenAI-совместимый провайдер
│   ├── zai.go           # z.ai (Claude) провайдер
│   └── minimax.go      # Minimax провайдер
├── telegram/             # Telegram Bot API
│   ├── bot.go           # Основной код бота
│   ├── middleware.go     # Rate limiting, логирование
│   └── streaming.go      # Отправка длинных сообщений
├── game/                 # Игровая логика
│   ├── session.go       # Управление сессиями
│   ├── character.go     # Карточки персонажей
│   ├── context.go       # Загрузка контекста
│   └── rag.go           # Проверка правил
├── storage/              # Файловое хранилище
│   ├── markdown.go       # Парсинг Markdown
│   ├── campaign.go      # Управление кампаниями
│   └── history.go        # История сессий
└── .env.example         # Пример конфигурации
```

## Работа с кампаниями

### Создание новой кампании

```bash
# Через бота в Telegram
/newcampaign <имя>

# Или вручную создать директорию
mkdir -p history/<кампания>/characters
mkdir -p history/<кампания>/sessions
```

### Структура кампании

```
history/<кампания>/
├── characters/              # Карточки персонажей (текущее состояние)
│   ├── player1.md
│   ├── player2.md
│   └── ...
├── sessions/                # История сессий (хронологически)
│   ├── 2024-02-15_10-00_first_session.md
│   ├── 2024-02-15_14-30_combat.md
│   └── ...
├── party_summary.md        # Сводка по группе
└── campaign.md            # Метаданные кампании
```

## Формат карточек персонажей

Карточки персонажей должны соответствовать правилам WFRP 4e:

```markdown
# Имя: Имя Персонажа

## Характеристики
- В: 40
- С: 30
- Лов: 35
- Инт: 40
- ВН: 30
- Об: 35

## Навыки
- Стрельба: 40
- Ближний бой: 30
- ...

## Экипировка
- Меч длинный: 1
- Латы: 1
- ...

## История
- [2024-02-15] Получил уровень в карьере Милитари
```

## Конфигурация провайдеров LLM

### OpenAI

```env
DEFAULT_PROVIDER=openai
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-4o
```

### z.ai (Claude)

```env
DEFAULT_PROVIDER=zai
ZAI_API_KEY=sk-ant-...
ZAI_BASE_URL=https://api.z.ai/v1
ZAI_MODEL=claude-3-5-sonnet-20240228
```

### Minimax

```env
DEFAULT_PROVIDER=minimax
MINIMAX_API_KEY=your_key
MINIMAX_BASE_URL=https://api.minimax.chat/v1
MINIMAX_MODEL=minimax-text
```

### Custom (OpenAI-compatible)

```env
DEFAULT_PROVIDER=custom
CUSTOM_API_KEY=your_key
CUSTOM_BASE_URL=https://your-api.com/v1
CUSTOM_MODEL=your-model
```

## Получение API ключей

### Telegram Bot Token

1. Найдите @BotFather в Telegram
2. Напишите `/newbot` и следуйте инструкциям
3. Скопируйте полученный токен в `TELEGRAM_BOT_TOKEN`

### Group Chat ID

1. Найдите @userinfobot в Telegram
2. Перешлите бота в вашу группу
3. Отправьте любое сообщение группе
4. Ответьте на сообщение @userinfobot
5. Скопируйте `id` в `TELEGRAM_GROUP_ID`

### LLM API Keys

Получите ключи на соответствующих платформах:
- OpenAI: https://platform.openai.com/api-keys
- z.ai: https://console.z.ai/
- Minimax: https://api.minimax.chat/

## Режимы работы

### Режим мастера игры (GM)

Бот может работать в режиме мастера игры, где:
- Описывает сцены и ситуации
- Обрабатывает действия игроков
- Формирует запросы к LLM с контекстом
- Обновляет карточки персонажей по правилам
- Проверяет правила через RAG-MCP-Server

### Режим ассистента

Бот может также работать в режиме ассистента:
- Помощь в создании кампаний
- Быстрый поиск правил WFRP
- Генерация NPC и сюжетных идей

## Траблшутинг

### Бот не отвечает

1. Проверьте логи: `tail -f bot.log`
2. Убедитесь, что API ключи валидные
3. Проверьте подключение к интернету

### Проблемы с Telegram

1. Убедитесь, что бот добавлен в группу
2. Проверьте `TELEGRAM_GROUP_ID`
3. Убедитесь, что бот имеет права на отправку сообщений

### Ошибки LLM

1. Проверьте квоты и лимиты API
2. Попробуйте переключиться на другой провайдер
3. Проверьте формат запроса

## Разработка

### Сборка проекта

```bash
go build -o wfrp-bot
```

### Запуск тестов

```bash
go test ./...
```

### Форматирование кода

```bash
go fmt ./...
go vet ./...
```

## Логирование

Логи пишутся в stdout/stderr. Для записи в файл:

```bash
./wfrp-bot 2>&1 | tee bot.log
```

## Безопасность

- **НИКОГДА** не коммитьте файл `.env` с реальными API ключами
- Используйте разные `.env.*` файлы для разных окружений
- Ограничьте доступ к боту только авторизованным пользователям
- Регулярно обновляйте зависимости для безопасности

## Лицензия

MIT License - см. файл LICENSE в репозитории

## Поддержка

Для отчёта о багах или вопросов:
- Создайте issue на GitHub
- Или свяжитесь с администратором проекта

## Версия

Версия: 1.0.0
Дата: 2025-02-15
