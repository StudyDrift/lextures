package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PeerReviewLogicTest {
    @Test
    fun pendingExcludesSubmitted() {
        val allocations = listOf(
            allocation("a", PeerReviewAllocationStatus.assigned),
            allocation("b", PeerReviewAllocationStatus.submitted),
        )
        assertEquals(listOf("a"), PeerReviewLogic.pending(allocations).map { it.id })
        assertEquals(1, PeerReviewLogic.completedCount(allocations))
    }

    @Test
    fun rubricTotalAndCompletion() {
        val rubric = RubricDefinition(
            title = "Essay",
            criteria = listOf(
                RubricCriterion(
                    id = "c1",
                    title = "Thesis",
                    description = null,
                    levels = listOf(RubricLevel(label = "Good", points = 4.0, description = null)),
                ),
                RubricCriterion(
                    id = "c2",
                    title = "Evidence",
                    description = null,
                    levels = listOf(RubricLevel(label = "Good", points = 6.0, description = null)),
                ),
            ),
        )
        val partial = mapOf("c1" to 4.0)
        assertEquals(4.0, PeerReviewLogic.rubricTotal(rubric, partial), 0.001)
        assertFalse(PeerReviewLogic.rubricScoresComplete(rubric, partial))
        val complete = mapOf("c1" to 4.0, "c2" to 6.0)
        assertEquals(10.0, PeerReviewLogic.rubricTotal(rubric, complete), 0.001)
        assertTrue(PeerReviewLogic.rubricScoresComplete(rubric, complete))
    }

    private fun allocation(id: String, status: PeerReviewAllocationStatus) = PeerReviewAllocation(
        id = id,
        configId = "cfg",
        assignmentId = "assign",
        courseId = "course",
        courseCode = "C-101",
        targetSubmissionId = "sub",
        status = status,
        assignedAt = "2024-01-01T00:00:00Z",
        anonymity = PeerReviewAnonymity.double_blind,
    )
}
