// HostLab DJ Wall — Cloudflare Worker + Durable Object backend.
// Replaces the free json.extendsclass.com JSON-storage service, which caps
// out at 10,000 total requests — a party wall polling every 2s per device
// burns through that in well under an hour. Workers/Durable Objects give
// 100,000 requests/day free, which comfortably covers a real party.
//
// One Durable Object instance = one room. A DO processes requests to itself
// one at a time, so version-conflict checks here are actually atomic —
// unlike the old client-only optimistic-concurrency dance against a dumb
// blob store, two people voting at the same instant can no longer silently
// overwrite each other.

function corsHeaders() {
  return {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type',
  };
}

function json(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json', 'Cache-Control': 'no-store', ...corsHeaders() },
  });
}

// No 0/O/1/I/L — those get misread when someone's reading a code aloud or
// squinting at a phone screen across a grill.
const CODE_CHARS = 'ABCDEFGHJKMNPQRSTUVWXYZ23456789';
function randomRoomCode(len = 6) {
  const bytes = crypto.getRandomValues(new Uint8Array(len));
  let out = '';
  for (let i = 0; i < len; i++) out += CODE_CHARS[bytes[i] % CODE_CHARS.length];
  return out;
}

export default {
  async fetch(request, env) {
    if (request.method === 'OPTIONS') return new Response(null, { headers: corsHeaders() });

    const url = new URL(request.url);
    const parts = url.pathname.split('/').filter(Boolean);
    if (parts[0] !== 'room') return json({ error: 'not found' }, 404);

    // POST /room — create a new room. Client sends the bits it lets people
    // choose at creation (theme, room name); everything else is set here,
    // including createdAt off the server's own clock rather than trusting
    // whatever time the creating device's browser reports.
    if (parts.length === 1 && request.method === 'POST') {
      let picked = {};
      try { picked = await request.json(); } catch (e) {}
      const initialState = {
        candidates: [],
        queue: [],
        settings: { upThreshold: 5, downThreshold: 5, submitCooldown: 15 },
        theme: (picked && picked.theme) || 'original',
        roomName: (picked && picked.roomName) || '',
        version: 0,
        createdAt: Date.now(),
      };

      let code = randomRoomCode();
      for (let i = 0; i < 5; i++) {
        const stub = env.ROOMS.get(env.ROOMS.idFromName(code));
        const check = await stub.fetch('https://do/state');
        if (check.status === 404) break;
        code = randomRoomCode();
      }

      const stub = env.ROOMS.get(env.ROOMS.idFromName(code));
      const initRes = await stub.fetch('https://do/state', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(initialState),
      });
      if (!initRes.ok) return json({ error: 'failed to create room' }, 500);
      return json({ id: code });
    }

    // GET/PUT/DELETE /room/:id — forward straight to that room's DO.
    if (parts.length === 2) {
      const stub = env.ROOMS.get(env.ROOMS.idFromName(parts[1]));
      const init = { method: request.method };
      if (request.method === 'PUT') {
        init.headers = { 'Content-Type': 'application/json' };
        init.body = await request.text();
      }
      const doRes = await stub.fetch('https://do/state', init);
      const body = await doRes.text();
      return new Response(body, {
        status: doRes.status,
        headers: { 'Content-Type': 'application/json', 'Cache-Control': 'no-store', ...corsHeaders() },
      });
    }

    return json({ error: 'not found' }, 404);
  },
};

export class RoomDO {
  constructor(state, env) {
    this.storage = state.storage;
  }

  async fetch(request) {
    if (request.method === 'GET') {
      const state = await this.storage.get('state');
      if (state === undefined) return new Response(JSON.stringify({ error: 'not found' }), { status: 404 });
      return new Response(JSON.stringify(state), { status: 200 });
    }

    if (request.method === 'PUT') {
      let incoming;
      try { incoming = await request.json(); } catch (e) { return new Response(JSON.stringify({ error: 'bad json' }), { status: 400 }); }
      const current = await this.storage.get('state');
      // No existing state yet means this write is the room being created —
      // always allowed. Otherwise the incoming version must be exactly
      // current+1, the same check-and-set optimistic concurrency was
      // approximating client-side, except this time nothing can race it.
      if (current !== undefined) {
        const currentVersion = typeof current.version === 'number' ? current.version : 0;
        const incomingVersion = typeof incoming.version === 'number' ? incoming.version : 0;
        if (incomingVersion !== currentVersion + 1) {
          return new Response(JSON.stringify({ error: 'version conflict', currentVersion }), { status: 409 });
        }
      }
      await this.storage.put('state', incoming);
      return new Response(JSON.stringify({ ok: true }), { status: 200 });
    }

    if (request.method === 'DELETE') {
      await this.storage.delete('state');
      return new Response(JSON.stringify({ ok: true }), { status: 200 });
    }

    return new Response(JSON.stringify({ error: 'method not allowed' }), { status: 405 });
  }
}
