package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class GamificationBadge(
    val badgeType: String,
    val awardedAt: String,
)

@Serializable
data class GamificationProfile(
    val xpTotal: Int = 0,
    val level: Int = 0,
    val xpToNextLevel: Int = 0,
    val levelProgressPct: Double = 0.0,
    val currentStreak: Int = 0,
    val longestStreak: Int = 0,
    val streakFreezes: Int = 0,
    val streakAtRisk: Boolean = false,
    val streakHoursLeft: Double? = null,
    val streakEnded: Boolean? = null,
    val leaderboardVisible: Boolean = true,
    val badges: List<GamificationBadge>? = null,
    val recentBadges: List<GamificationBadge>? = null,
)

@Serializable
data class LeaderboardEntry(
    val rank: Int,
    val userId: String,
    val displayName: String,
    val xpEarned: Int,
    val isCurrentUser: Boolean? = null,
)

@Serializable
data class CourseLeaderboardResponse(
    val topEntries: List<LeaderboardEntry>? = null,
    val currentUser: LeaderboardEntry? = null,
)

@Serializable
data class GamificationBadgesListResponse(
    val badges: List<GamificationBadge>? = null,
)