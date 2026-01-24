# Project Documentation Rules (Non-Obvious Only)

## Rule Sources and Conflicts
- **Primary rule source**: Use `rules/` directory split into 12 chapters, not the monolithic `WFRPG4E.ru.FULL.md`
- **Career scheme errors**: Text rules in `rules/03_классы_и_карьеры.md` contain errors; always use `КАРЬЕРЫ_ПРАВИЛЬНЫЕ_СХЕМЫ.md`
- **Quick reference**: `ТАБЛИЦЫ_БЫСТРЫЙ_ДОСТУП.md` provides condensed tables for fast lookup during gameplay

## Character Creation Documentation
- **Common mistakes**: `ЧАСТЫЕ_ОШИБКИ.md` documents frequent calculation errors (Toughened talent, bonus XP, duplicate skills)
- **Character templates**: Active characters in `characters/` follow specific markdown structure with Russian headers
- **Historical characters**: Archived characters are in `history/%game_name%/characters/` with session context

## AI Assistant Documentation
- **Dual-role architecture**: AI must act as both Game Master and Rules Checker simultaneously (see WARP.md and CLAUDE.md)
- **RAG-MCP-SERVER requirement**: Semantic search is mandatory for rule lookups; direct file searches are insufficient
- **Game session structure**: Sessions documented in `history/%game_name%/` with timestamped markdown files

## Project Organization
- **Python utilities**: Three main scripts for PDF generation, PDF-to-markdown conversion, and rule splitting
- **No package.json**: Pure Python environment without npm scripts or Node.js dependencies
- **Russian language**: All user-facing content, comments, and documentation are in Russian with Cyrillic encoding