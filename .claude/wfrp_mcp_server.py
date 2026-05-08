#!/usr/bin/env python3
"""
WFRP MCP Server — Model Context Protocol сервер для WFRP 4E
Инструменты:
  - wfrp_search_rules: поиск по правилам
  - wfrp_search_history: поиск по истории сессий
  - wfrp_get_character: текущее состояние персонажа
"""

import os
import json
import pickle
import numpy as np
from sentence_transformers import SentenceTransformer
from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp.types import Tool, TextContent

# ── Paths ──────────────────────────────────────────────────────────────
BASE_DIR = "/home/node/.openclaw/workspace"
RULES_DIR = os.path.join(BASE_DIR, "wfrp-repo/rules")
HISTORY_DIR = os.path.join(BASE_DIR, "wfrp-repo/history")
RAG_INDEX = os.path.join(BASE_DIR, "wfrp-repo/.rag_index/index.pkl")
EMBEDDING_MODEL = "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2"

# ── Globals (lazy loaded) ──────────────────────────────────────────────
_model = None
_rules_data = None
_hist_docs = None
_hist_embs = None


def get_model():
    global _model
    if _model is None:
        _model = SentenceTransformer(EMBEDDING_MODEL)
    return _model


def load_rules():
    global _rules_data
    if _rules_data is not None:
        return _rules_data
    if os.path.exists(RAG_INDEX):
        with open(RAG_INDEX, 'rb') as f:
            _rules_data = pickle.load(f)
    return _rules_data


def load_history():
    global _hist_docs, _hist_embs
    if _hist_docs is not None:
        return _hist_docs, _hist_embs

    cache = os.path.join(BASE_DIR, "wfrp-repo/.rag_index/history_index.pkl")
    if os.path.exists(cache):
        with open(cache, 'rb') as f:
            d = pickle.load(f)
        _hist_docs, _hist_embs = d['documents'], d['embeddings']
        return _hist_docs, _hist_embs

    docs = []
    for root, _, files in os.walk(HISTORY_DIR):
        for fn in files:
            if not fn.endswith('.md'):
                continue
            fp = os.path.join(root, fn)
            rel = os.path.relpath(fp, HISTORY_DIR)
            with open(fp, 'r', encoding='utf-8') as f:
                content = f.read()
            chunks, cur = [], ""
            for para in content.split('\n\n'):
                if len(cur) + len(para) > 1000 and cur:
                    chunks.append(cur.strip())
                    cur = para
                else:
                    cur += "\n\n" + para
            if cur.strip():
                chunks.append(cur.strip())
            for i, ch in enumerate(chunks):
                if len(ch) < 50:
                    continue
                docs.append({'content': ch[:1500], 'source': rel, 'chunk': i})

    if docs:
        m = get_model()
        embs = m.encode([d['content'] for d in docs], show_progress_bar=False)
    else:
        embs = np.array([])

    _hist_docs, _hist_embs = docs, embs
    os.makedirs(os.path.dirname(cache), exist_ok=True)
    with open(cache, 'wb') as f:
        pickle.dump({'documents': docs, 'embeddings': embs}, f)
    return _hist_docs, _hist_embs


def cosine_top(embeddings, query_vec, k=5):
    if len(embeddings) == 0:
        return []
    sims = np.dot(embeddings, query_vec) / (
        np.linalg.norm(embeddings, axis=1) * np.linalg.norm(query_vec) + 1e-10
    )
    top = np.argsort(sims)[::-1][:k]
    return [(int(i), float(sims[i])) for i in top]


# ── Characters ─────────────────────────────────────────────────────────
CHARACTERS = {
    "тронгольд": {
        "name": "Тронгольд Пиикберссон", "race": "Гном", "player": "Тушкан",
        "hp": "17/17", "status": "✅", "money": "57сш+2мп+50зк",
        "location": "Часовня (День 17)", "notes": "Клан Пиикберссон"
    },
    "гюнтер": {
        "name": "Гюнтер Штальфауст", "race": "Человек", "player": "Sebastian",
        "hp": "12/14", "status": "⚠️ Ранен", "money": "7зк+48сш+27мп",
        "location": "Часовня (День 17)", "notes": "Перевязан"
    },
    "фелирон": {
        "name": "Фелирон", "race": "Эльф", "player": "Ден",
        "hp": "24/24", "status": "✅", "money": "20зк+55сш+6мп",
        "location": "Часовня (День 17)", "notes": "Лук «Квингальф»"
    },
}


