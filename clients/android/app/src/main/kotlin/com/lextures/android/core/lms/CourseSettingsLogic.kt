package com.lextures.android.core.lms

import java.util.TimeZone
import java.util.regex.Pattern

/** Course settings helpers (M13.1). */
object CourseSettingsLogic {
    fun courseItemCreatePermission(courseCode: String): String = "course:$courseCode:item:create"

    fun canManageCourse(courseCode: String, permissions: List<String>): Boolean =
        permissions.contains(courseItemCreatePermission(courseCode))

    fun settingsEnabled(features: com.lextures.android.core.navigation.MobilePlatformFeatures): Boolean =
        features.ffMobileCourseSettings

    fun shouldShowWorkspaceSection(
        course: CourseSummary,
        permissions: List<String>,
        features: com.lextures.android.core.navigation.MobilePlatformFeatures,
    ): Boolean {
        if (!settingsEnabled(features)) return false
        return canManageCourse(course.courseCode, permissions)
    }

    enum class CourseSettingsSection(val labelRes: String) {
        General("mobile_courseSettings_section_general"),
        Features("mobile_courseSettings_section_features"),
        Marketplace("mobile_courseSettings_section_marketplace"),
        Sections("mobile_courseSettings_section_sections"),
        Grading("mobile_courseSettings_section_grading"),
        Outcomes("mobile_courseSettings_section_outcomes"),
        GradingAgents("mobile_courseSettings_section_gradingAgents"),
        Plagiarism("mobile_courseSettings_section_plagiarism"),
        Accessibility("mobile_courseSettings_section_accessibility"),
        Translations("mobile_courseSettings_section_translations"),
        ImportExport("mobile_courseSettings_section_importExport"),
        Blueprint("mobile_courseSettings_section_blueprint"),
        Archive("mobile_courseSettings_section_archive"),
    }

    fun visibleSettingsSections(
        course: CourseSummary,
        features: com.lextures.android.core.navigation.MobilePlatformFeatures,
    ): List<CourseSettingsSection> = buildList {
        add(CourseSettingsSection.General)
        add(CourseSettingsSection.Features)
        if (features.ffCourseMarketplace) add(CourseSettingsSection.Marketplace)
        if (course.isSectionsEnabled) add(CourseSettingsSection.Sections)
        add(CourseSettingsSection.Grading)
        add(CourseSettingsSection.Outcomes)
        if (features.graderAgentEnabled) add(CourseSettingsSection.GradingAgents)
        if (features.ffPlagiarismChecks) add(CourseSettingsSection.Plagiarism)
        if (features.altTextEnforcementEnabled) add(CourseSettingsSection.Accessibility)
        if (features.translationMemoryEnabled) add(CourseSettingsSection.Translations)
        add(CourseSettingsSection.ImportExport)
        add(CourseSettingsSection.Blueprint)
        add(CourseSettingsSection.Archive)
    }

    val gradeLevels = listOf(
        "", "K", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12",
        "K-2", "3-5", "6-8", "9-12", "K-12",
    )

    val markdownThemePresets = listOf("classic", "reader", "serif", "contrast", "night", "accent", "custom")

    enum class RelativeDurationUnit { D, W, M, Y }

    enum class CourseHomeLanding { data, calendar, content_page }

    enum class ScheduleMode { fixed, relative }

    data class ValidationError(val title: String? = null, val courseHome: String? = null)

    fun validateGeneralForm(
        title: String,
        courseHomeLanding: CourseHomeLanding,
        courseHomeContentItemId: String,
    ): ValidationError? {
        var titleError: String? = null
        var homeError: String? = null
        if (title.trim().isEmpty()) titleError = "title-required"
        if (courseHomeLanding == CourseHomeLanding.content_page && courseHomeContentItemId.trim().isEmpty()) {
            homeError = "content-page-required"
        }
        if (titleError == null && homeError == null) return null
        return ValidationError(titleError, homeError)
    }

    fun normalizeCourseHomeLanding(value: String?): CourseHomeLanding = when (value) {
        "calendar" -> CourseHomeLanding.calendar
        "content_page" -> CourseHomeLanding.content_page
        else -> CourseHomeLanding.data
    }

