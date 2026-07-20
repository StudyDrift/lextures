import XCTest
@testable import Lextures

final class PlatformSettingsAdminLogicTests: XCTestCase {
    func testEntryRequiresFlagAndRbacPermission() {
        // Legacy settings entry only shows when admin console is off and admin settings is on.
        let consoleOn = MobilePlatformFeatures(ffMobileAdminConsole: true, ffMobileAdminSettings: true)
        XCTAssertFalse(PlatformSettingsAdminLogic.shouldShowEntry(
            features: consoleOn,
            permissions: [PlatformSettingsAdminLogic.rbacManagePermission]
        ))
        let legacy = MobilePlatformFeatures(ffMobileAdminConsole: false, ffMobileAdminSettings: true)
        XCTAssertFalse(PlatformSettingsAdminLogic.shouldShowEntry(features: legacy, permissions: []))
        XCTAssertTrue(PlatformSettingsAdminLogic.shouldShowEntry(
            features: legacy,
            permissions: [PlatformSettingsAdminLogic.rbacManagePermission]
        ))
    }

    func testAllowlistExcludesLockoutAndInfrastructureFlags() {
        let keys = Set(PlatformSettingsAdminLogic.featureDefinitions.map(\.key))
        XCTAssertFalse(keys.contains("ffMobileAdminSettings"))
        XCTAssertFalse(keys.contains("samlSsoEnabled"))
        XCTAssertFalse(keys.contains("ffPaymentsEnabled"))
        XCTAssertFalse(keys.contains("ffFeedback"))
        XCTAssertTrue(keys.contains("ffPublicCatalog"))
    }

    func testSecretFieldsAreIgnoredWhenDecodingSnapshot() throws {
        let data = Data(#"""
        {
          "ffFeedback":true,"ffPublicCatalog":false,"ffCourseMarketplace":true,
          "ffLearningPaths":false,"ffPeerReview":false,"ffCompletionCredentials":false,
          "ffPersistentTutor":false,"ffAiStudyBuddy":false,"ffClassroomSignals":false,
          "ffBroadcasts":false,"ffCalendarFeeds":true,"learnerProfileEnabled":true,
          "samlSsoEnabled":false,"samlPublicBaseUrl":"","samlSpEntityId":"",
          "mfaEnabled":false,"mfaEnforcement":"none","smtpHost":"","smtpPort":587,
          "smtpFrom":"","smtpPassword":"••••••••••••","samlSpPrivateKeyPem":"••••••••••••"
        }
        """#.utf8)
        let snapshot = try JSONDecoder().decode(PlatformSettingsSnapshot.self, from: data)
        XCTAssertTrue(snapshot.ffFeedback)
        XCTAssertFalse(PlatformSettingsAdminLogic.value(for: "ffFeedback", in: snapshot))
        XCTAssertFalse(PlatformSettingsAdminLogic.value(for: "ffPublicCatalog", in: snapshot))

        let effective = try JSONDecoder().decode(
            PlatformFeatureStates.self,
            from: Data(#"{"ffPublicCatalog":true}"#.utf8)
        )
        let merged = PlatformSettingsAdminLogic.applyingEffectiveFeatures(effective, to: snapshot)
        XCTAssertTrue(merged.ffPublicCatalog)
    }
}
