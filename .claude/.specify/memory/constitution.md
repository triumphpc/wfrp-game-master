<!--
Sync Impact Report
==================
Version Change: 2.0.0 → 3.0.0
Modified Principles: All principles tailored for WFRP RPG game system context
Added Sections: None
Removed Sections: None

Templates Requiring Updates:
- plan-template.md - Constitution Check aligned with game system context
- spec-template.md - No changes needed (flexible structure maintained)
- tasks-template.md - No changes needed (generic task structure maintained)
- CLAUDE.md - Updated operational procedures for WFRP game system

Follow-up TODOs: None - All placeholders filled with project-specific values
-->

# WFRP Game Master Constitution

> **Authority**: This constitution supersedes all other development practices. Runtime guidance in `CLAUDE.md` implements these principles operationally.

## Core Principles

### I. Context-First Development (NON-NEGOTIABLE)

Before any implementation or game session actions:
- Read existing character cards in `history/[campaign]/characters/`
- Search rules in `rules/dict/` for specific mechanics
- Review session history in `history/[campaign]/` files
- Understand game state, character conditions, and narrative continuity

**Rationale**: Prevents inconsistent rulings, ensures character state integrity, maintains narrative coherence across sessions.

### II. Single Source of Truth

All game rules, reference tables, and character data MUST be maintained in designated locations:
- **Rules**: `rules/dict/` directory (e.g., `rules/dict/КАРЬЕРЫ.md`)
- **Characters**: `history/[campaign_name]/characters/[character_name].md`
- **Session Logs**: `history/[campaign_name]/YYYY-MM-DD_HH-MM_description.md`

Duplication is forbidden — all references must point to these sources.

**Rationale**: Eliminates rule drift between sessions, ensures consistent character progression, prevents conflicting interpretations.

### III. Library-First Development

Before implementing new game mechanics or utilities:
1. Search existing codebase in `scripts/` for similar functionality
2. Evaluate existing Python implementations (e.g., `generate_character_cards.py`)
3. Prefer adapting and extending over duplicating

**Rationale**: Reduces maintenance burden, leverages existing patterns, ensures consistent behavior across utilities.

### IV. Code Reuse & DRY

Before creating new utilities, templates, or agent skills:
1. Search existing codebase for reusable implementations
2. Prefer adapting existing `.claude/skills/` patterns
3. Document why new implementation was necessary if creating new

**Rationale**: Reduces skill duplication, ensures consistent skill invocation patterns, lowers cognitive load.

### V. Russian Language Priority (NON-NEGOTIABLE)

- ALL user-facing communication: Russian language
- Game descriptions and GM prompts: Russian
- Character sheets and documentation: Russian
- Exception: Code logs and comments in English

**Rationale**: User and players communicate in Russian, ensures accessibility, maintains immersion.

### VI. Atomic Session Execution

Each game session must be independently manageable, loggable, and trackable:
- Mark session state before starting (read character cards)
- Execute game actions (GM prompts, dice rolls, rule checks)
- Log changes immediately after significant events
- Update character cards with changes

**Atomic Log Rule**: One session file per game session — never batch multiple sessions into single file.
- Session filename format: `YYYY-MM-DD_HH-MM_description.md`
- Update character cards after every session

**Rationale**: Enables session rollback, clear progress tracking, preserves narrative history, maintains character state integrity.

### VII. Quality Gates (NON-NEGOTIABLE)

Before committing session logs or character updates:
- [ ] Character state verified against rules
- [ ] No contradictory information in session log
- [ ] Dice rolls documented with modifiers
- [ ] Character changes traceable to session events

**Rationale**: Prevents broken game state in main history, maintains campaign integrity.

### VIII. Progressive Character Development

Character progression follows mandatory phases:
1. **Check** — Verify experience points and eligibility
2. **Select** — Choose characteristics/skills/talants to improve
3. **Apply** — Update character sheet with changes
4. **Record** — Document development in character history section

No phase can be skipped. Each update validated against rules in `rules/dict/РАЗВИТИЕ_ОПЫТ.md`.

**Rationale**: Reduces character errors, validates progression before irreversible changes, ensures rule compliance.

## Operational Excellence

### IX. Error Handling

- Invalid rule interpretations: Check `rules/dict/` first, flag for GM review
- Character state conflicts: Log in session, update character card accordingly
- Missing information: Mark as TODO in session, resolve before next session
- Never silently ignore rule violations

### X. Observability

- Session logs in chronological order (by filename date)
- Character history sections track all changes with dates
- Party summary in `party_summary.md` for quick reference
- Session logs include timestamp and location context

### XI. Documentation Standards

- Character cards use template from `.claude/skills/create-character/template.md`
- Rules references point to `rules/dict/` files, not main rulebook
- Session logs include: participants, location, major events, character changes
- All dates in ISO format: YYYY-MM-DD

## Game System Requirements

### Rule Adherence

- Always use `rules/dict/КАРЬЕРЫ.md` (corrected career schemes)
- Check `rules/dict/БОЙ.md` for combat mechanics
- Reference `rules/dict/ПРОВЕРКИ.md` for skill test procedures
- Consult `rules/dict/ЧАСТЫЕ_ОШИБКИ.md` for common pitfalls

### Character State Management

- HP tracking: Current/Maximum in character card
- Experience: Total received, spent, available
- Inventory: Quantities tracked (arrows, potions, etc.)
- Conditions: Active effects, injuries, status effects

### Session Workflow

1. Load campaign context (last session log + current character cards)
2. Present scene with narrative description
3. Process player actions with rule verification
4. Resolve conflicts and consequences
5. Log session events to new file
6. Update character cards with all changes
7. Save to `history/[campaign_name]/`

## Technology Standards

### Core Stack

| Layer | Technology |
|-------|------------|
| Language | Python 3.13 |
| Libraries | reportlab, PyMuPDF (fitz), pillow |
| Hosting | Local (Claude Code CLI) |
| Storage | Markdown files + Qdrant (optional) |

### File Organization

```
.claude/
├── agents/            # GM agent definitions
├── commands/          # User-facing slash commands
└── skills/            # Reusable game skills (create-character, etc.)

history/
├── characters/        # Reusable character templates
├── sessions/          # General session logs
└── [campaign_name]/    # Campaign-specific data
    ├── characters/    # Campaign character cards (current state)
    └── [dates].md    # Session logs (chronological)

rules/
├── dict/             # Quick reference tables (PREFER THIS)
├── 01-12 chapters/  # Full rulebook chapters
└── README.md          # Rulebook table of contents

scripts/               # Python utilities (PDF generation, conversion, etc.)

venv/                  # Python virtual environment (git ignored)
qdrant_storage/        # Vector database storage (git ignored)
```

## Governance

### Amendment Procedure

Constitution changes require:
1. Documented rationale
2. Impact analysis on templates/workflows
3. Version bump (MAJOR: breaking, MINOR: additive, PATCH: clarification)
4. Sync Impact Report in header
5. Update dependent documentation (CLAUDE.md, skill templates)

### Exception Process

Principle violations require justification in session log:
- Why violation is necessary
- Alternatives considered and rejected
- Mitigation strategies

### Document Hierarchy

1. **Constitution** (this file) — Principles and laws
2. **CLAUDE.md** — Operational procedures for game system
3. **WARP.md** — Alternative runtime guidance
4. **Skill templates** — Specific workflows for character creation, campaigns

---

**Version**: 3.0.0 | **Ratified**: 2025-02-07 | **Last Amended**: 2025-02-15
