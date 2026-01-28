---
name: create-new-campaign
description: Создание новой кампании с нуля
version: 1.0.0
---

# Создание новой кампании WFRP 4E

## Phase 1: Настройка кампании
1. Спросить название кампании
2. Создать директорию: `history/[campaign_name]/`
3. Создать поддиректории: `characters/`, `sessions/`

## Phase 2: Создание персонажей
1. Запустить агента GM: `Task.run("gm-agent", "начать создание персонажей")`
2. Для каждого игрока:
    - Запустить `.claude/skills/create-character`
    - Сохранить в `history/[campaign_name]/characters/`

## Phase 3: Настройка начальной сцены
1. Спросить у GM начальную ситуацию
2. Создать файл первой сессии: `sessions/001_introduction.md`

## Phase 4: Запуск игры
1. Передать управление основному GM агенту
2. Начать первую сессию