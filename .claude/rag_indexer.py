#!/usr/bin/env python3
"""
WFRP RAG Indexer - индексация правил Warhammer Fantasy Roleplay 4E
Упрощённая версия без проблем с зависимостями.
"""

import os
import re
import json
import pickle
from pathlib import Path
from collections import defaultdict

# Настройки
RULES_DIR = "/home/node/.openclaw/workspace/wfrp-repo/rules"
CHROMA_DIR = "/home/node/.openclaw/workspace/wfrp-repo/.rag_index"
EMBEDDING_MODEL = "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2"

# Для эмбеддингов
from sentence_transformers import SentenceTransformer
import numpy as np


def load_markdown_files():
    """Загрузка всех markdown файлов"""
    documents = []
    
    for root, dirs, files in os.walk(RULES_DIR):
        for file in files:
            if file.endswith('.md'):
                filepath = os.path.join(root, file)
                with open(filepath, 'r', encoding='utf-8') as f:
                    content = f.read()
                    
                # Извлечение заголовков для метаданных
                title = Path(file).stem
                rel_path = os.path.relpath(filepath, RULES_DIR)
                
                # Разбиение на секции по заголовкам
                sections = split_by_headers(content, title)
                
                for section in sections:
                    documents.append({
                        'content': section['content'],
                        'title': section['title'],
                        'source': rel_path,
                        'full_path': filepath
                    })
    
    return documents


def split_by_headers(content, default_title="Без названия"):
    """Разбиение markdown на секции по заголовкам"""
    sections = []
    lines = content.split('\n')
    
    current_title = default_title
    current_content = []
    current_level = 0
    
    for line in lines:
        # Определение заголовка
        header_match = re.match(r'^(#{1,6})\s+(.+)$', line)
        
        if header_match:
            # Сохраняем предыдущую секцию
            if current_content:
                sections.append({
                    'title': current_title,
                    'content': '\n'.join(current_content).strip()
                })
            
            # Начинаем новую секцию
            current_title = header_match.group(2).strip()
            current_level = len(header_match.group(1))
            current_content = [line]
        else:
            current_content.append(line)
    
    # Последняя секция
    if current_content:
        sections.append({
            'title': current_title,
            'content': '\n'.join(current_content).strip()
        })
    
    # Если слишком мало секций, возвращаем весь документ
    if len(sections) <= 1:
        return [{'title': default_title, 'content': content}]
    
    return sections


def create_index():
    """Создание индекса"""
    print(f"📂 Загрузка правил из: {RULES_DIR}")
    
    documents = load_markdown_files()
    print(f"📄 Загружено {len(documents)} секций")
    
    # Загрузка модели
    print(f"🤖 Загрузка модели: {EMBEDDING_MODEL}")
    model = SentenceTransformer(EMBEDDING_MODEL)
    
    # Создание эмбеддингов
    print("🔢 Создание эмбеддингов...")
    texts = [doc['content'] for doc in documents]
    embeddings = model.encode(texts, show_progress_bar=True)
    
    # Сохранение
    os.makedirs(CHROMA_DIR, exist_ok=True)
    
    index_data = {
        'documents': documents,
        'embeddings': embeddings,
        'model': EMBEDDING_MODEL
    }
    
    with open(os.path.join(CHROMA_DIR, 'index.pkl'), 'wb') as f:
        pickle.dump(index_data, f)
    
    print(f"✅ Индексация завершена!")
    print(f"   Секций: {len(documents)}")
    print(f"   База: {CHROMA_DIR}")


def search_rules(query: str, k: int = 5) -> list:
    """Поиск релевантных правил"""
    # Загрузка индекса
    index_path = os.path.join(CHROMA_DIR, 'index.pkl')
    if not os.path.exists(index_path):
        return [{"error": "Индекс не найден. Запустите индексацию."}]
    
    with open(index_path, 'rb') as f:
        index_data = pickle.load(f)
    
    documents = index_data['documents']
    embeddings = index_data['embeddings']
    
    # Модель
    model = SentenceTransformer(index_data['model'])
    
    # Эмбеддинг запроса
    query_embedding = model.encode([query])[0]
    
    # Косинусное сходство
    similarities = np.dot(embeddings, query_embedding) / (
        np.linalg.norm(embeddings, axis=1) * np.linalg.norm(query_embedding)
    )
    
    # Топ-K результатов
    top_indices = np.argsort(similarities)[::-1][:k]
    
    results = []
    for idx in top_indices:
        results.append({
            'title': documents[idx]['title'],
            'content': documents[idx]['content'][:500],  # Первые 500 символов
            'source': documents[idx]['source'],
            'score': float(similarities[idx])
        })
    
    return results


if __name__ == "__main__":
    import sys
    
    if len(sys.argv) > 1:
        if sys.argv[1] == "index":
            create_index()
        elif sys.argv[1] == "search" and len(sys.argv) > 2:
            query = " ".join(sys.argv[2:])
            results = search_rules(query)
            print(json.dumps(results, indent=2, ensure_ascii=False))
        else:
            print("Usage:")
            print("  python3 rag_indexer.py index          # Индексировать правила")
            print("  python3 rag_indexer.py search <query> # Поискать правила")
    else:
        create_index()
