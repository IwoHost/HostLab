/* ── CARD MOTION ─────────────────────────────────────────
   Shared by the hub and the section pages. Pair with
   cardmotion.css, which owns the transform these values feed.

   Two things happen as the pointer moves. Cards in the row the
   pointer is level with swell by their horizontal distance from
   it, like a dock — the rows above and below stay put, and the
   curve is bent so the far end of the reach adds almost nothing,
   meaning a card that is not genuinely near does not move at all.
   The card under the pointer additionally leans toward it.

   Each grid is its own group, so one grid on a page never stirs
   when the pointer is inside another.

   Card centres are cached in document coordinates, so scrolling
   needs no re-measure. Call CardMotion.measure() after any render
   that moves cards. One listener, rAF-throttled to a single pass
   per frame.
   ──────────────────────────────────────────────────────── */
window.CardMotion = (() => {
  const TILT    = 7;     // degrees at the edge of the card under the pointer
  const SWELL   = 0.06;  // scale added when the pointer is on a card's centre
  const REACH   = 720;   // px — horizontal reach along the row, about one card
  const FALLOFF = 1.2;   // >1 bends the curve down, so the far end of the reach
                         // contributes almost nothing
  const CARD_SEL  = '.card, .section-card';
  const GROUP_SEL = '.grid, .sections';

  let groups = [];

  function measure() {
    groups = [...document.querySelectorAll(GROUP_SEL)].map(root => {
      const cards = [...root.querySelectorAll(CARD_SEL)];
      if (!cards.length) return null;
      const b = root.getBoundingClientRect();
      return {
        root,
        top:    b.top    + scrollY,
        bottom: b.bottom + scrollY,
        left:   b.left   + scrollX,
        right:  b.right  + scrollX,
        cards: cards.map(el => {
          const r = el.getBoundingClientRect();
          return {
            el,
            cx: r.left + scrollX + r.width / 2,
            cy: r.top  + scrollY + r.height / 2,
            hh: r.height / 2,
          };
        }),
      };
    }).filter(Boolean);
  }

  const relaxGroup = g => g.cards.forEach(c => c.el.style.removeProperty('--sc'));

  function init() {
    if (matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    if (!matchMedia('(hover: hover)').matches) return;

    let queued = false, ev = null, active = null, swellOnly = false, lit = null;

    const clearTilt = el => {
      if (!el) return;
      el.classList.remove('tilt');
      el.style.removeProperty('--rx');
      el.style.removeProperty('--ry');
    };

    const relax = () => {
      clearTilt(active); active = null;
      groups.forEach(relaxGroup);
      lit = null;
    };

    const paint = () => {
      queued = false;
      if (!ev) return;
      const mx = ev.clientX + scrollX, my = ev.clientY + scrollY;

      /* only the grid the pointer is actually inside reacts */
      const group = groups.find(g => mx >= g.left && mx <= g.right && my >= g.top && my <= g.bottom);
      if (group !== lit) { if (lit) relaxGroup(lit); lit = group || null; }
      if (!group) return;

      for (const c of group.cards) {
        /* only the row the pointer is level with: the swell runs along the
           row like a dock, it does not spread into the rows above or below */
        const inRow = Math.abs(my - c.cy) <= c.hh + 4;
        const d = inRow ? Math.abs(mx - c.cx) : Infinity;   // horizontal distance only
        const f = d >= REACH ? 0 : Math.pow(1 - d / REACH, FALLOFF);
        const add = SWELL * f;
        // below a twentieth of a percent nothing is visible, so leave the card
        // entirely alone rather than animating it imperceptibly
        if (add < 0.0005) c.el.style.removeProperty('--sc');
        else c.el.style.setProperty('--sc', (1 + add).toFixed(4));
      }

      /* the scroll pass only resizes: the element the pointer was last
         over is stale once the page has moved under it */
      if (swellOnly) { swellOnly = false; return; }

      /* lean: only the card actually under the pointer */
      const el = ev.target.closest(CARD_SEL);
      if (el !== active) { clearTilt(active); active = el; }
      if (!el) return;

      const r = el.getBoundingClientRect();
      const dx = ((ev.clientX - r.left) / r.width)  * 2 - 1;   // -1..1 from centre
      const dy = ((ev.clientY - r.top)  / r.height) * 2 - 1;

      el.classList.add('tilt');
      // negate both: the side under the cursor rotates toward the viewer
      el.style.setProperty('--ry', (-dx * TILT).toFixed(2) + 'deg');
      el.style.setProperty('--rx', ( dy * TILT).toFixed(2) + 'deg');
    };

    document.addEventListener('pointermove', e => {
      ev = e; swellOnly = false;
      if (!queued) { queued = true; requestAnimationFrame(paint); }
    }, { passive: true });

    // a card scrolled out from under a stationary pointer must not stay tilted,
    // and the sizes should follow the page as it moves
    document.addEventListener('scroll', () => {
      clearTilt(active); active = null;
      if (ev && !queued) { swellOnly = true; queued = true; requestAnimationFrame(paint); }
    }, { passive: true });
    document.addEventListener('pointerleave', relax);
    addEventListener('resize', measure);
    addEventListener('load', measure);

    measure();
  }

  if (document.readyState === 'loading') addEventListener('DOMContentLoaded', init);
  else init();

  return { measure, init };
})();
