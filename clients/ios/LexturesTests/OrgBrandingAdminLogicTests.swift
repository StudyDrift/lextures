import XCTest
@testable import Lextures

final class OrgBrandingAdminLogicTests: XCTestCase {
    func testAdminSettingsFlag() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(OrgBrandingAdminLogic.adminSettingsEnabled(features))
        features.ffMobileAdminSettings = true
        XCTAssertTrue(OrgBrandingAdminLogic.adminSettingsEnabled(features))
    }

    func testPermissions() {
        XCTAssertFalse(OrgBrandingAdminLogic.canManageOrgBranding(permissions: []))
        XCTAssertTrue(
            OrgBrandingAdminLogic.canManageOrgBranding(
                permissions: [OrgBrandingAdminLogic.rbacManagePermission]
            )
        )
        XCTAssertTrue(
            OrgBrandingAdminLogic.canManageOrgBranding(
                permissions: [OrgBrandingAdminLogic.orgUnitsAdminPermission]
            )
        )
    }

    func testShouldShowEntry() {
        let features = MobilePlatformFeatures(ffMobileAdminSettings: true)
        XCTAssertFalse(
            OrgBrandingAdminLogic.shouldShowEntry(features: features, permissions: [])
        )
        XCTAssertTrue(
            OrgBrandingAdminLogic.shouldShowEntry(
                features: features,
                permissions: [OrgBrandingAdminLogic.orgUnitsAdminPermission]
            )
        )
    }

    func testHexColorValidation() {
        XCTAssertTrue(OrgBrandingAdminLogic.isValidHexColor("#4F46E5"))
        XCTAssertEqual(OrgBrandingAdminLogic.normalizedHexColor("#4f46e5"), "#4F46E5")
        XCTAssertFalse(OrgBrandingAdminLogic.isValidHexColor("blue"))
        XCTAssertFalse(OrgBrandingAdminLogic.isValidHexColor("#FFF"))
    }

    func testContrastWarning() {
        XCTAssertTrue(OrgBrandingAdminLogic.showsContrastWarning(
            primaryColor: "#FFFF00",
            serverWarning: false,
            serverRatio: nil
        ))
        XCTAssertFalse(OrgBrandingAdminLogic.showsContrastWarning(
            primaryColor: "#111827",
            serverWarning: false,
            serverRatio: nil
        ))
    }

    func testSecretPlaceholderHandling() {
        XCTAssertFalse(
            OrgBrandingAdminLogic.shouldSendByokKey(OrgBrandingAdminLogic.platformSecretPlaceholder)
        )
        XCTAssertTrue(OrgBrandingAdminLogic.shouldSendByokKey("sk-live-secret"))
        let request = OrgBrandingAdminLogic.buildAiProviderSaveRequest(
            provider: "openrouter",
            modelAlias: "claude-3-5-sonnet",
            fallbackProvider: "",
            byokKey: OrgBrandingAdminLogic.platformSecretPlaceholder
        )
        XCTAssertNil(request.byokApiKey)
        let withKey = OrgBrandingAdminLogic.buildAiProviderSaveRequest(
            provider: "openrouter",
            modelAlias: "claude-3-5-sonnet",
            fallbackProvider: "",
            byokKey: "sk-live-secret"
        )
        XCTAssertEqual(withKey.byokApiKey, "sk-live-secret")
    }

    func testParseAllowedModels() {
        XCTAssertEqual(
            OrgBrandingAdminLogic.parseAllowedModels("gpt-4o\nclaude-3-5-sonnet"),
            ["gpt-4o", "claude-3-5-sonnet"]
        )
        XCTAssertEqual(
            OrgBrandingAdminLogic.parseAllowedModels("gpt-4o, claude-3-5-sonnet"),
            ["gpt-4o", "claude-3-5-sonnet"]
        )
    }

    func testBuildAiConfigSaveRequestDefaultsEnabled() {
        let request = OrgBrandingAdminLogic.buildAiConfigSaveRequest(
            enabled: ["ai_tutor": false],
            allowedModelsText: ""
        )
        XCTAssertEqual(request.featuresEnabled["ai_tutor"], false)
        XCTAssertEqual(request.featuresEnabled["translation"], true)
        XCTAssertNil(request.allowedModels)
    }
}
