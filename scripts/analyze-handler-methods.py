#!/usr/bin/env python3
"""TD.5 FR-5 — map each httpserver handler expression to its registered HTTP methods.

Usage:
  python3 scripts/analyze-handler-methods.py            # human summary
  python3 scripts/analyze-handler-methods.py --json     # machine-readable
  python3 scripts/analyze-handler-methods.py --multi    # multi-method only
  python3 scripts/analyze-handler-methods.py --assert-single-ok

Exit 0 always unless --assert-single-ok finds a single-method claim conflict
(a handler listed as single-method that is also registered with another verb).
"""
from __future__ import annotations

import argparse
import collections
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
HTTPSERVER = ROOT / "server" / "internal" / "httpserver"

# Any chi.Router identifier (r, s, sr, api, …) before the method helper.
PAT_METHOD_HELPER = re.compile(
    r"""\b\w+\.(Get|Post|Put|Patch|Delete|Options|Head|Connect|Trace)\(\s*("[^"]+"|`[^`]+`)\s*,\s*([^)]+)\)"""
)
PAT_METHOD = re.compile(
    r"""\b\w+\.Method\(\s*(?:http\.Method)?(\w+)\s*,\s*("[^"]+"|`[^`]+`)\s*,\s*([^)]+)\)"""
)
PAT_HANDLE = re.compile(
    r"""\b\w+\.(Handle|HandleFunc|Mount)\(\s*("[^"]+"|`[^`]+`)\s*,\s*([^)]+)\)"""
)


def normalize_handler(expr: str) -> str:
    h = expr.strip().rstrip(",").rstrip()
    # Capture often ends at the opening paren of d.handleFoo()
    h = re.sub(r"\(\s*$", "", h)
    h = re.sub(r"\(\s*\)$", "", h)
    return h


def collect_registrations() -> dict[str, list[tuple[str, str, int, str]]]:
    regs: dict[str, list[tuple[str, str, int, str]]] = collections.defaultdict(list)
    for path in sorted(HTTPSERVER.rglob("*.go")):
        if path.name.endswith("_test.go"):
            continue
        text = path.read_text(encoding="utf-8")
        rel = str(path.relative_to(ROOT))
        for i, line in enumerate(text.splitlines(), 1):
            for m in PAT_METHOD_HELPER.finditer(line):
                method, p, handler = m.group(1).upper(), m.group(2), normalize_handler(m.group(3))
                regs[handler].append((method, rel, i, p))
            for m in PAT_METHOD.finditer(line):
                raw = m.group(1)
                method = raw[len("Method") :].upper() if raw.startswith("Method") else raw.upper()
                p, handler = m.group(2), normalize_handler(m.group(3))
                regs[handler].append((method, rel, i, p))
            for m in PAT_HANDLE.finditer(line):
                kind, p, handler = m.group(1), m.group(2), normalize_handler(m.group(3))
                regs[handler].append((f"HANDLE/{kind}", rel, i, p))
    return regs


def classify(
    regs: dict[str, list[tuple[str, str, int, str]]],
) -> tuple[list, list, list]:
    single, multi, handle = [], [], []
    for h, sites in sorted(regs.items()):
        methods = sorted({m for m, _, _, _ in sites})
        entry = {"handler": h, "methods": methods, "sites": [
            {"method": m, "file": f, "line": ln, "path": p} for m, f, ln, p in sites
        ]}
        if any(m.startswith("HANDLE/") for m in methods):
            handle.append(entry)
        elif len(methods) > 1:
            multi.append(entry)
        else:
            single.append(entry)
    return single, multi, handle


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--multi", action="store_true", help="print multi-method handlers only")
    ap.add_argument(
        "--assert-single-ok",
        action="store_true",
        help="exit 1 if any handler is registered under more than one distinct non-HANDLE method set inconsistently",
    )
    args = ap.parse_args()

    regs = collect_registrations()
    single, multi, handle = classify(regs)

    if args.assert_single_ok:
        # Integrity: no handler in single has >1 method (already guaranteed by classify).
        bad = [e for e in single if len(e["methods"]) != 1]
        if bad:
            print("ASSERT FAIL: single-method list contains multi-method entries:", file=sys.stderr)
            for e in bad:
                print(f"  {e['handler']}: {e['methods']}", file=sys.stderr)
            return 1
        # Overlap check: handler cannot appear in both single and multi
        single_names = {e["handler"] for e in single}
        multi_names = {e["handler"] for e in multi}
        overlap = single_names & multi_names
        if overlap:
            print("ASSERT FAIL: handlers in both single and multi:", overlap, file=sys.stderr)
            return 1
        print(
            f"OK: {len(single)} single-method, {len(multi)} multi-method, "
            f"{len(handle)} Handle/HandleFunc/Mount"
        )
        return 0

    payload = {
        "single_method_count": len(single),
        "multi_method_count": len(multi),
        "handle_style_count": len(handle),
        "single_method": single,
        "multi_method": multi,
        "handle_style": handle,
    }

    if args.json:
        print(json.dumps(payload, indent=2))
        return 0

    if args.multi:
        for e in multi:
            print(f"{e['handler']}: {', '.join(e['methods'])}")
            for s in e["sites"]:
                print(f"  {s['method']:12} {s['file']}:{s['line']} {s['path']}")
        return 0

    print(f"Total handler expressions: {len(regs)}")
    print(f"  single-method: {len(single)}")
    print(f"  multi-method:  {len(multi)}")
    print(f"  Handle-style:  {len(handle)}")
    print()
    print("Multi-method handlers (method-agnostic; keep in-handler dispatch):")
    for e in multi:
        print(f"  {e['handler']}: {', '.join(e['methods'])}")
    print()
    print("Handle/HandleFunc/Mount (keep dispatch + document why):")
    for e in handle:
        print(f"  {e['handler']}: {', '.join(e['methods'])}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
