package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures

/** Intro course onboarding helpers (IC07). */
object IntroCourseLogic {
    fun introCourseEnabled(features: MobilePlatformFeatures): Boolean = features.introCourseEnabled

    fun cacheKeyProgress(): String = "intro-course:progress"

    fun cardState(progress: IntroCourseProgress?, loading: Boolean, error: Boolean): IntroCourseCardState {
        if (loading) return IntroCourseCardState.Loading
        if (error || progress == null) return IntroCourseCardState.Error
        if (!progress.enrolled) return IntroCourseCardState.Hidden
        if (progress.completedAt != null) return IntroCourseCardState.Completed
        if (progress.modulesComplete <= 0) return IntroCourseCardState.NotStarted
        return IntroCourseCardState.InProgress
    }

    fun shouldShowCelebration(progress: IntroCourseProgress?): Boolean {
        if (progress?.enrolled != true || progress.completedAt == null) return false
        return progress.celebrationSeen != true
    }

    fun fallbackRoute(courseCode: String = IntroCourseConstants.courseCode): String =
        "/courses/$courseCode"

    fun ctaRoute(progress: IntroCourseProgress): String =
        progress.nextItem?.route ?: fallbackRoute(progress.courseCode ?: IntroCourseConstants.courseCode)

    fun isIntroCourse(courseCode: String): Boolean = courseCode == IntroCourseConstants.courseCode
}