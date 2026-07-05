"""WFRP FTS5 Indexer — SQLite full-text search for WFRP rules and session history."""

import logging
import os
import re
import sqlite3
import time
from datetime import datetime
from pathlib import Path

logger = logging.getLogger(__name__)

RUSSIAN_STOPWORDS = frozenset({
    "без", "более", "бы", "был", "была", "были", "было", "быть", "в", "вам",
    "вас", "весь", "во", "вот", "все", "всего", "всех", "вы", "где", "да",
    "даже", "для", "до", "его", "ее", "если", "есть", "еще", "же", "за",
    "здесь", "и", "из", "или", "им", "их", "к", "как", "ко", "когда", "ли",
    "либо", "мне", "может", "мы", "на", "над", "наш", "не", "него", "нее",
    "нет", "ни", "них", "но", "ну", "о", "об", "однако", "он", "она", "они",
    "оно", "от", "по", "под", "при", "со", "так", "также", "там", "те", "то",
    "того", "только", "том", "ты", "у", "уже", "хотя", "чего", "чей", "чем",
    "что", "чтобы", "чье", "чья", "эта", "эти", "это", "этого", "этой", "этот",
})

WFRP_TERM_DICT = {
    "атак": "ближний бой атака OR дальний бой стрельба",
    "провер": "проверка навык успех провал уровень",
    "маги": "магия заклинание концентрация ошибка",
    "заклин": "магия заклинание концентрация ошибка",
    "молитв": "молитва божество гнев вера",
    "крит": "критическая рана таблица",
    "обремен": "обременение лимит вес перегруз",
    "карьер": "карьера ступень развитие ранг",
    "опыт": "опыт XP развитие характеристика",
    "страх": "страх ужас психология хладнокровие",
    "инициатив": "инициатива порядок раунд ход",
    "оружи": "оружие качество грозное пронзающее",
    "ран": "рана урон HP выносливость",
    "заминк": "заминка fumble несчастие",
    "преимущ": "преимущество advantage боевой",
    "состоя": "состояние кровотечение оглушение усталость",
    "усталост": "усталость штраф проверка",
    "кровотеч": "кровотечение HP раунд состояние",
    "оглуш": "оглушение состояние проверка",
    "торговл": "торговля экономика цена монеты",
    "путешеств": "путешествие расстояние маршрут карта",
    "брон": "броня доспех очки защита",
    "щит": "щит защита парирование",
    "парир": "парирование защита реакция",
    "уклон": "уклонение защита реакция",
    "натиск": "натиск charge атака разбег",
    "преимущ": "преимущество advantage боевой",
    "художеств": "талант развитие персонаж",
    "навык": "навык проверка характеристика",
    "характер": "характеристика сила выносливость ловкость",
    "раса": "раса человек эльф гном полурослик",
    "класс": "класс карьера ступень",
}

_KEYWORD_RE = re.compile(r"\b[а-яА-ЯёЁ]{4,}\b")


def split_by_headers(content: str, default_title: str = "") -> list[dict]:
    sections = []
    lines = content.split("\n")
    current_title = default_title or Path("untitled").stem
    current_content: list[str] = []
    for line in lines:
        header_match = re.match(r"^(#{1,6})\s+(.+)$", line)
        if header_match:
            if current_content:
                sections.append({
                    "title": current_title,
                    "content": "\n".join(current_content).strip(),
                })
            current_title = header_match.group(2).strip()
            current_content = [line]
        else:
            current_content.append(line)
    if current_content:
        sections.append({
            "title": current_title,
            "content": "\n".join(current_content).strip(),
        })
    if len(sections) <= 1:
        return [{"title": default_title or current_title, "content": content}]
    return sections


def extract_keywords(message: str) -> list[str]:
    words = _KEYWORD_RE.findall(message.lower())
    keywords = [w for w in words if w not in RUSSIAN_STOPWORDS]
    return [w[:5] + "*" for w in keywords if len(w) >= 4]


