#!/usr/bin/env python3
"""hostlab — terminal toolkit inspired by HostLab"""
from __future__ import annotations

import json
import ipaddress
import base64
import zlib
import random
from datetime import date, timedelta
from pathlib import Path
from typing import Optional

import typer
from rich.console import Console
from rich.panel import Panel
from rich.table import Table
from rich.text import Text
from rich.align import Align
from rich.rule import Rule
from rich.prompt import Confirm
from rich import box

console = Console()

app = typer.Typer(
    name="hostlab",
    help="HostLab terminal toolkit",
    no_args_is_help=True,
    add_completion=False,
    rich_markup_mode="rich",
)
check_app = typer.Typer(no_args_is_help=False)
burn_app  = typer.Typer(no_args_is_help=True)
app.add_typer(check_app, name="check", help="[green]●[/green] Persistent checklist manager")
app.add_typer(burn_app,  name="burn",  help="[green]●[/green] Encode / decode secret notes")

# ── paths ─────────────────────────────────────────────────────────────────
DATA_DIR       = Path.home() / ".hostlab"
CHECKLIST_FILE = DATA_DIR / "checklist.json"

def _ensure_dir():
    DATA_DIR.mkdir(parents=True, exist_ok=True)

# ── layout helpers ────────────────────────────────────────────────────────
BANNER = """\
[bold bright_green] H O S T L A B [/bold bright_green][dim]terminal[/dim]"""

def _header(title: str):
    console.print()
    console.rule(f"[bold bright_green]{title}[/bold bright_green]", style="dim green")
    console.print()


# ═══════════════════════════════════════════════════════════════════════════
# ABOUT
# ═══════════════════════════════════════════════════════════════════════════

@app.command("about")
def about():
    """Show the tool list and credits"""
    console.print()
    console.print(Panel(
        Align.center(
            "\n"
            "[bold bright_green] H O S T L A B [/bold bright_green][dim]terminal[/dim]\n\n"
            "[dim]a CLI port of the personal experiment hub[/dim]\n"
        ),
        border_style="green",
        padding=(0, 4),
    ))
    console.print()

    t = Table(box=box.SIMPLE, show_header=False, padding=(0, 2))
    t.add_column("cmd",  style="bright_green", width=16)
    t.add_column("desc", style="dim")

    t.add_row("ip <CIDR>",  "Subnet visualizer — decompose any IP/CIDR")
    t.add_row("check",      "Persistent checklist (add / done / rm / clear)")
    t.add_row("spin",       "App idea spinner — get unstuck, build something")
    t.add_row("gap",        "How many days did two lives share?")
    t.add_row("burn enc",   "Compress a message into a shareable token")
    t.add_row("burn dec",   "Recover a message from a token")

    console.print(t)
    console.print(f"\n  [dim]github.com/iwohost/HostLab[/dim]\n")


# ═══════════════════════════════════════════════════════════════════════════
# IP VISUALIZER
# ═══════════════════════════════════════════════════════════════════════════

def _addr_kind(ip: ipaddress.IPv4Address | ipaddress.IPv6Address) -> str:
    if ip.is_loopback:     return "loopback"
    if ip.is_link_local:   return "link-local"
    if ip.is_private:      return "private"
    if ip.is_multicast:    return "multicast"
    return "public"

