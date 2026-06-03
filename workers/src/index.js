// HostLab Terminal — Cloudflare Worker
// Responds to curl and Lynx with plain text / minimal HTML

// ── helpers ────────────────────────────────────────────────────────────────
const ipToInt = s => s.split('.').reduce((a, n) => ((a << 8) | +n) >>> 0, 0);
const intToIP = n => [(n>>>24)&255,(n>>>16)&255,(n>>>8)&255,n&255].join('.');

function addrKind(n) {
  const A = (n>>>24)&255, B = (n>>>16)&255;
  if (A===127) return 'loopback';
  if (A===10)  return 'private';
  if (A===172 && B>=16 && B<=31) return 'private';
  if (A===192 && B===168) return 'private';
  if (A===169 && B===254) return 'link-local';
  return 'public';
}

function parseArgs(raw) {
  const out = []; let cur = '', inQ = false, qc = '';
  for (const ch of raw) {
    if (inQ) { if (ch===qc) inQ=false; else cur+=ch; }
    else if (ch==='"'||ch==="'") { inQ=true; qc=ch; }
    else if (ch===' ') { if (cur) { out.push(cur); cur=''; } }
    else cur+=ch;
  }
  if (cur) out.push(cur);
  return out;
}

const line = (w=50) => '─'.repeat(w);

// ══════════════════════════════════════════════════════════════════════════
// IP VISUALIZER
// ══════════════════════════════════════════════════════════════════════════
function doIP(args) {
  const address = args[0];
  if (!address) return 'usage: ip 192.168.1.1/24';

  const [ipStr, pStr='32'] = address.split('/');
  const prefix = parseInt(pStr);

  if (!/^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(ipStr)
      || ipStr.split('.').some(o=>+o>255)
      || isNaN(prefix) || prefix<0 || prefix>32)
    return `✗  invalid address: ${address}`;

  const ipInt    = ipToInt(ipStr);
  const maskInt  = prefix===0 ? 0 : ((0xFFFFFFFF << (32-prefix)) >>> 0);
  const netInt   = (ipInt & maskInt) >>> 0;
  const bcastInt = (netInt | (~maskInt>>>0)) >>> 0;
  const total    = Math.pow(2, 32-prefix);
  const usable   = Math.max(0, total-2);

  const rows = [
    ['Address',      `${ipStr}  (${addrKind(ipInt)})`],
    ['Network',      intToIP(netInt)],
    ['Broadcast',    intToIP(bcastInt)],
    ['Subnet mask',  intToIP(maskInt)],
    ['Wildcard',     intToIP((~maskInt)>>>0)],
    ['Prefix',       `/${prefix}`],
    ['Total IPs',    total.toLocaleString()],
    ['Usable hosts', usable.toLocaleString()],
  ];
  if (usable>=1) rows.push(['First host', intToIP(netInt+1)]);
  if (usable>=2) rows.push(['Last host',  intToIP(bcastInt-1)]);

  let out = `\n${line()}\n  IP VISUALIZER\n${line()}\n\n`;
  rows.forEach(([k,v]) => out += `  ${k.padEnd(16)} ${v}\n`);

  out += `\n  binary  (NET = uppercase)\n\n`;
  const labels = ['A','B','C','D'];
  for (let o=0; o<4; o++) {
    const sh = (3-o)*8;
    const ib = (ipInt   >>> sh) & 255;
    const mb = (maskInt >>> sh) & 255;
    let bits = '';
    for (let b=7; b>=0; b--) {
      const val  = (ib>>>b)&1;
      const isNet = !!((mb>>>b)&1);
      bits += isNet ? String(val).toUpperCase() : String(val);
      if (b===4) bits += ' ';
    }
    out += `  ·${labels[o]}·  ${bits}  ${ib.toString().padStart(3)}\n`;
  }
  out += '\n';
  return out;
}

