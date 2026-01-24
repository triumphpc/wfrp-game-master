# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Project Overview
This is a Warhammer Fantasy Roleplay 4e (WFRP 4e) Game Master assistant project. The AI acts as both Game Master and Rules Checker for tabletop RPG sessions. Most content is in Russian.

## Non-Obvious Discoveries

### 1. Critical Rule Source Conflicts
- **Text rules in `rules/03_классы_и_карьеры.md` contain ERRORS in career advancement schemes**
- **ALWAYS use schemes from `КАРЬЕРЫ_ПРАВИЛЬНЫЕ_СХЕМЫ.md` or WARP.md** (taken from official PDF)
- Example: Hunter career scheme in text rules (lines 3233-3248) is wrong

### 2. Dual-Role AI Architecture
- AI must simultaneously act as:
  1. **Game Master** (narrating, making rulings)
  2. **Rules Checker** (verifying character sheets, calculations)
- This dual role is enforced throughout WARP.md and CLAUDE.md

### 3. RAG-MCP-SERVER Requirement
- **CRITICAL**: When searching game rules, you MUST use RAG-MCP-SERVER
- Direct file searches are insufficient due to rule complexity
- The server provides semantic search across all rule documents

### 4. Character File Organization
- **Two separate character directories exist:**
  - `characters/` - Current active character sheets
  - `history/%game_name%/characters/` - Historical/archived characters
- Character cards are generated as PDFs using `generate_character_cards.py`

### 5. Common Calculation Errors
- **Toughened talent** increases HP (Wounds), NOT Endurance characteristic
- **Bonus XP** from random generation (up to 100 XP) cannot be spent during character creation
- **Duplicate skills** are GOOD - they stack (race + career)
- **Starting wealth** formula: Copper X = 2d10 mp × X, Silver X = 1d10 ss × X, Gold X = 1 gc × X

### 6. Python Utilities
- `generate_character_cards.py` - Creates PDF character cards using reportlab
- `pdf_to_md.py` - Converts PDFs to Markdown (uses PyMuPDF)
- `split_rules.py` - Splits large rule files into chapters
- No package.json or npm scripts - pure Python environment

### 7. Game Session Structure
- History organized by campaign: `history/%game_name%/`
- Each session has timestamped markdown files
- Party summaries in `party_summary.md` files

## Commands
- Generate character cards: `python generate_character_cards.py`
- Convert PDF to Markdown: `python pdf_to_md.py input.pdf`
- Split rules: `python split_rules.py`

## Code Style
- Python scripts follow PEP 8
- Russian comments and variable names are acceptable
- Markdown files use Russian with extensive tables