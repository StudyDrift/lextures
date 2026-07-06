package com.lextures.android.features.parent

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.Event
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.material.icons.filled.Videocam
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
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
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.ConferenceAvailability
import com.lextures.android.core.lms.ConferenceLogic
import com.lextures.android.core.lms.ConferenceSlot
import com.lextures.android.core.lms.ConferenceSlotsResponse
import com.lextures.android.core.lms.ConferenceTeacher
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ParentConferenceBooking
import com.lextures.android.core.lms.ParentLogic
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSegmentedChips
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

private enum class ConferenceTab(val id: String) {
    Available("available"),
    MyBookings("my"),
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ConferenceBookingScreen(
    session: AuthSession,
    studentId: String,
    childName: String,
    onBack: () -> Unit,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()

    var tab by remember { mutableStateOf(ConferenceTab.Available.id) }
    var teachers by remember { mutableStateOf<List<ConferenceTeacher>>(emptyList()) }
    var selectedTeacherId by remember { mutableStateOf("") }
    var conferenceDate by remember { mutableStateOf(ConferenceLogic.todayDateString()) }
    var slots by remember { mutableStateOf<List<ConferenceSlot>>(emptyList()) }
    var availability by remember { mutableStateOf<ConferenceAvailability?>(null) }
    var myBookings by remember { mutableStateOf<List<ParentConferenceBooking>>(emptyList()) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var slotsLoading by remember { mutableStateOf(false) }
    var bookingsLoading by remember { mutableStateOf(false) }
    var booking by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var confirmationSlot by remember { mutableStateOf<ConferenceSlot?>(null) }
    var rescheduleBooking by remember { mutableStateOf<ParentConferenceBooking?>(null) }

    suspend fun loadSlots(force: Boolean = false) {
        val token = accessToken ?: return
        if (selectedTeacherId.isEmpty() || conferenceDate.isEmpty()) return
        if (!force && slots.isNotEmpty()) return
        slotsLoading = true
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.conferenceSlots(selectedTeacherId, conferenceDate),
                accessToken = token,
                serializer = ConferenceSlotsResponse.serializer(),
            ) {
                LmsApi.fetchConferenceSlots(selectedTeacherId, conferenceDate, token)
            }
            slots = result.first.slots
            availability = result.first.availability
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
            errorMessage = null
        } catch (e: Exception) {
            errorMessage = e.message ?: L.text(context, localePrefs, R.string.mobile_parent_conferences_error_load)
        } finally {
            slotsLoading = false
        }
    }

    suspend fun loadMyBookings() {
        val token = accessToken ?: return
        bookingsLoading = true
        try {
            myBookings = ConferenceLogic.loadParentBookings(
                listOf(studentId to childName),
                token,
            )
        } finally {
            bookingsLoading = false
        }
    }

