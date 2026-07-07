package com.lextures.android.features.courses.settings

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.AddCrossListMemberBody
import com.lextures.android.core.lms.CourseSection
import com.lextures.android.core.lms.CourseSectionsLogic
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.CreateCrossListGroupBody
import com.lextures.android.core.lms.CrossListGroup
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch

@Composable
fun CourseCrossListingSection(
    session: AuthSession,
    course: CourseSummary,
    sections: List<CourseSection>,
    permissions: List<String>,
    onReload: suspend () -> Unit,
) {
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val canOrgAdmin = CourseSectionsLogic.canManageCrossListing(permissions)
    val orgId = course.orgId?.trim()?.takeIf { it.isNotEmpty() }
    val activeSections = remember(sections) { CourseSectionsLogic.activeSections(sections) }

    var group by remember { mutableStateOf<CrossListGroup?>(null) }
    var loading by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf(false) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var actionSuccess by remember { mutableStateOf<String?>(null) }
    var primaryPick by remember { mutableStateOf("") }
    var groupName by remember { mutableStateOf("") }
    var addPick by remember { mutableStateOf("") }
    var pendingRemoveSectionId by remember { mutableStateOf<String?>(null) }

    val addCandidates = remember(activeSections, group) {
        CourseSectionsLogic.crossListAddCandidates(activeSections, group)
    }

    LaunchedEffect(accessToken, course.courseCode, canOrgAdmin, orgId) {
        val token = accessToken ?: return@LaunchedEffect
        if (!canOrgAdmin || orgId == null) return@LaunchedEffect
        loading = true
        loadError = null
        runCatching {
            val groups = LmsApi.fetchOrgCrossListGroups(orgId, token)
            group = CourseSectionsLogic.crossListGroup(course.id, groups)
        }.onFailure { loadError = session.mapError(it) }
        loading = false
    }

    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(L.text(R.string.mobile_courseSettings_sections_crossListingTitle), fontWeight = FontWeight.SemiBold)
            Text(L.text(R.string.mobile_courseSettings_sections_crossListingDescription))

            if (!canOrgAdmin || orgId == null) {
                Text(L.text(R.string.mobile_courseSettings_sections_crossListingAdminHint))
            } else {
                loadError?.let { LmsErrorBanner(it) }
                actionError?.let { LmsErrorBanner(it) }
                actionSuccess?.let {
                    Text(it, color = androidx.compose.ui.graphics.Color(0xFF0D9488), fontWeight = FontWeight.SemiBold)
                }

                when {
                    loading && group == null -> CircularProgressIndicator()
                    group == null && activeSections.size < 2 ->
                        Text(L.text(R.string.mobile_courseSettings_sections_crossListingNeedTwoSections))
                    group == null -> {
                        Text(L.text(R.string.mobile_courseSettings_sections_crossListingCreateTitle), fontWeight = FontWeight.Medium)
                        OutlinedTextField(
                            value = primaryPick,
                            onValueChange = { primaryPick = it },
                            modifier = Modifier.fillMaxWidth(),
                            label = { Text(L.text(R.string.mobile_courseSettings_sections_crossListingPrimarySection)) },
                            placeholder = { Text(L.text(R.string.mobile_courseSettings_sections_selectPlaceholder)) },
                        )
                        OutlinedTextField(
                            value = groupName,
                            onValueChange = { groupName = it },
                            modifier = Modifier.fillMaxWidth(),
                            label = { Text(L.text(R.string.mobile_courseSettings_sections_crossListingLabelOptional)) },
                        )
                        Button(
                            onClick = {
                                scope.launch {
                                    val token = accessToken ?: return@launch
                                    busy = true
                                    actionError = null
                                    runCatching {
                                        LmsApi.postOrgCrossListGroup(
                                            orgId,
                                            CreateCrossListGroupBody(
                                                courseCode = course.courseCode,
                                                primarySectionId = primaryPick,
                                                name = groupName.trim().ifEmpty { null },
                                            ),
                                            token,
                                        )
                                        primaryPick = ""
                                        groupName = ""
                                        actionSuccess = L.text(R.string.mobile_courseSettings_sections_crossListingCreateSuccess)
                                        val groups = LmsApi.fetchOrgCrossListGroups(orgId, token)
                                        group = CourseSectionsLogic.crossListGroup(course.id, groups)
                                        onReload()
                                    }.onFailure { actionError = session.mapError(it) }
                                    busy = false
                                }
                            },
                            enabled = !busy && primaryPick.isNotEmpty(),
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text(L.text(R.string.mobile_courseSettings_sections_crossListingCreateButton))
                        }
                    }
                    group != null -> {
                        val current = group!!
                        Text(
                            current.name?.trim()?.takeIf { it.isNotEmpty() }
                                ?: L.text(R.string.mobile_courseSettings_sections_crossListingDefaultName),
                            fontWeight = FontWeight.Medium,
                        )
                        Text(L.format(R.string.mobile_courseSettings_sections_crossListingMemberCount, current.members.size))
                        current.members.forEach { member ->
                            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                Text(member.displayLabel)
                                if (!member.isPrimary) {
                                    TextButton(onClick = { pendingRemoveSectionId = member.sectionId }) {
                                        Text(L.text(R.string.mobile_courseSettings_sections_crossListingRemove))
                                    }
                                }
                            }
                        }
                        if (addCandidates.isNotEmpty()) {
                            OutlinedTextField(
                                value = addPick,
                                onValueChange = { addPick = it },
                                modifier = Modifier.fillMaxWidth(),
                                label = { Text(L.text(R.string.mobile_courseSettings_sections_crossListingAddSection)) },
                            )
                            OutlinedButton(
                                onClick = {
                                    scope.launch {
                                        val token = accessToken ?: return@launch
                                        busy = true
                                        runCatching {
                                            LmsApi.postOrgCrossListMember(orgId, current.id, addPick, token)
                                            addPick = ""
                                            actionSuccess = L.text(R.string.mobile_courseSettings_sections_crossListingAddSuccess)
                                            val groups = LmsApi.fetchOrgCrossListGroups(orgId, token)
                                            group = CourseSectionsLogic.crossListGroup(course.id, groups)
                                            onReload()
                                        }.onFailure { actionError = session.mapError(it) }
                                        busy = false
                                    }
                                },
                                enabled = !busy && addPick.isNotEmpty(),
                                modifier = Modifier.fillMaxWidth(),
                            ) {
                                Text(L.text(R.string.mobile_courseSettings_sections_crossListingAddButton))
                            }
                        } else {
                            Text(L.text(R.string.mobile_courseSettings_sections_crossListingAllLinked))
                        }
                    }
                }
            }
        }
    }

    pendingRemoveSectionId?.let { sectionId ->
        AlertDialog(
            onDismissRequest = { pendingRemoveSectionId = null },
            title = { Text(L.text(R.string.mobile_courseSettings_sections_crossListingRemoveConfirmTitle)) },
            text = { Text(L.text(R.string.mobile_courseSettings_sections_crossListingRemoveConfirmMessage)) },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        val token = accessToken ?: return@launch
                        val current = group ?: return@launch
                        val oid = orgId ?: return@launch
                        busy = true
                        runCatching {
                            LmsApi.deleteOrgCrossListMember(oid, current.id, sectionId, token)
                            actionSuccess = L.text(R.string.mobile_courseSettings_sections_crossListingRemoveSuccess)
                            val groups = LmsApi.fetchOrgCrossListGroups(oid, token)
                            group = CourseSectionsLogic.crossListGroup(course.id, groups)
                            onReload()
                        }.onFailure { actionError = session.mapError(it) }
                        busy = false
                        pendingRemoveSectionId = null
                    }
                }) {
                    Text(L.text(R.string.mobile_courseSettings_sections_crossListingRemove))
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingRemoveSectionId = null }) {
                    Text(L.text(R.string.mobile_courseSettings_sections_cancel))
                }
            },
        )
    }
}
