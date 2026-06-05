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
const STYLES = [
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
];
const TYPES = [
  'Dashboard','Portfolio','Social App','Dev Tool','Game',
  'Productivity','Marketplace','Data Viz','Landing Page','Note-Taking',
  'Music Player','Weather App','Habit Tracker','Chat UI','Code Editor',
];
const VIBES = [
  'Dark & Moody','Bright & Fun','Glitchy','Elegant','Chaotic',
  'Calm & Clean','Bold & Loud','Mysterious','Futuristic','Nostalgic',
  'Warm & Cozy','Cold & Sharp',
];

function doSpin(args) {
  let count = 1;
  for (let i=0; i<args.length; i++) {
    if ((args[i]==='-n'||args[i]==='--count') && args[i+1]) { count=Math.min(parseInt(args[++i])||1,5); }
  }
  const pick = arr => arr[Math.floor(Math.random()*arr.length)];
  let out = `\n${line()}\n  SPIN\n${line()}\n\n`;
  for (let i=0; i<count; i++) {
    if (i) out += '\n';
    out += `  ${pick(STYLES)} × ${pick(TYPES)} × ${pick(VIBES)}\n`;
  }
  out += `\n  spin again · spin -n 3 for more\n\n`;
  return out;
}

// ══════════════════════════════════════════════════════════════════════════
// PORT LOOKUP
// ══════════════════════════════════════════════════════════════════════════
// [name, proto, desc, used, secure: 1=yes 0=no null=neutral, tip]
const PORT_DB = {
  20:['FTP Data','TCP','FTP file transfer data channel','vsftpd · FileZilla',0,'use SFTP on port 22 instead'],
  21:['FTP','TCP','FTP control channel — login + commands','vsftpd · FileZilla · WinSCP',0,'unencrypted — prefer SFTP (22)'],
  22:['SSH / SFTP','TCP','Secure remote shell and file transfer','OpenSSH · PuTTY · all servers',1,null],
  23:['Telnet','TCP','Unencrypted remote shell — legacy','legacy routers · old gear',0,'completely insecure — use SSH (22)'],
  25:['SMTP','TCP','Email relay between mail servers','Postfix · Sendmail · Exim',0,'server-to-server only; clients use 587'],
  53:['DNS','TCP+UDP','Translates domain names to IP addresses','BIND · Unbound · dnsmasq',null,null],
  67:['DHCP Server','UDP','Assigns IP addresses to devices','ISC DHCP · dnsmasq',null,null],
  68:['DHCP Client','UDP','Client side of DHCP IP assignment','all OS network stacks',null,null],
  69:['TFTP','UDP','Trivial file transfer — no auth, no encrypt','PXE boot · network devices',0,'no authentication whatsoever'],
  80:['HTTP','TCP','Unencrypted web traffic','nginx · Apache · Caddy',0,'use HTTPS (443)'],
  88:['Kerberos','TCP+UDP','Network authentication protocol','Active Directory · MIT Kerberos',1,null],
  110:['POP3','TCP','Download email from server (older)','Thunderbird · Outlook',0,'use POP3S (995)'],
  111:['RPCBind','TCP+UDP','Maps RPC services to ports','NFS stack · Sun RPC',null,null],
  123:['NTP','UDP','Syncs system clocks over the network','ntpd · chrony · timesyncd',null,null],
  135:['RPC','TCP','Windows remote procedure calls','Windows services',null,'often targeted — firewall externally'],
  139:['NetBIOS-SSN','TCP','NetBIOS session service','Windows file sharing',null,null],
  143:['IMAP','TCP','Access email on server (keeps messages)','Thunderbird · Outlook',0,'use IMAPS (993)'],
  161:['SNMP','UDP','Monitor and manage network devices','Nagios · Zabbix · routers',0,'use SNMPv3 with auth'],
  179:['BGP','TCP','Routes traffic between internet networks','ISP routers · Quagga · FRR',null,null],
  194:['IRC','TCP','Internet Relay Chat','ircd · InspIRCd',0,null],
  389:['LDAP','TCP+UDP','Directory services — users and groups','OpenLDAP · Active Directory',0,'use LDAPS (636)'],
  443:['HTTPS','TCP','Encrypted web traffic over TLS','nginx · Apache · Caddy',1,null],
  445:['SMB','TCP','Windows file and printer sharing','Windows · Samba',null,'block on internet-facing firewalls'],
  465:['SMTPS','TCP','SMTP wrapped in TLS (implicit)','mail servers',1,null],
  500:['IKE / IPSec','UDP','IPSec VPN key exchange','Cisco VPN · StrongSwan',1,null],
  514:['Syslog','UDP','Send system log messages to a log server','rsyslog · syslog-ng',null,null],
  587:['SMTP Submission','TCP','Client sends email to outgoing mail server','Thunderbird · Outlook',1,'use this + STARTTLS, not port 25'],
  636:['LDAPS','TCP','LDAP over TLS — secure directory access','OpenLDAP · Active Directory',1,null],
  853:['DNS-over-TLS','TCP','Encrypted DNS queries','Unbound · Cloudflare 1.1.1.1',1,null],
  873:['rsync','TCP','Fast file sync and backup','rsync daemon',null,'use over SSH tunnel for encryption'],
  993:['IMAPS','TCP','IMAP over TLS — secure email access','all mail clients',1,null],
  995:['POP3S','TCP','POP3 over TLS — secure email download','all mail clients',1,null],
  1080:['SOCKS','TCP','Generic proxy protocol','proxies · Tor · SSH tunnels',null,null],
  1194:['OpenVPN','TCP+UDP','Open-source VPN protocol','OpenVPN client/server',1,null],
  1433:['MSSQL','TCP','Microsoft SQL Server database','SQL Server · SSMS',null,'never expose to internet'],
  1521:['Oracle DB','TCP','Oracle Database listener','Oracle DB · SQL*Plus',null,'never expose to internet'],
  1723:['PPTP','TCP','Point-to-Point Tunneling Protocol','Windows VPN (legacy)',0,'broken encryption — do not use'],
  1883:['MQTT','TCP','Lightweight IoT messaging protocol','Mosquitto · Home Assistant',0,'use MQTT over TLS (8883)'],
  2049:['NFS','TCP+UDP','Network file system — share drives','Linux/Unix file servers',null,null],
  2181:['ZooKeeper','TCP','Distributed coordination service','Kafka · Hadoop · HBase',null,null],
  2375:['Docker','TCP','Docker daemon API — no TLS','Docker',0,'never expose — full root access'],
  2376:['Docker TLS','TCP','Docker daemon API over TLS','Docker',1,null],
  3000:['Dev Server','TCP','Common dev server port','Node.js · Rails · Grafana',null,null],
  3306:['MySQL','TCP','MySQL and MariaDB database','MySQL · MariaDB',null,'never expose to internet'],
  3389:['RDP','TCP','Windows Remote Desktop Protocol','Windows Remote Desktop',null,'brute-forced heavily — use VPN'],
  5000:['Flask / UPnP','TCP','Flask dev server · UPnP control','Flask · Docker registry',null,null],
  5432:['PostgreSQL','TCP','PostgreSQL database','PostgreSQL',null,'never expose to internet'],
  5601:['Kibana','TCP','Elasticsearch visualization UI','Kibana · ELK stack',null,null],
  5900:['VNC','TCP','Virtual Network Computing — remote desktop','TigerVNC · RealVNC',0,'use over SSH tunnel'],
  5984:['CouchDB','TCP','CouchDB database HTTP API','CouchDB',null,null],
  6379:['Redis','TCP','Redis in-memory cache and data store','Redis',0,'no auth by default — bind to localhost'],
  6443:['Kubernetes API','TCP','Kubernetes API server','kubectl · k8s control plane',1,null],
  6881:['BitTorrent','TCP+UDP','BitTorrent peer-to-peer file transfer','qBittorrent · Transmission',null,null],
  8000:['HTTP Alt','TCP','Alternate HTTP — common dev port','Django dev · Python http.server',null,null],
  8080:['HTTP Proxy / Alt','TCP','Common HTTP alt and proxy port','Tomcat · Jenkins · proxies',null,null],
  8443:['HTTPS Alt','TCP','Alternate HTTPS — Tomcat · panels','Tomcat · cPanel',1,null],
  8883:['MQTT TLS','TCP','MQTT over TLS — secure IoT messaging','Mosquitto · AWS IoT',1,null],
  8888:['Jupyter','TCP','Jupyter Notebook web interface','Jupyter · JupyterLab',null,'bind to localhost only'],
  9000:['PHP-FPM','TCP','PHP FastCGI process manager','PHP-FPM · SonarQube',null,null],
  9090:['Prometheus','TCP','Prometheus metrics server · Cockpit UI','Prometheus · Cockpit',null,null],
  9092:['Kafka','TCP','Apache Kafka message broker','Kafka',null,null],
  9200:['Elasticsearch','TCP','Elasticsearch REST API','Elasticsearch · ELK stack',0,'no auth by default — never expose'],
  11211:['Memcached','TCP+UDP','Memcached distributed memory cache','Memcached',0,'no auth — bind to localhost only'],
  15672:['RabbitMQ UI','TCP','RabbitMQ management web interface','RabbitMQ',null,null],
  27017:['MongoDB','TCP','MongoDB database','MongoDB',0,'no auth by default — bind to localhost'],
  51820:['WireGuard','UDP','Modern fast VPN protocol','WireGuard',1,null],
};

