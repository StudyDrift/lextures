#!/usr/bin/env python3
"""Add M12.1 Portfolio Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("PortfolioLogic.swift", "8B601C2D3E4F5061728394", "8B601D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("LMSFeatureModelsEportfolio.swift", "8B602C2D3E4F5061728394", "8B602D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("LMSAPIEportfolio.swift", "8B603C2D3E4F5061728394", "8B603D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("PortfolioView.swift", "8B604C2D3E4F5061728394", "8B604D3E4F5061728394A5", "8B60PORT5C6D7E8F901234567", False),
    ("PortfolioDetailView.swift", "8B605C2D3E4F5061728394", "8B605D3E4F5061728394A5", "8B60PORT5C6D7E8F901234567", False),
    ("ArtifactEditorView.swift", "8B606C2D3E4F5061728394", "8B606D3E4F5061728394A5", "8B60PORT5C6D7E8F901234567", False),
    ("ArtifactDetailView.swift", "8B607C2D3E4F5061728394", "8B607D3E4F5061728394A5", "8B60PORT5C6D7E8F901234567", False),
    ("PortfolioArtifactUploader.swift", "8B608C2D3E4F5061728394", "8B608D3E4F5061728394A5", "8B60PORT5C6D7E8F901234567", False),
    ("PortfolioLogicTests.swift", "8B609C2D3E4F5061728394", "8B609D3E4F5061728394A5", "D177268EB0164406B86F0376", True),
]

NEW_GROUPS = """
\t\t8B60PORT5C6D7E8F901234567 /* Portfolio */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t8B604D3E4F5061728394A5 /* PortfolioView.swift */,
\t\t\t\t8B605D3E4F5061728394A5 /* PortfolioDetailView.swift */,
\t\t\t\t8B606D3E4F5061728394A5 /* ArtifactEditorView.swift */,
\t\t\t\t8B607D3E4F5061728394A5 /* ArtifactDetailView.swift */,
\t\t\t\t8B608D3E4F5061728394A5 /* PortfolioArtifactUploader.swift */,
\t\t\t);
\t\t\tpath = Portfolio;
\t\t\tsourceTree = "<group>";
\t\t};
"""

APP_SOURCES = "884DCB9FDFFD4EE782E89BF7"
TEST_SOURCES = "D82CD7B605B94283B2415697"
FEATURES_GROUP = "9675A7A45CC240A0A4F7B883"
PORTFOLIO_SUBGROUP = "8B60PORT5C6D7E8F901234567 /* Portfolio */"
LEAF_FEATURE_GROUPS = {"8B60PORT5C6D7E8F901234567"}


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


def main() -> None:
    text = PBX.read_text()
    for name, build_id, file_id, group_id, is_test in ENTRIES:
        if f"{file_id} /* {name} */" in text:
            continue
        text += f"\n\t\t{build_id} /* {name} in Sources */ = {{isa = PBXBuildFile; fileRef = {file_id} /* {name} */; }};"
        text += f"\n\t\t{file_id} /* {name} */ = {{isa = PBXFileReference; lastKnownFileType = sourcecode.swift; path = {name}; sourceTree = \"<group>\"; }};"
        sources = TEST_SOURCES if is_test else APP_SOURCES
        text = insert_into_children(text, sources, f"{build_id} /* {name} in Sources */")
        if not is_test and group_id == "EE4734853C3443449D42D6DC":
            text = insert_into_children(text, group_id, f"{file_id} /* {name} */")
        elif is_test:
            text = insert_into_children(text, "D177268EB0164406B86F0376", f"{file_id} /* {name} */")

    if "8B60PORT5C6D7E8F901234567 /* Portfolio */" not in text:
        text = insert_before(text, "/* End PBXGroup section */", NEW_GROUPS)
        text = insert_into_children(text, FEATURES_GROUP, PORTFOLIO_SUBGROUP)

    PBX.write_text(text)
    print("patched", PBX)


if __name__ == "__main__":
    main()