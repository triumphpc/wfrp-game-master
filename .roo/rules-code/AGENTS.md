# Project Coding Rules (Non-Obvious Only)

## Python Scripts
- **Character card generation**: `generate_character_cards.py` uses reportlab with custom Russian fonts; must handle UTF-8 encoding
- **PDF parsing**: `pdf_to_md.py` uses PyMuPDF (fitz) with specific extraction patterns for WFRP rulebooks
- **Rule splitting**: `split_rules.py` expects specific chapter markers in `rules/WFRPG4E.ru.FULL.md`

## File Organization
- **Character sheets**: Must follow exact markdown template with specific headers (## Характеристики, ## Навыки, ## Таланты)
- **Game history**: Session files use naming pattern `YYYY-MM-DD_HH-MM_session_name.md` in `history/%game_name%/`
- **Party summaries**: `party_summary.md` files contain aggregated character stats and campaign notes

## Data Validation
- **Career advancement**: Always validate against `КАРЬЕРЫ_ПРАВИЛЬНЫЕ_СХЕМЫ.md`, not text rules
- **Character creation**: Implement checks from `ЧАСТЫЕ_ОШИБКИ.md` (common mistakes document)
- **Rule references**: Use semantic search via RAG-MCP-SERVER, not direct file searches

## Code Conventions
- **Russian text**: All user-facing content must be in Russian
- **Markdown tables**: Use pipe tables with Russian headers
- **Character stats**: Format as "ББ (Ближний бой) = 48" not just "ББ = 48"
- **File encoding**: Always UTF-8 with proper Cyrillic support