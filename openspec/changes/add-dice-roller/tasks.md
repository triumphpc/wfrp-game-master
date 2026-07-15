## 1. Project scaffolding

- [x] 1.1 Create `docs/` directory at repo root
- [x] 1.2 Create `docs/index.html` with HTML5 boilerplate, `<meta viewport>` for mobile, and section comment placeholders (`<!-- ===== STYLES ===== -->`, `<!-- ===== HTML BODY ===== -->`, `<!-- ===== DICE LOGIC ===== -->`, `<!-- ===== SKILL CHECK ===== -->`, `<!-- ===== DAMAGE ===== -->`, `<!-- ===== SCATTER ===== -->`, `<!-- ===== HISTORY ===== -->`)
- [x] 1.3 Add Google Fonts `<link>` for Cinzel (weights 400, 600, 700) with `font-family` fallback chain `Cinzel, Trajan Pro, Times New Roman, serif`

## 2. Visual theme (CSS)

- [x] 2.1 Define CSS custom properties for the Dark Empire palette: `--bg: #0d0d0d`, `--gold: #c9a227`, `--blood: #8b0000`, `--parchment: #d4c4a0`, `--stone: #2a2a2a`, `--text: #e0d8c8`
- [x] 2.2 Style base layout: dark background, centered container (max-width 900px), Cinzel headings, system-serif body text, subtle vignette/grain texture via CSS gradients
- [x] 2.3 Style the dice-tray section with a stone-textured panel background and gold border
- [x] 2.4 Add `@media (prefers-reduced-motion: reduce)` rule to disable CSS 3D animations (instant result display)
- [x] 2.5 Add responsive breakpoints: single-column layout below 640px, larger cube sizes above 1024px

## 3. CSS 3D dice cubes

- [x] 3.1 Create `.dice-cube` CSS class with `transform-style: preserve-3d`, `transition: transform 0.8s cubic-bezier(...)`, and 6 `.face` child elements positioned via `rotateX/Y/Z` + `translateZ(half-size)`
- [x] 3.2 Create tens-cube variant: 6 faces showing 00, 10, 20, 30, 40, 50 (cycling for the 60–90 range via JS-driven face content swap)
- [x] 3.3 Create units-cube variant: 6 faces showing 1, 2, 3, 4, 5, 6 (cycling for 7–8–9–0 via JS)
- [x] 3.4 Create d10 cube variant: same geometry, faces labeled 1–6 (JS maps to full 1–10 range)
- [x] 3.5 Implement `rollCube(cubeEl, value)` JS function: computes a random multi-turn rotation (`rotateX(360*n + faceAngle) rotateY(...)`) that lands the correct face toward the camera, applies it with transition
- [x] 3.6 Verify cubes render correctly in Chrome, Firefox, Safari (desktop) and mobile Safari

## 4. Dice rolling logic (JS)

- [x] 4.1 Implement `rollD100()` → returns integer 1–100; split into tens (`Math.floor(v/10) * 10`) and units (`v % 10`) for cube display
- [x] 4.2 Implement `rollD10()` → returns integer 1–10 (never 0)
- [x] 4.3 Implement `rollD10s(n)` → returns array of n integers 1–10
- [x] 4.4 Wire d100 control click → `rollD100()` → animate both cubes → display result → append to history
- [x] 4.5 Wire d10 control click → `rollD10()` → animate cube → display result → append to history

## 5. Skill check resolution

- [x] 5.1 Build skill-check UI: target number input (0–100), modifier input (−30 to +30, default 0), "Roll Skill Check" button
- [x] 5.2 Implement modifier clamping: any input > +30 becomes +30, < −30 becomes −30
- [x] 5.3 Implement resolution function `resolveSkillCheck(target, modifier, roll)`:
  - Effective target = `target + modifier` (clamped)
  - Success if `roll ≤ effectiveTarget`
  - SL = `Math.floor((effectiveTarget − roll) / 10)` on success; `Math.floor((roll − effectiveTarget) / 10)` on failure
  - Guaranteed success if `roll ≤ 5`; guaranteed failure if `roll ≥ 96`
  - Critical if double digits (11, 22, …, 88) and success; fumble if double and failure (00 = 100 is always fumble)
