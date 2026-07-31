#!/usr/bin/env python3
"""Add CT.M6 pack-2 Swift files to the committed Xcode project."""

from __future__ import annotations

import re
import uuid
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

FILES_APP = [
    ("ContentToolPack2Logic.swift", "LMS"),
    ("ContentToolDraftStore.swift", "ContentTools"),
    ("ToolComposerView.swift", "ContentTools"),
    ("AskQuestionsToolView.swift", "Tools"),
    ("ExplainItBackToolView.swift", "Tools"),
    ("InlineDiscussionToolView.swift", "Tools"),
]
FILES_TEST = [
    ("ContentToolPack2LogicTests.swift", "Tests"),
]


def uid() -> str:
    return uuid.uuid4().hex[:24].upper()


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
    if child_entry in block or f"{child_line}," in block:
        return text
    updated = block.replace("\t\t\tchildren = (\n", f"\t\t\tchildren = (\n{child_entry}\n", 1)
    return text.replace(block, updated)


def insert_into_files(text: str, phase_id: str, file_line: str) -> str:
    match = re.search(
        rf"\t\t{re.escape(phase_id)} /\* Sources \*/ = \{{.*?\n\t\t\}};",
        text,
        re.S,
    )
    if not match:
        raise SystemExit(f"sources phase not found: {phase_id}")
    block = match.group(0)
    entry = f"\t\t\t\t{file_line},"
    if entry in block:
        return text
    updated = block.replace("\t\t\tfiles = (\n", f"\t\t\tfiles = (\n{entry}\n", 1)
    return text.replace(block, updated)


def find_group_containing(text: str, filename: str) -> str:
    for m in re.finditer(r"\t\t([A-F0-9]{24}) /\* [^*]+ \*/ = \{\n\t\t\tisa = PBXGroup;", text):
        block = group_block(text, m.group(1))
        if block and f"/* {filename} */" in block:
            return m.group(1)
    raise SystemExit(f"group containing {filename} not found")


def find_sources_phase_containing(text: str, filename: str) -> str:
    for m in re.finditer(r"\t\t([A-F0-9]{24}) /\* Sources \*/ = \{", text):
        block_match = re.search(
            rf"\t\t{re.escape(m.group(1))} /\* Sources \*/ = \{{.*?\n\t\t\}};",
            text,
            re.S,
        )
        if block_match and f"{filename} in Sources" in block_match.group(0):
            return m.group(1)
    raise SystemExit(f"sources phase containing {filename} not found")


def main() -> None:
    text = PBX.read_text()

    lms_group = find_group_containing(text, "ContentToolHostLogic.swift")
    content_tools_group = find_group_containing(text, "ToolRendererRegistry.swift")
    try:
        tools_group = find_group_containing(text, "InlineQuestionsToolView.swift")
    except SystemExit:
        tools_group = content_tools_group
    tests_group = find_group_containing(text, "ContentToolPack1LogicTests.swift")
    app_sources = find_sources_phase_containing(text, "ContentToolPack1Logic.swift")
    test_sources = find_sources_phase_containing(text, "ContentToolPack1LogicTests.swift")

    group_map = {
        "LMS": lms_group,
        "ContentTools": content_tools_group,
        "Tools": tools_group,
        "Tests": tests_group,
    }

    build_block = ""
    file_block = ""
    for name, group_key in FILES_APP + [(n, g) for n, g in FILES_TEST]:
        if f"/* {name} */" in text and f"path = {name};" in text:
            continue
        build_id = uid()
        file_id = uid()
        is_test = group_key == "Tests"
        build_block += (
            f"\n\t\t{build_id} /* {name} in Sources */ = "
            f"{{isa = PBXBuildFile; fileRef = {file_id} /* {name} */; }};"
        )
        file_block += (
            f"\n\t\t{file_id} /* {name} */ = {{isa = PBXFileReference; "
            f"lastKnownFileType = sourcecode.swift; path = {name}; sourceTree = \"<group>\"; }};"
        )
        phase = test_sources if is_test else app_sources
        text = insert_into_files(text, phase, f"{build_id} /* {name} in Sources */")
        text = insert_into_children(text, group_map[group_key], f"{file_id} /* {name} */")

    if build_block:
        text = insert_before(text, "/* End PBXBuildFile section */", build_block)
    if file_block:
        text = insert_before(text, "/* End PBXFileReference section */", file_block)

    PBX.write_text(text)
    print("patched", PBX)
    print("lms", lms_group, "contentTools", content_tools_group, "tools", tools_group)
    print("appSources", app_sources, "testSources", test_sources)


if __name__ == "__main__":
    main()
