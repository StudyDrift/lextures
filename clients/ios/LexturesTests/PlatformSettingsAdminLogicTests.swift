import XCTest
@testable import Lextures

final class PlatformSettingsAdminLogicTests: XCTestCase {
    func testEntryRequiresFlagAndRbacPermission() {
        let off = MobilePlatformFeatures(ffMobileAdminSettings: false)
        XCTAssertFalse(PlatformSettingsAdminLogic.shouldShowEntry(
            features: off,
            permissions: [PlatformSettingsAdminLogic.rbacManagePermission]
        ))
        let on = MobilePlatformFeatures(ffMobileAdminSettings: true)
        XCTAssertFalse(PlatformSettingsAdminLogic.shouldShowEntry(features: on, permissions: []))
        XCTAssertTrue(PlatformSettingsAdminLogic.shouldShowEntry(
            features: on,
            permissions: [PlatformSettingsAdminLogic.rbacManagePermission]
        ))
    }

    func testAllowlistExcludesLockoutAndInfrastructureFlags() {
        let keys = Set(PlatformSettingsAdminLogic.featureDefinitions.map(\.key))
        XCTAssertFalse(keys.contains("ffMobileAdminSettings"))
        XCTAssertFalse(keys.contains("samlSsoEnabled"))
        XCTAssertFalse(keys.contains("ffPaymentsEnabled"))
        XCTAssertTrue(keys.contains("ffFeedback"))
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
        XCTAssertTrue(PlatformSettingsAdminLogic.value(for: "ffFeedback", in: snapshot))

        let effective = try JSONDecoder().decode(
            PlatformFeatureStates.self,
            from: Data(#"{"ffFeedback":false}"#.utf8)
        )
        let merged = PlatformSettingsAdminLogic.applyingEffectiveFeatures(effective, to: snapshot)
        XCTAssertFalse(merged.ffFeedback)
    }
}
