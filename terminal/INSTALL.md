# HostLab Terminal — Install Guide

## Windows (PowerShell or Windows Terminal)

**1. Make sure Python 3.10+ is installed**
Download from https://python.org — tick "Add Python to PATH" during install.

**2. Install the toolkit**
```powershell
cd terminal
pip install -e .
```

**3. Run it**
```powershell
hostlab --help
hostlab ip 192.168.1.50/24
hostlab check add "fix the bug"
hostlab spin -c fun
```

---

## Linux / macOS

```bash
cd terminal
pip install -e .
hostlab --help
```

---

## Commands

| Command | What it does |
|---|---|
| `hostlab ip 192.168.1.50/24` | Subnet breakdown + binary octets |
| `hostlab check` | List your tasks |
| `hostlab check add "task"` | Add a task |
| `hostlab check done 1` | Toggle task #1 done |
| `hostlab check rm 1` | Remove task #1 |
| `hostlab check clear` | Remove all completed |
| `hostlab spin` | Random app idea |
| `hostlab spin -c visual -n 3` | 3 visual ideas |
| `hostlab gap Alice 1990-05-15 Bob 1985-03-20` | Lifespan overlap |
| `hostlab burn enc "message"` | Encode a note |
| `hostlab burn dec <token>` | Decode a note |
