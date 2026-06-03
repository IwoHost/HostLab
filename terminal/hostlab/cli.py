#!/usr/bin/env python3
"""hostlab — interactive terminal toolkit"""
from __future__ import annotations

import json
import ipaddress
import base64
import zlib
import random
import shlex
import os
from datetime import date, timedelta
from pathlib import Path
from typing import Optional

import typer
from rich.console import Console
from rich.panel import Panel
from rich.table import Table
from rich.text import Text
from rich.align import Align
from rich import box

# readline for arrow-key history (pre-installed on Linux, optional on Windows)
try:
    import readline
    readline.set_history_length(200)
except ImportError:
    pass

console = Console()

# ── paths ─────────────────────────────────────────────────────────────────
DATA_DIR       = Path.home() / ".hostlab"
CHECKLIST_FILE = DATA_DIR / "checklist.json"

def _ensure_dir():
    DATA_DIR.mkdir(parents=True, exist_ok=True)

# ── helpers ───────────────────────────────────────────────────────────────
def _header(title: str):
    console.print()
    console.rule(f"[bold bright_green]{title}[/bold bright_green]", style="dim green")
    console.print()

def _fmt_date(d: date) -> str:
    return f"{d.day} {d.strftime('%B %Y')}"

# ── boot screen ───────────────────────────────────────────────────────────
LOGO = [
    " ██╗  ██╗  ██████╗  ███████╗████████╗██╗      █████╗  ██████╗ ",
    " ██║  ██║ ██╔═══██╗ ██╔════╝╚══██╔══╝██║     ██╔══██╗ ██╔══██╗",
    " ███████║ ██║   ██║ ███████╗   ██║   ██║     ███████║ ██████╔╝",
    " ██╔══██║ ██║   ██║ ╚════██║   ██║   ██║     ██╔══██║ ██╔══██╗",
    " ██║  ██║ ╚██████╔╝ ███████║   ██║   ███████╗██║  ██║ ██████╔╝",
    " ╚═╝  ╚═╝  ╚═════╝  ╚══════╝   ╚═╝   ╚══════╝╚═╝  ╚═╝ ╚═════╝",
]

def _boot():
    console.print()
    for row in LOGO:
        console.print(f"[bright_green]{row}[/bright_green]")
    console.print()
    console.print("  [dim]terminal edition · v1.0 · github.com/iwohost/HostLab[/dim]")
    console.print()
    console.rule(style="dim green")
    console.print()

    t = Table(box=box.SIMPLE, show_header=False, padding=(0, 2))
    t.add_column("cmd",  style="bright_green", width=22)
    t.add_column("desc", style="dim")
    t.add_row("ip <CIDR>",        "subnet visualizer")
    t.add_row("check",            "persistent checklist")
    t.add_row("spin",             "app idea spinner  (-c visual/fun/…  -n 3)")
    t.add_row("gap <n1> <dob1> <n2> <dob2>", "lifespan overlap")
    t.add_row("burn enc <msg>",   "encode a note")
    t.add_row("burn dec <token>", "decode a note")
    t.add_row("clear · exit",     "")
    console.print(t)
    console.print()
    console.rule(style="dim green")
    console.print()


# ═══════════════════════════════════════════════════════════════════════════
# TOOL IMPLEMENTATIONS
# ═══════════════════════════════════════════════════════════════════════════

# ── IP visualizer ──────────────────────────────────────────────────────────
def _addr_kind(ip) -> str:
    if ip.is_loopback:   return "loopback"
    if ip.is_link_local: return "link-local"
    if ip.is_private:    return "private"
    if ip.is_multicast:  return "multicast"
    return "public"

