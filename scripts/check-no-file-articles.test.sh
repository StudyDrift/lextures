#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fixture="$repo_root/www/src/blog/new-post.md"
mkdir -p "$(dirname "$fixture")"
trap 'rm -f "$fixture"; rmdir "$(dirname "$fixture")" 2>/dev/null || true' EXIT

bash "$repo_root/scripts/check-no-file-articles.sh"
printf '%s\n' '# planted guardrail fixture' > "$fixture"
if bash "$repo_root/scripts/check-no-file-articles.sh" >/dev/null 2>&1; then
  echo "guardrail accepted a file-based article" >&2
  exit 1
fi

echo "File-article guardrail self-test passed."
