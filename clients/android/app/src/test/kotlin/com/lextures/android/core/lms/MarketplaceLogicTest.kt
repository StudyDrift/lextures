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
            MarketplaceLogic.purchaseEnabled(MobilePlatformFeatures(ffCourseMarketplace = true)),
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
}
