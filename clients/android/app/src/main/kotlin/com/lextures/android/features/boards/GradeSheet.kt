package com.lextures.android.features.boards

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.BoardEngagementApi
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.core.network.ApiError
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun GradeSheet(
    courseCode: String,
    boardId: String,
    post: BoardPost,
    accessToken: String?,
    assignmentLinked: Boolean,
    onPostUpdate: (BoardPost) -> Unit,
    onAnnounce: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val scope = rememberCoroutineScope()
    var gradeText by remember {
        mutableStateOf(
            (BoardsLogic.visibleGrade(post) ?: post.myReaction?.value)
                ?.let { BoardsLogic.formatGrade(it) }
                .orEmpty(),
        )
    }
    var busy by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    val forbiddenMsg = L.text(R.string.mobile_boards_react_forbidden)
    val errorMsg = L.text(R.string.mobile_boards_react_error)
    val gradeSetTemplate = L.text(R.string.mobile_boards_react_gradeSet)
    val syncedTemplate = L.text(R.string.mobile_boards_react_gradeSynced)

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
        ) {
            Text(
                L.text(R.string.mobile_boards_grade_title),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                modifier = Modifier.padding(bottom = 12.dp),
            )
            OutlinedTextField(
                value = gradeText,
                onValueChange = { gradeText = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(L.text(R.string.mobile_boards_react_gradeInput)) },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                singleLine = true,
            )
            errorMessage?.let {
                Text(it, color = androidx.compose.ui.graphics.Color.Red, modifier = Modifier.padding(top = 8.dp))
            }
            TextButton(
                onClick = {
                    scope.launch {
                        val token = accessToken ?: return@launch
                        val value = gradeText.trim().toDoubleOrNull()
                        if (value == null || !value.isFinite() || busy) {
                            errorMessage = errorMsg
                            return@launch
                        }
                        busy = true
                        errorMessage = null
                        try {
                            val result = BoardEngagementApi.putReaction(
                                courseCode, boardId, post.id,
                                kind = "grade", value = value, accessToken = token,
                            )
                            onPostUpdate(BoardsLogic.applyReactionResult(post, result))
                            onAnnounce(
                                gradeSetTemplate
                                    .replace("%1\$s", BoardsLogic.formatGrade(value))
                                    .replace("%@", BoardsLogic.formatGrade(value)),
                            )
                            onDismiss()
                        } catch (e: ApiError.HttpStatus) {
                            errorMessage = if (e.code == 403) forbiddenMsg else errorMsg
                        } catch (_: Exception) {
                            errorMessage = errorMsg
                        } finally {
                            busy = false
                        }
                    }
                },
                enabled = !busy && gradeText.isNotBlank(),
                modifier = Modifier.padding(top = 8.dp),
            ) {
                Text(L.text(R.string.mobile_boards_grade_save))
            }
            if (assignmentLinked) {
                TextButton(
                    onClick = {
                        scope.launch {
                            val token = accessToken ?: return@launch
                            if (busy || BoardsLogic.visibleGrade(post) == null) return@launch
                            busy = true
                            errorMessage = null
                            try {
                                val result = BoardEngagementApi.syncGrade(
                                    courseCode, boardId, post.id, token,
                                )
                                onAnnounce(
                                    syncedTemplate
                                        .replace("%1\$s", BoardsLogic.formatGrade(result.pointsEarned))
                                        .replace("%@", BoardsLogic.formatGrade(result.pointsEarned)),
                                )
                                onDismiss()
                            } catch (_: Exception) {
                                errorMessage = errorMsg
                            } finally {
                                busy = false
                            }
                        }
                    },
                    enabled = !busy && BoardsLogic.visibleGrade(post) != null,
                ) {
                    Text(L.text(R.string.mobile_boards_react_sendGradebook))
                }
            }
        }
    }
}
