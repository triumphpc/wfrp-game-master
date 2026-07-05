# WFRP Game Master — Hermes Agent Setup

This guide covers setting up the WFRP GM on Hermes Agent (Nous Research) on a VPS.

## Prerequisites

- Linux VPS (Ubuntu/Debian recommended)
- Git
- Hermes Agent installed: `curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash`

## Installation

### 1. Clone the repo

```bash
git clone <repo-url> ~/wfrp-game-master
```

### 2. Install the WFRP RAG plugin

```bash
# Symlink the plugin into Hermes plugins directory
ln -s ~/wfrp-game-master/plugins/wfrp-rag ~/.hermes/plugins/wfrp-rag

# Enable the plugin
hermes plugins enable wfrp-rag
```

### 3. Configure Hermes

```bash
# Copy the template config
cp ~/wfrp-game-master/hermes.config.yaml ~/.hermes/config.yaml

# Edit paths if your repo is not at ~/wfrp-game-master
# Set current_campaign to the campaign you're playing
```

### 4. Verify

```bash
# Start Hermes
hermes chat

# Check plugin loaded and index built
/wfrp-rag status

# Expected output:
#   WFRP RAG Index
#     Rules sections: 2801
#     History entries: 2192
#     Current campaign: Тени Старого Тракта
#     Campaigns:
#       Тени Старого Тракта: 1230 entries
#       Властелин_болот: 517 entries
#       ...

# Test search
/wfrp-rag search критическая рана

# Switch campaign
/wfrp-rag campaign Властелин_болот
```

## How it works

### RAG context injection (mandatory, every turn)

```
Player message → pre_llm_call hook
  ↓
  1. Extract keywords from message
  2. Search rules_fts (top-3 by BM25)
  3. Search history_fts (top-2, filtered by current_campaign)
  4. Inject {"context": "📚 Правила:... 📖 История:..."} into user message
  ↓
LLM sees player message + relevant rules + relevant history
```

The LLM cannot skip this — it happens before every LLM call.

### History auto-indexing

- **`post_llm_call`**: After every turn, the (user_message, assistant_response) pair is indexed into `history_fts` with the current campaign.
- **`post_tool_call`**: When the GM writes a session log file (via `write_file` to `history/`), the file is reindexed immediately.

### Campaign filtering

History search is scoped to the current campaign. Set it via:

- `hermes.config.yaml`: `wfrp-rag.current_campaign: "Тени Старого Тракта"`
- Runtime: `/wfrp-rag campaign "Тени Старого Тракта"`
- Runtime (all campaigns): `/wfrp-rag campaign ""`

### On-demand deep search

The `wfrp_rag_search` tool lets the LLM search for detailed rules when auto-injected context is insufficient:

```
LLM: wfrp_rag_search(query="критическая рана таблица голова", source="rules", k=5)
  → Returns full section content (not snippets) with BM25 scores
```

## Configuration reference

| Key | Default | Description |
|---|---|---|
| `wfrp-rag.rules_dir` | `./rules` | Path to WFRP rules directory |
| `wfrp-rag.history_dir` | `./history` | Path to session history directory |
| `wfrp-rag.rules_k` | `3` | Rule snippets injected per turn |
| `wfrp-rag.history_k` | `2` | History snippets injected per turn |
| `wfrp-rag.max_context_chars` | `2000` | Max chars of injected context |
| `wfrp-rag.current_campaign` | `(empty)` | Campaign name for history filtering |

## Slash commands

| Command | Description |
|---|---|
| `/wfrp-rag status` | Index stats, campaign list, current campaign |
| `/wfrp-rag campaign <name>` | Set current campaign (or list available if no arg) |
| `/wfrp-rag reindex` | Force full reindex of all files |
| `/wfrp-rag search <query>` | Manual search (rules + current campaign history) |

## Files

```
plugins/wfrp-rag/
├── plugin.yaml          # manifest + config keys
├── __init__.py          # register(): hooks, tool, slash command
├── schemas.py           # wfrp_rag_search tool schema
├── tools.py             # on-demand search handler
├── indexer.py           # FTS5 indexing + search + WFRP term dictionary
├── conftest.py          # pytest config
├── pytest.ini           # test config
├── data/                # (gitignored) SQLite index, auto-created
└── tests/
    ├── test_indexer.py  # unit tests for indexer
    └── test_hooks.py    # unit tests for hooks (mocked)

hermes.config.yaml       # template config for VPS deployment
```

## Zero dependencies

The plugin uses only Python stdlib (`sqlite3` with FTS5 compiled in). No `pip install` needed. No external servers (Qdrant, inference) needed.

## Fallback for opencode (non-Hermes)

The original `.claude/wfrp-rules` shell script (ripgrep fallback) is still available for running the GM on opencode without Hermes. It does not have campaign filtering or auto-indexing.
