import XCTest
@testable import Lextures

final class IntegrationsAdminLogicTests: XCTestCase {
    func testEntryRequiresFlagAndRbacPermission() {
        let off = MobilePlatformFeatures(ffMobileAdminConsole: false, ffMobileAdminSettings: false)
        XCTAssertFalse(IntegrationsAdminLogic.shouldShowEntry(
            features: off,
            permissions: [IntegrationsAdminLogic.rbacManagePermission]
        ))
        let on = MobilePlatformFeatures(ffMobileAdminConsole: false, ffMobileAdminSettings: true)
        XCTAssertFalse(IntegrationsAdminLogic.shouldShowEntry(features: on, permissions: []))
        XCTAssertTrue(IntegrationsAdminLogic.shouldShowEntry(
            features: on,
            permissions: [IntegrationsAdminLogic.rbacManagePermission]
        ))
    }

    func testSectionVisibilityByFlags() {
        var features = MobilePlatformFeatures(ffMobileAdminConsole: false, ffMobileAdminSettings: true)
        features.oerLibraryEnabled = false
        features.xapiEmissionEnabled = false

        let withoutScim = IntegrationsAdminLogic.visibleSections(features: features, scimEnabled: false)
        XCTAssertEqual(Set(withoutScim), [.lti, .cloud])

        features.oerLibraryEnabled = true
        features.xapiEmissionEnabled = true
        let withAll = IntegrationsAdminLogic.visibleSections(features: features, scimEnabled: true)
        XCTAssertEqual(Set(withAll), Set(IntegrationsAdminLogic.Section.allCases))
    }

    func testCloudStatusExcludesSecrets() {
        XCTAssertTrue(IntegrationsAdminLogic.cloudStatusExcludesSecrets(["provider", "enabled", "updatedAt"]))
        XCTAssertFalse(IntegrationsAdminLogic.cloudStatusExcludesSecrets(["provider", "clientId", "enabled"]))
        XCTAssertFalse(IntegrationsAdminLogic.cloudStatusExcludesSecrets(["apiKey"]))
    }

    func testDecodeCloudProviderOmitsSecrets() throws {
        let data = Data(#"""
        [{"provider":"google_drive","enabled":true,"clientId":"secret-id","apiKey":"secret-key","appKey":"secret-app","updatedAt":"2026-01-01T00:00:00Z"}]
        """#.utf8)
        let rows = try JSONDecoder().decode([CloudProviderStatus].self, from: data)
        XCTAssertEqual(rows.count, 1)
        XCTAssertEqual(rows[0].provider, "google_drive")
        XCTAssertTrue(rows[0].enabled)
        // Mirror keys that must not be part of the mobile model.
        let mirror = Mirror(reflecting: rows[0])
        let labels = Set(mirror.children.compactMap(\.label))
        XCTAssertFalse(labels.contains("clientId"))
        XCTAssertFalse(labels.contains("apiKey"))
        XCTAssertFalse(labels.contains("appKey"))
    }

    func testDecodeLtiAndScimStatus() throws {
        let ltiData = Data(#"""
        {
          "parentPlatforms":[{"id":"p1","name":"Canvas","clientId":"c1","platformIss":"https://canvas.example","active":true}],
          "externalTools":[{"id":"t1","name":"Tool","clientId":"c2","toolIssuer":"https://tool.example","active":false}]
        }
        """#.utf8)
        let lti = try JSONDecoder().decode(LtiRegistrationsResponse.self, from: ltiData)
        XCTAssertEqual(IntegrationsAdminLogic.ltiActiveCount(platforms: lti.parentPlatforms, tools: lti.externalTools), 1)

        let tokensData = Data(#"""
        {"tokens":[
          {"id":"1","institutionId":"i1","label":"okta","createdAt":"2026-01-01T00:00:00Z"},
          {"id":"2","institutionId":"i1","label":"old","createdAt":"2025-01-01T00:00:00Z","revokedAt":"2025-06-01T00:00:00Z"}
        ]}
        """#.utf8)
        let tokens = try JSONDecoder().decode(ScimTokensResponse.self, from: tokensData).tokens ?? []
        XCTAssertEqual(IntegrationsAdminLogic.activeTokenCount(tokens), 1)
    }

    func testApplyingToggles() {
        let platforms = [LtiParentPlatform(id: "p1", name: "A", clientId: "c", platformIss: "iss", active: true)]
        XCTAssertFalse(IntegrationsAdminLogic.applyingLtiPlatformActive(platforms, id: "p1", active: false)[0].active)

        let tools = [LtiExternalTool(id: "t1", name: "T", clientId: "c", toolIssuer: "iss", active: false)]
        XCTAssertTrue(IntegrationsAdminLogic.applyingLtiToolActive(tools, id: "t1", active: true)[0].active)

        let cloud = [CloudProviderStatus(provider: "dropbox", enabled: true, updatedAt: nil)]
        XCTAssertFalse(IntegrationsAdminLogic.applyingCloudEnabled(cloud, provider: "dropbox", enabled: false)[0].enabled)

        let lrs = [LrsEndpointStatus(
            id: "e1", label: "L", endpointUrl: "https://lrs", authType: "basic",
            username: nil, enabled: true, hasPassword: true, hasOauthSecret: false, updatedAt: nil
        )]
        XCTAssertFalse(IntegrationsAdminLogic.applyingLrsEnabled(lrs, id: "e1", enabled: false)[0].enabled)

        let oer = [OerProviderStatus(provider: "merlot", enabled: false, updatedAt: nil)]
        XCTAssertTrue(IntegrationsAdminLogic.applyingOerEnabled(oer, provider: "merlot", enabled: true)[0].enabled)
    }

    func testWebPaths() {
        XCTAssertEqual(IntegrationsAdminLogic.Section.lti.webPath, "/settings/lti-tools")
        XCTAssertEqual(IntegrationsAdminLogic.Section.scim.webPath, "/settings/scim-provisioning")
        XCTAssertEqual(IntegrationsAdminLogic.Section.cloud.webPath, "/settings/cloud-providers")
        XCTAssertEqual(IntegrationsAdminLogic.Section.lrs.webPath, "/settings/lrs-integrations")
        XCTAssertEqual(IntegrationsAdminLogic.Section.oer.webPath, "/settings/oer-providers")
    }
}
