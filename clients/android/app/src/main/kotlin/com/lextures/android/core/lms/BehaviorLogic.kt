package com.lextures.android.core.lms

import java.time.Instant
import java.time.format.DateTimeFormatterBuilder
import java.time.temporal.ChronoUnit

enum class BehaviorAwardMode {
    Award,
    Referral,
}

data class HallPassCountdown(
    val remainingSeconds: Int,
    val isExpired: Boolean,
    val isOverdue: Boolean,
)

object BehaviorLogic {
    private val studentRoles = setOf("student", "learner")
    val hallPassDestinations = listOf("bathroom", "office", "library", "nurse", "other")
    const val defaultPassMinutes = 5

    fun studentRoster(enrollments: List<CourseEnrollment>): List<CourseEnrollment> =
        enrollments.filter { it.role.lowercase() in studentRoles }

    fun studentLabel(enrollment: CourseEnrollment): String {
        val name = enrollment.displayName?.trim().orEmpty()
        return name.ifEmpty { "Student" }
    }

    fun activeCategories(categories: List<BehaviorCategory>): List<BehaviorCategory> =
        categories.filter { it.active }

    fun positiveCategories(categories: List<BehaviorCategory>): List<BehaviorCategory> =
        activeCategories(categories).filter { it.isPositive }

    fun negativeCategories(categories: List<BehaviorCategory>): List<BehaviorCategory> =
        activeCategories(categories).filter { it.isNegative }

    fun awardPayload(
        studentIds: Set<String>,
        categoryId: String,
        note: String?,
    ): List<PbisAwardInput> {
        val trimmed = note?.trim().orEmpty()
        return studentIds.map { studentId ->
            PbisAwardInput(
                studentId = studentId,
                categoryId = categoryId,
                points = 1,
                note = trimmed.takeIf { it.isNotEmpty() },
            )
        }
    }

    fun isActiveHallPass(pass: HallPass): Boolean {
        val status = pass.status.lowercase()
        return status == "requested" || status == "approved"
    }

    fun hallPassCountdown(pass: HallPass, now: Instant = Instant.now()): HallPassCountdown? {
        if (pass.status.lowercase() != "approved") return null
        val approvedAt = parseInstant(pass.approvedAt) ?: return null
        val estimated = pass.estimatedMins ?: defaultPassMinutes
        val deadline = approvedAt.plus(estimated.toLong(), ChronoUnit.MINUTES)
        val remaining = ChronoUnit.SECONDS.between(now, deadline).toInt()
        val overdue = pass.overdue == true || remaining <= 0
        return HallPassCountdown(
            remainingSeconds = maxOf(remaining, 0),
            isExpired = remaining <= 0,
            isOverdue = overdue,
        )
    }

    fun formatCountdown(countdown: HallPassCountdown): String {
        val minutes = countdown.remainingSeconds / 60
        val seconds = countdown.remainingSeconds % 60
        return "%d:%02d".format(minutes, seconds)
    }

    fun destinationLabelRes(destination: String): String = when (destination.lowercase()) {
        "bathroom" -> "mobile_hallpass_destination_bathroom"
        "office" -> "mobile_hallpass_destination_office"
        "library" -> "mobile_hallpass_destination_library"
        "nurse" -> "mobile_hallpass_destination_nurse"
        else -> "mobile_hallpass_destination_other"
    }

    fun statusLabelRes(status: String): String = when (status.lowercase()) {
        "requested" -> "mobile_hallpass_status_requested"
        "approved" -> "mobile_hallpass_status_approved"
        "denied" -> "mobile_hallpass_status_denied"
        "returned" -> "mobile_hallpass_status_returned"
        else -> status
    }

    fun storedPassKey(sectionId: String): String = "hallPass:$sectionId"

    private fun parseInstant(value: String?): Instant? {
        if (value.isNullOrBlank()) return null
        return runCatching {
            Instant.from(
                DateTimeFormatterBuilder()
                    .appendInstant(3)
                    .toFormatter()
                    .parse(value),
            )
        }.getOrNull()
    }
}
