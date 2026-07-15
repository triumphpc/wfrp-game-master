## ADDED Requirements

### Requirement: d100 roll with CSS 3D animation
The system SHALL provide a clickable d100 control that, when activated, generates a random integer from 1 to 100 (displayed as 01–100) and animates two CSS 3D cubes (a "tens" cube showing 00/10/20/30/40/50/60/70/80/90 and a "units" cube showing 0–9) settling on the rolled value.

#### Scenario: User clicks the d100 control
- **WHEN** the user clicks or taps the d100 control
- **THEN** the system generates a random value from 1 to 100
- **AND** both the tens and units CSS 3D cubes animate (rotate) for approximately 600–1000ms
- **AND** after animation completes, the tens cube face shows the tens digit and the units cube face shows the units digit of the rolled value
- **AND** the result is displayed numerically below the cubes
- **AND** the roll is appended to the history

#### Scenario: d100 roll of exactly 100
- **WHEN** the generated value is 100
- **THEN** the tens cube shows "00" (or "100" representation) and the units cube shows "0"
- **AND** the numeric result displays as "100" (not "00")

#### Scenario: Reduced-motion preference
- **WHEN** the user's system has `prefers-reduced-motion: reduce` set
- **THEN** the cubes skip the rotation animation and display the result immediately

### Requirement: d10 roll with CSS 3D animation
The system SHALL provide a clickable d10 control that, when activated, generates a random integer from 1 to 10 (10 displayed as "10", not "0") and animates a single CSS 3D cube settling on the rolled value.

#### Scenario: User clicks the d10 control
- **WHEN** the user clicks or taps the d10 control
- **THEN** the system generates a random value from 1 to 10
- **AND** the CSS 3D cube animates (rotates) for approximately 600–1000ms
- **AND** after animation completes, the cube face shows the rolled value
- **AND** the result is displayed numerically
- **AND** the roll is appended to the history

### Requirement: Skill check resolution
The system SHALL provide a skill-check mode where the user enters a target number (the skill or characteristic value) and an optional modifier (clamped to ±30), then rolls d100. The system SHALL compute the outcome, Success Level (SL), critical/fumble status, and hit location per WFRP 4e rules.

#### Scenario: Successful skill check
- **WHEN** the user enters target 50 and modifier 0, then rolls
- **AND** the d100 result is 47 (which is ≤ 50)
- **THEN** the outcome is "Success"
- **AND** the SL is `+0` (computed as `(50 − 47) / 10 = 0.3`, floored to 0)
- **AND** the hit location is derived from the reversed roll (74 → right arm)
- **AND** the result, SL, outcome, and hit location are displayed and appended to history

#### Scenario: Failed skill check
- **WHEN** the user enters target 50 and modifier 0, then rolls
- **AND** the d100 result is 72 (which is > 50)
- **THEN** the outcome is "Fail"
- **AND** the SL is `+2` against the attacker (computed as `(72 − 50) / 10 = 2.2`, floored to 2)
- **AND** the result is displayed and appended to history

#### Scenario: Guaranteed success (01–05)
- **WHEN** the d100 result is between 01 and 05 inclusive
- **THEN** the outcome is "Success" regardless of the target number
- **AND** the outcome is labeled as a guaranteed success

#### Scenario: Guaranteed failure (96–100)
- **WHEN** the d100 result is between 96 and 100 inclusive
- **THEN** the outcome is "Fail" regardless of the target number
- **AND** the outcome is labeled as a guaranteed failure (fumble)

#### Scenario: Double roll on success (critical)
- **WHEN** the d100 result is a double (11, 22, 33, 44, 55, 66, 77, 88) and the roll succeeds (result ≤ target)
- **THEN** the outcome is "Critical Success"
- **AND** the SL is calculated normally

#### Scenario: Double roll on failure (fumble)
- **WHEN** the d100 result is a double (11, 22, 33, 44, 55, 66, 77, 88, 00) and the roll fails (result > target)
- **THEN** the outcome is "Fumble"
- **AND** the SL is calculated normally

