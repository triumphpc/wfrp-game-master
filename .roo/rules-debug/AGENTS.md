# Project Debug Rules (Non-Obvious Only)

## Character Sheet Debugging
- **HP calculation errors**: Check `ЧАСТЫЕ_ОШИБКИ.md` - "Закалка" talent increases HP, not Endurance characteristic
- **Career advancement validation**: Use `КАРЬЕРЫ_ПРАВИЛЬНЫЕ_СХЕМЫ.md` to verify schemes, not text rules
- **Duplicate skills**: They should stack (race + career), not be treated as errors

## Python Script Debugging
- **PDF generation**: `generate_character_cards.py` requires Russian fonts; check font paths if PDFs show missing characters
- **PDF parsing**: `pdf_to_md.py` may fail on malformed PDFs; use PyMuPDF's text extraction with specific patterns
- **Rule splitting**: `split_rules.py` depends on chapter markers in `rules/WFRPG4E.ru.FULL.md`; verify markers exist

## Game Session Debugging
- **Session file naming**: Must follow `YYYY-MM-DD_HH-MM_session_name.md` pattern for proper chronological sorting
- **Character references**: Historical characters are in `history/%game_name%/characters/`, not root `characters/`
- **Party summaries**: `party_summary.md` files should aggregate stats from all characters in campaign

## Rule Lookup Debugging
- **RAG-MCP-SERVER**: Must be running for semantic rule searches; direct file searches are insufficient
- **Rule conflicts**: Text rules contain errors; always cross-reference with corrected schemes
- **Quick reference**: Use `ТАБЛИЦЫ_БЫСТРЫЙ_ДОСТУП.md` for common tables, not full rule search