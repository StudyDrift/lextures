package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures

object GamificationLogic {
    fun gamificationEnabled(features: MobilePlatformFeatures): Boolean = features.ffGamification

    fun cacheKey(): String = "gamification:profile"

    fun leaderboardCacheKey(courseCode: String): String = "gamification:leaderboard:$courseCode"

    fun badgeLabel(badgeType: String): String = when (badgeType) {
        "streak_7" -> "7-day streak"
        "streak_30" -> "30-day streak"
        "xp_100" -> "100 XP"
        "xp_1000" -> "1000 XP"
        "first_course_complete" -> "First course complete"
        else -> badgeType.replace('_', ' ').replaceFirstChar { it.uppercase() }
    }

    fun shouldShowLeaderboard(profile: GamificationProfile): Boolean = profile.leaderboardVisible

    fun canUseStreakFreeze(profile: GamificationProfile): Boolean =
        profile.streakFreezes > 0 && profile.currentStreak > 0
}