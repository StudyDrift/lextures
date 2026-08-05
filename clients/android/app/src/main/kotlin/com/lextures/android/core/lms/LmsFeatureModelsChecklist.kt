package com.lextures.android.core.lms

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// region Course checklist (CC.9)

@Serializable
enum class ChecklistStatus {
    @SerialName("done") Done,
    @SerialName("todo") Todo,
    @SerialName("in_progress") InProgress,
    @SerialName("unknown") Unknown,
    @SerialName("not_applicable") NotApplicable,
}

@Serializable
enum class ChecklistTier {
    @SerialName("essential") Essential,
    @SerialName("recommended") Recommended,
}

@Serializable
enum class ChecklistDismissReason {
    @SerialName("not_applicable") NotApplicable,
    @SerialName("done_elsewhere") DoneElsewhere,
    @SerialName("disagree") Disagree,
    @SerialName("later") Later,
    @SerialName("other") Other,
    ;

    val wireValue: String
        get() = when (this) {
            NotApplicable -> "not_applicable"
            DoneElsewhere -> "done_elsewhere"
            Disagree -> "disagree"
            Later -> "later"
            Other -> "other"
        }
}

@Serializable
data class ChecklistNavTarget(
    val route: String,
    val anchor: String? = null,
    val entityKey: String? = null,
)

@Serializable
data class ChecklistEvidenceRow(
    val label: String,
    val sublabel: String? = null,
    val status: String = "",
    val target: ChecklistNavTarget? = null,
)

@Serializable
data class ChecklistEvidence(
    val columns: List<String> = emptyList(),
    val rows: List<ChecklistEvidenceRow> = emptyList(),
    val truncatedAt: Int? = null,
)

@Serializable
data class ChecklistDismissal(
    val dismissedAt: String = "",
    val byUserId: String = "",
    val byDisplayName: String = "",
    val reason: String = "",
    val note: String = "",
)

@Serializable
data class ChecklistProgress(
    val done: Int = 0,
    val total: Int = 0,
)

/** Optional assisted-fix primary action (CC.10 FR-5). Unknown kinds render nothing. */
@Serializable
data class ChecklistAction(
    val kind: String = "",
    val labelKey: String = "",
    val label: String = "",
    val endpoint: String = "",
    val requiresAi: Boolean = true,
)

@Serializable
data class ChecklistItem(
    val id: String,
    val titleKey: String = "",
    val title: String = "",
    val whyKey: String = "",
    val why: String = "",
    val tier: ChecklistTier = ChecklistTier.Recommended,
    val status: String = "unknown",
    val detail: String? = null,
    val progress: ChecklistProgress? = null,
    val sources: List<String> = emptyList(),
    val helpRef: String? = null,
    val target: ChecklistNavTarget? = null,
    val evidence: ChecklistEvidence? = null,
    val action: ChecklistAction? = null,
    val dismissal: ChecklistDismissal? = null,
)

@Serializable
data class ChecklistCategory(
    val id: String,
    val titleKey: String = "",
    val title: String = "",
    val items: List<ChecklistItem> = emptyList(),
)

@Serializable
data class CourseChecklistSummary(
    val outstandingEssential: Int = 0,
    val outstandingTotal: Int = 0,
    val done: Int = 0,
    val total: Int = 0,
    val dismissed: Int = 0,
    val computedAt: String = "",
    val stale: Boolean = false,
)

@Serializable
data class CourseChecklist(
    val courseCode: String = "",
    val engineVersion: Int = 0,
    val catalogVersion: String = "",
    val computedAt: String = "",
    val stale: Boolean = false,
    val evidenceTruncated: Boolean = false,
    val summary: CourseChecklistSummary = CourseChecklistSummary(),
    val categories: List<ChecklistCategory> = emptyList(),
    val dismissed: List<ChecklistItem> = emptyList(),
)

@Serializable
data class ChecklistDismissBody(
    val reason: String,
    val note: String = "",
)

// endregion
