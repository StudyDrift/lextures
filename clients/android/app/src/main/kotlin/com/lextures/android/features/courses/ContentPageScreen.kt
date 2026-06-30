package com.lextures.android.features.courses

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
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
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.accessibility.ReadAloudControls
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ModuleContentLogic
import com.lextures.android.core.lms.ModuleItemDetail
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.notebooks.NotebookContentView
import kotlinx.coroutines.launch

/** Native content page reader with offline cache and completion (M3.1). */
@Composable
fun ContentPageScreen(
    session: AuthSession,
    course: CourseSummary,
    item: CourseStructureItem,
    onBack: () -> Unit,
    onProgressChanged: suspend () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()

    var detail by remember { mutableStateOf<ModuleItemDetail?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var markingComplete by remember { mutableStateOf(false) }
    var isComplete by remember { mutableStateOf(false) }

    val markDoneLabel = moduleMarkDoneLabel()
    val markingDoneLabel = moduleMarkingDoneLabel()
    val completeLabel = moduleCompleteLabel()

    BackHandler(onBack = onBack)

    LaunchedEffect(accessToken, item.id) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.contentPage(course.courseCode, item.id),
                accessToken = token,
                serializer = ModuleItemDetail.serializer(),
            ) {
                LmsApi.fetchItemDetail(course.courseCode, item, token)
                    ?: throw IllegalStateException("missing content page")
            }
            detail = result.first
            val cached = result.second
            cacheLabel = if (cached != null && cached.isStale(isOnline)) cached.lastUpdatedLabel() else null
            isComplete = ModuleContentLogic.isComplete(
                LmsApi.fetchModulesProgress(course.courseCode, token),
                item.id,
            )
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    Column(modifier = modifier) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp, end = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = item.title,
                fontSize = 17.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(
                        text = detail?.title ?: item.title,
                        style = LexturesType.display(22),
                        color = textPrimary(),
                    )
                    if (isComplete) {
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(6.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Icon(Icons.Default.CheckCircle, contentDescription = null, tint = LexturesColors.Primary)
                            Text(
                                text = completeLabel,
                                fontSize = 12.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = LexturesColors.Primary,
                            )
                        }
                    }
                }
            }
            errorMessage?.let { message -> item { LmsErrorBanner(message) } }
            cacheLabel?.let { label -> item { StalenessChip(label = label) } }
            if (loading) {
                item {
                    Box(Modifier.fillMaxWidth().padding(vertical = 40.dp), contentAlignment = Alignment.Center) {
                        CircularProgressIndicator(color = LexturesColors.Primary)
                    }
                }
            } else {
                detail?.markdown?.trim()?.takeIf { it.isNotEmpty() }?.let { markdown ->
                    item {
                        LmsCard {
                            ReadAloudControls(text = markdown)
                            NotebookContentView(
                                markdown = markdown,
                                onToggleTask = {},
                                onEditTaskDue = {},
                                accessToken = accessToken,
                            )
                        }
                    }
                }
                if (!loading && course.viewerIsStudent && !isComplete) {
                    item {
                        AuthPrimaryButton(
                            text = if (markingComplete) markingDoneLabel else markDoneLabel,
                            onClick = {
                                val token = accessToken ?: return@AuthPrimaryButton
                                scope.launch {
                                    markingComplete = true
                                    try {
                                        offline.enqueueMutation(
                                            method = "POST",
                                            path = "/api/v1/courses/${course.courseCode}/items/${item.id}/complete",
                                            bodyJson = null,
                                            label = markDoneLabel,
                                            accessToken = token,
                                        )
                                        isComplete = true
                                        onProgressChanged()
                                    } catch (e: Exception) {
                                        errorMessage = session.mapError(e)
                                    } finally {
                                        markingComplete = false
                                    }
                                }
                            },
                            enabled = !markingComplete,
                            modifier = Modifier.fillMaxWidth(),
                        )
                    }
                }
            }
        }
    }
}
