#!/usr/bin/env python3
"""Add M13.1 Course Settings Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("CourseSettingsLogic.swift", "8B801C2D3E4F5061728394", "8B801D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSFeatureModelsCourseSettings.swift", "8B802C2D3E4F5061728394", "8B802D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSAPICourseSettings.swift", "8B803C2D3E4F5061728394", "8B803D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("UnsavedChangesBanner.swift", "8B804C2D3E4F5061728394", "8B804D3E4F5061728394A5", "551D819CB69643C594BC15DC", False),
    ("CourseSettingsHostView.swift", "8B805C2D3E4F5061728394", "8B805D3E4F5061728394A5", "8B80SETT5C6D7E8F901234567", False),
    ("CourseGeneralSettingsView.swift", "8B806C2D3E4F5061728394", "8B806D3E4F5061728394A5", "8B80SETT5C6D7E8F901234567", False),
    ("CourseHeroImageEditor.swift", "8B807C2D3E4F5061728394", "8B807D3E4F5061728394A5", "8B80SETT5C6D7E8F901234567", False),
    ("CourseSettingsLogicTests.swift", "8B808C2D3E4F5061728394", "8B808D3E4F5061728394A5", "FB04F8A19314441A8AB2F273", True),
]

NEW_GROUPS = """
\t\t8B80SETT5C6D7E8F901234567 /* Settings */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t8B805D3E4F5061728394A5 /* CourseSettingsHostView.swift */,
\t\t\t\t8B806D3E4F5061728394A5 /* CourseGeneralSettingsView.swift */,
\t\t\t\t8B807D3E4F5061728394A5 /* CourseHeroImageEditor.swift */,
\t\t\t);
\t\t\tpath = Settings;
\t\t\tsourceTree = "<group>";
\t\t};
"""

APP_SOURCES = "BCBCFDB66CBE4C89874A943E"
TEST_SOURCES = "CBC24580FE1340BCBAAB8A18"
COURSES_GROUP = "FD10AEF33AA94E6FB2220EF0"
LMS_GROUP = "0192C31B7A97444D9236A8A1"
DESIGN_GROUP = "551D819CB69643C594BC15DC"
SETTINGS_SUBGROUP = "8B80SETT5C6D7E8F901234567 /* Settings */"
TESTS_GROUP = "FB04F8A19314441A8AB2F273"


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
    for key in ("files = (", "children = ("):
        marker = f"\t\t\t{key}\n"
        if marker in block:
            updated = block.replace(marker, f"{marker}{child_entry}\n", 1)
            return text.replace(block, updated)
    raise SystemExit(f"no files/children marker in group: {group_id}")


def main() -> None:
    text = PBX.read_text()
    for name, build_id, file_id, group_id, is_test in ENTRIES:
        if f"{file_id} /* {name} */" in text:
            continue
        build_line = f"\t\t{build_id} /* {name} in Sources */ = {{isa = PBXBuildFile; fileRef = {file_id} /* {name} */; }};\n"
        file_line = f"\t\t{file_id} /* {name} */ = {{isa = PBXFileReference; lastKnownFileType = sourcecode.swift; path = {name}; sourceTree = \"<group>\"; }};\n"
        text = insert_before(text, "/* End PBXBuildFile section */", build_line)
        text = insert_before(text, "/* End PBXFileReference section */", file_line)
        sources = TEST_SOURCES if is_test else APP_SOURCES
        text = insert_into_children(text, sources, f"{build_id} /* {name} in Sources */")
        if is_test:
            text = insert_into_children(text, TESTS_GROUP, f"{file_id} /* {name} */")
        elif group_id == LMS_GROUP:
            text = insert_into_children(text, LMS_GROUP, f"{file_id} /* {name} */")
        elif group_id == DESIGN_GROUP:
            text = insert_into_children(text, DESIGN_GROUP, f"{file_id} /* {name} */")

    if "8B80SETT5C6D7E8F901234567 /* Settings */" not in text:
        text = insert_before(text, "/* End PBXGroup section */", NEW_GROUPS)
        text = insert_into_children(text, COURSES_GROUP, SETTINGS_SUBGROUP)

    PBX.write_text(text)
    print("patched", PBX)


if __name__ == "__main__":
    main()
