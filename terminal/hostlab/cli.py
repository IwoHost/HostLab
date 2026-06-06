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
    t.add_row("spin",             "random app idea combo  (-n 3 for more)")
    t.add_row("gap <n1> <dob1> <n2> <dob2>", "lifespan overlap")
    t.add_row("burn enc <msg>",   "encode a note")
    t.add_row("burn dec <token>", "decode a note")
    t.add_row("qr <url or text>", "generate a QR code (needs: pip install qrcode)")
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
_STYLES = [
    'Retro Terminal','Cyberpunk','Vaporwave','Dark Academia','Brutalist',
    'Neon Noir','Minimal Zen','Art Deco','Cottagecore','Memphis',
    'Glassmorphism','Neumorphism','Bauhaus','Psychedelic','Lo-Fi Pixel',
    'Swiss Grid','Bento Grid','Y2K Throwback','Solarpunk','Maximalist',
    'Monochrome Ink','Aurora','Industrial Grit','Kawaii Soft','Steampunk',
    'Futuristic Flat','Organic Blob','Bold Type','Comic Book','Deep Space',
    'Earthy Muted','High Contrast','Glitch Art','Corp. Memphis','Fantasy Map',
    'Isometric 3D','Hand-Drawn','Blueprint','Luxury Black','Paper Cut',
    'Neon Pastel','Retro Future','Analog Noise','Wabi-Sabi','Cybernetic',
    'Ocean Depth','Desert Sand','Nordic Frost','Tokyo Pop','Rainforest',
]
_TYPES = [
    'Dashboard','Portfolio','Social App','Dev Tool','Game',
    'Productivity','Marketplace','Data Viz','Landing Page','Note-Taking',
    'Music Player','Weather App','Habit Tracker','Chat UI','Code Editor',
]
_VIBES = [
    'Dark & Moody','Bright & Fun','Glitchy','Elegant','Chaotic',
    'Calm & Clean','Bold & Loud','Mysterious','Futuristic','Nostalgic',
    'Warm & Cozy','Cold & Sharp',
]

def _do_spin(args: list[str]):
    count = 1
    i = 0
    while i < len(args):
        if args[i] in ("-n", "--count") and i + 1 < len(args):
            try: count = int(args[i + 1])
            except: pass
            i += 2
        else:
            i += 1
    count = min(max(count, 1), 5)

    _header("SPIN")
    for idx in range(count):
        if idx: console.print()
        style = random.choice(_STYLES)
        kind  = random.choice(_TYPES)
        vibe  = random.choice(_VIBES)
        console.print(f"  [bright_green]{style}[/bright_green] [dim]×[/dim] [bright_green]{kind}[/bright_green] [dim]×[/dim] [bright_green]{vibe}[/bright_green]")
    console.print()
    console.print("  [dim]spin again · spin -n 3 for more[/dim]")
    console.print()


# ── qr ──────────────────────────────────────────────────────────────────────
def _do_qr(args: list[str]):
    text = " ".join(args)
    if not text:
        console.print("  usage: qr <url or text>"); return
    try:
        import qrcode as _qr
    except ImportError:
        console.print("  [red]✗[/red]  qrcode not installed — run: pip install qrcode")
        return

    qr = _qr.QRCode(border=2, error_correction=_qr.constants.ERROR_CORRECT_M)
    qr.add_data(text)
    qr.make(fit=True)

    _header("QR FORGE")
    for row in qr.modules:
        line = "  "
        for cell in row:
            # dark module = 2 spaces (dark bg), light module = 2 blocks (bright)
            line += "  " if cell else "██"
        console.print(line, markup=False)
    console.print()
    preview = text[:50] + ("…" if len(text) > 50 else "")
    console.print(f"  [dim]encoded · {preview}[/dim]")
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


