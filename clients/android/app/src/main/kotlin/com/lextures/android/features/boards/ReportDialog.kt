package com.lextures.android.features.boards

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.MenuAnchorType
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.BoardModerationApi
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.core.network.ApiError
import kotlinx.coroutines.launch

private enum class ReportReasonKey { Hurtful, Inappropriate, Spam, Other }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReportDialog(
    courseCode: String,
    boardId: String,
    accessToken: String?,
    postId: String? = null,
    commentId: String? = null,
    onDismiss: () -> Unit,
    onSubmitted: () -> Unit = {},
) {
    val scope = rememberCoroutineScope()
    var reasonKey by remember { mutableStateOf(ReportReasonKey.Hurtful) }
    var details by remember { mutableStateOf("") }
    var submitting by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var alreadyReported by remember { mutableStateOf(false) }
    var reasonMenuOpen by remember { mutableStateOf(false) }

    val reasonLabels = mapOf(
        ReportReasonKey.Hurtful to L.text(R.string.mobile_boards_report_reason_hurtful),
        ReportReasonKey.Inappropriate to L.text(R.string.mobile_boards_report_reason_inappropriate),
        ReportReasonKey.Spam to L.text(R.string.mobile_boards_report_reason_spam),
        ReportReasonKey.Other to L.text(R.string.mobile_boards_report_reason_other),
    )
    val rateLimited = L.text(R.string.mobile_boards_report_rateLimited)
    val reportError = L.text(R.string.mobile_boards_report_error)

    LaunchedEffect(postId, commentId) {
        alreadyReported = BoardsLogic.hasReported(postId, commentId)
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(L.text(R.string.mobile_boards_report_title)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text(L.text(R.string.mobile_boards_report_subtitle), color = textSecondary())
                if (alreadyReported) {
                    Text(L.text(R.string.mobile_boards_report_alreadyReported), color = textSecondary())
                }
                errorMessage?.let { Text(it, color = androidx.compose.ui.graphics.Color.Red) }
                ExposedDropdownMenuBox(
                    expanded = reasonMenuOpen && !alreadyReported,
                    onExpandedChange = { if (!alreadyReported) reasonMenuOpen = it },
                ) {
                    OutlinedTextField(
                        value = reasonLabels[reasonKey].orEmpty(),
                        onValueChange = {},
                        readOnly = true,
                        enabled = !alreadyReported,
                        label = { Text(L.text(R.string.mobile_boards_report_reasonLabel)) },
                        trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = reasonMenuOpen) },
                        modifier = Modifier
                            .menuAnchor(MenuAnchorType.PrimaryNotEditable)
                            .fillMaxWidth(),
                    )
                    ExposedDropdownMenu(
                        expanded = reasonMenuOpen && !alreadyReported,
                        onDismissRequest = { reasonMenuOpen = false },
                    ) {
                        ReportReasonKey.entries.forEach { key ->
                            DropdownMenuItem(
                                text = { Text(reasonLabels[key].orEmpty()) },
                                onClick = {
                                    reasonKey = key
                                    reasonMenuOpen = false
                                },
                            )
                        }
                    }
                }
                OutlinedTextField(
                    value = details,
                    onValueChange = { details = it },
                    enabled = !alreadyReported,
                    label = { Text(L.text(R.string.mobile_boards_report_detailsLabel)) },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 3,
                )
            }
        },
        confirmButton = {
            TextButton(
                enabled = !submitting && !alreadyReported && accessToken != null,
                onClick = {
                    val token = accessToken ?: return@TextButton
                    if (BoardsLogic.hasReported(postId, commentId)) {
                        alreadyReported = true
                        return@TextButton
                    }
                    scope.launch {
                        submitting = true
                        errorMessage = null
                        val reasonLabel = reasonLabels[reasonKey].orEmpty()
                        val detail = details.trim()
                        val reason = if (detail.isEmpty()) reasonLabel else "$reasonLabel — $detail"
                        try {
                            BoardModerationApi.reportContent(
                                courseCode = courseCode,
                                boardId = boardId,
                                postId = postId,
                                commentId = commentId,
                                reason = reason,
                                accessToken = token,
                            )
                            BoardsLogic.markReported(postId, commentId)
                            onSubmitted()
                            onDismiss()
                        } catch (e: ApiError.HttpStatus) {
                            errorMessage = if (e.code == 429) rateLimited else (e.message ?: reportError)
                        } catch (_: Exception) {
                            errorMessage = reportError
                        } finally {
                            submitting = false
                        }
                    }
                },
            ) {
                Text(L.text(R.string.mobile_boards_report_submit))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(L.text(R.string.mobile_common_cancel))
            }
        },
        modifier = Modifier.padding(8.dp),
    )
}