    private val durationPattern = Pattern.compile("^P(\\d+)([DWMY])$", Pattern.CASE_INSENSITIVE)

    fun isoDurationToParts(iso: String?): Pair<String, RelativeDurationUnit> {
        if (iso.isNullOrBlank()) return "" to RelativeDurationUnit.M
        val matcher = durationPattern.matcher(iso.trim())
        if (!matcher.matches()) return "" to RelativeDurationUnit.M
        val unit = runCatching { RelativeDurationUnit.valueOf(matcher.group(2)!!.uppercase()) }
            .getOrDefault(RelativeDurationUnit.M)
        return matcher.group(1)!! to unit
    }

    fun partsToIsoDuration(amount: String, unit: RelativeDurationUnit): String? {
        val n = amount.trim().toIntOrNull() ?: return null
        if (n < 1) return null
        return "P$n$unit"
    }

    fun parseHeroObjectPosition(pos: String?): Pair<Double, Double> {
        if (pos.isNullOrBlank()) return 50.0 to 50.0
        val matcher = Pattern.compile("^(\\d+(?:\\.\\d+)?)%\\s+(\\d+(?:\\.\\d+)?)%$").matcher(pos.trim())
        if (!matcher.matches()) return 50.0 to 50.0
        val x = matcher.group(1)!!.toDoubleOrNull()?.coerceIn(0.0, 100.0) ?: 50.0
        val y = matcher.group(2)!!.toDoubleOrNull()?.coerceIn(0.0, 100.0) ?: 50.0
        return x to y
    }

    fun formatHeroObjectPosition(x: Double, y: Double): String? {
        val rx = kotlin.math.round(x)
        val ry = kotlin.math.round(y)
        if (rx == 50.0 && ry == 50.0) return null
        return "${rx.toInt()}% ${ry.toInt()}%"
    }

    fun defaultImagePrompt(title: String, description: String): String =
        """
        Generate an image for a course banner with the following title and description:
        Title: $title
        Description: $description
        """.trimIndent()

    fun defaultTimezone(): String = TimeZone.getDefault().id

    fun timezoneOptions(): List<String> = TimeZone.getAvailableIDs().sorted()

    fun contentPages(items: List<CourseStructureItem>): List<CourseStructureItem> =
        items.filter { it.kind == "content_page" }

    fun cacheKeySettings(courseCode: String): String = "course:$courseCode:settings"

    fun applyCourseToForm(course: CourseSummary): CourseGeneralFormState {
        val endParts = isoDurationToParts(course.relativeEndAfter)
        val hiddenParts = isoDurationToParts(course.relativeHiddenAfter)
        return CourseGeneralFormState(
            title = course.title,
            description = course.description,
            published = course.published == true,
            gradeLevel = course.gradeLevel.orEmpty(),
            courseTimezone = course.courseTimezone?.trim()?.takeIf { it.isNotEmpty() } ?: defaultTimezone(),
            courseHomeLanding = normalizeCourseHomeLanding(course.courseHomeLanding),
            courseHomeContentItemId = course.courseHomeContentItemId.orEmpty().trim(),
            scheduleMode = if (course.scheduleMode == ScheduleMode.relative.name) ScheduleMode.relative else ScheduleMode.fixed,
            startsAt = course.startsAt.orEmpty(),
            endsAt = course.endsAt.orEmpty(),
            visibleFrom = course.visibleFrom.orEmpty(),
            hiddenAt = course.hiddenAt.orEmpty(),
            relEndAmount = endParts.first,
            relEndUnit = endParts.second,
            relHiddenAmount = hiddenParts.first,
            relHiddenUnit = hiddenParts.second,
            markdownThemePreset = course.markdownThemePreset ?: "default",
            customDraft = course.markdownThemeCustom ?: MarkdownThemeCustom.seed(),
        )
    }

