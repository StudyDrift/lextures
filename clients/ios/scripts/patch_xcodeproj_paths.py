#!/usr/bin/env python3
"""Add M8.2 Paths Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("PathsLogic.swift", "8B201C2D3E4F5061728394", "8B201D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("LMSAPIPaths.swift", "8B202C2D3E4F5061728394", "8B202D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("LMSFeatureModelsPaths.swift", "8B203C2D3E4F5061728394", "8B203D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("MyPathsView.swift", "8B204C2D3E4F5061728394", "8B204D3E4F5061728394A5", "8B2PATH5C6D7E8F901234567", False),
    ("PathRunnerView.swift", "8B205C2D3E4F5061728394", "8B205D3E4F5061728394A5", "8B2PATH5C6D7E8F901234567", False),
    ("PathsCatalogView.swift", "8B206C2D3E4F5061728394", "8B206D3E4F5061728394A5", "8B2PATH5C6D7E8F901234567", False),
    ("PathLandingView.swift", "8B207C2D3E4F5061728394", "8B207D3E4F5061728394A5", "8B2PATH5C6D7E8F901234567", False),
    ("DashboardStudySection.swift", "8B208C2D3E4F5061728394", "8B208D3E4F5061728394A5", "8B2PATH5C6D7E8F901234567", False),
    ("PathsLogicTests.swift", "8B209C2D3E4F5061728394", "8B209D3E4F5061728394A5", "D177268EB0164406B86F0376", True),
]

NEW_GROUP = """
\t\t8B2PATH5C6D7E8F901234567 /* Paths */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t8B204D3E4F5061728394A5 /* MyPathsView.swift */,
\t\t\t\t8B205D3E4F5061728394A5 /* PathRunnerView.swift */,
\t\t\t\t8B206D3E4F5061728394A5 /* PathsCatalogView.swift */,
\t\t\t\t8B207D3E4F5061728394A5 /* PathLandingView.swift */,
\t\t\t\t8B208D3E4F5061728394A5 /* DashboardStudySection.swift */,
\t\t\t);
\t\t\tpath = Paths;
\t\t\tsourceTree = "<group>";
\t\t};
"""

APP_SOURCES = "884DCB9FDFFD4EE782E89BF7"
TEST_SOURCES = "D82CD7B605B94283B2415697"
FEATURES_GROUP = "9675A7A45CC240A0A4F7B883"
PATHS_SUBGROUP = "8B2PATH5C6D7E8F901234567 /* Paths */"
LEAF_FEATURE_GROUPS = {"8B2PATH5C6D7E8F901234567"}


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
    pattern = rf"(\t\t{re.escape(phase_id)} /\* Sources \*/ = \{{\n\t\t\tisa = PBXSourcesBuildPhase;\n\t\t\tbuildActionMask = 2147483647;\n\t\t\tfiles = \()\n"
    match = re.search(pattern, text)
    if not match:
        raise SystemExit(f"sources phase not found: {phase_id}")
    if entry in text[match.start() : match.start() + 8000]:
        return text
    return text[: match.end()] + f"\n{entry}" + text[match.end() :]


def main() -> None:
    text = PBX.read_text()
    missing = [e for e in ENTRIES if e[2] not in text]
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

    text = insert_into_children(text, FEATURES_GROUP, PATHS_SUBGROUP)

    if "8B2PATH5C6D7E8F901234567 /* Paths */ = {" not in text:
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