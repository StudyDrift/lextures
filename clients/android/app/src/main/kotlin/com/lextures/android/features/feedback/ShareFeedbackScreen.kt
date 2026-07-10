package com.lextures.android.features.feedback

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
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
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.FeedbackLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ShareFeedbackSheet(
    session: AuthSession,
    localePrefs: LocalePreferences,
    isOnline: Boolean,
    onDismiss: () -> Unit,
    onSuccess: () -> Unit,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val accessToken = session.accessToken.value
    val configuration = LocalConfiguration.current
    val viewport = "${configuration.screenWidthDp}x${configuration.screenHeightDp}"

    var message by remember { mutableStateOf("") }
    var category by remember { mutableStateOf("") }
    var categoryExpanded by remember { mutableStateOf(false) }
    var submitting by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    val canSend = FeedbackLogic.messageValid(message) && !submitting && isOnline

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(Modifier.padding(16.dp)) {
            Text(L.text(R.string.mobile_feedback_title))
            Text(
                L.text(R.string.mobile_feedback_privacy),
                modifier = Modifier.padding(top = 4.dp, bottom = 12.dp),
            )

            errorMessage?.let {
                LmsErrorBanner(message = it, modifier = Modifier.padding(bottom = 8.dp))
            } ?: if (!isOnline) {
                LmsErrorBanner(
                    message = L.text(R.string.mobile_feedback_offline),
                    modifier = Modifier.padding(bottom = 8.dp),
                )
            } else {
                Unit
            }

            OutlinedTextField(
                value = message,
                onValueChange = { if (it.length <= FeedbackLogic.MAX_MESSAGE_LEN) message = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(L.text(R.string.mobile_feedback_message_label)) },
                placeholder = { Text(L.text(R.string.mobile_feedback_message_placeholder)) },
                minLines = 4,
                enabled = !submitting,
            )
            Text(
                L.format(
                    context,
                    localePrefs,
                    R.string.mobile_feedback_message_counter,
                    FeedbackLogic.trimmedMessageLength(message),
                    FeedbackLogic.MAX_MESSAGE_LEN,
                ),
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 4.dp, bottom = 8.dp),
            )

            ExposedDropdownMenuBox(
                expanded = categoryExpanded,
                onExpandedChange = { categoryExpanded = !categoryExpanded },
                modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp),
            ) {
                OutlinedTextField(
                    value = L.text(context, localePrefs, FeedbackLogic.categoryLabelRes(category)),
                    onValueChange = {},
                    readOnly = true,
                    label = { Text(L.text(R.string.mobile_feedback_category_label)) },
                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = categoryExpanded) },
                    modifier = Modifier.menuAnchor().fillMaxWidth(),
                )
                ExposedDropdownMenu(
                    expanded = categoryExpanded,
                    onDismissRequest = { categoryExpanded = false },
                ) {
                    DropdownMenuItem(
                        text = { Text(L.text(context, localePrefs, R.string.mobile_feedback_category_none)) },
                        onClick = {
                            category = ""
                            categoryExpanded = false
                        },
                    )
                    FeedbackLogic.CATEGORIES.forEach { value ->
                        DropdownMenuItem(
                            text = { Text(L.text(context, localePrefs, FeedbackLogic.categoryLabelRes(value))) },
                            onClick = {
                                category = value
                                categoryExpanded = false
                            },
                        )
                    }
                }
            }

            Button(
                onClick = {
                    if (!isOnline) {
                        errorMessage = L.text(context, localePrefs, R.string.mobile_feedback_offline)
                        return@Button
                    }
                    val token = accessToken ?: return@Button
                    submitting = true
                    errorMessage = null
                    scope.launch {
                        try {
                            LmsApi.submitFeedback(
                                FeedbackLogic.buildSubmitRequest(
                                    message = message,
                                    category = category,
                                    route = "profile",
                                    locale = localePrefs.effectiveTag,
                                    viewport = viewport,
                                ),
                                token,
                            )
                            onSuccess()
                            onDismiss()
                        } catch (error: Exception) {
                            val outcome = FeedbackLogic.mapSubmitError(error, isOnline)
                            errorMessage = L.text(context, localePrefs, FeedbackLogic.errorMessageRes(outcome))
                        } finally {
                            submitting = false
                        }
                    }
                },
                enabled = canSend,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    if (submitting) {
                        L.text(context, localePrefs, R.string.mobile_feedback_sending)
                    } else {
                        L.text(R.string.mobile_feedback_send)
                    },
                )
            }

            TextButton(
                onClick = onDismiss,
                enabled = !submitting,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(L.text(R.string.mobile_feedback_cancel))
            }
        }
    }
}
