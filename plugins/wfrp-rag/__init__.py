"""WFRP RAG plugin — mandatory rules + history context injection via FTS5."""

import logging
import os
from pathlib import Path

from . import schemas, tools
from .indexer import WfrpIndexer, build_search_query, format_context

logger = logging.getLogger(__name__)

_indexer: WfrpIndexer | None = None
_config: dict = {}

DEFAULTS = {
    "rules_dir": "./rules",
    "history_dir": "./history",
    "rules_k": 3,
    "history_k": 2,
    "max_context_chars": 2000,
    "current_campaign": None,
}


def _resolve_config(ctx) -> dict:
    cfg = dict(DEFAULTS)
    try:
        hermes_cfg = ctx.config if hasattr(ctx, "config") else {}
    except Exception:
        hermes_cfg = {}
    for key in DEFAULTS:
        full_key = f"wfrp-rag.{key}"
        if full_key in hermes_cfg:
            val = hermes_cfg[full_key]
            if key in ("rules_k", "history_k", "max_context_chars"):
                cfg[key] = int(val)
            else:
                cfg[key] = val if val else None
    return cfg


def on_pre_llm_call(session_id: str, user_message: str, is_first_turn: bool = False, **kwargs):
    if _indexer is None or not _indexer.available:
        return None
    if not user_message or len(user_message.strip()) < 3:
        return None
    try:
        query = build_search_query(user_message)
        if not query:
            return None
        rules_k = _config.get("rules_k", 3)
        history_k = _config.get("history_k", 2)
        max_chars = _config.get("max_context_chars", 2000)
        campaign = _config.get("current_campaign")

        rules_results = _indexer.search_rules(query, k=rules_k)
        history_results = _indexer.search_history(query, k=history_k, campaign=campaign)

        context = format_context(rules_results, history_results, max_chars=max_chars)
        if context:
            return {"context": context}
        return None
    except Exception as e:
        logger.warning("pre_llm_call error: %s", e, exc_info=True)
        return None


def on_post_llm_call(
    session_id: str,
    user_message: str,
    assistant_response: str,
    **kwargs,
):
    if _indexer is None or not _indexer.available:
        return None
    try:
        campaign = _config.get("current_campaign", "general") or "general"
        if user_message:
            _indexer.add_history_entry(session_id, "user", user_message, source="live", campaign=campaign)
        if assistant_response:
            _indexer.add_history_entry(
                session_id, "assistant", assistant_response, source="live", campaign=campaign
            )
    except Exception as e:
        logger.warning("post_llm_call error: %s", e, exc_info=True)
    return None


def on_post_tool_call(
    tool_name: str,
    args: dict,
    result: str,
    task_id: str = "",
    **kwargs,
):
    if _indexer is None or not _indexer.available:
        return None
    if tool_name not in ("write_file", "patch", "file_edit"):
        return None
    try:
        filepath = args.get("path", "") or args.get("file_path", "")
        if not filepath:
            return None
        rules_dir = _config.get("rules_dir", "./rules")
        history_dir = _config.get("history_dir", "./history")
        abs_path = os.path.abspath(filepath)
        abs_rules = os.path.abspath(rules_dir)
        abs_history = os.path.abspath(history_dir)

        if abs_path.startswith(abs_rules):
            _indexer.index_file(filepath, "rules_fts", base_dir=rules_dir)
            logger.debug("Reindexed rules file: %s", filepath)
        elif abs_path.startswith(abs_history):
            _indexer.index_file(filepath, "history_fts", base_dir=history_dir)
            logger.debug("Reindexed history file: %s", filepath)
    except Exception as e:
        logger.warning("post_tool_call error: %s", e, exc_info=True)
    return None


