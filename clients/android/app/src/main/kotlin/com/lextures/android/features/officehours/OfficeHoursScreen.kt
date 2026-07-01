package com.lextures.android.features.officehours

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.material.icons.filled.Videocam
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.AppointmentSlot
import com.lextures.android.core.lms.AvailabilityWindow
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.OfficeHoursAvailability
import com.lextures.android.core.lms.OfficeHoursLogic
import com.lextures.android.core.lms.isOfficeHoursEnabled
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSegmentedChips
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

private enum class OfficeHoursTab(val id: String) {
    Available("available"),
    MyBookings("my"),
}

@Composable
fun CourseOfficeHoursSection(
    session: AuthSession,
    course: CourseSummary,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()

    var tab by remember { mutableStateOf(OfficeHoursTab.Available.id) }
    var availability by remember { mutableStateOf<OfficeHoursAvailability?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var bookingSlot by remember { mutableStateOf<AppointmentSlot?>(null) }
    var confirmationSlot by remember { mutableStateOf<AppointmentSlot?>(null) }
    var rescheduleSlot by remember { mutableStateOf<AppointmentSlot?>(null) }

    suspend fun load(force: Boolean = false) {
        val token = accessToken ?: return
        if (!force && availability != null) return
        loading = true
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.officeHours(course.courseCode),
                accessToken = token,
                serializer = OfficeHoursAvailability.serializer(),
            ) {
                LmsApi.fetchOfficeHoursAvailability(course.courseCode, token)
            }
            availability = result.first
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken, course.courseCode) { load() }

    LaunchedEffect(rescheduleSlot) {
        val slot = rescheduleSlot ?: return@LaunchedEffect
        val token = accessToken ?: return@LaunchedEffect
        runCatching { LmsApi.cancelOfficeHoursBooking(slot.id, token) }
        OfficeHoursReminderScheduler.cancelReminder(context, slot.id)
        rescheduleSlot = null
        tab = OfficeHoursTab.Available.id
        load(force = true)
    }

    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (!isOnline) OfflineBanner()
        cacheLabel?.let { StalenessChip(label = it) }
        errorMessage?.let { LmsErrorBanner(it) }

        LmsSegmentedChips(
            options = listOf(
                OfficeHoursTab.Available.id to officeHoursTabAvailable(),
                OfficeHoursTab.MyBookings.id to officeHoursTabMyBookings(),
            ),
            selectedId = tab,
            onSelect = { tab = it },
        )

        when {
            loading -> LmsSkeletonList(count = 3)
            availability == null -> Unit
            tab == OfficeHoursTab.Available.id -> AvailableSlotsList(
                slots = OfficeHoursLogic.upcomingAvailableSlots(availability!!.slots),
                windows = OfficeHoursLogic.windowMap(availability!!.windows),
                onBook = { bookingSlot = it },
            )
            else -> MyBookingsList(
                slots = OfficeHoursLogic.myBookedSlots(availability!!.slots),
                windows = OfficeHoursLogic.windowMap(availability!!.windows),
                course = course,
                session = session,
                onCancel = {
                    scope.launch {
                        val token = accessToken ?: return@launch
                        runCatching { LmsApi.cancelOfficeHoursBooking(it.id, token) }
                        OfficeHoursReminderScheduler.cancelReminder(context, it.id)
                        load(force = true)
                    }
                },
                onReschedule = { rescheduleSlot = it },
            )
        }
    }

    bookingSlot?.let { slot ->
        BookingDialog(
            slot = slot,
            session = session,
            isOnline = isOnline,
            onDismiss = { bookingSlot = null },
            onBooked = { booked ->
                bookingSlot = null
                confirmationSlot = booked
                OfficeHoursReminderScheduler.scheduleReminder(context, booked, course.displayTitle)
                scope.launch { load(force = true) }
            },
        )
    }

    confirmationSlot?.let { slot ->
        AlertDialog(
            onDismissRequest = { confirmationSlot = null },
            icon = { Icon(Icons.Default.CalendarMonth, contentDescription = null, tint = accentColor()) },
            title = { Text(officeHoursBookingConfirmed()) },
            text = {
                Column {
                    Text(OfficeHoursLogic.formatSlotTime(slot))
                    OfficeHoursLogic.locationLabel(OfficeHoursLogic.windowMap(availability?.windows.orEmpty())[slot.windowId])
                        ?.let { Text(it, fontSize = 13.sp, color = textSecondary()) }
                }
            },
            confirmButton = {
                TextButton(onClick = { confirmationSlot = null }) { Text("Done") }
            },
        )
    }
}

