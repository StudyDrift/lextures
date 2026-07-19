#!/usr/bin/env python3
"""Add MOB.8 boards-advanced Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

# name, build_id, ref_id, group_id, is_test
ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("BoardsAdvancedLogic.swift", "M860B01C2D3E4F5061728394", "M860B01D3E4F5061728394A5", "B62295116F7F4935A884363D", False),
    ("BoardsGovernanceAdminLogic.swift", "M860B02C2D3E4F5061728394", "M860B02D3E4F5061728394A5", "B62295116F7F4935A884363D", False),
    ("LMSAPIBoardTemplates.swift", "M860B03C2D3E4F5061728394", "M860B03D3E4F5061728394A5", "B62295116F7F4935A884363D", False),
    ("LMSAPIBoardExport.swift", "M860B04C2D3E4F5061728394", "M860B04D3E4F5061728394A5", "B62295116F7F4935A884363D", False),
    ("LMSAPIBoardAnalytics.swift", "M860B05C2D3E4F5061728394", "M860B05D3E4F5061728394A5", "B62295116F7F4935A884363D", False),
    ("BoardTemplatePickerView.swift", "M860B06C2D3E4F5061728394", "M860B06D3E4F5061728394A5", "5A215B5D66E342278E63F8C3", False),
    ("BoardSaveAsTemplateSheet.swift", "M860B07C2D3E4F5061728394", "M860B07D3E4F5061728394A5", "5A215B5D66E342278E63F8C3", False),
    ("BoardExportSheet.swift", "M860B08C2D3E4F5061728394", "M860B08D3E4F5061728394A5", "5A215B5D66E342278E63F8C3", False),
    ("BoardPresentModeView.swift", "M860B09C2D3E4F5061728394", "M860B09D3E4F5061728394A5", "5A215B5D66E342278E63F8C3", False),
    ("BoardAnalyticsSheet.swift", "M860B0AC2D3E4F5061728394", "M860B0AD3E4F5061728394A5", "5A215B5D66E342278E63F8C3", False),
    ("BoardsGovernanceAdminView.swift", "M860B0BC2D3E4F5061728394", "M860B0BD3E4F5061728394A5", "387083D5EF7A4F1DA444B863", False),
    ("BoardsAdvancedLogicTests.swift", "M860B0CC2D3E4F5061728394", "M860B0CD3E4F5061728394A5", "98D67F731BF5451A9402BAA2", True),
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
    print("Patched Xcode project for MOB.8 boards-advanced files.")


if __name__ == "__main__":
    main()
