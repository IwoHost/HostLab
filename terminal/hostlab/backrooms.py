#!/usr/bin/env python3
"""
The Backrooms — Terminal Edition
Infinite procedurally-generated ASCII maze. WASD / arrow keys to move, Q to quit.
"""
import curses
import math
import time

CT   = 24        # tiles per chunk side
SEED = 0xBA1C5EED

ZONES = [
    'ALPHA', 'BETA', 'GAMMA', 'DELTA', 'EPSILON',
    'ZETA', 'ETA', 'THETA', 'IOTA', 'KAPPA', 'LAMBDA', 'MU',
]

# ── Seeded RNG (mulberry32 port) ──────────────────────────────────────────────
def make_rng(seed):
    s = [int(seed) & 0xFFFFFFFF or 1]
    def rng():
        s[0] = (s[0] + 0x6D2B79F5) & 0xFFFFFFFF
        t = ((s[0] ^ (s[0] >> 15)) * (1 | s[0])) & 0xFFFFFFFF
        t = (t + ((t ^ (t >> 7)) * (61 | t))) & 0xFFFFFFFF
        return ((t ^ (t >> 14)) & 0xFFFFFFFF) / 0x100000000
    return rng

def chunk_hash(cx, cy):
    h = (SEED ^ 0xDEADBEEF) & 0xFFFFFFFF
    h = ((h ^ (cx & 0xFFFFFFFF)) * 0x9E3779B9) & 0xFFFFFFFF
    h = ((h ^ (cy & 0xFFFFFFFF)) * 0x6B43A9B5) & 0xFFFFFFFF
    return (h ^ (h >> 16)) & 0xFFFFFFFF

def edge_hash(ax, ay, bx, by):
    if ax < bx or (ax == bx and ay < by):
        k1 = (ax * 100003 + ay) & 0x7FFFFFFF
        k2 = (bx * 100003 + by) & 0x7FFFFFFF
    else:
        k1 = (bx * 100003 + by) & 0x7FFFFFFF
        k2 = (ax * 100003 + ay) & 0x7FFFFFFF
    h = (SEED ^ 0xC0FFEE) & 0xFFFFFFFF
    h = ((h ^ k1) * 0x9E3779B9) & 0xFFFFFFFF
    h = ((h ^ k2) * 0x6B43A9B5) & 0xFFFFFFFF
    return (h ^ (h >> 16)) & 0xFFFFFFFF

# ── BSP chunk generator (faithful port of the browser version) ────────────────
def biased_split(rng, size, MIN):
    r = size - MIN * 2 - 1
    if r <= 0:
        return MIN + 1
    if rng() < 0.30:
        small = MIN + 1 + int(rng() * min(3, r))
        return small if rng() < 0.5 else size - small
    return MIN + 1 + int(rng() * r)

