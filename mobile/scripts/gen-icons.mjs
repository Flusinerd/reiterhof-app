// Generates the Stallfunk logo and the app icons: a classic horseshoe, open to the top, with a
// radio sign in its bow ("Stall" + "Funk"). Cream shoe and golden sign on the primary green. No font and no image library needed (pure JS, PNG through node:zlib).
//
//   node scripts/gen-icons.mjs
//
// Output (committed, regenerate only when the design changes):
//   public/icon-192.png, public/icon-512.png       purpose "any"
//   public/icon-maskable-512.png                    purpose "maskable" (mark inside the 80 % safe zone)
//   public/apple-touch-icon.png                     180 x 180, opaque (iOS rounds it itself)
//   public/favicon.png                              48 x 48
//   public/logo.svg                                 vector logo (e-mails link the PNG, docs the SVG)
//   assets/icon-1024.png                            native app icon (expo.icon), store size
//   assets/logo-256.png                             logo inside the app (sign-in screen)
//   ../backend/internal/auth/mail-logo.png          88 x 88, inline logo of the mails (44 px, 2x)
//
// The mark is a list of shapes (an even-odd SVG path, arcs with round or butt caps, rectangles,
// disks) centered on (0,0), y pointing down; later shapes paint over earlier ones. Every pixel
// is supersampled 4 x 4 for smooth edges.

import { mkdirSync, writeFileSync } from "node:fs";
import { deflateSync } from "node:zlib";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");

const hex = (h) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16));
const GREEN = "#2d5a3d"; // colors.primary
const CREAM = "#f6f4ee"; // colors.background
const GOLD = "#fbbf24"; // gait-trot, the warm highlight on dark green

const rad = (deg) => (deg * Math.PI) / 180;

// Horseshoe: a classic shoe (outline and nail holes as one even-odd SVG path, open to the top,
// viewBox 116.9 x 122.88) in cream. The golden radio sign (dot and two waves) sits in its bow.
// All coordinates of the mark are in the units of that viewBox, centered on (0,0).
const SHOE_W = 116.9;
const SHOE_H = 122.88;
const SHOE_PATH =
  "M25,80.18A33.31,33.31,0,0,0,36.67,94.3a38.86,38.86,0,0,0,43.44.09A33.37,33.37,0,0,0,92,80.18c5.83-13.42,4.56-36.7-1.84-50-1.67-3.46-3.81-6.34-7.25-12.4C80.1,13,78.79,8.63,83.73,4.59A22,22,0,0,1,97.47,0c4.67.14,7.54,3.4,9.14,7.61,1.07,2.78,2.25,8.35,1.5,11.35-.37,1.53-1.16,2-1.65,3-.72,1.46.09,2.95,1.18,5.07,16.4,32,11,69.78-15.88,86.81-17.14,10.86-41.64,11.89-60,3.55C-.75,102.61-8.4,61.46,9.27,27c1.08-2.12,1.89-3.61,1.17-5.07-.49-1-1.27-1.42-1.65-3-.74-3,.44-8.57,1.5-11.35C11.89,3.41,14.76.15,19.43,0A22,22,0,0,1,33.17,4.59c4.94,4,3.63,8.36.87,13.23-3.43,6.06-5.57,8.94-7.25,12.4-6.4,13.26-7.66,36.54-1.84,50ZM21.87,12.29a3.3,3.3,0,1,1-3.3,3.3,3.29,3.29,0,0,1,3.3-3.3Zm36.58,94.77a3.66,3.66,0,1,1-3.65,3.66,3.65,3.65,0,0,1,3.65-3.66Zm41-19.31a3.66,3.66,0,1,1-3.65,3.65,3.65,3.65,0,0,1,3.65-3.65Zm-81.9,0a3.66,3.66,0,1,1-3.65,3.65,3.66,3.66,0,0,1,3.65-3.65Zm88.76-26.1a3.65,3.65,0,1,1-3.66,3.65,3.65,3.65,0,0,1,3.66-3.65Zm-95.61,0A3.65,3.65,0,1,1,7,65.3a3.65,3.65,0,0,1,3.66-3.65Zm91.87-26.11a3.66,3.66,0,1,1-3.66,3.66,3.66,3.66,0,0,1,3.66-3.66Zm-88.13,0a3.66,3.66,0,1,1-3.66,3.66,3.66,3.66,0,0,1,3.66-3.66ZM95,12.29a3.3,3.3,0,1,1-3.3,3.3,3.29,3.29,0,0,1,3.3-3.3Z";
