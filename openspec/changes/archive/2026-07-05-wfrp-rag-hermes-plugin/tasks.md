## 1. Plugin scaffold

- [x] 1.1 Create `plugins/wfrp-rag/plugin.yaml` manifest with name, version, `provides_tools: [wfrp_rag_search]`, `provides_hooks: [pre_llm_call, post_llm_call, post_tool_call]`, and `metadata.hermes.config` keys (rules_dir, history_dir, rules_k, history_k, max_context_chars)
- [x] 1.2 Create empty `plugins/wfrp-rag/__init__.py` with stub `register(ctx)` function that logs "wfrp-rag plugin loaded"
- [x] 1.3 Add `plugins/wfrp-rag/data/` to `.gitignore` (index DB is runtime-generated, not committed)
- [x] 1.4 Verify plugin discovery: run `HERMES_PLUGINS_DEBUG=1 hermes plugins list` and confirm `wfrp-rag` appears (if Hermes available locally; otherwise document for VPS)

## 2. FTS5 indexer module

- [x] 2.1 Create `plugins/wfrp-rag/indexer.py` with `WfrpIndexer` class: `__init__(db_path, rules_dir, history_dir)`, opens SQLite connection, creates FTS5 tables if not exist (`rules_fts`, `history_fts`, `files_meta`)
- [x] 2.2 Implement `split_by_headers(content, default_title)` — parse markdown into sections by `#{1,6}` headers, return list of `{title, content}` dicts
- [x] 2.3 Implement `index_file(filepath, table)` — read file, split into sections, delete old rows by source, insert new rows, update `files_meta` mtime
- [x] 2.4 Implement `index_all(force=False)` — walk `rules_dir` and `history_dir`, compare mtimes against `files_meta`, reindex only changed files (or all if `force=True`)
- [x] 2.5 Implement `search_rules(query, k=5)` — FTS5 MATCH query on `rules_fts`, return `[{title, source, content, score}]` sorted by BM25
- [x] 2.6 Implement `search_history(query, k=5)` — same for `history_fts`, include `session_id` and `timestamp` in results
- [x] 2.7 Implement `add_history_entry(session_id, role, content, source="live")` — insert a single row into `history_fts` with current timestamp
- [x] 2.8 Define `WFRP_TERM_DICT` — Python dict mapping ~20 common WFRP stems to curated FTS5 query phrases (атак, провер, маги, молитв, крит, обремен, карьер, опыт, страх, инициатив, оружие, рана, заминка, преимущество, состояние, усталость, кровотеч, оглуш, торговл, путешеств)
- [x] 2.9 Implement `extract_keywords(message)` — regex `\b[а-яА-ЯёЁ]{4,}\b`, filter Russian stopwords (~30 words), return list of `prefix*` strings
- [x] 2.10 Implement `build_search_query(message)` — combine WFRP term dictionary matches + keyword extraction into a single FTS5 MATCH string joined by ` OR `
- [x] 2.11 Add `PRAGMA compile_options` check on init — if FTS5 not available, log error and set a `self.available = False` flag so hooks degrade gracefully

## 3. Hook handlers

- [x] 3.1 Implement `on_pre_llm_call(session_id, user_message, is_first_turn, **kwargs)` in `__init__.py` — call `indexer.build_search_query(user_message)`, search rules (k=rules_k) and history (k=history_k), format as context string, return `{"context": text}` or `None`
- [x] 3.2 Implement `on_post_llm_call(session_id, user_message, assistant_response, **kwargs)` — call `indexer.add_history_entry()` for both user and assistant messages
- [x] 3.3 Implement `on_post_tool_call(tool_name, args, result, **kwargs)` — check if `tool_name` in `("write_file", "patch")` and `args["path"]` is under `rules_dir` or `history_dir`; if so, call `indexer.index_file(path, table)`
- [x] 3.4 Wire all three hooks in `register(ctx)`: `ctx.register_hook("pre_llm_call", on_pre_llm_call)`, etc.
- [x] 3.5 Add error handling: all hooks wrapped in try/except, return `None` on failure, log to `~/.hermes/logs/agent.log`

## 4. On-demand search tool

- [x] 4.1 Create `plugins/wfrp-rag/schemas.py` with `WFRP_RAG_SEARCH` schema: name, description, parameters (query: string required, source: enum [rules, history, all] default all, k: integer default 5)
- [x] 4.2 Create `plugins/wfrp-rag/tools.py` with `handle_rag_search(args, **kwargs)` — parse args, call `indexer.search_rules` and/or `search_history`, return JSON string
- [x] 4.3 Register tool in `register(ctx)`: `ctx.register_tool(name="wfrp_rag_search", toolset="wfrp_rag", schema=schemas.WFRP_RAG_SEARCH, handler=tools.handle_rag_search)`

## 5. Slash command