    fun buildCourseUpdateRequest(form: CourseGeneralFormState): CourseUpdateRequest {
        val mode = form.scheduleMode
        return CourseUpdateRequest(
            title = form.title.trim(),
            description = form.description.trim(),
            published = form.published,
            startsAt = if (mode == ScheduleMode.relative) null else form.startsAt.trim().ifEmpty { null },
            endsAt = if (mode == ScheduleMode.relative) null else form.endsAt.trim().ifEmpty { null },
            visibleFrom = if (mode == ScheduleMode.relative) null else form.visibleFrom.trim().ifEmpty { null },
            hiddenAt = if (mode == ScheduleMode.relative) null else form.hiddenAt.trim().ifEmpty { null },
            scheduleMode = mode.name,
            relativeEndAfter = if (mode == ScheduleMode.relative) partsToIsoDuration(form.relEndAmount, form.relEndUnit) else null,
            relativeHiddenAfter = if (mode == ScheduleMode.relative) partsToIsoDuration(form.relHiddenAmount, form.relHiddenUnit) else null,
            courseHomeLanding = form.courseHomeLanding.name,
            courseHomeContentItemId = if (form.courseHomeLanding == CourseHomeLanding.content_page) {
                form.courseHomeContentItemId.trim().ifEmpty { null }
            } else null,
            courseTimezone = form.courseTimezone.trim().ifEmpty { null },
            gradeLevel = form.gradeLevel.trim().ifEmpty { null },
        )
    }

    fun buildMarkdownThemePatch(form: CourseGeneralFormState): CourseMarkdownThemePatch =
        CourseMarkdownThemePatch(
            preset = form.markdownThemePreset,
            custom = if (form.markdownThemePreset == "custom") form.customDraft else null,
        )

    fun isGeneralFormDirty(form: CourseGeneralFormState, course: CourseSummary): Boolean =
        form != applyCourseToForm(course)

    fun courseNeedsUpdate(form: CourseGeneralFormState, course: CourseSummary): Boolean =
        buildCourseUpdateRequest(form) != buildCourseUpdateRequest(applyCourseToForm(course))

    fun themeNeedsUpdate(form: CourseGeneralFormState, course: CourseSummary): Boolean =
        buildMarkdownThemePatch(form) != buildMarkdownThemePatch(applyCourseToForm(course))

    fun isoToLocalDateString(iso: String?): String {
        if (iso.isNullOrBlank()) return ""
        val instant = runCatching { java.time.Instant.parse(iso) }.getOrNull() ?: return ""
        val zoned = instant.atZone(java.time.ZoneId.systemDefault())
        return java.time.format.DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm").format(zoned)
    }

    fun localDateStringToIso(value: String): String? {
        val trimmed = value.trim()
        if (trimmed.isEmpty()) return null
        return runCatching {
            val local = java.time.LocalDateTime.parse(trimmed, java.time.format.DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm"))
            local.atZone(java.time.ZoneId.systemDefault()).toInstant().toString()
        }.getOrNull()
    }
}

data class CourseGeneralFormState(
    val title: String = "",
    val description: String = "",
    val published: Boolean = false,
    val gradeLevel: String = "",
    val courseTimezone: String = CourseSettingsLogic.defaultTimezone(),
    val courseHomeLanding: CourseSettingsLogic.CourseHomeLanding = CourseSettingsLogic.CourseHomeLanding.data,
    val courseHomeContentItemId: String = "",
    val scheduleMode: CourseSettingsLogic.ScheduleMode = CourseSettingsLogic.ScheduleMode.fixed,
    val startsAt: String = "",
    val endsAt: String = "",
    val visibleFrom: String = "",
    val hiddenAt: String = "",
    val relEndAmount: String = "",
    val relEndUnit: CourseSettingsLogic.RelativeDurationUnit = CourseSettingsLogic.RelativeDurationUnit.M,
    val relHiddenAmount: String = "",
    val relHiddenUnit: CourseSettingsLogic.RelativeDurationUnit = CourseSettingsLogic.RelativeDurationUnit.M,
    val markdownThemePreset: String = "default",
    val customDraft: MarkdownThemeCustom = MarkdownThemeCustom.seed(),
)