#### Scenario: Modifier clamped to ±30
- **WHEN** the user enters a modifier greater than +30 or less than −30
- **THEN** the system clamps the effective modifier to +30 or −30 respectively before resolving the roll

#### Scenario: Hit location derived from reversed roll
- **WHEN** a d100 skill check is rolled
- **THEN** the hit location is computed by reversing the two digits of the roll (e.g., 47 → 74) and mapping to the WFRP 4e hit-location table:
  - 01–10: Head
  - 11–20: Right Arm
  - 21–30: Left Arm
  - 31–55: Body
  - 56–80: Body
  - 81–90: Right Leg
  - 91–00: Left Leg
- **AND** the hit location is displayed alongside the roll result

### Requirement: Damage roll (Nd10 + X with weapon quality)
The system SHALL provide a damage mode where the user specifies a number of d10 dice (N), a flat modifier (X), and an optional weapon-quality setting (`none`, `-1s`, `-2s`). The system SHALL roll N d10 dice, apply weapon-quality cancellation (removing the lowest 1 or 2 dice), apply the flat modifier, and support exploding 10s.

#### Scenario: Simple damage roll
- **WHEN** the user enters 2 dice, modifier +4, weapon quality "none"
- **AND** clicks roll
- **THEN** the system rolls 2d10 (e.g., 8 and 3), adds the modifier (+4), and displays the total (15)
- **AND** each individual die result is shown alongside the total
- **AND** the roll is appended to history

#### Scenario: Weapon quality cancels lowest die
- **WHEN** the user enters 3 dice, modifier +2, weapon quality "-1s"
- **AND** the dice roll 8, 3, 5
- **THEN** the lowest die (3) is cancelled (shown struck-through)
- **AND** the total is computed as 8 + 5 + 2 = 15
- **AND** the cancelled die is visible in the result display and history

#### Scenario: Exploding 10s
- **WHEN** a d10 in a damage roll shows 10
- **THEN** that die is rolled again and the new value is added
- **AND** if the new value is also 10, it explodes again (repeat until non-10)
- **AND** the total accumulates all explosion values
- **AND** the history entry notes the explosion chain

### Requirement: Scatter helper
The system SHALL provide a scatter helper that, when activated, rolls 1d10 for direction and 2d10 for distance, per WFRP 4e missile-fire scatter rules.

#### Scenario: Scatter direction and distance
- **WHEN** the user clicks the scatter helper
- **THEN** the system rolls 1d10 for direction:
  - 1–8: directional (mapped to a compass rose: 1=N, 2=NE, 3=E, 4=SE, 5=S, 6=SW, 7=W, 8=NW)
  - 9: scatter toward the attacker
  - 10: scatter toward the target (no deviation)
- **AND** rolls 2d10 for distance in yards
- **AND** displays the direction (with a visual arrow/indicator) and distance
- **AND** appends the result to history

### Requirement: Quick-skills reference
The system SHALL display a hardcoded reference list of basic WFRP 4e skills, each labeled with its governing characteristic. Clicking a skill SHALL pre-fill the skill-check target input (empty, for the user to enter their value) and display the characteristic as a hint.

#### Scenario: User clicks a skill in the reference list
- **WHEN** the user clicks "Athletics (S)" in the quick-skills list
- **THEN** the skill-check mode is focused
- **AND** the target input is cleared and focused for entry
- **AND** a hint shows "Rolls against Strength (С)" near the target input

#### Scenario: Reference list content
- **WHEN** the page loads
- **THEN** the quick-skills list includes at minimum: Athletics (S), Charm (WP), Cool (I), Endurance (T), Entertain (Fel), Gossip (Fel), Intimidate (S), Leadership (Fel), Navigation (I), Perception (I), Ride (D), Stealth (D)
- **AND** each skill shows its characteristic abbreviation

