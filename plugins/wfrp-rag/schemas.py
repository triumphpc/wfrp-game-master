"""Tool schemas for the WFRP RAG plugin."""

WFRP_RAG_SEARCH = {
    "name": "wfrp_rag_search",
    "description": (
        "Глубокий поиск по правилам WFRP 4E и истории сессий. "
        "Возвращает полные тексты релевантных секций с BM25 ранжированием. "
        "Используй, когда автоинжектируемого контекста недостаточно — "
        "для точных механик, таблиц, прошлых событий."
    ),
    "parameters": {
        "type": "object",
        "properties": {
            "query": {
                "type": "string",
                "description": "Поисковый запрос (например: 'критическая рана таблица голова')",
            },
            "source": {
                "type": "string",
                "enum": ["rules", "history", "all"],
                "default": "all",
                "description": "Где искать: rules — только правила, history — только история, all — везде",
            },
            "k": {
                "type": "integer",
                "default": 5,
                "description": "Количество результатов",
            },
        },
        "required": ["query"],
    },
}
