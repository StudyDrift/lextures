package com.lextures.android.features.paths

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import kotlinx.coroutines.launch
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.LearningPathDetail
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PathsLogic
import com.lextures.android.core.network.ApiError
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSectionHeader
import com.lextures.android.features.home.LmsSkeletonList
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Warning

@Composable
fun PathLandingScreen(
    session: AuthSession,
    slug: String,
    onEnrolled: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()

    var detail by remember { mutableStateOf<LearningPathDetail?>(null) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var enrolling by remember { mutableStateOf(false) }
    var enrollError by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(accessToken, slug) {
        loading = true
        errorMessage = null
        try {
            detail = LmsApi.fetchCatalogPathDetail(slug, accessToken)
            if (detail == null) {
                errorMessage = L.text(context, localePrefs, R.string.mobile_paths_landingNotFound)
            }
        } catch (e: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_paths_error_landing)
        } finally {
            loading = false
        }
    }

    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        androidx.compose.material3.TextButton(onClick = onBack) {
            Text(L.text(context, localePrefs, R.string.mobile_ia_close))
        }

        when {
            loading -> LmsSkeletonList(count = 3)
            errorMessage != null && detail == null -> LmsEmptyState(
                icon = Icons.Default.Warning,
                title = L.text(context, localePrefs, R.string.mobile_paths_landingErrorTitle),
                message = errorMessage!!,
            )
            detail != null -> {
                val pathDetail = detail!!
                LazyColumn(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    item {
                        LmsCard {
                            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                                Text(
                                    L.text(context, localePrefs, R.string.mobile_paths_landingBadge),
                                    fontSize = 12.sp,
                                    fontWeight = FontWeight.SemiBold,
                                    color = textSecondary(),
                                )
                                if (pathDetail.path.description.isNotBlank()) {
                                    Text(pathDetail.path.description, fontSize = 14.sp, color = textSecondary())
                                }
                                Text(
                                    context.getString(
                                        R.string.mobile_paths_landingMeta,
                                        pathDetail.courses.size,
                                        PathsLogic.formatDuration(pathDetail.totalDurationMinutes),
                                    ),
                                    fontSize = 12.sp,
                                    color = textSecondary(),
                                )
                            }
                        }
                    }
                    item {
                        LmsSectionHeader(L.text(context, localePrefs, R.string.mobile_paths_coursesInPath))
                    }
                    items(PathsLogic.sortedCourses(pathDetail.courses), key = { it.courseId }) { course ->
                        LmsCard {
                            Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                                Text(course.title, fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
                                Text(course.courseCode.uppercase(), fontSize = 11.sp, color = textSecondary())
                            }
                        }
                    }
                    item {
                        enrollError?.let { LmsErrorBanner(it) }
                        if (PathsLogic.isPaid(pathDetail.path.bundlePriceCents)) {
                            LmsCard {
                                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                                    Text(
                                        L.text(context, localePrefs, R.string.mobile_paths_paidTitle),
                                        fontSize = 15.sp,
                                        fontWeight = FontWeight.SemiBold,
                                        color = textPrimary(),
                                    )
                                    Text(
                                        L.text(context, localePrefs, R.string.mobile_paths_paidHint),
                                        fontSize = 12.sp,
                                        color = textSecondary(),
                                    )
                                    Button(
                                        onClick = {
                                            val url = AppConfiguration.webUrl(PathsLogic.catalogWebPath(slug))
                                            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                                        },
                                        modifier = Modifier.fillMaxWidth(),
                                    ) {
                                        Text(L.text(context, localePrefs, R.string.mobile_paths_openOnWeb))
                                    }
                                }
                            }
                        } else {
                            Button(
                                onClick = {
                                    val token = accessToken ?: return@Button
                                    enrolling = true
                                    enrollError = null
                                    scope.launch {
                                        try {
                                            LmsApi.enrollInPath(pathDetail.path.id, token)
                                            onEnrolled()
                                        } catch (e: ApiError.HttpStatus) {
                                            enrollError = if (e.code == 402) {
                                                L.text(context, localePrefs, R.string.mobile_paths_paidRequired)
                                            } else {
                                                L.text(context, localePrefs, R.string.mobile_paths_error_enroll)
                                            }
                                        } catch (e: Exception) {
                                            enrollError = L.text(context, localePrefs, R.string.mobile_paths_error_enroll)
                                        } finally {
                                            enrolling = false
                                        }
                                    }
                                },
                                enabled = !enrolling && accessToken != null,
                                modifier = Modifier.fillMaxWidth(),
                            ) {
                                Text(
                                    L.text(
                                        context,
                                        localePrefs,
                                        if (enrolling) R.string.mobile_paths_enrolling else R.string.mobile_paths_startFree,
                                    ),
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}