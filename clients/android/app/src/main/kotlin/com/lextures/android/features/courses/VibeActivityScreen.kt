package com.lextures.android.features.courses

import android.content.Intent
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.AutoAwesome
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.RadioButtonUnchecked
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.Language
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.net.toUri
import com.lextures.android.core.accessibility.ReadAloudControls
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ModuleLastVisited
import com.lextures.android.core.lms.ModuleVibeActivityPayload
import com.lextures.android.core.lms.VibeActivityBlock
import com.lextures.android.core.lms.VibeActivityBlockKind
import com.lextures.android.core.lms.VibeActivityDocument
import com.lextures.android.core.lms.VibeActivityLogic
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.notebooks.NotebookContentView
/** Native vibe activity reader (M3.5): parses instructor HTML into markdown + interactions. */
@Composable
fun VibeActivityScreen(
    session: AuthSession,
    course: CourseSummary,
    item: CourseStructureItem,
    nativeEnabled: Boolean = true,
    onBack: () -> Unit,
    onProgressChanged: suspend () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var payload by remember { mutableStateOf<ModuleVibeActivityPayload?>(null) }
    var document by remember { mutableStateOf(VibeActivityDocument(emptyList(), false)) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    val revealedIds = remember { mutableStateMapOf<Int, Boolean>() }
    val checkedIds = remember { mutableStateMapOf<Int, Boolean>() }
    val freeResponses = remember { mutableStateMapOf<Int, String>() }

    BackHandler(onBack = onBack)

    LaunchedEffect(accessToken, item.id, nativeEnabled) {
        if (!nativeEnabled) {
            loading = false
            return@LaunchedEffect
        }
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.vibeActivity(course.courseCode, item.id),
                accessToken = token,
                serializer = ModuleVibeActivityPayload.serializer(),
            ) {
                LmsApi.fetchModuleVibeActivity(course.courseCode, item.id, token)
            }
            payload = result.first
            document = VibeActivityLogic.parse(result.first.html)
            val cached = result.second
            cacheLabel = if (cached != null && cached.isStale(isOnline)) cached.lastUpdatedLabel() else null
            ModuleLastVisited.record(
                context = context,
                courseCode = course.courseCode,
                itemId = item.id,
                kind = item.kind,
                title = result.first.title.ifEmpty { item.title },
            )
            onProgressChanged()
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    Column(modifier = modifier) {
        RowHeader(title = payload?.title ?: item.title, onBack = onBack)

        if (!nativeEnabled) {
            Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Column(
                    modifier = Modifier.padding(24.dp),
                    verticalArrangement = Arrangement.spacedBy(16.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    LmsEmptyState(
                        icon = Icons.Default.AutoAwesome,
                        title = item.title,
                        message = vibeWebOnlyMessage(),
                    )
                    OpenOnWebButton(courseCode = course.courseCode, itemId = item.id)
                }
            }
            return
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(
                        text = payload?.title ?: item.title,
                        style = LexturesType.display(22),
                        color = textPrimary(),
                    )
                    Text(
                        text = vibeActivityLabel(),
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = com.lextures.android.core.design.accentColor(),
                    )
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
                if (document.requiresWebFallback) {
                    item { OpenOnWebCard(courseCode = course.courseCode, itemId = item.id, primary = true) }
                }
                items(document.blocks.size) { index ->
                    val block = document.blocks[index]
                    VibeActivityBlockView(
                        block = block,
                        revealed = revealedIds[block.id] == true,
                        checked = checkedIds[block.id] == true,
                        freeResponse = freeResponses[block.id].orEmpty(),
                        onToggleReveal = { revealedIds[block.id] = !(revealedIds[block.id] ?: false) },
                        onToggleCheck = { checkedIds[block.id] = !(checkedIds[block.id] ?: false) },
                        onFreeResponseChange = { freeResponses[block.id] = it },
                        courseCode = course.courseCode,
                        itemId = item.id,
                        accessToken = accessToken,
                    )
                }
            }
        }
    }
}

