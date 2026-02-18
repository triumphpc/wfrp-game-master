---

description: "Task list for 002-character-creation feature"
---

# Tasks: Улучшение создания персонажей WFRP

**Input**: Design documents from `/specs/002-character-creation/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Не требуются - ручное тестирование через Telegram

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Foundational (Blocking Prerequisites)

**Purpose**: Infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T001 Добавить константы русских характеристик в game/character_creation.go
- [X] T002 [P] Создать функцию маппинга WS→ББ, BS→ДБ и т.д. в game/character_creation.go
- [X] T003 Создать функцию детекта вопроса к LLM в процессе создания в game/character_creation.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 2: User Story 1 - Создание персонажа с именем (Priority: P1) 🎯 MVP

**Goal**: Команда /character с именем начинает создание, без имени - показывает справку

**Independent Test**: Отправить /character - должна показаться справка; отправить /character Тест - начало диалога

### Implementation for User Story 1

- [X] T004 [P] [US1] Добавить вывод справки в telegram/handlers.go при /character без аргументов
- [X] T005 [US1] Интегрировать существующий CharacterCreator в обработчик /character в telegram/handlers.go
- [X] T006 [US1] Добавить проверку уникальности имени персонажа в game/character_creation.go

**Checkpoint**: /character работает со справкой

---

## Phase 3: User Story 2 - Запрос пояснений у LLM (Priority: P1)

**Goal**: На любом этапе создания персонажа можно задать вопрос к LLM

**Independent Test**: В процессе создания спросить "как распределить характеристики" - получить ответ от LLM

### Implementation for User Story 2

- [X] T007 [P] [US2] Интегрировать LLM провайдер в CharacterCreator в game/character_creation.go
- [X] T008 [US2] Добавить обработку вопросов к LLM в ProcessInput() в game/character_creation.go
- [X] T009 [US2] Добавить промт для объяснения правил WFRP в game/character_creation.go
- [X] T010 [US2] Обработать ошибки LLM с понятным сообщением в telegram/handlers.go

**Checkpoint**: Вопросы к LLM работают на всех этапах

---

## Phase 4: User Story 3 - Автогенерация имени (Priority: P2)

**Goal**: Команда "сгенери имя" генерирует имя персонажа через LLM

**Independent Test**: На этапе имени написать "сгенери имя" - получить сгенерированное имя

### Implementation for User Story 3

- [X] T011 [P] [US3] Добавить обработку "сгенери имя" / "сгенери сам" в ProcessInput() в game/character_creation.go
- [X] T012 [US3] Создать промт для генерации имени в game/character_creation.go

**Checkpoint**: Автогенерация имени работает

---

## Phase 5: User Story 4 + User Story 5 - Карточка персонажа с русскими характеристиками (Priority: P1)

**Goal**: Полная карточка персонажа с характеристиками ББ, ДБ, СС, И, Л, О, СТ, К

**Independent Test**: Завершить создание персонажа - проверить что все 8 характеристик на русском

### Implementation for User Story 4-5

- [X] T013 [P] [US4] Обновить GenerateCharacterMarkdown() для вывода русских характеристик в game/character_creation.go
- [X] T014 [US4] Обновить generateReview() для вывода русских характеристик в game/character_creation.go
- [X] T015 [US4] Обновить getStatsSummary() для вывода русских характеристик в game/character_creation.go

**Checkpoint**: Карточка персонажа содержит все характеристики на русском

---

## Phase 6: User Story 6 - Список всех персонажей (Priority: P2)

**Goal**: Команда /characters выводит список всех персонажей кампании

**Independent Test**: Отправить /characters - получить список персонажей

### Implementation for User Story 6

- [ ] T016 [P] [US6] Добавить команду /characters в telegram/handlers.go
- [ ] T017 [US6] Реализовать вывод списка персонажей с именем и профессией в telegram/handlers.go
- [ ] T018 [US6] Добавить обработку случая "нет персонажей" в telegram/handlers.go

**Checkpoint**: /characters выводит корректный список

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T019 [P] Обновить справку в /help с новыми командами в telegram/handlers.go
- [X] T020 Протестировать полный флоу создания персонажа через Telegram
- [X] T021 Проверить edge cases (дубликат имени, отмена, LLM недоступен)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 1)**: No dependencies - can start immediately
- **User Stories (Phase 2-6)**: All depend on Foundational phase completion
  - US1 (P1) → US2 (P1) → US4-5 (P1) → US3 (P2) → US6 (P2)
  - US3 и US6 могут развиваться параллельно с US2 после Phase 1
- **Polish (Final Phase)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational - No dependencies on other stories
- **User Story 2 (P1)**: Depends on Foundational - требует LLM интеграцию
- **User Story 3 (P2)**: Depends on Foundational + US2 - использует LLM
- **User Story 4-5 (P1)**: Depends on Foundational - требует только маппинг характеристик
- **User Story 6 (P2)**: Can start after Foundational - независим от других stories

### Within Each User Story

- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- T001, T002, T003 - Foundational phase - могут выполняться параллельно
- T004, T005 - US1 - могут выполняться параллельно (разные файлы)
- T007, T008 - US2 - могут выполняться параллельно
- T013, T014, T015 - US4-5 - могут выполняться параллельно
- T016, T017, T018 - US6 - могут выполняться параллельно

---

## Parallel Example: Foundational Phase

```bash
# Запустить все Foundational задачи параллельно:
Task: "Добавить константы русских характеристик в game/character_creation.go"
Task: "Создать функцию маппинга WS→ББ, BS→ДБ и т.д. в game/character_creation.go"  
Task: "Создать функцию детекта вопроса к LLM в процессе создания в game/character_creation.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 + US4-5)

1. Complete Phase 1: Foundational
2. Complete Phase 2: User Story 1
3. Complete Phase 5: User Story 4-5 (карточка с русскими характеристиками)
4. **STOP and VALIDATE**: Базовое создание персонажа работает
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Foundational → Foundation ready
2. Add User Story 1 → Test independently → /character работает со справкой
3. Add User Story 2 → Test independently → Вопросы к LLM работают
4. Add User Story 3 → Test independently → Автогенерация имени
5. Add User Story 4-5 → Test independently → Карточка на русском
6. Add User Story 6 → Test independently → /characters работает
7. Polish → Тестирование edge cases

### Single Developer Strategy

1. Foundational → User Story 1 → User Story 2 → User Story 4-5 → User Story 3 → User Story 6 → Polish

---

## Notes

- [P] tasks = разные файлы, нет зависимостей
- [Story] label связывает задачу с конкретным user story
- Каждый user story должен быть независимо завершаемым и тестируемым
- Проект уже существует - Setup phase не требуется
- Тестирование через Telegram бот
- Go 1.21 + telegram-bot-api + go-openai
