import XCTest
@testable import Lextures

final class TranscriptsAdvisingAdminLogicTests: XCTestCase {
    func testEntryRequiresFlagPermissionAndFeature() {
        var offAdmin = MobilePlatformFeatures(ffMobileAdminSettings: false)
        offAdmin.ffTranscripts = true
        XCTAssertFalse(TranscriptsAdvisingAdminLogic.shouldShowEntry(
            features: offAdmin,
            permissions: [TranscriptsAdvisingAdminLogic.rbacManagePermission]
        ))

        var on = MobilePlatformFeatures(ffMobileAdminSettings: true)
        on.ffTranscripts = false
        on.ffAdvisingIntegration = false
        XCTAssertFalse(TranscriptsAdvisingAdminLogic.shouldShowEntry(
            features: on,
            permissions: [TranscriptsAdvisingAdminLogic.rbacManagePermission]
        ))

        on.ffTranscripts = true
        XCTAssertFalse(TranscriptsAdvisingAdminLogic.shouldShowEntry(features: on, permissions: []))
        XCTAssertTrue(TranscriptsAdvisingAdminLogic.shouldShowEntry(
            features: on,
            permissions: [TranscriptsAdvisingAdminLogic.rbacManagePermission]
        ))
    }

    func testSectionVisibilityByFlags() {
        var features = MobilePlatformFeatures(ffMobileAdminSettings: true)
        features.ffTranscripts = false
        features.ffAdvisingIntegration = false
        XCTAssertTrue(TranscriptsAdvisingAdminLogic.visibleSections(features: features).isEmpty)

        features.ffTranscripts = true
        XCTAssertEqual(TranscriptsAdvisingAdminLogic.visibleSections(features: features), [.transcripts])

        features.ffAdvisingIntegration = true
        XCTAssertEqual(
            Set(TranscriptsAdvisingAdminLogic.visibleSections(features: features)),
            Set(TranscriptsAdvisingAdminLogic.Section.allCases)
        )
    }

    func testSubViewGating() {
        var features = MobilePlatformFeatures(ffMobileAdminSettings: true)
        features.ffTranscripts = true
        features.ffAdvisingIntegration = false
        let perms = [TranscriptsAdvisingAdminLogic.rbacManagePermission]
        XCTAssertTrue(TranscriptsAdvisingAdminLogic.canViewTranscripts(features: features, permissions: perms))
        XCTAssertFalse(TranscriptsAdvisingAdminLogic.canViewAdvising(features: features, permissions: perms))

        features.ffTranscripts = false
        features.ffAdvisingIntegration = true
        XCTAssertFalse(TranscriptsAdvisingAdminLogic.canViewTranscripts(features: features, permissions: perms))
        XCTAssertTrue(TranscriptsAdvisingAdminLogic.canViewAdvising(features: features, permissions: perms))
    }

    func testTranscriptsSavePayloadOmitsPlaceholderSecret() throws {
        let keep = TranscriptsAdvisingAdminLogic.buildTranscriptsSaveRequest(
            webhookUrl: " https://sis.example.edu/hook ",
            webhookSecret: TranscriptsAdvisingAdminLogic.secretPlaceholder,
            pickupInstructions: " Room 101 "
        )
        XCTAssertEqual(keep.webhookUrl, "https://sis.example.edu/hook")
        XCTAssertNil(keep.webhookSecret)
        XCTAssertEqual(keep.pickupInstructions, "Room 101")

        let update = TranscriptsAdvisingAdminLogic.buildTranscriptsSaveRequest(
            webhookUrl: "https://sis.example.edu/hook",
            webhookSecret: " new-secret ",
            pickupInstructions: ""
        )
        XCTAssertEqual(update.webhookSecret, "new-secret")

        let data = try JSONEncoder().encode(keep)
        let obj = try JSONSerialization.jsonObject(with: data) as? [String: Any]
        XCTAssertNil(obj?["webhookSecret"])
        XCTAssertEqual(obj?["webhookUrl"] as? String, "https://sis.example.edu/hook")
    }

