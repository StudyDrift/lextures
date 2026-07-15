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

    func testSelfLearnerBase() {
        XCTAssertEqual(SchoolCodeLogic.selfLearnerAPIBase, "https://self.lextures.com")
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

    func testSelectSelfLearner() {
        let store = EnvironmentStore(defaults: defaults)
        XCTAssertFalse(store.hasSelection)
        store.selectSelfLearner()
        XCTAssertTrue(store.hasSelection)
        XCTAssertEqual(store.kind, .selfLearner)
        XCTAssertEqual(store.apiBaseURLString, "https://self.lextures.com")
        XCTAssertNil(store.schoolCode)
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
        store.selectSelfLearner()
        store.clearSelection()
        XCTAssertFalse(store.hasSelection)
        XCTAssertNil(store.apiBaseURLString)
        XCTAssertNil(store.kind)
        XCTAssertNil(store.schoolCode)
    }
}
