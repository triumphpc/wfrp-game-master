## Context

The WFRP Game Master project currently runs on opencode with a bash-based RAG fallback (`.claude/wfrp-rules` → ripgrep). The GM prompt (`gm.md`) instructs the LLM to call `wfrp-rules` before every rules decision, but compliance is voluntary — the model can skip it. There is no indexing of session history at all.

The project is being migrated to **Hermes Agent** (Nous Research) on a VPS. Hermes provides a plugin system with lifecycle hooks, most notably `pre_llm_call`, which injects context into every turn *before* the LLM decides whether to search. This makes RAG mandatory rather than opt-in.

The rules corpus is ~1448 markdown sections across `rules/dict/*.md` (17 files) and `rules/WFRPG4E/WFRPG4E.md` (2.7MB). Session history lives in `history/<campaign>/sessions/*.md` and grows over time. The corpus is Russian-language.

Constraints:
- VPS has no GPU; inference for embeddings would be slow/costly
- Qdrant runs in Docker locally but not (yet) on the VPS
- Hermes itself uses SQLite + FTS5 for its own `session_search` — proven pattern
- The plugin must be testable locally before VPS deployment

## Goals / Non-Goals

**Goals:**
- Guarantee RAG context injection on every GM turn (rules + history) via `pre_llm_call` hook — the LLM cannot skip it
- Index WFRP rules into FTS5 with BM25 ranking and Russian-language tokenizer
- Auto-index session history: every (user_message, assistant_response) exchange via `post_llm_call`, plus file-level reindex via `post_tool_call` when the GM writes session logs
- Provide an on-demand `wfrp_rag_search` tool for deeper queries the LLM can call voluntarily
- Zero external dependencies beyond Python stdlib (`sqlite3` with FTS5 compiled in)
- Local testability: verify hooks fire and inject context before VPS deployment

**Non-Goals:**
- Semantic/embedding-based search (MiniLM, Jina) — FTS5 + WFRP term dictionary covers ~90% of cases at zero cost
- Qdrant integration — not needed without embeddings
- Replacing `.claude/wfrp-rules` ripgrep fallback for opencode — kept as-is for non-Hermes runs
- Modifying `gm.md` or `AGENTS.md` — the plugin transparently augments whatever the prompt already instructs
- Cross-session memory beyond FTS5 history index (Hermes has its own `session_search` for that)
- Multi-language support — corpus is Russian only

## Decisions

### D1: SQLite FTS5 over embeddings + Qdrant

**Choice:** Use SQLite FTS5 virtual tables with `unicode61` tokenizer and BM25 ranking.

**Alternatives considered:**
1. **sentence-transformers (MiniLM-L12-v2) + pickle** — current `rag_indexer.py` approach. Requires ~500MB deps, 5-8s per query (loads model each time), no incremental updates. Rejected for VPS deployment.
2. **Qdrant + embeddings** — best semantic quality, but requires Docker on the VPS, embedding model inference, and network round-trips. Overkill for ~1448 sections.
3. **ripgrep only** — current fallback. No ranking, no morphology, no history. Baseline, not target.

**Rationale:** FTS5 is built into Python's `sqlite3` (FTS5 has been compiled into SQLite by default since Python 3.10+ on all major platforms). BM25 ranking gives relevance scoring. `unicode61` tokenizer handles Russian Cyrillic. Prefix matching (`атак*` → атака, атакует) covers morphology. Hermes itself uses FTS5 for `session_search` — proven pattern in this exact runtime. Query latency: 1-5ms.

### D2: `pre_llm_call` hook for mandatory context injection

**Choice:** Register a `pre_llm_call` hook that searches the FTS5 index and returns `{"context": "..."}` appended to the user message.

**Rationale:** Per Hermes docs, `pre_llm_call` is "the mechanism for memory plugins, RAG integrations, guardrails." It fires once per turn before the tool-calling loop. The return value is injected into the **user message** (not system prompt), preserving Anthropic/OpenRouter prompt cache. This is the only hook whose return value matters — all others are observers.

**Alternative considered:** Rely solely on the `wfrp_rag_search` tool and instruct the LLM in `gm.md` to always call it. Rejected — this is the current unreliable pattern; the LLM can skip it.

### D3: Hybrid query construction (WFRP term dictionary + keyword extraction)

**Choice:** Build FTS5 queries by (a) matching player message against a WFRP term dictionary that expands to curated FTS5 phrases, and (b) extracting general keywords with prefix wildcards.

**Term dictionary example:**
```
"атак"  → "ближний бой атака OR дальний бой стрельба"
"провер" → "проверка навык успех провал УУ"
"маги"  → "магия заклинание концентрация ошибка"
"крит"  → "критическая рана таблица заминка"
"обремен" → "обременение лимит вес перегруз"
```

**Keyword extraction:** regex `\b[а-яА-ЯёЁ]{4,}\b`, filter Russian stopwords, truncate to 5-char prefix + `*` for FTS5 prefix matching.

**Rationale:** Pure keyword search misses synonyms ("убедить" should find "красноречие"). Pure embeddings would catch this but cost too much. The term dictionary bridges the gap for the ~20 most common WFRP mechanics, and keyword extraction handles the rest. The dictionary is a static Python dict in `indexer.py` — easy to extend.

### D4: Two FTS5 tables — `rules_fts` and `history_fts`

