#!/usr/bin/env bash
# Verify a container image digest matches the CI-built artifact (plan 17.9 FR-6 / Security).
#
# Usage:
#   deploy/scripts/verify_image_digest.sh ghcr.io/org/server-go:abc123 abc123
#
# The expected digest may be passed directly, or as a git SHA (resolved via
# `docker buildx imagetools inspect` when crane/skopeo is unavailable).

set -euo pipefail

IMAGE="${1:-}"
EXPECTED="${2:-}"

die() { printf 'verify_image_digest: %s\n' "$*" >&2; exit 1; }

[ -n "$IMAGE" ] && [ -n "$EXPECTED" ] || die "usage: $0 <image-ref> <expected-sha-or-digest>"

resolve_digest() {
  local image="$1"
  if command -v crane >/dev/null 2>&1; then
    crane digest "$image"
    return
  fi
  if command -v skopeo >/dev/null 2>&1; then
    skopeo inspect --format '{{.Digest}}' "docker://${image}"
    return
  fi
  docker buildx imagetools inspect "$image" --format '{{json .}}' | python3 - <<'PY'
import json, sys
data = json.load(sys.stdin)
manifest = data.get("manifest") or data
digest = manifest.get("digest")
if not digest:
    raise SystemExit("could not resolve digest")
print(digest)
PY
}

ACTUAL="$(resolve_digest "$IMAGE")"

if [[ "$EXPECTED" == sha256:* ]]; then
  [ "$ACTUAL" = "$EXPECTED" ] || die "digest mismatch: got ${ACTUAL}, want ${EXPECTED}"
else
  # Tag must equal the git SHA when using immutable :${GITHUB_SHA} tags.
  TAG="${IMAGE##*:}"
  [ "$TAG" = "$EXPECTED" ] || die "image tag ${TAG} does not match expected commit ${EXPECTED}"
  echo "image tag matches commit ${EXPECTED}; digest ${ACTUAL}"
fi

echo "image verification ok: ${IMAGE} (${ACTUAL})"
