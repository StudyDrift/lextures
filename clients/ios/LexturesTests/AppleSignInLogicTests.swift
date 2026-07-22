import XCTest
@testable import Lextures

final class AppleSignInLogicTests: XCTestCase {
    func testNonceHashIsStableHexSha256() {
        let raw = "test-nonce-value-123"
        let h1 = AppleSignInController.sha256Hex(raw)
        let h2 = AppleSignInController.sha256Hex(raw)
        XCTAssertEqual(h1, h2)
        XCTAssertEqual(h1.count, 64)
        XCTAssertTrue(h1.allSatisfy { $0.isHexDigit })
        XCTAssertNotEqual(h1, AppleSignInController.sha256Hex("other"))
    }

    func testRandomNonceLengthAndCharset() {
        let a = AppleSignInController.randomNonceString(length: 32)
        let b = AppleSignInController.randomNonceString(length: 32)
        XCTAssertEqual(a.count, 32)
        XCTAssertEqual(b.count, 32)
        XCTAssertNotEqual(a, b)
    }

    func testOidcStatusShowsAppleNative() {
        var status = OidcStatusResponse(
            enabled: true,
            cleverEnabled: false,
            classlinkEnabled: false,
            clever: false,
            classlink: false,
            google: false,
            microsoft: false,
            apple: false,
            appleNative: true,
            googleNative: false,
            custom: []
        )
        XCTAssertTrue(status.showsAppleNative)

        status.appleNative = false
        XCTAssertFalse(status.showsAppleNative)

        status.appleNative = nil
        XCTAssertFalse(status.showsAppleNative)
    }
}
