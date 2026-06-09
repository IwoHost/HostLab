# /vibe — HostLab Page Generator

**Usage:** `/vibe [tool name] — [short description] — [optional: style hint]`

**Examples:**
- `/vibe Pixel Clock — a live clock with pixel-art digits`
- `/vibe Mood Board — drag-and-drop color palette builder — warm brutalist`
- `/vibe Noise Lab — interactive Perlin noise texture generator — dark sci-fi`
- `/vibe just vibe` ← pick a random fun tool idea and surprise the user

---

You are building a new self-contained page for **HostLab** — a personal collection of browser-based tools at `iwohost.github.io/HostLab`. Every page is a **single `.html` file** in `/websites/` with all CSS and JS inline. No frameworks, no build step, no CDN libraries unless absolutely essential (qrcode, waveform libs are fine).

## Stack rules

- Vanilla HTML5 / CSS3 / JS (ES2022+)
- All `<style>` inline in `<head>`
- All `<script>` inline before `</body>`
- Google Fonts via `<link>` in head (always include at least one from the list below)
- Always include the analytics + Cookiebot block at the very top of `<head>` (copy exactly from existing pages)
- Always include `<link rel="stylesheet" href="../other/backbutton.css">` for the back button
- Use `localStorage` for any state that should persist
- Use Canvas API, Web Audio API, CSS animations — whatever fits the tool natively

## Design system

### CSS custom properties — always define in `:root`
Every page picks ONE theme personality from the list below (or invent a new one that fits the tool). Use this as the foundation:

```css
/* THEME: [name] */
:root {
  --bg: #…;        /* page background — very dark */
  --panel: #…;     /* slightly lighter panels/sidebars */
  --border: #…;    /* subtle divider lines */
  --border2: #…;   /* more visible borders */
  --text: #…;      /* primary readable text */
  --text2: #…;     /* secondary/dim labels */
  --text3: #…;     /* very muted, ghost text */
  --accent: #…;    /* main interactive color */
  --accent2: #…;   /* hover/brighter variant */
  --accent3: #…;   /* lightest tint */
  --mono: FontName, monospace;
}
```

### Established themes (for inspiration or reuse):
| Name | Accent | Feel |
|------|--------|------|
| Neon Cyberpunk | `#4444ff` / `#ff00ff` / `#00ffff` | Dither Lab — electric, retro-future |
| Amber Burn | `#ff9500` | Burn Notes — warm, urgent, vintage |
| Teal Signal | `#00c9a0` | Signal/Noise Radio — calm, clinical |
| Lime Terminal | `#00ff41` | VoxSynth — pure terminal green |
| Lavender Drift | `#e8d0ff` | Gap Relativity — soft, ethereal |
| Gold Warm | `#c8a96e` | Checklist — earthy, grounded |
| Rust Focus | `#c8784a` | Focus Guy — warm, productive |
| QR Forge | `#00cc44` / `#aaffcc` | Clean data-green |

### NEW themes to try (fresh, polished, haven't been used yet):
| Name | Accent | Feel |
|------|--------|------|
| Bone Noir | `#e8e0d0` on `#0a0908` | Minimal film-noir — ink on parchment |
| Blood Sport | `#ff2244` on `#0d0305` | Intense red — fast, dangerous |
| Ice Station | `#a8d8ff` on `#01080f` | Cold blue — precision instrument |
| Moss Lab | `#7ec87e` on `#080c08` | Muted organic green — calm hacker |
| Void Rose | `#ff8fc0` on `#0a0508` | Soft pink on deep void — dreamy |
| Copper Circuit | `#d4824a` on `#0b0805` | Warm copper — steampunk machine |
| UV Pulse | `#cc88ff` + `#ff44cc` | Deep purple neon — club, glitchy |
| Static Grey | `#c0c0b8` on `#0c0c0c` | Clean minimal — Swiss design meets terminal |

### Typography rules
- **Always use at least one of these font families** (already in your Google Fonts rotation):
  - `Syne` — display headings, hub pages
  - `Syne Mono` — metadata, labels, hub
  - `Share Tech Mono` — terminal/data UIs
  - `VT323` — retro/pixel flavor text
  - `DM Mono` — clean readable mono
  - `Bebas Neue` — big impact headers
  - `Fraunces` — editorial, organic serif contrast
  - `Playfair Display` — elegant, high-contrast

