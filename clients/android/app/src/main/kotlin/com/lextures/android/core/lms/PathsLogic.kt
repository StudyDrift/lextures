package com.lextures.android.core.lms

object PathsLogic {
    val recommendationSurfaces = listOf("continue", "review", "strengthen", "challenge")

    fun sortedCourses(courses: List<PathCourseProgress>): List<PathCourseProgress> =
        courses.sortedBy { it.position }

    fun nextCourse(progress: PathProgress): PathCourseProgress? =
        sortedCourses(progress.courses).firstOrNull { !it.isCompleted }

    fun isLocked(course: PathCourseProgress): Boolean =
        !course.isCompleted && !course.isRecommended

    fun isPaid(bundlePriceCents: Int?): Boolean =
        bundlePriceCents != null && bundlePriceCents > 0

    fun catalogWebPath(slug: String): String = "/paths/$slug"

    fun formatPrice(cents: Int, currency: String = "USD"): String {
        val formatter = java.text.NumberFormat.getCurrencyInstance()
        formatter.currency = java.util.Currency.getInstance(currency)
        return formatter.format(cents / 100.0)
    }

    fun formatDuration(minutes: Int): String = when {
        minutes < 60 -> "$minutes min"
        minutes % 60 == 0 -> "${minutes / 60} hr"
        else -> "${minutes / 60} hr ${minutes % 60} min"
    }

    data class MergedRecommendations(
        val primary: LearnerRecommendationItem?,
        val chips: List<LearnerRecommendationItem>,
        val degraded: Boolean,
    )

    fun mergeRecommendations(responses: List<LearnerRecommendationsResponse>): MergedRecommendations {
        val merged = responses.flatMap { it.recommendations }.sortedByDescending { it.score }
        return MergedRecommendations(
            primary = merged.firstOrNull(),
            chips = merged.drop(1).take(3),
            degraded = responses.any { it.degraded == true },
        )
    }

    fun structureItemKind(itemType: String): String? = when (itemType) {
        "quiz" -> "quiz"
        "content_page" -> "content_page"
        "assignment" -> "assignment"
        "external_link" -> "external_link"
        else -> null
    }

    fun surfaceLabelRes(surface: String): Int = when (surface) {
        "continue" -> com.lextures.android.R.string.mobile_paths_surface_continue
        "strengthen" -> com.lextures.android.R.string.mobile_paths_surface_strengthen
        "challenge" -> com.lextures.android.R.string.mobile_paths_surface_challenge
        "review" -> com.lextures.android.R.string.mobile_paths_surface_review
        else -> com.lextures.android.R.string.mobile_paths_surface_other
    }

    fun structureItem(
        recommendation: LearnerRecommendationItem,
        items: List<CourseStructureItem>,
    ): CourseStructureItem? {
        items.firstOrNull { it.id == recommendation.itemId }?.let { return it }
        val kind = structureItemKind(recommendation.itemType) ?: return null
        return CourseStructureItem(
            id = recommendation.itemId,
            sortOrder = 0,
            kind = kind,
            title = recommendation.title,
            parentId = null,
            published = true,
            dueAt = null,
            pointsWorth = null,
            pointsPossible = null,
        )
    }
}