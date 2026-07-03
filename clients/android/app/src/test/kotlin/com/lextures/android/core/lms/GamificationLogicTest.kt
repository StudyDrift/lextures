package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class GamificationLogicTest {
    @Test
    fun gamificationEnabled() {
        assertFalse(GamificationLogic.gamificationEnabled(MobilePlatformFeatures()))
        assertTrue(GamificationLogic.gamificationEnabled(MobilePlatformFeatures(ffGamification = true)))
    }

    @Test
    fun badgeLabel() {
        assertEquals("7-day streak", GamificationLogic.badgeLabel("streak_7"))
    }

    @Test
    fun leaderboardVisibility() {
        val hidden = GamificationProfile(leaderboardVisible = false)
        val visible = GamificationProfile(leaderboardVisible = true)
        assertFalse(GamificationLogic.shouldShowLeaderboard(hidden))
        assertTrue(GamificationLogic.shouldShowLeaderboard(visible))
    }
}