package com.lextures.android.core.lms

import com.lextures.android.R

object PeerReviewLogic {
    fun targetLabel(allocation: PeerReviewAllocation): String? =
        allocation.targetLabel?.takeIf { it.isNotBlank() }

    fun isComplete(allocation: PeerReviewAllocation): Boolean =
        allocation.status == PeerReviewAllocationStatus.submitted

    fun pending(allocations: List<PeerReviewAllocation>): List<PeerReviewAllocation> =
        allocations.filterNot(::isComplete)

    fun completedCount(allocations: List<PeerReviewAllocation>): Int =
        allocations.count(::isComplete)

    fun rubricTotal(rubric: RubricDefinition, scores: Map<String, Double>): Double =
        rubric.criteria.sumOf { scores[it.id] ?: 0.0 }

    fun rubricGradedCount(rubric: RubricDefinition, scores: Map<String, Double>): Int =
        rubric.criteria.count { scores.containsKey(it.id) }

    fun rubricScoresComplete(rubric: RubricDefinition, scores: Map<String, Double>): Boolean {
        if (rubric.criteria.isEmpty()) return true
        return rubricGradedCount(rubric, scores) == rubric.criteria.size
    }

    fun statusLabelRes(status: PeerReviewAllocationStatus): Int = when (status) {
        PeerReviewAllocationStatus.assigned -> R.string.mobile_peerReview_statusAssigned
        PeerReviewAllocationStatus.in_progress -> R.string.mobile_peerReview_statusInProgress
        PeerReviewAllocationStatus.submitted -> R.string.mobile_peerReview_statusSubmitted
        PeerReviewAllocationStatus.expired -> R.string.mobile_peerReview_statusExpired
    }

    fun cacheKeyAssigned(): String = "peer-review:assigned"

    fun cacheKeyAllocation(id: String): String = "peer-review:allocation:$id"

    fun cacheKeyReceived(courseCode: String, assignmentId: String): String =
        "peer-review:received:$courseCode:$assignmentId"
}
