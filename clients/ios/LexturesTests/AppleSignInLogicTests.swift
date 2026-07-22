import XCTest
@testable import Lextures

final class AppleSignInLogicTests: XCTestCase {
    func testNonceHashIsStableHexSha256() {
        let raw = "test-nonce-value-123"
        let hashOne = AppleSignInController.sha256Hex(raw)
        let hashTwo = AppleSignInController.sha256Hex(raw)
        XCTAssertEqual(hashOne, hashTwo)
        XCTAssertEqual(hashOne.count, 64)
        XCTAssertTrue(hashOne.allSatisfy { $0.isHexDigit })
        XCTAssertNotEqual(hashOne, AppleSignInController.sha256Hex("other"))
    }

    func testRandomNonceLengthAndCharset() {
        let firstNonce = AppleSignInController.randomNonceString(length: 32)
        let secondNonce = AppleSignInController.randomNonceString(length: 32)
        XCTAssertEqual(firstNonce.count, 32)
        XCTAssertEqual(secondNonce.count, 32)
        XCTAssertNotEqual(firstNonce, secondNonce)
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