def handle_wfrp_rag_command(raw_args: str) -> str:
    if _indexer is None:
        return "WFRP RAG plugin not loaded."
    parts = raw_args.strip().split(None, 1)
    subcommand = parts[0].lower() if parts else "status"
    sub_args = parts[1] if len(parts) > 1 else ""

    if subcommand == "status":
        stats = _indexer.get_stats()
        if not stats.get("available"):
            return "RAG index unavailable (FTS5 not compiled?)"
        size_kb = stats["db_size_bytes"] / 1024
        current = _config.get("current_campaign") or "(all)"
        campaigns = stats.get("campaigns", {})
        camp_lines = "\n".join(f"    {c}: {n} entries" for c, n in sorted(campaigns.items()))
        return (
            f"WFRP RAG Index\n"
            f"  Rules sections: {stats['rules_sections']}\n"
            f"  History entries: {stats['history_entries']}\n"
            f"  Current campaign: {current}\n"
            f"  Campaigns:\n{camp_lines}\n"
            f"  Index size: {size_kb:.1f} KB\n"
            f"  Last index: {stats.get('last_index', 'never')}\n"
            f"  DB path: {_indexer.db_path}"
        )

    if subcommand == "campaign":
        if not sub_args:
            campaigns = _indexer.list_campaigns()
            current = _config.get("current_campaign") or "(all)"
            return f"Current campaign: {current}\nAvailable: {', '.join(campaigns) if campaigns else '(none)'}"
        camp = sub_args.strip()
        _config["current_campaign"] = camp
        tools.set_campaign(camp)
        return f"Campaign set to: {camp}"

    if subcommand == "reindex":
        rules, history = _indexer.index_all(force=True)
        return f"Reindexed: {rules} rules sections, {history} history sections"

    if subcommand == "search":
        if not sub_args:
            return "Usage: /wfrp-rag search <query>"
        campaign = _config.get("current_campaign")
        results = _indexer.search_rules(sub_args, k=5)
        hist_results = _indexer.search_history(sub_args, k=3, campaign=campaign)
        if not results and not hist_results:
            return "No results found."
        lines = [f"Found {len(results)} rules + {len(hist_results)} history results:"]
        for i, r in enumerate(results, 1):
            lines.append(f"\n{i}. [{r['source']}] {r['title']}")
            lines.append(f"   Score: {r['score']:.2f}")
            lines.append(f"   {r.get('snippet', '')}")
        for i, r in enumerate(hist_results, 1):
            lines.append(f"\nH{i}. [{r.get('campaign', '?')}/{r.get('source', '?')}] {r.get('title', '')}")
            lines.append(f"   Score: {r['score']:.2f}")
            lines.append(f"   {r.get('snippet', '')}")
        return "\n".join(lines)

    return "Usage: /wfrp-rag [status|campaign <name>|reindex|search <query>]"


def register(ctx):
    global _indexer, _config
    logger.info("wfrp-rag plugin loaded")

    _config = _resolve_config(ctx)

    plugin_dir = Path(__file__).parent
    db_path = plugin_dir / "data" / "wfrp_index.db"

    _indexer = WfrpIndexer(
        db_path=str(db_path),
        rules_dir=_config["rules_dir"],
        history_dir=_config["history_dir"],
    )

    if _indexer.available:
        rules_count, history_count = _indexer.index_all()
        logger.info(
            "WFRP RAG indexed: %d rules, %d history sections",
            rules_count,
            history_count,
        )

    tools.set_indexer(_indexer)
    tools.set_campaign(_config.get("current_campaign"))

    ctx.register_hook("pre_llm_call", on_pre_llm_call)
    ctx.register_hook("post_llm_call", on_post_llm_call)
    ctx.register_hook("post_tool_call", on_post_tool_call)

    ctx.register_tool(
        name="wfrp_rag_search",
        toolset="wfrp_rag",
        schema=schemas.WFRP_RAG_SEARCH,
        handler=tools.handle_rag_search,
    )

    ctx.register_command(
        "wfrp-rag",
        handler=handle_wfrp_rag_command,
        description="WFRP RAG index management (status, reindex, search)",
    )