# ── port lookup ──────────────────────────────────────────────────────────────
# (name, proto, desc, used, secure: True/False/None, tip)
_PORTS: dict[int, tuple] = {
    20:    ('FTP Data',         'TCP',     'FTP file transfer data channel',              'vsftpd · FileZilla',            False, 'use SFTP on port 22 instead'),
    21:    ('FTP',              'TCP',     'FTP control channel — login + commands',       'vsftpd · FileZilla · WinSCP',   False, 'unencrypted — prefer SFTP (22)'),
    22:    ('SSH / SFTP',       'TCP',     'Secure remote shell and file transfer',        'OpenSSH · PuTTY · all servers', True,  None),
    23:    ('Telnet',           'TCP',     'Unencrypted remote shell — legacy',            'legacy routers · old gear',     False, 'completely insecure — use SSH (22)'),
    25:    ('SMTP',             'TCP',     'Email relay between mail servers',             'Postfix · Sendmail · Exim',     False, 'server-to-server only; clients use 587'),
    53:    ('DNS',              'TCP+UDP', 'Translates domain names to IP addresses',      'BIND · Unbound · dnsmasq',      None,  None),
    67:    ('DHCP Server',      'UDP',     'Assigns IP addresses to devices',              'ISC DHCP · dnsmasq',            None,  None),
    68:    ('DHCP Client',      'UDP',     'Client side of DHCP IP assignment',            'all OS network stacks',         None,  None),
    69:    ('TFTP',             'UDP',     'Trivial file transfer — no auth, no encrypt',  'PXE boot · network devices',    False, 'no authentication whatsoever'),
    80:    ('HTTP',             'TCP',     'Unencrypted web traffic',                      'nginx · Apache · Caddy',        False, 'use HTTPS (443)'),
    88:    ('Kerberos',         'TCP+UDP', 'Network authentication protocol',              'Active Directory · MIT Kerberos',True, None),
    110:   ('POP3',             'TCP',     'Download email from server (older)',           'Thunderbird · Outlook',         False, 'use POP3S (995)'),
    111:   ('RPCBind',          'TCP+UDP', 'Maps RPC services to ports',                  'NFS stack · Sun RPC',           None,  None),
    123:   ('NTP',              'UDP',     'Syncs system clocks over the network',        'ntpd · chrony · timesyncd',     None,  None),
    135:   ('RPC',              'TCP',     'Windows remote procedure calls',              'Windows services',              None,  'often targeted — firewall externally'),
    137:   ('NetBIOS-NS',       'UDP',     'NetBIOS name service',                        'Windows networking',            None,  None),
    139:   ('NetBIOS-SSN',      'TCP',     'NetBIOS session service',                    'Windows file sharing',          None,  None),
    143:   ('IMAP',             'TCP',     'Access email on server (keeps messages)',      'Thunderbird · Outlook',         False, 'use IMAPS (993)'),
    161:   ('SNMP',             'UDP',     'Monitor and manage network devices',          'Nagios · Zabbix · routers',     False, 'use SNMPv3 with auth'),
    179:   ('BGP',              'TCP',     'Routes traffic between internet networks',    'ISP routers · Quagga · FRR',    None,  None),
    194:   ('IRC',              'TCP',     'Internet Relay Chat',                         'ircd · InspIRCd',               False, None),
    389:   ('LDAP',             'TCP+UDP', 'Directory services — users and groups',       'OpenLDAP · Active Directory',   False, 'use LDAPS (636)'),
    443:   ('HTTPS',            'TCP',     'Encrypted web traffic over TLS',             'nginx · Apache · Caddy',        True,  None),
    445:   ('SMB',              'TCP',     'Windows file and printer sharing',           'Windows · Samba',               None,  'block on internet-facing firewalls'),
    465:   ('SMTPS',            'TCP',     'SMTP wrapped in TLS (implicit)',              'mail servers',                  True,  None),
    500:   ('IKE / IPSec',      'UDP',     'IPSec VPN key exchange',                     'Cisco VPN · StrongSwan',        True,  None),
    514:   ('Syslog',           'UDP',     'Send system log messages to a log server',   'rsyslog · syslog-ng',           None,  None),
    587:   ('SMTP Submission',  'TCP',     'Client sends email to outgoing mail server', 'Thunderbird · Outlook',         True,  'use this + STARTTLS, not port 25'),
    636:   ('LDAPS',            'TCP',     'LDAP over TLS — secure directory access',    'OpenLDAP · Active Directory',   True,  None),
    853:   ('DNS-over-TLS',     'TCP',     'Encrypted DNS queries',                      'Unbound · Cloudflare 1.1.1.1',  True,  None),
    873:   ('rsync',            'TCP',     'Fast file sync and backup',                  'rsync daemon',                  None,  'use over SSH tunnel for encryption'),
    993:   ('IMAPS',            'TCP',     'IMAP over TLS — secure email access',        'all mail clients',              True,  None),
    995:   ('POP3S',            'TCP',     'POP3 over TLS — secure email download',      'all mail clients',              True,  None),
    1080:  ('SOCKS',            'TCP',     'Generic proxy protocol',                     'proxies · Tor · SSH tunnels',   None,  None),
    1194:  ('OpenVPN',          'TCP+UDP', 'Open-source VPN protocol',                   'OpenVPN client/server',         True,  None),
    1433:  ('MSSQL',            'TCP',     'Microsoft SQL Server database',              'SQL Server · SSMS',             None,  'never expose to internet'),
    1521:  ('Oracle DB',        'TCP',     'Oracle Database listener',                   'Oracle DB · SQL*Plus',          None,  'never expose to internet'),
    1723:  ('PPTP',             'TCP',     'Point-to-Point Tunneling Protocol',          'Windows VPN (legacy)',          False, 'broken encryption — do not use'),
    1883:  ('MQTT',             'TCP',     'Lightweight IoT messaging protocol',          'Mosquitto · Home Assistant',    False, 'use MQTT over TLS (8883)'),
    2049:  ('NFS',              'TCP+UDP', 'Network file system — share drives',         'Linux/Unix file servers',       None,  None),
    2181:  ('ZooKeeper',        'TCP',     'Distributed coordination service',            'Kafka · Hadoop · HBase',        None,  None),
    2375:  ('Docker',           'TCP',     'Docker daemon API — no TLS',                 'Docker',                        False, 'never expose — full root access'),
    2376:  ('Docker TLS',       'TCP',     'Docker daemon API over TLS',                 'Docker',                        True,  None),
    3000:  ('Dev Server',       'TCP',     'Common dev server port',                     'Node.js · Rails · Grafana',     None,  None),
    3306:  ('MySQL',            'TCP',     'MySQL and MariaDB database',                 'MySQL · MariaDB',               None,  'never expose to internet'),
    3389:  ('RDP',              'TCP',     'Windows Remote Desktop Protocol',            'Windows Remote Desktop',        None,  'brute-forced heavily — use VPN'),
    5000:  ('Flask / UPnP',     'TCP',     'Flask dev server · UPnP control',            'Flask · Docker registry',       None,  None),
    5432:  ('PostgreSQL',       'TCP',     'PostgreSQL database',                        'PostgreSQL',                    None,  'never expose to internet'),
    5601:  ('Kibana',           'TCP',     'Elasticsearch visualization UI',             'Kibana · ELK stack',            None,  None),
    5900:  ('VNC',              'TCP',     'Virtual Network Computing — remote desktop', 'TigerVNC · RealVNC',            False, 'use over SSH tunnel'),
    5984:  ('CouchDB',          'TCP',     'CouchDB database HTTP API',                  'CouchDB',                       None,  None),
    6379:  ('Redis',            'TCP',     'Redis in-memory cache and data store',       'Redis',                         False, 'no auth by default — bind to localhost'),
    6443:  ('Kubernetes API',   'TCP',     'Kubernetes API server',                      'kubectl · k8s control plane',   True,  None),
    6881:  ('BitTorrent',       'TCP+UDP', 'BitTorrent peer-to-peer file transfer',      'qBittorrent · Transmission',    None,  None),
    8000:  ('HTTP Alt',         'TCP',     'Alternate HTTP — common dev port',           'Django dev · Python http.server',None, None),
    8080:  ('HTTP Proxy / Alt', 'TCP',     'Common HTTP alt and proxy port',             'Tomcat · Jenkins · proxies',    None,  None),
    8443:  ('HTTPS Alt',        'TCP',     'Alternate HTTPS — Tomcat · panels',          'Tomcat · cPanel',               True,  None),
    8883:  ('MQTT TLS',         'TCP',     'MQTT over TLS — secure IoT messaging',       'Mosquitto · AWS IoT',           True,  None),
    8888:  ('Jupyter',          'TCP',     'Jupyter Notebook web interface',             'Jupyter · JupyterLab',          None,  'bind to localhost only'),
    9000:  ('PHP-FPM',          'TCP',     'PHP FastCGI process manager',               'PHP-FPM · SonarQube',           None,  None),
    9090:  ('Prometheus',       'TCP',     'Prometheus metrics server · Cockpit UI',     'Prometheus · Cockpit',          None,  None),
    9092:  ('Kafka',            'TCP',     'Apache Kafka message broker',               'Kafka',                         None,  None),
    9200:  ('Elasticsearch',    'TCP',     'Elasticsearch REST API',                    'Elasticsearch · ELK stack',     False, 'no auth by default — never expose'),
    11211: ('Memcached',        'TCP+UDP', 'Memcached distributed memory cache',        'Memcached',                     False, 'no auth — bind to localhost only'),
    15672: ('RabbitMQ UI',      'TCP',     'RabbitMQ management web interface',         'RabbitMQ',                      None,  None),
    27017: ('MongoDB',          'TCP',     'MongoDB database',                          'MongoDB',                       False, 'no auth by default — bind to localhost'),
    51820: ('WireGuard',        'UDP',     'Modern fast VPN protocol',                  'WireGuard',                     True,  None),
}

