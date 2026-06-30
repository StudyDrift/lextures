import XCTest
@testable import Lextures

final class InteractiveLaunchLogicTests: XCTestCase {
    func testKindForItemKinds() {
        XCTAssertEqual(InteractiveLaunchLogic.kind(for: "h5p"), .h5p)
        XCTAssertEqual(InteractiveLaunchLogic.kind(for: "scorm"), .scorm)
        XCTAssertEqual(InteractiveLaunchLogic.kind(for: "lti_link"), .ltiLink)
        XCTAssertEqual(InteractiveLaunchLogic.kind(for: "vibe_activity"), .vibeActivity)
        XCTAssertNil(InteractiveLaunchLogic.kind(for: "quiz"))
    }

    func testH5pRenderPath() {
        let path = InteractiveLaunchLogic.h5pRenderPath(courseCode: "CS101", packageId: "pkg-1")
        XCTAssertTrue(path.contains("/courses/CS101/h5p/pkg-1/render"))
    }

    func testLtiFramePathIncludesTicket() {
        let path = InteractiveLaunchLogic.ltiFramePath(ticket: "abc123")
        XCTAssertTrue(path.contains("ticket=abc123"))
    }

    func testScormHasResumeDetectsSuspendData() {
        XCTAssertTrue(InteractiveLaunchLogic.scormHasResume(initialCmi: ["cmi.core.suspend_data": "state"]))
        XCTAssertTrue(InteractiveLaunchLogic.scormHasResume(initialCmi: ["cmi.core.entry": "resume"]))
        XCTAssertFalse(InteractiveLaunchLogic.scormHasResume(initialCmi: [:]))
    }

    func testVibeActivityHTMLFallback() {
        let html = InteractiveLaunchLogic.vibeActivityHTML(nil)
        XCTAssertTrue(html.contains("Empty activity"))
    }

    func testAuthInjectionScriptIncludesTokenMarker() {
        let script = InteractiveLaunchLogic.authInjectionScript(accessToken: "tok", apiBase: "http://localhost:8080")
        XCTAssertTrue(script.contains("Bearer "))
        XCTAssertTrue(script.contains("tok"))
        XCTAssertTrue(script.contains("h5p-xapi"))
    }
}