    LaunchedEffect(accessToken, studentId) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        try {
            teachers = LmsApi.fetchParentConferenceTeachers(studentId, token)
            selectedTeacherId = teachers.firstOrNull()?.teacherId.orEmpty()
            loadMyBookings()
        } catch (e: Exception) {
            errorMessage = e.message
        } finally {
            loading = false
        }
    }

    LaunchedEffect(selectedTeacherId, conferenceDate, accessToken) { loadSlots() }

    LaunchedEffect(rescheduleBooking) {
        val item = rescheduleBooking ?: return@LaunchedEffect
        val token = accessToken ?: return@LaunchedEffect
        runCatching { LmsApi.cancelConferenceBooking(item.slot.id, token) }
        ConferenceReminderScheduler.cancelReminder(context, item.slot.id)
        selectedTeacherId = item.teacher.teacherId
        item.availability?.date?.takeIf { it.isNotEmpty() }?.let { conferenceDate = it }
        rescheduleBooking = null
        tab = ConferenceTab.Available.id
        loadSlots(force = true)
        loadMyBookings()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_parent_bookConferences)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> LmsSkeletonList(count = 3, modifier = Modifier.padding(padding).fillMaxSize())
            else -> Column(
                modifier = Modifier
                    .padding(padding)
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                if (!isOnline) OfflineBanner()
                cacheLabel?.let { StalenessChip(label = it) }
                Text(
                    localePrefs.localizedContext(context).getString(R.string.mobile_parent_conferences_subtitle, childName),
                    fontSize = 14.sp,
                    color = textSecondary(),
                )
                errorMessage?.let { LmsErrorBanner(it) }
                if (teachers.isEmpty()) {
                    Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_noTeachers), color = textSecondary())
                } else {
                    Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_teacher), fontWeight = FontWeight.Bold)
                    teachers.forEach { teacher ->
                        val selected = teacher.teacherId == selectedTeacherId
                        OutlinedButton(onClick = { selectedTeacherId = teacher.teacherId }) {
                            Text(
                                ParentLogic.teacherLabel(context, teacher),
                                fontWeight = if (selected) FontWeight.Bold else FontWeight.Normal,
                            )
                        }
                    }
                    LmsSegmentedChips(
                        options = listOf(
                            ConferenceTab.Available.id to L.text(context, localePrefs, R.string.mobile_parent_conferences_tab_available),
                            ConferenceTab.MyBookings.id to L.text(context, localePrefs, R.string.mobile_parent_conferences_tab_myBookings),
                        ),
                        selectedId = tab,
                        onSelect = { tab = it },
                    )
                    when (tab) {
                        ConferenceTab.Available.id -> {
                            OutlinedTextField(
                                value = conferenceDate,
                                onValueChange = { conferenceDate = it },
                                label = { Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_date)) },
                                modifier = Modifier.fillMaxWidth(),
                            )
                            AvailableConferenceSlots(
                                slots = ConferenceLogic.upcomingAvailableSlots(slots),
                                availability = availability,
                                booking = booking,
                                isOnline = isOnline,
                                slotsLoading = slotsLoading,
                                onBook = { slot ->
                                    val token = accessToken ?: return@AvailableConferenceSlots
                                    scope.launch {
                                        booking = true
                                        try {
                                            val booked = LmsApi.bookConferenceSlot(
                                                slot.id,
                                                studentId,
                                                token,
                                                L.text(context, localePrefs, R.string.mobile_parent_conferences_conflict),
                                            )
                                            val teacher = teachers.find { it.teacherId == selectedTeacherId }
                                            if (teacher != null) {
                                                ConferenceReminderScheduler.scheduleReminder(
                                                    context,
                                                    booked,
                                                    ParentLogic.teacherLabel(context, teacher),
                                                    childName,
                                                )
                                            }
                                            confirmationSlot = booked
                                            loadSlots(force = true)
                                            loadMyBookings()
                                        } catch (e: Exception) {
                                            errorMessage = e.message
                                        } finally {
                                            booking = false
                                        }
                                    }
                                },
                            )
                        }
                        else -> MyConferencesList(
                            bookings = myBookings,
                            bookingsLoading = bookingsLoading,
                            booking = booking,
                            onCancel = { item ->
                                val token = accessToken ?: return@MyConferencesList
                                scope.launch {
                                    booking = true
                                    try {
                                        LmsApi.cancelConferenceBooking(item.slot.id, token)
                                        ConferenceReminderScheduler.cancelReminder(context, item.slot.id)
                                        loadSlots(force = true)
                                        loadMyBookings()
                                    } catch (e: Exception) {
                                        errorMessage = e.message
                                    } finally {
                                        booking = false
                                    }
                                }
                            },
                            onReschedule = { rescheduleBooking = it },
                        )
                    }
                }
            }
        }
    }

    confirmationSlot?.let { slot ->
        val teacher = teachers.find { it.teacherId == selectedTeacherId }
        AlertDialog(
            onDismissRequest = { confirmationSlot = null },
            icon = { Icon(Icons.Default.CalendarMonth, contentDescription = null, tint = accentColor()) },
            title = { Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_booking_confirmed)) },
            text = {
                Column {
                    Text(ConferenceLogic.formatSlotTime(slot))
                    teacher?.let {
                        Text(ParentLogic.teacherLabel(context, it), fontSize = 13.sp, color = textSecondary())
                    }
                    ConferenceLogic.locationLabel(context, availability)?.let {
                        Text(it, fontSize = 13.sp, color = textSecondary())
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = { confirmationSlot = null }) { Text("Done") }
            },
        )
    }
}

