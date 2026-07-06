package com.lextures.android.features.behavior

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.DirectionsWalk
import androidx.compose.ui.platform.LocalContext
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.BehaviorLogic
import com.lextures.android.core.lms.CourseEnrollment
import com.lextures.android.core.lms.CourseSection
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.HallPass
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import com.lextures.android.features.navigation.drawerString
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.time.Instant

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HallPassScreen(
    session: AuthSession,
    course: CourseSummary,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current

    var sections by remember { mutableStateOf<List<CourseSection>>(emptyList()) }
    var enrollments by remember { mutableStateOf<List<CourseEnrollment>>(emptyList()) }
    var selectedSectionId by remember { mutableStateOf("") }
    var passes by remember { mutableStateOf<List<HallPass>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var updatingPassId by remember { mutableStateOf<String?>(null) }
    var now by remember { mutableStateOf(Instant.now()) }
    var reloadKey by remember { mutableStateOf(0) }

    val rosterById = remember(enrollments) {
        BehaviorLogic.studentRoster(enrollments).associateBy { it.userId }
    }

    LaunchedEffect(Unit) {
        while (true) {
            delay(1000)
            now = Instant.now()
        }
    }

    LaunchedEffect(accessToken, selectedSectionId, reloadKey) {
        val token = accessToken ?: return@LaunchedEffect
        if (selectedSectionId.isBlank()) return@LaunchedEffect
        loading = true
        errorMessage = null
        runCatching {
            passes = LmsApi.fetchActiveHallPasses(selectedSectionId, token)
        }.onFailure {
            errorMessage = it.message ?: L.text(context, localePrefs, R.string.mobile_hallpass_loadError)
        }
        loading = false
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        runCatching {
            sections = LmsApi.fetchCourseSections(course.courseCode, token)
            enrollments = LmsApi.fetchCourseEnrollments(course.courseCode, token)
            if (selectedSectionId.isBlank()) {
                selectedSectionId = sections.firstOrNull()?.id.orEmpty()
            }
        }.onFailure {
            errorMessage = it.message ?: L.text(context, localePrefs, R.string.mobile_hallpass_loadError)
        }
    }

    Column(
        modifier = modifier
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        errorMessage?.let { LmsErrorBanner(message = it) }

        if (sections.isNotEmpty()) {
            LmsCard {
                var expanded by remember { mutableStateOf(false) }
                val selectedLabel = sections.firstOrNull { it.id == selectedSectionId }?.displayName.orEmpty()
                ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = it }) {
                    TextField(
                        value = selectedLabel,
                        onValueChange = {},
                        readOnly = true,
                        label = { Text(L.text(R.string.mobile_hallpass_section)) },
                        trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded) },
                        modifier = Modifier
                            .fillMaxWidth()
                            .menuAnchor(),
                    )
                    ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                        sections.forEach { section ->
                            DropdownMenuItem(
                                text = { Text(section.displayName) },
                                onClick = {
                                    selectedSectionId = section.id
                                    expanded = false
                                    reloadKey++
                                },
                            )
                        }
                    }
                }
            }
        }

        when {
            loading && passes.isEmpty() -> LmsSkeletonList(count = 3)
            passes.isEmpty() -> LmsEmptyState(
                icon = Icons.Filled.DirectionsWalk,
                title = L.text(R.string.mobile_hallpass_teacher_empty_title),
                message = L.text(R.string.mobile_hallpass_teacher_empty_message),
            )
            else -> passes.forEach { pass ->
                PassCard(
                    pass = pass,
                    studentName = pass.studentId?.let { rosterById[it] }?.let(BehaviorLogic::studentLabel)
                        ?: L.text(R.string.mobile_hallpass_studentFallback),
                    now = now,
                    updating = updatingPassId == pass.id,
                    onApprove = {
                        scope.launch { updatePass(session, pass.id, "approved", updatingPassId, onUpdating = { updatingPassId = it }, onDone = { reloadKey++ }, onError = { errorMessage = it }) }
                    },
                    onDeny = {
                        scope.launch { updatePass(session, pass.id, "denied", updatingPassId, onUpdating = { updatingPassId = it }, onDone = { reloadKey++ }, onError = { errorMessage = it }) }
                    },
                    onReturn = {
                        scope.launch { updatePass(session, pass.id, "returned", updatingPassId, onUpdating = { updatingPassId = it }, onDone = { reloadKey++ }, onError = { errorMessage = it }) }
                    },
                )
            }
        }
    }
}

@Composable
private fun PassCard(
    pass: HallPass,
    studentName: String,
    now: Instant,
    updating: Boolean,
    onApprove: () -> Unit,
    onDeny: () -> Unit,
    onReturn: () -> Unit,
) {
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text(studentName, fontWeight = FontWeight.SemiBold)
                Text(drawerString(BehaviorLogic.statusLabelRes(pass.status)))
            }
            Text(drawerString(BehaviorLogic.destinationLabelRes(pass.destination)))
            BehaviorLogic.hallPassCountdown(pass, now)?.let { countdown ->
                Text(
                    if (countdown.isExpired) {
                        L.text(R.string.mobile_hallpass_overdue)
                    } else {
                        L.format(R.string.mobile_hallpass_countdown, BehaviorLogic.formatCountdown(countdown))
                    },
                    fontWeight = FontWeight.SemiBold,
                )
            }
            when (pass.status.lowercase()) {
                "requested" -> Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    ActionChip(L.text(R.string.mobile_hallpass_approve), updating, onApprove)
                    ActionChip(L.text(R.string.mobile_hallpass_deny), updating, onDeny)
                }
                "approved" -> ActionChip(L.text(R.string.mobile_hallpass_return), updating, onReturn)
            }
        }
    }
}

@Composable
private fun ActionChip(label: String, busy: Boolean, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .clickable(enabled = !busy, onClick = onClick)
            .padding(vertical = 8.dp),
    ) {
        if (busy) CircularProgressIndicator() else Text(label, fontWeight = FontWeight.SemiBold)
    }
}

private suspend fun updatePass(
    session: AuthSession,
    passId: String,
    status: String,
    updatingPassId: String?,
    onUpdating: (String?) -> Unit,
    onDone: () -> Unit,
    onError: (String) -> Unit,
) {
    val token = session.accessToken.value ?: return
    onUpdating(passId)
    runCatching {
        LmsApi.updateHallPass(passId, status, token)
        onDone()
    }.onFailure {
        onError(it.message ?: "Update failed")
    }
    onUpdating(null)
}