def gen_chunk(cx, cy):
    tiles = bytearray(CT * CT)  # 0 = void, 1 = floor
    rng   = make_rng(chunk_hash(cx, cy))
    rooms = []

    def mark(x, y):
        if 0 <= x < CT and 0 <= y < CT:
            tiles[y * CT + x] = 1

    def carve_line(x0, y0, x1, y1, w=2):
        dx = 1 if x1 > x0 else (-1 if x1 < x0 else 0)
        dy = 1 if y1 > y0 else (-1 if y1 < y0 else 0)
        x, y = x0, y0
        while True:
            mark(x, y)
            if w >= 2:
                if dx: mark(x, y + 1)
                if dy: mark(x + 1, y)
            if x == x1 and y == y1:
                break
            if x != x1: x += dx
            else: y += dy

    def carve_leaf(rx, ry, rw, rh):
        def rm(n): return int(rng() * (min(n, 3) + 1))
        ml, mr = rm(rw >> 2), rm(rw >> 2)
        mt, mb = rm(rh >> 2), rm(rh >> 2)  # noqa: F841
        fx, fy = rx + ml, ry + mt
        fw, fh = rw - ml - mr, rh - mt - mb
        if fw < 2 or fh < 2:
            return
        asp = max(fw, fh) / min(fw, fh)
        v = rng()
        if asp >= 2.0 or (asp >= 1.5 and v < 0.45):
            sw = 2 if (min(fw, fh) >= 3 and rng() < 0.55) else 1
            if fw >= fh:
                sy = fy + int(rng() * max(1, fh - sw))
                for y in range(sy, sy + sw):
                    for x in range(fx, fx + fw): mark(x, y)
                rooms.append((fx + (fw >> 1), sy))
            else:
                sx = fx + int(rng() * max(1, fw - sw))
                for y in range(fy, fy + fh):
                    for x in range(sx, sx + sw): mark(x, y)
                rooms.append((sx, fy + (fh >> 1)))
        elif v < 0.45 and fw >= 3 and fh >= 3:
            for y in range(fy, fy + fh):
                for x in range(fx, fx + fw): tiles[y * CT + x] = 1
            c  = int(rng() * 4)
            cw = 2 + int(rng() * max(1, (fw >> 1) - 1))
            ch = 2 + int(rng() * max(1, (fh >> 1) - 1))
            for y in range(fy, fy + fh):
                for x in range(fx, fx + fw):
                    if ((c == 0 and x < fx + cw     and y < fy + ch) or
                        (c == 1 and x >= fx+fw-cw   and y < fy + ch) or
                        (c == 2 and x < fx + cw     and y >= fy+fh-ch) or
                        (c == 3 and x >= fx+fw-cw   and y >= fy+fh-ch)):
                        tiles[y * CT + x] = 0
            ox = -(fw >> 2) if c in (1, 3) else (fw >> 2)
            oy = -(fh >> 2) if c in (2, 3) else (fh >> 2)
            rooms.append((fx + (fw >> 1) + ox, fy + (fh >> 1) + oy))
        else:
            for y in range(fy, fy + fh):
                for x in range(fx, fx + fw): tiles[y * CT + x] = 1
            rooms.append((fx + (fw >> 1), fy + (fh >> 1)))

    def split(rx, ry, rw, rh, depth):
        MIN = 4
        cv = rw >= MIN * 2 + 2
        ch = rh >= MIN * 2 + 2
        if depth >= 4 or (not cv and not ch):
            carve_leaf(rx, ry, rw, rh)
            return
        if cv and ch:
            go_v = rng() < (0.72 if rw > rh+3 else 0.28 if rw < rh-3 else 0.5)
        else:
            go_v = cv
        if go_v:
            at = biased_split(rng, rw, MIN)
            split(rx, ry, at, rh, depth + 1)
            split(rx + at, ry, rw - at, rh, depth + 1)
        else:
            at = biased_split(rng, rh, MIN)
            split(rx, ry, rw, at, depth + 1)
            split(rx, ry + at, rw, rh - at, depth + 1)

    split(0, 0, CT, CT, 0)

    # Nibble convex corners
    for y in range(1, CT - 1):
        for x in range(1, CT - 1):
            if not tiles[y * CT + x]: continue
            n4 = (int(tiles[(y-1)*CT+x]) + int(tiles[(y+1)*CT+x]) +
                  int(tiles[y*CT+x-1])   + int(tiles[y*CT+x+1]))
            if n4 >= 3 and rng() < 0.14:
                tiles[y * CT + x] = 0

    # Connect rooms via corridors
    if len(rooms) >= 2:
        idx = list(range(len(rooms)))
        for i in range(len(idx) - 1, 0, -1):
            j = int(rng() * (i + 1))
            idx[i], idx[j] = idx[j], idx[i]
        for i in range(1, len(idx)):
            a, b = rooms[idx[i-1]], rooms[idx[i]]
            w = 1 if rng() < 0.38 else 2
            ex = a[0] if rng() < 0.5 else b[0]
            carve_line(a[0], a[1], ex,   a[1], w)
            carve_line(ex,   a[1], ex,   b[1], w)
            carve_line(ex,   b[1], b[0], b[1], w)

    # Cross-chunk doorways
    def door_pos(h): return 3 + int(make_rng(h)() * (CT - 6))

    def punch_v(ex, pos):
        dx = 1 if ex == 0 else -1
        mark(ex, pos); mark(ex, pos + 1)
        tx = ex + dx
        while abs(tx - ex) < CT:
            was = tiles[pos * CT + tx] == 1
            mark(tx, pos); mark(tx, pos + 1)
            if was or abs(tx - ex) >= CT - 2: break
            tx += dx

    def punch_h(pos, ey, dy):
        mark(pos, ey); mark(pos + 1, ey)
        ty = ey + dy
        while abs(ty - ey) < CT:
            was = tiles[ty * CT + pos] == 1
            mark(pos, ty); mark(pos + 1, ty)
            if was or abs(ty - ey) >= CT - 2: break
            ty += dy

    punch_v(CT-1, door_pos(edge_hash(cx, cy, cx+1, cy)))
    punch_v(0,    door_pos(edge_hash(cx-1, cy, cx, cy)))
    punch_h(door_pos(edge_hash(cx, cy, cx, cy+1)), CT-1, -1)
    punch_h(door_pos(edge_hash(cx, cy-1, cx, cy)), 0, 1)

    # Flicker zones (dying fluorescent lights)
    fz = []
    fr = make_rng(chunk_hash(cx, cy) ^ 0xF11C1337)
    if fr() < 0.40 and rooms:
        cnt = min(2, len(rooms)) if fr() < 0.35 else 1
        order = list(range(len(rooms)))
        for i in range(len(order) - 1, 0, -1):
            j = int(fr() * (i + 1))
            order[i], order[j] = order[j], order[i]
        for i in range(cnt):
            rm = rooms[order[i]]
            hw = 3 + int(fr() * 3)
            hh = 3 + int(fr() * 3)
            fz.append({
                'x': cx * CT + rm[0] - hw,
                'y': cy * CT + rm[1] - hh,
                'w': hw * 2,
                'h': hh * 2,
                's': fr(),
            })
    return tiles, fz