const WAVE_X = SHOE_W / 2; // dot and wave center (viewBox units)
const WAVE_Y = 80;
const shapes = [
  { kind: "path", rings: flattenPath(SHOE_PATH, -SHOE_W / 2, -SHOE_H / 2), color: CREAM },
  { kind: "disk", cx: 0, cy: WAVE_Y - SHOE_H / 2, r: 5.6, color: GOLD },
  ...[18, 31].map((r) => ({
    kind: "arc",
    cx: WAVE_X - SHOE_W / 2,
    cy: WAVE_Y - SHOE_H / 2,
    r,
    w: 6.6,
    from: rad(-136),
    to: rad(-44),
    cap: "round",
    color: GOLD,
  })),
];
const MARK_HEIGHT = SHOE_H;
const MARK_MID = 0;

// --- SVG path flattening (M L H V C S Q T A Z, absolute and relative) ---------------------------

/** Flattens an SVG path into closed polygons (rings), shifted by (dx, dy). */
function flattenPath(d, dx, dy) {
  const tokens = d.match(/[MmLlHhVvCcSsQqTtAaZz]|-?(?:\d+\.?\d*|\.\d+)(?:e[-+]?\d+)?/g);
  const rings = [];
  let ring = [];
  let i = 0;
  let cmd = "";
  let [x, y, sx, sy] = [0, 0, 0, 0];
  let [lcx, lcy] = [0, 0]; // last control point (for S/T)
  const num = () => parseFloat(tokens[i++]);
  const push = (px, py) => ring.push([px + dx, py + dy]);
  const cubic = (x1, y1, x2, y2, ex, ey) => {
    for (let k = 1; k <= 16; k++) {
      const t = k / 16;
      const u = 1 - t;
      push(
        u * u * u * x + 3 * u * u * t * x1 + 3 * u * t * t * x2 + t * t * t * ex,
        u * u * u * y + 3 * u * u * t * y1 + 3 * u * t * t * y2 + t * t * t * ey,
      );
    }
    [lcx, lcy, x, y] = [x2, y2, ex, ey];
  };
  // Endpoint to center parameterization (SVG 1.1, appendix F.6.5).
  const arc = (rx, ry, phiDeg, large, sweep, ex, ey) => {
    const phi = rad(phiDeg);
    const [cos, sin] = [Math.cos(phi), Math.sin(phi)];
    const x1p = (cos * (x - ex)) / 2 + (sin * (y - ey)) / 2;
    const y1p = (-sin * (x - ex)) / 2 + (cos * (y - ey)) / 2;
    rx = Math.abs(rx);
    ry = Math.abs(ry);
    const lambda = (x1p * x1p) / (rx * rx) + (y1p * y1p) / (ry * ry);
    if (lambda > 1) [rx, ry] = [rx * Math.sqrt(lambda), ry * Math.sqrt(lambda)];
    const num2 = rx * rx * ry * ry - rx * rx * y1p * y1p - ry * ry * x1p * x1p;
    const den = rx * rx * y1p * y1p + ry * ry * x1p * x1p;
    const coef = (large === sweep ? -1 : 1) * Math.sqrt(Math.max(0, num2 / den));
    const cxp = (coef * rx * y1p) / ry;
    const cyp = (-coef * ry * x1p) / rx;
    const cx = cos * cxp - sin * cyp + (x + ex) / 2;
    const cy = sin * cxp + cos * cyp + (y + ey) / 2;
    const ang = (ux, uy, vx, vy) => Math.atan2(ux * vy - uy * vx, ux * vx + uy * vy);
    const t1 = ang(1, 0, (x1p - cxp) / rx, (y1p - cyp) / ry);
    let dt = ang((x1p - cxp) / rx, (y1p - cyp) / ry, (-x1p - cxp) / rx, (-y1p - cyp) / ry);
    if (!sweep && dt > 0) dt -= 2 * Math.PI;
    if (sweep && dt < 0) dt += 2 * Math.PI;
    const n = Math.max(4, Math.ceil(Math.abs(dt) / rad(6)));
    for (let k = 1; k <= n; k++) {
      const t = t1 + (dt * k) / n;
      push(cx + rx * Math.cos(t) * cos - ry * Math.sin(t) * sin, cy + rx * Math.cos(t) * sin + ry * Math.sin(t) * cos);
    }
    [x, y, lcx, lcy] = [ex, ey, ex, ey];
  };
  while (i < tokens.length) {
    if (/[a-zA-Z]/.test(tokens[i])) cmd = tokens[i++];
    const rel = cmd === cmd.toLowerCase();
    const ox = rel ? x : 0;
    const oy = rel ? y : 0;
    switch (cmd.toUpperCase()) {
      case "M":
        if (ring.length) rings.push(ring);
        ring = [];
        x = ox + num();
        y = oy + num();
        [sx, sy, lcx, lcy] = [x, y, x, y];
        push(x, y);
        cmd = rel ? "l" : "L"; // further pairs are line-tos
        break;
      case "L":
        x = ox + num();
        y = oy + num();
        [lcx, lcy] = [x, y];
        push(x, y);
        break;
      case "H":
        x = ox + num();
        [lcx, lcy] = [x, y];
        push(x, y);
        break;
      case "V":
        y = oy + num();
        [lcx, lcy] = [x, y];
        push(x, y);
        break;
      case "C": {
        const [x1, y1, x2, y2, ex, ey] = [ox + num(), oy + num(), ox + num(), oy + num(), ox + num(), oy + num()];
        cubic(x1, y1, x2, y2, ex, ey);
        break;
      }
      case "S": {
        const [x2, y2, ex, ey] = [ox + num(), oy + num(), ox + num(), oy + num()];
        cubic(2 * x - lcx, 2 * y - lcy, x2, y2, ex, ey);
        break;
      }
      case "Q": {
        const [qx, qy, ex, ey] = [ox + num(), oy + num(), ox + num(), oy + num()];
        cubic(x + (2 / 3) * (qx - x), y + (2 / 3) * (qy - y), ex + (2 / 3) * (qx - ex), ey + (2 / 3) * (qy - ey), ex, ey);
        [lcx, lcy] = [qx, qy];
        break;
      }
      case "T": {
        const [qx, qy] = [2 * x - lcx, 2 * y - lcy];
        const [ex, ey] = [ox + num(), oy + num()];
        cubic(x + (2 / 3) * (qx - x), y + (2 / 3) * (qy - y), ex + (2 / 3) * (qx - ex), ey + (2 / 3) * (qy - ey), ex, ey);
        [lcx, lcy] = [qx, qy];
        break;
      }
      case "A": {
        const [rx, ry, phi, large, sweep] = [num(), num(), num(), num(), num()];
        arc(rx, ry, phi, large, sweep, ox + num(), oy + num());
        break;
      }
      case "Z":
        [x, y] = [sx, sy];
        if (ring.length) rings.push(ring);
        ring = [];
        break;
      default:
        throw new Error(`unsupported path command ${cmd}`);
    }
  }
  if (ring.length) rings.push(ring);
  return rings;
}

