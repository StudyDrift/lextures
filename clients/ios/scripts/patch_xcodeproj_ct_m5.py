#!/usr/bin/env python3
"""Add CT.M5 pack-1 Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

# (filename, build_id, file_id, group_id, is_test)
ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("ContentToolPack1Logic.swift", "C5M501C2D3E4F5061728394", "C5M501D3E4F5061728394A5", "D7AC7435149B4B17A073AD6F", False),
    ("InlineQuestionsToolView.swift", "C5M502C2D3E4F5061728394", "C5M502D3E4F5061728394A5", "C5M50TOOLS015C6D7E8F90123", False),
    ("PredictRevealToolView.swift", "C5M503C2D3E4F5061728394", "C5M503D3E4F5061728394A5", "C5M50TOOLS015C6D7E8F90123", False),
    ("ClassPulseToolView.swift", "C5M504C2D3E4F5061728394", "C5M504D3E4F5061728394A5", "C5M50TOOLS015C6D7E8F90123", False),
    ("FlashcardsToolView.swift", "C5M505C2D3E4F5061728394", "C5M505D3E4F5061728394A5", "C5M50TOOLS015C6D7E8F90123", False),
    ("ContentToolPack1LogicTests.swift", "C5M506C2D3E4F5061728394", "C5M506D3E4F5061728394A5", "8D3F9FB222A34107B54468F2", True),
]

NEW_GROUPS = """
\t\tC5M50TOOLS015C6D7E8F90123 /* Tools */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\tC5M502D3E4F5061728394A5 /* InlineQuestionsToolView.swift */,
\t\t\t\tC5M503D3E4F5061728394A5 /* PredictRevealToolView.swift */,
\t\t\t\tC5M504D3E4F5061728394A5 /* ClassPulseToolView.swift */,
\t\t\t\tC5M505D3E4F5061728394A5 /* FlashcardsToolView.swift */,
\t\t\t);
\t\t\tpath = Tools;
\t\t\tsourceTree = "<group>";
\t\t};
"""

APP_SOURCES = "B3991B6DD6F9495DB8B2870B"
TEST_SOURCES = "693EA1D9511A4AB8B7F10A82"
CONTENT_TOOLS_GROUP = None  # resolved dynamically
TOOLS_SUBGROUP = "C5M50TOOLS015C6D7E8F90123 /* Tools */"


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
    updated = block.replace("\t\t\tchildren = (\n", f"\t\t\tchildren = (\n{child_entry}\n", 1)
    return text.replace(block, updated)


def find_content_tools_group(text: str) -> str:
    match = re.search(r"\t\t([A-F0-9]{24}) /\* ContentTools \*/ = \{", text)
    if not match:
        raise SystemExit("ContentTools group not found")
    return match.group(1)


def find_lms_group_for_pack1(text: str) -> str:
    # ContentToolHostLogic.swift lives in Core/LMS group
    match = re.search(
        r"\t\t([A-F0-9]{24}) /\* LMS \*/ = \{.*?ContentToolHostLogic\.swift",
        text,
        re.S,
    )
    if match:
        return match.group(1)
    # fallback: group that already has ContentToolHostLogic file ref as child
    for m in re.finditer(r"\t\t([A-F0-9]{24}) /\* [^*]+ \*/ = \{\n\t\t\tisa = PBXGroup;", text):
        block = group_block(text, m.group(1))
        if block and "ContentToolHostLogic.swift" in block and "ContentToolSandboxLogic.swift" in block:
            return m.group(1)
    raise SystemExit("LMS group containing ContentToolHostLogic not found")


def main() -> None:
    text = PBX.read_text()
    content_tools_group = find_content_tools_group(text)
    lms_group = find_lms_group_for_pack1(text)

    # Fix group for ContentToolPack1Logic to discovered LMS group
    global ENTRIES
    entries = []
    for name, build_id, file_id, group_id, is_test in ENTRIES:
        if name == "ContentToolPack1Logic.swift":
            group_id = lms_group
        entries.append((name, build_id, file_id, group_id, is_test))

    if "C5M50TOOLS015C6D7E8F90123 /* Tools */" not in text:
        text = insert_before(text, "/* End PBXGroup section */", NEW_GROUPS)
        text = insert_into_children(text, content_tools_group, TOOLS_SUBGROUP)

    build_block = ""
    file_block = ""
    for name, build_id, file_id, group_id, is_test in entries:
        if f"{file_id} /* {name} */" in text:
            continue
        build_block += f"\n\t\t{build_id} /* {name} in Sources */ = {{isa = PBXBuildFile; fileRef = {file_id} /* {name} */; }};"
        file_block += f"\n\t\t{file_id} /* {name} */ = {{isa = PBXFileReference; lastKnownFileType = sourcecode.swift; path = {name}; sourceTree = \"<group>\"; }};"
        sources = TEST_SOURCES if is_test else APP_SOURCES
        text = insert_into_children(text, sources, f"{build_id} /* {name} in Sources */")
        text = insert_into_children(text, group_id, f"{file_id} /* {name} */")
    if build_block:
        text = insert_before(text, "/* End PBXBuildFile section */", build_block)
    if file_block:
        text = insert_before(text, "/* End PBXFileReference section */", file_block)

    PBX.write_text(text)
    print("patched", PBX)


if __name__ == "__main__":
    main()