def _do_ip(address: str):
    try:
        iface = ipaddress.ip_interface(address)
    except ValueError as e:
        console.print(f"\n  [red]✗[/red] {e}\n"); return

    net    = iface.network
    prefix = net.prefixlen
    total  = net.num_addresses
    usable = max(0, total - 2)

    _header("IP VISUALIZER")

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
        t.add_row("First host", str(hosts[0]))
        if usable >= 2:
            t.add_row("Last host", str(hosts[-1]))
    console.print(t)
    console.print()

    console.print("  [dim]binary — [bright_green]network bits[/bright_green]  host bits[/dim]\n")
    ip_int   = int(iface.ip)
    mask_int = int(net.netmask)
    for o in range(4):
        sh  = (3 - o) * 8
        ib  = (ip_int   >> sh) & 0xFF
        mb  = (mask_int >> sh) & 0xFF
        row = Text(f"  ·{'ABCD'[o]}·  ", style="dim")
        for b in range(7, -1, -1):
            row.append(str((ib >> b) & 1),
                       style="bold bright_green" if (mb >> b) & 1 else "white")
            if b == 4: row.append(" ")
        row.append(f"  {ib:>3}", style="dim")
        console.print(row)

    console.print()
    if prefix <= 30:
        console.print(f"  [dim]capacity[/dim]  [bright_green]{'█' * 46}[/bright_green]  [dim]{usable:,} usable[/dim]")
    console.print()

# ── checklist ──────────────────────────────────────────────────────────────
def _load() -> list[dict]:
    _ensure_dir()
    if not CHECKLIST_FILE.exists(): return []
    try: return json.loads(CHECKLIST_FILE.read_text())
    except: return []

def _save(items):
    _ensure_dir()
    CHECKLIST_FILE.write_text(json.dumps(items, indent=2))

def _do_check(args: list[str]):
    sub = args[0].lower() if args else ""

    if not sub:
        items = _load()
        _header("CHECKLIST")
        if not items:
            console.print('  [dim]Empty. Try:  check add "do the thing"[/dim]\n'); return
        for i, item in enumerate(items, 1):
            icon  = "[bright_green]✓[/bright_green]" if item.get("done") else "[dim]○[/dim]"
            label = f"[dim strike]{item['text']}[/dim strike]" if item.get("done") else item["text"]
            console.print(f"  [dim]{i:>2}.[/dim]  {icon}  {label}")
        done  = sum(1 for i in items if i.get("done"))
        total = len(items)
        bw    = 30; fill = int(done / total * bw)
        console.print(f"\n  [bright_green]{'█'*fill}[/bright_green][dim]{'░'*(bw-fill)}[/dim]  [dim]{done}/{total}[/dim]")
        console.print(); return

    if sub == "add":
        text = " ".join(args[1:])
        if not text: console.print("  usage: check add <task>"); return
        items = _load(); items.append({"text": text, "done": False, "created": date.today().isoformat()})
        _save(items); console.print(f"\n  [bright_green]+[/bright_green]  {text}\n"); return

    if sub == "done":
        try: n = int(args[1])
        except: console.print("  usage: check done <number>"); return
        items = _load()
        if not 1 <= n <= len(items): console.print(f"  [red]✗[/red]  no task #{n}"); return
        items[n-1]["done"] = not items[n-1]["done"]; _save(items)
        s = "[bright_green]done ✓[/bright_green]" if items[n-1]["done"] else "[dim]undone ○[/dim]"
        console.print(f"\n  {items[n-1]['text']}  →  {s}\n"); return

    if sub == "rm":
        try: n = int(args[1])
        except: console.print("  usage: check rm <number>"); return
        items = _load()
        if not 1 <= n <= len(items): console.print(f"  [red]✗[/red]  no task #{n}"); return
        removed = items.pop(n-1); _save(items)
        console.print(f"\n  [dim]removed:[/dim]  {removed['text']}\n"); return

    if sub == "clear":
        items = _load(); before = len(items)
        items = [i for i in items if not i.get("done")]; _save(items)
        console.print(f"\n  [dim]cleared {before - len(items)} completed task(s)[/dim]\n"); return

    console.print(f"  [red]✗[/red]  unknown subcommand: {sub}")