/** Even-odd test against all rings; the crossings of one row are cached (rows repeat per pixel). */
function insidePath(shape, x, y) {
  if (shape.rowY !== y) {
    const xs = [];
    for (const pts of shape.rings) {
      for (let i = 0, j = pts.length - 1; i < pts.length; j = i++) {
        const [xi, yi] = pts[i];
        const [xj, yj] = pts[j];
        if (yi > y !== yj > y) xs.push(((xj - xi) * (y - yi)) / (yj - yi) + xi);
      }
    }
    shape.rowY = y;
    shape.rowXs = xs;
  }
  let inside = false;
  for (const cx of shape.rowXs) if (x < cx) inside = !inside;
  return inside;
}

function angleIn(a, from, to) {
  while (a < from) a += 2 * Math.PI;
  while (a >= from + 2 * Math.PI) a -= 2 * Math.PI;
  return a <= to;
}

function covers(shape, x, y) {
  const dx = x - shape.cx;
  const dy = y - shape.cy;
  const d = Math.hypot(dx, dy);
  if (shape.kind === "path") return insidePath(shape, x, y);
  if (shape.kind === "rect") return x >= shape.x0 && x <= shape.x1 && y >= shape.y0 && y <= shape.y1;
  if (shape.kind === "disk") return d <= shape.r;
  if (angleIn(Math.atan2(dy, dx), shape.from, shape.to)) return Math.abs(d - shape.r) <= shape.w / 2;
  if (shape.cap !== "round") return false;
  return [shape.from, shape.to].some(
    (t) => Math.hypot(x - (shape.cx + shape.r * Math.cos(t)), y - (shape.cy + shape.r * Math.sin(t))) <= shape.w / 2,
  );
}

function colorAt(x, y) {
  let c = GREEN;
  for (const s of shapes) if (covers(s, x, y)) c = s.color;
  return c;
}

/** RGBA pixels, `size` x `size`; the mark is `markHeight` (fraction of the size) tall. */
function render(size, markHeight) {
  const scale = (markHeight * size) / MARK_HEIGHT; // pixels per unit
  const cache = new Map();
  const rgb = (c) => cache.get(c) ?? (cache.set(c, hex(c)), cache.get(c));
  const buf = Buffer.alloc(size * size * 4);
  const SS = 4;
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const sum = [0, 0, 0];
      for (let sy = 0; sy < SS; sy++) {
        for (let sx = 0; sx < SS; sx++) {
          const ux = (x + (sx + 0.5) / SS - size / 2) / scale;
          const uy = (y + (sy + 0.5) / SS - size / 2) / scale + MARK_MID;
          const [r, g, b] = rgb(colorAt(ux, uy));
          sum[0] += r;
          sum[1] += g;
          sum[2] += b;
        }
      }
      const o = (y * size + x) * 4;
      buf[o] = Math.round(sum[0] / (SS * SS));
      buf[o + 1] = Math.round(sum[1] / (SS * SS));
      buf[o + 2] = Math.round(sum[2] / (SS * SS));
      buf[o + 3] = 255;
    }
  }
  return buf;
}

