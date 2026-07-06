package com.lextures.android.features.behavior

import android.content.Context
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.BehaviorLogic
import com.lextures.android.core.lms.CourseSection
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.HallPass
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.navigation.drawerString
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import java.time.Instant

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MyHallPassScreen(
    session: AuthSession,
    course: CourseSummary,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    val json = remember { Json { ignoreUnknownKeys = true } }

    var sections by remember { mutableStateOf<List<CourseSection>>(emptyList()) }
    var selectedSectionId by remember { mutableStateOf("") }
    var destination by remember { mutableStateOf(BehaviorLogic.hallPassDestinations.first()) }
    var estimatedMins by remember { mutableIntStateOf(BehaviorLogic.defaultPassMinutes) }
    var activePass by remember { mutableStateOf<HallPass?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var successMessage by remember { mutableStateOf<String?>(null) }
    var submitting by remember { mutableStateOf(false) }
    var now by remember { mutableStateOf(Instant.now()) }

    LaunchedEffect(Unit) {
        while (true) {
            delay(1000)
            now = Instant.now()
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        runCatching {
            sections = LmsApi.fetchCourseSections(course.courseCode, token)
            if (selectedSectionId.isBlank()) {
                selectedSectionId = sections.firstOrNull()?.id.orEmpty()
            }
            activePass = loadStoredPass(context, json, selectedSectionId)
        }.onFailure {
            errorMessage = it.message ?: L.text(context, localePrefs, R.string.mobile_hallpass_loadError)
        }
    }

    LaunchedEffect(selectedSectionId) {
        activePass = loadStoredPass(context, json, selectedSectionId)
    }

    Column(
        modifier = modifier
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        errorMessage?.let { LmsErrorBanner(message = it) }
        successMessage?.let { Text(it) }

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
                                },
                            )
                        }
                    }
                }
            }
        }

        val pass = activePass
        if (pass != null && BehaviorLogic.isActiveHallPass(pass)) {
            LmsCard {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text(L.text(R.string.mobile_hallpass_student_activeTitle), fontWeight = FontWeight.SemiBold)
                    Text(drawerString(BehaviorLogic.destinationLabelRes(pass.destination)))
                    Text(drawerString(BehaviorLogic.statusLabelRes(pass.status)))
                    BehaviorLogic.hallPassCountdown(pass, now)?.let { countdown ->
                        Text(
                            if (countdown.isExpired) {
                                L.text(R.string.mobile_hallpass_overdue)
                            } else {
                                L.format(R.string.mobile_hallpass_countdown, BehaviorLogic.formatCountdown(countdown))
                            },
                            fontWeight = FontWeight.Bold,
                        )
                    }
                    if (pass.status.equals("approved", ignoreCase = true)) {
                        StudentActionButton(submitting, L.text(R.string.mobile_hallpass_imBack)) {
                            val token = accessToken ?: return@StudentActionButton
                            scope.launch {
                                submitting = true
                                runCatching {
                                    LmsApi.updateHallPass(pass.id, "returned", token)
                                    clearStoredPass(context, selectedSectionId)
                                    activePass = null
                                    successMessage = L.text(context, localePrefs, R.string.mobile_hallpass_returnedSuccess)
                                }.onFailure {
                                    errorMessage = it.message ?: L.text(context, localePrefs, R.string.mobile_hallpass_updateError)
                                }
                                submitting = false
                            }
                        }
                    } else {
                        Text(L.text(R.string.mobile_hallpass_student_pendingHint))
                    }
                }
            }
        } else {
            LmsCard {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text(L.text(R.string.mobile_hallpass_student_requestHint))
                    var destExpanded by remember { mutableStateOf(false) }
                    ExposedDropdownMenuBox(expanded = destExpanded, onExpandedChange = { destExpanded = it }) {
                        TextField(
                            value = drawerString(BehaviorLogic.destinationLabelRes(destination)),
                            onValueChange = {},
                            readOnly = true,
                            label = { Text(L.text(R.string.mobile_hallpass_destination)) },
                            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(destExpanded) },
                            modifier = Modifier
                                .fillMaxWidth()
                                .menuAnchor(),
                        )
                        ExposedDropdownMenu(expanded = destExpanded, onDismissRequest = { destExpanded = false }) {
                            BehaviorLogic.hallPassDestinations.forEach { value ->
                                DropdownMenuItem(
                                    text = { Text(drawerString(BehaviorLogic.destinationLabelRes(value))) },
                                    onClick = {
                                        destination = value
                                        destExpanded = false
                                    },
                                )
                            }
                        }
                    }
                    Text(L.format(R.string.mobile_hallpass_duration, estimatedMins))
                    Slider(
                        value = estimatedMins.toFloat(),
                        onValueChange = { estimatedMins = it.toInt() },
                        valueRange = 1f..30f,
                        steps = 28,
                    )
                    StudentActionButton(submitting, L.text(R.string.mobile_hallpass_request)) {
                        val token = accessToken ?: return@StudentActionButton
                        scope.launch {
                            submitting = true
                            errorMessage = null
                            successMessage = null
                            runCatching {
                                val created = LmsApi.requestHallPass(selectedSectionId, destination, estimatedMins, token)
                                storePass(context, json, selectedSectionId, created)
                                activePass = created
                                successMessage = L.text(context, localePrefs, R.string.mobile_hallpass_requested)
                            }.onFailure {
                                errorMessage = it.message ?: L.text(context, localePrefs, R.string.mobile_hallpass_requestError)
                            }
                            submitting = false
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun StudentActionButton(submitting: Boolean, label: String, onClick: () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(enabled = !submitting, onClick = onClick)
            .padding(vertical = 8.dp),
        horizontalAlignment = androidx.compose.ui.Alignment.CenterHorizontally,
    ) {
        if (submitting) {
            CircularProgressIndicator()
        } else {
            Text(label, fontWeight = FontWeight.SemiBold)
        }
    }
}

private fun storePass(context: Context, json: Json, sectionId: String, pass: HallPass) {
    context.getSharedPreferences("hall_pass", Context.MODE_PRIVATE)
        .edit()
        .putString(BehaviorLogic.storedPassKey(sectionId), json.encodeToString(HallPass.serializer(), pass))
        .apply()
}

private fun loadStoredPass(context: Context, json: Json, sectionId: String): HallPass? {
    if (sectionId.isBlank()) return null
    val raw = context.getSharedPreferences("hall_pass", Context.MODE_PRIVATE)
        .getString(BehaviorLogic.storedPassKey(sectionId), null) ?: return null
    return runCatching { json.decodeFromString(HallPass.serializer(), raw) }
        .getOrNull()
        ?.takeIf(BehaviorLogic::isActiveHallPass)
}

private fun clearStoredPass(context: Context, sectionId: String) {
    context.getSharedPreferences("hall_pass", Context.MODE_PRIVATE)
        .edit()
        .remove(BehaviorLogic.storedPassKey(sectionId))
        .apply()
}
