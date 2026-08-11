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

/** Android purchase decision for a coupon preview (MKTC.6). */
sealed class AndroidPurchaseRoute {
    data object FreeGrant : AndroidPurchaseRoute()
    data object StripeCheckout : AndroidPurchaseRoute()
    data object Reject : AndroidPurchaseRoute()
    data object FlagOff : AndroidPurchaseRoute()
}

/** Marketplace helpers (MKT6 / MOB.7 / MKTC.6). Paid path: Stripe checkout handoff when flagged. */
object MarketplaceLogic {
    const val MAX_PRICE_MAJOR = 99_999.99
    const val MIN_PAID_CENTS = 50
    /** Max characters retained from a URL or typed code (server normalizes further). */
    const val COUPON_INPUT_MAX_LEN = 32

    private val COUPON_REASON_KEYS = setOf(
        "ok",
        "not_found",
        "inactive",
        "not_started",
        "expired",
        "exhausted",
        "already_used",
        "currency_mismatch",
        "course_free",
        "owned",
    )

    val currencies = listOf(
        "usd", "eur", "gbp", "cad", "aud", "jpy", "chf", "sek", "nok", "dkk", "nzd", "sgd", "hkd", "mxn",
    )

    /** In-app claim/buy + Purchased courses library (MOB.7). Default off via platform flag. */
    fun purchaseEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffCourseMarketplace && features.ffMobileMarketplacePurchase

    /** Coupons UI + preview when marketplace and coupons flags are both on (MKTC.6). */
    fun couponsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffCourseMarketplace && features.ffCourseCoupons

    fun isPaid(priceCents: Int): Boolean = priceCents > 0

    fun isFree(priceCents: Int): Boolean = priceCents <= 0

    fun formatPrice(cents: Int, currency: String = "usd", freeLabel: String = "Free"): String {
        if (cents <= 0) return freeLabel
        return PathsLogic.formatPrice(cents, currency.uppercase())
    }

    fun marketplaceWebPath(slug: String): String = "/marketplace/$slug"

    /** Upper-case, strip whitespace, drop non [A-Z0-9_-], truncate to [COUPON_INPUT_MAX_LEN]. */
    fun normalizeCouponCode(raw: String): String {
        val upper = raw.replace(Regex("\\s+"), "").uppercase()
        val filtered = upper.replace(Regex("[^A-Z0-9_-]"), "")
        return filtered.take(COUPON_INPUT_MAX_LEN)
    }

    /**
     * Validate a raw deep-link/query coupon before use (length ≤ 32, usable charset after normalize).
     * Empty / oversized / only-illegal characters → false.
     */
    fun isValidCouponParam(raw: String?): Boolean {
        val trimmed = raw?.trim().orEmpty()
        if (trimmed.isEmpty() || trimmed.length > COUPON_INPUT_MAX_LEN) return false
        return normalizeCouponCode(trimmed).isNotEmpty()
    }

    /**
     * Read `?coupon=` from a full URL, path+query, or bare query string.
     * Returns a normalized code or null when missing/invalid.
     */
    fun parseCouponFromQuery(rawUrlOrQuery: String): String? {
        val raw = rawUrlOrQuery.trim()
        if (raw.isEmpty()) return null
        val query = when {
            raw.contains('?') -> raw.substringAfter('?').substringBefore('#')
            raw.contains('=') && !raw.contains("://") && !raw.startsWith("/") -> raw.removePrefix("?")
            else -> {
                runCatching {
                    val uriString = when {
                        raw.startsWith("lextures://") -> {
                            val stripped = raw.removePrefix("lextures://")
                            "https://lextures.com/" + stripped.trimStart('/')
                        }
                        raw.startsWith("http://") || raw.startsWith("https://") -> raw
                        raw.startsWith("/") -> "https://lextures.com$raw"
                        else -> return null
                    }
                    java.net.URI(uriString).rawQuery
                }.getOrNull() ?: return null
            }
        }
        if (query.isNullOrBlank()) return null
        var found: String? = null
        for (part in query.split('&')) {
            if (part.isEmpty()) continue
            val eq = part.indexOf('=')
            val key = if (eq >= 0) part.substring(0, eq) else part
            val value = if (eq >= 0) part.substring(eq + 1) else ""
            if (key.equals("coupon", ignoreCase = true) && value.isNotEmpty()) {
                found = runCatching {
                    java.net.URLDecoder.decode(value, Charsets.UTF_8.name())
                }.getOrDefault(value)
                break
            }
        }
        if (found == null || !isValidCouponParam(found)) return null
        return normalizeCouponCode(found)
    }

