## Why

Current WFRP GM setup relies on the LLM voluntarily calling `.claude/wfrp-rules` (ripgrep keyword search) before each rules decision. This is unreliable — the model can skip the call "for speed" — and produces no recall of past session history. When the GM runs on Hermes Agent, we get a better primitive: the `pre_llm_call` plugin hook, which injects context into every turn *before* the LLM decides anything, guaranteeing that relevant rules and session history are always in scope. This change replaces the opt-in bash-script RAG with a mandatory, file-based (SQLite FTS5) RAG plugin for Hermes — no inference servers, no Qdrant, no heavy ML dependencies.

## What Changes

- **NEW** Hermes plugin `wfrp-rag` at `plugins/wfrp-rag/` providing:
  - `pre_llm_call` hook: auto-searches rules + history FTS5 index on every turn, injects top-K snippets as context (mandatory, LLM cannot skip)
  - `post_llm_call` hook: indexes each (user_message, assistant_response) exchange into `history_fts` for future recall
  - `post_tool_call` hook: watches `write_file`/`patch` calls into `rules/` and `history/`, triggers incremental reindex of the touched file
  - `wfrp_rag_search` tool: on-demand deeper FTS5 search the LLM can call when auto-injected context is insufficient
  - `/wfrp-rag` slash command: `status`, `reindex`, `search <query>` for manual control
- **NEW** `indexer.py` module: SQLite + FTS5 indexing of `rules/**/*.md` (split by markdown headers into ~1448 sections) and `history/**/*.md` session logs, with BM25 ranking and `unicode61` tokenizer for Russian
- **NEW** WFRP term dictionary mapping common player phrases to FTS5 query expansions (e.g. "атак*" → "ближний бой атака OR дальний бой стрельба")
- **KEEP** `.claude/wfrp-rules` shell script and `rag_indexer.py` as fallback for opencode/local non-Hermes runs (unchanged)
- **NO CHANGE** to `.claude/agents/gm.md` or `AGENTS.md` — the plugin transparently augments whatever the GM prompt already instructs

## Capabilities

### New Capabilities

- `wfrp-rag-plugin`: Hermes plugin that provides mandatory RAG context injection (rules + history) via `pre_llm_call` hook, automatic history indexing via `post_llm_call`/`post_tool_call` hooks, and an on-demand `wfrp_rag_search` tool — all backed by SQLite FTS5 with zero external dependencies

### Modified Capabilities

(none — this is a new, self-contained plugin; existing `.claude/wfrp-rules` fallback is untouched)

## Impact

- **New code**: `plugins/wfrp-rag/` (~250 lines Python across 5 files + `plugin.yaml`)
- **Runtime dependency**: none beyond Python stdlib `sqlite3` (FTS5 is compiled into SQLite by default on all platforms Hermes supports)
- **Disk**: `plugins/wfrp-rag/data/wfrp_index.db` (~5MB for rules index, grows with session history)
- **Hermes config**: user must add `wfrp-rag` to `plugins.enabled` in `~/.hermes/config.yaml` after cloning the repo onto the VPS
- **No breaking changes**: existing opencode-based GM workflow continues to work via `.claude/wfrp-rules` ripgrep fallback
- **Testing**: must verify `pre_llm_call` injection appears in the LLM context and `post_llm_call` writes to `history_fts` before VPS deployment