def _do_port(args: list[str]):
    if not args:
        console.print("  usage: port <number>  e.g. port 443")
        console.print("  tip:   port 443 80 22  for multiple at once")
        return
    nums = []
    for a in args:
        try: nums.append(int(a))
        except ValueError: console.print(f"  [red]✗[/red]  not a number: {a}")
    for idx, num in enumerate(nums):
        if idx: console.print()
        p = _PORTS.get(num)
        _header(f"PORT {num}")
        if not p:
            console.print("  [dim]not in database — may be unassigned or custom[/dim]")
            console.print(); continue
        name, proto, desc, used, secure, tip = p
        sec_str = "[bright_green]✓ encrypted[/bright_green]" if secure is True else \
                  "[red]✗ unencrypted[/red]" if secure is False else "[dim]—[/dim]"
        console.print(f"  [bright_green]{name:<24}[/bright_green][dim]{proto}[/dim]")
        console.print(f"  {desc}")
        console.print()
        if used: console.print(f"  [dim]used by  [/dim]{used}")
        console.print(f"  [dim]secure   [/dim]{sec_str}")
        if tip:  console.print(f"  [dim]note     [/dim][yellow]{tip}[/yellow]")
        console.print()


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
            ("spin",                   "random app combo  ·  spin -n 3"),
            ("gap <n1> <dob1> <n2> <dob2>", "lifespan overlap"),
            ("burn enc <message>",     "encode note"),
            ("burn dec <token>",       "decode note"),
            ("qr <url or text>",       "QR code in terminal"),
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

    if cmd == "qr":
        _do_qr(rest); return

    if cmd == "port":
        _do_port(rest); return

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
def spin_cmd(count: int = typer.Option(1, "-n", "--count")):
    """Random app idea combo"""
    _do_spin(["-n", str(count)] if count != 1 else [])

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

@app.command("qr")
def qr_cmd(text: list[str] = typer.Argument(...)):
    """Generate a QR code in the terminal"""
    _do_qr(list(text))

@app.command("port")
def port_cmd(numbers: list[str] = typer.Argument(...)):
    """Look up what a port number is used for"""
    _do_port(list(numbers))


def main():
    app()

if __name__ == "__main__":
    main()