@app.command("ip")
def ip_viz(
    address: str = typer.Argument(..., help="IP/CIDR — e.g. 192.168.1.50/24 or 10.0.0.1/8"),
):
    """[green]●[/green] Subnet visualizer — decompose an IP/CIDR into network details"""
    try:
        iface = ipaddress.ip_interface(address)
    except ValueError as e:
        console.print(f"\n  [red]✗[/red] {e}\n")
        raise typer.Exit(1)

    net    = iface.network
    prefix = net.prefixlen
    total  = net.num_addresses
    usable = max(0, total - 2)

    _header("IP VISUALIZER")

    # ── info table ────────────────────────────────────────────────────────
    t = Table(box=box.SIMPLE, show_header=False, padding=(0, 2))
    t.add_column("k", style="dim",          width=20)
    t.add_column("v", style="bright_white")

    t.add_row("Address",       f"{iface.ip}  [dim]({_addr_kind(iface.ip)})[/dim]")
    t.add_row("Network",       str(net.network_address))
    t.add_row("Broadcast",     str(net.broadcast_address))
    t.add_row("Subnet mask",   str(net.netmask))
    t.add_row("Wildcard mask", str(net.hostmask))
    t.add_row("Prefix",        f"/{prefix}")
    t.add_row("Total IPs",     f"{total:,}")
    t.add_row("Usable hosts",  f"{usable:,}")

    if usable >= 1:
        hosts = list(net.hosts())
        t.add_row("First host",  str(hosts[0]))
        if usable >= 2:
            t.add_row("Last host", str(hosts[-1]))

    console.print(t)
    console.print()

    # ── binary breakdown ──────────────────────────────────────────────────
    console.print("  [dim]binary — [bright_green]network bits[/bright_green]  host bits[/dim]\n")

    ip_int   = int(iface.ip)
    mask_int = int(net.netmask)

    labels = ["·A·", "·B·", "·C·", "·D·"]
    for octet_idx in range(4):
        shift     = (3 - octet_idx) * 8
        ip_byte   = (ip_int   >> shift) & 0xFF
        mask_byte = (mask_int >> shift) & 0xFF

        row = Text(f"  {labels[octet_idx]}  ", style="dim")
        for bit_pos in range(7, -1, -1):
            bit    = (ip_byte   >> bit_pos) & 1
            is_net = bool((mask_byte >> bit_pos) & 1)
            row.append(str(bit), style="bold bright_green" if is_net else "white")
            if bit_pos == 4:
                row.append(" ")
        row.append(f"  {ip_byte:>3}", style="dim")
        console.print(row)

    console.print()

    # ── capacity bar ──────────────────────────────────────────────────────
    if prefix <= 30:
        bar_w = 46
        filled = min(bar_w, bar_w)  # always show full bar (capacity, not usage)
        console.print(
            f"  [dim]capacity[/dim]  "
            f"[bright_green]{'█' * bar_w}[/bright_green]  "
            f"[dim]{usable:,} usable hosts[/dim]"
        )
    console.print()


# ═══════════════════════════════════════════════════════════════════════════
# CHECKLIST
# ═══════════════════════════════════════════════════════════════════════════

def _load() -> list[dict]:
    _ensure_dir()
    if not CHECKLIST_FILE.exists():
        return []
    try:
        return json.loads(CHECKLIST_FILE.read_text())
    except Exception:
        return []

def _save(items: list[dict]):
    _ensure_dir()
    CHECKLIST_FILE.write_text(json.dumps(items, indent=2))

def _print_list(items: list[dict]):
    if not items:
        console.print('  [dim]Empty. Add a task with:  hostlab check add "do the thing"[/dim]')
        return
    for i, item in enumerate(items, 1):
        done  = item.get("done", False)
        icon  = "[bright_green]✓[/bright_green]" if done else "[dim]○[/dim]"
        label = f"[dim strike]{item['text']}[/dim strike]" if done else item["text"]
        console.print(f"  [dim]{i:>2}.[/dim]  {icon}  {label}")

@check_app.callback(invoke_without_command=True)
def check_default(ctx: typer.Context):
    """[green]●[/green] Persistent checklist — your tasks, in your terminal"""
    if ctx.invoked_subcommand is not None:
        return
    items = _load()
    _header("CHECKLIST")
    _print_list(items)
    if items:
        done  = sum(1 for i in items if i.get("done"))
        total = len(items)
        pct   = int(done / total * 100)
        bar_w = 30
        filled = int(done / total * bar_w)
        bar = f"[bright_green]{'█' * filled}[/bright_green][dim]{'░' * (bar_w - filled)}[/dim]"
        console.print(f"\n  {bar}  [dim]{done}/{total}  ({pct}%)[/dim]")
    console.print()

@check_app.command("add")
def check_add(text: str = typer.Argument(..., help="Task description")):
    """Add a task"""
    items = _load()
    items.append({"text": text, "done": False, "created": date.today().isoformat()})
    _save(items)
    console.print(f"\n  [bright_green]+[/bright_green]  {text}\n")

