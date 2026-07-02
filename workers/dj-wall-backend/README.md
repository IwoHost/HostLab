# hostlab-dj-wall

Cloudflare Worker + Durable Object backing the room storage for
`websites/dj-queue.html` ("THE WALL"). One Durable Object instance per room,
so concurrent writes to the same room are processed one at a time instead of
racing against each other.

Replaces the free `json.extendsclass.com` JSON-storage service, which caps
out at 10,000 total requests — a party wall polling every 2s per connected
device burns through that in well under an hour. This gives 100,000
requests/day free.

## API

All responses are JSON, CORS-open (`Access-Control-Allow-Origin: *`).

- `POST /room` — create a room. Body may include `{ theme, roomName }`;
  everything else (empty candidates/queue, default settings, `version: 0`,
  `createdAt`) is set server-side. Returns `{ id }`, a 6-character room code.
- `GET /room/:id` — fetch the room's current state. `404` if it doesn't
  exist or was deleted.
- `PUT /room/:id` — replace the room's state. The body's `version` must be
  exactly `currentVersion + 1` or the write is rejected with `409` — this is
  what makes concurrent edits (e.g. two people voting at once) safe instead
  of silently overwriting each other.
- `DELETE /room/:id` — delete the room's state. Subsequent `GET`s 404.

## Deploy

```
npm install
npx wrangler login   # if not already logged in
npx wrangler deploy
```

Update the `API` constant in `websites/dj-queue.html` if your deployed URL
differs from `https://hostlab-dj-wall.iwohost.workers.dev/room`.