@Composable
private fun VibeActivityBlockView(
    block: VibeActivityBlock,
    revealed: Boolean,
    checked: Boolean,
    freeResponse: String,
    onToggleReveal: () -> Unit,
    onToggleCheck: () -> Unit,
    onFreeResponseChange: (String) -> Unit,
    courseCode: String,
    itemId: String,
    accessToken: String?,
) {
    when (val kind = block.kind) {
        is VibeActivityBlockKind.Heading -> LmsCard {
            NotebookContentView(markdown = "${"#".repeat(kind.level.coerceAtMost(3))} ${kind.text}", onToggleTask = {}, onEditTaskDue = {}, accessToken = accessToken)
        }
        is VibeActivityBlockKind.Paragraph -> LmsCard {
            ReadAloudControls(text = kind.text)
            NotebookContentView(markdown = kind.text, onToggleTask = {}, onEditTaskDue = {}, accessToken = accessToken)
        }
        is VibeActivityBlockKind.BulletList -> LmsCard {
            NotebookContentView(
                markdown = kind.items.joinToString("\n") { "- $it" },
                onToggleTask = {},
                onEditTaskDue = {},
                accessToken = accessToken,
            )
        }
        is VibeActivityBlockKind.OrderedList -> LmsCard {
            NotebookContentView(
                markdown = kind.items.mapIndexed { index, item -> "${index + 1}. $item" }.joinToString("\n"),
                onToggleTask = {},
                onEditTaskDue = {},
                accessToken = accessToken,
            )
        }
        is VibeActivityBlockKind.Reveal -> LmsCard {
            TextButton(onClick = onToggleReveal, modifier = Modifier.fillMaxWidth()) {
                Icon(Icons.Default.Visibility, contentDescription = null)
                Text(text = kind.trigger.ifEmpty { vibeRevealLabel() }, modifier = Modifier.padding(start = 8.dp))
            }
            if (revealed && kind.body.isNotEmpty()) {
                HorizontalDivider(modifier = Modifier.padding(vertical = 4.dp))
                NotebookContentView(markdown = kind.body, onToggleTask = {}, onEditTaskDue = {}, accessToken = accessToken)
            }
        }
        is VibeActivityBlockKind.CheckButton -> LmsCard {
            TextButton(onClick = onToggleCheck, modifier = Modifier.fillMaxWidth()) {
                Icon(
                    if (checked) Icons.Default.CheckCircle else Icons.Default.RadioButtonUnchecked,
                    contentDescription = null,
                )
                Text(text = kind.label, modifier = Modifier.padding(start = 8.dp))
            }
            kind.feedback?.takeIf { it.isNotEmpty() && checked }?.let {
                Text(text = it, fontSize = 12.sp, color = textSecondary())
            }
        }
        is VibeActivityBlockKind.FreeResponse -> LmsCard {
            if (kind.prompt.isNotEmpty()) {
                NotebookContentView(markdown = kind.prompt, onToggleTask = {}, onEditTaskDue = {}, accessToken = accessToken)
            }
            OutlinedTextField(
                value = freeResponse,
                onValueChange = onFreeResponseChange,
                placeholder = { Text(kind.placeholder ?: vibeFreeResponsePlaceholder()) },
                modifier = Modifier.fillMaxWidth(),
                minLines = 3,
            )
        }
        is VibeActivityBlockKind.Unsupported -> UnsupportedBlockCard(
            message = kind.message,
            courseCode = courseCode,
            itemId = itemId,
        )
        VibeActivityBlockKind.Divider -> HorizontalDivider()
    }
}

@Composable
private fun UnsupportedBlockCard(message: String, courseCode: String, itemId: String) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .background(LexturesColors.Amber.copy(alpha = 0.1f))
            .padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(text = vibeOpenOnWebLabel(), fontSize = 14.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
        Text(text = message, fontSize = 12.sp, color = textSecondary())
        OpenOnWebButton(courseCode = courseCode, itemId = itemId)
    }
}

@Composable
private fun OpenOnWebCard(courseCode: String, itemId: String, primary: Boolean) {
    if (primary) {
        LmsCard {
            Text(text = vibeWebOnlyHint(), fontSize = 13.sp, color = textSecondary())
            OpenOnWebButton(courseCode = courseCode, itemId = itemId)
        }
    }
}

@Composable
private fun OpenOnWebButton(courseCode: String, itemId: String) {
    val context = LocalContext.current
    AuthPrimaryButton(
        text = moduleOpenExternalLabel(),
        onClick = {
            val url = AppConfiguration.webUrl(VibeActivityLogic.webPath(courseCode, itemId))
            runCatching { context.startActivity(Intent(Intent.ACTION_VIEW, url.toUri())) }
        },
        modifier = Modifier.fillMaxWidth(),
    )
}