@Composable
private fun AvailableConferenceSlots(
    slots: List<ConferenceSlot>,
    availability: ConferenceAvailability?,
    booking: Boolean,
    isOnline: Boolean,
    slotsLoading: Boolean,
    onBook: (ConferenceSlot) -> Unit,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_available), fontWeight = FontWeight.Bold)
    when {
        slotsLoading -> CircularProgressIndicator()
        slots.isEmpty() -> Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_noSlots), color = textSecondary())
        else -> slots.forEach { slot ->
            LmsCard {
                Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f)) {
                        Text(ConferenceLogic.formatSlotTime(slot))
                        ConferenceLogic.locationLabel(context, availability)?.let {
                            Text(it, fontSize = 12.sp, color = textSecondary())
                        }
                    }
                    Button(onClick = { onBook(slot) }, enabled = !booking && isOnline) {
                        Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_book))
                    }
                }
            }
        }
    }
}

@Composable
private fun MyConferencesList(
    bookings: List<ParentConferenceBooking>,
    bookingsLoading: Boolean,
    booking: Boolean,
    onCancel: (ParentConferenceBooking) -> Unit,
    onReschedule: (ParentConferenceBooking) -> Unit,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    when {
        bookingsLoading -> CircularProgressIndicator()
        bookings.isEmpty() -> LmsEmptyState(
            icon = Icons.Default.Event,
            title = L.text(context, localePrefs, R.string.mobile_parent_conferences_myBookings_empty_title),
            message = L.text(context, localePrefs, R.string.mobile_parent_conferences_myBookings_empty_message),
        )
        else -> bookings.forEach { item ->
            val canJoin = ConferenceLogic.isJoinWindow(item.slot, item.availability)
            val videoLink = item.availability?.videoLink?.trim().orEmpty()
            LmsCard {
                Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(ParentLogic.teacherLabel(context, item.teacher), fontWeight = FontWeight.Bold)
                    Text(ConferenceLogic.formatSlotTime(item.slot))
                    ConferenceLogic.locationLabel(context, item.availability)?.let {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Icon(
                                if (videoLink.isNotEmpty()) Icons.Default.Videocam else Icons.Default.LocationOn,
                                contentDescription = null,
                                modifier = Modifier.padding(end = 4.dp),
                            )
                            Text(it, fontSize = 12.sp, color = textSecondary())
                        }
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        if (canJoin && videoLink.isNotEmpty()) {
                            Button(onClick = {
                                context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(videoLink)))
                            }) {
                                Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_joinMeeting))
                            }
                        }
                        OutlinedButton(onClick = {
                            context.startActivity(
                                Intent(Intent.ACTION_VIEW, Uri.parse(ConferenceLogic.icalUrl(item.slot.id))),
                            )
                        }) {
                            Icon(Icons.Default.CalendarMonth, contentDescription = L.text(context, localePrefs, R.string.mobile_parent_conferences_addToCalendar))
                        }
                        OutlinedButton(onClick = { onReschedule(item) }, enabled = !booking) {
                            Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_reschedule))
                        }
                        OutlinedButton(onClick = { onCancel(item) }, enabled = !booking) {
                            Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_cancel))
                        }
                    }
                }
            }
        }
    }
}
