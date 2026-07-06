package com.lextures.android.core.design

import com.lextures.android.core.navigation.MobileDestinations
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.navigation.MobileRoleContext
import com.lextures.android.core.navigation.MoreDestination
import com.lextures.android.core.navigation.RootDestination
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class UIModeLogicTest {
    @Test
    fun gradeToUIModeMapsYoungGrades() {
        assertEquals(UIMode.K2, UIModeLogic.gradeToUIMode("1"))
        assertEquals(UIMode.Elementary, UIModeLogic.gradeToUIMode("4"))
        assertEquals(UIMode.Standard, UIModeLogic.gradeToUIMode("9"))
        assertEquals(UIMode.Standard, UIModeLogic.gradeToUIMode(null))
    }

    @Test
    fun serverOverrideBeatsLocalPreference() {
        val mode = UIModeLogic.effectiveMode(
            featureEnabled = true,
            roleContext = MobileRoleContext.Learning,
            serverOverride = "k2",
            serverEffective = "standard",
            localPreference = UIModePreference.Standard,
        )
        assertEquals(UIMode.K2, mode)
    }

    @Test
    fun teachingContextAlwaysStandard() {
        val mode = UIModeLogic.effectiveMode(
            featureEnabled = true,
            roleContext = MobileRoleContext.Teaching,
            serverOverride = "k2",
            serverEffective = "k2",
            localPreference = UIModePreference.K2,
        )
        assertEquals(UIMode.Standard, mode)
    }

    @Test
    fun k2DrawerHidesNotebooks() {
        val groups = MobileDestinations.globalDrawerGroups(
            MobileRoleContext.Learning,
            MobilePlatformFeatures(),
            UIMode.K2,
        )
        val items = groups.flatMap { it.items }
        assertFalse(RootDestination.Notebooks in items)
        assertTrue(RootDestination.Dashboard in items)
    }

    @Test
    fun k2MoreHubHidesAdvancedDestinations() {
        val platform = MobilePlatformFeatures(
            ffLibrary = true,
            ffPeerReview = true,
            ffGamification = true,
        )
        val destinations = MobileDestinations.moreDestinations(
            MobileRoleContext.Learning,
            platform,
            UIMode.K2,
        )
        assertTrue(MoreDestination.Reading in destinations)
        assertFalse(MoreDestination.PeerReviews in destinations)
        assertFalse(MoreDestination.Gamification in destinations)
    }
}
