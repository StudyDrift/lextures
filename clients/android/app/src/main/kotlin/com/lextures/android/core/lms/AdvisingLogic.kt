package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

object AdvisingLogic {
    fun advisingEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffAdvisingIntegration && features.ffMobileAdvising

    fun notesCacheKey(): String = "advising:notes"

    fun degreeProgressCacheKey(): String = "advising:degree-progress"

    fun visibleNotes(notes: List<AdvisingNote>): List<AdvisingNote> =
        notes.filter { it.visibleToStudent }

    fun sortedNotes(notes: List<AdvisingNote>): List<AdvisingNote> =
        visibleNotes(notes).sortedByDescending { parseDate(it.createdAt) }

    fun advisorFromNotes(notes: List<AdvisingNote>): AdvisingAdvisorInfo? {
        val newest = sortedNotes(notes).firstOrNull() ?: return null
        return AdvisingAdvisorInfo(
            displayName = advisorLabel(newest.advisorDisplayName, newest.advisorEmail),
            email = newest.advisorEmail,
        )
    }

    fun advisorLabel(displayName: String?, email: String?): String {
        displayName?.trim()?.takeIf { it.isNotEmpty() }?.let { return it }
        email?.trim()?.takeIf { it.isNotEmpty() }?.let { return it }
        return "Your advisor"
    }

    fun appointmentUrl(progress: DegreeProgress?, config: MyAdvisingConfig?): String? {
        progress?.appointmentUrl?.trim()?.takeIf { it.isNotEmpty() }?.let { return it }
        return config?.appointmentUrl?.trim()?.takeIf { it.isNotEmpty() }
    }

    fun canBookAppointment(isOnline: Boolean, appointmentUrl: String?): Boolean =
        !appointmentUrl.isNullOrBlank() && isOnline

    fun formatNoteDate(iso: String): String = formatIsoDate(iso)

    fun formatAuditDate(iso: String): String = formatIsoDate(iso)

    private fun formatIsoDate(iso: String): String =
        runCatching {
            Instant.parse(iso)
                .atZone(ZoneId.systemDefault())
                .format(DateTimeFormatter.ofLocalizedDateTime(FormatStyle.MEDIUM, FormatStyle.SHORT))
        }.getOrDefault(iso)

    private fun parseDate(iso: String): Instant =
        runCatching { Instant.parse(iso) }.getOrDefault(Instant.EPOCH)
}
