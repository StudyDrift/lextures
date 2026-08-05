package com.lextures.android.features.checklist

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.WifiOff
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.ChecklistApi
import com.lextures.android.core.lms.ChecklistDismissReason
import com.lextures.android.core.lms.ChecklistItem
import com.lextures.android.core.lms.ChecklistNavTarget
import com.lextures.android.core.lms.CourseChecklist
import com.lextures.android.core.lms.CourseChecklistLogic
import com.lextures.android.core.lms.CourseChecklistSummaryStore
import com.lextures.android.core.lms.CourseChecklistTargetTable
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.routing.LinkOpener
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch

@Composable
fun CourseChecklistSection(
    session: AuthSession,
    course: CourseSummary,
    shell: HomeShellState?,
    isOnline: Boolean,
    initialFocus: String? = null,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    val table = remember { CourseChecklistTargetTable.shared(context) }

    var checklist by remember { mutableStateOf<CourseChecklist?>(null) }
    var loading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var rateLimitMessage by remember { mutableStateOf<String?>(null) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var showCompleted by remember { mutableStateOf(false) }
    var expandedCategories by remember { mutableStateOf(setOf<String>()) }
    var expandedItems by remember { mutableStateOf(setOf<String>()) }
    var dismissTarget by remember { mutableStateOf<ChecklistItem?>(null) }

    fun errorText(e: Exception): String = when (e) {
        is ApiError.HttpStatus -> e.apiMessage ?: L.text(context, localePrefs, R.string.mobile_checklist_loadError)
        else -> e.message ?: L.text(context, localePrefs, R.string.mobile_checklist_loadError)
    }

    suspend fun loadFull(force: Boolean = false) {
        if (!isOnline) return
        val token = session.accessToken.value ?: return
        if (!force && checklist != null) return
        loading = true
        errorMessage = null
        try {
            val result = ChecklistApi.fetchChecklist(course.courseCode, token)
            checklist = result
            CourseChecklistSummaryStore.applyChecklist(result)
            expandedCategories = result.categories
                .filter { CourseChecklistLogic.outstandingCount(it) > 0 }
                .map { it.id }
                .toSet()
        } catch (e: ApiError.HttpStatus) {
            if (e.code == 403) {
                CourseChecklistSummaryStore.markForbidden(course.courseCode)
                checklist = null
                errorMessage = null
            } else {
                errorMessage = errorText(e)
            }
        } catch (e: Exception) {
            errorMessage = errorText(e)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(course.courseCode, isOnline) {
        loadFull()
    }

    fun openTarget(target: ChecklistNavTarget?) {
        val resolved = CourseChecklistLogic.resolveTarget(target, course.courseCode, table)
        when (resolved.kind) {
            CourseChecklistLogic.TargetKind.Native -> {
                resolved.workspaceSection?.let { shell?.activeCourseSection = it }
            }
            CourseChecklistLogic.TargetKind.Web -> {
                resolved.webPath?.let { path ->
                    val url = com.lextures.android.core.config.AppConfiguration.webUrl(path)
                    LinkOpener.open(context, url, shell, "checklist")
                }
            }
            CourseChecklistLogic.TargetKind.Unresolved -> {
                resolved.workspaceSection?.let { shell?.activeCourseSection = it }
                    ?: resolved.webPath?.let { path ->
                        val url = com.lextures.android.core.config.AppConfiguration.webUrl(path)
                        LinkOpener.open(context, url, shell, "checklist")
                    }
            }
        }
    }

    Column(modifier = modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_checklist_pageTitle),
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                modifier = Modifier.weight(1f),
            )
            IconButton(
                onClick = {
                    scope.launch {
                        val token = session.accessToken.value ?: return@launch
                        if (!isOnline) return@launch
                        loading = true
                        rateLimitMessage = null
                        try {
                            val result = ChecklistApi.refreshChecklist(course.courseCode, token)
                            checklist = result
                            CourseChecklistSummaryStore.applyChecklist(result)
                        } catch (e: ApiError.HttpStatus) {
                            if (e.code == 429) {
                                rateLimitMessage = CourseChecklistLogic.RATE_LIMITED_MESSAGE
                            } else {
                                errorMessage = errorText(e)
                            }
                        } catch (e: Exception) {
                            errorMessage = errorText(e)
                        } finally {
                            loading = false
                        }
                    }
                },
                enabled = isOnline && !loading,
            ) {
                Icon(Icons.Default.Refresh, contentDescription = L.text(context, localePrefs, R.string.mobile_checklist_recheck))
            }
        }

        val summary = checklist?.summary ?: CourseChecklistSummaryStore.cached(course.courseCode)
        if (summary != null) {
            val progressLabel = CourseChecklistLogic.progressLabel(summary.done, summary.total)
            Text(progressLabel, color = textSecondary(), fontSize = 14.sp)
            LinearProgressIndicator(
                progress = { CourseChecklistLogic.progressFraction(summary.done, summary.total).toFloat() },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(6.dp)
                    .clip(RoundedCornerShape(3.dp)),
                color = accentColor(),
            )
        }

        when {
            !isOnline -> {
                LmsEmptyState(
                    icon = Icons.Default.WifiOff,
                    title = L.text(context, localePrefs, R.string.mobile_checklist_offlineTitle),
                    message = L.text(context, localePrefs, R.string.mobile_checklist_offlineBody),
                )
            }
            errorMessage != null -> {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    LmsErrorBanner(errorMessage!!)
                    OutlinedButton(onClick = { scope.launch { loadFull(force = true) } }) {
                        Text(L.text(context, localePrefs, R.string.mobile_checklist_retry))
                    }
                }
            }
            loading && checklist == null -> {
                CircularProgressIndicator(modifier = Modifier.padding(24.dp))
            }
            checklist != null -> {
                val data = checklist!!
                val allDone = data.summary.outstandingTotal == 0 && data.summary.total > 0
                if (allDone && !showCompleted && data.dismissed.isEmpty()) {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(12.dp))
                            .padding(12.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_checklist_allDoneTitle),
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        Text(
                            L.text(context, localePrefs, R.string.mobile_checklist_allDoneBody),
                            color = textSecondary(),
                        )
                        OutlinedButton(onClick = { showCompleted = true }) {
                            Text(L.text(context, localePrefs, R.string.mobile_checklist_showCompleted))
                        }
                    }
                } else if (data.categories.isEmpty() && data.dismissed.isEmpty()) {
                    Text(
                        L.text(context, localePrefs, R.string.mobile_checklist_catalogEmpty),
                        color = textSecondary(),
                    )
                } else {
                    // Offer show/hide whenever any items are done (not only when everything is done).
                    if (data.summary.done > 0) {
                        TextButton(onClick = { showCompleted = !showCompleted }) {
                            Text(
                                L.text(
                                    context,
                                    localePrefs,
                                    if (showCompleted) R.string.mobile_checklist_hideCompleted
                                    else R.string.mobile_checklist_showCompleted,
                                ),
                            )
                        }
                    }
                    data.categories.forEach { category ->
                        val outstanding = CourseChecklistLogic.outstandingCount(category)
                        val expanded = category.id in expandedCategories || outstanding > 0
                        val items = CourseChecklistLogic.visibleItems(category, showCompleted)
                        if (items.isEmpty()) return@forEach
                        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .clickable {
                                        expandedCategories = if (category.id in expandedCategories) {
                                            expandedCategories - category.id
                                        } else {
                                            expandedCategories + category.id
                                        }
                                    }
                                    .padding(vertical = 6.dp),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Text(
                                    text = category.title,
                                    fontWeight = FontWeight.SemiBold,
                                    color = textPrimary(),
                                    modifier = Modifier.weight(1f),
                                )
                                if (outstanding > 0) {
                                    Text(
                                        text = context.getString(R.string.mobile_checklist_outstandingCount, outstanding),
                                        color = textSecondary(),
                                        fontSize = 12.sp,
                                    )
                                }
                            }
                            if (expanded) {
                                items.forEach { item ->
                                    ChecklistItemCard(
                                        item = item,
                                        expanded = item.id in expandedItems,
                                        isOnline = isOnline,
                                        onToggleEvidence = {
                                            expandedItems = if (item.id in expandedItems) {
                                                expandedItems - item.id
                                            } else {
                                                expandedItems + item.id
                                            }
                                        },
                                        onOpen = { openTarget(item.target) },
                                        onDismiss = { dismissTarget = item },
                                        onRecheck = {
                                            scope.launch {
                                                val token = session.accessToken.value ?: return@launch
                                                try {
                                                    ChecklistApi.recheckItem(course.courseCode, item.id, token)
                                                    loadFull(force = true)
                                                } catch (e: Exception) {
                                                    actionError = errorText(e)
                                                }
                                            }
                                        },
                                        onOpenEvidence = { openTarget(it) },
                                    )
                                }
                            }
                        }
                    }
                    if (data.dismissed.isNotEmpty()) {
                        Text(
                            text = context.getString(R.string.mobile_checklist_dismissedSection, data.dismissed.size),
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                            modifier = Modifier.padding(top = 8.dp),
                        )
                        data.dismissed.forEach { item ->
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(vertical = 6.dp),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Text(item.title, modifier = Modifier.weight(1f), color = textPrimary())
                                TextButton(
                                    onClick = {
                                        scope.launch {
                                            val token = session.accessToken.value ?: return@launch
                                            try {
                                                ChecklistApi.restoreItem(course.courseCode, item.id, token)
                                                loadFull(force = true)
                                            } catch (e: Exception) {
                                                actionError = errorText(e)
                                            }
                                        }
                                    },
                                    enabled = isOnline,
                                ) {
                                    Text(L.text(context, localePrefs, R.string.mobile_checklist_restore))
                                }
                            }
                        }
                    }
                }
            }
        }

        rateLimitMessage?.let {
            Text(it, color = textSecondary(), fontSize = 13.sp)
        }
        actionError?.let {
            Text(it, color = com.lextures.android.core.design.LexturesColors.Coral, fontSize = 13.sp)
        }
    }

    dismissTarget?.let { item ->
        ChecklistDismissDialog(
            item = item,
            isOnline = isOnline,
            onDismissRequest = { dismissTarget = null },
            onConfirm = { reason, note ->
                dismissTarget = null
                scope.launch {
                    val token = session.accessToken.value ?: return@launch
                    try {
                        ChecklistApi.dismissItem(course.courseCode, item.id, reason, note, token)
                        loadFull(force = true)
                    } catch (e: Exception) {
                        actionError = errorText(e)
                    }
                }
            },
        )
    }

    // silence unused
    @Suppress("UNUSED_VARIABLE")
    val focus = initialFocus
}

