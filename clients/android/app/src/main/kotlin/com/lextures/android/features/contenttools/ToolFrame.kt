package com.lextures.android.features.contenttools

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolGovernanceLogic
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ToolScore

@Composable
fun ToolFrame(
    title: String,
    status: String,
    syncStatus: ContentToolHostLogic.SyncStatus,
    score: ToolScore?,
    readOnly: Boolean,
    readOnlyMessage: String?,
    studentResetAllowed: Boolean,
    onReset: (() -> Unit)?,
    showSandboxBadge: Boolean = false,
    showNonConformantNote: Boolean = false,
    canReport: Boolean = true,
    canModerate: Boolean = false,
    onReport: (() -> Unit)? = null,
    onModerate: (() -> Unit)? = null,
    frameModifier: Modifier = Modifier,
    disclosure: @Composable () -> Unit = {},
    content: @Composable () -> Unit,
) {
    var menuOpen by remember { mutableStateOf(false) }
    val resolvedStatus = toolStatusLabel(status)
    val resolvedSync = toolSyncLabel(syncStatus)
    val scoreLabel = L.text(R.string.mobile_contentTools_runtime_score)
    val sandboxBadge = L.text(R.string.mobile_contentTools_sandbox_badge)
    val nonConformantNote = L.text(R.string.mobile_contentTools_governance_nonConformant)
    val helpLabel = L.text(R.string.mobile_contentTools_runtime_help)
    val reportLabel = L.text(R.string.mobile_contentTools_governance_reportTitle)
    val moderateLabel = L.text(R.string.mobile_contentTools_governance_moderateTitle)
    val resetLabel = L.text(R.string.mobile_contentTools_runtime_reset)
    val a11y = buildString {
        append(ContentToolHostLogic.accessibleName(title, status))
        if (resolvedSync != null) append(", ").append(resolvedSync)
        if (score != null) append(", ").append(scoreLabel).append(" ${score.raw}/${score.max}")
        if (showSandboxBadge) append(", ").append(sandboxBadge)
        if (showNonConformantNote) append(", ").append(nonConformantNote)
    }
    val showReset = ContentToolGovernanceLogic.studentResetVisible(studentResetAllowed, readOnly) &&
        onReset != null
    Column(
        frameModifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .background(if (isDarkTheme()) sceneBackground() else LexturesColors.SceneBackground)
            .border(1.dp, fieldBorder(), RoundedCornerShape(12.dp))
            .padding(12.dp)
            .alpha(if (readOnly) 0.85f else 1f)
            .semantics(mergeDescendants = true) { contentDescription = a11y },
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(
                Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(4.dp),
            ) {
                Text(text = title, fontSize = 15.sp, color = textPrimary())
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    ToolChip(resolvedStatus)
                    resolvedSync?.let { ToolChip(it) }
                    score?.let { ToolChip("$scoreLabel ${it.raw}/${it.max}") }
                    if (showSandboxBadge) ToolChip(sandboxBadge)
                }
            }
            TextButton(
                onClick = { menuOpen = true },
                modifier = Modifier.semantics { contentDescription = helpLabel },
            ) {
                Text("⋯")
            }
            DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                DropdownMenuItem(
                    text = { Text(helpLabel) },
                    onClick = { menuOpen = false },
                    modifier = Modifier.semantics { contentDescription = helpLabel },
                )
                if (showReset) {
                    DropdownMenuItem(
                        text = { Text(resetLabel) },
                        onClick = {
                            menuOpen = false
                            onReset?.invoke()
                        },
                        modifier = Modifier.semantics { contentDescription = resetLabel },
                    )
                }
                if (canReport && onReport != null) {
                    DropdownMenuItem(
                        text = { Text(reportLabel) },
                        onClick = {
                            menuOpen = false
                            onReport()
                        },
                        modifier = Modifier.semantics { contentDescription = reportLabel },
                    )
                }
                if (canModerate && onModerate != null) {
                    DropdownMenuItem(
                        text = { Text(moderateLabel) },
                        onClick = {
                            menuOpen = false
                            onModerate()
                        },
                        modifier = Modifier.semantics { contentDescription = moderateLabel },
                    )
                }
            }
        }
        if (!readOnlyMessage.isNullOrBlank()) {
            Text(text = readOnlyMessage, fontSize = 12.sp, color = textSecondary())
        }
        if (showNonConformantNote) {
            Text(
                text = nonConformantNote,
                fontSize = 12.sp,
                color = textSecondary(),
                modifier = Modifier.semantics { contentDescription = nonConformantNote },
            )
        }
        // FR-6: disclosure is native frame chrome above tool content.
        disclosure()
        content()
    }
}

@Composable
private fun ToolChip(label: String) {
    Text(
        text = label,
        fontSize = 11.sp,
        color = textSecondary(),
        modifier = Modifier
            .clip(RoundedCornerShape(6.dp))
            .background(fieldBorder().copy(alpha = 0.25f))
            .padding(horizontal = 8.dp, vertical = 3.dp),
    )
}

@Composable
private fun toolStatusLabel(status: String): String = when (ContentToolHostLogic.statusChip(status)) {
    "completed" -> L.text(R.string.mobile_contentTools_runtime_status_completed)
    "submitted" -> L.text(R.string.mobile_contentTools_runtime_status_submitted)
    "in_progress" -> L.text(R.string.mobile_contentTools_runtime_status_in_progress)
    else -> L.text(R.string.mobile_contentTools_runtime_status_not_started)
}

@Composable
private fun toolSyncLabel(status: ContentToolHostLogic.SyncStatus): String? = when (status) {
    ContentToolHostLogic.SyncStatus.SAVING -> L.text(R.string.mobile_contentTools_runtime_saving)
    ContentToolHostLogic.SyncStatus.SAVED -> L.text(R.string.mobile_contentTools_runtime_saved)
    ContentToolHostLogic.SyncStatus.UNSYNCED -> L.text(R.string.mobile_contentTools_runtime_unsynced)
    ContentToolHostLogic.SyncStatus.ERROR -> L.text(R.string.mobile_contentTools_runtime_retry)
    ContentToolHostLogic.SyncStatus.IDLE -> null
}
