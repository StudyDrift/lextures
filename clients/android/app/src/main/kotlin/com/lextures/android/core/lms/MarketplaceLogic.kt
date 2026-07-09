package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures

enum class MarketplacePriceFilter {
    Any,
    Free,
    Paid,
    ;

    val freeOnly: Boolean
        get() = this == Free

    val priceMax: Int?
        get() = when (this) {
            Free -> 0
            Any, Paid -> null
        }
}

enum class MarketplaceLevelFilter(val queryValue: String?) {
    Any(null),
    Beginner("beginner"),
    Intermediate("intermediate"),
    Advanced("advanced"),
}

enum class MarketplaceSortMode(val apiValue: String) {
    Popular("popular"),
    Rating("rating"),
    Newest("newest"),
    Relevance("relevance"),
    Price("price"),
}

/** Marketplace helpers (plan MKT6). Paid purchases use Path B (web hand-off). */
object MarketplaceLogic {
    const val MAX_PRICE_MAJOR = 99_999.99
    const val MIN_PAID_CENTS = 50

    val currencies = listOf(
        "usd", "eur", "gbp", "cad", "aud", "jpy", "chf", "sek", "nok", "dkk", "nzd", "sgd", "hkd", "mxn",
    )

    fun isPaid(priceCents: Int): Boolean = priceCents > 0

    fun isFree(priceCents: Int): Boolean = priceCents <= 0

    fun formatPrice(cents: Int, currency: String = "usd", freeLabel: String = "Free"): String {
        if (cents <= 0) return freeLabel
        return PathsLogic.formatPrice(cents, currency.uppercase())
    }

    fun marketplaceWebPath(slug: String): String = "/marketplace/$slug"

    fun cacheKey(
        query: String,
        category: String,
        level: MarketplaceLevelFilter,
        price: MarketplacePriceFilter,
        sort: MarketplaceSortMode,
    ): String = "$query|$category|${level.name}|${price.name}|${sort.name}"

    fun cardAccessibleName(
        title: String,
        priceLabel: String,
        owned: Boolean,
        ownedLabel: String,
    ): String = if (owned) "$title, $ownedLabel, $priceLabel" else "$title, $priceLabel"

    fun shouldShowPurchasedBadge(
        features: MobilePlatformFeatures,
        course: CourseSummary,
    ): Boolean = features.ffCourseMarketplace && course.acquiredViaMarketplace

    fun previewParagraphs(description: String, limit: Int = 3): List<String> =
        description
            .lineSequence()
            .map { it.trim() }
            .filter { it.isNotEmpty() }
            .take(limit)
            .toList()

    fun majorUnitsToPriceCents(amount: String): Int? {
        val trimmed = amount.trim()
        if (trimmed.isEmpty()) return 0
        if (!Regex("""^\d+(\.\d{1,2})?$""").matches(trimmed)) return null
        val value = trimmed.toDoubleOrNull() ?: return null
        if (value < 0 || value > MAX_PRICE_MAJOR) return null
        return Math.round(value * 100.0).toInt()
    }

    fun priceCentsToMajorUnits(priceCents: Int): String {
        if (priceCents <= 0) return ""
        return String.format("%.2f", priceCents / 100.0)
    }

    fun validateAmount(amount: String): String? {
        if (amount.trim().isEmpty()) return null
        val cents = majorUnitsToPriceCents(amount) ?: return "invalid"
        if (cents < 0) return "negative"
        if (cents > 0 && cents < MIN_PAID_CENTS) return "min"
        if (cents > Math.round(MAX_PRICE_MAJOR * 100).toInt()) return "max"
        return null
    }

    fun buildListingPutBody(
        listing: CourseCatalogListing,
        marketplaceListed: Boolean,
        priceCents: Int,
        priceCurrency: String,
    ): CourseCatalogListingPutBody = CourseCatalogListingPutBody(
        isPublic = listing.isPublic,
        category = listing.category,
        difficultyLevel = listing.difficultyLevel,
        language = listing.language,
        priceCents = priceCents,
        priceCurrency = priceCurrency,
        slug = listing.slug,
        marketplaceListed = marketplaceListed,
    )

    fun ctaLabelKey(owned: Boolean, priceCents: Int): String = when {
        owned -> "goToCourse"
        isFree(priceCents) -> "enrollFree"
        else -> "buyOnWeb"
    }
}
