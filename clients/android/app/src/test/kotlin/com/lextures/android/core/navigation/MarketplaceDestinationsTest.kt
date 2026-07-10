package com.lextures.android.core.navigation

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MarketplaceDestinationsTest {
    @Test
    fun marketplaceAppearsWhenFlagOn() {
        val destinations = MobileDestinations.moreDestinations(
            MobileRoleContext.Learning,
            MobilePlatformFeatures(ffCourseMarketplace = true),
        )
        assertTrue(MoreDestination.Marketplace in destinations)
    }

    @Test
    fun marketplaceHiddenWhenFlagOff() {
        val destinations = MobileDestinations.moreDestinations(
            MobileRoleContext.Learning,
            MobilePlatformFeatures(ffCourseMarketplace = false),
        )
        assertFalse(MoreDestination.Marketplace in destinations)
    }

    @Test
    fun marketplaceHiddenInK2EvenWhenFlagOn() {
        val destinations = MobileDestinations.moreDestinations(
            MobileRoleContext.Learning,
            MobilePlatformFeatures(ffCourseMarketplace = true, ffLibrary = true),
            com.lextures.android.core.design.UIMode.K2,
        )
        assertFalse(MoreDestination.Marketplace in destinations)
    }

    @Test
    fun marketplaceHiddenForTeaching() {
        val destinations = MobileDestinations.moreDestinations(
            MobileRoleContext.Teaching,
            MobilePlatformFeatures(ffCourseMarketplace = true),
        )
        assertFalse(MoreDestination.Marketplace in destinations)
    }
}
