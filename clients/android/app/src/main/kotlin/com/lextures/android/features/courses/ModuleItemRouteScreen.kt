package com.lextures.android.features.courses

import android.content.Intent
import android.net.Uri
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
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
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ModuleContentLogic
import com.lextures.android.core.lms.ModuleItemDestination
import com.lextures.android.core.lms.FilePreviewTarget
import com.lextures.android.features.files.FilePreviewScreen
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.quiz.QuizIntroScreen

/** Routes a structure item to its native destination (M3.1). */
@Composable
fun ModuleItemRouteScreen(
    session: AuthSession,
    course: CourseSummary,
    item: CourseStructureItem,
    onBack: () -> Unit,
    onProgressChanged: suspend () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    when (ModuleContentLogic.destination(item.kind)) {
        ModuleItemDestination.ContentPage -> ContentPageScreen(
            session = session,
            course = course,
            item = item,
            onBack = onBack,
            onProgressChanged = onProgressChanged,
            modifier = modifier,
        )
        ModuleItemDestination.Quiz -> QuizIntroScreen(
            session = session,
            course = course,
            item = item,
            onBack = onBack,
            onProgressChanged = onProgressChanged,
            modifier = modifier,
        )
        ModuleItemDestination.Assignment -> ItemDetailScreen(
            session = session,
            course = course,
            item = item,
            onBack = onBack,
            modifier = modifier,
        )
        ModuleItemDestination.ExternalLink, ModuleItemDestination.WebContent -> WebItemLoaderScreen(
            session = session,
            course = course,
            item = item,
            onBack = onBack,
            modifier = modifier,
        )
        ModuleItemDestination.Interactive -> LaunchContainerScreen(
            session = session,
            course = course,
            item = item,
            onBack = onBack,
            onProgressChanged = onProgressChanged,
            modifier = modifier,
        )
        ModuleItemDestination.File -> FilePreviewScreen(
            session = session,
            target = FilePreviewTarget.from(item, course.courseCode),
            onBack = onBack,
            modifier = modifier,
        )
        ModuleItemDestination.Unsupported -> ModuleItemPlaceholderScreen(
            item = item,
            messageKey = "mobile.modules.placeholder.unsupported",
            onBack = onBack,
            modifier = modifier,
        )
    }
}

@Composable
fun ModuleItemPlaceholderScreen(
    item: CourseStructureItem,
    messageKey: String,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    BackHandler(onBack = onBack)
    val placeholderMessage = modulePlaceholderLabel(modulePlaceholderRes(messageKey))
    Column(modifier = modifier) {
        RowHeader(title = item.title, onBack = onBack)
        Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            LmsEmptyState(
                icon = ItemKind.icon(item.kind),
                title = item.title,
                message = placeholderMessage,
            )
        }
    }
}

@Composable
private fun WebItemLoaderScreen(
    session: AuthSession,
    course: CourseSummary,
    item: CourseStructureItem,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    var url by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    val noLinkLabel = moduleNoLinkLabel()
    val loadErrorFallback = moduleLoadErrorLabel()

    BackHandler(onBack = onBack)

    LaunchedEffect(accessToken, item.id) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        try {
            val detail = LmsApi.fetchItemDetail(course.courseCode, item, token)
            val link = detail?.url?.trim().orEmpty()
            if (link.isNotEmpty()) {
                url = link
            } else {
                errorMessage = noLinkLabel
            }
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    Column(modifier = modifier) {
        RowHeader(title = item.title, onBack = onBack)
        when {
            loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = LexturesColors.Primary)
            }
            url != null -> WebItemScreen(
                title = item.title,
                urlString = url!!,
                accessToken = accessToken,
                onOpenExternal = { uri ->
                    context.startActivity(Intent(Intent.ACTION_VIEW, uri))
                },
                modifier = Modifier.fillMaxSize(),
            )
            else -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                LmsEmptyState(
                    icon = Icons.AutoMirrored.Filled.ArrowBack,
                    title = item.title,
                    message = errorMessage ?: loadErrorFallback,
                )
            }
        }
    }
}

@Composable
internal fun RowHeader(title: String, onBack: () -> Unit) {
    androidx.compose.foundation.layout.Row(
        modifier = Modifier.fillMaxWidth().padding(top = 8.dp, end = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        IconButton(onClick = onBack) {
            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
        }
        Text(
            text = title,
            fontSize = 17.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}
