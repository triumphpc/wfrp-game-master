# wfrp-rag-plugin Specification

## Purpose
TBD - created by archiving change wfrp-rag-hermes-plugin. Update Purpose after archive.
## Requirements
### Requirement: Plugin registers `pre_llm_call` hook for mandatory RAG injection

The plugin SHALL register a `pre_llm_call` hook that fires before every LLM turn. The hook SHALL search the FTS5 index (rules + history) using keywords extracted from the user message, and return a `{"context": "..."}` dict appended to the user message. This injection is mandatory — the LLM cannot skip it.

#### Scenario: Player sends a combat action

- **WHEN** the player sends "Гюнтер атакует орка двуручным мечом"
- **THEN** the `pre_llm_call` hook extracts keywords ("атак*", "орк*", "двуручн*", "меч*")
- **AND** searches `rules_fts` returning top-3 matching sections (e.g. "Ближний бой", "Двуручный меч", "Качества оружия")
- **AND** searches `history_fts` returning top-2 matching past exchanges
- **AND** returns `{"context": "📚 Правила:\n1. ...\n\n📖 История:\n1. ..."}` appended to the user message

#### Scenario: Player sends a trivial message with no rules relevance

- **WHEN** the player sends "ок" or "продолжай"
- **THEN** the `pre_llm_call` hook extracts no meaningful keywords
- **AND** returns `None` (no context injected)

#### Scenario: FTS5 search returns no results

- **WHEN** the player sends a message whose keywords match nothing in the index
- **THEN** the hook returns `None` (no context injected)
- **AND** the LLM proceeds normally without RAG context

### Requirement: Plugin registers `post_llm_call` hook for history indexing

The plugin SHALL register a `post_llm_call` hook that fires after every successful LLM turn. The hook SHALL index the (user_message, assistant_response) pair into `history_fts` with the current session_id and timestamp.

#### Scenario: GM responds to a player action

- **WHEN** the LLM completes a turn with user_message="Гюнтер бьёт орка" and assistant_response="Орк получает 12 урона..."
- **THEN** the `post_llm_call` hook inserts two rows into `history_fts`: one for the user message (role="user"), one for the assistant response (role="assistant")
- **AND** both rows share the same session_id and timestamp

#### Scenario: LLM turn fails or is interrupted

- **WHEN** the LLM turn fails (API error, interruption)
- **THEN** the `post_llm_call` hook does NOT fire (per Hermes docs, it fires on successful turns only)
- **AND** no partial data is written to `history_fts`

### Requirement: Plugin registers `post_tool_call` hook for file reindex

The plugin SHALL register a `post_tool_call` hook that monitors `write_file` and `patch` tool calls. When the file path is under `rules/` or `history/`, the hook SHALL reindex that file: delete existing sections with matching `source` and insert fresh sections parsed from the updated file.

#### Scenario: GM writes a session log

- **WHEN** the GM calls `write_file` with path `history/Тени Старого Тракта/sessions/003_battle.md`
- **THEN** the `post_tool_call` hook detects the path is under `history/`
- **AND** deletes all existing rows from `history_fts` where `source` matches that file
- **AND** parses the new file content into sections by markdown headers
- **AND** inserts the new sections into `history_fts`

#### Scenario: GM writes a file outside rules/ and history/

- **WHEN** the GM calls `write_file` with path `notes/random.md`
- **THEN** the `post_tool_call` hook detects the path is NOT under `rules/` or `history/`
- **AND** does nothing (no reindex)

### Requirement: Plugin provides `wfrp_rag_search` tool for on-demand deep search

The plugin SHALL register a `wfrp_rag_search` tool that the LLM can call voluntarily for deeper FTS5 search than the auto-injected context. The tool SHALL accept `query` (string), `source` (enum: rules, history, all), and `k` (integer, default 5), and return JSON with matching sections including full content (not snippets).

#### Scenario: LLM needs detailed critical hit rules

- **WHEN** the LLM calls `wfrp_rag_search(query="критическая рана таблица голова", source="rules", k=5)`
- **THEN** the tool searches `rules_fts` with BM25 ranking
- **AND** returns JSON: `{"results": [{"title": "...", "source": "dict/БОЙ.md", "content": "полный текст секции", "score": -1.2}, ...]}`

#### Scenario: LLM searches across both rules and history

- **WHEN** the LLM calls `wfrp_rag_search(query="бой в таверне Шрам", source="all")`
- **THEN** the tool searches both `rules_fts` and `history_fts`
- **AND** returns merged results with source labels indicating which index each came from

### Requirement: FTS5 index built from markdown files split by headers

