# Implementation Plan: Улучшение создания персонажей WFRP

**Branch**: `002-character-creation` | **Date**: 2026-02-17 | **Spec**: specs/002-character-creation/spec.md

**Input**: Feature specification from `/specs/002-character-creation/spec.md`

## Summary

Улучшение процесса создания персонажей в Telegram-боте для WFRP 4E: добавление справки по команде /character, возможность задавать вопросы LLM на любом этапе, автогенерация имени, полная карточка персонажа с русскими характеристиками (ББ, ДБ, СС, И, Л, О, СТ, К), команда списка персонажей /characters.

## Technical Context

**Language/Version**: Go 1.21  
**Primary Dependencies**: github.com/go-telegram-bot-api/telegram-bot-api/v5, github.com/sashabaranov/go-openai  
**Storage**: Файловая система (markdown файлы в characters/)  
**Testing**: Ручное тестирование через Telegram  
**Target Platform**: Linux server (Telegram bot)  
**Project Type**: CLI Telegram bot  
**Performance Goals**: <100ms время отклика на команды  
**Constraints**: Офлайн режим без LLM - базовые ответы  
**Scale**: Несколько десятков персонажей на кампанию

## Constitution Check

*GATE: Constitution file not found in project - skipping check*

## Project Structure

### Documentation (this feature)

```text
specs/002-character-creation/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── checklists/
    └── requirements.md   # Quality checklist
```

### Source Code (repository root)

```text
wfrp-bot (Go 1.21)
├── main.go                    # Entry point
├── telegram/
│   ├── bot.go                 # Bot initialization
│   ├── handlers.go            # Command handlers
│   └── middleware.go          # Middleware
├── game/
│   ├── character.go           # Character management (существующий)
│   ├── character_creation.go  # Character creation workflow (существующий)
│   └── session_manager.go     # Session management
├── llm/
│   ├── provider.go            # LLM interface
│   ├── openai.go              # OpenAI provider
│   └── minimax.go             # MiniMax provider
└── storage/
    └── campaign.go            # Campaign storage
```

**Structure Decision**: Go Telegram bot - существующая структура проекта сохранена. Добавление функциональности в существующие файлы character_creation.go и handlers.go.

## Phase 0: Research

### Research Tasks

- [ ] Изучить существующую структуру команды /character в handlers.go
- [ ] Определить формат хранения персонажей (markdown)
- [ ] Проверить интеграцию с LLM провайдерами

### Unknowns to Resolve

1. Как именно реализовать "вопрос к LLM" - через отдельный промт или контекстное?
2. Формат карточки персонажа - текущий GenerateCharacterMarkdown() нужно адаптировать под русские ББ/ДБ
3. Как хранить состояние между шагами создания персонажа

## Phase 1: Design

### Data Model Changes

**Новая/изменённая сущность**: Character (расширение)

| Поле | Тип | Описание |
|------|-----|----------|
| Name | string | Имя персонажа |
| RussianStats | map[string]int | ББ, ДБ, СС, И, Л, О, СТ, К |
| OriginalStats | map[string]int | WS, BS, S, I, Ag, WP, Fel, T |

### API Contracts

Команды Telegram:

| Команда | Описание | Выход |
|---------|----------|-------|
| /character | Начать создание или показать справку | Текст + кнопки |
| /characters | Список всех персонажей | Список |
| "сгенери имя" | Сгенерировать имя через LLM | Текст |
| [вопрос] | Вопрос о правилах WFRP | Ответ LLM |

### Quickstart

1. Обновить handlers.go - добавить /characters команду
2. Обновить character_creation.go - добавить русские характеристики
3. Добавить LLM вопросы в процесс создания
4. Протестировать через Telegram
