package com.lextures.android.features.inbox

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Reply
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.filled.StarBorder
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.MailboxMessage
import com.lextures.android.core.lms.MailboxPatchRequest
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch

@Composable
fun MessageDetailScreen(
    session: AuthSession,
    message: MailboxMessage,
    onChanged: () -> Unit,
    onReply: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    var starred by remember { mutableStateOf(message.starred) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    BackHandler(onBack = onBack)

    Column(modifier = modifier) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp, end = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Box(modifier = Modifier.weight(1f))
            IconButton(onClick = {
                scope.launch {
                    val token = accessToken ?: return@launch
                    runCatching {
                        LmsApi.patchMailbox(message.id, MailboxPatchRequest(starred = !starred), token)
                    }.onSuccess {
                        starred = !starred
                        onChanged()
                    }.onFailure {
                        errorMessage = session.mapError(it)
                    }
                }
            }) {
                Icon(
                    if (starred) Icons.Default.Star else Icons.Default.StarBorder,
                    contentDescription = if (starred) "Unstar" else "Star",
                    tint = if (starred) Color(0xFFF59E0B) else textSecondary(),
                )
            }
            IconButton(onClick = onReply) {
                Icon(Icons.AutoMirrored.Filled.Reply, contentDescription = "Reply", tint = textSecondary())
            }
            IconButton(onClick = {
                scope.launch {
                    val token = accessToken ?: return@launch
                    runCatching {
                        LmsApi.patchMailbox(message.id, MailboxPatchRequest(folder = "trash"), token)
                    }.onSuccess {
                        onChanged()
                        onBack()
                    }.onFailure {
                        errorMessage = session.mapError(it)
                    }
                }
            }) {
                Icon(Icons.Default.Delete, contentDescription = "Trash", tint = textSecondary())
            }
        }

        Column(
            modifier = Modifier
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            errorMessage?.let { LmsErrorBanner(it) }

            Text(
                text = message.subject.ifBlank { "(no subject)" },
                fontSize = 19.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )

            LmsCard {
                Row(modifier = Modifier.fillMaxWidth()) {
                    Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Text(
                            text = message.from.name.ifBlank { message.from.email },
                            fontSize = 14.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        Text(text = message.from.email, fontSize = 12.sp, color = textSecondary())
                        if (message.to.isNotBlank()) {
                            Text(text = "To: ${message.to}", fontSize = 12.sp, color = textSecondary())
                        }
                    }
                    Text(
                        text = LmsDates.shortDateTime(message.sentAt),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
            }

            LmsCard {
                Text(
                    text = message.body.ifBlank { message.snippet },
                    fontSize = 15.sp,
                    color = textPrimary(),
                )
            }
        }
    }
}
