package com.lextures.android.features.parent

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
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
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
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.ConferenceLogic
import com.lextures.android.core.lms.ConferenceSlot
import com.lextures.android.core.lms.ConferenceTeacher
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ParentLogic
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

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
    val scope = rememberCoroutineScope()

    var teachers by remember { mutableStateOf<List<ConferenceTeacher>>(emptyList()) }
    var selectedTeacherId by remember { mutableStateOf("") }
    var conferenceDate by remember { mutableStateOf(ConferenceLogic.todayDateString()) }
    var slots by remember { mutableStateOf<List<ConferenceSlot>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var slotsLoading by remember { mutableStateOf(false) }
    var booking by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var successMessage by remember { mutableStateOf<String?>(null) }

    suspend fun loadSlots() {
        val token = accessToken ?: return
        if (selectedTeacherId.isEmpty() || conferenceDate.isEmpty()) return
        slotsLoading = true
        try {
            slots = LmsApi.fetchConferenceSlots(selectedTeacherId, conferenceDate, token).slots
            errorMessage = null
        } catch (e: Exception) {
            errorMessage = e.message
        } finally {
            slotsLoading = false
        }
    }

    LaunchedEffect(accessToken, studentId) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        try {
            teachers = LmsApi.fetchParentConferenceTeachers(studentId, token)
            selectedTeacherId = teachers.firstOrNull()?.teacherId.orEmpty()
        } catch (e: Exception) {
            errorMessage = e.message
        } finally {
            loading = false
        }
    }

    LaunchedEffect(selectedTeacherId, conferenceDate, accessToken) { loadSlots() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_parent_book_conferences)) },
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
                Text(
                    localePrefs.localizedContext(context).getString(R.string.mobile_parent_conferences_subtitle, childName),
                    fontSize = 14.sp,
                    color = textSecondary(),
                )
                errorMessage?.let { LmsErrorBanner(it) }
                successMessage?.let { LmsCard { Text(it) } }
                if (teachers.isEmpty()) {
                    Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_no_teachers), color = textSecondary())
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
                    OutlinedTextField(
                        value = conferenceDate,
                        onValueChange = { conferenceDate = it },
                        label = { Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_date)) },
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_available), fontWeight = FontWeight.Bold)
                    if (slotsLoading) {
                        CircularProgressIndicator()
                    } else if (ConferenceLogic.upcomingAvailableSlots(slots).isEmpty()) {
                        Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_no_slots), color = textSecondary())
                    } else {
                        ConferenceLogic.upcomingAvailableSlots(slots).forEach { slot ->
                            LmsCard {
                                Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                                    Text(ConferenceLogic.formatSlotTime(slot), modifier = Modifier.weight(1f))
                                    Button(
                                        onClick = {
                                            val token = accessToken ?: return@Button
                                            scope.launch {
                                                booking = true
                                                try {
                                                    LmsApi.bookConferenceSlot(
                                                        slot.id,
                                                        studentId,
                                                        token,
                                                        L.text(context, localePrefs, R.string.mobile_parent_conferences_conflict),
                                                    )
                                                    successMessage = L.text(context, localePrefs, R.string.mobile_parent_conferences_booked)
                                                    loadSlots()
                                                } catch (e: Exception) {
                                                    errorMessage = e.message
                                                } finally {
                                                    booking = false
                                                }
                                            }
                                        },
                                        enabled = !booking,
                                    ) {
                                        Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_book))
                                    }
                                }
                            }
                        }
                    }
                    val booked = ConferenceLogic.myBookedSlots(slots, null, studentId)
                    if (booked.isNotEmpty()) {
                        Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_my_bookings), fontWeight = FontWeight.Bold)
                        booked.forEach { slot ->
                            LmsCard {
                                Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                                    Text(ConferenceLogic.formatSlotTime(slot), modifier = Modifier.weight(1f))
                                    OutlinedButton(
                                        onClick = {
                                            val token = accessToken ?: return@OutlinedButton
                                            scope.launch {
                                                booking = true
                                                try {
                                                    LmsApi.cancelConferenceBooking(slot.id, token)
                                                    successMessage = L.text(context, localePrefs, R.string.mobile_parent_conferences_cancelled)
                                                    loadSlots()
                                                } catch (e: Exception) {
                                                    errorMessage = e.message
                                                } finally {
                                                    booking = false
                                                }
                                            }
                                        },
                                        enabled = !booking,
                                    ) {
                                        Text(L.text(context, localePrefs, R.string.mobile_parent_conferences_cancel))
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