@check_app.command("done")
def check_done(n: int = typer.Argument(..., help="Task number to toggle")):
    """Toggle a task done/undone"""
    items = _load()
    if not 1 <= n <= len(items):
        console.print(f"\n  [red]✗[/red]  No task #{n}\n"); raise typer.Exit(1)
    item = items[n - 1]
    item["done"] = not item["done"]
    _save(items)
    state = "[bright_green]done ✓[/bright_green]" if item["done"] else "[dim]undone ○[/dim]"
    console.print(f"\n  {item['text']}  →  {state}\n")

@check_app.command("rm")
def check_rm(n: int = typer.Argument(..., help="Task number to remove")):
    """Remove a task"""
    items = _load()
    if not 1 <= n <= len(items):
        console.print(f"\n  [red]✗[/red]  No task #{n}\n"); raise typer.Exit(1)
    removed = items.pop(n - 1)
    _save(items)
    console.print(f"\n  [dim]removed:[/dim]  {removed['text']}\n")

@check_app.command("clear")
def check_clear():
    """Remove all completed tasks"""
    items  = _load()
    before = len(items)
    items  = [i for i in items if not i.get("done")]
    _save(items)
    n = before - len(items)
    console.print(f"\n  [dim]cleared {n} completed task{'s' if n != 1 else ''}[/dim]\n")


# ═══════════════════════════════════════════════════════════════════════════
# APP IDEA SPINNER
# ═══════════════════════════════════════════════════════════════════════════

_IDEAS: dict[str, list[tuple[str, str]]] = {
    "visual": [
        ("Pixel Sorter",            "Sort pixels by hue or brightness in real time"),
        ("ASCII Cam",               "Turn a webcam feed into live ASCII art"),
        ("Gradient Mesh Editor",    "Drag control points to sculpt smooth color gradients"),
        ("Reaction Diffusion",      "Interactive Turing-pattern simulator on a canvas"),
        ("Chromatic Aberration Lab","Apply lens-split RGB fringing to any uploaded image"),
        ("Halftone Engine",         "Simulate newspaper halftone at variable screen angles"),
        ("Voronoi Painter",         "Click to drop seeds and watch a Voronoi diagram grow"),
        ("Perlin Terrain",          "Real-time 2D terrain generator with color biomes"),
        ("SVG Morpher",             "Draw a path and morph it into another with easing"),
        ("Glitch Generator",        "Apply datamoshing and compression artifacts to images"),
    ],
    "audio": [
        ("Chord Namer",             "Click piano keys and instantly see the chord name"),
        ("Binaural Beat Gen",       "Set two slightly-offset tones for focus or sleep"),
        ("Lo-fi Degrader",          "Apply vinyl crackle and tape hiss to any audio"),
        ("Arpeggiator",             "Play a chord and loop through its notes in patterns"),
        ("Spectral Freeze",         "Freeze a moment of sound and let it drone forever"),
        ("Euclidean Drummer",       "Generative drum machine built on Euclidean rhythms"),
        ("Pitch Visualizer",        "Real-time pitch detection displayed on a musical staff"),
    ],
    "productivity": [
        ("Habit Punch Card",        "Heatmap-style tracker for daily habit streaks"),
        ("Timezone Overlap",        "Find meeting windows across multiple time zones"),
        ("Meeting Cost Clock",      "Watch money drain as your meeting runs over"),
        ("One-liner Journal",       "Date-stamped single lines — simple, searchable log"),
        ("Markdown Flashcards",     "Write Q/A pairs in Markdown, quiz yourself later"),
        ("Decision Matrix",         "Weighted criteria table that scores options fairly"),
        ("Focus Sprint Timer",      "Pomodoro with adaptive break lengths and history"),
    ],
    "utility": [
        ("Regex Explainer",         "Match groups with human-readable step-by-step output"),
        ("Color Blind Sim",         "Preview any image through 8 types of color vision"),
        ("CSS Easing Playground",   "Drag bezier handles, copy the CSS value"),
        ("Password Entropy Meter",  "Crack-time estimate and actionable suggestions"),
        ("Font Pairing Lab",        "Drag-and-drop font combos with live preview"),
        ("Contrast Ratio Checker",  "Pick two colors, get WCAG pass/fail instantly"),
        ("Base Converter",          "Type in any numeric base, all others update live"),
    ],
    "data": [
        ("Spending Heatmap",        "Paste a CSV bank export → calendar heatmap"),
        ("Word Frequency Map",      "Paste text → word cloud sorted by count"),
        ("Sort Algorithm Race",     "Watch bubble, merge, and quicksort compete live"),
        ("Network Graph Builder",   "Type node-edge pairs, see a force-directed graph"),
        ("Correlation Explorer",    "Upload two CSVs, find statistical correlations"),
        ("Log Timeline",            "Paste any log file → visual event timeline"),
    ],
    "fun": [
        ("Vibe Checker",            "5 sliders → your totally unscientific vibe score"),
        ("Excuse Generator",        "500 plausible excuses by situation and severity"),
        ("Commit Message Oracle",   "Suspiciously accurate random commit messages"),
        ("Error Message Glossary",  "Human-readable dictionary of confusing system errors"),
        ("Fake Loading Screen",     "Convincing progress bar with dramatic log output"),
        ("Which Framework Are You?","5 questions → get roasted by a JS framework"),
        ("Keyboard Smash Analyzer", "Rate the quality of your asdfghjkl moments"),
    ],
}

