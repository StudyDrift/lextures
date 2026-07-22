import XCTest
@testable import Lextures

final class SchoolCodeLogicTests: XCTestCase {
    func testNormalizeTrimsAndLowercases() {
        XCTAssertEqual(SchoolCodeLogic.normalize("  Example-School "), "example-school")
    }

    func testValidSchoolCode() {
        XCTAssertTrue(SchoolCodeLogic.isValid("example"))
        XCTAssertTrue(SchoolCodeLogic.isValid("my-school"))
        XCTAssertTrue(SchoolCodeLogic.isValid("local"))
    }

    func testInvalidSchoolCodes() {
        XCTAssertFalse(SchoolCodeLogic.isValid(""))
        XCTAssertFalse(SchoolCodeLogic.isValid("a"))
        XCTAssertFalse(SchoolCodeLogic.isValid("bad_code"))
        XCTAssertFalse(SchoolCodeLogic.isValid("-leading"))
        XCTAssertFalse(SchoolCodeLogic.isValid("self"))
        XCTAssertFalse(SchoolCodeLogic.isValid("www"))
    }

    func testMixedCaseIsNormalizedBeforeValidation() {
        XCTAssertTrue(SchoolCodeLogic.isValid("Example"))
        XCTAssertEqual(SchoolCodeLogic.normalize("Example"), "example")
    }

    func testHomeschoolBase() {
        XCTAssertEqual(SchoolCodeLogic.homeschoolAPIBase, "https://self.lextures.com")
    }

    func testHomeschoolKindRawValuePinned() {
        // DO NOT RENAME — installed devices persist "selfLearner" in UserDefaults.
        XCTAssertEqual(EnvironmentStore.Kind.homeschool.rawValue, "selfLearner")
    }

    func testSchoolAPIBase() {
        XCTAssertEqual(SchoolCodeLogic.apiBaseURL(schoolCode: "example"), "https://example.lextures.com")
        XCTAssertEqual(SchoolCodeLogic.apiBaseURL(schoolCode: "Local"), "http://127.0.0.1:8080")
    }

    func testPreviewHost() {
        XCTAssertEqual(SchoolCodeLogic.previewHost(schoolCode: ""), "your-school.lextures.com")
        XCTAssertEqual(SchoolCodeLogic.previewHost(schoolCode: "local"), "127.0.0.1:8080")
        XCTAssertEqual(SchoolCodeLogic.previewHost(schoolCode: "demo-uni"), "demo-uni.lextures.com")
    }
}

final class EnvironmentStoreTests: XCTestCase {
    private var defaults: UserDefaults!
    private var suiteName: String!

    override func setUp() {
        super.setUp()
        suiteName = "EnvironmentStoreTests.\(UUID().uuidString)"
        defaults = UserDefaults(suiteName: suiteName)
    }

    override func tearDown() {
        if let suiteName {
            defaults.removePersistentDomain(forName: suiteName)
        }
        defaults = nil
        suiteName = nil
        super.tearDown()
    }

    func testSelectHomeschool() {
        let store = EnvironmentStore(defaults: defaults)
        XCTAssertFalse(store.hasSelection)
        store.selectHomeschool()
        XCTAssertTrue(store.hasSelection)
        XCTAssertEqual(store.kind, .homeschool)
        XCTAssertEqual(store.apiBaseURLString, "https://self.lextures.com")
        XCTAssertNil(store.schoolCode)
        // Persisted raw value must remain the pre-rebrand token.
        XCTAssertEqual(defaults.string(forKey: "lextures.environment.kind"), "selfLearner")
    }

    func testSelectSchoolAndLocal() {
        let store = EnvironmentStore(defaults: defaults)
        store.selectSchool(code: "acme")
        XCTAssertEqual(store.kind, .school)
        XCTAssertEqual(store.schoolCode, "acme")
        XCTAssertEqual(store.apiBaseURLString, "https://acme.lextures.com")

        store.selectSchool(code: "local")
        XCTAssertEqual(store.apiBaseURLString, "http://127.0.0.1:8080")
        XCTAssertEqual(store.schoolCode, "local")
    }

    func testClearSelection() {
        let store = EnvironmentStore(defaults: defaults)
        store.selectHomeschool()
        store.clearSelection()
        XCTAssertFalse(store.hasSelection)
        XCTAssertNil(store.apiBaseURLString)
        XCTAssertNil(store.kind)
        XCTAssertNil(store.schoolCode)
    }

    /// Upgrade-in-place: a pre-rename install that stored "selfLearner" must map to homeschool.
    func testLegacySelfLearnerRawValueReadsAsHomeschool() {
        defaults.set("selfLearner", forKey: "lextures.environment.kind")
        defaults.set("https://self.lextures.com", forKey: "lextures.environment.apiBaseURL")

        let store = EnvironmentStore(defaults: defaults)
        XCTAssertTrue(store.hasSelection)
        XCTAssertEqual(store.kind, .homeschool)
        XCTAssertEqual(store.apiBaseURLString, "https://self.lextures.com")
        XCTAssertEqual(SchoolCodeLogic.homeschoolAPIBase, store.apiBaseURLString)
    }
}
