// Copies the MapLibre web worker (and the chunk it imports) into public/maplibre/, so that
// `expo export --platform web` serves it at /maplibre/. MapLibre derives the worker URL from
// `import.meta.url`, which does not point at a real file inside a Metro bundle, so
// components/tracking-map.web.tsx sets it with `setWorkerUrl("/maplibre/maplibre-gl-worker.mjs")`.
// Runs as `postinstall`, so the copy always matches the installed maplibre-gl version.
// public/maplibre/ is git-ignored.

import { copyFileSync, existsSync, mkdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const from = join(root, "node_modules", "maplibre-gl", "dist");
const to = join(root, "public", "maplibre");

if (!existsSync(from)) {
  console.log("maplibre-gl not installed, skipping the worker copy");
  process.exit(0);
}
mkdirSync(to, { recursive: true });
for (const file of ["maplibre-gl-worker.mjs", "maplibre-gl-shared.mjs"]) copyFileSync(join(from, file), join(to, file));
console.log("copied the maplibre-gl worker to public/maplibre/");
