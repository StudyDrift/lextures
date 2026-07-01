import Foundation

// MARK: - Assignment submission (M5.1)

enum AssignmentSubmissionStatus: Equatable {
    case notStarted
    case submitted
    case late
    case graded
    case revisionRequested
}

enum AssignmentLogic {
    static let maxUploadBytes: Int64 = 20 * 1024 * 1024

    static func draftStorageKey(courseCode: String, itemId: String) -> String {
        "assignment-draft:\(courseCode):\(itemId)"
    }

    static func acceptsAnySubmission(_ detail: ModuleItemDetail?) -> Bool {
        guard let detail else { return false }
        return (detail.submissionAllowText ?? false)
            || (detail.submissionAllowFileUpload ?? false)
            || (detail.submissionAllowUrl ?? false)
    }

    static func status(
        submission: AssignmentSubmission?,
        grade: SubmissionGrade?,
        detail: ModuleItemDetail?,
        now: Date = Date()
    ) -> AssignmentSubmissionStatus {
        if submission?.resubmissionRequested == true {
            return .revisionRequested
        }
        if grade?.posted == true {
            return .graded
        }
        if submission != nil {
            if isLate(detail: detail, submittedAt: submission?.submittedAt, now: now) {
                return .late
            }
            return .submitted
        }
        return .notStarted
    }

    static func isPastDue(detail: ModuleItemDetail?, now: Date = Date()) -> Bool {
        guard let due = LMSDates.parse(detail?.dueAt ?? detail?.availableUntil) else { return false }
        return due < now
    }

    static func isPastAvailableUntil(detail: ModuleItemDetail?, now: Date = Date()) -> Bool {
        guard let until = LMSDates.parse(detail?.availableUntil) else { return false }
        return until < now
    }

    static func isRevisionOverdue(submission: AssignmentSubmission?, now: Date = Date()) -> Bool {
        guard submission?.resubmissionRequested == true,
              let due = LMSDates.parse(submission?.revisionDueAt) else { return false }
        return due < now
    }

    static func isLate(detail: ModuleItemDetail?, submittedAt: String?, now: Date = Date()) -> Bool {
        guard let due = LMSDates.parse(detail?.dueAt),
              let submitted = LMSDates.parse(submittedAt) else { return false }
        return submitted > due
    }

    static func hasAttachment(_ submission: AssignmentSubmission?) -> Bool {
        guard let submission else { return false }
        if let name = submission.attachmentFilename, !name.isEmpty { return true }
        if let path = submission.attachmentContentPath, !path.isEmpty { return true }
        return false
    }

    static func canReplaceFile(
        submission: AssignmentSubmission?,
        resubmissionWorkflowEnabled: Bool
    ) -> Bool {
        if submission?.resubmissionRequested == true { return true }
        if !resubmissionWorkflowEnabled { return true }
        return !hasAttachment(submission)
    }

    static func canSubmitText(
        detail: ModuleItemDetail?,
        submission: AssignmentSubmission?,
        resubmissionWorkflowEnabled: Bool,
        now: Date = Date()
    ) -> Bool {
        guard detail?.submissionAllowText == true else { return false }
        return canSubmit(
            detail: detail,
            submission: submission,
            resubmissionWorkflowEnabled: resubmissionWorkflowEnabled,
            now: now
        )
    }

    static func canSubmitFile(
        detail: ModuleItemDetail?,
        submission: AssignmentSubmission?,
        resubmissionWorkflowEnabled: Bool,
        now: Date = Date()
    ) -> Bool {
        guard detail?.submissionAllowFileUpload == true else { return false }
        guard canSubmit(
            detail: detail,
            submission: submission,
            resubmissionWorkflowEnabled: resubmissionWorkflowEnabled,
            now: now
        ) else { return false }
        return canReplaceFile(submission: submission, resubmissionWorkflowEnabled: resubmissionWorkflowEnabled)
    }

    static func canSubmit(
        detail: ModuleItemDetail?,
        submission: AssignmentSubmission?,
        resubmissionWorkflowEnabled: Bool,
        now: Date = Date()
    ) -> Bool {
        guard acceptsAnySubmission(detail) else { return false }
        if isPastAvailableUntil(detail: detail, now: now) { return false }
        if isRevisionOverdue(submission: submission, now: now) { return false }
        if submission?.resubmissionRequested == true { return true }
        let policy = detail?.lateSubmissionPolicy ?? "allow"
        if isPastDue(detail: detail, now: now) && policy == "block" {
            return false
        }
        if resubmissionWorkflowEnabled, hasAttachment(submission), submission?.resubmissionRequested != true {
            // File locked until revision requested; text-only resubmit may still be allowed elsewhere.
            return detail?.submissionAllowText == true || detail?.submissionAllowUrl == true
        }
        return true
    }

    static func submitDisabledReasonKey(
        detail: ModuleItemDetail?,
        submission: AssignmentSubmission?,
        resubmissionWorkflowEnabled: Bool,
        now: Date = Date()
    ) -> String? {
        if !acceptsAnySubmission(detail) {
            return "mobile.assignment.noSubmissionTypes"
        }
        if isPastAvailableUntil(detail: detail, now: now) {
            return "mobile.assignment.closed"
        }
        if isRevisionOverdue(submission: submission, now: now) {
            return "mobile.assignment.revisionPastDue"
        }
        let policy = detail?.lateSubmissionPolicy ?? "allow"
        if isPastDue(detail: detail, now: now) && policy == "block" && submission?.resubmissionRequested != true {
            return "mobile.assignment.pastDueBlocked"
        }
        if resubmissionWorkflowEnabled,
           hasAttachment(submission),
           submission?.resubmissionRequested != true,
           detail?.submissionAllowFileUpload == true,
           detail?.submissionAllowText != true,
           detail?.submissionAllowUrl != true {
            return "mobile.assignment.fileLocked"
        }
        return nil
    }

    static func isAllowedMimeType(_ mimeType: String?) -> Bool {
        let ct = (mimeType ?? "").lowercased()
        if ct.isEmpty || ct == "application/octet-stream" { return true }
        if ct.hasPrefix("text/") { return true }
        if ct.hasPrefix("image/") { return true }
        if ct == "application/pdf" { return true }
        return false
    }

    static func isAllowedFileSize(_ byteCount: Int64) -> Bool {
        byteCount > 0 && byteCount <= maxUploadBytes
    }

    static func gradeColumn(item: CourseStructureItem, detail: ModuleItemDetail?) -> GradeColumn {
        GradeColumn(
            id: item.id,
            kind: "assignment",
            title: detail?.title ?? item.title,
            maxPoints: detail?.pointsWorth.map(Double.init),
            dueAt: detail?.dueAt ?? item.dueAt,
            assignmentGroupId: nil,
            neverDrop: false,
            replaceWithFinal: false,
            rubric: nil
        )
    }

    static func submissionTypeLabelKeys(detail: ModuleItemDetail?) -> [String] {
        guard let detail else { return [] }
        var keys: [String] = []
        if detail.submissionAllowText == true { keys.append("mobile.assignment.typeText") }
        if detail.submissionAllowFileUpload == true { keys.append("mobile.assignment.typeFile") }
        if detail.submissionAllowUrl == true { keys.append("mobile.assignment.typeUrl") }
        return keys
    }
}