**Choice:** Separate FTS5 virtual tables for rules and history, searched independently and merged in the hook.

**Schema:**
```sql
-- rules_fts: indexed from rules/**/*.md, split by markdown headers
CREATE VIRTUAL TABLE rules_fts USING fts5(
    title, source, content,
    tokenize = 'unicode61'
);

-- history_fts: indexed from post_llm_call exchanges + history/**/*.md files
CREATE VIRTUAL TABLE history_fts USING fts5(
    session_id, timestamp, role, content, source,
    tokenize = 'unicode61'
);

-- files_meta: for incremental reindex (mtime check)
CREATE TABLE files_meta (
    path TEXT PRIMARY KEY,
    mtime REAL,
    section_count INTEGER
);
```

**Rationale:** Rules are static (rarely change), history grows continuously. Separate tables allow different reindex strategies: rules check mtime on plugin load, history appends on every `post_llm_call`. Searching independently lets us control K per source (e.g. top-3 rules + top-2 history).

### D5: `post_tool_call` hook for file-level incremental reindex

**Choice:** Monitor `write_file` and `patch` tool calls. If the path is under `rules/` or `history/`, reindex just that file (delete old sections by `source`, insert new ones).

**Rationale:** The GM writes session logs via the `log` skill (`write` to `history/<campaign>/sessions/*.md`). Without this hook, history_fts would only contain raw (user_message, assistant_response) exchanges, missing the structured combat tables and state changes the GM logs. The hook catches both: raw exchanges via `post_llm_call`, structured logs via `post_tool_call`.

### D6: Plugin structure — 5 files + plugin.yaml

```
plugins/wfrp-rag/
├── plugin.yaml          # manifest (name, version, provides_tools, provides_hooks)
├── __init__.py          # register(): wire hooks + tool, load index on startup
├── schemas.py           # wfrp_rag_search tool schema (JSON Schema for LLM)
├── tools.py             # handle_rag_search() — on-demand FTS5 search
├── indexer.py           # FTS5 indexing + search logic + WFRP term dictionary
└── data/
    └── wfrp_index.db    # SQLite database (created on first run, ~5MB)
```

**Rationale:** Follows the Hermes plugin guide's "four files, clear separation" pattern (manifest, schemas, handlers, registration). `indexer.py` is the only non-trivial module (~150 lines). The `data/` directory is gitignored — the index is rebuilt from source files.

### D7: Config via plugin.yaml `metadata.hermes.config`

**Choice:** Expose config through Hermes' skill config system:
```yaml
metadata:
  hermes:
    config:
      - key: wfrp-rag.rules_dir
        description: Path to WFRP rules directory
        default: "./rules"
      - key: wfrp-rag.history_dir
        description: Path to session history directory
        default: "./history"
      - key: wfrp-rag.rules_k
        description: Number of rule snippets to inject per turn
        default: "3"
      - key: wfrp-rag.history_k
        description: Number of history snippets to inject per turn
        default: "2"
      - key: wfrp-rag.max_context_chars
        description: Max characters of injected context per turn
        default: "2000"
```

**Rationale:** Hermes injects resolved config values into the plugin context automatically. Users can override via `config.yaml` or `hermes config set`. No env var juggling.

## Risks / Trade-offs

- **[FTS5 not available]** Some niche SQLite builds lack FTS5 compiled in → **Mitigation:** Check `PRAGMA compile_options` on plugin load; if FTS5 missing, log error and fall back to `LIKE '%query%'` (slower, no ranking). Hermes requires Python 3.11+ where FTS5 is standard.

- **[FTS5 morphology limits]** Russian morphology via `unicode61` is prefix-only (стеммер Porter не встроен) → "бежать" и " бег" не совпадут → **Mitigation:** WFRP term dictionary covers the 20 most common cases; `wfrp_rag_search` tool lets the LLM retry with refined query.

- **[Context bloat]** If K is too high or sections are long, injected context could dominate the user message → **Mitigation:** `max_context_chars` config (default 2000) truncates; sections are snippeted (FTS5 `snippet()` function) not full-text in `pre_llm_call`.

- **[History index grows unbounded]** Long campaigns produce hundreds of session files → **Mitigation:** `history_fts` is just text in SQLite; even 1000 sessions × 50 exchanges = 50K rows, FTS5 handles millions. Add `/wfrp-rag prune` slash command in a future iteration if needed.

- **[Plugin not loaded]** User forgets `hermes plugins enable wfrp-rag` on VPS → **Mitigation:** `plugin.yaml` declares `requires_env: []` (no env gates), and the `/wfrp-rag status` command prints a clear "plugin not loaded" message if invoked when disabled.

- **[Testing hooks locally without Hermes]** Hooks fire inside the Hermes agent loop, hard to unit-test in isolation → **Mitigation:** `indexer.py` is pure Python (testable directly); hook functions accept the documented kwargs and can be called with mock arguments. Create a `test_hooks.py` that simulates `pre_llm_call` / `post_llm_call` / `post_tool_call` with fixture data.

- **[rules/ path differs on VPS]** The repo might be cloned to a different path on the VPS → **Mitigation:** Config keys `wfrp-rag.rules_dir` / `wfrp-rag.history_dir` default to `./rules` and `./history` (relative to Hermes working directory). User can override.
