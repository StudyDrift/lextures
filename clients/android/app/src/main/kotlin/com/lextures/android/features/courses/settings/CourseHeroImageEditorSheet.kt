package com.lextures.android.features.courses.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseHeroImageURLRequest
import com.lextures.android.core.lms.CourseSettingsLogic
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineService
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val heroJson = Json { ignoreUnknownKeys = true }

@Composable
fun CourseHeroImageEditorSheet(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    onDismiss: () -> Unit,
    onSaved: (CourseSummary) -> Unit,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var prompt by remember(course.courseCode) {
        mutableStateOf(CourseSettingsLogic.defaultImagePrompt(course.title, course.description))
    }
    var previewUrl by remember { mutableStateOf<String?>(null) }
    var status by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }

    Dialog(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(L.text(R.string.mobile_courseSettings_heroImage))
            OutlinedTextField(
                value = prompt,
                onValueChange = { prompt = it },
                label = { Text(L.text(R.string.mobile_courseSettings_hero_prompt)) },
                modifier = Modifier.fillMaxWidth(),
            )
            Button(
                onClick = {
                    scope.launch {
                        val token = session.accessToken.value ?: return@launch
                        busy = true
                        runCatching {
                            previewUrl = LmsApi.generateCourseImage(course.courseCode, prompt.trim(), token).imageUrl
                        }.onFailure { status = it.message }
                        busy = false
                    }
                },
                enabled = !busy && prompt.isNotBlank(),
            ) {
                Text(L.text(R.string.mobile_courseSettings_hero_generatePreview))
            }
            previewUrl?.let { url ->
                Text(L.text(R.string.mobile_courseSettings_hero_generatedLabel))
                Button(
                    onClick = {
                        scope.launch {
                            val token = session.accessToken.value ?: return@launch
                            busy = true
                            runCatching {
                                offline.enqueueMutation(
                                    method = "PUT",
                                    path = "/api/v1/courses/${course.courseCode}/hero-image",
                                    bodyJson = heroJson.encodeToString(
                                        CourseHeroImageURLRequest.serializer(),
                                        CourseHeroImageURLRequest(url),
                                    ),
                                    label = L.text(context, localePrefs, R.string.mobile_courseSettings_hero_saveLabel),
                                    accessToken = token,
                                    idempotencyKey = "course-hero:${course.courseCode}:image",
                                )
                                onSaved(LmsApi.fetchCourse(course.courseCode, token))
                            }.onFailure { status = it.message }
                            busy = false
                        }
                    },
                    enabled = !busy,
                ) {
                    Text(L.text(R.string.mobile_courseSettings_hero_saveImage))
                }
            }
            status?.let { Text(it) }
            Button(onClick = onDismiss) { Text(L.text(R.string.mobile_common_cancel)) }
        }
    }
}
