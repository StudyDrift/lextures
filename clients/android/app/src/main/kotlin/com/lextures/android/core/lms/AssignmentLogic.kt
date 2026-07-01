package com.lextures.android.core.lms

import java.time.Instant

/** Pure assignment submission rules (M5.1). */
enum class AssignmentSubmissionStatus {
    NotStarted,
    Submitted,
    Late,
    Graded,
    RevisionRequested,
}

object AssignmentLogic {
    const val MAX_UPLOAD_BYTES: Long = 20L * 1024 * 1024

    fun draftStorageKey(courseCode: String, itemId: String): String =
        "assignment-draft:$courseCode:$itemId"

    fun acceptsAnySubmission(detail: ModuleItemDetail?): Boolean {
        if (detail == null) return false
        return detail.submissionAllowText == true ||
            detail.submissionAllowFileUpload == true ||
            detail.submissionAllowUrl == true
    }

    fun status(
        submission: AssignmentSubmission?,
        grade: SubmissionGrade?,
        detail: ModuleItemDetail?,
        now: Instant = Instant.now(),
    ): AssignmentSubmissionStatus {
        if (submission?.resubmissionRequested == true) return AssignmentSubmissionStatus.RevisionRequested
        if (grade?.posted == true) return AssignmentSubmissionStatus.Graded
        if (submission != null) {
            if (isLate(detail, submission.submittedAt, now)) return AssignmentSubmissionStatus.Late
            return AssignmentSubmissionStatus.Submitted
        }
        return AssignmentSubmissionStatus.NotStarted
    }

    fun isPastDue(detail: ModuleItemDetail?, now: Instant = Instant.now()): Boolean {
        val due = LmsDates.parse(detail?.dueAt ?: detail?.availableUntil) ?: return false
        return due.isBefore(now)
    }

    fun isPastAvailableUntil(detail: ModuleItemDetail?, now: Instant = Instant.now()): Boolean {
        val until = LmsDates.parse(detail?.availableUntil) ?: return false
        return until.isBefore(now)
    }

    fun isRevisionOverdue(submission: AssignmentSubmission?, now: Instant = Instant.now()): Boolean {
        if (submission?.resubmissionRequested != true) return false
        val due = LmsDates.parse(submission.revisionDueAt) ?: return false
        return due.isBefore(now)
    }

    fun isLate(detail: ModuleItemDetail?, submittedAt: String?, now: Instant = Instant.now()): Boolean {
        val due = LmsDates.parse(detail?.dueAt) ?: return false
        val submitted = LmsDates.parse(submittedAt) ?: return false
        return submitted.isAfter(due)
    }

    fun hasAttachment(submission: AssignmentSubmission?): Boolean {
        if (submission == null) return false
        if (!submission.attachmentFilename.isNullOrBlank()) return true
        if (!submission.attachmentContentPath.isNullOrBlank()) return true
        return false
    }

    fun canReplaceFile(submission: AssignmentSubmission?, resubmissionWorkflowEnabled: Boolean): Boolean {
        if (submission?.resubmissionRequested == true) return true
        if (!resubmissionWorkflowEnabled) return true
        return !hasAttachment(submission)
    }

    fun canSubmitText(
        detail: ModuleItemDetail?,
        submission: AssignmentSubmission?,
        resubmissionWorkflowEnabled: Boolean,
        now: Instant = Instant.now(),
    ): Boolean {
        if (detail?.submissionAllowText != true) return false
        return canSubmit(detail, submission, resubmissionWorkflowEnabled, now)
    }

    fun canSubmitFile(
        detail: ModuleItemDetail?,
        submission: AssignmentSubmission?,
        resubmissionWorkflowEnabled: Boolean,
        now: Instant = Instant.now(),
    ): Boolean {
        if (detail?.submissionAllowFileUpload != true) return false
        if (!canSubmit(detail, submission, resubmissionWorkflowEnabled, now)) return false
        return canReplaceFile(submission, resubmissionWorkflowEnabled)
    }

    fun canSubmit(
        detail: ModuleItemDetail?,
        submission: AssignmentSubmission?,
        resubmissionWorkflowEnabled: Boolean,
        now: Instant = Instant.now(),
    ): Boolean {
        if (!acceptsAnySubmission(detail)) return false
        if (isPastAvailableUntil(detail, now)) return false
        if (isRevisionOverdue(submission, now)) return false
        if (submission?.resubmissionRequested == true) return true
        val policy = detail?.lateSubmissionPolicy ?: "allow"
        if (isPastDue(detail, now) && policy == "block") return false
        if (resubmissionWorkflowEnabled && hasAttachment(submission) && submission?.resubmissionRequested != true) {
            return detail?.submissionAllowText == true || detail?.submissionAllowUrl == true
        }
        return true
    }

    fun submitDisabledReasonKey(
        detail: ModuleItemDetail?,
        submission: AssignmentSubmission?,
        resubmissionWorkflowEnabled: Boolean,
        now: Instant = Instant.now(),
    ): String? {
        if (!acceptsAnySubmission(detail)) return "mobile.assignment.noSubmissionTypes"
        if (isPastAvailableUntil(detail, now)) return "mobile.assignment.closed"
        if (isRevisionOverdue(submission, now)) return "mobile.assignment.revisionPastDue"
        val policy = detail?.lateSubmissionPolicy ?: "allow"
        if (isPastDue(detail, now) && policy == "block" && submission?.resubmissionRequested != true) {
            return "mobile.assignment.pastDueBlocked"
        }
        if (resubmissionWorkflowEnabled &&
            hasAttachment(submission) &&
            submission?.resubmissionRequested != true &&
            detail?.submissionAllowFileUpload == true &&
            detail?.submissionAllowText != true &&
            detail?.submissionAllowUrl != true
        ) {
            return "mobile.assignment.fileLocked"
        }
        return null
    }

    fun isAllowedMimeType(mimeType: String?): Boolean {
        val ct = mimeType?.lowercase().orEmpty()
        if (ct.isEmpty() || ct == "application/octet-stream") return true
        if (ct.startsWith("text/")) return true
        if (ct.startsWith("image/")) return true
        if (ct == "application/pdf") return true
        return false
    }

    fun isAllowedFileSize(byteCount: Long): Boolean =
        byteCount > 0 && byteCount <= MAX_UPLOAD_BYTES

    fun gradeColumn(item: CourseStructureItem, detail: ModuleItemDetail?): GradeColumn =
        GradeColumn(
            id = item.id,
            kind = "assignment",
            title = detail?.title ?: item.title,
            maxPoints = detail?.pointsWorth?.toDouble(),
            dueAt = detail?.dueAt ?: item.dueAt,
        )
}
