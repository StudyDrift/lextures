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
    ("CourseImportExportLogic.swift", "8B809C2D3E4F5061728394", "8B809D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSAPICourseImportExport.swift", "8B80AC2D3E4F5061728394", "8B80AD3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("CourseImportExportView.swift", "8B80BC2D3E4F5061728394", "8B80BD3E4F5061728394A5", "8B80SETT5C6D7E8F901234567", False),
    ("CourseImportExportLogicTests.swift", "8B80CC2D3E4F5061728394", "8B80CD3E4F5061728394A5", "FB04F8A19314441A8AB2F273", True),
    ("CourseBlueprintLogic.swift", "8B80DC2D3E4F5061728394", "8B80DD3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSAPICourseBlueprint.swift", "8B80EC2D3E4F5061728394", "8B80ED3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("CourseBlueprintSettingsView.swift", "8B80FC2D3E4F5061728394", "8B80FD3E4F5061728394A5", "8B80SETT5C6D7E8F901234567", False),
    ("CourseBlueprintLogicTests.swift", "8B810C2D3E4F5061728394", "8B810D3E4F5061728394A5", "FB04F8A19314441A8AB2F273", True),
    ("CourseArchivedContentLogic.swift", "8B811C2D3E4F5061728394", "8B811D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSAPICourseArchivedContent.swift", "8B812C2D3E4F5061728394", "8B812D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("CourseArchivedContentView.swift", "8B813C2D3E4F5061728394", "8B813D3E4F5061728394A5", "8B80SETT5C6D7E8F901234567", False),
    ("CourseArchivedContentLogicTests.swift", "8B814C2D3E4F5061728394", "8B814D3E4F5061728394A5", "FB04F8A19314441A8AB2F273", True),
    ("CourseFeaturesLogic.swift", "8B815C2D3E4F5061728394", "8B815D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSAPICourseFeatures.swift", "8B816C2D3E4F5061728394", "8B816D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("CourseFeaturesSettingsView.swift", "8B817C2D3E4F5061728394", "8B817D3E4F5061728394A5", "8B80SETT5C6D7E8F901234567", False),
    ("CourseFeaturesLogicTests.swift", "8B818C2D3E4F5061728394", "8B818D3E4F5061728394A5", "FB04F8A19314441A8AB2F273", True),
    ("CourseGradingLogic.swift", "8B819C2D3E4F5061728394", "8B819D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSFeatureModelsCourseGrading.swift", "8B81AC2D3E4F5061728394", "8B81AD3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSAPICourseGrading.swift", "8B81BC2D3E4F5061728394", "8B81BD3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("CourseGradingSettingsView.swift", "8B81CC2D3E4F5061728394", "8B81CD3E4F5061728394A5", "8B80SETT5C6D7E8F901234567", False),
    ("CourseGradingLogicTests.swift", "8B81DC2D3E4F5061728394", "8B81DD3E4F5061728394A5", "FB04F8A19314441A8AB2F273", True),
    ("CourseOutcomesLogic.swift", "8B81EC2D3E4F5061728394", "8B81ED3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSFeatureModelsCourseOutcomes.swift", "8B81FC2D3E4F5061728394", "8B81FD3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSAPIOutcomes.swift", "8B825C2D3E4F5061728394", "8B825D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("CourseOutcomesSettingsView.swift", "8B826C2D3E4F5061728394", "8B826D3E4F5061728394A5", "8B80SETT5C6D7E8F901234567", False),
    ("CourseOutcomesLogicTests.swift", "8B827C2D3E4F5061728394", "8B827D3E4F5061728394A5", "FB04F8A19314441A8AB2F273", True),
    ("CourseGradingAgentsLogic.swift", "8B828C2D3E4F5061728394", "8B828D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSFeatureModelsCourseGradingAgents.swift", "8B829C2D3E4F5061728394", "8B829D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSAPICourseGradingAgents.swift", "8B82AC2D3E4F5061728394", "8B82AD3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("CourseGradingAgentsView.swift", "8B82BC2D3E4F5061728394", "8B82BD3E4F5061728394A5", "8B80SETT5C6D7E8F901234567", False),
    ("CourseGradingAgentsLogicTests.swift", "8B82CC2D3E4F5061728394", "8B82CD3E4F5061728394A5", "FB04F8A19314441A8AB2F273", True),
    ("CoursePlagiarismLogic.swift", "8B82DC2D3E4F5061728394", "8B82DD3E4F5061728394A5", "C4481A1E74894DAAB6174AC6", False),
    ("LMSFeatureModelsCoursePlagiarism.swift", "8B82EC2D3E4F5061728394", "8B82ED3E4F5061728394A5", "C4481A1E74894DAAB6174AC6", False),
    ("LMSAPICoursePlagiarism.swift", "8B82FC2D3E4F5061728394", "8B82FD3E4F5061728394A5", "C4481A1E74894DAAB6174AC6", False),
    ("CoursePlagiarismSettingsView.swift", "8B830C2D3E4F5061728394", "8B830D3E4F5061728394A5", "6C450D4856A44427B5DCBF7A", False),
    ("CoursePlagiarismLogicTests.swift", "8B831C2D3E4F5061728394", "8B831D3E4F5061728394A5", "FB04F8A19314441A8AB2F273", True),
    ("CourseAccessibilityReviewLogic.swift", "8B832C2D3E4F5061728394", "8B832D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSFeatureModelsCourseAccessibility.swift", "8B833C2D3E4F5061728394", "8B833D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("LMSAPICourseAccessibility.swift", "8B834C2D3E4F5061728394", "8B834D3E4F5061728394A5", "0192C31B7A97444D9236A8A1", False),
    ("CourseAccessibilityReviewView.swift", "8B835C2D3E4F5061728394", "8B835D3E4F5061728394A5", "8B80SETT5C6D7E8F901234567", False),
    ("CourseAccessibilityReviewLogicTests.swift", "8B836C2D3E4F5061728394", "8B836D3E4F5061728394A5", "FB04F8A19314441A8AB2F273", True),
]

NEW_GROUPS = """
\t\t8B80SETT5C6D7E8F901234567 /* Settings */ = {
\t\t\tisa = PBXGroup;
\t\t\tchildren = (
\t\t\t\t8B805D3E4F5061728394A5 /* CourseSettingsHostView.swift */,
\t\t\t\t8B806D3E4F5061728394A5 /* CourseGeneralSettingsView.swift */,
\t\t\t\t8B807D3E4F5061728394A5 /* CourseHeroImageEditor.swift */,
\t\t\t\t8B80BD3E4F5061728394A5 /* CourseImportExportView.swift */,
\t\t\t\t8B80FD3E4F5061728394A5 /* CourseBlueprintSettingsView.swift */,
\t\t\t\t8B813D3E4F5061728394A5 /* CourseArchivedContentView.swift */,
\t\t\t\t8B817D3E4F5061728394A5 /* CourseFeaturesSettingsView.swift */,
\t\t\t\t8B81CD3E4F5061728394A5 /* CourseGradingSettingsView.swift */,
\t\t\t\t8B826D3E4F5061728394A5 /* CourseOutcomesSettingsView.swift */,
\t\t\t\t8B82BD3E4F5061728394A5 /* CourseGradingAgentsView.swift */,
\t\t\t\t8B830D3E4F5061728394A5 /* CoursePlagiarismSettingsView.swift */,
\t\t\t\t8B835D3E4F5061728394A5 /* CourseAccessibilityReviewView.swift */,
\t\t\t);
\t\t\tpath = Settings;
\t\t\tsourceTree = "<group>";
\t\t};
"""

APP_SOURCES = "D727EF5963B5497EB142F2E0"
TEST_SOURCES = "157E2CDA7CCA47FDBFE4FFFD"
COURSES_GROUP = "FD10AEF33AA94E6FB2220EF0"
LMS_GROUP = "C4481A1E74894DAAB6174AC6"
DESIGN_GROUP = "551D819CB69643C594BC15DC"
SETTINGS_SUBGROUP = "6C450D4856A44427B5DCBF7A"
SETTINGS_SUBGROUP_LABEL = "6C450D4856A44427B5DCBF7A /* Settings */"
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
        elif group_id == SETTINGS_SUBGROUP:
            text = insert_into_children(text, SETTINGS_SUBGROUP, f"{file_id} /* {name} */")

    if "8B80SETT5C6D7E8F901234567 /* Settings */" not in text:
        text = insert_before(text, "/* End PBXGroup section */", NEW_GROUPS)
        text = insert_into_children(text, COURSES_GROUP, SETTINGS_SUBGROUP_LABEL)

    PBX.write_text(text)
    print("patched", PBX)


if __name__ == "__main__":
    main()