The indexer SHALL parse all `*.md` files under the configured `rules_dir` and `history_dir`, split each file into sections by markdown headers (`#` through `######`), and insert each section as a row in the corresponding FTS5 table with `title` (header text), `source` (relative file path), and `content` (section body).

#### Scenario: Indexing rules/dict/БОЙ.md

- **WHEN** the indexer processes `rules/dict/БОЙ.md` containing headers "## Ближний бой", "### Натиск", "## Дальний бой"
- **THEN** it creates 3+ sections (one per header, plus preamble if any)
- **AND** each section has `title` = header text, `source` = "dict/БОЙ.md", `content` = lines from that header until the next

#### Scenario: File with no headers

- **WHEN** the indexer processes a `.md` file with no markdown headers
- **THEN** it creates a single section with `title` = filename stem, `source` = relative path, `content` = entire file

### Requirement: Incremental reindex based on file mtime

On plugin load (`register()`), the indexer SHALL compare each file's mtime against `files_meta` table. Only files with newer mtime than the stored value SHALL be reindexed. Files not present in `files_meta` SHALL be indexed for the first time. This ensures fast startup when no rules have changed.

#### Scenario: First run — no index exists

- **WHEN** the plugin loads and `data/wfrp_index.db` does not exist
- **THEN** the indexer creates the database, FTS5 tables, and `files_meta` table
- **AND** indexes all `*.md` files under `rules_dir` and `history_dir` from scratch
- **AND** stores each file's mtime in `files_meta`

#### Scenario: Subsequent run — no files changed

- **WHEN** the plugin loads and all files in `rules_dir` have mtime ≤ stored mtime in `files_meta`
- **THEN** the indexer skips reindexing entirely
- **AND** startup completes in <100ms

#### Scenario: One rules file was edited

- **WHEN** the plugin loads and `rules/dict/БОЙ.md` has mtime newer than stored
- **THEN** the indexer reindexes only that file (delete old sections by source, insert new)
- **AND** updates the stored mtime in `files_meta`

### Requirement: WFRP term dictionary expands queries

The indexer SHALL maintain a dictionary mapping common WFRP-related Russian word stems to curated FTS5 query phrases. When extracting keywords from a player message, if a word stem matches a dictionary key, the corresponding FTS5 phrase SHALL be used instead of (or in addition to) the raw prefix wildcard.

#### Scenario: Player mentions "атакует"

- **WHEN** the `pre_llm_call` hook processes "Гюнтер атакует орка"
- **THEN** the stem "атак" matches the dictionary
- **AND** the FTS5 query includes `ближний бой атака OR дальний бой стрельба` (from dictionary) in addition to `атак*` (from keyword extraction)

#### Scenario: Player mentions magic

- **WHEN** the hook processes "Фелирон кастует заклинание огня"
- **THEN** the stem "каст" or "заклин" matches the dictionary
- **AND** the FTS5 query includes `магия заклинание концентрация ошибка`

### Requirement: `/wfrp-rag` slash command for manual control

The plugin SHALL register a `/wfrp-rag` slash command with three subcommands: `status` (print index stats), `reindex` (force full reindex), and `search <query>` (manual search).

#### Scenario: Check index status

- **WHEN** the user types `/wfrp-rag status`
- **THEN** the plugin prints: number of rules sections, number of history entries, index file size, last reindex timestamp

#### Scenario: Force full reindex

- **WHEN** the user types `/wfrp-rag reindex`
- **THEN** the plugin drops and recreates the FTS5 tables
- **AND** reindexes all files from scratch
- **AND** prints confirmation with section counts

#### Scenario: Manual search

- **WHEN** the user types `/wfrp-rag search критическая рана`
- **THEN** the plugin executes an FTS5 search and prints formatted results (title, source, snippet, score)

### Requirement: Configurable K and context limits

The plugin SHALL read configuration from Hermes' plugin config system (`metadata.hermes.config` in `plugin.yaml`), with defaults: `rules_k=3`, `history_k=2`, `max_context_chars=2000`, `rules_dir=./rules`, `history_dir=./history`.

#### Scenario: User increases rules_k via config

- **WHEN** the user sets `wfrp-rag.rules_k: 5` in `config.yaml`
- **THEN** the `pre_llm_call` hook injects up to 5 rule snippets per turn instead of 3

#### Scenario: Context exceeds max_context_chars

- **WHEN** the combined rules + history snippets exceed `max_context_chars` (default 2000)
- **THEN** the hook truncates the injected context to `max_context_chars` characters
- **AND** appends a "...[truncated]" marker

