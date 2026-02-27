# Implementation Plan: WFRP Game Master Bot

**Branch**: `001-wfrp` | **Date**: 2025-02-15 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-wfrp/spec.md`

## Summary

Разработка Telegram бота для ведения игр WFRP 4E с интеграцией провайдеров LLM. Бот充当 агента GM (Master) — взаимодействует с игроками в группе @WHFR4, загружает контекст из карточек персонажей и истории сессий, формирует запросы к LLM с промтами, и обновляет состояние игры. Бот написан на Go с использованием библиотеки telebot, поддерживает несколько провайдеров LLM с hot-reload, конфигурацию через переменные окружения, и проверку правил через RAG-MCP-Server.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: github.com/Varbyt/go-telegram-bot-api/v5 (telebot), OpenAI SDK (для LLM провайдеров), RAG-MCP-Server integration
**Storage**: Markdown файлы (history/[campaign]/), конфигурационные файлы
**Testing**: Нет автоматических тестов (ручное тестирование через Telegram)
**Target Platform**: Linux server (VPS/облако)
**Project Type**: single (Go приложение для Telegram бота)
**Performance Goals**: Ответ на команды в течение 3 секунд, проверка вводных данных каждую секунду, обновление карточек в течение 30 секунд
**Constraints**: Длинные сообщения (>4096 символов) отправляются частями (до 4096 символов), лимит Telegram API (30 запросов/секунда)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Все принципы из `~/.claude/.specify/memory/constitution.md` (v3.0.0 WFRP Game Master) соблюдены:
- ✅ Context-First Development — чтение карточек, правил, истории
- ✅ Single Source of Truth — карточки в `history/[campaign]/characters/`, правила в `rules/dict/`
- ✅ Russian Language Priority — все интерфейсы на русском
- ✅ Atomic Session Execution — один файл сессии, формат YYYY-MM-DD_HH-MM_description.md
- ✅ Quality Gates — проверка состояния перед сохранением
- ✅ Error Handling — ошибки провайдеров информируются администратору

**Результат**: Проверка пройдена. Проект соответствует конституции WFRP Game Master v3.0.0.

## Project Structure

### Documentation (this feature)

```text
specs/001-wfrp/
├── plan.md              # Этот файл (/speckit.plan command output)
├── spec.md              # Feature specification
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
└── checklists/
    └── requirements.md    # Quality checklist
```

### Source Code (repository root)

```text
bot/                    # Основной код бота на Go
├── config/               # Загрузка конфигурации (env файлы)
├── llm/                  # Интеграция с провайдерами LLM
│   ├── provider.go      # Интерфейс провайдера (openai, z.ai, minimax)
│   ├── openai.go       # Реализация для OpenAI-compatible
│   ├── zai.go           # Реализация для z.ai
│   └── minimax.go      # Реализация для minimax
├── telegram/             # Интеграция с Telegram
│   ├── handlers.go       # Обработчики команд (/start, /help и т.д.)
│   ├── middleware.go     # Middleware (rate limiting, logging)
│   ├── streaming.go      # Отправка длинных сообщений частями
│   └── game/            # Игровая логика
│       ├── session.go        # Управление игровой сессией
│       ├── character.go      # Работа с карточками персонажей
│       ├── context.go        # Загрузка контекста (персонажи, правила)
│       └── rag.go            # Проверка правил через RAG-MCP-Server
├── storage/              # Файловое хранилище
│   ├── markdown.go        # Чтение/запись Markdown файлов
│   ├── campaign.go       # Управление кампаниями
│   └── history.go         # Работа с историей сессий

config/                  # Конфигурационные файлы
.env.example              # Пример переменных окружения
```

**Structure Decision**: Выбрана структура Go приложения с разделением на модули (config, llm, telegram, game, storage). Все компоненты организованы для независимого тестирования и масштабирования.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Нарушение | Почему нужно | Альтернатива отклонена |
|-----------|-------------------|---------------------|
| N/A       | N/A          | N/A                                  |

Нарушений конституции нет. Проект соответствует всем принципам WFRP Game Master Constitution v3.0.0.
