#!/usr/bin/env python3
"""Add M9.3 Credentials/Gamification Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("CredentialsLogic.swift", "8B501C2D3E4F5061728394", "8B501D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("GamificationLogic.swift", "8B502C2D3E4F5061728394", "8B502D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("CourseReviewLogic.swift", "8B503C2D3E4F5061728394", "8B503D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("LMSFeatureModelsCredentials.swift", "8B504C2D3E4F5061728394", "8B504D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("LMSFeatureModelsGamification.swift", "8B505C2D3E4F5061728394", "8B505D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("LMSFeatureModelsCourseReviews.swift", "8B506C2D3E4F5061728394", "8B506D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("LMSAPICredentials.swift", "8B507C2D3E4F5061728394", "8B507D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("LMSAPIGamification.swift", "8B508C2D3E4F5061728394", "8B508D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("LMSAPICourseReviews.swift", "8B509C2D3E4F5061728394", "8B509D3E4F5061728394A5", "EE4734853C3443449D42D6DC", False),
    ("CredentialsView.swift", "8B50AC2D3E4F5061728394", "8B50AD3E4F5061728394A5", "8B50CRED5C6D7E8F901234567", False),
    ("CredentialDetailView.swift", "8B50BC2D3E4F5061728394", "8B50BD3E4F5061728394A5", "8B50CRED5C6D7E8F901234567", False),
    ("GamificationView.swift", "8B50CC2D3E4F5061728394", "8B50CD3E4F5061728394A5", "8B50GAM5C6D7E8F9012345678", False),
    ("ReviewComposer.swift", "8B50DC2D3E4F5061728394", "8B50DD3E4F5061728394A5", "8B39CAT5C6D7E8F901234567", False),
    ("CredentialsLogicTests.swift", "8B50EC2D3E4F5061728394", "8B50ED3E4F5061728394A5", "D177268EB0164406B86F0376", True),
    ("GamificationLogicTests.swift", "8B50FC2D3E4F5061728394", "8B50FD3E4F5061728394A5", "D177268EB0164406B86F0376", True),
    ("CourseReviewLogicTests.swift", "8B510C2D3E4F5061728394", "8B510D3E4F5061728394A5", "D177268EB0164406B86F0376", True),
]

NEW_GROUPS = """
\t\t8B50CRED5C6D7E8F901234567 /* Credentials */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t8B50AD3E4F5061728394A5 /* CredentialsView.swift */,
\t\t\t\t8B50BD3E4F5061728394A5 /* CredentialDetailView.swift */,
\t\t\t);
\t\t\tpath = Credentials;
\t\t\tsourceTree = "<group>";
\t\t};
\t\t8B50GAM5C6D7E8F9012345678 /* Gamification */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t8B50CD3E4F5061728394A5 /* GamificationView.swift */,
\t\t\t);
\t\t\tpath = Gamification;
\t\t\tsourceTree = "<group>";
\t\t};
"""

APP_SOURCES = "884DCB9FDFFD4EE782E89BF7"
TEST_SOURCES = "D82CD7B605B94283B2415697"
FEATURES_GROUP = "9675A7A45CC240A0A4F7B883"
CATALOG_GROUP = "8B39CAT5C6D7E8F901234567"
CREDENTIALS_SUBGROUP = "8B50CRED5C6D7E8F901234567 /* Credentials */"
GAMIFICATION_SUBGROUP = "8B50GAM5C6D7E8F9012345678 /* Gamification */"
LEAF_FEATURE_GROUPS = {"8B50CRED5C6D7E8F901234567", "8B50GAM5C6D7E8F9012345678", "8B39CAT5C6D7E8F901234567"}


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

    text = insert_into_children(text, FEATURES_GROUP, CREDENTIALS_SUBGROUP)
    text = insert_into_children(text, FEATURES_GROUP, GAMIFICATION_SUBGROUP)

    if "8B50CRED5C6D7E8F901234567 /* Credentials */ = {" not in text:
        text = insert_before(text, "/* End PBXGroup section */", NEW_GROUPS)

    for name, build_id, ref_id, group_id, is_test in missing:
        if group_id not in LEAF_FEATURE_GROUPS:
            text = insert_into_children(text, group_id, f"{ref_id} /* {name} */")
        phase_id = TEST_SOURCES if is_test else APP_SOURCES
        text = insert_into_sources(text, phase_id, f"{build_id} /* {name} in Sources */,")

    PBX.write_text(text)
    print(f"Patched {len(missing)} files into {PBX}")


if __name__ == "__main__":
    main()