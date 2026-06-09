# HostLab

A personal collection of browser-based tools and experiments at `iwohost.github.io/HostLab`.

## Stack

- **Pure vanilla** HTML/CSS/JS — no framework, no build step
- Each tool is a **single self-contained `.html` file** in `/websites/`
- Shared: `other/backbutton.css` (back nav), analytics + Cookiebot in every `<head>`
- GitHub Pages deployment from `main` branch

## File locations

| Path | What lives there |
|------|-----------------|
| `index.html` | Main hub — card grid, tag filter, JS data array |
| `websites/*.html` | Individual tools (34+) |
| `pages/` | Category pages (games, study, archives) |
| `other/logo/` | Per-project PNG icons |
| `other/backbutton.css` | Shared circular back-button |
| `workers/` | Cloudflare Workers (wrangler) |
| `terminal/` | Terminal implementation |

## Adding a new tool

1. Create `websites/your-tool.html` — use `/vibe` skill for HostLab-native generation
2. Add an entry to the JS data array in `index.html` (look for `const projects = [`)
3. Add a 16:9 thumbnail PNG to `other/` prefixed with `p_`
4. Add an icon PNG to `other/logo/`

## Design rules

- All colors via CSS custom properties (`--bg`, `--accent`, etc.) in `:root`
- Monospace-first typography: `Syne Mono`, `Share Tech Mono`, `VT323`, `DM Mono`
- Dark themes, per-project personality, retro/terminal aesthetic
- 1px solid borders, tight spacing, micro-animations (fadeUp, blink, flicker)
- Always include: bottom status bar, header with tool name, mobile breakpoint

## Analytics (copy into every new page head, top of `<head>`)

```html
<script id="Cookiebot" src="https://consent.cookiebot.com/uc.js" data-cbid="6ebe47b9-15fa-4d2b-a359-233f4d1d0178" data-blockingmode="auto" type="text/javascript"></script>
<script type="text/plain" data-cookiecategory="statistics" async src="https://www.googletagmanager.com/gtag/js?id=G-R7MJHBKJVY"></script>
<script type="text/plain" data-cookiecategory="statistics">
    window.dataLayer = window.dataLayer || [];
    function gtag(){dataLayer.push(arguments);}
    gtag('js', new Date());
    gtag('config', 'G-R7MJHBKJVY');
</script>
```

## Custom skills

- `/vibe [tool name] — [description] — [optional style hint]` — generate a new HostLab page
