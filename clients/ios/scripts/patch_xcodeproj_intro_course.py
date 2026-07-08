#!/usr/bin/env python3
"""Add Intro Course Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("LMSFeatureModelsIntroCourse.swift", "8B721C2D3E4F5061728394", "8B721D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSAPIIntroCourse.swift", "8B722C2D3E4F5061728394", "8B722D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("IntroCourseLogic.swift", "8B723C2D3E4F5061728394", "8B723D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("IntroCourseObservability.swift", "8B724C2D3E4F5061728394", "8B724D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("IntroCourseEntryCard.swift", "8B725C2D3E4F5061728394", "8B725D3E4F5061728394A5", "8B71IC015C6D7E8F901234568", False),
    ("IntroCourseProgressRail.swift", "8B726C2D3E4F5061728394", "8B726D3E4F5061728394A5", "8B71IC015C6D7E8F901234568", False),
    ("IntroCompletionCelebrationSheet.swift", "8B727C2D3E4F5061728394", "8B727D3E4F5061728394A5", "8B71IC015C6D7E8F901234568", False),
    ("IntroCelebrationPresenter.swift", "8B728C2D3E4F5061728394", "8B728D3E4F5061728394A5", "8B71IC015C6D7E8F901234568", False),
    ("IntroCourseLogicTests.swift", "8B729C2D3E4F5061728394", "8B729D3E4F5061728394A5", "FB04F8A19314441A8AB2F273", True),
]

NEW_GROUPS = """
\t\t8B71IC015C6D7E8F901234568 /* IntroCourse */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t8B725D3E4F5061728394A5 /* IntroCourseEntryCard.swift */,
\t\t\t\t8B726D3E4F5061728394A5 /* IntroCourseProgressRail.swift */,
\t\t\t\t8B727D3E4F5061728394A5 /* IntroCompletionCelebrationSheet.swift */,
\t\t\t\t8B728D3E4F5061728394A5 /* IntroCelebrationPresenter.swift */,
\t\t\t);
\t\t\tpath = IntroCourse;
\t\t\tsourceTree = "<group>";
\t\t};
"""

APP_SOURCES = "BCBCFDB66CBE4C89874A943E"
TEST_SOURCES = "CBC24580FE1340BCBAAB8A18"
FEATURES_GROUP = "88F34DDD6A74453E9D9C0FAA"
INTRO_COURSE_SUBGROUP = "8B71IC015C6D7E8F901234568 /* IntroCourse */"


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
    if "8B71IC015C6D7E8F901234568 /* IntroCourse */" not in text:
        text = insert_before(text, "/* End PBXGroup section */", NEW_GROUPS)
        text = insert_into_children(text, FEATURES_GROUP, INTRO_COURSE_SUBGROUP)

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