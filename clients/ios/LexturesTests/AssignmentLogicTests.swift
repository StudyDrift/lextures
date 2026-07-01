import XCTest
@testable import Lextures

final class AssignmentLogicTests: XCTestCase {
    func testAcceptsAnySubmission() {
        var detail = ModuleItemDetail()
        detail.submissionAllowText = true
        XCTAssertTrue(AssignmentLogic.acceptsAnySubmission(detail))
        XCTAssertFalse(AssignmentLogic.acceptsAnySubmission(nil))
    }

    func testCanReplaceFileWhenNoAttachment() {
        XCTAssertTrue(AssignmentLogic.canReplaceFile(submission: nil, resubmissionWorkflowEnabled: true))
    }

    func testCannotReplaceFileWhenLocked() {
        let submission = AssignmentSubmission(
            id: "s1",
            attachmentFilename: "work.pdf",
            submittedAt: "2026-01-01T00:00:00Z"
        )
        XCTAssertFalse(AssignmentLogic.canReplaceFile(submission: submission, resubmissionWorkflowEnabled: true))
    }

    func testFileLockedWhenWorkflowEnabledAndHasAttachment() {
        var detail = ModuleItemDetail()
        detail.submissionAllowFileUpload = true
        let submission = AssignmentSubmission(
            id: "s1",
            attachmentFilename: "work.pdf",
            submittedAt: "2026-01-01T00:00:00Z"
        )
        XCTAssertFalse(AssignmentLogic.canSubmitFile(
            detail: detail,
            submission: submission,
            resubmissionWorkflowEnabled: true
        ))
    }

    func testLateSubmissionBlocked() {
        let pastDue = ISO8601DateFormatter().string(from: Date(timeIntervalSinceNow: -3600))
        var detail = ModuleItemDetail()
        detail.dueAt = pastDue
        detail.lateSubmissionPolicy = "block"
        detail.submissionAllowText = true
        XCTAssertFalse(AssignmentLogic.canSubmit(detail: detail, submission: nil, resubmissionWorkflowEnabled: false))
        XCTAssertEqual(
            AssignmentLogic.submitDisabledReasonKey(detail: detail, submission: nil, resubmissionWorkflowEnabled: false),
            "mobile.assignment.pastDueBlocked"
        )
    }

    func testAllowedMimeTypes() {
        XCTAssertTrue(AssignmentLogic.isAllowedMimeType("image/jpeg"))
        XCTAssertTrue(AssignmentLogic.isAllowedMimeType("application/pdf"))
        XCTAssertFalse(AssignmentLogic.isAllowedMimeType("application/zip"))
    }

    func testMaxUploadSize() {
        XCTAssertTrue(AssignmentLogic.isAllowedFileSize(1024))
        XCTAssertFalse(AssignmentLogic.isAllowedFileSize(AssignmentLogic.maxUploadBytes + 1))
    }
}
