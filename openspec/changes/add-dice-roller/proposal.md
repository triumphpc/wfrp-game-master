## Why

The repository has no web-facing tooling for the WFRP 4e game master workflow. GM and players currently roll physical dice (or ad-hoc web rollers) without the Old World atmosphere or WFRP-specific rules baked in. A dedicated, self-contained dice roller page — deployable to GitHub Pages from this repo — gives a fast, themeable tool that speaks WFRP 4e mechanics natively (d100 roll-under with SL, hit location, weapon-quality damage, scatter) and fits the grimdark aesthetic of the setting.

## What Changes

- **NEW**: Single-file static HTML page (`docs/index.html`) serving as a WFRP 4e dice roller, deployable to GitHub Pages via the `/docs` folder (no build step, no server, no dependencies).
- **NEW**: Clickable CSS 3D d100 (rendered as two d10 cubes: tens + units) and d10 cube that animate on roll.
- **NEW**: Skill check mode — target number + modifier (±30 cap) → d100 roll-under resolution with automatic Success Level (SL) calculation, critical/fumble detection (doubles, 01–05, 96–00), and hit location derived from the reversed roll.
- **NEW**: Damage mode — `Nd10 + X` with optional weapon quality modifier (`-1s` / `-2s` cancels lowest die) and exploding-10 support.
- **NEW**: Scatter helper — d10 direction (1–8) + 2d10 distance in yards, per WFRP 4e missile-failure rules.
- **NEW**: Built-in quick-skills reference (hardcoded list of basic WFRP 4e skills keyed to their governing characteristic) that pre-fills the skill-check target and reminds the user which characteristic applies.
- **NEW**: Persistent roll history in `localStorage` (reverse-chronological, color-coded for crit/fumble), with one-click export as a Markdown table to clipboard.
- **NEW**: "Dark Empire grunge" visual theme — `#0d0d0d` / `#c9a227` / `#8b0000` palette, Cinzel accent font (Google Fonts CDN), responsive layout for desktop and mobile.
- **NEW**: GitHub Pages deployment configuration (repo-level setting pointing at `main` / `/docs`; no workflow file required for the no-build approach).

## Capabilities

### New Capabilities
- `dice-roller`: Interactive WFRP 4e dice rolling — d100 and d10 with CSS 3D animation, skill-check resolution (SL, crit/fumble, hit location), damage calculation (Nd10+X with weapon quality), scatter helper, quick-skills reference, and persistent history with Markdown export.

### Modified Capabilities
<!-- None. The dice roller is a standalone new capability; it does not change the existing wfrp-rag-plugin spec. -->

## Impact

- **New files**: `docs/index.html` (single self-contained file, ~40KB, all HTML/CSS/JS inline). Optionally `docs/favicon.svg`.
- **Repo configuration**: GitHub repository settings → Pages → Source set to `main` branch / `/docs` folder (one-time UI configuration, not a committed file).
- **Dependencies**: None at build time. Runtime fetches Cinzel font from Google Fonts CDN (graceful fallback to system serif if offline).
- **No existing code affected**: The change is additive and isolated under `docs/`. It does not touch `.claude/`, `rules/`, `characters/`, `history/`, or the `wfrp-rag-plugin` spec.
- **No integrations**: By explicit decision, the roller does NOT read character sheets, rules dictionaries, or crit tables from the repo — it is a standalone tool with a hardcoded quick-skills list.
