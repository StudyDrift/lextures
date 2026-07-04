package com.lextures.android.core.lms

import com.lextures.android.core.config.AppConfiguration
import java.time.Duration
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

/** Live virtual-meeting formatting, grouping, and planner calendar mapping (M7.5). */
object LiveMeetingsLogic {
    private val soonWindow: Duration = Duration.ofMinutes(30)

    data class GroupedMeetings(
        val live: List<VirtualMeeting>,
        val upcoming: List<VirtualMeeting>,
        val past: List<VirtualMeeting>,
    )

    data class LiveUpcomingItem(
        val courseCode: String,
        val courseTitle: String,
        val meeting: VirtualMeeting,
    )

    fun groupMeetings(meetings: List<VirtualMeeting>): GroupedMeetings = GroupedMeetings(
        live = meetings.filter { it.status == "live" },
        upcoming = meetings.filter { it.status == "scheduled" },
        past = meetings.filter { it.status == "ended" },
    )

    fun isLiveOrSoon(meeting: VirtualMeeting, now: Instant = Instant.now()): Boolean {
        if (meeting.status == "live") return true
        if (meeting.status != "scheduled") return false
        val start = LmsDates.parse(meeting.scheduledStart) ?: return false
        val diff = Duration.between(now, start)
        return !diff.isNegative && diff <= soonWindow
    }

    fun canJoin(meeting: VirtualMeeting, now: Instant = Instant.now()): Boolean {
        if (meeting.status == "cancelled" || meeting.status == "ended") return false
        return meeting.status == "live" || isLiveOrSoon(meeting, now)
    }

    fun formatMeetingTime(meeting: VirtualMeeting): String {
        val startRaw = meeting.scheduledStart ?: return "No time set"
        val start = LmsDates.parse(startRaw) ?: return startRaw
        val zone = ZoneId.systemDefault()
        val startText = DateTimeFormatter.ofLocalizedDateTime(FormatStyle.MEDIUM, FormatStyle.SHORT)
            .withZone(zone)
            .format(start)
        val endRaw = meeting.scheduledEnd ?: return startText
        val end = LmsDates.parse(endRaw) ?: return startText
        val endText = DateTimeFormatter.ofLocalizedTime(FormatStyle.SHORT)
            .withZone(zone)
            .format(end)
        return "$startText – $endText"
    }

    fun countdownText(scheduledStart: String, now: Instant = Instant.now()): String? {
        val start = LmsDates.parse(scheduledStart) ?: return null
        val seconds = Duration.between(now, start).seconds.coerceAtLeast(0)
        if (seconds <= 0) return null
        val minutes = seconds / 60
        val remainder = seconds % 60
        return "Starting in ${minutes}m ${"%02d".format(remainder)}s"
    }

    fun calendarEvents(
        courseCode: String,
        courseTitle: String,
        meetings: List<VirtualMeeting>,
    ): List<PlannerCalendarEvent> = meetings.mapNotNull { meeting ->
        if (meeting.status != "scheduled" && meeting.status != "live") return@mapNotNull null
        val start = LmsDates.parse(meeting.scheduledStart) ?: return@mapNotNull null
        val end = meeting.scheduledEnd?.let(LmsDates::parse)
        PlannerCalendarEvent(
            id = "live-meeting:${meeting.id}",
            title = meeting.title,
            courseCode = courseCode,
            courseTitle = courseTitle,
            startsAt = start,
            endsAt = end,
            allDay = false,
            kind = PlannerCalendarEventKind.LiveMeeting,
            meetingId = meeting.id,
        )
    }

    fun collectCalendarEvents(
        studentCourses: List<CourseSummary>,
        meetingsByCourseCode: Map<String, List<VirtualMeeting>>,
    ): List<PlannerCalendarEvent> = studentCourses.flatMap { course ->
        if (!course.isLiveSessionsEnabled) return@flatMap emptyList()
        calendarEvents(
            courseCode = course.courseCode,
            courseTitle = course.displayTitle,
            meetings = meetingsByCourseCode[course.courseCode].orEmpty(),
        )
    }

    fun collectLiveAndUpcoming(
        courses: List<CourseSummary>,
        meetingsByCourseCode: Map<String, List<VirtualMeeting>>,
        limit: Int = 5,
        now: Instant = Instant.now(),
    ): List<LiveUpcomingItem> = courses
        .asSequence()
        .filter { it.isLiveSessionsEnabled }
        .flatMap { course ->
            meetingsByCourseCode[course.courseCode].orEmpty()
                .asSequence()
                .filter { it.status == "live" || isLiveOrSoon(it, now) }
                .map { LiveUpcomingItem(course.courseCode, course.displayTitle, it) }
        }
        .sortedWith(
            compareBy<LiveUpcomingItem> { it.meeting.status != "live" }
                .thenBy { LmsDates.parse(it.meeting.scheduledStart) ?: Instant.MAX },
        )
        .take(limit)
        .toList()

    fun meetingIcalUrl(meetingId: String): String =
        AppConfiguration.apiUrl("/api/v1/meetings/$meetingId/ical").toString()
}