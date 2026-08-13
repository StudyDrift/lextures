#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
found="$(find "$repo_root/www/src/blog" "$repo_root/www/src/docs" -type f \( -name '*.md' -o -name '*.mdx' \) -print 2>/dev/null || true)"

if [[ -n "$found" ]]; then
  echo "File-based marketing articles are no longer supported. Publish through the Marketing Content workspace:" >&2
  echo "$found" >&2
  exit 1
fi

echo "No file-based marketing articles found."
