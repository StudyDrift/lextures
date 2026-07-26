#!/usr/bin/env python3
"""TD.3 — validate OpenAPI JSON and documentation coverage baseline.

Environment:
  SPEC       path to openapi.json
  BASELINE   path to openapi-coverage.txt (min_documented_paths=N)
  INVENTORY  path to route_inventory.golden (METHOD\\tPATTERN\\tAUTH)
"""
from __future__ import annotations

import json
import os
import re
import sys
from pathlib import Path

PARAM_RE = re.compile(r"\{[^/]+\}")


def die(msg: str, code: int = 1) -> None:
    print(f"openapi-check: {msg}", file=sys.stderr)
    raise SystemExit(code)


def load_json_strict(path: Path) -> dict:
    raw = path.read_bytes()
    try:
        text = raw.decode("utf-8")
    except UnicodeDecodeError as e:
        die(f"{path}: not utf-8: {e}")
    dec = json.JSONDecoder()
    try:
        obj, idx = dec.raw_decode(text)
    except json.JSONDecodeError as e:
        die(f"{path}: invalid JSON: {e}")
    rest = text[idx:].strip()
    if rest:
        line = text.count("\n", 0, idx) + 1
        die(f"{path}: trailing data after first JSON value (byte {idx}, ~line {line}): {rest[:80]!r}")
    if not isinstance(obj, dict):
        die(f"{path}: root must be an object")
    return obj


def collect_refs(node, out: list[str]) -> None:
    if isinstance(node, dict):
        ref = node.get("$ref")
        if isinstance(ref, str):
            out.append(ref)
        for v in node.values():
            collect_refs(v, out)
    elif isinstance(node, list):
        for v in node:
            collect_refs(v, out)


def resolve_local_ref(doc: dict, ref: str) -> bool:
    if not ref.startswith("#/"):
        return False
    cur: object = doc
    for part in ref[2:].split("/"):
        part = part.replace("~1", "/").replace("~0", "~")
        if not isinstance(cur, dict) or part not in cur:
            return False
        cur = cur[part]
    return True


def normalize_path(p: str) -> str:
    p = p.rstrip("/")
    if p == "":
        p = "/"
    return PARAM_RE.sub("{}", p)


def load_inventory_patterns(path: Path) -> list[str]:
    patterns: list[str] = []
    seen: set[str] = set()
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        parts = line.split("\t")
        if len(parts) < 2:
            continue
        pat = parts[1]
        if pat in seen:
            continue
        seen.add(pat)
        patterns.append(pat)
    return patterns


def path_covered(doc_path: str, registered: list[str]) -> bool:
    doc_norm = normalize_path(doc_path)
    for reg in registered:
        if reg.endswith("/*"):
            prefix = reg[: -len("/*")]
            if doc_path == prefix or doc_path.startswith(prefix + "/"):
                return True
            continue
        if normalize_path(reg) == doc_norm:
            return True
    return False


def load_baseline(path: Path) -> int:
    min_paths = 0
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("min_documented_paths="):
            try:
                min_paths = int(line.split("=", 1)[1].strip())
            except ValueError:
                die(f"{path}: invalid min_documented_paths")
    if min_paths <= 0:
        die(f"{path}: missing min_documented_paths")
    return min_paths


def main() -> None:
    spec_path = Path(os.environ.get("SPEC", ""))
    baseline_path = Path(os.environ.get("BASELINE", ""))
    inv_path = Path(os.environ.get("INVENTORY", ""))

    if not spec_path.is_file():
        die(f"spec not found: {spec_path}")
    if not baseline_path.is_file():
        die(f"baseline not found: {baseline_path}")
    if not inv_path.is_file():
        die(f"route inventory not found: {inv_path} (TD.1 dependency)")

    doc = load_json_strict(spec_path)

    if doc.get("openapi") != "3.0.3":
        die(f"openapi version must be 3.0.3, got {doc.get('openapi')!r}")
    info = doc.get("info")
    if not isinstance(info, dict) or not info.get("title") or not info.get("version"):
        die("info.title and info.version required")
    paths = doc.get("paths")
    if not isinstance(paths, dict) or not paths:
        die("paths must be a non-empty object")
    components = doc.get("components")
    if not isinstance(components, dict):
        die("components must be present")
    schemes = components.get("securitySchemes")
    if not isinstance(schemes, dict) or "bearerAuth" not in schemes:
        die("components.securitySchemes.bearerAuth missing")
    bearer = schemes["bearerAuth"]
    if not isinstance(bearer, dict) or bearer.get("type") != "http" or bearer.get("scheme") != "bearer":
        die("components.securitySchemes.bearerAuth must be type=http scheme=bearer")

    refs: list[str] = []
    collect_refs(doc, refs)
    missing = []
    for ref in sorted(set(refs)):
        if not ref.startswith("#/"):
            missing.append(f"{ref} (non-local)")
            continue
        if not resolve_local_ref(doc, ref):
            missing.append(ref)
    if missing:
        die("unresolved $ref(s):\n  " + "\n  ".join(missing))

    registered = load_inventory_patterns(inv_path)
    phantoms = sorted(p for p in paths if not path_covered(p, registered))
    if phantoms:
        die(
            f"{len(phantoms)} documented path(s) have no matching registered route:\n  "
            + "\n  ".join(phantoms)
        )

    n = len(paths)
    total = len(registered)
    pct = (100.0 * n / total) if total else 0.0
    min_paths = load_baseline(baseline_path)
    print(f"openapi-check: documentation coverage: {n} / {total} unique routes ({pct:.1f}%)")
    print(f"openapi-check: baseline min_documented_paths={min_paths}")
    print(f"openapi-check: $refs resolved: {len(set(refs))}")
    if n < min_paths:
        die(
            f"documented paths {n} < baseline {min_paths} — coverage decreased (TD.3 ratchet). "
            "Restore path docs or raise documentation; do not lower the baseline without review."
        )
    print("openapi-check: OK")


if __name__ == "__main__":
    main()
