#!/usr/bin/env python3
"""
WFRP RAG Indexer - индексация и поиск правил Warhammer Fantasy Roleplay 4E
Пути настраиваются через переменные окружения или определяются автоматически.
"""

import os
import re
import json
import pickle
from pathlib import Path
from collections import defaultdict

SCRIPT_DIR = Path(__file__).parent.resolve()
PROJECT_DIR = SCRIPT_DIR.parent.parent

RULES_DIR = os.environ.get("WFRP_RULES_DIR", str(PROJECT_DIR / "rules"))
CHROMA_DIR = os.environ.get("WFRP_RAG_INDEX", str(PROJECT_DIR / ".rag_index"))
EMBEDDING_MODEL = os.environ.get(
    "WFRP_EMBEDDING_MODEL",
    "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2",
)

try:
    from sentence_transformers import SentenceTransformer
    import numpy as np
    HAS_DEPS = True
except ImportError:
    HAS_DEPS = False


def load_markdown_files():
    documents = []
    for root, dirs, files in os.walk(RULES_DIR):
        for file in files:
            if file.endswith('.md'):
                filepath = os.path.join(root, file)
                with open(filepath, 'r', encoding='utf-8') as f:
                    content = f.read()
                title = Path(file).stem
                rel_path = os.path.relpath(filepath, RULES_DIR)
                sections = split_by_headers(content, title)
                for section in sections:
                    documents.append({
                        'content': section['content'],
                        'title': section['title'],
                        'source': rel_path,
                        'full_path': filepath,
                    })
    return documents


def split_by_headers(content, default_title="Без названия"):
    sections = []
    lines = content.split('\n')
    current_title = default_title
    current_content = []
    for line in lines:
        header_match = re.match(r'^(#{1,6})\s+(.+)$', line)
        if header_match:
            if current_content:
                sections.append({
                    'title': current_title,
                    'content': '\n'.join(current_content).strip(),
                })
            current_title = header_match.group(2).strip()
            current_content = [line]
        else:
            current_content.append(line)
    if current_content:
        sections.append({
            'title': current_title,
            'content': '\n'.join(current_content).strip(),
        })
    if len(sections) <= 1:
        return [{'title': default_title, 'content': content}]
    return sections


def create_index():
    if not HAS_DEPS:
        print("ERROR: sentence-transformers не установлен.")
        print("Установите: pip install sentence-transformers numpy")
        return False
    print(f"Загрузка правил из: {RULES_DIR}")
    documents = load_markdown_files()
    print(f"Загружено {len(documents)} секций")
    print(f"Загрузка модели: {EMBEDDING_MODEL}")
    model = SentenceTransformer(EMBEDDING_MODEL)
    print("Создание эмбеддингов...")
    texts = [doc['content'] for doc in documents]
    embeddings = model.encode(texts, show_progress_bar=True)
    os.makedirs(CHROMA_DIR, exist_ok=True)
    index_data = {
        'documents': documents,
        'embeddings': embeddings,
        'model': EMBEDDING_MODEL,
    }
    with open(os.path.join(CHROMA_DIR, 'index.pkl'), 'wb') as f:
        pickle.dump(index_data, f)
    print(f"Индексация завершена! Секций: {len(documents)}, База: {CHROMA_DIR}")
    return True


def search_rules(query: str, k: int = 5) -> list:
    if not HAS_DEPS:
        return [{"error": "sentence-transformers не установлен. Используйте rg -iC5 'query' rules/"}]
    index_path = os.path.join(CHROMA_DIR, 'index.pkl')
    if not os.path.exists(index_path):
        return [{"error": f"Индекс не найден ({CHROMA_DIR}). Запустите: python3 {__file__} index"}]
    with open(index_path, 'rb') as f:
        index_data = pickle.load(f)
    documents = index_data['documents']
    embeddings = index_data['embeddings']
    model = SentenceTransformer(index_data['model'])
    query_embedding = model.encode([query])[0]
    similarities = np.dot(embeddings, query_embedding) / (
        np.linalg.norm(embeddings, axis=1) * np.linalg.norm(query_embedding)
    )
    top_indices = np.argsort(similarities)[::-1][:k]
    results = []
    for idx in top_indices:
        results.append({
            'title': documents[idx]['title'],
            'content': documents[idx]['content'][:500],
            'source': documents[idx]['source'],
            'score': float(similarities[idx]),
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
            print(f"  python3 {sys.argv[0]} index          # Индексировать правила")
            print(f"  python3 {sys.argv[0]} search <query> # Поискать правила")
    else:
        create_index()
