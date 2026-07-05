"""Tool handlers for the WFRP RAG plugin."""

import json

from .indexer import WfrpIndexer


def handle_rag_search(args: dict, **kwargs) -> str:
    idx = _get_indexer(kwargs)
    if idx is None:
        return json.dumps({"error": "RAG index not available"}, ensure_ascii=False)

    indexer: WfrpIndexer = idx

    query = args.get("query", "").strip()
    source = args.get("source", "all")
    k = args.get("k", 5)

    if not query:
        return json.dumps({"error": "query is required"}, ensure_ascii=False)

    results: list[dict] = []

    if source in ("rules", "all"):
        rules = indexer.search_rules_full(query, k=k)
        for r in rules:
            r["index"] = "rules"
        results.extend(rules)

    if source in ("history", "all"):
        campaign = _get_campaign()
        history = indexer.search_history_full(query, k=k, campaign=campaign)
        for r in history:
            r["index"] = "history"
        results.extend(history)

    return json.dumps(
        {"query": query, "count": len(results), "results": results},
        ensure_ascii=False,
        indent=2,
    )


_indexer: WfrpIndexer | None = None
_campaign: str | None = None


def set_indexer(indexer: WfrpIndexer) -> None:
    global _indexer
    _indexer = indexer


def set_campaign(campaign: str | None) -> None:
    global _campaign
    _campaign = campaign


def _get_campaign() -> str | None:
    return _campaign


def _get_indexer(kwargs: dict) -> WfrpIndexer | None:
    return _indexer