@Composable
private fun ChecklistItemCard(
    item: ChecklistItem,
    expanded: Boolean,
    isOnline: Boolean,
    onToggleEvidence: () -> Unit,
    onOpen: () -> Unit,
    onDismiss: () -> Unit,
    onRecheck: () -> Unit,
    onOpenEvidence: (ChecklistNavTarget?) -> Unit,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val done = CourseChecklistLogic.isDone(item.status)
    val evidenceCount = item.evidence?.rows?.size ?: 0
    var menuOpen by remember { mutableStateOf(false) }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .padding(10.dp)
            .semantics {
                contentDescription = "${item.title}, ${CourseChecklistLogic.accessibilityStatusValue(item.status)}"
            },
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Row(verticalAlignment = Alignment.Top) {
            if (done) {
                Icon(Icons.Default.CheckCircle, contentDescription = null, tint = accentColor(), modifier = Modifier.size(20.dp))
            }
            Column(
                modifier = Modifier
                    .weight(1f)
                    .clickable(enabled = !done || evidenceCount > 0) {
                        if (evidenceCount > 0) onToggleEvidence() else if (!done) onOpen()
                    }
                    .padding(start = 8.dp),
            ) {
                Text(
                    text = item.title,
                    fontWeight = FontWeight.Medium,
                    color = textPrimary(),
                    textDecoration = if (done) TextDecoration.LineThrough else null,
                )
                item.detail?.takeIf { it.isNotBlank() }?.let {
                    Text(it, color = textSecondary(), fontSize = 13.sp)
                }
            }
            IconButton(onClick = { menuOpen = true }) {
                Icon(Icons.Default.MoreVert, contentDescription = L.text(context, localePrefs, R.string.mobile_checklist_overflowMenu))
            }
            DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                if (!done) {
                    DropdownMenuItem(
                        text = { Text(L.text(context, localePrefs, R.string.mobile_checklist_open)) },
                        onClick = { menuOpen = false; onOpen() },
                    )
                    DropdownMenuItem(
                        text = { Text(L.text(context, localePrefs, R.string.mobile_checklist_dismiss)) },
                        onClick = { menuOpen = false; onDismiss() },
                        enabled = isOnline,
                    )
                }
                DropdownMenuItem(
                    text = { Text(L.text(context, localePrefs, R.string.mobile_checklist_recheckItem)) },
                    onClick = { menuOpen = false; onRecheck() },
                    enabled = isOnline,
                )
            }
        }
        if (evidenceCount > 0) {
            TextButton(onClick = onToggleEvidence) {
                Text(
                    if (expanded) {
                        L.text(context, localePrefs, R.string.mobile_checklist_hideEvidence)
                    } else {
                        context.getString(R.string.mobile_checklist_showEvidence, evidenceCount)
                    },
                )
            }
        }
        if (expanded) {
            item.evidence?.rows?.forEach { row ->
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable { onOpenEvidence(row.target) }
                        .padding(vertical = 8.dp, horizontal = 8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(row.label, color = textPrimary())
                        row.sublabel?.takeIf { it.isNotBlank() }?.let {
                            Text(it, color = textSecondary(), fontSize = 12.sp)
                        }
                    }
                    Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null, tint = textSecondary())
                }
            }
        }
    }
}
