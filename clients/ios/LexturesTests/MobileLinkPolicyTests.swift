import XCTest
@testable import Lextures

final class MobileLinkPolicyTests: XCTestCase {
    private func fixtureURL() -> URL {
        let thisFile = URL(fileURLWithPath: #filePath)
        let direct = thisFile
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("mobile/fixtures/browser/link-policy.json")
        if FileManager.default.fileExists(atPath: direct.path) { return direct }
        var dir = thisFile.deletingLastPathComponent()
        for _ in 0 ..< 8 {
            let candidate = dir.appendingPathComponent("clients/mobile/fixtures/browser/link-policy.json")
            if FileManager.default.fileExists(atPath: candidate.path) { return candidate }
            dir = dir.deletingLastPathComponent()
        }
        return direct
    }

    private func fixtureRoot() throws -> [String: Any] {
        let data = try Data(contentsOf: fixtureURL())
        return try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }

    private func string(_ value: Any?) throws -> String {
        try XCTUnwrap(value as? String)
    }

    private func bool(_ value: Any?) throws -> Bool {
        try XCTUnwrap(value as? Bool)
    }

    private func objects(_ value: Any?) throws -> [[String: Any]] {
        try XCTUnwrap(value as? [[String: Any]])
    }

    func testClassifyMatchesSharedFixture() throws {
        let cases = try objects(try fixtureRoot()["cases"])
        for item in cases {
            let name = try string(item["name"])
            let url = try string(item["url"])
            let policy = MobileLinkPolicy.Handling.parse(try string(item["policy"]))
            let flagOn = try bool(item["flagOn"])
            let apiHost = try string(item["apiHost"])
            let want = try string(item["want"])
            let state = MobileLinkPolicy.State(
                handling: policy,
                inAppBrowserEnabled: flagOn,
                apiHost: apiHost
            )
            let got = MobileLinkPolicy.classify(urlString: url, state: state).rawValue
            XCTAssertEqual(got, want, name)
        }
    }

    func testBearerAttachmentMatchesSharedFixture() throws {
        let cases = try objects(try fixtureRoot()["bearerAttachment"])
        for item in cases {
            let name = try string(item["name"])
            let requestHost = try string(item["requestHost"])
            let apiHost = try string(item["apiHost"])
            let want = try bool(item["want"])
            let got = MobileLinkPolicy.shouldAttachBearer(requestHost: requestHost, apiHost: apiHost)
            XCTAssertEqual(got, want, name)
        }
    }

    func testLookalikeHostNeverGetsBearer() {
        XCTAssertFalse(
            MobileLinkPolicy.shouldAttachBearer(
                requestHost: "api.lextures.com.example.net",
                apiHost: "api.lextures.com"
            )
        )
    }

    func testTelemetryDropsForbiddenKeys() {
        let raw = [
            "source": "content_page",
            "classification": "external",
            "outcome": "opened",
            "url": "https://evil.example/x",
            "host": "evil.example",
            "title": "Nope",
        ]
        let sanitized = MobileLinkPolicy.sanitizeTelemetry(raw)
        XCTAssertEqual(sanitized["source"], "content_page")
        XCTAssertEqual(sanitized["classification"], "external")
        XCTAssertNil(sanitized["url"])
        XCTAssertNil(sanitized["host"])
        XCTAssertNil(sanitized["title"])
    }

    func testUnknownHandlingFallsBackToInApp() {
        XCTAssertEqual(MobileLinkPolicy.Handling.parse("nonsense"), .inApp)
        XCTAssertEqual(MobileLinkPolicy.Handling.parse(nil), .inApp)
    }
}