// ══════════════════════════════════════════════════════════════════════════
// SPIN
// ══════════════════════════════════════════════════════════════════════════
const IDEAS = {
  visual: [
    ['Pixel Sorter',            'Sort pixels by hue or brightness in real time'],
    ['ASCII Cam',               'Turn a webcam feed into live ASCII art'],
    ['Gradient Mesh Editor',    'Drag control points to sculpt smooth color gradients'],
    ['Reaction Diffusion',      'Interactive Turing-pattern simulator on a canvas'],
    ['Chromatic Aberration Lab','Apply lens-split RGB fringing to any uploaded image'],
    ['Halftone Engine',         'Simulate newspaper halftone at variable screen angles'],
    ['Voronoi Painter',         'Click to drop seeds and watch a Voronoi diagram grow'],
    ['Glitch Generator',        'Apply datamoshing and compression artifacts to images'],
  ],
  audio: [
    ['Chord Namer',             'Click piano keys and instantly see the chord name'],
    ['Binaural Beat Gen',       'Set two slightly-offset tones for focus or sleep'],
    ['Lo-fi Degrader',          'Apply vinyl crackle and tape hiss to any audio'],
    ['Arpeggiator',             'Play a chord and loop through its notes in patterns'],
    ['Spectral Freeze',         'Freeze a moment of sound and let it drone forever'],
    ['Euclidean Drummer',       'Generative drum machine built on Euclidean rhythms'],
  ],
  productivity: [
    ['Habit Punch Card',        'Heatmap-style tracker for daily habit streaks'],
    ['Timezone Overlap',        'Find meeting windows across multiple time zones'],
    ['Meeting Cost Clock',      'Watch money drain as your meeting runs over'],
    ['One-liner Journal',       'Date-stamped single lines — simple, searchable log'],
    ['Decision Matrix',         'Weighted criteria table that scores options fairly'],
  ],
  utility: [
    ['Regex Explainer',         'Match groups with human-readable step-by-step output'],
    ['Color Blind Sim',         'Preview any image through 8 types of color vision'],
    ['Password Entropy Meter',  'Crack-time estimate and actionable suggestions'],
    ['Contrast Ratio Checker',  'Pick two colors, get WCAG pass/fail instantly'],
    ['Base Converter',          'Type in any numeric base, all others update live'],
  ],
  data: [
    ['Spending Heatmap',        'Paste a CSV bank export → calendar heatmap'],
    ['Word Frequency Map',      'Paste text → word cloud sorted by count'],
    ['Sort Algorithm Race',     'Watch bubble, merge, and quicksort compete live'],
    ['Network Graph Builder',   'Type node-edge pairs, see a force-directed graph'],
  ],
  fun: [
    ['Vibe Checker',            '5 sliders → your totally unscientific vibe score'],
    ['Excuse Generator',        '500 plausible excuses by situation and severity'],
    ['Commit Message Oracle',   'Suspiciously accurate random commit messages'],
    ['Fake Loading Screen',     'Convincing progress bar with dramatic log output'],
    ['Which Framework Are You?','5 questions → get roasted by a JS framework'],
    ['Keyboard Smash Analyzer', 'Rate the quality of your asdfghjkl moments'],
  ],
};
const ALL_IDEAS = Object.entries(IDEAS).flatMap(([cat,items])=>items.map(([n,d])=>[cat,n,d]));

function doSpin(args) {
  let cat=null, count=1;
  for (let i=0; i<args.length; i++) {
    if ((args[i]==='-c'||args[i]==='--category') && args[i+1]) { cat=args[++i]; }
    else if ((args[i]==='-n'||args[i]==='--count') && args[i+1]) { count=parseInt(args[++i])||1; }
  }

  let pool = cat ? (IDEAS[cat]||[]).map(([n,d])=>[cat,n,d]) : ALL_IDEAS;
  if (cat && !IDEAS[cat])
    return `✗  unknown category: ${cat}\n   options: ${Object.keys(IDEAS).join('  ')}`;

  count = Math.min(count, pool.length, 9);
  const picks=[], tmp=[...pool];
  for (let i=0; i<count; i++) {
    const idx = Math.floor(Math.random()*tmp.length);
    picks.push(tmp.splice(idx,1)[0]);
  }

  let out = `\n${line()}\n  SPIN\n${line()}\n\n`;
  picks.forEach(([c,name,desc],i)=>{
    if (i) out+='\n';
    out += `  [${c}]\n  ${name}\n  ${desc}\n`;
  });
  out += `\n  spin again · spin -n 3 · spin -c fun\n\n`;
  return out;
}

