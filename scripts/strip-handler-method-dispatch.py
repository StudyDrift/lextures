#!/usr/bin/env python3
"""TD.5 — remove unreachable in-handler method-dispatch prologues from single-method handlers.

Removes only the standard patterns that chi + corsAll already handle:

  if r.Method == http.MethodOptions {
      w.WriteHeader(http.StatusNoContent)
      return
  }
  if r.Method != http.MethodX {
      w.Header().Set("Allow", ...)
      http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
      return
  }

Preserves:
  - cors.go (corsAll is load-bearing)
  - not_found_response.go (central chi 404/405)
  - multi-method handlers (switch / multi-or method checks) — FR-6
  - Handle/HandleFunc/Mount handlers (CalDAV) — FR-4
  - any non-standard method check that is not the exact prologue form

Usage:
  python3 scripts/strip-handler-method-dispatch.py           # apply
  python3 scripts/strip-handler-method-dispatch.py --dry-run # counts only
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
HTTPSERVER = ROOT / "server" / "internal" / "httpserver"

# Files that own real method/OPTIONS behaviour and must never be stripped.
SKIP_FILES = {
    "cors.go",
    "not_found_response.go",
    "unimplemented_v1.go",  # Handle-style 501 stub (empty registration list today)
}

def load_multi_method_funcs() -> set[str]:
    """FR-6: handlers registered under more than one verb, plus switch/multi-or bodies."""
    import subprocess
    import json as _json

    out = subprocess.check_output(
        ["python3", str(ROOT / "scripts" / "analyze-handler-methods.py"), "--json"],
        text=True,
    )
    data = _json.loads(out)
    names: set[str] = set()
    for e in data.get("multi_method", []) + data.get("handle_style", []):
        h = e["handler"]
        # d.handleFoo → handleFoo
        if h.startswith("d."):
            h = h[2:]
        names.add(h)

    # Also protect any Deps handler whose body uses switch r.Method or multi-or checks
    # (covers SCIM and similar where registration uses a local router var).
    func_re = re.compile(r"^func \(d Deps\) (handle\w+)\(\) http\.HandlerFunc \{", re.M)
    switch_re = re.compile(r"switch r\.Method \{")
    multi_or_re = re.compile(
        r"r\.Method != http\.Method\w+ && r\.Method != http\.Method\w+"
    )
    for path in HTTPSERVER.rglob("*.go"):
        if path.name.endswith("_test.go"):
            continue
        text = path.read_text(encoding="utf-8")
        matches = list(func_re.finditer(text))
        for i, m in enumerate(matches):
            start = m.start()
            end = matches[i + 1].start() if i + 1 < len(matches) else len(text)
            body = text[start:end]
            if switch_re.search(body) or multi_or_re.search(body):
                names.add(m.group(1))
    return names


MULTI_METHOD_FUNCS: set[str] | None = None

OPTIONS_BLOCK = re.compile(
    r"^[ \t]*if r\.Method == http\.MethodOptions \{\n"
    r"(?:[ \t]*w\.Header\(\)\.Set\(\"Access-Control-[^\"]+\", [^\n]+\)\n)*"
    r"[ \t]*w\.WriteHeader\(http\.StatusNoContent\)\n"
    r"[ \t]*return\n"
    r"[ \t]*\}\n",
    re.M,
)

# Broader single-method check (Allow expression not backref-tied):
SINGLE_METHOD_CHECK_LOOSE = re.compile(
    r"^[ \t]*if r\.Method != http\.Method(Get|Post|Put|Patch|Delete|Head) \{\n"
    r"[ \t]*w\.Header\(\)\.Set\(\"Allow\", [^\n]+\)\n"
    r"[ \t]*http\.Error\(w, http\.StatusText\(http\.StatusMethodNotAllowed\), http\.StatusMethodNotAllowed\)\n"
    r"[ \t]*return\n"
    r"[ \t]*\}\n",
    re.M,
)

# Helper wrappers used by admin jobs/scheduler/content-tools and rbac settings.
HELPER_METHOD_CHECK = re.compile(
    r"^[ \t]*if r\.Method != http\.Method(Get|Post|Put|Patch|Delete|Head) \{\n"
    r"[ \t]*(?:jobsMethodNotAllowed\(w, http\.Method\1\)|allowGet\(w, r\))\n"
    r"[ \t]*return\n"
    r"[ \t]*\}\n",
    re.M,
)

# No Allow header — just http.Error 405.
METHOD_CHECK_NO_ALLOW = re.compile(
    r"^[ \t]*if r\.Method != http\.Method(Get|Post|Put|Patch|Delete|Head) \{\n"
    r"[ \t]*http\.Error\(w, http\.StatusText\(http\.StatusMethodNotAllowed\), http\.StatusMethodNotAllowed\)\n"
    r"[ \t]*return\n"
    r"[ \t]*\}\n",
    re.M,
)

# SCIM-specific 405 body helper.
SCIM_METHOD_CHECK = re.compile(
    r"^[ \t]*if r\.Method != http\.Method(Get|Post|Put|Patch|Delete|Head) \{\n"
    r"[ \t]*w\.Header\(\)\.Set\(\"Allow\", [^\n]+\)\n"
    r"[ \t]*writeSCIMError\(w, http\.StatusMethodNotAllowed, [^\n]+\)\n"
    r"[ \t]*return\n"
    r"[ \t]*\}\n",
    re.M,
)

# Multi-or method checks must never be removed by the loose pattern — they won't match
# because they use &&.

FUNC_START = re.compile(
    r"^func \(d Deps\) (handle\w+)\([^)]*\) http\.HandlerFunc \{",
    re.M,
)


def split_handler_funcs(text: str) -> list[tuple[str | None, str, int, int]]:
    """Return list of (func_name|None, body_slice, start, end) covering the whole file.

    Non-handler regions get func_name=None.
    """
    matches = list(FUNC_START.finditer(text))
    if not matches:
        return [(None, text, 0, len(text))]

    parts: list[tuple[str | None, str, int, int]] = []
    if matches[0].start() > 0:
        parts.append((None, text[: matches[0].start()], 0, matches[0].start()))

    for i, m in enumerate(matches):
        start = m.start()
        end = matches[i + 1].start() if i + 1 < len(matches) else len(text)
        name = m.group(1)
        parts.append((name, text[start:end], start, end))
    return parts


def strip_prologue(body: str) -> tuple[str, int, int]:
    """Strip OPTIONS + single-method checks. Returns (new_body, opt_removed, method_removed)."""
    opt_n = 0
    method_n = 0

    def rem_opt(m: re.Match) -> str:
        nonlocal opt_n
        opt_n += 1
        return ""

    def rem_method(m: re.Match) -> str:
        nonlocal method_n
        method_n += 1
        return ""

    body = OPTIONS_BLOCK.sub(rem_opt, body)
    body = SINGLE_METHOD_CHECK_LOOSE.sub(rem_method, body)
    body = HELPER_METHOD_CHECK.sub(rem_method, body)
    body = METHOD_CHECK_NO_ALLOW.sub(rem_method, body)
    body = SCIM_METHOD_CHECK.sub(rem_method, body)
    return body, opt_n, method_n


def process_file(path: Path, dry_run: bool, multi: set[str]) -> tuple[int, int]:
    if path.name in SKIP_FILES:
        return 0, 0
    text = path.read_text(encoding="utf-8")
    parts = split_handler_funcs(text)
    out: list[str] = []
    total_opt = 0
    total_method = 0
    for name, body, _, _ in parts:
        if name is not None and name in multi:
            # FR-6: leave multi-method dispatch intact (including OPTIONS short-circuit).
            out.append(body)
            continue
        new_body, opt_n, method_n = strip_prologue(body)
        total_opt += opt_n
        total_method += method_n
        out.append(new_body)
    if total_opt or total_method:
        new_text = "".join(out)
        # Collapse triple blank lines introduced by removals
        new_text = re.sub(r"\n{3,}", "\n\n", new_text)
        if not dry_run and new_text != text:
            path.write_text(new_text, encoding="utf-8")
    return total_opt, total_method


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()

    multi = load_multi_method_funcs()
    print(f"Protecting {len(multi)} multi-method / Handle-style handlers")

    total_opt = total_method = files = 0
    for path in sorted(HTTPSERVER.rglob("*.go")):
        if path.name.endswith("_test.go"):
            continue
        o, m = process_file(path, args.dry_run, multi)
        if o or m:
            files += 1
            total_opt += o
            total_method += m
            print(f"{path.relative_to(ROOT)}: OPTIONS={o} MethodNotAllowed={m}")

    mode = "dry-run" if args.dry_run else "applied"
    print(
        f"\n[{mode}] files={files} OPTIONS_removed={total_opt} "
        f"MethodNotAllowed_removed={total_method}"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
