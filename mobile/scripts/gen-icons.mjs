// Generates the Stallfunk logo and the app icons: a horseshoe, open to the top, with two radio
// waves rising out of the opening ("Stall" + "Funk"). Cream shoe and golden waves on the primary
// green. No font and no image library needed (pure JS, PNG through node:zlib).
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
//
// The mark is a list of shapes (arcs with round or butt caps, rectangles, disks) in a unit box centered on
// (0,0), y pointing down; later shapes paint over earlier ones. Every pixel is supersampled
// 4 x 4 for smooth edges.

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

// Horseshoe: a U of a half circle (center (0, SHOE_Y), radius SHOE_R) and two straight legs of
// length LEG up to the heels. The waves rise from a dot between the legs.
const SHOE_Y = 0.2;
const SHOE_R = 0.46;
const SHOE_W = 0.24;
const LEG = 0.34;
const WAVE_Y = SHOE_Y - 0.12; // center of the dot and the waves
const leg = (x) => ({
  kind: "rect",
  x0: x - SHOE_W / 2,
  x1: x + SHOE_W / 2,
  y0: SHOE_Y - LEG,
  y1: SHOE_Y + 0.01, // overlaps the bend, so the SVG shows no seam
  color: CREAM,
});
const hole = (x, y) => ({ kind: "disk", cx: x, cy: y, r: 0.032, color: GREEN });
const onArc = (deg) => [SHOE_R * Math.cos(rad(deg)), SHOE_Y + SHOE_R * Math.sin(rad(deg))];
const shapes = [
  { kind: "arc", cx: 0, cy: SHOE_Y, r: SHOE_R, w: SHOE_W, from: rad(0), to: rad(180), cap: "butt", color: CREAM },
  leg(-SHOE_R),
  leg(SHOE_R),
  // nail holes: two on each leg, one on each side of the bend
  ...[-SHOE_R, SHOE_R].flatMap((x) => [hole(x, SHOE_Y - LEG + 0.1), hole(x, SHOE_Y - 0.02)]),
  hole(...onArc(45)),
  hole(...onArc(135)),
  // radio waves out of the opening
  { kind: "disk", cx: 0, cy: WAVE_Y, r: 0.085, color: GOLD },
  { kind: "arc", cx: 0, cy: WAVE_Y, r: 0.26, w: 0.095, from: rad(-135), to: rad(-45), cap: "round", color: GOLD },
  { kind: "arc", cx: 0, cy: WAVE_Y, r: 0.46, w: 0.095, from: rad(-128), to: rad(-52), cap: "round", color: GOLD },
];
// Extent of the mark: from the top of the outer wave to the bottom of the shoe.
const MARK_TOP = WAVE_Y - 0.46 - 0.05;
const MARK_BOTTOM = SHOE_Y + SHOE_R + SHOE_W / 2;
const MARK_HEIGHT = MARK_BOTTOM - MARK_TOP;
const MARK_MID = (MARK_TOP + MARK_BOTTOM) / 2;

function angleIn(a, from, to) {
  while (a < from) a += 2 * Math.PI;
  while (a >= from + 2 * Math.PI) a -= 2 * Math.PI;
  return a <= to;
}

function covers(shape, x, y) {
  const dx = x - shape.cx;
  const dy = y - shape.cy;
  const d = Math.hypot(dx, dy);
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
];

for (const [file, size, markHeight] of outputs) {
  const path = join(root, file);
  mkdirSync(dirname(path), { recursive: true });
  writeFileSync(path, png(size, render(size, markHeight)));
  console.log(`${file} (${size}x${size})`);
}
writeFileSync(join(root, "public/logo.svg"), svg());
console.log("public/logo.svg");