// ══════════════════════════════════════════════════════════════════════════
// GAP VISUALIZER
// ══════════════════════════════════════════════════════════════════════════
function doGap(args) {
  if (args.length < 4)
    return 'usage: gap <name1> <YYYY-MM-DD> <name2> <YYYY-MM-DD>\ne.g.   gap Alice 1990-05-15 Bob 1985-03-20';

  const [name1, dob1s, name2, dob2s] = args;
  const d1 = new Date(dob1s+'T00:00:00'), d2 = new Date(dob2s+'T00:00:00');
  if (isNaN(d1)) return `✗  invalid date: ${dob1s}`;
  if (isNaN(d2)) return `✗  invalid date: ${dob2s}`;

  const LIFE      = 90;
  const spanMs    = LIFE * 365.25 * 86400000;
  const death1    = new Date(d1.getTime()+spanMs);
  const death2    = new Date(d2.getTime()+spanMs);
  const today     = new Date();
  const ovStart   = d1>d2 ? d1 : d2;
  const ovEnd     = death1<death2 ? death1 : death2;
  const fullOv    = Math.max(0, Math.round((ovEnd-ovStart)/86400000));
  const livedEnd  = ovEnd<today ? ovEnd : today;
  const livedOv   = Math.max(0, Math.round((livedEnd-ovStart)/86400000));
  const spanDays  = Math.round(LIFE*365.25);
  const pctFull   = (fullOv/spanDays*100).toFixed(1);
  const pctLived  = (livedOv/spanDays*100).toFixed(1);
  const gapDays   = Math.abs(Math.round((d2-d1)/86400000));
  const younger   = d1<=d2 ? name2 : name1;
  const fmt = d => `${d.getDate()} ${d.toLocaleString('en',{month:'long'})} ${d.getFullYear()}`;

  let out = `\n${line()}\n  GAP VISUALIZER\n${line()}\n\n`;
  const rows = [
    [`${name1} born`,     fmt(d1)],
    [`${name2} born`,     fmt(d2)],
    ['Age gap',           `${(gapDays/365.25).toFixed(1)} yrs  (${gapDays.toLocaleString()} days)`],
    ['Overlap starts',    `${fmt(ovStart)}  (when ${younger} was born)`],
    ['Potential overlap', `${fullOv.toLocaleString()} days  (${pctFull}% of a ${LIFE}-yr life)`],
    ['Lived so far',      `${livedOv.toLocaleString()} days  (${pctLived}%)`],
  ];
  rows.forEach(([k,v]) => out += `  ${k.padEnd(22)} ${v}\n`);

  const BAR=50, earliest=d1<d2?d1:d2;
  const totalSpan = (Math.max(death1,death2)-earliest)/86400000;

  function bar(birth, name) {
    const s = Math.round((birth-earliest)/86400000/totalSpan*BAR);
    const l = Math.max(1, Math.min(Math.round(spanDays/totalSpan*BAR), BAR-s));
    return `  ${name.padEnd(10)}  ${'·'.repeat(s)}${'█'.repeat(l)}${'·'.repeat(Math.max(0,BAR-s-l))}`;
  }

  const ovS = Math.round((ovStart-earliest)/86400000/totalSpan*BAR);
  const ovL = Math.max(1, Math.round(fullOv/totalSpan*BAR));

  out += `\n  Timeline  (${LIFE}-year lifespan assumed)\n\n`;
  out += bar(d1, name1) + '\n';
  out += bar(d2, name2) + '\n\n';
  out += `  ${'overlap'.padEnd(10)}  ${'·'.repeat(ovS)}${'▓'.repeat(Math.min(ovL,BAR-ovS))}${'·'.repeat(Math.max(0,BAR-ovS-ovL))}  ${pctFull}%\n\n`;
  return out;
}

