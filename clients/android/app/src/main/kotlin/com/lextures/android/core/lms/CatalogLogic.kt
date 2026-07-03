package com.lextures.android.core.lms

enum class CatalogBrowseTab {
    Courses,
    Paths,
}

enum class CatalogPriceFilter {
    Any,
    Free,
    Paid,
    ;

    val priceMax: Int?
        get() = when (this) {
            Free -> 0
            Any, Paid -> null
        }
}

enum class CatalogLevelFilter(val queryValue: String?) {
    Any(null),
    Beginner("beginner"),
    Intermediate("intermediate"),
    Advanced("advanced"),
}

enum class CatalogSortMode(val apiValue: String) {
    Popular("popular"),
    Rating("rating"),
    Newest("newest"),
    Relevance("relevance"),
}

object CatalogLogic {
    fun isPaid(priceCents: Int): Boolean = priceCents > 0

    fun isFree(priceCents: Int): Boolean = priceCents <= 0

    fun formatPrice(cents: Int): String =
        if (cents <= 0) "Free" else PathsLogic.formatPrice(cents)

    fun catalogWebPath(slug: String): String = "/explore/$slug"

    fun cacheKey(
        query: String,
        category: String,
        level: CatalogLevelFilter,
        price: CatalogPriceFilter,
        sort: CatalogSortMode,
    ): String = "$query|$category|${level.name}|${price.name}|${sort.name}"

    fun isEnrolled(courseCode: String, courses: List<CourseSummary>): Boolean {
        val code = courseCode.trim().lowercase()
        if (code.isEmpty()) return false
        return courses.any { it.courseCode.trim().lowercase() == code }
    }

    fun enrolledCourse(courseCode: String, courses: List<CourseSummary>): CourseSummary? {
        val code = courseCode.trim().lowercase()
        return courses.firstOrNull { it.courseCode.trim().lowercase() == code }
    }

    fun previewParagraphs(description: String, limit: Int = 3): List<String> =
        description
            .lineSequence()
            .map { it.trim() }
            .filter { it.isNotEmpty() }
            .take(limit)
            .toList()
}