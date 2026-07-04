#!/usr/bin/env python3
"""Add M7.5 Live Meetings Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("LiveMeetingsLogic.swift", "M750L01C2D3E4F5061728394", "M750L01D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("WhiteboardRenderer.swift", "M750L02C2D3E4F5061728394", "M750L02D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("MeetingListView.swift", "M750L03C2D3E4F5061728394", "M750L03D3E4F5061728394A5", "M750LGRP5C6D7E8F901234568", False),
    ("MeetingDetailView.swift", "M750L04C2D3E4F5061728394", "M750L04D3E4F5061728394A5", "M750LGRP5C6D7E8F901234568", False),
    ("WhiteboardView.swift", "M750L05C2D3E4F5061728394", "M750L05D3E4F5061728394A5", "M750LGRP5C6D7E8F901234568", False),
    ("LiveMeetingsRail.swift", "M750L06C2D3E4F5061728394", "M750L06D3E4F5061728394A5", "1920B879A116481BA2D9B736", False),
    ("LiveMeetingsLogicTests.swift", "M750L07C2D3E4F5061728394", "M750L07D3E4F5061728394A5", "D177268EB0164406B86F0376", True),
]

NEW_GROUPS = """
\t\tM750LGRP5C6D7E8F901234568 /* Live */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\tM750L03D3E4F5061728394A5 /* MeetingListView.swift */,
\t\t\t\tM750L04D3E4F5061728394A5 /* MeetingDetailView.swift */,
\t\t\t\tM750L05D3E4F5061728394A5 /* WhiteboardView.swift */,
\t\t\t);
\t\t\tpath = Live;
\t\t\tsourceTree = "<group>";
\t\t};
"""

APP_SOURCES = "884DCB9FDFFD4EE782E89BF7"
TEST_SOURCES = "D82CD7B605B94283B2415697"
FEATURES_GROUP = "9675A7A45CC240A0A4F7B883"
LMS_GROUP = "EE4734853C3443449D42D6DC"
DASHBOARD_GROUP = "1920B879A116481BA2D9B736"
LIVE_SUBGROUP = "M750LGRP5C6D7E8F901234568 /* Live */"


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
    pattern = rf"(\t\t{re.escape(phase_id)} /\* Sources \*/ = \{{\n\t\t\tisa = PBXSourcesBuildPhase;\n\t\t\tbuildActionMask = 2147483647;\n\t\t\tfiles = \()\n"
    match = re.search(pattern, text)
    if not match:
        raise SystemExit(f"sources phase not found: {phase_id}")
    if entry in text:
        return text
    return text[: match.end(1)] + "\n" + entry + "\n" + text[match.end(1) :]


def main() -> None:
    text = PBX.read_text(encoding="utf-8")

    if LIVE_SUBGROUP not in text:
        text = insert_before(text, "/* End PBXGroup section */", NEW_GROUPS)
        text = insert_into_children(text, FEATURES_GROUP, LIVE_SUBGROUP)

    for name, build_id, ref_id, group_id, _ in ENTRIES:
        if f"{ref_id} /* {name} */ = {{isa = PBXFileReference" in text:
            continue
        file_ref = f"\t\t{ref_id} /* {name} */ = {{isa = PBXFileReference; lastKnownFileType = sourcecode.swift; path = {name}; sourceTree = \"<group>\"; }};\n"
        build_file = f"\t\t{build_id} /* {name} in Sources */ = {{isa = PBXBuildFile; fileRef = {ref_id} /* {name} */; }};\n"
        text = insert_before(text, "/* End PBXBuildFile section */", build_file)
        text = insert_before(text, "/* End PBXFileReference section */", file_ref)
        text = insert_into_children(text, group_id, f"{ref_id} /* {name} */")

    for name, build_id, _, _, is_test in ENTRIES:
        phase = TEST_SOURCES if is_test else APP_SOURCES
        text = insert_into_sources(text, phase, f"{build_id} /* {name} in Sources */")

    PBX.write_text(text, encoding="utf-8")
    print("Updated project.pbxproj for Live Meetings (M7.5)")


if __name__ == "__main__":
    main()