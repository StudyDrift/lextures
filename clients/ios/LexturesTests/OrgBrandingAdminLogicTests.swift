import XCTest
@testable import Lextures

final class OrgBrandingAdminLogicTests: XCTestCase {
    func testAdminSettingsEnabled() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(OrgBrandingAdminLogic.adminSettingsEnabled(features))
        features.ffMobileAdminSettings = true
        XCTAssertTrue(OrgBrandingAdminLogic.adminSettingsEnabled(features))
    }

    func testCanManage() {
        XCTAssertFalse(OrgBrandingAdminLogic.canManage(permissions: []))
        XCTAssertTrue(
            OrgBrandingAdminLogic.canManage(
                permissions: [OrgBrandingAdminLogic.rbacManagePermission]
            )
        )
        XCTAssertTrue(
            OrgBrandingAdminLogic.canManage(
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
        XCTAssertTrue(OrgBrandingAdminLogic.isValidHexColor("#abc"))
        XCTAssertFalse(OrgBrandingAdminLogic.isValidHexColor("4F46E5"))
        XCTAssertFalse(OrgBrandingAdminLogic.isValidHexColor("#GG0000"))
        XCTAssertFalse(OrgBrandingAdminLogic.isValidHexColor(""))
    }

    func testNormalizeHexColor() {
        XCTAssertEqual(
            OrgBrandingAdminLogic.normalizeHexColor("#abc", fallback: "#000000"),
            "#AABBCC"
        )
        XCTAssertEqual(
            OrgBrandingAdminLogic.normalizeHexColor("bad", fallback: "#4F46E5"),
            "#4F46E5"
        )
    }

    func testSecretPlaceholderHandling() {
        XCTAssertNil(OrgBrandingAdminLogic.byokKeyForSave(""))
        XCTAssertNil(OrgBrandingAdminLogic.byokKeyForSave(OrgBrandingAdminLogic.secretPlaceholder))
        XCTAssertEqual(OrgBrandingAdminLogic.byokKeyForSave(" sk-live-xyz "), "sk-live-xyz")
        XCTAssertEqual(
            OrgBrandingAdminLogic.displaySecretField(byokConfigured: true),
            OrgBrandingAdminLogic.secretPlaceholder
        )
        XCTAssertEqual(OrgBrandingAdminLogic.displaySecretField(byokConfigured: false), "")
        XCTAssertTrue(OrgBrandingAdminLogic.isSecretPlaceholder(OrgBrandingAdminLogic.secretPlaceholder))
    }

    func testAllowedModelsParsing() {
        XCTAssertNil(OrgBrandingAdminLogic.parseAllowedModels("  \n  "))
        XCTAssertEqual(
            OrgBrandingAdminLogic.parseAllowedModels("claude-3-5-sonnet\ngpt-4o, gemini-1.5-pro"),
            ["claude-3-5-sonnet", "gpt-4o", "gemini-1.5-pro"]
        )
        XCTAssertEqual(
            OrgBrandingAdminLogic.allowedModelsText(["a", "b"]),
            "a\nb"
        )
    }

    func testFeaturesEnabledPayloadDefaultsTrue() {
        let payload = OrgBrandingAdminLogic.featuresEnabledPayload([:])
        for item in OrgBrandingAdminLogic.featureKeys {
            XCTAssertEqual(payload[item.key], true)
        }
        let disabled = OrgBrandingAdminLogic.featuresEnabledPayload(["ai_tutor": false])
        XCTAssertEqual(disabled["ai_tutor"], false)
        XCTAssertEqual(disabled["translation"], true)
    }

    func testAiProviderPutBodyOmitsPlaceholderSecret() {
        let body = OrgBrandingAdminLogic.aiProviderPutBody(
            provider: "openai",
            modelAlias: "gpt-4o",
            fallbackProvider: "",
            byokKey: OrgBrandingAdminLogic.secretPlaceholder
        )
        XCTAssertEqual(body.provider, "openai")
        XCTAssertEqual(body.modelAlias, "gpt-4o")
        XCTAssertNil(body.fallbackProvider)
        XCTAssertNil(body.byokApiKey)
    }

    func testContrastAgainstWhite() {
        let white = OrgBrandingAdminLogic.contrastRatioAgainstWhite("#FFFFFF")
        XCTAssertNotNil(white)
        XCTAssertEqual(white!, 1.0, accuracy: 0.01)
        let black = OrgBrandingAdminLogic.contrastRatioAgainstWhite("#000000")
        XCTAssertNotNil(black)
        XCTAssertGreaterThan(black!, 20)
        XCTAssertTrue(
            OrgBrandingAdminLogic.hasContrastWarning(
                primaryColor: "#EEEEEE",
                serverWarning: false,
                serverRatio: nil
            )
        )
        XCTAssertFalse(
            OrgBrandingAdminLogic.hasContrastWarning(
                primaryColor: "#111111",
                serverWarning: false,
                serverRatio: 12
            )
        )
    }

    func testBrandingPutBodyTrimsEmail() {
        let body = OrgBrandingAdminLogic.brandingPutBody(
            logoUrl: nil,
            faviconUrl: nil,
            primaryColor: "#4f46e5",
            secondaryColor: "#7c3aed",
            customEmailDisplayName: "  District  "
        )
        XCTAssertEqual(body.primaryColor, "#4F46E5")
        XCTAssertEqual(body.customEmailDisplayName, "District")
        XCTAssertNil(body.customDomain)
    }

    func testWebPath() {
        XCTAssertEqual(OrgBrandingAdminLogic.webBrandingPath(), "/settings/org-branding")
    }
}
