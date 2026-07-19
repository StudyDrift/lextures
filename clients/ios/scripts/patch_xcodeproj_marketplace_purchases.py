#!/usr/bin/env python3
"""Add MOB.7 marketplace purchase Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

# name, build_id, ref_id, group_id, is_test
ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("MyPurchasesView.swift", "M770P01C2D3E4F5061728394", "M770P01D3E4F5061728394A5", "CABDDB4B529B4EE5A34A5A32", False),
]

APP_SOURCES = "B314AA5BFE0F4373814CBCC1"
TEST_SOURCES = "96E47D434BD040F5ABB658F0"


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
    entry = f"\t\t\t\t{build_line},"
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
    if entry in text:
        return text
    return text[: match.end(1)] + "\n" + entry + "\n" + text[match.end(1) :]


def main() -> None:
    text = PBX.read_text(encoding="utf-8")

    for name, build_id, ref_id, group_id, is_test in ENTRIES:
        if f"{ref_id} /* {name} */ = {{isa = PBXFileReference" in text:
            continue
        file_ref = (
            f"\t\t{ref_id} /* {name} */ = {{isa = PBXFileReference; lastKnownFileType = sourcecode.swift; "
            f'path = {name}; sourceTree = "<group>"; }};\n'
        )
        build_file = f"\t\t{build_id} /* {name} in Sources */ = {{isa = PBXBuildFile; fileRef = {ref_id} /* {name} */; }};\n"
        text = insert_before(text, "/* End PBXBuildFile section */", build_file)
        text = insert_before(text, "/* End PBXFileReference section */", file_ref)
        text = insert_into_children(text, group_id, f"{ref_id} /* {name} */")
        text = insert_into_sources(
            text,
            TEST_SOURCES if is_test else APP_SOURCES,
            f"{build_id} /* {name} in Sources */",
        )

    PBX.write_text(text, encoding="utf-8")
    print("Patched Xcode project for MOB.7 marketplace purchase files.")


if __name__ == "__main__":
    main()