    func testWebhookUrlValidation() {
        XCTAssertTrue(TranscriptsAdvisingAdminLogic.isValidHttpUrl("https://example.edu/hook"))
        XCTAssertTrue(TranscriptsAdvisingAdminLogic.isValidHttpUrl("http://localhost:8080/hook"))
        XCTAssertFalse(TranscriptsAdvisingAdminLogic.isValidHttpUrl(""))
        XCTAssertFalse(TranscriptsAdvisingAdminLogic.isValidHttpUrl("ftp://example.edu"))
        XCTAssertFalse(TranscriptsAdvisingAdminLogic.isValidHttpUrl("not-a-url"))
        XCTAssertTrue(TranscriptsAdvisingAdminLogic.isTranscriptsSaveDisabled(saving: false, webhookUrl: ""))
        XCTAssertFalse(TranscriptsAdvisingAdminLogic.isTranscriptsSaveDisabled(
            saving: false,
            webhookUrl: "https://example.edu"
        ))
    }

    func testAdvisingSavePayload() {
        let none = TranscriptsAdvisingAdminLogic.buildAdvisingSaveRequest(
            appointmentUrl: " https://navigate.example.edu ",
            provider: .none,
            baseUrl: "https://should-clear.example",
            credentialsRef: "secret-ref",
            atRiskBannerEnabled: true
        )
        XCTAssertEqual(none.appointmentUrl, "https://navigate.example.edu")
        XCTAssertEqual(none.degreeAuditProvider, "none")
        XCTAssertEqual(none.degreeAuditBaseUrl, "")
        XCTAssertEqual(none.apiCredentialsRef, "")
        XCTAssertFalse(none.atRiskBannerEnabled)

        let full = TranscriptsAdvisingAdminLogic.buildAdvisingSaveRequest(
            appointmentUrl: "",
            provider: .degreeworks,
            baseUrl: " https://degreeworks.example.edu/api ",
            credentialsRef: " cred-1 ",
            atRiskBannerEnabled: true
        )
        XCTAssertEqual(full.degreeAuditProvider, "degreeworks")
        XCTAssertEqual(full.degreeAuditBaseUrl, "https://degreeworks.example.edu/api")
        XCTAssertEqual(full.apiCredentialsRef, "cred-1")
        XCTAssertTrue(full.atRiskBannerEnabled)
    }

    func testDecodeModels() throws {
        let transcriptsData = Data(#"""
        {
          "webhookUrl":"https://sis.example.edu/hook",
          "webhookSecret":"••••••••••••",
          "hasWebhookSecret":true,
          "pickupInstructions":"Room 101"
        }
        """#.utf8)
        let cfg = try JSONDecoder().decode(AdminTranscriptsConfig.self, from: transcriptsData)
        XCTAssertEqual(cfg.webhookUrl, "https://sis.example.edu/hook")
        XCTAssertTrue(cfg.hasWebhookSecret)
        XCTAssertEqual(TranscriptsAdvisingAdminLogic.webhookSecretField(from: cfg), TranscriptsAdvisingAdminLogic.secretPlaceholder)

        let advisingData = Data(#"""
        {
          "appointmentUrl":"https://navigate.example.edu",
          "degreeAuditProvider":"stellic",
          "degreeAuditBaseUrl":"https://stellic.example.edu",
          "apiCredentialsRef":"ref-1",
          "atRiskBannerEnabled":true
        }
        """#.utf8)
        let advising = try JSONDecoder().decode(AdminAdvisingConfig.self, from: advisingData)
        XCTAssertEqual(
            TranscriptsAdvisingAdminLogic.DegreeAuditProvider.normalized(advising.degreeAuditProvider),
            .stellic
        )
        XCTAssertTrue(advising.atRiskBannerEnabled)
    }

    func testWebPaths() {
        XCTAssertEqual(TranscriptsAdvisingAdminLogic.Section.transcripts.webPath, "/settings/transcripts")
        XCTAssertEqual(TranscriptsAdvisingAdminLogic.Section.advising.webPath, "/settings/advising")
        XCTAssertEqual(TranscriptsAdvisingAdminLogic.webHubPath(), "/settings/transcripts")
    }
}