# ── spin ────────────────────────────────────────────────────────────────────
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
    ],
    "utility": [
        ("Regex Explainer",         "Match groups with human-readable step-by-step output"),
        ("Color Blind Sim",         "Preview any image through 8 types of color vision"),
        ("CSS Easing Playground",   "Drag bezier handles, copy the CSS value instantly"),
        ("Password Entropy Meter",  "Crack-time estimate and actionable suggestions"),
        ("Contrast Ratio Checker",  "Pick two colors, get WCAG pass/fail instantly"),
        ("Base Converter",          "Type in any numeric base, all others update live"),
    ],
    "data": [
        ("Spending Heatmap",        "Paste a CSV bank export → calendar heatmap"),
        ("Word Frequency Map",      "Paste text → word cloud sorted by count"),
        ("Sort Algorithm Race",     "Watch bubble, merge, and quicksort compete live"),
        ("Network Graph Builder",   "Type node-edge pairs, see a force-directed graph"),
        ("Correlation Explorer",    "Upload two CSVs, find statistical correlations"),
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
_ALL = [(c, n, d) for c, pairs in _IDEAS.items() for n, d in pairs]

def _do_spin(args: list[str]):
    cat   = None
    count = 1
    i = 0
    while i < len(args):
        if args[i] in ("-c", "--category") and i + 1 < len(args):
            cat = args[i + 1]; i += 2
        elif args[i] in ("-n", "--count") and i + 1 < len(args):
            try: count = int(args[i + 1])
            except: pass
            i += 2
        else:
            i += 1

    pool = [(c, n, d) for c, n, d in _ALL if cat is None or c == cat]
    if cat and not pool:
        console.print(f"\n  [red]✗[/red]  unknown category: {cat}")
        console.print(f"  [dim]options: {', '.join(_IDEAS)}[/dim]\n"); return

    picks = random.sample(pool, min(count, len(pool)))
    _header("SPIN")
    for idx, (c, name, desc) in enumerate(picks):
        if idx: console.print()
        console.print(Text(f"  [{c}]", style="dim"))
        console.print(Text(f"  {name}", style="bold bright_green"))
        console.print(Text(f"  {desc}", style="dim"))
    console.print()
    console.print("  [dim]run again · spin -n 3 · spin -c fun[/dim]")
    console.print()

# ── gap ─────────────────────────────────────────────────────────────────────
def _do_gap(name1: str, dob1: str, name2: str, dob2: str, lifespan: int = 90):
    try:
        d1 = date.fromisoformat(dob1)
        d2 = date.fromisoformat(dob2)
    except ValueError:
        console.print("\n  [red]✗[/red]  dates must be YYYY-MM-DD\n"); return

    span          = timedelta(days=int(lifespan * 365.25))
    death1, death2 = d1 + span, d2 + span
    today         = date.today()
    ov_start      = max(d1, d2)
    ov_end        = min(death1, death2)
    full_ov       = max(0, (ov_end - ov_start).days)
    lived_ov      = max(0, (min(ov_end, today) - ov_start).days)
    td            = span.days
    younger       = name2 if d1 <= d2 else name1

    _header("GAP VISUALIZER")
    t = Table(box=box.SIMPLE, show_header=False, padding=(0, 2))
    t.add_column("k", style="dim", width=24)
    t.add_column("v", style="bright_white")
    t.add_row(f"{name1} born",     _fmt_date(d1))
    t.add_row(f"{name2} born",     _fmt_date(d2))
    t.add_row("Age gap",           f"{abs((d2-d1).days)/365.25:.1f} yrs  ({abs((d2-d1).days):,} days)")
    t.add_row("Overlap starts",    f"{_fmt_date(ov_start)}  (when {younger} was born)")
    t.add_row("Potential overlap", f"{full_ov:,} days  ({full_ov/td*100:.1f}% of a {lifespan}-yr life)")
    t.add_row("Lived so far",      f"{lived_ov:,} days  ({lived_ov/td*100:.1f}%)")
    console.print(t); console.print()

    console.print(f"  [dim]Timeline  (assumed {lifespan}-year lifespan)[/dim]\n")
    BAR      = 52
    earliest = min(d1, d2)
    total_sp = (max(death1, death2) - earliest).days

    def _bar(birth, name, color):
        s = int((birth - earliest).days / total_sp * BAR)
        l = max(1, min(int(span.days / total_sp * BAR), BAR - s))
        row = Text(f"  {name:<10}  ", style="dim")
        row.append("·" * s, style="dim")
        row.append("█" * l, style=color)
        row.append("·" * (BAR - s - l), style="dim")
        return row

    console.print(_bar(d1, name1, "bright_green"))
    console.print(_bar(d2, name2, "cyan"))
    console.print()
    ov_s = int((ov_start - earliest).days / total_sp * BAR)
    ov_l = max(1, int(full_ov / total_sp * BAR))
    row  = Text(f"  {'overlap':<10}  ", style="dim")
    row.append("·" * ov_s, style="dim")
    row.append("▓" * min(ov_l, BAR - ov_s), style="bright_yellow")
    row.append("·" * max(0, BAR - ov_s - ov_l), style="dim")
    row.append(f"  {full_ov/td*100:.0f}%", style="dim")
    console.print(row); console.print()

# ── burn ─────────────────────────────────────────────────────────────────────
def _enc(text: str) -> str:
    return base64.urlsafe_b64encode(zlib.compress(text.encode(), 9)).decode().rstrip("=")

def _dec(token: str) -> str:
    pad = token + "=" * (-len(token) % 4)
    return zlib.decompress(base64.urlsafe_b64decode(pad)).decode()

def _do_burn(args: list[str]):
    sub = args[0].lower() if args else ""
    if sub == "enc":
        msg = " ".join(args[1:])
        if not msg: console.print("  usage: burn enc <message>"); return
        token = _enc(msg)
        _header("BURN — ENCODE")
        console.print(f"  [dim]token[/dim]\n")
        from rich.panel import Panel
        console.print(Panel(f"[bright_green]{token}[/bright_green]", border_style="dim green", padding=(0, 2)))
        console.print(f"\n  [dim]decode with:[/dim]  burn dec {token[:16]}…\n")
        return
    if sub == "dec":
        token = args[1] if len(args) > 1 else ""
        if not token: console.print("  usage: burn dec <token>"); return
        try:
            msg = _dec(token)
            _header("BURN — DECODE")
            from rich.panel import Panel
            console.print(Panel(f"[bright_white]{msg}[/bright_white]", border_style="dim green", padding=(0, 2)))
            console.print()
        except Exception:
            console.print("  [red]✗[/red]  invalid or corrupted token")
        return
    console.print("  usage: burn enc <message>  ·  burn dec <token>")


# ═══════════════════════════════════════════════════════════════════════════
# REPL DISPATCHER
# ═══════════════════════════════════════════════════════════════════════════

def _dispatch(raw: str):
    try:
        parts = shlex.split(raw)
    except ValueError:
        parts = raw.split()
    if not parts: return

    cmd  = parts[0].lower()
    rest = parts[1:]

    if cmd in ("exit", "quit", "q", "bye"):
        console.print("\n  [dim]bye[/dim]\n")
        raise SystemExit(0)

    if cmd in ("clear", "cls"):
        os.system("cls" if os.name == "nt" else "clear")
        _boot(); return

    if cmd == "help":
        _header("HELP")
        rows = [
            ("ip <CIDR>",              "subnet visualizer"),
            ("check",                  "list tasks"),
            ("check add <task>",       "add a task"),
            ("check done <n>",         "toggle done"),
            ("check rm <n>",           "remove"),
            ("check clear",            "remove completed"),
            ("spin",                   "random idea  ·  -c visual  ·  -n 3"),
            ("gap <n1> <dob1> <n2> <dob2>", "lifespan overlap"),
            ("burn enc <message>",     "encode note"),
            ("burn dec <token>",       "decode note"),
            ("clear · exit",           ""),
        ]
        for c, d in rows:
            console.print(f"  [bright_green]{c:<32}[/bright_green][dim]{d}[/dim]")
        console.print(); return

    if cmd == "ip":
        if not rest: console.print("  usage: ip <CIDR>  e.g. ip 192.168.1.1/24"); return
        _do_ip(rest[0]); return

    if cmd == "check":
        _do_check(rest); return

    if cmd == "spin":
        _do_spin(rest); return

    if cmd == "gap":
        if len(rest) < 4:
            console.print("  usage: gap <name1> <YYYY-MM-DD> <name2> <YYYY-MM-DD>"); return
        _do_gap(rest[0], rest[1], rest[2], rest[3]); return

    if cmd == "burn":
        _do_burn(rest); return

    console.print(f"  [red]✗[/red]  unknown command: [bold]{cmd}[/bold]  [dim](type help)[/dim]")


# ═══════════════════════════════════════════════════════════════════════════
# REPL LOOP
# ═══════════════════════════════════════════════════════════════════════════

def _repl():
    _boot()
    while True:
        try:
            raw = input("hostlab  ").strip()
        except (EOFError, KeyboardInterrupt):
            console.print("\n  [dim]bye[/dim]\n")
            break
        if not raw:
            continue
        _dispatch(raw)


# ═══════════════════════════════════════════════════════════════════════════
# TYPER APP  (for scripting: hostlab ip ..., hostlab spin, etc.)
# ═══════════════════════════════════════════════════════════════════════════

app = typer.Typer(
    name="hostlab",
    help="HostLab terminal toolkit  —  run with no args for interactive mode",
    no_args_is_help=False,
    add_completion=False,
    rich_markup_mode="rich",
    invoke_without_command=True,
)
check_app = typer.Typer(no_args_is_help=False)
burn_app  = typer.Typer(no_args_is_help=True)
app.add_typer(check_app, name="check", help="Persistent checklist")
app.add_typer(burn_app,  name="burn",  help="Encode / decode notes")

@app.callback()
def _root(ctx: typer.Context):
    if ctx.invoked_subcommand is None:
        _repl()
        raise typer.Exit()

@app.command("ip")
def ip_cmd(address: str = typer.Argument(...)):
    """Subnet visualizer"""
    _do_ip(address)

@app.command("spin")
def spin_cmd(
    category: Optional[str] = typer.Option(None, "-c", "--category"),
    count: int = typer.Option(1, "-n", "--count"),
):
    """App idea spinner"""
    args = []
    if category: args += ["-c", category]
    if count != 1: args += ["-n", str(count)]
    _do_spin(args)

@app.command("gap")
def gap_cmd(
    name1: str, dob1: str, name2: str, dob2: str,
    lifespan: int = typer.Option(90, "-l", "--lifespan"),
):
    """Lifespan overlap visualizer"""
    _do_gap(name1, dob1, name2, dob2, lifespan)

@check_app.callback(invoke_without_command=True)
def check_default(ctx: typer.Context):
    """Persistent checklist"""
    if ctx.invoked_subcommand is None:
        _do_check([])

@check_app.command("add")
def check_add(text: str = typer.Argument(...)):
    _do_check(["add", text])

@check_app.command("done")
def check_done(n: int = typer.Argument(...)):
    _do_check(["done", str(n)])

@check_app.command("rm")
def check_rm(n: int = typer.Argument(...)):
    _do_check(["rm", str(n)])

@check_app.command("clear")
def check_clear():
    _do_check(["clear"])

@burn_app.command("enc")
def burn_enc(message: str = typer.Argument(...)):
    """Encode a message"""
    _do_burn(["enc", message])

@burn_app.command("dec")
def burn_dec(token: str = typer.Argument(...)):
    """Decode a token"""
    _do_burn(["dec", token])


def main():
    app()

if __name__ == "__main__":
    main()
