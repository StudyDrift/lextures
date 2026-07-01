package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
enum class PeerReviewAnonymity {
    @Suppress("unused") double_blind,
    @Suppress("unused") reviewer_anon,
    @Suppress("unused") named,
}

@Serializable
enum class PeerReviewAllocationStatus {
    @Suppress("unused") assigned,
    @Suppress("unused") in_progress,
    @Suppress("unused") submitted,
    @Suppress("unused") expired,
}

@Serializable
data class PeerReviewAllocation(
    val id: String,
    val configId: String,
    val assignmentId: String,
    val courseId: String,
    val courseCode: String,
    val targetSubmissionId: String,
    val status: PeerReviewAllocationStatus,
    val assignedAt: String,
    val anonymity: PeerReviewAnonymity,
    val targetLabel: String? = null,
    val targetUserId: String? = null,
    val closesAt: String? = null,
    val opensAt: String? = null,
)

@Serializable
data class PeerReviewAssignedResponse(
    val allocations: List<PeerReviewAllocation> = emptyList(),
)

@Serializable
data class PeerReviewReceivedItem(
    val id: String,
    val score: Double? = null,
    val comments: String? = null,
    val submittedAt: String,
    val reviewerLabel: String? = null,
)

@Serializable
data class PeerReviewReceivedResponse(
    val reviews: List<PeerReviewReceivedItem> = emptyList(),
)

@Serializable
data class PeerReviewExistingReview(
    val id: String,
    val allocationId: String,
    val score: Double? = null,
    val rubricScores: Map<String, Double>? = null,
    val comments: String? = null,
    val submittedAt: String,
)

@Serializable
data class PeerReviewAllocationDetail(
    val allocation: PeerReviewAllocation,
    val assignmentTitle: String? = null,
    val submission: AssignmentSubmission,
    val rubric: RubricDefinition? = null,
    val review: PeerReviewExistingReview? = null,
)

@Serializable
data class PeerReviewSubmitRequest(
    val score: Double? = null,
    val rubricScores: Map<String, Double>? = null,
    val comments: String? = null,
)

@Serializable
data class PeerReviewSubmitResponse(
    val id: String,
    val allocationId: String,
    val score: Double? = null,
    val submittedAt: String,
)