- [x] 5.1 Implement `handle_wfrp_rag_command(raw_args)` — parse subcommand (`status`, `reindex`, `search <query>`)
- [x] 5.2 `status` subcommand: print rules section count, history entry count, index file size, last reindex time
- [x] 5.3 `reindex` subcommand: call `indexer.index_all(force=True)`, print confirmation
- [x] 5.4 `search` subcommand: call `indexer.search_rules(query, k=5)`, print formatted results
- [x] 5.5 Register in `register(ctx)`: `ctx.register_command("wfrp-rag", handler=handle_wfrp_rag_command, description="WFRP RAG index management")`

## 6. Context formatting

- [x] 6.1 Implement `format_context(rules_results, history_results, max_chars)` — build a formatted string with `📚 Правила:\n` section and `📖 История:\n` section, each entry as `1. [source] title: snippet`
- [x] 6.2 Truncate to `max_context_chars` with `...[truncated]` marker if exceeded
- [x] 6.3 Use FTS5 `snippet()` function in search queries for highlighted excerpts instead of full section content in `pre_llm_call` (full content only in `wfrp_rag_search` tool)

## 7. Config integration

- [x] 7.1 Read config values in `register(ctx)` from `ctx` or Hermes config system: `rules_dir`, `history_dir`, `rules_k`, `history_k`, `max_context_chars`
- [x] 7.2 Apply defaults if config keys missing: `./rules`, `./history`, 3, 2, 2000
- [x] 7.3 Pass resolved config to `WfrpIndexer` constructor

## 8. Tests (local, without Hermes)

- [x] 8.1 Create `plugins/wfrp-rag/tests/test_indexer.py` — test `split_by_headers()` with fixture markdown (multi-header file, no-header file, nested headers)
- [x] 8.2 Test `index_all()` on `rules/dict/` — verify section count > 0, verify `files_meta` populated
- [x] 8.3 Test `search_rules("атак")` — verify results include `dict/БОЙ.md` with BM25 score
- [x] 8.4 Test `search_rules("критическая рана")` — verify results include critical hit rules
- [x] 8.5 Test `extract_keywords("Гюнтер атакует орка")` — verify returns `["атак*", "орк*"]` (stopword "Гюнтер" filtered if <4 chars or in stoplist)
- [x] 8.6 Test `build_search_query("Гюнтер атакует орка")` — verify WFRP term dict expands "атак" to "ближний бой атака OR дальний бой стрельба"
- [x] 8.7 Test `add_history_entry()` + `search_history()` round-trip — insert a fake exchange, search for it, verify it's found
- [x] 8.8 Test incremental reindex: index file, modify mtime, reindex — verify only changed file is reprocessed
- [x] 8.9 Create `plugins/wfrp-rag/tests/test_hooks.py` — mock `pre_llm_call` / `post_llm_call` / `post_tool_call` kwargs, verify hook functions return correct shapes
- [x] 8.10 Run all tests: `cd plugins/wfrp-rag && python3 -m pytest tests/ -v`

## 9. Hook verification on Hermes (local or VPS)

- [ ] 9.1 Install plugin: copy `plugins/wfrp-rag/` to `~/.hermes/plugins/wfrp-rag/` on the target machine
- [ ] 9.2 Enable plugin: `hermes plugins enable wfrp-rag`
- [ ] 9.3 Start Hermes: `hermes chat` — verify "wfrp-rag" appears in plugin list and index builds on first load
- [ ] 9.4 Send a test message: "Гюнтер атакует орка" — verify RAG context is injected (check Hermes logs or LLM response for evidence of rules knowledge)
- [ ] 9.5 Send a follow-up message — verify `history_fts` now contains the previous exchange (check via `/wfrp-rag status`)
- [ ] 9.6 Test `/wfrp-rag status` — verify it prints section counts and index size
- [ ] 9.7 Test `/wfrp-rag search критическая рана` — verify formatted results
- [ ] 9.8 Test `/wfrp-rag reindex` — verify full reindex runs and counts update
- [ ] 9.9 Verify `wfrp_rag_search` tool is callable: ask the LLM "найди правила критических ран" and confirm it calls the tool

## 10. VPS deployment

- [ ] 10.1 Push the repo (with `plugins/wfrp-rag/`) to the VPS via git
- [ ] 10.2 On VPS: clone repo, symlink or copy `plugins/wfrp-rag/` to `~/.hermes/plugins/wfrp-rag/`
- [ ] 10.3 On VPS: `hermes plugins enable wfrp-rag`
- [ ] 10.4 On VPS: set `wfrp-rag.rules_dir` and `wfrp-rag.history_dir` in `~/.hermes/config.yaml` to point to the repo paths on the VPS
- [ ] 10.5 Start Hermes gateway on VPS: `hermes gateway start` — verify plugin loads and index builds
- [ ] 10.6 Send a test message via Telegram (or configured gateway) — verify RAG injection works end-to-end
- [ ] 10.7 Run a short test session (~5 turns) — verify `history_fts` grows and past exchanges are recalled in context
