package com.lextures.android.features.home

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.accessibility.DictationField
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.AnnouncementLogic
import com.lextures.android.core.lms.Broadcast
import com.lextures.android.core.lms.BroadcastComposeType
import com.lextures.android.core.lms.LmsApi
import kotlinx.coroutines.launch

@Composable
fun BroadcastComposerScreen(
    session: AuthSession,
    orgId: String,
    onDone: (Broadcast?) -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()

    var type by remember { mutableStateOf(BroadcastComposeType.Announcement) }
    var subject by remember { mutableStateOf("") }
    var bodyText by remember { mutableStateOf("") }
    var sending by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var showConfirm by remember { mutableStateOf(false) }

    val canSend = AnnouncementLogic.canSubmitBroadcast(subject, bodyText) && !sending

    BackHandler { onDone(null) }

    if (showConfirm) {
        val reach = L.text(R.string.mobile_broadcast_reach_org)
        AlertDialog(
            onDismissRequest = { showConfirm = false },
            title = {
                Text(
                    if (type == BroadcastComposeType.Emergency) {
                        L.text(R.string.mobile_broadcast_compose_emergency_confirm_title)
                    } else {
                        L.text(R.string.mobile_broadcast_compose_confirm_title)
                    },
                )
            },
            text = {
                Text(
                    if (type == BroadcastComposeType.Emergency) {
                        L.format(R.string.mobile_broadcast_compose_emergency_confirm_message, reach)
                    } else {
                        L.format(R.string.mobile_broadcast_compose_confirm_message, reach)
                    },
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        showConfirm = false
                        val token = accessToken ?: return@TextButton
                        sending = true
                        errorMessage = null
                        scope.launch {
                            try {
                                val created = LmsApi.createBroadcast(
                                    orgId = orgId,
                                    type = type.wire,
                                    subject = subject.trim(),
                                    body = bodyText.trim(),
                                    accessToken = token,
                                )
                                onDone(created)
                            } catch (e: Exception) {
                                errorMessage = session.mapError(e)
                            } finally {
                                sending = false
                            }
                        }
                    },
                ) {
                    Text(
                        if (type == BroadcastComposeType.Emergency) {
                            L.text(R.string.mobile_broadcast_compose_send_emergency)
                        } else {
                            L.text(R.string.mobile_broadcast_compose_send)
                        },
                    )
                }
            },
            dismissButton = {
                TextButton(onClick = { showConfirm = false }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }

    Column(modifier = modifier.padding(bottom = 16.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp, end = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = { onDone(null) }) {
                Icon(Icons.Default.Close, contentDescription = L.text(R.string.mobile_common_cancel), tint = textPrimary())
            }
            Text(
                text = L.text(R.string.mobile_broadcast_compose_nav_title),
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                modifier = Modifier.weight(1f),
            )
            if (sending) {
                CircularProgressIndicator(color = LexturesColors.Primary, strokeWidth = 2.dp)
            } else {
                TextButton(onClick = { showConfirm = true }, enabled = canSend) {
                    Text(
                        text = L.text(R.string.mobile_broadcast_compose_review),
                        fontWeight = FontWeight.SemiBold,
                        color = if (canSend) LexturesColors.Primary else textSecondary(),
                    )
                }
            }
        }

        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            errorMessage?.let { LmsErrorBanner(message = it) }

            if (type == BroadcastComposeType.Emergency) {
                LmsCard(accent = LexturesColors.Coral) {
                    Text(
                        text = L.text(R.string.mobile_broadcast_compose_emergency_warning),
                        color = LexturesColors.Coral,
                        fontSize = 13.sp,
                        fontWeight = FontWeight.SemiBold,
                    )
                }
            }

            Text(
                text = L.text(R.string.mobile_broadcast_compose_type),
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = textSecondary(),
            )
            LmsSegmentedChips(
                options = listOf(
                    L.text(R.string.mobile_broadcast_compose_type_announcement),
                    L.text(R.string.mobile_broadcast_compose_type_emergency),
                ),
                selectedIndex = if (type == BroadcastComposeType.Announcement) 0 else 1,
                onSelect = { index ->
                    type = if (index == 0) BroadcastComposeType.Announcement else BroadcastComposeType.Emergency
                },
            )

            OutlinedTextField(
                value = subject,
                onValueChange = { subject = it },
                label = { Text(L.text(R.string.mobile_broadcast_compose_subject)) },
                modifier = Modifier.fillMaxWidth(),
                colors = OutlinedTextFieldDefaults.colors(
                    focusedTextColor = textPrimary(),
                    unfocusedTextColor = textPrimary(),
                ),
            )

            DictationField(
                title = L.text(R.string.mobile_broadcast_compose_body),
                text = bodyText,
                onTextChange = { bodyText = it },
                placeholder = L.text(R.string.mobile_broadcast_compose_body_placeholder),
            )
        }
    }
}