function doPort(args) {
  if (!args.length) return 'usage: port <number>  e.g. port 443\n';
  let out = '';
  args.forEach((arg, idx) => {
    const num = parseInt(arg);
    if (isNaN(num)) { out += `✗  not a number: ${arg}\n`; return; }
    if (idx) out += '\n';
    const p = PORT_DB[num];
    out += `\n${line()}\n  PORT ${num}\n${line()}\n\n`;
    if (!p) { out += `  not in database — may be unassigned or custom\n\n`; return; }
    const [name, proto, desc, used, secure, tip] = p;
    const secStr = secure===1 ? '✓ encrypted' : secure===0 ? '✗ unencrypted' : '—';
    out += `  ${name.padEnd(24)} ${proto}\n  ${desc}\n\n`;
    if (used) out += `  used by  ${used}\n`;
    out += `  secure   ${secStr}\n`;
    if (tip)  out += `  note     ${tip}\n`;
    out += '\n';
  });
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
  ${u('port+443')}

${line()}
  commands
${line()}

  ip <CIDR>                     subnet visualizer
  spin                          random app idea combo
  spin -n <count>               multiple combos (up to 5)
  gap <n1> <dob1> <n2> <dob2>  lifespan overlap  (YYYY-MM-DD)
  burn enc <message>            encode a note
  burn dec <token>              decode a note
  port <number>                 look up a well-known port
  help                          show this

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
  if (cmd==='port') return doPort(args);
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
