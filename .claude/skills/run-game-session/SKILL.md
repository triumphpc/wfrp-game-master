---
name: run-game-session  
description: Проведение игровой сессии с тремя агентами
---

# Проведение игровой сессии

## Трех-агентная модель:
GM (`.claude/agents/GM.md`) ←→ Checker (`.claude/agents/CHECKER.md`) ←→ Logger (`.claude/agents/LOGGER.md`)

##  При боях
GM (`.claude/agents/GM.md`) ←→ Checker-battle (`.claude/agents/CHECKER-battle.md`) ←→ Logger (`.claude/agents/LOGGER-battle.md`)

##  При социальном взаимодействии 
GM-social (`.claude/agents/GM-social.md`) ←→ Checker (`.claude/agents/CHECKER.md`) ←→ Logger (`.claude/agents/LOGGER.md`)

## При 
## Phase 1: Подготовка сессии
1. GM загружает контекст предыдущей сессии
2. Checker проверяет актуальность данных персонажей
3. Logger создает новый файл сессии

## Phase 2: Цикл игры (основной loop)
**Для каждого действия:**
1. **GM** описывает ситуацию и предлагает варианты
2. **Player** выбирает действие
3. **Checker** проверяет корректность (правила, навыки, снаряжение)
4. **GM** объявляет проверку с учетом замечаний Checker
5. **Player** бросает кубики
6. **GM** описывает результат
7. **Logger** записывает всё в историю и обновляет карточки

## Phase 3: Перерывы и паузы
При паузе:
1. Logger сохраняет текущее состояние
2. Checker фиксирует промежуточные результаты
3. GM делает заметки для продолжения

## Phase 4: Завершение сессии
1. Награждение опытом (Checker проверяет справедливость)
2. Обновление карточек (Logger фиксирует изменения)
3. GM готовит хук для следующей сессии