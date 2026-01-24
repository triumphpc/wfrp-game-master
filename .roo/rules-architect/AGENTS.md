# Project Architecture Rules (Non-Obvious Only)

## Dual-Role AI Architecture
- **Simultaneous roles**: AI must function as both Game Master (narrative, rulings) and Rules Checker (validation, calculations)
- **Architectural enforcement**: This dual role is explicitly designed in WARP.md and CLAUDE.md
- **Rule lookup integration**: RAG-MCP-SERVER is required for semantic search; cannot be bypassed

## Data Flow and Validation
- **Character validation pipeline**: Must check against `ЧАСТЫЕ_ОШИБКИ.md` and `КАРЬЕРЫ_ПРАВИЛЬНЫЕ_СХЕМЫ.md`
- **Rule source hierarchy**: Corrected schemes > WARP.md > text rules (which contain errors)
- **Session data organization**: Campaigns in `history/%game_name%/` with strict timestamp naming

## System Constraints
- **Python-only environment**: No Node.js, npm, or package.json; all utilities are pure Python scripts
- **UTF-8 with Cyrillic**: All text processing must handle Russian characters properly
- **PDF generation dependency**: `generate_character_cards.py` uses reportlab with custom fonts

## Component Coupling
- **Character cards**: Generated from markdown templates in `characters/` directory
- **Rule documents**: Split from monolithic PDF into 12 chapters via `split_rules.py`
- **Game history**: Session files reference characters in campaign-specific subdirectories

## Performance Considerations
- **RAG-MCP-SERVER**: Semantic search is computationally intensive but required for accurate rule lookups
- **PDF processing**: PyMuPDF extraction patterns are optimized for WFRP rulebook structure
- **Character validation**: Must be fast enough for real-time gameplay assistance