- **Font loading link example:**
  ```html
  <link href="https://fonts.googleapis.com/css2?family=Share+Tech+Mono&family=VT323&display=swap" rel="stylesheet">
  ```

- Labels and metadata: `font-size: 9-11px; letter-spacing: 0.2em; text-transform: uppercase;`
- Body text: `font-size: 12-13px; line-height: 1.5;`
- Big display: `font-size: clamp(36px, 6vw, 72px);`

### Layout patterns
Use grid for the main layout. Pick the one that fits:

```css
/* 3-panel tool (left controls / center canvas / right stats) */
#app { display:grid; grid-template-rows:36px 1fr 40px; grid-template-columns:180px 1fr 200px; height:100vh; }

/* 2-panel (sidebar + main) */
#app { display:grid; grid-template-columns:240px 1fr; height:100vh; }

/* Full-width with header/footer */
#app { display:grid; grid-template-rows:48px 1fr 40px; height:100vh; }

/* Card grid (gallery/list tools) */
.grid { display:grid; grid-template-columns:repeat(auto-fill, minmax(280px, 1fr)); gap:1px; }
```

### Mandatory micro-details (make it feel alive)
Always include at least 3 of these:
- `@keyframes fadeUp { from { opacity:0; transform:translateY(8px) } to { opacity:1; transform:translateY(0) } }` on main content
- `@keyframes blink { 0%,100%{opacity:1} 50%{opacity:0} }` on cursor elements
- `@keyframes flicker` subtle opacity variation for neon elements
- Scanlines via `repeating-linear-gradient(0deg, transparent 3px, rgba(0,0,0,.10) 3px)` as a pseudo-element overlay
- Glow text: `text-shadow: 0 0 20px rgba(accent, 0.6)`
- Hover lift: `transition: transform 0.2s; &:hover { transform: translateY(-2px); }`
- A status bar / info strip at the bottom of the page showing metadata
- Corner decoration: `position:absolute; corners with border lines` (like a scope crosshair)

### Header pattern (standard — use unless the tool vibe needs something different)
```html
<div id="header">
  <span id="logo">TOOL<span>·</span>NAME</span>
  <div id="hinfo">
    <span>STATUS <b id="status-val">READY</b></span>
    <span>VER <b>1.0</b></span>
  </div>
</div>
```

### Bottom status bar (always include)
```html
<div id="footer">
  <span id="fl">HOSTLAB / TOOL-NAME</span>
  <span id="fc" id="center-info">——</span>
  <span id="fr">© 2025</span>
</div>
```

### Back button (already handled by the CSS link — no extra HTML needed)
The `backbutton.css` injects a fixed circular button. Nothing else needed.

## Quality checklist before outputting the page
- [ ] Analytics + Cookiebot block at very top of head
- [ ] Correct `og:` meta tags (title, description, type, url, image)
- [ ] Google Fonts link present
- [ ] `backbutton.css` link present
- [ ] CSS uses `--variables` for ALL colors (no hardcoded hex outside `:root`)
- [ ] Tool is actually functional (not a stub) — has real working JS logic
- [ ] At least 3 micro-details from the list above
- [ ] Mobile responsive (`@media (max-width: 768px)` breakpoint)
- [ ] Readable at 12px mono (test your font size choices)
- [ ] Status bar footer is present
- [ ] The tool has a clear and satisfying core loop (something to DO with it)

## Tone + ambition
- Each tool should feel like a **deliberate creative choice**, not a template fill-in
- The color theme should feel **inevitable for that tool's personality**
- Weird is good. Unexpected font pairings are good. Unusual layouts are good.
- BUT: it should still look **clean** — 1px borders, tight spacing, good contrast ratios
- Aim for something that feels like it belongs in HostLab AND pushes the collection forward

## Output format
1. First: one short sentence on the theme/vibe choice and why it fits
2. Then: the complete, working HTML file — nothing truncated
3. After the file: a one-liner on what to add to `index.html` (the JS data array entry)

If the user wrote `/vibe just vibe` — pick a surprising but genuinely useful tool idea yourself, explain the concept in 2 sentences, then build it.
