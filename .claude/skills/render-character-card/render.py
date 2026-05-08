#!/usr/bin/env python3
"""
Рендер карточки персонажа WFRP из Markdown в PNG.

Использование:
    python3 render.py <path_to_md> [--theme parchment|sylvan|dwarven-rune] [--out <png>]

Зависимости:
    pip install markdown jinja2 playwright
    playwright install chromium
"""

import argparse
import sys
import re
from pathlib import Path

try:
    import markdown
    from jinja2 import Template
except ImportError as e:
    print(f"ERROR: missing dependency: {e}", file=sys.stderr)
    print("Install: pip install markdown jinja2 playwright && playwright install chromium", file=sys.stderr)
    sys.exit(2)


SCRIPT_DIR = Path(__file__).resolve().parent
TEMPLATE_PATH = SCRIPT_DIR / "template.html"
THEMES_DIR = SCRIPT_DIR / "themes"
DEFAULT_THEME = "parchment"


def list_themes() -> list[str]:
    return sorted(p.stem for p in THEMES_DIR.glob("*.css"))


def parse_card(md_text: str) -> dict:
    """Извлекает имя/подзаголовок из карточки. Остальное — как HTML markdown."""
    name = "Безымянный"
    subtitle = ""

    m = re.search(r"^#\s+(.+?)$", md_text, re.MULTILINE)
    if m:
        title = m.group(1).strip().strip('"').strip("«»")
        parts = re.split(r"\s+[—–-]\s+", title, maxsplit=1)
        name = parts[0].strip().strip('"').strip("«»")
        if len(parts) > 1:
            subtitle = parts[1].strip().strip('"').strip("«»")

    body_md = re.sub(r"^#\s+.+?$", "", md_text, count=1, flags=re.MULTILINE).strip()
    html_body = markdown.markdown(
        body_md,
        extensions=["tables", "fenced_code", "nl2br", "sane_lists"],
    )
    return {"name": name, "subtitle": subtitle, "body_html": html_body}


def build_html(md_path: Path, theme: str) -> str:
    md_text = md_path.read_text(encoding="utf-8")
    data = parse_card(md_text)

    css_path = THEMES_DIR / f"{theme}.css"
    if not css_path.exists():
        available = ", ".join(list_themes())
        raise FileNotFoundError(f"Theme '{theme}' not found. Available: {available}")

    template = Template(TEMPLATE_PATH.read_text(encoding="utf-8"))
    css = css_path.read_text(encoding="utf-8")

    return template.render(
        name=data["name"],
        subtitle=data["subtitle"],
        body=data["body_html"],
        css=css,
        theme=theme,
    )


def render_png(html: str, output: Path):
    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        print("ERROR: playwright not installed. Run: pip install playwright && playwright install chromium", file=sys.stderr)
        sys.exit(2)
    output.parent.mkdir(parents=True, exist_ok=True)
    with sync_playwright() as p:
        browser = p.chromium.launch()
        context = browser.new_context(viewport={"width": 900, "height": 1200}, device_scale_factor=2)
        page = context.new_page()
        page.set_content(html, wait_until="networkidle")
        page.evaluate("document.fonts.ready")
        page.screenshot(path=str(output), full_page=True, omit_background=False)
        browser.close()


def main():
    ap = argparse.ArgumentParser(description="Render WFRP character card from Markdown to PNG")
    ap.add_argument("md_path", help="Path to character markdown file")
    ap.add_argument("--theme", default=DEFAULT_THEME, choices=list_themes() or [DEFAULT_THEME],
                    help=f"Visual theme (default: {DEFAULT_THEME})")
    ap.add_argument("--out", help="Output PNG path (default: same as md, .png suffix)")
    args = ap.parse_args()

    md_path = Path(args.md_path).resolve()
    if not md_path.exists():
        print(f"ERROR: file not found: {md_path}", file=sys.stderr)
        sys.exit(1)

    if args.out:
        out_path = Path(args.out).resolve()
    else:
        # Если тема не дефолтная — добавим суффикс к имени
        suffix = "" if args.theme == DEFAULT_THEME else f"_{args.theme}"
        out_path = md_path.with_name(md_path.stem + suffix + ".png")

    html = build_html(md_path, args.theme)
    render_png(html, out_path)
    print(str(out_path))


if __name__ == "__main__":
    main()
