// HostLab Spotify Search — Cloudflare Worker
// Proxies Spotify's Client Credentials flow so the app-only API secret
// never has to live in client-side code. Search-only: no user login,
// no playback, just "look up a real track to attach to a submission."

let cachedToken = null; // { token, expiresAt } — reused across requests on this isolate

async function getAccessToken(env) {
  if (cachedToken && cachedToken.expiresAt > Date.now()) return cachedToken.token;
  const creds = btoa(`${env.SPOTIFY_CLIENT_ID}:${env.SPOTIFY_CLIENT_SECRET}`);
  const res = await fetch('https://accounts.spotify.com/api/token', {
    method: 'POST',
    headers: {
      'Authorization': `Basic ${creds}`,
      'Content-Type': 'application/x-www-form-urlencoded',
    },
    body: 'grant_type=client_credentials',
  });
  if (!res.ok) throw new Error('spotify auth failed: ' + res.status);
  const data = await res.json();
  cachedToken = { token: data.access_token, expiresAt: Date.now() + (data.expires_in - 60) * 1000 };
  return cachedToken.token;
}

function corsHeaders() {
  return {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type',
  };
}

function json(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json', ...corsHeaders() },
  });
}

export default {
  async fetch(request, env) {
    if (request.method === 'OPTIONS') return new Response(null, { headers: corsHeaders() });

    const url = new URL(request.url);
    const q = (url.searchParams.get('q') || '').trim().slice(0, 100);
    if (!q) return json({ tracks: [] });

    if (!env.SPOTIFY_CLIENT_ID || !env.SPOTIFY_CLIENT_SECRET) {
      return json({ tracks: [], error: 'Spotify credentials not configured on this worker' }, 500);
    }

    try {
      const token = await getAccessToken(env);
      const searchRes = await fetch(
        `https://api.spotify.com/v1/search?type=track&limit=6&q=${encodeURIComponent(q)}`,
        { headers: { Authorization: `Bearer ${token}` } }
      );
      if (!searchRes.ok) throw new Error('spotify search failed: ' + searchRes.status);
      const data = await searchRes.json();
      const tracks = (data.tracks && data.tracks.items || []).map(t => ({
        id: t.id,
        title: t.name,
        artist: (t.artists || []).map(a => a.name).join(', '),
        album: (t.album && t.album.name) || '',
        image: (t.album && t.album.images && (t.album.images[2] || t.album.images[0]) || {}).url || '',
        url: (t.external_urls && t.external_urls.spotify) || '',
      }));
      return json({ tracks });
    } catch (e) {
      return json({ tracks: [], error: e.message }, 502);
    }
  },
};
