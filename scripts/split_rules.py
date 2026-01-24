#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
Скрипт для разделения файла правил WFRPG4E.ru.md на отдельные главы.
Создаёт структуру каталогов в rules/ и разбивает большой файл на управляемые части.
"""

import os
import re

# Определяем основные главы и их ключевые маркеры
CHAPTERS = [
    {
        'name': '01_введение',
        'title': 'ВВЕДЕНИЕ',
        'start_marker': 'ВВЕДЕНИЕ',
        'end_page': 23  # До страницы "ПЕРСОНАЖ"
    },
    {
        'name': '02_персонаж',
        'title': 'ПЕРСОНАЖ',
        'start_marker': 'СОЗДАНИЕ ПЕРСОНАЖА',
        'end_marker': 'КЛАССЫ И КАРЬЕРЫ\nВ Старом Свете великое множество'
    },
    {
        'name': '03_классы_и_карьеры',
        'title': 'КЛАССЫ И КАРЬЕРЫ',
        'start_marker': 'КЛАССЫ\nБюргеры:',
        'end_marker': 'НАВЫКИ И ТАЛАНТЫ\nВсе способности'
    },
    {
        'name': '04_навыки_и_таланты',
        'title': 'НАВЫКИ И ТАЛАНТЫ',
        'start_marker': 'НАВЫКИ И ТАЛАНТЫ',
        'end_marker': 'ПРАВИЛА'
    },
    {
        'name': '05_правила',
        'title': 'ПРАВИЛА',
        'start_marker': 'ПРАВИЛА\nТочно подобранное название',
        'end_marker': 'МЕЖДУ ПРИКЛЮЧЕНИЯМИ'
    },
    {
        'name': '06_между_приключениями',
        'title': 'МЕЖДУ ПРИКЛЮЧЕНИЯМИ',
        'start_marker': 'МЕЖДУ ПРИКЛЮЧЕНИЯМИ',
        'end_marker': 'РЕЛИГИЯ И ВЕРА'
    },
    {
        'name': '07_религия_и_вера',
        'title': 'РЕЛИГИЯ И ВЕРА',
        'start_marker': 'РЕЛИГИЯ И ВЕРА',
        'end_marker': 'МАГИЯ'
    },
    {
        'name': '08_магия',
        'title': 'МАГИЯ',
        'start_marker': 'МАГИЯ\nЛишь злонамеренный',
        'end_marker': 'ВЕДУЩИЙ'
    },
    {
        'name': '09_ведущий',
        'title': 'ВЕДУЩИЙ',
        'start_marker': 'ВЕДУЩИЙ',
        'end_marker': 'СЛАВНЫЙ РЕЙКЛАНД'
    },
    {
        'name': '10_славный_рейкланд',
        'title': 'СЛАВНЫЙ РЕЙКЛАНД',
        'start_marker': 'СЛАВНЫЙ РЕЙКЛАНД',
        'end_marker': 'РУКОВОДСТВО'
    },
    {
        'name': '11_руководство_покупателя',
        'title': 'РУКОВОДСТВО ПОКУПАТЕЛЯ',
        'start_marker': 'РУКОВОДСТВО',
        'end_marker': 'БЕСТИАРИЙ'
    },
    {
        'name': '12_бестиарий',
        'title': 'БЕСТИАРИЙ',
        'start_marker': 'БЕСТИАРИЙ',
        'end_marker': 'Бланк персонажа'
    },
]


def read_file(file_path):
    """Читает файл и возвращает его содержимое."""
    with open(file_path, 'r', encoding='utf-8') as f:
        return f.read()


def find_chapter_boundaries(content):
    """Находит границы глав в содержимом файла."""
    chapters_data = []
    
    for i, chapter in enumerate(CHAPTERS):
        start_pos = content.find(chapter['start_marker'])
        
        if start_pos == -1:
            print(f"Предупреждение: не найден маркер начала главы '{chapter['name']}'")
            continue
        
        # Ищем конец главы
        if i < len(CHAPTERS) - 1:
            next_chapter = CHAPTERS[i + 1]
            end_pos = content.find(next_chapter['start_marker'], start_pos + 1)
        else:
            # Последняя глава - до конца файла или до бланка персонажа
            end_marker = chapter.get('end_marker', None)
            if end_marker:
                end_pos = content.find(end_marker, start_pos + 1)
            else:
                end_pos = len(content)
        
        if end_pos == -1:
            end_pos = len(content)
        
        chapters_data.append({
            'name': chapter['name'],
            'title': chapter['title'],
            'start': start_pos,
            'end': end_pos,
            'content': content[start_pos:end_pos]
        })
    
    return chapters_data


def save_chapters(chapters_data, output_dir):
    """Сохраняет главы в отдельные файлы."""
    os.makedirs(output_dir, exist_ok=True)
    
    # Создаём README для навигации
    readme_content = "# Правила WFRP 4e\n\n"
    readme_content += "Файл правил разбит на отдельные главы для удобства использования и поиска.\n\n"
    readme_content += "## Содержание\n\n"
    
    for chapter in chapters_data:
        file_name = f"{chapter['name']}.md"
        file_path = os.path.join(output_dir, file_name)
        
        # Формируем содержимое файла главы
        chapter_content = f"# {chapter['title']}\n\n"
        chapter_content += chapter['content']
        
        # Сохраняем главу
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(chapter_content)
        
        print(f"Создан файл: {file_name}")
        
        # Добавляем в README
        readme_content += f"- [{chapter['title']}]({file_name})\n"
    
    # Сохраняем README
    readme_path = os.path.join(output_dir, 'README.md')
    with open(readme_path, 'w', encoding='utf-8') as f:
        f.write(readme_content)
    
    print(f"\nСоздан файл: README.md")


def main():
    """Основная функция скрипта."""
    input_file = 'WFRPG4E.ru.md'
    output_dir = 'rules'
    
    print("Чтение исходного файла...")
    content = read_file(input_file)
    
    print(f"Размер файла: {len(content)} символов")
    print("\nПоиск границ глав...")
    
    chapters_data = find_chapter_boundaries(content)
    
    print(f"\nНайдено глав: {len(chapters_data)}")
    print("\nСохранение глав в отдельные файлы...")
    
    save_chapters(chapters_data, output_dir)
    
    print(f"\n✓ Готово! Все главы сохранены в директории '{output_dir}/'")
    print(f"✓ Создано файлов: {len(chapters_data) + 1} (включая README.md)")


if __name__ == '__main__':
    main()
