package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class MarketplaceLogicTest {
    @Before
    fun setUp() {
        MarketplaceObservability.resetForTests()
    }

    @Test
    fun isPaidAndFree() {
        assertTrue(MarketplaceLogic.isPaid(1999))
        assertFalse(MarketplaceLogic.isPaid(0))
        assertTrue(MarketplaceLogic.isFree(0))
        assertFalse(MarketplaceLogic.isFree(500))
    }

    @Test
    fun cardAccessibleName() {
        assertEquals(
            "Spanish, Owned, Free",
            MarketplaceLogic.cardAccessibleName("Spanish", "Free", owned = true, ownedLabel = "Owned"),
        )
        assertEquals(
            "Spanish, $19.99",
            MarketplaceLogic.cardAccessibleName("Spanish", "$19.99", owned = false, ownedLabel = "Owned"),
        )
    }

    @Test
    fun shouldShowPurchasedBadge() {
        val course = CourseSummary(
            id = "1",
            courseCode = "SPAN101",
            title = "Spanish",
            description = "",
            acquiredViaMarketplace = true,
        )
        assertFalse(MarketplaceLogic.shouldShowPurchasedBadge(MobilePlatformFeatures(), course))
        assertTrue(
            MarketplaceLogic.shouldShowPurchasedBadge(
                MobilePlatformFeatures(ffCourseMarketplace = true),
                course,
            ),
        )
        assertFalse(
            MarketplaceLogic.shouldShowPurchasedBadge(
                MobilePlatformFeatures(ffCourseMarketplace = true),
                course.copy(acquiredViaMarketplace = false),
            ),
        )
    }

    @Test
    fun majorUnitsAndValidation() {
        assertEquals(0, MarketplaceLogic.majorUnitsToPriceCents(""))
        assertEquals(1999, MarketplaceLogic.majorUnitsToPriceCents("19.99"))
        assertEquals(1000, MarketplaceLogic.majorUnitsToPriceCents("1000", "jpy"))
        assertNull(MarketplaceLogic.majorUnitsToPriceCents("abc"))
        assertNull(MarketplaceLogic.majorUnitsToPriceCents("1000.50", "jpy"))
        assertNull(MarketplaceLogic.validateAmount(""))
        assertEquals("invalid", MarketplaceLogic.validateAmount("12.345"))
        assertEquals("min", MarketplaceLogic.validateAmount("0.10"))
        assertNull(MarketplaceLogic.validateAmount("19.99"))
    }

    @Test
    fun ctaAndWebPath() {
        assertEquals("goToCourse", MarketplaceLogic.ctaLabelKey(owned = true, priceCents = 0))
        assertEquals("enrollFree", MarketplaceLogic.ctaLabelKey(owned = false, priceCents = 0))
        assertEquals("buyOnWeb", MarketplaceLogic.ctaLabelKey(owned = false, priceCents = 500))
        assertEquals(
            "buy",
            MarketplaceLogic.ctaLabelKey(owned = false, priceCents = 500, purchaseEnabled = true),
        )
        assertEquals("/marketplace/spanish-a1", MarketplaceLogic.marketplaceWebPath("spanish-a1"))
    }

    @Test
    fun purchaseEnabledRequiresBothFlags() {
        assertFalse(MarketplaceLogic.purchaseEnabled(MobilePlatformFeatures()))
        assertFalse(
            MarketplaceLogic.purchaseEnabled(MobilePlatformFeatures(ffCourseMarketplace = true, ffMobileMarketplacePurchase = false)),
        )
        assertTrue(
            MarketplaceLogic.purchaseEnabled(
                MobilePlatformFeatures(ffCourseMarketplace = true, ffMobileMarketplacePurchase = true),
            ),
        )
    }

    @Test
    fun purchaseSourceAndAcquiredFormatting() {
        assertEquals(
            "mobile.marketplace.purchases.source.free",
            MarketplaceLogic.purchaseSourceLabelKey("free"),
        )
        assertEquals(
            "mobile.marketplace.purchases.source.stripe",
            MarketplaceLogic.purchaseSourceLabelKey("stripe"),
        )
        assertEquals("2026-07-19", MarketplaceLogic.formatAcquiredAt("2026-07-19T12:00:00Z"))
    }

    @Test
    fun observabilityCounters() {
        MarketplaceObservability.record("marketplace_viewed")
        MarketplaceObservability.record("marketplace_claim", mapOf("already_owned" to "0"))
        assertEquals(1, MarketplaceObservability.count("marketplace_viewed"))
        assertEquals(1, MarketplaceObservability.count("marketplace_claim"))
    }

    @Test
    fun normalizeCouponCode() {
        assertEquals("LAUNCH25", MarketplaceLogic.normalizeCouponCode("  launch 25 "))
        assertEquals("ABSCRIPTCD", MarketplaceLogic.normalizeCouponCode("ab<script>!@#cd"))
        assertEquals("A".repeat(32), MarketplaceLogic.normalizeCouponCode("A".repeat(40)))
        assertEquals("SAVE_10-NOW", MarketplaceLogic.normalizeCouponCode("save_10-now"))
    }

    @Test
    fun isValidCouponParam() {
        assertTrue(MarketplaceLogic.isValidCouponParam("LAUNCH25"))
        assertTrue(MarketplaceLogic.isValidCouponParam("save_10"))
        assertFalse(MarketplaceLogic.isValidCouponParam(""))
        assertFalse(MarketplaceLogic.isValidCouponParam(null))
        assertFalse(MarketplaceLogic.isValidCouponParam("!!!"))
        assertFalse(MarketplaceLogic.isValidCouponParam("A".repeat(33)))
    }

    @Test
    fun parseCouponFromQuery() {
        assertEquals(
            "LAUNCH25",
            MarketplaceLogic.parseCouponFromQuery("/marketplace/spanish-a1?coupon=launch25"),
        )
        assertEquals(
            "SAVE10",
            MarketplaceLogic.parseCouponFromQuery("https://lextures.com/marketplace/cs101?coupon=SAVE10&x=1"),
        )
        assertEquals(
            "FREE100",
            MarketplaceLogic.parseCouponFromQuery("lextures://marketplace/slug?coupon=free100"),
        )
        assertEquals("ABC", MarketplaceLogic.parseCouponFromQuery("?COUPON=abc"))
        assertNull(MarketplaceLogic.parseCouponFromQuery("/marketplace/slug"))
        assertNull(MarketplaceLogic.parseCouponFromQuery("/marketplace/slug?coupon="))
        assertNull(MarketplaceLogic.parseCouponFromQuery("/marketplace/slug?coupon=" + "X".repeat(40)))
    }

    @Test
    fun couponReasonKey() {
        assertEquals(
            "mobile.marketplace.coupon.reason.expired",
            MarketplaceLogic.couponReasonKey("expired"),
        )
        assertEquals(
            "mobile.marketplace.coupon.reason.not_found",
            MarketplaceLogic.couponReasonKey("weird_new_reason"),
        )
        assertEquals(
            "mobile.marketplace.coupon.reason.not_found",
            MarketplaceLogic.couponReasonKey(null),
        )
        assertEquals(
            "mobile.marketplace.coupon.reason.ok",
            MarketplaceLogic.couponReasonKey("OK"),
        )
    }

    @Test
    fun couponsEnabledRequiresBothFlags() {
        assertFalse(MarketplaceLogic.couponsEnabled(MobilePlatformFeatures()))
        assertFalse(
            MarketplaceLogic.couponsEnabled(
                MobilePlatformFeatures(ffCourseMarketplace = true, ffCourseCoupons = false),
            ),
        )
        assertTrue(
            MarketplaceLogic.couponsEnabled(
                MobilePlatformFeatures(ffCourseMarketplace = true, ffCourseCoupons = true),
            ),
        )
    }

    @Test
    fun purchaseRouteBranches() {
        val appliedPaid = CouponPreview(
            applied = true,
            code = "SAVE25",
            reason = "ok",
            listPriceCents = 4000,
            discountCents = 1000,
            chargedCents = 3000,
            currency = "usd",
            freeAfterDiscount = false,
        )
        val appliedFree = appliedPaid.copy(
            discountCents = 4000,
            chargedCents = 0,
            freeAfterDiscount = true,
            code = "FREE100",
        )
        val rejected = CouponPreview(
            applied = false,
            code = "BAD",
            reason = "expired",
            listPriceCents = 4000,
            chargedCents = 4000,
            currency = "usd",
        )

        assertEquals(
            AndroidPurchaseRoute.FlagOff,
            MarketplaceLogic.purchaseRoute(appliedPaid, couponsOn = false),
        )
        assertEquals(
            AndroidPurchaseRoute.FlagOff,
            MarketplaceLogic.purchaseRoute(appliedPaid, couponsOn = true, marketplaceOn = false),
        )
        assertEquals(
            AndroidPurchaseRoute.Reject,
            MarketplaceLogic.purchaseRoute(rejected, couponsOn = true),
        )
        assertEquals(
            AndroidPurchaseRoute.Reject,
            MarketplaceLogic.purchaseRoute(null, couponsOn = true),
        )
        assertEquals(
            AndroidPurchaseRoute.FreeGrant,
            MarketplaceLogic.purchaseRoute(appliedFree, couponsOn = true),
        )
        assertEquals(
            AndroidPurchaseRoute.StripeCheckout,
            MarketplaceLogic.purchaseRoute(appliedPaid, couponsOn = true),
        )
    }

    @Test
    fun parseCouponReasonFromBody() {
        assertEquals(
            "expired",
            MarketplaceLogic.parseCouponReasonFromBody(
                """{"error":{"code":"unprocessable_entity","message":"x"},"reason":"expired"}""",
            ),
        )
        assertNull(MarketplaceLogic.parseCouponReasonFromBody("{}"))
        assertNull(MarketplaceLogic.parseCouponReasonFromBody(""))
    }
}
