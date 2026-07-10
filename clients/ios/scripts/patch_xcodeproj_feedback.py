#!/usr/bin/env python3
"""Add FB3 Share Feedback Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("LMSFeatureModelsFeedback.swift", "8BFB01C2D3E4F5061728394", "8BFB01D3E4F5061728394A5", "D7AC7435149B4B17A073AD6F", False),
    ("LMSAPIFeedback.swift", "8BFB02C2D3E4F5061728394", "8BFB02D3E4F5061728394A5", "D7AC7435149B4B17A073AD6F", False),
    ("FeedbackLogic.swift", "8BFB03C2D3E4F5061728394", "8BFB03D3E4F5061728394A5", "D7AC7435149B4B17A073AD6F", False),
    ("ShareFeedbackView.swift", "8BFB04C2D3E4F5061728394", "8BFB04D3E4F5061728394A5", "8BFB07FB015C6D7E8F901234567", False),
    ("ShareFeedbackEntryCard.swift", "8BFB05C2D3E4F5061728394", "8BFB05D3E4F5061728394A5", "029927E4F879434FA36100DE", False),
    ("FeedbackLogicTests.swift", "8BFB06C2D3E4F5061728394", "8BFB06D3E4F5061728394A5", "8D3F9FB222A34107B54468F2", True),
]

NEW_GROUPS = """
\t\t8BFB07FB015C6D7E8F901234567 /* Feedback */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t8BFB04D3E4F5061728394A5 /* ShareFeedbackView.swift */,
\t\t\t);
\t\t\tpath = Feedback;
\t\t\tsourceTree = "<group>";
\t\t};
"""

APP_SOURCES = "B3991B6DD6F9495DB8B2870B"
TEST_SOURCES = "693EA1D9511A4AB8B7F10A82"
FEATURES_GROUP = "5FD4367FBF4B4225BB2007EF"
FEEDBACK_SUBGROUP = "8BFB07FB015C6D7E8F901234567 /* Feedback */"


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
    if "8BFB07FB015C6D7E8F901234567 /* Feedback */" not in text:
        text = insert_before(text, "/* End PBXGroup section */", NEW_GROUPS)
        text = insert_into_children(text, FEATURES_GROUP, FEEDBACK_SUBGROUP)

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
