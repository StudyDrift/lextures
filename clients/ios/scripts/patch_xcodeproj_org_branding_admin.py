#!/usr/bin/env python3
"""Add M14.5 Org Branding Admin Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("LMSFeatureModelsOrgBrandingAdmin.swift", "8B771C2D3E4F5061728394", "8B771D3E4F5061728394A5", "D7AC7435149B4B17A073AD6F", False),
    ("OrgBrandingAdminLogic.swift", "8B772C2D3E4F5061728394", "8B772D3E4F5061728394A5", "D7AC7435149B4B17A073AD6F", False),
    ("OrgBrandingAdminView.swift", "8B773C2D3E4F5061728394", "8B773D3E4F5061728394A5", "8B72ADM015C6D7E8F901234567", False),
    ("OrgBrandingView.swift", "8B774C2D3E4F5061728394", "8B774D3E4F5061728394A5", "8B72ADM015C6D7E8F901234567", False),
    ("AiGovernanceView.swift", "8B775C2D3E4F5061728394", "8B775D3E4F5061728394A5", "8B72ADM015C6D7E8F901234567", False),
    ("AiProviderSettingsView.swift", "8B776C2D3E4F5061728394", "8B776D3E4F5061728394A5", "8B72ADM015C6D7E8F901234567", False),
    ("OrgBrandingAdminEntryCard.swift", "8B777C2D3E4F5061728394", "8B777D3E4F5061728394A5", "029927E4F879434FA36100DE", False),
    ("OrgBrandingAdminLogicTests.swift", "8B778C2D3E4F5061728394", "8B778D3E4F5061728394A5", "8D3F9FB222A34107B54468F2", True),
]

APP_SOURCES = "B3991B6DD6F9495DB8B2870B"
TEST_SOURCES = "693EA1D9511A4AB8B7F10A82"


def insert_before(text: str, marker: str, block: str) -> str:
    idx = text.find(marker)
    if idx < 0:
        raise SystemExit(f"marker not found: {marker}")
    if not block.endswith("\n"):
        block += "\n"
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


def insert_into_files(text: str, phase_id: str, child_line: str) -> str:
    match = re.search(
        rf"\t\t{re.escape(phase_id)} /\* Sources \*/ = \{{.*?files = \(\n",
        text,
        re.S,
    )
    if match is None:
        raise SystemExit(f"sources phase not found: {phase_id}")
    insert_at = match.end()
    entry = f"\t\t\t\t{child_line},\n"
    if entry in text:
        return text
    return text[:insert_at] + entry + text[insert_at:]


def main() -> None:
    text = PBX.read_text()
    build_block = ""
    file_block = ""
    for name, build_id, file_id, group_id, is_test in ENTRIES:
        if f"{file_id} /* {name} */" in text:
            continue
        build_block += f"\n\t\t{build_id} /* {name} in Sources */ = {{isa = PBXBuildFile; fileRef = {file_id} /* {name} */; }};"
        file_block += f"\n\t\t{file_id} /* {name} */ = {{isa = PBXFileReference; lastKnownFileType = sourcecode.swift; path = {name}; sourceTree = \"<group>\"; }};"
        sources = TEST_SOURCES if is_test else APP_SOURCES
        text = insert_into_files(text, sources, f"{build_id} /* {name} in Sources */")
        text = insert_into_children(text, group_id, f"{file_id} /* {name} */")
    if build_block:
        text = insert_before(text, "/* End PBXBuildFile section */", build_block)
    if file_block:
        text = insert_before(text, "/* End PBXFileReference section */", file_block)

    PBX.write_text(text)
    print("patched", PBX)


if __name__ == "__main__":
    main()