- [x] 5.4 Implement hit-location lookup: reverse digits (`47 → 74`), map to table (01–10 Head, 11–20 R.Arm, 21–30 L.Arm, 31–55 Body, 56–80 Body, 81–90 R.Leg, 91–00 L.Leg). Add the table as a JS comment referencing `rules/dict/БОЙ.md`
- [x] 5.5 Display resolution result panel: roll value (large), outcome label (Success/Fail/Critical/Fumble with color), SL value, hit location
- [x] 5.6 Wire skill-check button → roll d100 → resolve → display → append to history with full details

## 6. Damage roll

- [x] 6.1 Build damage UI: dice-count input (1–10, default 2), modifier input (−20 to +20, default 0), weapon-quality select (`none` / `-1s` / `-2s`), "Roll Damage" button
- [x] 6.2 Implement `resolveDamage(n, mod, quality)`: roll n d10s, sort ascending, cancel lowest 1 or 2 dice if quality is `-1s`/`-2s`, sum remaining + modifier
- [x] 6.3 Implement exploding 10s: if a die rolls 10, roll again and add; repeat until non-10; track explosion chain
- [x] 6.4 Display damage result: each die (cancelled dice struck-through, exploded dice annotated), total prominently
- [x] 6.5 Wire damage button → resolve → display → append to history

## 7. Scatter helper

- [x] 7.1 Build scatter UI: "Roll Scatter" button, result display area with compass-rose visual
- [x] 7.2 Implement `resolveScatter()`: roll 1d10 for direction (1–8 = compass, 9 = toward attacker, 10 = toward target), roll 2d10 for distance in yards
- [x] 7.3 Display result: direction arrow/label, distance in yards, both dice values
- [x] 7.4 Wire scatter button → resolve → display → append to history

## 8. Quick-skills reference

- [x] 8.1 Hardcode skills object: `{Athletics: "S", Charm: "Fel", Cool: "I", Endurance: "T", Entertain: "Fel", Gossip: "Fel", Intimidate: "S", Leadership: "Fel", Navigation: "I", Perception: "I", Ride: "D", Stealth: "D"}` with Russian characteristic labels (С, Хар, И, В, Ловк)
- [x] 8.2 Render skills as clickable chips/buttons in a grid, each showing skill name + characteristic abbreviation
- [x] 8.3 Wire chip click → focus skill-check target input, clear it, show characteristic hint ("Rolls against Strength (С)")

## 9. Roll history

- [x] 9.1 Define history entry shape: `{id, timestamp, type, dice, target, modifier, result, sl, outcome, hitLocation, note}`
- [x] 9.2 Implement `loadHistory()` → read `localStorage["wfrp-dice-history"]` → parse JSON → return array (empty if missing/corrupt)
- [x] 9.3 Implement `saveHistory(entries)` → serialize to `localStorage` with try/catch; on quota error, drop oldest 50 entries and retry
- [x] 9.4 Implement `addToHistory(entry)` → prepend to array, cap at 500 (FIFO eviction), save, re-render
- [x] 9.5 Implement `renderHistory()` → reverse-chronological list; each entry shows time, type, roll, outcome (color-coded: gold border for critical, red for fumble), SL, hit location
- [x] 9.6 Add "Clear" button with confirmation dialog → `localStorage.removeItem` + re-render
- [x] 9.7 Add "Copy as Markdown" button → serialize history as Markdown table (columns: Time, Type, Roll, Target, Modifier, Outcome, SL, Hit Location) → `navigator.clipboard.writeText` → toast confirmation
- [x] 9.8 Handle empty history: "Copy as Markdown" shows "No rolls to copy" toast, does not modify clipboard

## 10. Polish and accessibility

- [x] 10.1 Add toast/notification system for clipboard copy and error messages (non-blocking, auto-dismiss after 3s)
- [x] 10.2 Add ARIA labels to all interactive controls (dice buttons, inputs, action buttons)
- [x] 10.3 Add keyboard support: Enter on focused dice/button triggers roll; Tab order is logical
- [x] 10.4 Add `favicon.svg` (a simple d10 silhouette in gold on dark) linked in `<head>`
- [x] 10.5 Test on mobile viewport (375px width): all controls tappable, cubes visible, history scrollable

## 11. Deployment

- [ ] 11.1 Commit `docs/index.html` (and `docs/favicon.svg`) to `main`
- [ ] 11.2 Push to `origin/main`
- [ ] 11.3 Configure GitHub Pages: repo Settings → Pages → Source = `main` branch / `/docs` folder
- [ ] 11.4 Verify page loads at `https://triumphpc.github.io/wfrp-game-master/` and all features work
- [ ] 11.5 Test offline behavior: disable network, reload page — confirm Cinzel falls back to serif, all rolling functionality intact
