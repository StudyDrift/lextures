package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class MarketplaceLogicTest {
    @Test
    fun isPaidAndFree() {
        assertTrue(MarketplaceLogic.isPaid(1999))
        assertFalse(MarketplaceLogic.isPaid(0))
        assertTrue(MarketplaceLogic.isFree(0))
        assertFalse(MarketplaceLogic.isFree(500))
    }

    @Test
    fun formatPriceUsesFreeLabel() {
        assertEquals("Free", MarketplaceLogic.formatPrice(0, freeLabel = "Free"))
    }

    @Test
    fun cardAccessibleNameIncludesOwned() {
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
    fun shouldShowPurchasedBadgeRequiresFlagAndField() {
        val course = CourseSummary(
            id = "1",
            courseCode = "SPAN101",
            title = "Spanish",
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
    fun majorUnitsToPriceCents() {
        assertEquals(0, MarketplaceLogic.majorUnitsToPriceCents(""))
        assertEquals(1999, MarketplaceLogic.majorUnitsToPriceCents("19.99"))
        assertNull(MarketplaceLogic.majorUnitsToPriceCents("abc"))
    }

    @Test
    fun validateAmount() {
        assertNull(MarketplaceLogic.validateAmount(""))
        assertEquals("invalid", MarketplaceLogic.validateAmount("12.345"))
        assertEquals("min", MarketplaceLogic.validateAmount("0.10"))
        assertNull(MarketplaceLogic.validateAmount("19.99"))
    }

    @Test
    fun ctaLabelKey() {
        assertEquals("goToCourse", MarketplaceLogic.ctaLabelKey(owned = true, priceCents = 0))
        assertEquals("enrollFree", MarketplaceLogic.ctaLabelKey(owned = false, priceCents = 0))
        assertEquals("buyOnWeb", MarketplaceLogic.ctaLabelKey(owned = false, priceCents = 500))
    }

    @Test
    fun marketplaceWebPath() {
        assertEquals("/marketplace/spanish-a1", MarketplaceLogic.marketplaceWebPath("spanish-a1"))
    }
}
