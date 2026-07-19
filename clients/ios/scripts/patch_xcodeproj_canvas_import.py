#!/usr/bin/env python3
"""Add MOB.2 Canvas import Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

# (filename, build_id, file_ref_id, group_id, is_test)
ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("CanvasImportLogic.swift", "B02C1A11C4A5A51B00A70101", "B02C1A11C4A5A51B00A71101", "B62295116F7F4935A884363D", False),
    ("LMSFeatureModelsCanvasImport.swift", "B02C1A11C4A5A51B00A70202", "B02C1A11C4A5A51B00A71202", "B62295116F7F4935A884363D", False),
    ("LMSAPICanvasImport.swift", "B02C1A11C4A5A51B00A70303", "B02C1A11C4A5A51B00A71303", "B62295116F7F4935A884363D", False),
    ("CanvasImportObservability.swift", "B02C1A11C4A5A51B00A70404", "B02C1A11C4A5A51B00A71404", "B62295116F7F4935A884363D", False),
    ("CanvasImportView.swift", "B02C1A11C4A5A51B00A70505", "B02C1A11C4A5A51B00A71505", "2B03EC44D1B1464BB3386389", False),
    ("CanvasImportLogicTests.swift", "B02C1A11C4A5A51B00A70606", "B02C1A11C4A5A51B00A71606", "98D67F731BF5451A9402BAA2", True),
]

APP_SOURCES = "B314AA5BFE0F4373814CBCC1"
TEST_SOURCES = "96E47D434BD040F5ABB658F0"


def phase_block(text: str, phase_id: str) -> str:
    match = re.search(
        rf"\t\t{re.escape(phase_id)} /\* Sources \*/ = \{{.*?\n\t\t\}};",
        text,
        re.S,
    )
    if not match:
        raise SystemExit(f"sources phase not found: {phase_id}")
    return match.group(0)


def group_block(text: str, group_id: str) -> str:
    match = re.search(
        rf"\t\t{re.escape(group_id)} /\* [^*]+ \*/ = \{{.*?\n\t\t\}};",
        text,
        re.S,
    )
    if not match:
        raise SystemExit(f"group not found: {group_id}")
    return match.group(0)


def insert_into_children(text: str, group_id: str, child_line: str) -> str:
    block = group_block(text, group_id)
    child_entry = f"\t\t\t\t{child_line},"
    if child_entry in block:
        return text
    marker = "\t\t\tchildren = (\n"
    if marker not in block:
        raise SystemExit(f"children missing in {group_id}")
    new_block = block.replace(marker, marker + child_entry + "\n", 1)
    return text.replace(block, new_block, 1)


def insert_into_files(text: str, sources_id: str, file_line: str) -> str:
    block = phase_block(text, sources_id)
    file_entry = f"\t\t\t\t{file_line},"
    if file_entry in block:
        return text
    marker = "\t\t\tfiles = (\n"
    if marker not in block:
        raise SystemExit(f"files missing in {sources_id}")
    new_block = block.replace(marker, marker + file_entry + "\n", 1)
    return text.replace(block, new_block, 1)


def main() -> None:
    text = PBX.read_text()
    for filename, build_id, file_ref_id, group_id, is_test in ENTRIES:
        if f"{file_ref_id} /* {filename} */" in text:
            print(f"skip existing {filename}")
            continue
        if len(build_id) != 24 or len(file_ref_id) != 24:
            raise SystemExit(f"invalid id length for {filename}")
        build_line = (
            f"\t\t{build_id} /* {filename} in Sources */ = "
            f"{{isa = PBXBuildFile; fileRef = {file_ref_id} /* {filename} */; }};\n"
        )
        file_ref_line = (
            f"\t\t{file_ref_id} /* {filename} */ = "
            f"{{isa = PBXFileReference; lastKnownFileType = sourcecode.swift; "
            f"path = {filename}; sourceTree = \"<group>\"; }};\n"
        )
        text = text.replace(
            "/* Begin PBXBuildFile section */\n",
            "/* Begin PBXBuildFile section */\n" + build_line,
            1,
        )
        text = text.replace(
            "/* Begin PBXFileReference section */\n",
            "/* Begin PBXFileReference section */\n" + file_ref_line,
            1,
        )
        text = insert_into_children(text, group_id, f"{file_ref_id} /* {filename} */")
        sources = TEST_SOURCES if is_test else APP_SOURCES
        text = insert_into_files(text, sources, f"{build_id} /* {filename} in Sources */")
        print(f"added {filename}")

    PBX.write_text(text)
    print("Updated project.pbxproj for Canvas import (MOB.2)")


if __name__ == "__main__":
    main()