def build_search_query(message: str) -> str:
    parts: list[str] = []
    msg_lower = message.lower()
    for stem, phrase in WFRP_TERM_DICT.items():
        if stem in msg_lower:
            parts.append(f"({phrase})")
    parts.extend(extract_keywords(message))
    if not parts:
        return ""
    return " OR ".join(parts)


class WfrpIndexer:
    def __init__(self, db_path: str, rules_dir: str, history_dir: str):
        self.db_path = str(db_path)
        self.rules_dir = str(rules_dir)
        self.history_dir = str(history_dir)
        self.available = False
        self._conn: sqlite3.Connection | None = None
        self._init_db()

    def _init_db(self):
        os.makedirs(os.path.dirname(self.db_path), exist_ok=True)
        self._conn = sqlite3.connect(self.db_path)
        self._conn.execute("PRAGMA journal_mode=WAL")
        try:
            cursor = self._conn.execute("PRAGMA compile_options")
            options = [row[0] for row in cursor.fetchall()]
            if not any("ENABLE_FTS5" in opt for opt in options):
                logger.error("FTS5 not available in SQLite — RAG disabled")
                self.available = False
                return
        except sqlite3.Error:
            pass
        self._conn.executescript(
            """
            CREATE VIRTUAL TABLE IF NOT EXISTS rules_fts USING fts5(
                title, source, content, tokenize='unicode61'
            );
            CREATE VIRTUAL TABLE IF NOT EXISTS history_fts USING fts5(
                campaign, session_id, timestamp, role, title, source, content, tokenize='unicode61'
            );
            CREATE TABLE IF NOT EXISTS files_meta (
                path TEXT PRIMARY KEY, mtime REAL, section_count INTEGER
            );
            """
        )
        self._conn.commit()
        self.available = True
        logger.info("WFRP RAG index ready: %s", self.db_path)

    def _relative_source(self, filepath: str, base_dir: str) -> str:
        try:
            return os.path.relpath(filepath, base_dir)
        except ValueError:
            return os.path.basename(filepath)

    def _detect_campaign(self, filepath: str, base_dir: str) -> str:
        try:
            rel = os.path.relpath(filepath, base_dir)
            parts = rel.split(os.sep)
            if len(parts) > 1:
                return parts[0]
        except ValueError:
            pass
        return "general"

    def index_file(self, filepath: str, table: str, base_dir: str | None = None) -> int:
        if not self.available:
            return 0
        if base_dir is None:
            base_dir = self.rules_dir if table == "rules_fts" else self.history_dir
        try:
            with open(filepath, "r", encoding="utf-8") as f:
                content = f.read()
        except OSError as e:
            logger.warning("Cannot read %s: %s", filepath, e)
            return 0
        source = self._relative_source(filepath, base_dir)
        sections = split_by_headers(content, default_title=Path(filepath).stem)
        cursor = self._conn.cursor()
        cursor.execute(f"DELETE FROM {table} WHERE source = ?", (source,))
        for section in sections:
            if table == "rules_fts":
                cursor.execute(
                    f"INSERT INTO {table} (title, source, content) VALUES (?, ?, ?)",
                    (section["title"], source, section["content"]),
                )
            else:
                campaign = self._detect_campaign(filepath, base_dir)
                cursor.execute(
                    f"INSERT INTO {table} (campaign, session_id, timestamp, role, title, source, content) VALUES (?, ?, ?, ?, ?, ?, ?)",
                    (campaign, "file", "", "file", section["title"], source, section["content"]),
                )
        mtime = os.path.getmtime(filepath)
        cursor.execute(
            "INSERT OR REPLACE INTO files_meta (path, mtime, section_count) VALUES (?, ?, ?)",
            (filepath, mtime, len(sections)),
        )
        self._conn.commit()
        logger.debug("Indexed %s: %d sections", source, len(sections))
        return len(sections)

    def index_all(self, force: bool = False) -> tuple[int, int]:
        if not self.available:
            return (0, 0)
        rules_count = self._index_dir(self.rules_dir, "rules_fts", force)
        history_count = self._index_dir(self.history_dir, "history_fts", force)
        logger.info(
            "Index complete: %d rules sections, %d history sections",
            rules_count,
            history_count,
        )
        return (rules_count, history_count)

    def _index_dir(self, dir_path: str, table: str, force: bool) -> int:
        if not os.path.isdir(dir_path):
            logger.warning("Directory not found: %s", dir_path)
            return 0
        total = 0
        for root, _dirs, files in os.walk(dir_path):
            for file in files:
                if not file.endswith(".md"):
                    continue
                filepath = os.path.join(root, file)
                if not force:
                    cursor = self._conn.cursor()
                    cursor.execute(
                        "SELECT mtime FROM files_meta WHERE path = ?", (filepath,)
                    )
                    row = cursor.fetchone()
                    if row and row[0] >= os.path.getmtime(filepath):
                        continue
                total += self.index_file(filepath, table, base_dir=dir_path)
        return total

    def search_rules(self, query: str, k: int = 5) -> list[dict]:
        return self._search("rules_fts", query, k)

    def search_history(self, query: str, k: int = 5, campaign: str | None = None) -> list[dict]:
        return self._search("history_fts", query, k, campaign=campaign)

    def _search(self, table: str, query: str, k: int, campaign: str | None = None) -> list[dict]:
        if not self.available or not query.strip():
            return []
        fts_query = build_search_query(query) if " OR " not in query else query
        if not fts_query:
            fts_query = query
        cursor = self._conn.cursor()
        try:
            if table == "rules_fts":
                cursor.execute(
                    f"""
                    SELECT title, source, snippet({table}, 2, '«', '»', '…', 20) as snippet,
                           bm25({table}) as score
                    FROM {table} WHERE {table} MATCH ?
                    ORDER BY score LIMIT ?
                    """,
                    (fts_query, k),
                )
            else:
                if campaign:
                    cursor.execute(
                        f"""
                        SELECT campaign, session_id, timestamp, role, title, source,
                               snippet({table}, 6, '«', '»', '…', 20) as snippet,
                               bm25({table}) as score
                        FROM {table} WHERE {table} MATCH ? AND campaign = ?
                        ORDER BY score LIMIT ?
                        """,
                        (fts_query, campaign, k),
                    )
                else:
                    cursor.execute(
                        f"""
                        SELECT campaign, session_id, timestamp, role, title, source,
                               snippet({table}, 6, '«', '»', '…', 20) as snippet,
                               bm25({table}) as score
                        FROM {table} WHERE {table} MATCH ?
                        ORDER BY score LIMIT ?
                        """,
                        (fts_query, k),
                    )
            rows = cursor.fetchall()
        except sqlite3.OperationalError as e:
            logger.warning("FTS5 search error: %s (query: %s)", e, fts_query)
            return []
        results = []
        for row in rows:
            if table == "rules_fts":
                results.append({
                    "title": row[0],
                    "source": row[1],
                    "snippet": row[2],
                    "score": row[3],
                })
            else:
                results.append({
                    "campaign": row[0],
                    "session_id": row[1],
                    "timestamp": row[2],
                    "role": row[3],
                    "title": row[4],
                    "source": row[5],
                    "snippet": row[6],
                    "score": row[7],
                })
        return results

    def search_rules_full(self, query: str, k: int = 5) -> list[dict]:
        return self._search_full("rules_fts", query, k)

    def search_history_full(self, query: str, k: int = 5, campaign: str | None = None) -> list[dict]:
        return self._search_full("history_fts", query, k, campaign=campaign)

    def _search_full(self, table: str, query: str, k: int, campaign: str | None = None) -> list[dict]:
        if not self.available or not query.strip():
            return []
        fts_query = build_search_query(query) if " OR " not in query else query
        if not fts_query:
            fts_query = query
        cursor = self._conn.cursor()
        try:
            if table == "rules_fts":
                cursor.execute(
                    f"""
                    SELECT title, source, content, bm25({table}) as score
                    FROM {table} WHERE {table} MATCH ?
                    ORDER BY score LIMIT ?
                    """,
                    (fts_query, k),
                )
            else:
                if campaign:
                    cursor.execute(
                        f"""
                        SELECT campaign, session_id, timestamp, role, title, source, content,
                               bm25({table}) as score
                        FROM {table} WHERE {table} MATCH ? AND campaign = ?
                        ORDER BY score LIMIT ?
                        """,
                        (fts_query, campaign, k),
                    )
                else:
                    cursor.execute(
                        f"""
                        SELECT campaign, session_id, timestamp, role, title, source, content,
                               bm25({table}) as score
                        FROM {table} WHERE {table} MATCH ?
                        ORDER BY score LIMIT ?
                        """,
                        (fts_query, k),
                    )
            rows = cursor.fetchall()
        except sqlite3.OperationalError as e:
            logger.warning("FTS5 search error: %s (query: %s)", e, fts_query)
            return []
        results = []
        for row in rows:
            if table == "rules_fts":
                results.append({
                    "title": row[0],
                    "source": row[1],
                    "content": row[2],
                    "score": row[3],
                })
            else:
                results.append({
                    "campaign": row[0],
                    "session_id": row[1],
                    "timestamp": row[2],
                    "role": row[3],
                    "title": row[4],
                    "source": row[5],
                    "content": row[6],
                    "score": row[7],
                })
        return results

    def add_history_entry(
        self, session_id: str, role: str, content: str,
        source: str = "live", campaign: str = "general",
    ) -> None:
        if not self.available:
            return
        timestamp = datetime.now().isoformat()
        self._conn.execute(
            "INSERT INTO history_fts (campaign, session_id, timestamp, role, title, source, content) VALUES (?, ?, ?, ?, ?, ?, ?)",
            (campaign, session_id, timestamp, role, "", source, content),
        )
        self._conn.commit()

    def get_stats(self) -> dict:
        if not self.available:
            return {"available": False}
        cursor = self._conn.cursor()
        cursor.execute("SELECT COUNT(*) FROM rules_fts")
        rules_count = cursor.fetchone()[0]
        cursor.execute("SELECT COUNT(*) FROM history_fts")
        history_count = cursor.fetchone()[0]
        cursor.execute("SELECT campaign, COUNT(*) FROM history_fts GROUP BY campaign")
        campaigns = {row[0]: row[1] for row in cursor.fetchall()}
        db_size = os.path.getsize(self.db_path) if os.path.exists(self.db_path) else 0
        cursor.execute("SELECT MAX(mtime) FROM files_meta")
        row = cursor.fetchone()
        last_index = datetime.fromtimestamp(row[0]).isoformat() if row[0] else None
        return {
            "available": True,
            "rules_sections": rules_count,
            "history_entries": history_count,
            "campaigns": campaigns,
            "db_size_bytes": db_size,
            "last_index": last_index,
        }

    def list_campaigns(self) -> list[str]:
        if not self.available:
            return []
        cursor = self._conn.cursor()
        cursor.execute("SELECT DISTINCT campaign FROM history_fts ORDER BY campaign")
        return [row[0] for row in cursor.fetchall()]

    def close(self):
        if self._conn:
            self._conn.close()
            self._conn = None


def format_context(
    rules_results: list[dict],
    history_results: list[dict],
    max_chars: int = 2000,
) -> str | None:
    parts: list[str] = []
    if rules_results:
        lines = ["📚 Правила:"]
        for i, r in enumerate(rules_results, 1):
            lines.append(f"  {i}. [{r['source']}] {r['title']}: {r.get('snippet', r.get('content', '')[:200])}")
        parts.append("\n".join(lines))
    if history_results:
        lines = ["📖 История:"]
        for i, r in enumerate(history_results, 1):
            lines.append(f"  {i}. [{r.get('source', '?')}] {r.get('snippet', r.get('content', '')[:200])}")
        parts.append("\n".join(lines))
    if not parts:
        return None
    text = "\n\n".join(parts)
    if len(text) > max_chars:
        text = text[: max_chars - 20] + "\n...[truncated]"
    return text
