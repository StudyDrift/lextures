package com.lextures.android.core.lms

import com.lextures.android.R

/** Course plagiarism / AI-authorship settings helpers (M13.7). */
object CoursePlagiarismLogic {
    data class ProviderOption(
        val value: String,
        val labelRes: Int,
    )

    data class FormDraft(
        val checksEnabled: Boolean = true,
        val provider: String = "",
        val thresholdPct: String = "40",
    )

    enum class ValidationError {
        ThresholdInvalid,
    }

    const val DEFAULT_THRESHOLD_PCT = 40.0

    val providerOptions: List<ProviderOption> = listOf(
        ProviderOption("", R.string.mobile_courseSettings_plagiarism_provider_default),
        ProviderOption("none", R.string.mobile_courseSettings_plagiarism_provider_none),
        ProviderOption("turnitin", R.string.mobile_courseSettings_plagiarism_provider_turnitin),
        ProviderOption("copyleaks", R.string.mobile_courseSettings_plagiarism_provider_copyleaks),
        ProviderOption("gptzero", R.string.mobile_courseSettings_plagiarism_provider_gptzero),
    )

    fun cacheKey(courseCode: String): String = "course:$courseCode:plagiarism-settings"

    fun saveIdempotencyKey(courseCode: String): String = "course-plagiarism:$courseCode:save"

    fun patchPath(courseCode: String): String = "/api/v1/courses/$courseCode/plagiarism-settings"

    fun draft(settings: CoursePlagiarismSettings?): FormDraft =
        FormDraft(
            checksEnabled = settings?.plagiarismChecksEnabled ?: true,
            provider = normalizedProvider(settings?.plagiarismProvider),
            thresholdPct = formatThreshold(settings?.plagiarismAlertThresholdPct),
        )

    fun isDirty(current: FormDraft, baseline: FormDraft): Boolean = current != baseline

    fun validateDraft(draft: FormDraft): ValidationError? =
        if (parsedThreshold(draft.thresholdPct) == null) ValidationError.ThresholdInvalid else null

    fun buildPatchBody(current: FormDraft): PatchCoursePlagiarismBody =
        PatchCoursePlagiarismBody(
            plagiarismChecksEnabled = current.checksEnabled,
            plagiarismProvider = current.provider.takeIf { it.isNotEmpty() },
            plagiarismAlertThresholdPct = parsedThreshold(current.thresholdPct) ?: DEFAULT_THRESHOLD_PCT,
        )

    fun normalizedProvider(provider: String?): String {
        val trimmed = provider.orEmpty().trim().lowercase()
        return providerOptions.firstOrNull { it.value == trimmed && it.value.isNotEmpty() }?.value.orEmpty()
    }

    fun formatThreshold(value: Double?): String {
        val resolved = value?.takeIf { it.isFinite() } ?: DEFAULT_THRESHOLD_PCT
        return if (resolved % 1.0 == 0.0) resolved.toInt().toString() else resolved.toString()
    }

    fun parsedThreshold(text: String): Double? {
        val value = text.trim().toDoubleOrNull() ?: return null
        if (!value.isFinite() || value < 0 || value > 100) return null
        return value
    }
}
