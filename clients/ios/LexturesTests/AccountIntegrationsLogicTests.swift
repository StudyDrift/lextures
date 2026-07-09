import XCTest
@testable import Lextures

final class AccountIntegrationsLogicTests: XCTestCase {
    func testIntegrationsEnabledRequiresFlag() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(AccountIntegrationsLogic.integrationsEnabled(features))
        features.ffMobileSettingsIntegrations = true
        XCTAssertTrue(AccountIntegrationsLogic.integrationsEnabled(features))
    }

    func testCanManageServiceTokensRequiresRbacManage() {
        XCTAssertFalse(AccountIntegrationsLogic.canManageServiceTokens(permissions: []))
        XCTAssertTrue(
            AccountIntegrationsLogic.canManageServiceTokens(permissions: ["global:app:rbac:manage"])
        )
    }

    func testShouldHideServiceTokensForNonAdmin() {
        XCTAssertFalse(
            AccountIntegrationsLogic.shouldShowServiceTokensSection(
                permissions: [],
                adminApiForbidden: true
            )
        )
        XCTAssertTrue(
            AccountIntegrationsLogic.shouldShowServiceTokensSection(
                permissions: ["global:app:rbac:manage"],
                adminApiForbidden: false
            )
        )
    }

    func testResolveCalendarFeedURLSubstitutesToken() {
        let url = AccountIntegrationsLogic.resolveCalendarFeedURL(
            template: "https://example.com/feed?token=<token>",
            token: "abc+def"
        )
        XCTAssertTrue(url.contains("abc"))
        XCTAssertFalse(url.contains("<token>"))
    }

    func testResolvedPersonalFeedURLPrefersCreatedFeed() {
        let created = CalendarTokenCreated(token: "t1", feedUrl: "https://example.com/ready", expiresAt: nil)
        let resolved = AccountIntegrationsLogic.resolvedPersonalFeedURL(info: nil, created: created)
        XCTAssertEqual(resolved, "https://example.com/ready")
    }

    func testActiveAccessKeysExcludesRevokedAndServiceTokens() {
        let tokens = [
            AccessKeySummary(
                id: "1",
                label: "Active",
                tokenMask: "ltk_***",
                scopes: ["mcp:connect"],
                isServiceToken: false,
                createdAt: "2026-01-01T00:00:00Z"
            ),
            AccessKeySummary(
                id: "2",
                label: "Revoked",
                tokenMask: "ltk_***",
                scopes: ["courses:read"],
                revokedAt: "2026-01-02T00:00:00Z",
                createdAt: "2026-01-01T00:00:00Z"
            ),
            AccessKeySummary(
                id: "3",
                label: "Service",
                tokenMask: "ltk_***",
                scopes: ["enrollments:read"],
                isServiceToken: true,
                createdAt: "2026-01-01T00:00:00Z"
            ),
        ]
        XCTAssertEqual(AccountIntegrationsLogic.activeAccessKeys(tokens).map(\.id), ["1"])
    }
}