    /** Map a server reason token to `mobile.marketplace.coupon.reason.*` key. */
    fun couponReasonKey(reason: String?): String {
        val r = reason?.lowercase()?.trim().orEmpty().ifEmpty { "not_found" }
        val key = if (r in COUPON_REASON_KEYS) r else "not_found"
        return "mobile.marketplace.coupon.reason.$key"
    }

    /**
     * Pure Android purchase branch for a coupon preview.
     * freeGrant | stripeCheckout | reject | flagOff
     */
    fun purchaseRoute(
        preview: CouponPreview?,
        couponsOn: Boolean,
        marketplaceOn: Boolean = true,
    ): AndroidPurchaseRoute {
        if (!marketplaceOn || !couponsOn) return AndroidPurchaseRoute.FlagOff
        if (preview == null || !preview.applied) return AndroidPurchaseRoute.Reject
        return if (preview.freeAfterDiscount || preview.chargedCents <= 0) {
            AndroidPurchaseRoute.FreeGrant
        } else {
            AndroidPurchaseRoute.StripeCheckout
        }
    }

    fun parseCouponReasonFromBody(body: String): String? {
        if (body.isBlank()) return null
        return runCatching {
            val el = kotlinx.serialization.json.Json.parseToJsonElement(body)
            val obj = el as? kotlinx.serialization.json.JsonObject ?: return@runCatching null
            obj["reason"]?.let {
                (it as? kotlinx.serialization.json.JsonPrimitive)?.content?.takeIf { c -> c.isNotBlank() }
            }
        }.getOrNull()
    }

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

    fun majorUnitsToPriceCents(amount: String, currency: String = "usd"): Int? {
        val trimmed = amount.trim()
        if (trimmed.isEmpty()) return 0
        val pattern = if (CurrencyExponent.isZeroDecimal(currency)) """^\d+$""" else """^\d+(\.\d{1,2})?$"""
        if (!Regex(pattern).matches(trimmed)) return null
        val value = trimmed.toDoubleOrNull() ?: return null
        val maxMajor = if (CurrencyExponent.isZeroDecimal(currency)) {
            CurrencyExponent.MAX_PRICE_MAJOR_ZERO_DECIMAL
        } else {
            MAX_PRICE_MAJOR
        }
        if (value < 0 || value > maxMajor) return null
        return CurrencyExponent.majorUnitsToMinorUnits(value, currency)
    }

    fun priceCentsToMajorUnits(priceCents: Int, currency: String = "usd"): String {
        if (priceCents <= 0) return ""
        val major = CurrencyExponent.minorUnitsToMajorUnits(priceCents, currency)
        return if (CurrencyExponent.isZeroDecimal(currency)) {
            Math.round(major).toString()
        } else {
            String.format("%.2f", major)
        }
    }

    fun validateAmount(amount: String, currency: String = "usd"): String? {
        if (amount.trim().isEmpty()) return null
        val cents = majorUnitsToPriceCents(amount, currency) ?: return "invalid"
        if (cents < 0) return "negative"
        if (cents > 0 && cents < MIN_PAID_CENTS) return "min"
        if (cents > CurrencyExponent.maxCatalogMinorUnits(currency)) return "max"
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

    fun ctaLabelKey(owned: Boolean, priceCents: Int, purchaseEnabled: Boolean = false): String = when {
        owned -> "goToCourse"
        isFree(priceCents) -> "enrollFree"
        purchaseEnabled -> "buy"
        else -> "buyOnWeb"
    }

    fun purchaseSourceLabelKey(source: String): String = when (source) {
        "free" -> "mobile.marketplace.purchases.source.free"
        "stripe" -> "mobile.marketplace.purchases.source.stripe"
        "comp" -> "mobile.marketplace.purchases.source.comp"
        else -> "mobile.marketplace.purchases.source.other"
    }

    fun formatAcquiredAt(iso: String): String {
        val trimmed = iso.trim()
        return if (trimmed.length >= 10) trimmed.take(10) else trimmed
    }
}

object MarketplaceObservability {
    private val counters = java.util.concurrent.ConcurrentHashMap<String, Int>()

    fun record(event: String, attributes: Map<String, String> = emptyMap()) {
        val key = if (attributes.isEmpty()) {
            event
        } else {
            event + "|" + attributes.toSortedMap().entries.joinToString(",") { "${it.key}=${it.value}" }
        }
        counters.merge(key, 1, Int::plus)
    }

    fun count(event: String): Int =
        counters.entries
            .filter { it.key == event || it.key.startsWith("$event|") }
            .sumOf { it.value }

    fun resetForTests() {
        counters.clear()
    }
}
