#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Генератор карточек персонажей WFRP в PDF формате
"""

from reportlab.lib.pagesizes import A4
from reportlab.lib.units import mm
from reportlab.pdfgen import canvas
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from reportlab.lib.colors import HexColor
import os

# Регистрируем шрифт с поддержкой кириллицы
try:
    pdfmetrics.registerFont(TTFont('DejaVuSans', '/System/Library/Fonts/Supplemental/Arial Unicode.ttf'))
    pdfmetrics.registerFont(TTFont('DejaVuSans-Bold', '/System/Library/Fonts/Supplemental/Arial Unicode.ttf'))
    FONT_NAME = 'DejaVuSans'
    FONT_BOLD = 'DejaVuSans-Bold'
except:
    # Fallback на стандартные шрифты
    FONT_NAME = 'Helvetica'
    FONT_BOLD = 'Helvetica-Bold'

# Цвета
COLOR_HEADER = HexColor('#8B0000')  # Темно-красный
COLOR_BORDER = HexColor('#000000')  # Черный
COLOR_TEXT = HexColor('#000000')    # Черный
COLOR_BG = HexColor('#F5F5DC')      # Бежевый фон

def draw_character_card(c, x, y, char_data, width=90*mm, height=130*mm):
    """
    Рисует карточку персонажа
    
    Args:
        c: Canvas объект
        x, y: Координаты левого верхнего угла карточки
        char_data: Словарь с данными персонажа
        width, height: Размеры карточки
    """
    
    # Фоновый прямоугольник
    c.setFillColor(COLOR_BG)
    c.rect(x, y - height, width, height, fill=1, stroke=0)
    
    # Рамка карточки
    c.setStrokeColor(COLOR_BORDER)
    c.setLineWidth(2)
    c.rect(x, y - height, width, height, fill=0, stroke=1)
    
    current_y = y - 10*mm
    
    # Заголовок (имя персонажа)
    c.setFillColor(COLOR_HEADER)
    c.setFont(FONT_BOLD, 14)
    c.drawString(x + 5*mm, current_y, char_data['name'])
    current_y -= 5*mm
    
    # Подзаголовок (раса и карьера)
    c.setFillColor(COLOR_TEXT)
    c.setFont(FONT_NAME, 9)
    c.drawString(x + 5*mm, current_y, f"{char_data['race']} | {char_data['career']}")
    current_y -= 7*mm
    
    # Линия-разделитель
    c.setStrokeColor(COLOR_HEADER)
    c.line(x + 5*mm, current_y, x + width - 5*mm, current_y)
    current_y -= 5*mm
    
    # HP - крупно и заметно
    c.setFont(FONT_BOLD, 11)
    hp_color = HexColor('#00AA00') if char_data['hp_current'] > char_data['hp_max'] * 0.5 else \
               HexColor('#FF8C00') if char_data['hp_current'] > char_data['hp_max'] * 0.3 else \
               HexColor('#AA0000')
    c.setFillColor(hp_color)
    c.drawString(x + 5*mm, current_y, f"HP: {char_data['hp_current']}/{char_data['hp_max']}")
    current_y -= 6*mm
    
    # Характеристики - компактно
    c.setFillColor(COLOR_TEXT)
    c.setFont(FONT_NAME, 8)
    
    stats = char_data['stats']
    stats_left = ['WS', 'BS', 'S', 'T', 'I']
    stats_right = ['Ag', 'Dex', 'Int', 'WP', 'Fel']
    
    # Левая колонка характеристик
    for stat in stats_left:
        c.drawString(x + 5*mm, current_y, f"{stat}: {stats[stat]}")
        current_y -= 4*mm
    
    current_y = y - 32*mm  # Возвращаемся для правой колонки
    # Правая колонка характеристик
    for stat in stats_right:
        c.drawString(x + 45*mm, current_y, f"{stat}: {stats[stat]}")
        current_y -= 4*mm
    
    current_y -= 3*mm
    
    # Судьба и Удача
    c.setFont(FONT_BOLD, 8)
    c.drawString(x + 5*mm, current_y, f"Судьба: {char_data['fate']}")
    c.drawString(x + 45*mm, current_y, f"Удача: {char_data['fortune']}")
    current_y -= 6*mm
    
    # Линия-разделитель
    c.setStrokeColor(COLOR_HEADER)
    c.line(x + 5*mm, current_y, x + width - 5*mm, current_y)
    current_y -= 5*mm
    
    # Оружие
    c.setFillColor(COLOR_HEADER)
    c.setFont(FONT_BOLD, 9)
    c.drawString(x + 5*mm, current_y, "Оружие:")
    current_y -= 5*mm
    
    c.setFillColor(COLOR_TEXT)
    c.setFont(FONT_NAME, 7)
    for weapon in char_data['weapons']:
        c.drawString(x + 7*mm, current_y, weapon)
        current_y -= 3.5*mm
    
    current_y -= 2*mm
    
    # Броня
    c.setFillColor(COLOR_HEADER)
    c.setFont(FONT_BOLD, 9)
    c.drawString(x + 5*mm, current_y, f"Броня: {char_data['armor']} AP")
    current_y -= 6*mm
    
    # Текущее состояние
    if char_data.get('status'):
        c.setFillColor(HexColor('#CC0000'))
        c.setFont(FONT_BOLD, 8)
        c.drawString(x + 5*mm, current_y, "СТАТУС:")
        current_y -= 4*mm
        
        c.setFont(FONT_NAME, 7)
        for status_line in char_data['status']:
            c.drawString(x + 7*mm, current_y, status_line)
            current_y -= 3.5*mm
    
    # Ключевые таланты (внизу)
    current_y = y - height + 10*mm
    c.setFillColor(COLOR_HEADER)
    c.setFont(FONT_BOLD, 8)
    c.drawString(x + 5*mm, current_y, "Ключевые таланты:")
    current_y -= 4*mm
    
    c.setFillColor(COLOR_TEXT)
    c.setFont(FONT_NAME, 6)
    for talent in char_data['key_talents'][:3]:  # Максимум 3 таланты
        c.drawString(x + 7*mm, current_y, talent)
        current_y -= 3*mm


def create_character_cards_pdf(filename='character_cards.pdf'):
    """
    Создает PDF с карточками всех персонажей
    """
    
    # Данные персонажей (текущее состояние после битвы)
    characters = [
        {
            'name': 'ФЕЛИРАН',
            'race': 'Лесной эльф',
            'career': 'Охотник (Ранг 1)',
            'hp_current': 13,
            'hp_max': 13,
            'stats': {
                'WS': 48, 'BS': 37, 'S': 36, 'T': 38, 'I': 54,
                'Ag': 41, 'Dex': 45, 'Int': 42, 'WP': 42, 'Fel': 31
            },
            'fate': 1,  # Потратил 1 Судьбу
            'fortune': 1,
            'weapons': [
                'Длинный лук (СЛОМАН!)',
                'Эльфийский короткий меч 1d10+4',
                'Охотничий нож 1d10+2'
            ],
            'armor': '1 AP (кожа)',
            'status': [
                'Невредим',
                'Лук сломан - нужна тетива',
            ],
            'key_talents': [
                'Меткая стрельба (игнор -10 дальность)',
                'Охотник (+1 УУ против животных/чудовищ)',
                'Бесшумное движение (уклон бесплатно)'
            ]
        },
        {
            'name': 'КУРТ ШТАЛЕР',
            'race': 'Человек',
            'career': 'Шарлатан (Ранг 1)',
            'hp_current': 13,
            'hp_max': 18,
            'stats': {
                'WS': 30, 'BS': 33, 'S': 29, 'T': 30, 'I': 26,
                'Ag': 28, 'Dex': 32, 'Int': 31, 'WP': 32, 'Fel': 31
            },
            'fate': 3,
            'fortune': 3,
            'weapons': [
                'Арбалет лёгкий 1d10+4 (16/32м)',
                'Кинжал 1d10+2 (Быстрый)'
            ],
            'armor': '1 AP (кожа)',
            'status': [
                'Ранен (-5 HP)',
                'Герой гребли!',
            ],
            'key_talents': [
                'Лжец (Обаяние вместо других)',
                'Быстрые руки (+1 УУ ловкость рук)',
                'Нюх на неприятности (чувствует опасность)'
            ]
        },
        {
            'name': 'ДИТРИХ РУССЕЛЬ',
            'race': 'Человек',
            'career': 'Ученик Огн. Волшебника (Р1)',
            'hp_current': 6,
            'hp_max': 15,
            'stats': {
                'WS': 29, 'BS': 28, 'S': 25, 'T': 28, 'I': 32,
                'Ag': 26, 'Dex': 20, 'Int': 24, 'WP': 25, 'Fel': 27
            },
            'fate': 1,  # Потратил 2 Судьбы!
            'fortune': 1,
            'weapons': [
                'Огненный снаряд (24м, 2+УУ урон)',
                'Огненный щит (2 урон в рукопашной)',
                'Посох 1d10+2'
            ],
            'armor': '0 AP (роба)',
            'status': [
                'ТЯЖЕЛО РАНЕН (-9 HP)',
                'Состояние: СЛОМЛЕН',
                'Спасён от жертвоприношения',
                'Возмущён!',
            ],
            'key_talents': [
                'Магический дар (+1 УУ колдовство)',
                'Второе зрение (видит магию)',
                'Пироманьяк (+10 огненные заклинания)'
            ]
        },
        {
            'name': 'ТОРГРИМ ЖЕЛЕЗНОБОРОД',
            'race': 'Дварф',
            'career': 'Воин (Ранг 1)',
            'hp_current': 5,
            'hp_max': 18,
            'stats': {
                'WS': 42, 'BS': 27, 'S': 27, 'T': 43, 'I': 30,
                'Ag': 14, 'Dex': 39, 'Int': 25, 'WP': 44, 'Fel': 21
            },
            'fate': 2,
            'fortune': 2,
            'weapons': [
                'Дварфийский боевой топор 1d10+6 (Hack)',
                'Щит 1d10+2 (Defensive)',
                'Метательный топор 1d10+4 (6/12м)'
            ],
            'armor': '2 AP (тело/руки/голова), 1 AP (ноги)',
            'status': [
                'КРИТИЧЕСКОЕ РАНЕНИЕ!',
                'БЕЗ СОЗНАНИЯ',
                'Сломано ребро',
                'В каюте с лекарем',
            ],
            'key_talents': [
                'Боевое мастерство (несколько врагов)',
                'Мастер молота (+10 топоры/молоты)',
                'Крепкий как камень (игнор Оглушён 1x)'
            ]
        }
    ]
    
    # Создаем PDF
    c = canvas.Canvas(filename, pagesize=A4)
    page_width, page_height = A4
    
    # Заголовок документа
    c.setFont(FONT_BOLD, 18)
    c.setFillColor(COLOR_HEADER)
    title = "WFRP: ВЛАСТЕЛИНЫ БОЛОТА"
    c.drawCentredString(page_width / 2, page_height - 20*mm, title)
    
    c.setFont(FONT_NAME, 12)
    c.setFillColor(COLOR_TEXT)
    subtitle = "Карточки персонажей - Текущее состояние"
    c.drawCentredString(page_width / 2, page_height - 28*mm, subtitle)
    
    # Размещаем карточки 2x2 на странице
    card_width = 90*mm
    card_height = 130*mm
    margin = 10*mm
    
    # Координаты для 2x2 сетки
    positions = [
        (margin, page_height - 50*mm),  # Верхняя левая
        (margin + card_width + margin, page_height - 50*mm),  # Верхняя правая
        (margin, page_height - 50*mm - card_height - margin),  # Нижняя левая
        (margin + card_width + margin, page_height - 50*mm - card_height - margin)  # Нижняя правая
    ]
    
    # Рисуем все 4 карточки
    for i, char in enumerate(characters):
        x, y = positions[i]
        draw_character_card(c, x, y, char, card_width, card_height)
    
    # Футер
    c.setFont(FONT_NAME, 8)
    c.setFillColor(HexColor('#666666'))
    footer = "Создано: 18.01.2026 | Сессия: Раунд 5 Финал"
    c.drawCentredString(page_width / 2, 10*mm, footer)
    
    # Сохраняем PDF
    c.save()
    print(f"PDF создан: {filename}")


if __name__ == '__main__':
    create_character_cards_pdf('wfrp_character_cards.pdf')
