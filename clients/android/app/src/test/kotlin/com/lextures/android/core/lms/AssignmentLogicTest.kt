package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.Instant
import java.util.Date

class AssignmentLogicTest {
    @Test
    fun acceptsAnySubmission() {
        val detail = ModuleItemDetail(submissionAllowText = true)
        assertTrue(AssignmentLogic.acceptsAnySubmission(detail))
        assertFalse(AssignmentLogic.acceptsAnySubmission(null))
    }

    @Test
    fun fileLockedWhenWorkflowEnabledAndHasAttachment() {
        val detail = ModuleItemDetail(submissionAllowFileUpload = true)
        val submission = AssignmentSubmission(
            id = "s1",
            attachmentFilename = "work.pdf",
            submittedAt = "2026-01-01T00:00:00Z",
        )
        assertFalse(
            AssignmentLogic.canSubmitFile(detail, submission, resubmissionWorkflowEnabled = true),
        )
    }

    @Test
    fun lateSubmissionBlocked() {
        val pastDue = Instant.now().minusSeconds(3600).toString()
        val detail = ModuleItemDetail(
            dueAt = pastDue,
            lateSubmissionPolicy = "block",
            submissionAllowText = true,
        )
        assertFalse(AssignmentLogic.canSubmit(detail, null, resubmissionWorkflowEnabled = false))
        assertEquals(
            "mobile.assignment.pastDueBlocked",
            AssignmentLogic.submitDisabledReasonKey(detail, null, resubmissionWorkflowEnabled = false),
        )
    }

    @Test
    fun allowedMimeTypes() {
        assertTrue(AssignmentLogic.isAllowedMimeType("image/jpeg"))
        assertTrue(AssignmentLogic.isAllowedMimeType("application/pdf"))
        assertFalse(AssignmentLogic.isAllowedMimeType("application/zip"))
    }

    @Test
    fun maxUploadSize() {
        assertTrue(AssignmentLogic.isAllowedFileSize(1024))
        assertFalse(AssignmentLogic.isAllowedFileSize(AssignmentLogic.MAX_UPLOAD_BYTES + 1))
    }
}
