#!/usr/bin/env python3
"""Add M10.1 Parent Portal Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("ParentLogic.swift", "8B601C2D3E4F5061728394", "8B601D3E4F5061728394A5", "D2384791A1E3420B9476C89D", False),
    ("ConferenceLogic.swift", "8B602C2D3E4F5061728394", "8B602D3E4F5061728394A5", "D2384791A1E3420B9476C89D", False),
    ("LMSFeatureModelsParent.swift", "8B603C2D3E4F5061728394", "8B603D3E4F5061728394A5", "D2384791A1E3420B9476C89D", False),
    ("LMSAPIParent.swift", "8B604C2D3E4F5061728394", "8B604D3E4F5061728394A5", "D2384791A1E3420B9476C89D", False),
    ("ParentDashboardView.swift", "8B605C2D3E4F5061728394", "8B605D3E4F5061728394A5", "8B60PAR5C6D7E8F901234567", False),
    ("ParentDashboardSections.swift", "8B60CC2D3E4F5061728394", "8B60CD3E4F5061728394A5", "8B60PAR5C6D7E8F901234567", False),
    ("ParentGradesDetailView.swift", "8B606C2D3E4F5061728394", "8B606D3E4F5061728394A5", "8B60PAR5C6D7E8F901234567", False),
    ("ParentAttendanceDetailView.swift", "8B607C2D3E4F5061728394", "8B607D3E4F5061728394A5", "8B60PAR5C6D7E8F901234567", False),
    ("ParentNotificationPrefsView.swift", "8B608C2D3E4F5061728394", "8B608D3E4F5061728394A5", "8B60PAR5C6D7E8F901234567", False),
    ("ConferenceBookingView.swift", "8B609C2D3E4F5061728394", "8B609D3E4F5061728394A5", "8B60PAR5C6D7E8F901234567", False),
    ("ParentLogicTests.swift", "8B60AC2D3E4F5061728394", "8B60AD3E4F5061728394A5", "C76648F9447549EBA4F966C1", True),
    ("ConferenceLogicTests.swift", "8B60BC2D3E4F5061728394", "8B60BD3E4F5061728394A5", "C76648F9447549EBA4F966C1", True),
]

NEW_GROUP = """
\t\t8B60PAR5C6D7E8F901234567 /* Parent */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t8B605D3E4F5061728394A5 /* ParentDashboardView.swift */,
\t\t\t\t8B60CD3E4F5061728394A5 /* ParentDashboardSections.swift */,
\t\t\t\t8B606D3E4F5061728394A5 /* ParentGradesDetailView.swift */,
\t\t\t\t8B607D3E4F5061728394A5 /* ParentAttendanceDetailView.swift */,
\t\t\t\t8B608D3E4F5061728394A5 /* ParentNotificationPrefsView.swift */,
\t\t\t\t8B609D3E4F5061728394A5 /* ConferenceBookingView.swift */,
\t\t\t);
\t\t\tpath = Parent;
\t\t\tsourceTree = "<group>";
\t\t};
"""

APP_SOURCES = "9AA69D924B67495B8D274F50"
TEST_SOURCES = "D6AFC42863AF4064A8A587A3"
FEATURES_GROUP = "660E34061B9749BA97FA683C"
PARENT_SUBGROUP = "8B60PAR5C6D7E8F901234567 /* Parent */"
LEAF_FEATURE_GROUPS = {"8B60PAR5C6D7E8F901234567"}


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
    entry = f"\t\t\t\t{build_line}"
    pattern = rf"(\t\t{re.escape(phase_id)} /\* Sources \*/ = \{{\n\t\t\tisa = PBXSourcesBuildPhase;\n\t\t\tbuildActionMask = 2147483647;\n\t\t\trunOnlyForDeploymentPostprocessing = 0;\n\t\t\tfiles = \()\n"
    match = re.search(pattern, text)
    if not match:
        raise SystemExit(f"sources phase not found: {phase_id}")
    if entry in text[match.start() : match.start() + 12000]:
        return text
    return text[: match.end()] + f"\n{entry}" + text[match.end() :]


def main() -> None:
    text = PBX.read_text()
    missing = [e for e in ENTRIES if e[1] not in text]
    if not missing:
        print("Nothing to patch")
        return

    build_lines = []
    ref_lines = []
    for name, build_id, ref_id, _, _ in missing:
        build_lines.append(
            f"\t\t{build_id} /* {name} in Sources */ = {{isa = PBXBuildFile; fileRef = {ref_id} /* {name} */; }};"
        )
        ref_lines.append(
            f"\t\t{ref_id} /* {name} */ = {{isa = PBXFileReference; lastKnownFileType = sourcecode.swift; path = {name}; sourceTree = \"<group>\"; }};"
        )

    text = insert_before(text, "/* End PBXBuildFile section */", "\n".join(build_lines) + "\n")
    text = insert_before(text, "/* End PBXFileReference section */", "\n".join(ref_lines) + "\n")

    text = insert_into_children(text, FEATURES_GROUP, PARENT_SUBGROUP)

    if "8B60PAR5C6D7E8F901234567 /* Parent */ = {" not in text:
        text = insert_before(text, "/* End PBXGroup section */", NEW_GROUP)

    for name, build_id, ref_id, group_id, is_test in missing:
        if group_id not in LEAF_FEATURE_GROUPS:
            text = insert_into_children(text, group_id, f"{ref_id} /* {name} */")
        phase_id = TEST_SOURCES if is_test else APP_SOURCES
        text = insert_into_sources(text, phase_id, f"{build_id} /* {name} in Sources */,")

    PBX.write_text(text)
    print(f"Patched {len(missing)} files into {PBX}")


if __name__ == "__main__":
    main()
