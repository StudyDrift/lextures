import XCTest
@testable import Lextures

final class AuthCallbackParserTests: XCTestCase {
    func testParsesAuthCallbackQueryParams() {
        let payload = AuthCallbackParser.parse(
            "lextures://auth/callback?access_token=abc123&refresh_token=def456"
        )
        XCTAssertEqual(payload?.accessToken, "abc123")
        XCTAssertEqual(payload?.refreshToken, "def456")
    }

    func testParsesAuthCallbackMfaParams() {
        let payload = AuthCallbackParser.parse(
            "lextures://auth/callback?mfa_pending_token=pending&requires_mfa=1&mfa_setup_required=1"
        )
        XCTAssertEqual(payload?.mfaPendingToken, "pending")
        XCTAssertTrue(payload?.requiresMFA == true)
        XCTAssertTrue(payload?.mfaSetupRequired == true)
    }

    func testParsesMagicLinkHttpsUrl() {
        let payload = AuthCallbackParser.parse(
            "https://lextures.com/login/magic-link?token=ml-token-123"
        )
        XCTAssertEqual(payload?.magicLinkToken, "ml-token-123")
    }

    func testIgnoresNavigationLinks() {
        XCTAssertNil(AuthCallbackParser.parse("/courses/cs101/grades"))
    }
}
