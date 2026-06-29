import XCTest
@testable import Lextures

final class BiometricGateTests: XCTestCase {
    func testShouldLockAfterTimeout() {
        XCTAssertTrue(BiometricGate.shouldLock(afterBackgroundDuration: 60))
        XCTAssertTrue(BiometricGate.shouldLock(afterBackgroundDuration: 120))
    }

    func testShouldNotLockBeforeTimeout() {
        XCTAssertFalse(BiometricGate.shouldLock(afterBackgroundDuration: 59))
        XCTAssertFalse(BiometricGate.shouldLock(afterBackgroundDuration: 0))
    }
}