/** The same mark as SVG: a rounded green square, 512 units, mark height ANY. */
function svg() {
  const size = 512;
  const scale = (ANY * size) / MARK_HEIGHT;
  const X = (u) => +(size / 2 + u * scale).toFixed(2);
  const Y = (u) => +(size / 2 + (u - MARK_MID) * scale).toFixed(2);
  const S = (u) => +(u * scale).toFixed(2);
  const parts = [`<rect width="${size}" height="${size}" rx="112" fill="${GREEN}"/>`];
  for (const s of shapes) {
    if (s.kind === "path") {
      // the original path, moved and scaled like the unit coordinates
      parts.push(
        `<path transform="translate(${X(-SHOE_W / 2)} ${Y(-SHOE_H / 2)}) scale(${+scale.toFixed(4)})" ` +
          `fill="${s.color}" fill-rule="evenodd" d="${SHOE_PATH}"/>`,
      );
      continue;
    }
    if (s.kind === "rect") {
      parts.push(
        `<rect x="${X(s.x0)}" y="${Y(s.y0)}" width="${S(s.x1 - s.x0)}" height="${S(s.y1 - s.y0)}" fill="${s.color}"/>`,
      );
      continue;
    }
    if (s.kind === "disk") {
      parts.push(`<circle cx="${X(s.cx)}" cy="${Y(s.cy)}" r="${S(s.r)}" fill="${s.color}"/>`);
      continue;
    }
    const p = (t) => `${X(s.cx + s.r * Math.cos(t))} ${Y(s.cy + s.r * Math.sin(t))}`;
    const large = s.to - s.from > Math.PI ? 1 : 0;
    parts.push(
      `<path d="M ${p(s.from)} A ${S(s.r)} ${S(s.r)} 0 ${large} 1 ${p(s.to)}" fill="none" stroke="${s.color}" ` +
        `stroke-width="${S(s.w)}" stroke-linecap="${s.cap}"/>`,
    );
  }
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${size} ${size}" role="img" aria-label="Stallfunk">\n  ${parts.join("\n  ")}\n</svg>\n`;
}

// --- PNG encoder --------------------------------------------------------------------------------

const crcTable = new Uint32Array(256).map((_, n) => {
  let c = n;
  for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
  return c >>> 0;
});
function crc32(buf) {
  let c = 0xffffffff;
  for (const b of buf) c = crcTable[(c ^ b) & 0xff] ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
}
function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const body = Buffer.concat([Buffer.from(type, "ascii"), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([len, body, crc]);
}
function png(size, rgba) {
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(size, 0);
  ihdr.writeUInt32BE(size, 4);
  ihdr[8] = 8; // bit depth
  ihdr[9] = 6; // RGBA
  const rows = Buffer.alloc(size * (size * 4 + 1));
  for (let y = 0; y < size; y++) {
    rows[y * (size * 4 + 1)] = 0; // filter: none
    rgba.copy(rows, y * (size * 4 + 1) + 1, y * size * 4, (y + 1) * size * 4);
  }
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", ihdr),
    chunk("IDAT", deflateSync(rows, { level: 9 })),
    chunk("IEND", Buffer.alloc(0)),
  ]);
}

// --- outputs ------------------------------------------------------------------------------------

const ANY = 0.62; // mark height as a fraction of the icon
const MASKABLE = 0.46; // stays inside the 80 % safe zone circle (diameter 0.8)
const outputs = [
  ["public/icon-192.png", 192, ANY],
  ["public/icon-512.png", 512, ANY],
  ["public/icon-maskable-512.png", 512, MASKABLE],
  ["public/apple-touch-icon.png", 180, ANY],
  ["public/favicon.png", 48, 0.8],
  ["assets/icon-1024.png", 1024, ANY],
  ["assets/logo-256.png", 256, ANY],
  ["../backend/internal/auth/mail-logo.png", 88, ANY],
];

for (const [file, size, markHeight] of outputs) {
  const path = join(root, file);
  mkdirSync(dirname(path), { recursive: true });
  writeFileSync(path, png(size, render(size, markHeight)));
  console.log(`${file} (${size}x${size})`);
}
writeFileSync(join(root, "public/logo.svg"), svg());
console.log("public/logo.svg");
