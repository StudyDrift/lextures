#!/usr/bin/env python3
"""Add LP10 Learner Profile Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("LMSFeatureModelsLearnerProfile.swift", "8B711C2D3E4F5061728394", "8B711D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSAPILearnerProfile.swift", "8B712C2D3E4F5061728394", "8B712D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LearnerProfileLogic.swift", "8B713C2D3E4F5061728394", "8B713D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LearnerProfileView.swift", "8B714C2D3E4F5061728394", "8B714D3E4F5061728394A5", "8B71LP015C6D7E8F901234567", False),
    ("LearnerProfileEntryCard.swift", "8B715C2D3E4F5061728394", "8B715D3E4F5061728394A5", "4CC73DD0F8E3490486591CC7", False),
    ("LearnerProfileLogicTests.swift", "8B716C2D3E4F5061728394", "8B716D3E4F5061728394A5", "FB04F8A19314441A8AB2F273", True),
]

NEW_GROUPS = """
\t\t8B71LP015C6D7E8F901234567 /* LearnerProfile */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t8B714D3E4F5061728394A5 /* LearnerProfileView.swift */,
\t\t\t);
\t\t\tpath = LearnerProfile;
\t\t\tsourceTree = "<group>";
\t\t};
"""

APP_SOURCES = "BCBCFDB66CBE4C89874A943E"
TEST_SOURCES = "CBC24580FE1340BCBAAB8A18"
FEATURES_GROUP = "88F34DDD6A74453E9D9C0FAA"
LEARNER_PROFILE_SUBGROUP = "8B71LP015C6D7E8F901234567 /* LearnerProfile */"


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
    if "8B71LP015C6D7E8F901234567 /* LearnerProfile */" not in text:
        text = insert_before(text, "/* End PBXGroup section */", NEW_GROUPS)
        text = insert_into_children(text, FEATURES_GROUP, LEARNER_PROFILE_SUBGROUP)

    build_block = ""
    file_block = ""
    for name, build_id, file_id, group_id, is_test in ENTRIES:
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