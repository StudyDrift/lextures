#!/usr/bin/env python3
"""Add M12.2 Wallet Swift files to the committed Xcode project."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PBX = ROOT / "Lextures.xcodeproj" / "project.pbxproj"

ENTRIES: list[tuple[str, str, str, str, bool]] = [
    ("WalletLogic.swift", "8B701C2D3E4F5061728394", "8B701D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSFeatureModelsWallet.swift", "8B702C2D3E4F5061728394", "8B702D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSAPIWallet.swift", "8B703C2D3E4F5061728394", "8B703D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("WalletView.swift", "8B704C2D3E4F5061728394", "8B704D3E4F5061728394A5", "8B70WALL5C6D7E8F901234567", False),
    ("WalletCCRDetailView.swift", "8B705C2D3E4F5061728394", "8B705D3E4F5061728394A5", "8B70WALL5C6D7E8F901234567", False),
    ("WalletCETranscriptDetailView.swift", "8B706C2D3E4F5061728394", "8B706D3E4F5061728394A5", "8B70WALL5C6D7E8F901234567", False),
    ("WalletOfficialTranscriptDetailView.swift", "8B707C2D3E4F5061728394", "8B707D3E4F5061728394A5", "8B70WALL5C6D7E8F901234567", False),
    ("WalletLogicTests.swift", "8B708C2D3E4F5061728394", "8B708D3E4F5061728394A5", "FB04F8A19314441A8AB2F273", True),
]

NEW_GROUPS = """
\t\t8B70WALL5C6D7E8F901234567 /* Wallet */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t8B704D3E4F5061728394A5 /* WalletView.swift */,
\t\t\t\t8B705D3E4F5061728394A5 /* WalletCCRDetailView.swift */,
\t\t\t\t8B706D3E4F5061728394A5 /* WalletCETranscriptDetailView.swift */,
\t\t\t\t8B707D3E4F5061728394A5 /* WalletOfficialTranscriptDetailView.swift */,
\t\t\t);
\t\t\tpath = Wallet;
\t\t\tsourceTree = "<group>";
\t\t};
"""

APP_SOURCES = "BCBCFDB66CBE4C89874A943E"
TEST_SOURCES = "CBC24580FE1340BCBAAB8A18"
FEATURES_GROUP = "88F34DDD6A74453E9D9C0FAA"
LMS_GROUP = "0192C31B7A97444D9236A8A1"
WALLET_SUBGROUP = "8B70WALL5C6D7E8F901234567 /* Wallet */"
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

    if "8B70WALL5C6D7E8F901234567 /* Wallet */" not in text:
        text = insert_before(text, "/* End PBXGroup section */", NEW_GROUPS)
        text = insert_into_children(text, FEATURES_GROUP, WALLET_SUBGROUP)

    PBX.write_text(text)
    print("patched", PBX)


if __name__ == "__main__":
    main()
