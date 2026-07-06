import XCTest
@testable import Lextures

final class WalletLogicTests: XCTestCase {
    func testWalletEnabled() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(WalletLogic.walletEnabled(features))
        features.ffCompletionCredentials = true
        XCTAssertTrue(WalletLogic.walletEnabled(features))
        features.ffCompletionCredentials = false
        features.ffCoCurricularTranscript = true
        XCTAssertTrue(WalletLogic.walletEnabled(features))
        features.ffCeuTracking = true
        features.ffTranscripts = true
        XCTAssertTrue(WalletLogic.walletEnabled(features))
    }

    func testTranscriptStatusLabel() {
        XCTAssertEqual(WalletLogic.transcriptStatusLabel("queued"), L.text("mobile.wallet.requestStatus.queued"))
        XCTAssertEqual(WalletLogic.transcriptStatusLabel("submitted"), L.text("mobile.wallet.requestStatus.submitted"))
        XCTAssertEqual(WalletLogic.transcriptStatusLabel("failed"), L.text("mobile.wallet.requestStatus.failed"))
    }

    func testCacheKeys() {
        XCTAssertEqual(WalletLogic.cacheKeyCCR(), "wallet:ccr")
        XCTAssertEqual(WalletLogic.cacheKeyCETranscript(), "wallet:ce-transcript")
        XCTAssertEqual(WalletLogic.cacheKeyTranscriptRequests(), "wallet:transcript-requests")
    }

    func testOfficialTranscriptWebURL() {
        XCTAssertTrue(WalletLogic.officialTranscriptWebURL().absoluteString.contains("/transcripts"))
    }
}