_ALL = [(cat, name, desc) for cat, pairs in _IDEAS.items() for name, desc in pairs]

@app.command("spin")
def spin(
    category: Optional[str] = typer.Option(
        None, "-c", "--category",
        help=f"Category: {', '.join(_IDEAS)}",
    ),
    count: int = typer.Option(1, "-n", "--count", help="How many ideas to show"),
):
    """[green]●[/green] App idea spinner — get unstuck, build something"""
    pool = [(c, n, d) for c, n, d in _ALL if category is None or c == category]

    if not pool:
        opts = ", ".join(_IDEAS)
        console.print(f"\n  [red]✗[/red]  Unknown category. Options: [dim]{opts}[/dim]\n")
        raise typer.Exit(1)

    picks = random.sample(pool, min(count, len(pool)))

    _header("SPIN")

    for i, (cat, name, desc) in enumerate(picks):
        if i:
            console.print()
        cat_tag = Text(f"  [{cat}]", style="dim")
        console.print(cat_tag)
        console.print(Text(f"  {name}", style="bold bright_green"))
        console.print(Text(f"  {desc}", style="dim"))

    console.print()
    hint = " " if count == 1 else f" -n {count} "
    console.print(f"  [dim]run again for a new idea · try -n 3 or -c visual[/dim]")
    console.print()


# ═══════════════════════════════════════════════════════════════════════════
# GAP VISUALIZER
# ═══════════════════════════════════════════════════════════════════════════

