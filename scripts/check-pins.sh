#!/usr/bin/env bash
# Fails if the repository contains unpinned dependencies:
#   - package.json versions starting with ^ ~ > < * or equal to "latest"
#   - GitHub Actions "uses:" references that are not pinned to a 40-char commit SHA
# Requires: bash, jq, find, grep.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

failures=0

fail() {
  echo "FAIL: $*" >&2
  failures=$((failures + 1))
}

# --- package.json -----------------------------------------------------------
# Checks every string leaf below the dependency sections (this also covers the
# nested objects that "overrides" may contain).
while IFS= read -r -d '' file; do
  while IFS=$'\t' read -r path version; do
    case "$version" in
      '^'* | '~'* | '>'* | '<'* | '*'* | latest)
        fail "$file: $path = \"$version\" is not an exact version"
        ;;
    esac
  done < <(jq -r '
    [ .dependencies, .devDependencies, .peerDependencies, .optionalDependencies, .overrides, .resolutions ]
    | map(select(. != null))
    | .[]
    | . as $section
    | [paths(type == "string")]
    | .[]
    | . as $p
    | [($p | map(tostring) | join(".")), ($section | getpath($p))]
    | @tsv
  ' "$file")
done < <(find . -name package.json -not -path '*/node_modules/*' -not -path './.git/*' -print0)

# --- GitHub Actions ----------------------------------------------------------
if [[ -d .github ]]; then
  while IFS= read -r -d '' file; do
    while IFS= read -r line; do
      # Strip "uses:" (with optional list dash), trailing comment, quotes, whitespace.
      ref="${line#*uses:}"
      ref="${ref%%#*}"
      ref="${ref//[[:space:]\"\']/}"
      [[ "$ref" == ./* ]] && continue # local action or reusable workflow
      if ! [[ "$ref" =~ @[0-9a-f]{40}$ ]]; then
        fail "$file: 'uses: $ref' is not pinned to a 40-char commit SHA"
      fi
    done < <(grep -E '^[[:space:]-]*uses:' "$file" || true)
  done < <(find .github -type f \( -name '*.yml' -o -name '*.yaml' \) -print0)
fi

if ((failures > 0)); then
  echo "$failures pin violation(s) found." >&2
  exit 1
fi
echo "All pins OK."
