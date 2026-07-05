"""Tests for Hermes hook handlers (mocked, no Hermes required)."""

import os
import sys

import pytest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from indexer import WfrpIndexer


@pytest.fixture
def setup_plugin(tmp_path):
    rules_dir = tmp_path / "rules"
    history_dir = tmp_path / "history"
    rules_dir.mkdir()
    history_dir.mkdir()
    (rules_dir / "test.md").write_text(
        "# Тест\n\n## Атака\nБлижний бой атака\n", encoding="utf-8"
    )
    db_path = tmp_path / "data" / "wfrp_index.db"
    indexer = WfrpIndexer(
        db_path=str(db_path),
        rules_dir=str(rules_dir),
        history_dir=str(history_dir),
    )
    indexer.index_all()

    import wfrp_rag_plugin as plugin  # type: ignore[import-not-found]
    plugin._indexer = indexer  # type: ignore[attr-defined]
    plugin._config = {  # type: ignore[attr-defined]
        "rules_dir": str(rules_dir),
        "history_dir": str(history_dir),
        "rules_k": 3,
        "history_k": 2,
        "max_context_chars": 2000,
    }

    yield plugin, indexer

    indexer.close()


class TestPreLlmCall:
    def test_combat_message_returns_context(self, setup_plugin):
        plugin, _ = setup_plugin
        result = plugin.on_pre_llm_call(
            session_id="s1",
            user_message="Гюнтер атакует орка мечом",
            is_first_turn=False,
        )
        assert result is not None
        assert "context" in result

    def test_trivial_message_returns_none(self, setup_plugin):
        plugin, _ = setup_plugin
        result = plugin.on_pre_llm_call(
            session_id="s1",
            user_message="ок",
            is_first_turn=False,
        )
        assert result is None

    def test_empty_message_returns_none(self, setup_plugin):
        plugin, _ = setup_plugin
        result = plugin.on_pre_llm_call(
            session_id="s1",
            user_message="",
            is_first_turn=False,
        )
        assert result is None


class TestPostLlmCall:
    def test_indexes_exchange(self, setup_plugin):
        plugin, indexer = setup_plugin
        plugin.on_post_llm_call(
            session_id="s1",
            user_message="Гюнтер бьёт орка",
            assistant_response="Орк получает 12 урона",
        )
        results = indexer.search_history("орк*", k=5)
        assert len(results) >= 1


class TestPostToolCall:
    def test_reindex_history_file(self, setup_plugin, tmp_path):
        plugin, indexer = setup_plugin
        history_file = tmp_path / "history" / "session_001.md"
        history_file.write_text("# Сессия 1\n\nБой в таверне", encoding="utf-8")
        plugin.on_post_tool_call(
            tool_name="write_file",
            args={"path": str(history_file)},
            result="ok",
        )
        results = indexer.search_history("таверн*", k=5)
        assert len(results) >= 1

    def test_ignores_unrelated_file(self, setup_plugin, tmp_path):
        plugin, indexer = setup_plugin
        plugin.on_post_tool_call(
            tool_name="write_file",
            args={"path": str(tmp_path / "notes.md")},
            result="ok",
        )
        assert len(indexer.search_history("notes", k=5)) == 0

    def test_ignores_non_write_tools(self, setup_plugin):
        plugin, _ = setup_plugin
        result = plugin.on_post_tool_call(
            tool_name="read_file",
            args={"path": "/some/path"},
            result="ok",
        )
        assert result is None