// ══════════════════════════════════════════════════════════════════════════
// BURN NOTES
// ══════════════════════════════════════════════════════════════════════════
function doBurn(args) {
  const sub = (args[0]||'').toLowerCase();

  if (sub==='enc') {
    const msg = args.slice(1).join(' ');
    if (!msg) return 'usage: burn enc <message>';
    const token = btoa(encodeURIComponent(msg)).replace(/\+/g,'-').replace(/\//g,'_').replace(/=/g,'');
    return `\n${line()}\n  BURN — ENCODE\n${line()}\n\n  token:\n\n  ${token}\n\n  decode with:  burn dec ${token.slice(0,20)}${token.length>20?'…':''}\n\n`;
  }

  if (sub==='dec') {
    const token = args[1]||'';
    if (!token) return 'usage: burn dec <token>';
    try {
      const pad = token.replace(/-/g,'+').replace(/_/g,'/')+('='.repeat((4-token.length%4)%4));
      const msg = decodeURIComponent(atob(pad));
      return `\n${line()}\n  BURN — DECODE\n${line()}\n\n  ${msg}\n\n`;
    } catch {
      return '✗  invalid or corrupted token';
    }
  }

  return 'usage: burn enc <message>  ·  burn dec <token>';
}

// ══════════════════════════════════════════════════════════════════════════
// DISPATCHER
// ══════════════════════════════════════════════════════════════════════════
function makeHelp(base) {
  const u = s => `curl "${base}?cmd=${s}"`;
  return `
${line()}
  HOSTLAB TERMINAL
${line()}

  ${u('help')}
  ${u('ip+192.168.1.1%2F24')}
  ${u('spin')}
  ${u('spin+-c+fun')}
  ${u('spin+-n+3')}
  ${u('gap+Alice+1990-05-15+Bob+1985-03-20')}
  ${u('burn+enc+your+message+here')}
  ${u('burn+dec+TOKEN')}

${line()}
  commands
${line()}

  ip <CIDR>                     subnet visualizer
  spin                          random app idea
  spin -c <cat>                 visual  audio  productivity  utility  data  fun
  spin -n <count>               show multiple ideas
  gap <n1> <dob1> <n2> <dob2>  lifespan overlap  (YYYY-MM-DD)
  burn enc <message>            encode a note
  burn dec <token>              decode a note

${line()}
`;
}

function dispatch(raw, base) {
  const parts = parseArgs(raw.trim());
  if (!parts.length) return makeHelp(base);
  const cmd = parts[0].toLowerCase();
  const args = parts.slice(1);

  if (cmd==='ip')   return doIP(args);
  if (cmd==='spin') return doSpin(args);
  if (cmd==='gap')  return doGap(args);
  if (cmd==='burn') return doBurn(args);
  if (cmd==='help') return makeHelp(base);
  return `✗  unknown command: ${cmd}\n   type 'help' to see commands\n`;
}

// ══════════════════════════════════════════════════════════════════════════
// HTML FOR LYNX
// ══════════════════════════════════════════════════════════════════════════
function lynxPage(result='', prevCmd='') {
  return `<!DOCTYPE html>
<html>
<head><title>HostLab Terminal</title></head>
<body>
<pre>
 HOSTLAB TERMINAL  ·  github.com/iwohost/HostLab
${line()}
</pre>
<form method="GET" action="/">
  command: <input name="cmd" size="55" value="${prevCmd.replace(/"/g,'&quot;')}">
  <input type="submit" value="run">
</form>
${result ? `<pre>${result.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')}</pre>` : `<pre>${HELP.replace(/&/g,'&amp;')}</pre>`}
</body>
</html>`;
}

// ══════════════════════════════════════════════════════════════════════════
// WORKER ENTRY
// ══════════════════════════════════════════════════════════════════════════
export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const cmd = (url.searchParams.get('cmd') || '').trim();

    // API key check (optional — set via: wrangler secret put API_KEY)
    if (env.API_KEY) {
      const key = url.searchParams.get('key') || request.headers.get('x-api-key') || '';
      if (key !== env.API_KEY) {
        return new Response('unauthorized\n', { status: 401, headers: { 'content-type': 'text/plain' } });
      }
    }

    const base = `${url.origin}/`;
    const ua = (request.headers.get('user-agent') || '').toLowerCase();
    const isLynx = ua.includes('lynx');

    // Lynx: return minimal HTML with a form
    if (isLynx) {
      const result = cmd ? dispatch(cmd, base) : '';
      return new Response(lynxPage(result, cmd), {
        headers: { 'content-type': 'text/html; charset=utf-8' },
      });
    }

    // curl / wget / browser / everything else: plain text
    const result = cmd ? dispatch(cmd, base) : makeHelp(base);
    return new Response(result, {
      headers: { 'content-type': 'text/plain; charset=utf-8' },
    });
  },
};
