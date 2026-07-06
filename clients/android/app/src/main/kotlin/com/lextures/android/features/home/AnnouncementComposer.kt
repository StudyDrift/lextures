package com.lextures.android.features.home

import android.net.Uri
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
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
import com.lextures.android.R
import com.lextures.android.core.accessibility.DictationField
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.AnnouncementAudience
import com.lextures.android.core.lms.AnnouncementLogic
import com.lextures.android.core.lms.CourseSection
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.home.LmsSegmentedChips
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

@Composable
fun AnnouncementComposerScreen(
    session: AuthSession,
    course: CourseSummary,
    onDone: (Boolean) -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePreferences = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()

    var title by remember { mutableStateOf("") }
    var bodyText by remember { mutableStateOf("") }
    var audience by remember { mutableStateOf(AnnouncementAudience.WholeCourse) }
    var sections by remember { mutableStateOf<List<CourseSection>>(emptyList()) }
    var selectedSectionId by remember { mutableStateOf("") }
    var announcementsChannelId by remember { mutableStateOf<String?>(null) }
    var pendingImageUri by remember { mutableStateOf<Uri?>(null) }
    var sending by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var showConfirm by remember { mutableStateOf(false) }

    val photoPicker = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        pendingImageUri = uri
    }

    val selectedSectionName = if (audience == AnnouncementAudience.Section) {
        sections.firstOrNull { it.id == selectedSectionId }?.displayName
    } else {
        null
    }

    val canSend = AnnouncementLogic.canSubmitCourseAnnouncement(title, bodyText) && !sending && !loading

    BackHandler { onDone(false) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            val channels = LmsApi.fetchFeedChannels(course.courseCode, token)
            announcementsChannelId = AnnouncementLogic.announcementsChannelId(channels)
            sections = if (course.isSectionsEnabled) {
                LmsApi.fetchCourseSections(course.courseCode, token)
            } else {
                emptyList()
            }
            selectedSectionId = sections.firstOrNull()?.id.orEmpty()
            if (announcementsChannelId == null) {
                errorMessage = L.text(context, localePreferences, R.string.mobile_announcement_compose_no_channel)
            }
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    if (showConfirm) {
        AlertDialog(
            onDismissRequest = { showConfirm = false },
            title = { Text(L.text(R.string.mobile_announcement_compose_confirm_title)) },
            text = {
                Text(
                    L.format(
                        R.string.mobile_announcement_compose_confirm_message,
                        AnnouncementLogic.audienceLabel(course, audience, selectedSectionName),
                    ),
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        showConfirm = false
                        val token = accessToken ?: return@TextButton
                        sending = true
                        errorMessage = null
                        scope.launch {
                            try {
                                val channelId = announcementsChannelId
                                    ?: throw IllegalStateException("missing channel")
                                var composedBody = bodyText
                                pendingImageUri?.let { uri ->
                                    val bytes = withContext(Dispatchers.IO) {
                                        context.contentResolver.openInputStream(uri)?.use { it.readBytes() }
                                    } ?: throw IllegalStateException("could not read image")
                                    val upload = LmsApi.uploadFeedImage(
                                        courseCode = course.courseCode,
                                        imageData = bytes,
                                        fileName = "photo.jpg",
                                        mimeType = "image/jpeg",
                                        accessToken = token,
                                    )
                                    val markdown = "![image](${upload.contentPath})"
                                    composedBody = if (composedBody.trim().isEmpty()) {
                                        markdown
                                    } else {
                                        "$composedBody\n\n$markdown"
                                    }
                                }
                                LmsApi.createCourseAnnouncement(
                                    courseCode = course.courseCode,
                                    channelId = channelId,
                                    title = title,
                                    body = composedBody,
                                    sectionName = selectedSectionName,
                                    mentionsEveryone = audience == AnnouncementAudience.WholeCourse,
                                    accessToken = token,
                                )
                                onDone(true)
                            } catch (e: Exception) {
                                errorMessage = session.mapError(e)
                            } finally {
                                sending = false
                            }
                        }
                    },
                ) {
                    Text(L.text(R.string.mobile_announcement_compose_post))
                }
            },
            dismissButton = {
                TextButton(onClick = { showConfirm = false }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }

    Column(modifier = modifier.padding(bottom = 16.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp, end = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = { onDone(false) }) {
                Icon(Icons.Default.Close, contentDescription = L.text(R.string.mobile_common_cancel), tint = textPrimary())
            }
            Text(
                text = L.text(R.string.mobile_announcement_compose_nav_title),
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                modifier = Modifier.weight(1f),
            )
            if (sending) {
                CircularProgressIndicator(color = LexturesColors.Primary, strokeWidth = 2.dp)
            } else {
                TextButton(onClick = { showConfirm = true }, enabled = canSend) {
                    Text(
                        text = L.text(R.string.mobile_announcement_compose_review),
                        fontWeight = FontWeight.SemiBold,
                        color = if (canSend) LexturesColors.Primary else textSecondary(),
                    )
                }
            }
        }

        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            errorMessage?.let { LmsErrorBanner(message = it) }

            OutlinedTextField(
                value = title,
                onValueChange = { title = it },
                label = { Text(L.text(R.string.mobile_announcement_compose_title)) },
                modifier = Modifier.fillMaxWidth(),
                colors = OutlinedTextFieldDefaults.colors(
                    focusedTextColor = textPrimary(),
                    unfocusedTextColor = textPrimary(),
                ),
            )

            DictationField(
                title = L.text(R.string.mobile_announcement_compose_body),
                text = bodyText,
                onTextChange = { bodyText = it },
                placeholder = L.text(R.string.mobile_announcement_compose_body_placeholder),
            )

            Text(
                text = L.text(R.string.mobile_announcement_compose_audience),
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = textSecondary(),
            )
            val audienceOptions = buildList {
                add(L.text(R.string.mobile_announcement_compose_audience_whole_course) to AnnouncementAudience.WholeCourse)
                if (course.isSectionsEnabled && sections.isNotEmpty()) {
                    add(L.text(R.string.mobile_announcement_compose_audience_section) to AnnouncementAudience.Section)
                }
            }
            LmsSegmentedChips(
                options = audienceOptions.map { it.first },
                selectedIndex = audienceOptions.indexOfFirst { it.second == audience }.coerceAtLeast(0),
                onSelect = { index -> audience = audienceOptions[index].second },
            )

            if (audience == AnnouncementAudience.Section && sections.isNotEmpty()) {
                OutlinedTextField(
                    value = sections.firstOrNull { it.id == selectedSectionId }?.displayName.orEmpty(),
                    onValueChange = {},
                    readOnly = true,
                    label = { Text(L.text(R.string.mobile_attendance_take_section)) },
                    modifier = Modifier.fillMaxWidth(),
                    colors = OutlinedTextFieldDefaults.colors(
                        focusedTextColor = textPrimary(),
                        unfocusedTextColor = textPrimary(),
                    ),
                )
                LmsSegmentedChips(
                    options = sections.map { it.displayName },
                    selectedIndex = sections.indexOfFirst { it.id == selectedSectionId }.coerceAtLeast(0),
                    onSelect = { index -> selectedSectionId = sections[index].id },
                )
            }

            TextButton(onClick = { photoPicker.launch("image/*") }) {
                Text(
                    text = if (pendingImageUri == null) {
                        L.text(R.string.mobile_announcement_compose_add_photo)
                    } else {
                        L.text(R.string.mobile_announcement_compose_change_photo)
                    },
                )
            }
            if (pendingImageUri != null) {
                TextButton(onClick = { pendingImageUri = null }) {
                    Text(L.text(R.string.mobile_announcement_compose_remove_photo))
                }
            }
        }
    }
}