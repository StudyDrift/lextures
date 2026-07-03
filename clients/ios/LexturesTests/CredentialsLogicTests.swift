import XCTest
@testable import Lextures

final class CredentialsLogicTests: XCTestCase {
    func testCredentialsEnabled() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(CredentialsLogic.credentialsEnabled(features))
        features.ffCompletionCredentials = true
        XCTAssertTrue(CredentialsLogic.credentialsEnabled(features))
    }

    func testSourceTypeLabel() {
        XCTAssertEqual(CredentialsLogic.sourceTypeLabel("course"), L.text("mobile.credentials.source.course"))
        XCTAssertEqual(CredentialsLogic.sourceTypeLabel("path"), L.text("mobile.credentials.source.path"))
    }

    func testCacheKey() {
        XCTAssertEqual(CredentialsLogic.cacheKey(), "credentials:list")
        XCTAssertEqual(CredentialsLogic.credentialDetailCacheKey(id: "abc"), "credentials:abc")
    }
}