@Composable
private fun AvailableSlotsList(
    slots: List<AppointmentSlot>,
    windows: Map<String, AvailabilityWindow>,
    onBook: (AppointmentSlot) -> Unit,
) {
    if (slots.isEmpty()) {
        LmsEmptyState(
            icon = Icons.Default.CalendarMonth,
            title = officeHoursEmptyTitle(),
            message = officeHoursEmptyMessage(),
        )
        return
    }
    slots.forEach { slot ->
        val window = windows[slot.windowId]
        LmsCard {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    OfficeHoursLogic.formatSlotTime(slot),
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                OfficeHoursLogic.locationLabel(window)?.let { location ->
                    Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                        Icon(
                            if (window?.isVirtual == true) Icons.Default.Videocam else Icons.Default.LocationOn,
                            contentDescription = null,
                            tint = textSecondary(),
                        )
                        Text(location, fontSize = 12.sp, color = textSecondary())
                    }
                }
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) {
                    Button(onClick = { onBook(slot) }) { Text(officeHoursBook()) }
                }
            }
        }
    }
}

@Composable
private fun MyBookingsList(
    slots: List<AppointmentSlot>,
    windows: Map<String, AvailabilityWindow>,
    course: CourseSummary,
    session: AuthSession,
    onCancel: (AppointmentSlot) -> Unit,
    onReschedule: (AppointmentSlot) -> Unit,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()

    if (slots.isEmpty()) {
        LmsEmptyState(
            icon = Icons.Default.CalendarMonth,
            title = officeHoursMyBookingsEmptyTitle(),
            message = officeHoursMyBookingsEmptyMessage(),
        )
        return
    }

    slots.forEach { slot ->
        val window = windows[slot.windowId]
        val canJoin = window?.isVirtual == true && !slot.meetingId.isNullOrBlank() && isJoinWindow(slot)
        LmsCard(accent = accentColor()) {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(OfficeHoursLogic.formatSlotTime(slot), fontWeight = FontWeight.SemiBold, color = textPrimary())
                OfficeHoursLogic.locationLabel(window)?.let {
                    Text(it, fontSize = 12.sp, color = textSecondary())
                }
                slot.studentNote?.trim()?.takeIf { it.isNotEmpty() }?.let {
                    Text(officeHoursMyBookingsNote(it), fontSize = 12.sp, color = textSecondary())
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    if (canJoin) {
                        Button(onClick = {
                            scope.launch {
                                val token = accessToken ?: return@launch
                                val meetingId = slot.meetingId ?: return@launch
                                val joinUrl = LmsApi.fetchMeetingJoinUrl(meetingId, token) ?: return@launch
                                context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(joinUrl)))
                            }
                        }) {
                            Text(officeHoursJoinMeeting())
                        }
                    }
                    OutlinedButton(onClick = {
                        val url = AppConfiguration.apiUrl("/api/v1/slots/${slot.id}/ical").toString()
                        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                    }) {
                        Text(officeHoursAddToCalendar())
                    }
                    OutlinedButton(onClick = { onReschedule(slot) }) { Text(officeHoursReschedule()) }
                    TextButton(onClick = { onCancel(slot) }) { Text(officeHoursCancel()) }
                }
            }
        }
    }
}

@Composable
private fun BookingDialog(
    slot: AppointmentSlot,
    session: AuthSession,
    isOnline: Boolean,
    onDismiss: () -> Unit,
    onBooked: (AppointmentSlot) -> Unit,
) {
    val accessToken by session.accessToken.collectAsState()
    var note by remember { mutableStateOf("") }
    var saving by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    AlertDialog(
        onDismissRequest = { if (!saving) onDismiss() },
        title = { Text(officeHoursBookingTitle()) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text(OfficeHoursLogic.formatSlotTime(slot), fontWeight = FontWeight.Medium)
                OutlinedTextField(
                    value = note,
                    onValueChange = { note = it },
                    label = { Text(officeHoursBookingNoteLabel()) },
                    placeholder = { Text(officeHoursBookingNotePlaceholder()) },
                    modifier = Modifier.fillMaxWidth(),
                )
                errorMessage?.let { Text(it, color = androidx.compose.ui.graphics.Color.Red, fontSize = 13.sp) }
                if (saving) {
                    CircularProgressIndicator(modifier = Modifier.align(Alignment.CenterHorizontally))
                }
            }
        },
        confirmButton = {
            Button(
                enabled = !saving && isOnline,
                onClick = {
                    scope.launch {
                        val token = accessToken ?: return@launch
                        saving = true
                        errorMessage = null
                        try {
                            val booked = LmsApi.bookOfficeHoursSlot(slot.id, note, token)
                            onBooked(booked)
                        } catch (e: Exception) {
                            errorMessage = session.mapError(e)
                        } finally {
                            saving = false
                        }
                    }
                },
            ) { Text(officeHoursBook()) }
        },
        dismissButton = {
            TextButton(enabled = !saving, onClick = onDismiss) { Text("Cancel") }
        },
    )
}

private fun isJoinWindow(slot: AppointmentSlot): Boolean {
    val start = LmsDates.parse(slot.slotStart) ?: return false
    val end = LmsDates.parse(slot.slotEnd) ?: start.plusSeconds(15 * 60)
    val now = java.time.Instant.now()
    return !now.isBefore(start.minusSeconds(10 * 60)) && !now.isAfter(end)
}
