// Generates the PWA icons: a monogram "S" in the primary color (#2d5a3d) on the warm background
// (#f6f4ee), no font and no image library needed (pure JS, PNG through node:zlib).
//
//   node scripts/gen-icons.mjs
//
// Output (committed, regenerate only when the design changes):
//   public/icon-192.png, public/icon-512.png       purpose "any"
//   public/icon-maskable-512.png                    purpose "maskable" (S inside the 80 % safe zone)
//   public/apple-touch-icon.png                     180 x 180, opaque (iOS rounds it itself)
//   public/favicon.png                              48 x 48
//   assets/icon-1024.png                            source size, for stores or later reuse
//
// The S is two circular arcs drawn as a thick stroke with round caps. Every pixel is
// supersampled 4 x 4 for smooth edges.

import { mkdirSync, writeFileSync } from "node:fs";
import { deflateSync } from "node:zlib";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");

const BG = [0xf6, 0xf4, 0xee];
const FG = [0x2d, 0x5a, 0x3d];

const rad = (deg) => (deg * Math.PI) / 180;

// The S in a unit box centered on (0,0): two circles of radius R, one above the other.
// Top arc: from -30 deg, counterclockwise (decreasing angle) over the top and the left to the joint.
// Bottom arc: from the joint, clockwise (increasing angle) over the right and the bottom to 150 deg.
const R = 0.5;
const STROKE = 0.3; // stroke width
const arcs = [
  { cx: 0, cy: -R, from: rad(-270), to: rad(-30) }, // top, angles in [-270, -30]
  { cx: 0, cy: R, from: rad(-90), to: rad(150) }, // bottom, angles in [-90, 150]
];
// Height of the S including the stroke: 4R + STROKE = 2.3 units.
const S_HEIGHT = 4 * R + STROKE;

function distanceToArc(px, py, arc) {
  const dx = px - arc.cx;
  const dy = py - arc.cy;
  let a = Math.atan2(dy, dx);
  // bring the angle into [from, from + 2pi)
  while (a < arc.from) a += 2 * Math.PI;
  while (a >= arc.from + 2 * Math.PI) a -= 2 * Math.PI;
  if (a <= arc.to) return Math.abs(Math.hypot(dx, dy) - R);
  const ends = [arc.from, arc.to].map((t) => [arc.cx + R * Math.cos(t), arc.cy + R * Math.sin(t)]);
  return Math.min(...ends.map(([ex, ey]) => Math.hypot(px - ex, py - ey)));
}

function inside(ux, uy) {
  const d = Math.min(...arcs.map((arc) => distanceToArc(ux, uy, arc)));
  return d <= STROKE / 2;
}

/** RGBA pixels, `size` x `size`; the S is `sHeight` (fraction of the size) tall. */
function render(size, sHeight) {
  const scale = (sHeight * size) / S_HEIGHT; // pixels per unit
  const cx = size / 2;
  const cy = size / 2;
  const buf = Buffer.alloc(size * size * 4);
  const SS = 4;
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      let hits = 0;
      for (let sy = 0; sy < SS; sy++) {
        for (let sx = 0; sx < SS; sx++) {
          const ux = (x + (sx + 0.5) / SS - cx) / scale;
          const uy = (y + (sy + 0.5) / SS - cy) / scale;
          if (inside(ux, uy)) hits++;
        }
      }
      const t = hits / (SS * SS);
      const o = (y * size + x) * 4;
      buf[o] = Math.round(BG[0] + (FG[0] - BG[0]) * t);
      buf[o + 1] = Math.round(BG[1] + (FG[1] - BG[1]) * t);
      buf[o + 2] = Math.round(BG[2] + (FG[2] - BG[2]) * t);
      buf[o + 3] = 255;
    }
  }
  return buf;
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

const ANY = 0.6; // S height as a fraction of the icon
const MASKABLE = 0.44; // stays well inside the 80 % safe zone circle (diameter 0.8)
const outputs = [
  ["public/icon-192.png", 192, ANY],
  ["public/icon-512.png", 512, ANY],
  ["public/icon-maskable-512.png", 512, MASKABLE],
  ["public/apple-touch-icon.png", 180, ANY],
  ["public/favicon.png", 48, 0.7],
  ["assets/icon-1024.png", 1024, ANY],
];

for (const [file, size, sHeight] of outputs) {
  const path = join(root, file);
  mkdirSync(dirname(path), { recursive: true });
  writeFileSync(path, png(size, render(size, sHeight)));
  console.log(`${file} (${size}x${size})`);
}