@app.command("gap")
def gap(
    name1:    str = typer.Argument(..., help="First person's name"),
    dob1:     str = typer.Argument(..., help="First person's birth date (YYYY-MM-DD)"),
    name2:    str = typer.Argument(..., help="Second person's name"),
    dob2:     str = typer.Argument(..., help="Second person's birth date (YYYY-MM-DD)"),
    lifespan: int = typer.Option(90, "-l", "--lifespan", help="Assumed lifespan in years"),
):
    """[green]●[/green] Gap visualizer — how many days did two lives share?"""
    try:
        d1 = date.fromisoformat(dob1)
        d2 = date.fromisoformat(dob2)
    except ValueError:
        console.print("\n  [red]✗[/red]  Dates must be YYYY-MM-DD\n")
        raise typer.Exit(1)

    today    = date.today()
    span     = timedelta(days=int(lifespan * 365.25))
    death1   = d1 + span
    death2   = d2 + span

    overlap_start = max(d1, d2)
    overlap_end   = min(death1, death2)
    full_overlap  = max(0, (overlap_end - overlap_start).days)

    lived_end    = min(overlap_end, today)
    lived_overlap = max(0, (lived_end - overlap_start).days)

    total_days    = span.days
    pct_potential = full_overlap  / total_days * 100
    pct_lived     = lived_overlap / total_days * 100

    age_gap_days  = abs((d2 - d1).days)
    age_gap_years = age_gap_days / 365.25

    younger = name2 if d1 <= d2 else name1

    _header("GAP VISUALIZER")

    t = Table(box=box.SIMPLE, show_header=False, padding=(0, 2))
    t.add_column("k", style="dim",          width=24)
    t.add_column("v", style="bright_white")

    t.add_row(f"{name1} born",       d1.strftime("%-d %B %Y"))
    t.add_row(f"{name2} born",       d2.strftime("%-d %B %Y"))
    t.add_row("Age gap",             f"{age_gap_years:.1f} yrs  ({age_gap_days:,} days)")
    t.add_row("Overlap starts",      f"{overlap_start.strftime('%-d %B %Y')}  (when {younger} was born)")
    t.add_row("Potential overlap",   f"{full_overlap:,} days  ({pct_potential:.1f}% of a {lifespan}-yr life)")
    t.add_row("Lived so far",        f"{lived_overlap:,} days  ({pct_lived:.1f}%)")

    console.print(t)
    console.print()

    # ── timeline bars ─────────────────────────────────────────────────────
    console.print(f"  [dim]Timeline  (assumed {lifespan}-year lifespan)[/dim]\n")

    BAR = 52
    earliest = min(d1, d2)
    latest_death = max(death1, death2)
    total_span = (latest_death - earliest).days

    def _bar(birth: date, death: date, name: str, color: str):
        s = int((birth - earliest).days / total_span * BAR)
        l = max(1, int(span.days / total_span * BAR))
        l = min(l, BAR - s)
        row = Text(f"  {name:<10}  ", style="dim")
        row.append("·" * s,             style="dim")
        row.append("█" * l,             style=color)
        row.append("·" * (BAR - s - l), style="dim")
        return row

    console.print(_bar(d1, death1, name1, "bright_green"))
    console.print(_bar(d2, death2, name2, "cyan"))
    console.print()

    # overlap indicator
    ov_s = int((overlap_start - earliest).days / total_span * BAR)
    ov_l = max(1, int(full_overlap / total_span * BAR))
    row = Text(f"  {'overlap':<10}  ", style="dim")
    row.append("·" * ov_s,               style="dim")
    row.append("▓" * ov_l,               style="bright_yellow")
    row.append("·" * (BAR - ov_s - ov_l), style="dim")
    row.append(f"  {pct_potential:.0f}%", style="dim")
    console.print(row)
    console.print()


# ═══════════════════════════════════════════════════════════════════════════
# BURN NOTES
# ═══════════════════════════════════════════════════════════════════════════

def _enc(text: str) -> str:
    return base64.urlsafe_b64encode(zlib.compress(text.encode(), 9)).decode().rstrip("=")

def _dec(token: str) -> str:
    pad = token + "=" * (-len(token) % 4)
    return zlib.decompress(base64.urlsafe_b64decode(pad)).decode()

@burn_app.command("enc")
def burn_enc(message: str = typer.Argument(..., help="Message to encode")):
    """Compress a message into a shareable token"""
    token = _enc(message)
    _header("BURN — ENCODE")
    console.print(f"  [dim]token[/dim]\n")
    console.print(Panel(f"[bright_green]{token}[/bright_green]", border_style="dim green", padding=(0, 2)))
    console.print(f"\n  [dim]decode with:[/dim]  hostlab burn dec {token[:16]}…\n")

@burn_app.command("dec")
def burn_dec(token: str = typer.Argument(..., help="Token to decode")):
    """Recover a message from a token"""
    try:
        msg = _dec(token)
    except Exception:
        console.print("\n  [red]✗[/red]  Invalid or corrupted token\n")
        raise typer.Exit(1)
    _header("BURN — DECODE")
    console.print(Panel(f"[bright_white]{msg}[/bright_white]", border_style="dim green", padding=(0, 2)))
    console.print()


# ═══════════════════════════════════════════════════════════════════════════
# ENTRY
# ═══════════════════════════════════════════════════════════════════════════

def main():
    app()

if __name__ == "__main__":
    main()
