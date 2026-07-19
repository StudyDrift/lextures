import XCTest
@testable import Lextures

final class CanvasImportLogicTests: XCTestCase {
    func testValidateCredentials() {
        XCTAssertEqual(
            CanvasImportLogic.validateCredentials(baseURL: "", accessToken: "tok"),
            "mobile.canvasImport.error.urlRequired"
        )
        XCTAssertEqual(
            CanvasImportLogic.validateCredentials(baseURL: "canvas.example.edu", accessToken: "tok"),
            "mobile.canvasImport.error.urlInvalid"
        )
        XCTAssertEqual(
            CanvasImportLogic.validateCredentials(baseURL: "https://canvas.example.edu", accessToken: "  "),
            "mobile.canvasImport.error.tokenRequired"
        )
        XCTAssertNil(
            CanvasImportLogic.validateCredentials(
                baseURL: "https://canvas.example.edu/",
                accessToken: "secret-token"
            )
        )
    }

    func testNormalizeBaseURL() {
        XCTAssertEqual(
            CanvasImportLogic.normalizeBaseURL(" https://canvas.example.edu/ "),
            "https://canvas.example.edu"
        )
    }

    func testIncludeSerializationDefaultsAllOn() {
        let include = CanvasImportLogic.Include.all
        for category in CanvasImportLogic.IncludeCategory.allCases {
            XCTAssertTrue(include.value(for: category))
        }
        var gradesOff = include
        gradesOff.set(.grades, false)
        XCTAssertFalse(gradesOff.grades)
        XCTAssertTrue(gradesOff.modules)
        XCTAssertEqual(gradesOff.enabledCategoryCounts["grades"], 0)
        XCTAssertEqual(gradesOff.enabledCategoryCounts["modules"], 1)
    }

    func testParseWSMessage() {
        let progress = #"{"type":"progress","message":"Importing modules"}"#.data(using: .utf8)!
        let parsed = CanvasImportLogic.parseWSMessage(from: progress)
        XCTAssertEqual(parsed?.type, .progress)
        XCTAssertEqual(parsed?.message, "Importing modules")
        XCTAssertTrue(CanvasImportLogic.isTerminal(.complete))
        XCTAssertTrue(CanvasImportLogic.isTerminal(.error))
        XCTAssertFalse(CanvasImportLogic.isTerminal(.progress))
    }

    func testEntryGateRequiresFlagPermissionAndOnline() {
        var features = MobilePlatformFeatures()
        features.ffMobileCourseCreateV2 = true
        features.ffMobileCanvasImport = false
        let perms = [CanvasImportLogic.courseCreatePermission]
        XCTAssertFalse(
            CanvasImportLogic.shouldShowCanvasImportEntry(
                permissions: perms,
                features: features,
                isOnline: true
            )
        )
        features.ffMobileCanvasImport = true
        XCTAssertTrue(
            CanvasImportLogic.shouldShowCanvasImportEntry(
                permissions: perms,
                features: features,
                isOnline: true
            )
        )
        XCTAssertFalse(
            CanvasImportLogic.shouldShowCanvasImportEntry(
                permissions: perms,
                features: features,
                isOnline: false
            )
        )
        XCTAssertFalse(
            CanvasImportLogic.shouldShowCanvasImportEntry(
                permissions: [],
                features: features,
                isOnline: true
            )
        )
    }

    func testTokenAbsenceFromStorageHaystacks() {
        let token = "canvas-secret-token-abc"
        XCTAssertFalse(
            CanvasImportLogic.storageContainsToken(
                haystacks: ["userDefaults:ok", "keychain:session-jwt"],
                token: token
            )
        )
        XCTAssertTrue(
            CanvasImportLogic.storageContainsToken(
                haystacks: ["oops \(token) leaked"],
                token: token
            )
        )
        XCTAssertEqual(
            CanvasImportLogic.tokenMustNotPersistPolicy.contains("never persisted"),
            true
        )
    }

    func testCancelErrorRecognition() {
        XCTAssertTrue(CanvasImportLogic.isCancelledError(CanvasImportLogic.CanvasImportError.cancelled))
        XCTAssertFalse(CanvasImportLogic.isCancelledError(CanvasImportLogic.CanvasImportError.missingJobId))
    }

    func testFilterCourses() {
        let courses = [
            CanvasCourseListItem(id: 1, name: "Biology 101", courseCode: "BIO101", workflowState: "available", termName: "Fall"),
            CanvasCourseListItem(id: 2, name: "Chemistry", courseCode: "CHEM", workflowState: "unpublished", termName: "Spring"),
        ]
        XCTAssertEqual(CanvasImportLogic.filterCourses(courses, query: "bio").count, 1)
        XCTAssertTrue(CanvasImportLogic.isUnpublished("unpublished"))
        XCTAssertFalse(CanvasImportLogic.isUnpublished("available"))
    }
}
