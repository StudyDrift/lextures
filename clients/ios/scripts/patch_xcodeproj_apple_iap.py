#!/usr/bin/env python3
"""Add Apple IAP (Path A) Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

# name, build_id, ref_id, group_id, is_test
# LMS group id from generate_xcodeproj / current pbxproj
LMS_GROUP = "8EFB3B79723747198F9CA7D8"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("AppleIAPModels.swift", "AIAP01C2D3E4F5061728394", "AIAP01D3E4F5061728394A5", LMS_GROUP, False),
    ("LMSAPIAppleIAP.swift", "AIAP02C2D3E4F5061728394", "AIAP02D3E4F5061728394A5", LMS_GROUP, False),
    ("StoreKitPurchaseService.swift", "AIAP03C2D3E4F5061728394", "AIAP03D3E4F5061728394A5", LMS_GROUP, False),
]

APP_SOURCES = "D6923400241A40BAA56CE419"
TEST_SOURCES = "8F1EC2F5497D49F7873CE22E"


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
    # Match both historical and current PBXSourcesBuildPhase layouts.
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