### Requirement: Roll history with localStorage persistence
The system SHALL maintain a persistent roll history in `localStorage` under the key `wfrp-dice-history`. Each entry SHALL record the timestamp, roll type, dice values, target (if applicable), modifier, outcome, SL, hit location, and any notes. The history SHALL be displayed in reverse-chronological order with color-coding for critical and fumble outcomes.

#### Scenario: Roll appended to history
- **WHEN** any roll is completed (d100, d10, skill check, damage, scatter)
- **THEN** a new history entry is created with timestamp, type, dice, target, modifier, result, outcome, SL, and hit location
- **AND** the entry appears at the top of the history list
- **AND** the entry is persisted to `localStorage`

#### Scenario: History survives page reload
- **WHEN** the page is reloaded
- **THEN** all previously stored history entries are loaded from `localStorage` and displayed in reverse-chronological order

#### Scenario: History cap with FIFO eviction
- **WHEN** the number of stored history entries exceeds 500
- **THEN** the oldest entry is removed (FIFO) before the new entry is added
- **AND** no error is shown to the user

#### Scenario: Clear history
- **WHEN** the user clicks the "Clear" button in the history section
- **THEN** a confirmation dialog is shown
- **AND** if confirmed, all history entries are removed from `localStorage` and the display

#### Scenario: Critical and fumble color-coding
- **WHEN** a history entry has outcome "Critical Success" or "Fumble"
- **THEN** the entry is visually highlighted (e.g., gold border for critical, red border for fumble) to distinguish it from normal rolls

### Requirement: Markdown export of history
The system SHALL provide a "Copy as Markdown" button that copies the current roll history to the clipboard as a formatted Markdown table.

#### Scenario: User copies history as Markdown
- **WHEN** the user clicks "Copy as Markdown"
- **THEN** the history is serialized as a Markdown table with columns: Time, Type, Roll, Target, Modifier, Outcome, SL, Hit Location
- **AND** the table is copied to the clipboard
- **AND** a non-blocking confirmation toast is shown ("Copied N rolls to clipboard")

#### Scenario: Empty history export
- **WHEN** the user clicks "Copy as Markdown" with no history entries
- **THEN** a toast is shown ("No rolls to copy") and the clipboard is not modified

### Requirement: Dark Empire grunge visual theme
The system SHALL apply a "Dark Empire grunge" visual theme using a dark background (`#0d0d0d`), gold accent color (`#c9a227`), blood-red highlight (`#8b0000`), Cinzel accent font (with serif fallback), and a layout that evokes the Old World aesthetic of WFRP.

#### Scenario: Page load visual theme
- **WHEN** the page loads
- **THEN** the background is dark (`#0d0d0d` or near-black)
- **AND** headings and accents use gold (`#c9a227`)
- **AND** critical/fumble highlights use blood-red (`#8b0000`)
- **AND** the Cinzel font is applied to headings and die faces (with `Trajan Pro`, `Times New Roman`, `serif` fallback)
- **AND** the layout is responsive (usable on mobile and desktop)

#### Scenario: Cinzel font unavailable
- **WHEN** the Google Fonts CDN is unreachable or blocked
- **THEN** the page falls back to `Times New Roman` or system serif
- **AND** all functionality remains intact

### Requirement: Single-file deployment to GitHub Pages
The system SHALL be deployable as a single `docs/index.html` file (with all CSS and JS inline) served from the `/docs` folder via GitHub Pages, with no build step and no server-side component.

#### Scenario: Deployment via git push
- **WHEN** `docs/index.html` is committed and pushed to the `main` branch
- **AND** GitHub Pages is configured to serve from `main` / `/docs`
- **THEN** the page is accessible at `https://<username>.github.io/wfrp-game-master/` within approximately one minute

#### Scenario: No external runtime dependencies
- **WHEN** the page is loaded
- **THEN** the only external resource requested is the Cinzel font from Google Fonts CDN (optional, with graceful fallback)
- **AND** no JavaScript libraries, CSS frameworks, or other assets are fetched