# ── MCP Server ─────────────────────────────────────────────────────────
app = Server("wfrp-rag")


@app.list_tools()
async def list_tools():
    return [
        Tool(
            name="wfrp_search_rules",
            description="Search WFRP 4E rules by semantic query. Returns relevant rule sections with source. Use before any check, combat, or mechanics question.",
            inputSchema={
                "type": "object",
                "properties": {
                    "query": {"type": "string", "description": "Search query in Russian"},
                    "top_k": {"type": "integer", "description": "Number of results (default 5)", "default": 5}
                },
                "required": ["query"]
            }
        ),
        Tool(
            name="wfrp_search_history",
            description="Search past WFRP game sessions by semantic query. Returns relevant passages from session logs. Use to recall what happened, find NPC names, locations, plot details.",
            inputSchema={
                "type": "object",
                "properties": {
                    "query": {"type": "string", "description": "Search query in Russian"},
                    "top_k": {"type": "integer", "description": "Number of results (default 5)", "default": 5}
                },
                "required": ["query"]
            }
        ),
        Tool(
            name="wfrp_get_character",
            description="Get current WFRP character status (HP, money, location, notes). Use to check character state before actions.",
            inputSchema={
                "type": "object",
                "properties": {
                    "name": {"type": "string", "description": "Character name (тронгольд/гюнтер/фелирон)"}
                },
                "required": ["name"]
            }
        ),
    ]


@app.call_tool()
async def call_tool(name, arguments):
    model = get_model()

    if name == "wfrp_search_rules":
        data = load_rules()
        if data is None:
            return [TextContent(type="text", text="RAG index not found. Run rag_indexer.py index first.")]
        q_vec = model.encode([arguments["query"]])[0]
        k = arguments.get("top_k", 5)
        results = cosine_top(data['embeddings'], q_vec, k)
        out = []
        for idx, score in results:
            d = data['documents'][idx]
            out.append(f"**{d['title']}** ({d['source']}) [score: {score:.3f}]\n{d['content'][:500]}")
        return [TextContent(type="text", text="\n\n---\n\n".join(out) if out else "No results found.")]

    elif name == "wfrp_search_history":
        docs, embs = load_history()
        if not docs:
            return [TextContent(type="text", text="No session history indexed.")]
        q_vec = model.encode([arguments["query"]])[0]
        k = arguments.get("top_k", 5)
        results = cosine_top(embs, q_vec, k)
        out = []
        for idx, score in results:
            d = docs[idx]
            out.append(f"**{d['source']}** (chunk {d['chunk']}) [score: {score:.3f}]\n{d['content'][:600]}")
        return [TextContent(type="text", text="\n\n---\n\n".join(out) if out else "No results found.")]

    elif name == "wfrp_get_character":
        char_name = arguments["name"].lower().strip()
        char = CHARACTERS.get(char_name)
        if char is None:
            available = ", ".join(CHARACTERS.keys())
            return [TextContent(type="text", text=f"Character not found. Available: {available}")]
        lines = [f"**{char['name']}** ({char['race']})",
                 f"Player: {char['player']}",
                 f"HP: {char['hp']} {char['status']}",
                 f"Money: {char['money']}",
                 f"Location: {char['location']}",
                 f"Notes: {char['notes']}"]
        return [TextContent(type="text", text="\n".join(lines))]

    return [TextContent(type="text", text=f"Unknown tool: {name}")]


async def main():
    async with stdio_server() as (read_stream, write_stream):
        await app.run(read_stream, write_stream, app.create_initialization_options())


if __name__ == "__main__":
    import asyncio
    asyncio.run(main())
