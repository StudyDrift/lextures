#!/usr/bin/env python3
"""Add self.lextures.com subscribe paywall Swift files to the Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

LMS_GROUP = "8EFB3B79723747198F9CA7D8"
# Features/Billing group.
BILLING_GROUP = "620DCE58AC484229AF0B0EAE"
APP_SOURCES = "D6923400241A40BAA56CE419"
TEST_SOURCES = "8F1EC2F5497D49F7873CE22E"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("HomeschoolSubscribeGateLogic.swift", "HSG01C2D3E4F5061728394A", "HSG01D3E4F5061728394A5", LMS_GROUP, False),
    ("HomeschoolSubscribeGateStore.swift", "HSG02C2D3E4F5061728394A", "HSG02D3E4F5061728394A5", LMS_GROUP, False),
    ("HomeschoolSubscribePaywallView.swift", "HSG03C2D3E4F5061728394A", "HSG03D3E4F5061728394A5", BILLING_GROUP, False),
    ("HomeschoolSubscribeGateLogicTests.swift", "HSG04C2D3E4F5061728394A", "HSG04D3E4F5061728394A5", "D177268EB0164406B86F0376", True),
]


def insert_before(text: str, marker: str, block: str) -> str:
    idx = text.find(marker)
    if idx < 0:
        raise SystemExit(f"marker not found: {marker}")
    return text[:idx] + block + text[idx:]


def group_block(text: str, group_id: str) -> str | None:
    match = re.search(
        rf"\t\t{re.escape(group_id)} /\* [^*]+ \*/ = \{{.*?\n\t\t\}};",
        text,
        re.S,
    )
    return match.group(0) if match else None


def insert_into_children(text: str, group_id: str, child_line: str) -> str:
    block = group_block(text, group_id)
    if block is None:
        # Fall back to LMS for billing files if group missing
        if group_id == BILLING_GROUP:
            return insert_into_children(text, LMS_GROUP, child_line)
        raise SystemExit(f"group not found: {group_id}")
    child_entry = f"\t\t\t\t{child_line},"
    if child_entry in block:
        return text
    updated = block.replace(
        "\t\t\tchildren = (\n",
        f"\t\t\tchildren = (\n{child_entry}\n",
        1,
    )
    return text.replace(block, updated, 1)


def insert_into_sources(text: str, phase_id: str, build_line: str) -> str:
    entry = f"\t\t\t\t{build_line}"
    if entry in text:
        return text
    pattern = (
        rf"(\t\t{re.escape(phase_id)} /\* Sources \*/ = \{{\n"
        rf"\t\t\tisa = PBXSourcesBuildPhase;\n"
        rf"\t\t\tbuildActionMask = 2147483647;\n"
        rf"(?:\t\t\trunOnlyForDeploymentPostprocessing = 0;\n)?"
        rf"\t\t\tfiles = \()\n"
    )
    match = re.search(pattern, text)
    if not match:
        raise SystemExit(f"sources phase not found: {phase_id}")
    return text[: match.end()] + f"{entry}\n" + text[match.end() :]


def main() -> None:
    text = PBX.read_text()
    # Resolve tests group
    if "D177268EB0164406B86F0376" not in text:
        # find LexturesTests group by name
        m = re.search(r"(\w+) /\* LexturesTests \*/ = \{\n\t\t\tisa = PBXGroup;", text)
        if m:
            global ENTRIES
            ENTRIES = [
                e if not e[4] else (e[0], e[1], e[2], m.group(1), True) for e in ENTRIES
            ]

    missing = [e for e in ENTRIES if e[2] not in text]
    if not missing:
        print("Nothing to patch")
        return

    build_lines = []
    ref_lines = []
    for name, build_id, ref_id, _, _ in missing:
        build_lines.append(
            f"\t\t{build_id} /* {name} in Sources */ = {{isa = PBXBuildFile; fileRef = {ref_id} /* {name} */; }};"
        )
        ref_lines.append(
            f'\t\t{ref_id} /* {name} */ = {{isa = PBXFileReference; lastKnownFileType = sourcecode.swift; path = {name}; sourceTree = "<group>"; }};'
        )

    text = insert_before(text, "/* End PBXBuildFile section */", "\n".join(build_lines) + "\n")
    text = insert_before(text, "/* End PBXFileReference section */", "\n".join(ref_lines) + "\n")

    for name, build_id, ref_id, group_id, is_test in missing:
        text = insert_into_children(text, group_id, f"{ref_id} /* {name} */")
        phase_id = TEST_SOURCES if is_test else APP_SOURCES
        text = insert_into_sources(text, phase_id, f"{build_id} /* {name} in Sources */,")

    PBX.write_text(text)
    print(f"Patched {len(missing)} files into {PBX}")


if __name__ == "__main__":
    main()