# ── Chunk cache ───────────────────────────────────────────────────────────────
_chunk_cache: dict = {}

def get_chunk(cx, cy):
    k = (cx, cy)
    if k not in _chunk_cache:
        if len(_chunk_cache) >= 400:
            _chunk_cache.pop(next(iter(_chunk_cache)))
        _chunk_cache[k] = gen_chunk(cx, cy)
    return _chunk_cache[k]

def tile_at(wx, wy):
    # Python's floor division handles negative coords correctly
    cx, lx = wx // CT, wx % CT
    cy, ly = wy // CT, wy % CT
    t, _ = get_chunk(cx, cy)
    return t[ly * CT + lx]

def is_flickering_dark(wx, wy, t):
    cx, cy = wx // CT, wy // CT
    _, zones = get_chunk(cx, cy)
    for z in zones:
        if z['x'] <= wx < z['x'] + z['w'] and z['y'] <= wy < z['y'] + z['h']:
            p = (t + z['s'] * 31.7) % 14
            return (10.0 <= p < 10.6 and int(p * 24) % 2 == 1) or (10.6 <= p < 12.8)
    return False

# ── Curses game loop ──────────────────────────────────────────────────────────
def run_game(stdscr):
    curses.curs_set(0)
    stdscr.nodelay(True)
    stdscr.keypad(True)
    curses.start_color()
    curses.use_default_colors()
    curses.init_pair(1, curses.COLOR_YELLOW, -1)   # floor
    curses.init_pair(2, curses.COLOR_YELLOW, -1)   # wall face
    curses.init_pair(3, curses.COLOR_WHITE,  -1)   # player (@)
    curses.init_pair(4, curses.COLOR_GREEN,  -1)   # HUD

    # Find a floor tile near the origin to spawn on
    px, py = 0, 0
    for _ in range(200):
        if tile_at(px, py): break
        px += 1

    last_move = 0.0
    MOVE_DELAY = 0.10  # seconds between steps (10 tiles/sec when held)

    while True:
        h, w = stdscr.getmaxyx()
        vh = h - 1          # one row reserved for HUD
        hh = vh // 2        # half-height (player row on screen)
        hw = (w - 1) // 2   # half-width  (player col on screen)
        now = time.monotonic()

        # Precompute tile values for the viewport (avoids repeated cache lookups)
        vw = w - 1  # safe render width (skip last column to avoid scroll)
        view = bytearray(vh * vw)
        for sy in range(vh):
            for sx in range(vw):
                view[sy * vw + sx] = tile_at(px - hw + sx, py - hh + sy)

        stdscr.erase()
        for sy in range(vh):
            for sx in range(vw):
                wx = px - hw + sx
                wy = py - hh + sy
                fl = view[sy * vw + sx]
                try:
                    if wx == px and wy == py:
                        stdscr.addch(sy, sx, '@',
                            curses.color_pair(3) | curses.A_BOLD)
                    elif fl:
                        if is_flickering_dark(wx, wy, now):
                            stdscr.addch(sy, sx, ',', curses.A_DIM)
                        else:
                            # Near-wall tiles get a dim shadow character
                            near = (
                                (sy > 0    and not view[(sy-1)*vw+sx]) or
                                (sy < vh-1 and not view[(sy+1)*vw+sx]) or
                                (sx > 0    and not view[sy*vw+sx-1])   or
                                (sx < vw-1 and not view[sy*vw+sx+1])
                            )
                            if near:
                                stdscr.addch(sy, sx, ',',
                                    curses.color_pair(1) | curses.A_DIM)
                            else:
                                stdscr.addch(sy, sx, '.',
                                    curses.color_pair(1))
                    else:
                        # Show wall-face '#' only where void is adjacent to floor
                        near = (
                            (sy > 0    and view[(sy-1)*vw+sx]) or
                            (sy < vh-1 and view[(sy+1)*vw+sx]) or
                            (sx > 0    and view[sy*vw+sx-1])   or
                            (sx < vw-1 and view[sy*vw+sx+1])
                        )
                        if near:
                            stdscr.addch(sy, sx, '#',
                                curses.color_pair(2) | curses.A_DIM)
                except curses.error:
                    pass

        depth = int(math.hypot(px, py) / CT)
        zi    = (abs(px // CT) + abs(py // CT)) % len(ZONES)
        hud   = (f"  DEPTH {depth:>4}  ·  ZONE {ZONES[zi]:<8}  "
                 f"[WASD / arrow keys = move   Q = quit]  ")
        try:
            stdscr.addstr(h - 1, 0, hud[:w - 1],
                curses.color_pair(4) | curses.A_DIM)
        except curses.error:
            pass

        stdscr.refresh()

        key = stdscr.getch()
        if key in (ord('q'), ord('Q'), 27):
            break
        if now - last_move >= MOVE_DELAY:
            nx, ny = px, py
            if   key in (ord('w'), ord('W'), curses.KEY_UP):    ny -= 1
            elif key in (ord('s'), ord('S'), curses.KEY_DOWN):  ny += 1
            elif key in (ord('a'), ord('A'), curses.KEY_LEFT):  nx -= 1
            elif key in (ord('d'), ord('D'), curses.KEY_RIGHT): nx += 1
            if (nx != px or ny != py) and tile_at(nx, ny):
                px, py = nx, ny
                last_move = now

        time.sleep(0.04)  # ~25 fps


def main():
    curses.wrapper(run_game)


if __name__ == '__main__':
    main()
