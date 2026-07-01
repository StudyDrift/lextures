#!/usr/bin/env python3
"""Add M5.1/M7.1/M7.2 Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("AssignmentLogic.swift", "0187C23C0AE54C1686409126", "DC0110C79EBC4E5B8402D50C", "EE4734853C3443449D42D6DC", False),
    ("DiscussionLogic.swift", "AAB7BEFB34404CEBB2360F45", "F3782F8285144A8282414028", "EE4734853C3443449D42D6DC", False),
    ("LMSAPIDiscussions.swift", "ECF3ECFE25CF4115A991111D", "080A92D8F9454E8A84D5C126", "EE4734853C3443449D42D6DC", False),
    ("LMSAPITutor.swift", "D592E24AC90241009951E220", "7034DAA0F439465CA60F01D6", "EE4734853C3443449D42D6DC", False),
    ("LMSFeatureModelsDiscussions.swift", "6B601EB5D5A0438D93F41073", "B5F23D4CB9B34608BDFF52C0", "EE4734853C3443449D42D6DC", False),
    ("LMSFeatureModelsTutor.swift", "08580A1065284E9FB955AE8D", "4BC9EE68C03F499385880A6E", "EE4734853C3443449D42D6DC", False),
    ("TutorLogic.swift", "E04DE250969449BAAD797198", "8BCDAC98B9E74C4DBFE626EF", "EE4734853C3443449D42D6DC", False),
    ("TutorStreamClient.swift", "3DD0BCEFF5E042B195C0FCDC", "0767A07BDABC4E57B5E531D6", "EE4734853C3443449D42D6DC", False),
    ("AssignmentDetailView.swift", "35BDEA0579904FCDB9C76DE5", "1CA4E942C00845B195DE0F22", "71025D7F411C42B8977AC4C2", False),
    ("AssignmentDraftStore.swift", "BC0CEE4E54464483ADEED9C5", "18DE37EE0E0E4F76B1EF3A04", "71025D7F411C42B8977AC4C2", False),
    ("AttachmentUploader.swift", "5053584B744240C692E235A0", "01770DCC545B460ABCF0C4AC", "71025D7F411C42B8977AC4C2", False),
    ("CourseDiscussionsSection.swift", "38BC1801903D47128558B1A0", "0740CD15B21C4D4A864FC6C5", "81F2E4188BBA4822A198C056", False),
    ("DiscussionThreadView.swift", "8FA81A74B405478ABD3BB327", "824D36B30B0A4F5F9B098CF6", "81F2E4188BBA4822A198C056", False),
    ("DiscussionsListView.swift", "3C53B179DFBC416985A91927", "638B158DE352445A93F6D0EF", "81F2E4188BBA4822A198C056", False),
    ("PostComposerView.swift", "2A03F4B98D3446D58654F3C1", "6221383657FA4F6A92F19886", "81F2E4188BBA4822A198C056", False),
    ("TutorChatModel.swift", "23299442725443BA8B9F163E", "12C3B4E710C3428C912C59FD", "A2F38050F87E47D8841DA2DD", False),
    ("TutorChatView.swift", "EA2656D1FF0444B3B75D3714", "DDFE68D874DC4BFA97FD7D23", "A2F38050F87E47D8841DA2DD", False),
    ("TutorFlowLayout.swift", "031CDB9867A246CB947C2520", "FF7F4E0EB4204B85ACFA1CE7", "A2F38050F87E47D8841DA2DD", False),
    ("TutorLauncher.swift", "CF494658471641B8B712C9D0", "D59BBA6832654A9397F03221", "A2F38050F87E47D8841DA2DD", False),
    ("AssignmentLogicTests.swift", "2D3A43490592470881DD625D", "31D7F54C6AE74B4A881E6668", "D177268EB0164406B86F0376", True),
    ("DiscussionLogicTests.swift", "29C3063A926C4569BA929FFE", "D409F0EB287D4F239C250117", "D177268EB0164406B86F0376", True),
    ("TutorLogicTests.swift", "FA4D4D6AD50D45EF87DBBBE1", "F79285C9BC784AEFA1C40F21", "D177268EB0164406B86F0376", True),
]

NEW_GROUPS = """
\t\t71025D7F411C42B8977AC4C2 /* Assignments */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t1CA4E942C00845B195DE0F22 /* AssignmentDetailView.swift */,
\t\t\t\t18DE37EE0E0E4F76B1EF3A04 /* AssignmentDraftStore.swift */,
\t\t\t\t01770DCC545B460ABCF0C4AC /* AttachmentUploader.swift */,
\t\t\t);
\t\t\tpath = Assignments;
\t\t\tsourceTree = "<group>";
\t\t};
\t\t81F2E4188BBA4822A198C056 /* Discussions */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t0740CD15B21C4D4A864FC6C5 /* CourseDiscussionsSection.swift */,
\t\t\t\t824D36B30B0A4F5F9B098CF6 /* DiscussionThreadView.swift */,
\t\t\t\t638B158DE352445A93F6D0EF /* DiscussionsListView.swift */,
\t\t\t\t6221383657FA4F6A92F19886 /* PostComposerView.swift */,
\t\t\t);
\t\t\tpath = Discussions;
\t\t\tsourceTree = "<group>";
\t\t};
\t\tA2F38050F87E47D8841DA2DD /* Tutor */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t12C3B4E710C3428C912C59FD /* TutorChatModel.swift */,
\t\t\t\tDDFE68D874DC4BFA97FD7D23 /* TutorChatView.swift */,
\t\t\t\tFF7F4E0EB4204B85ACFA1CE7 /* TutorFlowLayout.swift */,
\t\t\t\tD59BBA6832654A9397F03221 /* TutorLauncher.swift */,
\t\t\t);
\t\t\tpath = Tutor;
\t\t\tsourceTree = "<group>";
\t\t};
"""

APP_SOURCES = "884DCB9FDFFD4EE782E89BF7"
TEST_SOURCES = "D82CD7B605B94283B2415697"
FEATURES_GROUP = "9675A7A45CC240A0A4F7B883"
FEATURE_SUBGROUPS = (
    "71025D7F411C42B8977AC4C2 /* Assignments */",
    "81F2E4188BBA4822A198C056 /* Discussions */",
    "A2F38050F87E47D8841DA2DD /* Tutor */",
)
LEAF_FEATURE_GROUPS = {
    "71025D7F411C42B8977AC4C2",
    "81F2E4188BBA4822A198C056",
    "A2F38050F87E47D8841DA2DD",
}


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
        path = f'"{name}"' if "+" in name else name
        build_lines.append(
            f"\t\t{build_id} /* {name} in Sources */ = {{isa = PBXBuildFile; fileRef = {ref_id} /* {name} */; }};"
        )
        ref_lines.append(
            f"\t\t{ref_id} /* {name} */ = {{isa = PBXFileReference; lastKnownFileType = sourcecode.swift; path = {path}; sourceTree = \"<group>\"; }};"
        )

    text = insert_before(text, "/* End PBXBuildFile section */", "\n".join(build_lines) + "\n")
    text = insert_before(text, "/* End PBXFileReference section */", "\n".join(ref_lines) + "\n")

    for subgroup in FEATURE_SUBGROUPS:
        text = insert_into_children(text, FEATURES_GROUP, subgroup)

    if "71025D7F411C42B8977AC4C2 /* Assignments */ = {" not in text:
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