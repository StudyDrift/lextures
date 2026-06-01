#!/usr/bin/env python3
"""Generate Lextures.xcodeproj for the native iOS app."""

from __future__ import annotations

import os
import uuid
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
APP_NAME = "Lextures"
BUNDLE_ID = "com.lextures.ios"
DEPLOYMENT_TARGET = "17.0"


def gen_id() -> str:
    return uuid.uuid4().hex[:24].upper()


def collect_swift_files() -> list[str]:
    app_root = ROOT / APP_NAME
    paths: list[str] = []
    for path in sorted(app_root.rglob("*.swift")):
        if "Resources" in path.parts:
            continue
        paths.append(path.relative_to(ROOT).as_posix())
    return paths


def main() -> None:
    swift_files = collect_swift_files()

    project_id = gen_id()
    target_id = gen_id()
    sources_phase = gen_id()
    resources_phase = gen_id()
    frameworks_phase = gen_id()
    project_config_list = gen_id()
    target_config_list = gen_id()
    debug_config = gen_id()
    release_config = gen_id()
    target_debug = gen_id()
    target_release = gen_id()
    main_group = gen_id()
    products_group = gen_id()
    app_product = gen_id()
    app_group = gen_id()
    assets_ref = gen_id()
    info_plist_ref = gen_id()
    assets_build = gen_id()

    file_refs: dict[str, str] = {}
    build_files: dict[str, str] = {}
    for path in swift_files:
        file_refs[path] = gen_id()
        build_files[path] = gen_id()

    lines: list[str] = []
    w = lines.append

    w("// !$*UTF8*$!")
    w("{")
    w("\tarchiveVersion = 1;")
    w("\tclasses = {")
    w("\t};")
    w("\tobjectVersion = 56;")
    w("\tobjects = {")

    w("\n/* Begin PBXBuildFile section */")
    for path in swift_files:
        base = os.path.basename(path)
        w(
            f"\t\t{build_files[path]} /* {base} in Sources */ = {{isa = PBXBuildFile; fileRef = {file_refs[path]} /* {base} */; }};"
        )
    w(
        f"\t\t{assets_build} /* Assets.xcassets in Resources */ = {{isa = PBXBuildFile; fileRef = {assets_ref} /* Assets.xcassets */; }};"
    )
    w("/* End PBXBuildFile section */")

    w("\n/* Begin PBXFileReference section */")
    w(
        f"\t\t{app_product} /* {APP_NAME}.app */ = {{isa = PBXFileReference; explicitFileType = wrapper.application; includeInIndex = 0; path = {APP_NAME}.app; sourceTree = BUILT_PRODUCTS_DIR; }};"
    )
    for path in swift_files:
        base = os.path.basename(path)
        w(
            f"\t\t{file_refs[path]} /* {base} */ = {{isa = PBXFileReference; lastKnownFileType = sourcecode.swift; path = {base}; sourceTree = \"<group>\"; }};"
        )
    w(
        f"\t\t{assets_ref} /* Assets.xcassets */ = {{isa = PBXFileReference; lastKnownFileType = folder.assetcatalog; path = Assets.xcassets; sourceTree = \"<group>\"; }};"
    )
    w(
        f"\t\t{info_plist_ref} /* Info.plist */ = {{isa = PBXFileReference; lastKnownFileType = text.plist.xml; path = Info.plist; sourceTree = \"<group>\"; }};"
    )
    w("/* End PBXFileReference section */")

    w("\n/* Begin PBXFrameworksBuildPhase section */")
    w(f"\t\t{frameworks_phase} /* Frameworks */ = {{")
    w("\t\t\tisa = PBXFrameworksBuildPhase;")
    w("\t\t\tbuildActionMask = 2147483647;")
    w("\t\t\trunOnlyForDeploymentPostprocessing = 0;")
    w("\t\t\tfiles = (")
    w("\t\t\t);")
    w("\t\t};")
    w("/* End PBXFrameworksBuildPhase section */")

    # Groups
    dir_groups: dict[str, str] = {APP_NAME: app_group}
    resources_group = gen_id()
    dir_groups[f"{APP_NAME}/Resources"] = resources_group

    for path in swift_files:
        directory = os.path.dirname(path)
        parts = directory.split("/")
        for i in range(2, len(parts) + 1):
            sub = "/".join(parts[:i])
            if sub not in dir_groups:
                dir_groups[sub] = gen_id()

    w("\n/* Begin PBXGroup section */")
    for directory in sorted(dir_groups, key=lambda d: (d.count("/"), d)):
        gid = dir_groups[directory]
        name = os.path.basename(directory)
        children: list[str] = []

        if directory == f"{APP_NAME}/Resources":
            children = [
                f"{assets_ref} /* Assets.xcassets */",
                f"{info_plist_ref} /* Info.plist */",
            ]
        else:
            prefix = directory + "/"
            for sub, sub_gid in dir_groups.items():
                if sub.startswith(prefix) and sub.count("/") == directory.count("/") + 1:
                    children.append(f"{sub_gid} /* {os.path.basename(sub)} */")
            for path in swift_files:
                if os.path.dirname(path) == directory:
                    children.append(f"{file_refs[path]} /* {os.path.basename(path)} */")

        child_lines = ",\n\t\t\t\t".join(children)
        path_attr = f'path = {name};' if directory != APP_NAME else f"path = {APP_NAME};"
        w(f"\t\t{gid} /* {name} */ = {{")
        w("\t\t\tisa = PBXGroup;")
        w("\t\t\tchildren = (")
        w(f"\t\t\t\t{child_lines},")
        w("\t\t\t);")
        w(f"\t\t\t{path_attr}")
        w('\t\t\tsourceTree = "<group>";')
        w("\t\t};")

    w(f"\t\t{products_group} /* Products */ = {{")
    w("\t\t\tisa = PBXGroup;")
    w("\t\t\tchildren = (")
    w(f"\t\t\t\t{app_product} /* {APP_NAME}.app */,")
    w("\t\t\t);")
    w("\t\t\tname = Products;")
    w('\t\t\tsourceTree = "<group>";')
    w("\t\t};")

    w(f"\t\t{main_group} = {{")
    w("\t\t\tisa = PBXGroup;")
    w("\t\t\tchildren = (")
    w(f"\t\t\t\t{app_group} /* {APP_NAME} */,")
    w(f"\t\t\t\t{products_group} /* Products */,")
    w("\t\t\t);")
    w('\t\t\tsourceTree = "<group>";')
    w("\t\t};")
    w("/* End PBXGroup section */")

    w("\n/* Begin PBXNativeTarget section */")
    w(f"\t\t{target_id} /* {APP_NAME} */ = {{")
    w("\t\t\tisa = PBXNativeTarget;")
    w(f'\t\t\tbuildConfigurationList = {target_config_list} /* Build configuration list for PBXNativeTarget "{APP_NAME}" */;')
    w("\t\t\tbuildPhases = (")
    w(f"\t\t\t\t{sources_phase} /* Sources */,")
    w(f"\t\t\t\t{frameworks_phase} /* Frameworks */,")
    w(f"\t\t\t\t{resources_phase} /* Resources */,")
    w("\t\t\t);")
    w("\t\t\tbuildRules = (")
    w("\t\t\t);")
    w("\t\t\tdependencies = (")
    w("\t\t\t);")
    w(f'\t\t\tname = "{APP_NAME}";')
    w(f"\t\t\tproductName = {APP_NAME};")
    w(f"\t\t\tproductReference = {app_product} /* {APP_NAME}.app */;")
    w('\t\t\tproductType = "com.apple.product-type.application";')
    w("\t\t};")
    w("/* End PBXNativeTarget section */")

    w("\n/* Begin PBXProject section */")
    w(f"\t\t{project_id} /* Project object */ = {{")
    w("\t\t\tisa = PBXProject;")
    w(f"\t\t\tattributes = {{")
    w(f"\t\t\t\tBuildIndependentTargetsInParallel = 1;")
    w(f"\t\t\t\tLastSwiftUpdateCheck = 1600;")
    w(f"\t\t\t\tLastUpgradeCheck = 1600;")
    w(f"\t\t\t\tTargetAttributes = {{")
    w(f"\t\t\t\t\t{target_id} = {{")
    w(f"\t\t\t\t\t\tCreatedOnToolsVersion = 16.0;")
    w(f"\t\t\t\t\t}};")
    w(f"\t\t\t\t}};")
    w(f"\t\t\t}};")
    w(f"\t\t\tbuildConfigurationList = {project_config_list} /* Build configuration list for PBXProject \"{APP_NAME}\" */;")
    w("\t\t\tcompatibilityVersion = \"Xcode 15.0\";")
    w("\t\t\tdevelopmentRegion = en;")
    w("\t\t\thasScannedForEncodings = 0;")
    w("\t\t\tknownRegions = (")
    w("\t\t\t\ten,")
    w("\t\t\t\tBase,")
    w("\t\t\t);")
    w(f"\t\t\tmainGroup = {main_group};")
    w(f"\t\t\tproductRefGroup = {products_group} /* Products */;")
    w("\t\t\tprojectDirPath = \"\";")
    w("\t\t\tprojectRoot = \"\";")
    w("\t\t\ttargets = (")
    w(f"\t\t\t\t{target_id} /* {APP_NAME} */,")
    w("\t\t\t);")
    w("\t\t};")
    w("/* End PBXProject section */")

    w("\n/* Begin PBXResourcesBuildPhase section */")
    w(f"\t\t{resources_phase} /* Resources */ = {{")
    w("\t\t\tisa = PBXResourcesBuildPhase;")
    w("\t\t\tbuildActionMask = 2147483647;")
    w("\t\t\trunOnlyForDeploymentPostprocessing = 0;")
    w("\t\t\tfiles = (")
    w(f"\t\t\t\t{assets_build} /* Assets.xcassets in Resources */,")
    w("\t\t\t);")
    w("\t\t};")
    w("/* End PBXResourcesBuildPhase section */")

    w("\n/* Begin PBXSourcesBuildPhase section */")
    w(f"\t\t{sources_phase} /* Sources */ = {{")
    w("\t\t\tisa = PBXSourcesBuildPhase;")
    w("\t\t\tbuildActionMask = 2147483647;")
    w("\t\t\trunOnlyForDeploymentPostprocessing = 0;")
    w("\t\t\tfiles = (")
    for path in swift_files:
        base = os.path.basename(path)
        w(f"\t\t\t\t{build_files[path]} /* {base} in Sources */,")
    w("\t\t\t);")
    w("\t\t};")
    w("/* End PBXSourcesBuildPhase section */")

    w("\n/* Begin XCBuildConfiguration section */")
    for cfg_id, name in ((debug_config, "Debug"), (release_config, "Release")):
        w(f"\t\t{cfg_id} /* {name} */ = {{")
        w("\t\t\tisa = XCBuildConfiguration;")
        w("\t\t\tbuildSettings = {")
        w(f'\t\t\t\tALWAYS_SEARCH_USER_PATHS = NO;')
        w(f'\t\t\t\tCLANG_ENABLE_MODULES = YES;')
        w(f'\t\t\t\tCOPY_PHASE_STRIP = NO;')
        w(f'\t\t\t\tDEBUG_INFORMATION_FORMAT = {"dwarf" if name == "Debug" else "dwarf-with-dsym"};')
        w(f'\t\t\t\tENABLE_USER_SCRIPT_SANDBOXING = YES;')
        w(f'\t\t\t\tGCC_DYNAMIC_NO_PIC = NO;')
        w(f'\t\t\t\tGCC_OPTIMIZATION_LEVEL = {"0" if name == "Debug" else "s"};')
        w(f'\t\t\t\tIPHONEOS_DEPLOYMENT_TARGET = {DEPLOYMENT_TARGET};')
        w(f'\t\t\t\tMTL_ENABLE_DEBUG_INFO = {"INCLUDE_SOURCE" if name == "Debug" else "NO"};')
        w(f'\t\t\t\tONLY_ACTIVE_ARCH = {"YES" if name == "Debug" else "NO"};')
        w(f'\t\t\t\tSDKROOT = iphoneos;')
        if name == "Debug":
            w(f'\t\t\t\tSWIFT_ACTIVE_COMPILATION_CONDITIONS = DEBUG;')
        w(f'\t\t\t\tSWIFT_OPTIMIZATION_LEVEL = {"-Onone" if name == "Debug" else "-O"};')
        w("\t\t\t};")
        w(f'\t\t\tname = {name};')
        w("\t\t};")

    for cfg_id, name in ((target_debug, "Debug"), (target_release, "Release")):
        w(f"\t\t{cfg_id} /* {name} */ = {{")
        w("\t\t\tisa = XCBuildConfiguration;")
        w("\t\t\tbuildSettings = {")
        w(f'\t\t\t\tAPI_BASE_URL = "http://127.0.0.1:8080";')
        w(f'\t\t\t\tASSETCATALOG_COMPILER_APPICON_NAME = AppIcon;')
        w(f'\t\t\t\tASSETCATALOG_COMPILER_GLOBAL_ACCENT_COLOR_NAME = AccentColor;')
        w(f'\t\t\t\tCODE_SIGN_STYLE = Automatic;')
        w(f'\t\t\t\tCURRENT_PROJECT_VERSION = 1;')
        w(f'\t\t\t\tDEVELOPMENT_TEAM = "";')
        w(f'\t\t\t\tENABLE_PREVIEWS = YES;')
        w(f'\t\t\t\tGENERATE_INFOPLIST_FILE = NO;')
        w(f'\t\t\t\tINFOPLIST_FILE = {APP_NAME}/Resources/Info.plist;')
        w(f'\t\t\t\tLD_RUNPATH_SEARCH_PATHS = (')
        w(f'\t\t\t\t\t"$(inherited)",')
        w(f'\t\t\t\t\t"@executable_path/Frameworks",')
        w(f'\t\t\t\t);')
        w(f'\t\t\t\tMARKETING_VERSION = 0.1.0;')
        w(f'\t\t\t\tPRODUCT_BUNDLE_IDENTIFIER = {BUNDLE_ID};')
        w(f'\t\t\t\tPRODUCT_NAME = "$(TARGET_NAME)";')
        w(f'\t\t\t\tSUPPORTED_PLATFORMS = "iphoneos iphonesimulator";')
        w(f'\t\t\t\tSUPPORTS_MACCATALYST = NO;')
        w(f'\t\t\t\tSWIFT_EMIT_LOC_STRINGS = YES;')
        w(f'\t\t\t\tSWIFT_VERSION = 5.0;')
        w(f'\t\t\t\tTARGETED_DEVICE_FAMILY = 1;')
        w("\t\t\t};")
        w(f'\t\t\tname = {name};')
        w("\t\t};")
    w("/* End XCBuildConfiguration section */")

    w("\n/* Begin XCConfigurationList section */")
    w(f"\t\t{project_config_list} /* Build configuration list for PBXProject \"{APP_NAME}\" */ = {{")
    w("\t\t\tisa = XCConfigurationList;")
    w("\t\t\tbuildConfigurations = (")
    w(f"\t\t\t\t{debug_config} /* Debug */,")
    w(f"\t\t\t\t{release_config} /* Release */,")
    w("\t\t\t);")
    w("\t\t\tdefaultConfigurationIsVisible = 0;")
    w("\t\t\tdefaultConfigurationName = Release;")
    w("\t\t};")

    w(f"\t\t{target_config_list} /* Build configuration list for PBXNativeTarget \"{APP_NAME}\" */ = {{")
    w("\t\t\tisa = XCConfigurationList;")
    w("\t\t\tbuildConfigurations = (")
    w(f"\t\t\t\t{target_debug} /* Debug */,")
    w(f"\t\t\t\t{target_release} /* Release */,")
    w("\t\t\t);")
    w("\t\t\tdefaultConfigurationIsVisible = 0;")
    w("\t\t\tdefaultConfigurationName = Release;")
    w("\t\t};")
    w("/* End XCConfigurationList section */")

    w("\t};")
    w(f"\trootObject = {project_id} /* Project object */;")
    w("}")

    xcodeproj = ROOT / f"{APP_NAME}.xcodeproj"
    pbxproj_path = xcodeproj / "project.pbxproj"
    pbxproj_path.parent.mkdir(parents=True, exist_ok=True)
    pbxproj_path.write_text("\n".join(lines) + "\n", encoding="utf-8")

    scheme_dir = xcodeproj / "xcshareddata" / "xcschemes"
    scheme_dir.mkdir(parents=True, exist_ok=True)
    scheme_path = scheme_dir / f"{APP_NAME}.xcscheme"
    scheme_path.write_text(
        f"""<?xml version="1.0" encoding="UTF-8"?>
<Scheme
   LastUpgradeVersion = "1600"
   version = "1.7">
   <BuildAction
      parallelizeBuildables = "YES"
      buildImplicitDependencies = "YES">
      <BuildActionEntries>
         <BuildActionEntry
            buildForTesting = "YES"
            buildForRunning = "YES"
            buildForProfiling = "YES"
            buildForArchiving = "YES"
            buildForAnalyzing = "YES">
            <BuildableReference
               BuildableIdentifier = "primary"
               BlueprintIdentifier = "{target_id}"
               BuildableName = "{APP_NAME}.app"
               BlueprintName = "{APP_NAME}"
               ReferencedContainer = "container:{APP_NAME}.xcodeproj">
            </BuildableReference>
         </BuildActionEntry>
      </BuildActionEntries>
   </BuildAction>
   <LaunchAction
      buildConfiguration = "Debug"
      selectedDebuggerIdentifier = "Xcode.DebuggerFoundation.Debugger.LLDB"
      selectedLauncherIdentifier = "Xcode.DebuggerFoundation.Launcher.LLDB"
      launchStyle = "0"
      useCustomWorkingDirectory = "NO"
      ignoresPersistentStateOnLaunch = "NO"
      debugDocumentVersioning = "YES"
      debugServiceExtension = "internal"
      allowLocationSimulation = "YES">
      <BuildableProductRunnable
         runnableDebuggingMode = "0">
         <BuildableReference
            BuildableIdentifier = "primary"
            BlueprintIdentifier = "{target_id}"
            BuildableName = "{APP_NAME}.app"
            BlueprintName = "{APP_NAME}"
            ReferencedContainer = "container:{APP_NAME}.xcodeproj">
         </BuildableReference>
      </BuildableProductRunnable>
      <EnvironmentVariables>
         <EnvironmentVariable
            key = "API_BASE_URL"
            value = "http://127.0.0.1:8080"
            isEnabled = "YES">
         </EnvironmentVariable>
      </EnvironmentVariables>
   </LaunchAction>
</Scheme>
""",
        encoding="utf-8",
    )

    print(f"Wrote {pbxproj_path}")
    print(f"Wrote {scheme_path}")


if __name__ == "__main__":
    main()
