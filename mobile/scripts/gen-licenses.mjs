#!/usr/bin/env node
// Generates lib/licenses.generated.ts: the open-source notices the app shows under
// "Rechtliches > Lizenzen" (MIT, BSD, Apache and the like require the copyright notice and
// the license text to travel with the software). It runs on every `npm ci` (postinstall), so
// the notices always match the installed packages and a build without them cannot exist.
//
//   node scripts/gen-licenses.mjs           write lib/licenses.generated.ts
//   node scripts/gen-licenses.mjs --check   exit 1 if the file is out of date
//
// Rules (a violation fails the install and therefore CI and the deploy):
//   - every production package of package-lock.json needs a license that is in ALLOWED;
//     a dual license ("MIT OR Apache-2.0") counts when one of its parts is allowed;
//   - the license text comes from the package's LICENSE file; a package without one gets the
//     standard text of its license id (STANDARD_TEXTS) with its author line, or the install
//     fails when that id has no standard text.
// Identical texts are stored once. Optional platform packages that are not installed are
// skipped; they are not part of the app either.

import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
export const OUTPUT = join(root, "lib", "licenses.generated.ts");

/** Permissive licenses the app may ship. Copyleft licenses that would reach the app are not on it. */
const ALLOWED = new Set([
  "MIT",
  "MIT-0",
  "ISC",
  "BSD-2-Clause",
  "BSD-3-Clause",
  "0BSD",
  "Apache-2.0",
  "Unlicense",
  "CC0-1.0",
  "CC-BY-4.0",
  "MPL-2.0",
  "Python-2.0",
  "BlueOak-1.0.0",
  "OFL-1.1",
  "Zlib",
  "WTFPL",
  "BSL-1.0",
]);

const STANDARD_TEXTS = {
  MIT: `MIT License

{copyright}

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.`,
  ISC: `ISC License

{copyright}

Permission to use, copy, modify, and/or distribute this software for any purpose with or without fee is hereby granted, provided that the above copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.`,
  "BSD-2-Clause": `BSD 2-Clause License

{copyright}

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.`,
  "BSD-3-Clause": `BSD 3-Clause License

{copyright}

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.`,
  "Apache-2.0": `Apache License, Version 2.0

{copyright}

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.`,
  "0BSD": `BSD Zero Clause License

{copyright}

Permission to use, copy, modify, and/or distribute this software for any purpose with or without fee is hereby granted.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.`,
  Unlicense: `This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or distribute this software, either in source code form or as a compiled binary, for any purpose, commercial or non-commercial, and by any means.

For more information, please refer to <https://unlicense.org>`,
};

/** Splits "(MIT OR Apache-2.0)" or "MIT AND OFL-1.1" into its identifiers. */
export function licenseIds(expression) {
  return String(expression || "")
    .replace(/[()]/g, " ")
    .split(/\s+(?:OR|AND|WITH)\s+|\s+/i)
    .map((id) => id.trim())
    .filter(Boolean);
}

/** True when at least one alternative of the expression is a license the app may ship. */
export function isAllowed(expression) {
  const ids = licenseIds(expression);
  if (ids.length === 0) return false;
  if (/\bAND\b/i.test(String(expression))) return ids.every((id) => ALLOWED.has(id));
  return ids.some((id) => ALLOWED.has(id));
}

/** The license id whose standard text is used for a package without a LICENSE file. */
function standardIdFor(expression) {
  return licenseIds(expression).find((id) => STANDARD_TEXTS[id]) ?? null;
}

function authorOf(pkg) {
  const a = pkg.author;
  if (!a) return null;
  if (typeof a === "string") return a.replace(/\s*<[^>]*>|\s*\([^)]*\)/g, "").trim() || null;
  return a.name || null;
}

function licenseFileOf(dir) {
  let entries;
  try {
    entries = readdirSync(dir);
  } catch {
    return null;
  }
  const name = entries.find((e) => /^(LICEN[CS]E|COPYING)(\.|$|-)/i.test(e));
  return name ? readFileSync(join(dir, name), "utf8") : null;
}

/** Collects the notices of all installed production packages. Throws on a rule violation. */
export function collect() {
  const lock = JSON.parse(readFileSync(join(root, "package-lock.json"), "utf8"));
  const problems = [];
  const texts = [];
  const textIndex = new Map();
  const entries = [];
  for (const [key, meta] of Object.entries(lock.packages)) {
    if (!key || meta.dev) continue;
    const dir = join(root, key);
    let pkg;
    try {
      pkg = JSON.parse(readFileSync(join(dir, "package.json"), "utf8"));
    } catch {
      if (meta.optional) continue; // platform package for another OS or CPU, not installed
      problems.push(`${key}: not installed`);
      continue;
    }
    const license = pkg.license || meta.license || "";
    if (!isAllowed(license)) {
      problems.push(`${key}: license "${license || "unknown"}" is not allowed`);
      continue;
    }
    let text = licenseFileOf(dir);
    if (!text) {
      const id = standardIdFor(license);
      if (!id) {
        problems.push(`${key}: no LICENSE file and no standard text for "${license}"`);
        continue;
      }
      const author = authorOf(pkg);
      text = STANDARD_TEXTS[id].replace("{copyright}", author ? `Copyright (c) ${author}` : `Copyright (c) the ${pkg.name} authors`);
    }
    const normalized = text.replace(/\r\n/g, "\n").trim();
    let index = textIndex.get(normalized);
    if (index === undefined) {
      index = texts.length;
      texts.push(normalized);
      textIndex.set(normalized, index);
    }
    entries.push({ name: pkg.name || key.replace(/^.*node_modules\//, ""), version: pkg.version || meta.version || "", license, text: index });
  }
  if (problems.length > 0) {
    throw new Error("open-source notices cannot be generated:\n  " + problems.join("\n  "));
  }
  entries.sort((a, b) => a.name.localeCompare(b.name) || a.version.localeCompare(b.version));
  return { entries, texts };
}

export function render() {
  const { entries, texts } = collect();
  return [
    "// GENERATED by scripts/gen-licenses.mjs from package-lock.json and node_modules (postinstall). Do not",
    "// edit; run node scripts/gen-licenses.mjs. Shown under Rechtliches > Lizenzen (app/legal/licenses.tsx).",
    "",
    "export type LicenseEntry = { name: string; version: string; license: string; text: number };",
    "",
    `export const LICENSE_TEXTS: readonly string[] = ${JSON.stringify(texts)};`,
    "",
    `export const LICENSES: readonly LicenseEntry[] = ${JSON.stringify(entries)};`,
    "",
  ].join("\n");
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const next = render();
  if (process.argv.includes("--check")) {
    let current = "";
    try {
      current = readFileSync(OUTPUT, "utf8");
    } catch {
      // missing counts as out of date
    }
    if (current !== next) {
      console.error("lib/licenses.generated.ts is out of date: run node scripts/gen-licenses.mjs");
      process.exit(1);
    }
  } else {
    writeFileSync(OUTPUT, next);
    const { entries, texts } = collect();
    console.log(`wrote ${OUTPUT} (${entries.length} packages, ${texts.length} distinct texts, ${Math.round(next.length / 1024)} kB)`);
  }
}
