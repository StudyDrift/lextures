package com.lextures.android.features.behavior

import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.RadioButtonUnchecked
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import androidx.compose.ui.platform.LocalContext
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.BehaviorAwardMode
import com.lextures.android.core.lms.BehaviorCategory
import com.lextures.android.core.lms.BehaviorLogic
import com.lextures.android.core.lms.BehaviorReferralBody
import com.lextures.android.core.lms.CourseEnrollment
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSegmentedChips
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@Composable
fun BehaviorRosterScreen(
    session: AuthSession,
    course: CourseSummary,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current

    var enrollments by remember { mutableStateOf<List<CourseEnrollment>>(emptyList()) }
    var categories by remember { mutableStateOf<List<BehaviorCategory>>(emptyList()) }
    var selectedStudents by remember { mutableStateOf(setOf<String>()) }
    var selectedCategoryId by remember { mutableStateOf("") }
    var awardNote by remember { mutableStateOf("") }
    var mode by remember { mutableStateOf(BehaviorAwardMode.Award) }
    var refStudentId by remember { mutableStateOf("") }
    var refCategoryId by remember { mutableStateOf("") }
    var refDescription by remember { mutableStateOf("") }
    var refLocation by remember { mutableStateOf("") }
    var refResponse by remember { mutableStateOf("") }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var successMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var reloadKey by remember { mutableStateOf(0) }

    val roster = remember(enrollments) { BehaviorLogic.studentRoster(enrollments) }
    val positiveCategories = remember(categories) { BehaviorLogic.positiveCategories(categories) }
    val negativeCategories = remember(categories) { BehaviorLogic.negativeCategories(categories) }

    LaunchedEffect(accessToken, reloadKey) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        runCatching {
            enrollments = LmsApi.fetchCourseEnrollments(course.courseCode, token)
            val orgId = course.orgId
            if (!orgId.isNullOrBlank()) {
                categories = LmsApi.listBehaviorCategories(orgId, token)
            }
        }.onFailure {
            errorMessage = it.message ?: L.text(context, localePrefs, R.string.mobile_behavior_loadError)
        }
        loading = false
    }

    Column(
        modifier = modifier
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        errorMessage?.let { LmsErrorBanner(message = it) }
        successMessage?.let {
            Text(text = it, modifier = Modifier.padding(horizontal = 4.dp))
        }

        if (loading && roster.isEmpty()) {
            LmsSkeletonList(count = 4)
        } else {
            LmsSegmentedChips(
                options = listOf(
                    "award" to L.text(R.string.mobile_behavior_mode_award),
                    "referral" to L.text(R.string.mobile_behavior_mode_referral),
                ),
                selectedId = if (mode == BehaviorAwardMode.Award) "award" else "referral",
                onSelect = { selected ->
                    mode = if (selected == "award") BehaviorAwardMode.Award else BehaviorAwardMode.Referral
                },
            )

            if (mode == BehaviorAwardMode.Award) {
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        Text(L.text(R.string.mobile_behavior_award_hint))
                        CategoryChips(positiveCategories, selectedCategoryId) { selectedCategoryId = it }
                        TextField(
                            value = awardNote,
                            onValueChange = { awardNote = it },
                            label = { Text(L.text(R.string.mobile_behavior_noteOptional)) },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        SubmitButton(
                            title = L.text(R.string.mobile_behavior_award_submit),
                            saving = saving,
                            enabled = selectedStudents.isNotEmpty() && selectedCategoryId.isNotEmpty(),
                        ) {
                            val token = accessToken ?: return@SubmitButton
                            scope.launch {
                                saving = true
                                errorMessage = null
                                successMessage = null
                                runCatching {
                                    val payload = BehaviorLogic.awardPayload(selectedStudents, selectedCategoryId, awardNote)
                                    val result = LmsApi.awardPbisPoints(payload, token)
                                    val count = result.saved ?: payload.size
                                    successMessage = context.getString(R.string.mobile_behavior_award_success, count)
                                    selectedStudents = emptySet()
                                    awardNote = ""
                                }.onFailure {
                                    errorMessage = it.message ?: L.text(context, localePrefs, R.string.mobile_behavior_award_error)
                                }
                                saving = false
                            }
                        }
                    }
                }
            } else {
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        Text(L.text(R.string.mobile_behavior_referral_hint))
                        CategoryChips(negativeCategories, refCategoryId) { refCategoryId = it }
                        TextField(
                            value = refDescription,
                            onValueChange = { refDescription = it },
                            label = { Text(L.text(R.string.mobile_behavior_referral_description)) },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        TextField(
                            value = refLocation,
                            onValueChange = { refLocation = it },
                            label = { Text(L.text(R.string.mobile_behavior_referral_locationOptional)) },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        SubmitButton(
                            title = L.text(R.string.mobile_behavior_referral_submit),
                            saving = saving,
                            enabled = refStudentId.isNotEmpty() && refCategoryId.isNotEmpty() && refDescription.isNotBlank(),
                        ) {
                            val token = accessToken ?: return@SubmitButton
                            scope.launch {
                                saving = true
                                errorMessage = null
                                successMessage = null
                                runCatching {
                                    LmsApi.fileBehaviorReferral(
                                        BehaviorReferralBody(
                                            studentId = refStudentId,
                                            categoryId = refCategoryId,
                                            location = refLocation.trim().ifBlank { null },
                                            description = refDescription.trim(),
                                            response = refResponse.trim().ifBlank { null },
                                        ),
                                        token,
                                    )
                                    successMessage = L.text(context, localePrefs, R.string.mobile_behavior_referral_success)
                                    refDescription = ""
                                    refLocation = ""
                                    refResponse = ""
                                }.onFailure {
                                    errorMessage = it.message ?: L.text(context, localePrefs, R.string.mobile_behavior_referral_error)
                                }
                                saving = false
                            }
                        }
                    }
                }
            }

            LmsCard {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(L.text(R.string.mobile_behavior_roster), fontWeight = FontWeight.SemiBold)
                        if (mode == BehaviorAwardMode.Award) {
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                Text(
                                    L.text(R.string.mobile_behavior_selectAll),
                                    modifier = Modifier.clickable { selectedStudents = roster.map { it.userId }.toSet() },
                                )
                                Text(
                                    L.text(R.string.mobile_behavior_clearAll),
                                    modifier = Modifier.clickable { selectedStudents = emptySet() },
                                )
                            }
                        }
                    }
                    roster.forEach { student ->
                        val selected = when (mode) {
                            BehaviorAwardMode.Award -> selectedStudents.contains(student.userId)
                            BehaviorAwardMode.Referral -> refStudentId == student.userId
                        }
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
                                    when (mode) {
                                        BehaviorAwardMode.Award -> {
                                            selectedStudents = if (selected) {
                                                selectedStudents - student.userId
                                            } else {
                                                selectedStudents + student.userId
                                            }
                                        }
                                        BehaviorAwardMode.Referral -> refStudentId = student.userId
                                    }
                                }
                                .padding(vertical = 4.dp),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(10.dp),
                        ) {
                            Icon(
                                imageVector = if (selected) Icons.Filled.CheckCircle else Icons.Filled.RadioButtonUnchecked,
                                contentDescription = null,
                            )
                            Text(BehaviorLogic.studentLabel(student))
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun CategoryChips(
    categories: List<BehaviorCategory>,
    selectedId: String,
    onSelect: (String) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState()),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        categories.forEach { category ->
            FilterChip(
                selected = selectedId == category.id,
                onClick = { onSelect(category.id) },
                label = { Text(category.name) },
            )
        }
    }
}

@Composable
private fun SubmitButton(
    title: String,
    saving: Boolean,
    enabled: Boolean,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(enabled = enabled && !saving, onClick = onClick)
            .padding(vertical = 11.dp),
        horizontalArrangement = Arrangement.Center,
    ) {
        if (saving) {
            CircularProgressIndicator()
        } else {
            Text(title, fontWeight = FontWeight.SemiBold)
        }
    }
}
