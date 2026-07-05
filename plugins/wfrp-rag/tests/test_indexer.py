"""Tests for the WFRP RAG indexer module."""

import os
import sys
import tempfile
import time

import pytest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from indexer import (
    WfrpIndexer,
    build_search_query,
    extract_keywords,
    format_context,
    split_by_headers,
)


@pytest.fixture
def tmp_indexer(tmp_path):
    rules_dir = tmp_path / "rules"
    history_dir = tmp_path / "history"
    rules_dir.mkdir()
    history_dir.mkdir()
    db_path = tmp_path / "data" / "wfrp_index.db"
    indexer = WfrpIndexer(
        db_path=str(db_path),
        rules_dir=str(rules_dir),
        history_dir=str(history_dir),
    )
    yield indexer
    indexer.close()


@pytest.fixture
def sample_rules(tmp_path):
    rules_dir = tmp_path / "rules" / "dict"
    rules_dir.mkdir(parents=True)
    (rules_dir / "БОЙ.md").write_text(
        """# Бой

## Ближний бой

Встречная проверка: Ближний бой атакующего vs Ближний бой защищающегося.
Натиск: +10 к атаке, +1 УУ.

## Дальний бой

Простая проверка: Баллистическое мастерство.
Дистанция: короткая 0, дальняя -10.

## Критические раны

Критическое попадание: успех + дубль.
Бросок 1d100 + избыточный урон по таблице критических ран.
""",
        encoding="utf-8",
    )
    (rules_dir / "ПРОВЕРКИ.md").write_text(
        """# Проверки

## Уровень успеха

УУ = (целевое число - бросок) / 10

## Гарантированный провал

Бросок 96-00 — всегда провал.
""",
        encoding="utf-8",
    )
    return rules_dir


class TestSplitByHeaders:
    def test_multi_header(self):
        content = "# Title\n\nIntro\n\n## Section A\nContent A\n\n## Section B\nContent B"
        sections = split_by_headers(content, default_title="file")
        assert len(sections) >= 3
        assert sections[0]["title"] == "Title"
        assert sections[1]["title"] == "Section A"
        assert "Content A" in sections[1]["content"]

    def test_no_header(self):
        content = "Just some text without headers."
        sections = split_by_headers(content, default_title="nofile")
        assert len(sections) == 1
        assert sections[0]["title"] == "nofile"
        assert sections[0]["content"] == content

    def test_nested_headers(self):
        content = "# Top\n\n## Mid\n\n### Deep\nDeep content"
        sections = split_by_headers(content, default_title="file")
        assert len(sections) >= 3
        titles = [s["title"] for s in sections]
        assert "Top" in titles
        assert "Mid" in titles
        assert "Deep" in titles


class TestIndexAll:
    def test_index_populates_sections(self, tmp_path, sample_rules):
        db_path = tmp_path / "data" / "wfrp_index.db"
        indexer = WfrpIndexer(
            db_path=str(db_path),
            rules_dir=str(tmp_path / "rules"),
            history_dir=str(tmp_path / "history"),
        )
        try:
            rules_count, _ = indexer.index_all()
            assert rules_count > 0
            stats = indexer.get_stats()
            assert stats["rules_sections"] > 0
        finally:
            indexer.close()

    def test_incremental_reindex(self, tmp_path, sample_rules):
        db_path = tmp_path / "data" / "wfrp_index.db"
        indexer = WfrpIndexer(
            db_path=str(db_path),
            rules_dir=str(tmp_path / "rules"),
            history_dir=str(tmp_path / "history"),
        )
        try:
            indexer.index_all()
            first_count, _ = indexer.index_all()
            assert first_count == 0  # no changes

            time.sleep(0.1)
            (tmp_path / "rules" / "dict" / "БОЙ.md").write_text(
                "# Updated\n\nNew content here", encoding="utf-8"
            )
            second_count, _ = indexer.index_all()
            assert second_count > 0  # changed file reindexed
        finally:
            indexer.close()


class TestSearchRules:
    def test_search_attack(self, tmp_path, sample_rules):
        db_path = tmp_path / "data" / "wfrp_index.db"
        indexer = WfrpIndexer(
            db_path=str(db_path),
            rules_dir=str(tmp_path / "rules"),
            history_dir=str(tmp_path / "history"),
        )
        try:
            indexer.index_all()
            results = indexer.search_rules("атак*", k=3)
            assert len(results) > 0
            assert any("БОЙ" in r["source"] for r in results)
        finally:
            indexer.close()

    def test_search_critical(self, tmp_path, sample_rules):
        db_path = tmp_path / "data" / "wfrp_index.db"
        indexer = WfrpIndexer(
            db_path=str(db_path),
            rules_dir=str(tmp_path / "rules"),
            history_dir=str(tmp_path / "history"),
        )
        try:
            indexer.index_all()
            results = indexer.search_rules("критическая рана", k=5)
            assert len(results) > 0
        finally:
            indexer.close()


class TestKeywords:
    def test_extract_basic(self):
        kws = extract_keywords("Гюнтер атакует орка двуручным мечом")
        assert "атаку*" in kws
        assert "орк*" in kws

    def test_build_query_expands_wfrp_terms(self):
        query = build_search_query("Гюнтер атакует орка")
        assert "ближний бой атака" in query or "атак*" in query


class TestHistoryRoundTrip:
    def test_add_and_search(self, tmp_indexer):
        tmp_indexer.add_history_entry("s1", "user", "Гюнтер бьёт орка мечом", source="test")
        tmp_indexer.add_history_entry("s1", "assistant", "Орк получает 12 урона", source="test")
        results = tmp_indexer.search_history("орк*", k=5)
        assert len(results) >= 1


class TestFormatContext:
    def test_with_results(self):
        rules = [{"source": "БОЙ.md", "title": "Ближний бой", "snippet": "встречная проверка"}]
        history = [{"source": "s1", "snippet": "бой в таверне"}]
        text = format_context(rules, history, max_chars=2000)
        assert "Правила" in text
        assert "История" in text

    def test_empty(self):
        assert format_context([], []) is None

    def test_truncation(self):
        rules = [{"source": "x.md", "title": "T", "snippet": "x" * 100}] * 50
        text = format_context(rules, [], max_chars=200)
        assert "[truncated]" in text
