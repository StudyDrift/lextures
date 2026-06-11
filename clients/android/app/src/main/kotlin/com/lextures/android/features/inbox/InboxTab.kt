package com.lextures.android.features.inbox

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AttachFile
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Inbox
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Star
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.coverBrush
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.MailboxFolder
import com.lextures.android.core.lms.MailboxMessage
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsChipRow
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/** Mailbox folders, search, message list, detail, and compose (parity with web inbox). */
@Composable
fun InboxTab(
    session: AuthSession,
    onUnreadChanged: (Int) -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()

    var folder by remember { mutableStateOf(MailboxFolder.Inbox) }
    var messages by remember { mutableStateOf<List<MailboxMessage>>(emptyList()) }
    var searchText by remember { mutableStateOf("") }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var openMessage by remember { mutableStateOf<MailboxMessage?>(null) }
    var composeSeed by remember { mutableStateOf<Pair<String, String>?>(null) } // to / subject
    var revision by remember { mutableIntStateOf(0) }

    suspend fun refreshUnread() {
        val token = accessToken ?: return
        runCatching { LmsApi.fetchUnreadInboxCount(token) }.onSuccess(onUnreadChanged)
    }

    LaunchedEffect(accessToken, folder, searchText, revision) {
        val token = accessToken ?: return@LaunchedEffect
        if (searchText.isNotEmpty()) delay(300)
        loading = true
        errorMessage = null
        try {
            messages = LmsApi.fetchMailboxMessages(folder, searchText, token)
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
        refreshUnread()
    }

    composeSeed?.let { (to, subject) ->
        ComposeMessageScreen(
            session = session,
            initialTo = to,
            initialSubject = subject,
            onDone = { sent ->
                composeSeed = null
                if (sent) revision++
            },
            modifier = modifier,
        )
        return
    }

    openMessage?.let { message ->
        MessageDetailScreen(
            session = session,
            message = message,
            onChanged = { revision++ },
            onReply = {
                openMessage = null
                val subject = if (message.subject.startsWith("Re:")) message.subject else "Re: ${message.subject}"
                composeSeed = message.from.email to subject
            },
            onBack = { openMessage = null },
            modifier = modifier,
        )
        return
    }

    Scaffold(
        modifier = modifier,
        containerColor = Color.Transparent,
        floatingActionButton = {
            FloatingActionButton(
                onClick = { composeSeed = "" to "" },
                containerColor = LexturesColors.Primary,
                contentColor = Color.White,
            ) {
                Icon(Icons.Default.Edit, contentDescription = "Compose")
            }
        },
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            Text(
                text = "Inbox",
                fontSize = 22.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                modifier = Modifier.padding(start = 16.dp, top = 12.dp),
            )

            OutlinedTextField(
                value = searchText,
                onValueChange = { searchText = it },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                placeholder = { Text("Search mail", color = textSecondary()) },
                leadingIcon = { Icon(Icons.Default.Search, contentDescription = null, tint = textSecondary()) },
                singleLine = true,
                shape = RoundedCornerShape(10.dp),
                colors = OutlinedTextFieldDefaults.colors(
                    focusedBorderColor = LexturesColors.Primary,
                ),
            )

            LmsChipRow(
                options = MailboxFolder.entries.map { it.name to it.label },
                selectedId = folder.name,
                onSelect = { id -> folder = MailboxFolder.valueOf(id) },
            )

            errorMessage?.let {
                LmsErrorBanner(it, Modifier.padding(horizontal = 16.dp))
            }

            when {
                loading && messages.isEmpty() -> Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator(color = LexturesColors.Primary)
                }

                messages.isEmpty() -> LmsEmptyState(
                    icon = Icons.Default.Inbox,
                    title = if (searchText.isBlank()) {
                        "Nothing in ${folder.label.lowercase()}"
                    } else {
                        "No matching messages"
                    },
                    message = if (searchText.isBlank()) {
                        "Messages will appear here."
                    } else {
                        "Try different keywords, or clear search."
                    },
                )

                else -> LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 4.dp, bottom = 88.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    items(messages, key = { it.id }) { message ->
                        MessageRowCard(
                            message = message,
                            onClick = {
                                openMessage = message
                                if (!message.read) {
                                    scope.launch {
                                        val token = accessToken ?: return@launch
                                        runCatching {
                                            LmsApi.patchMailbox(
                                                message.id,
                                                com.lextures.android.core.lms.MailboxPatchRequest(read = true),
                                                token,
                                            )
                                        }
                                        refreshUnread()
                                    }
                                }
                            },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun MessageRowCard(message: MailboxMessage, onClick: () -> Unit) {
    LmsCard(onClick = onClick) {
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            Box(
                modifier = Modifier
                    .size(40.dp)
                    .background(coverBrush(message.from.email), CircleShape),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = initials(message.from.name.ifBlank { message.from.email }),
                    fontSize = 12.sp,
                    fontWeight = FontWeight.Bold,
                    color = Color.White,
                )
            }
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = message.from.name.ifBlank { message.from.email },
                        fontSize = 14.sp,
                        fontWeight = if (message.read) FontWeight.Normal else FontWeight.SemiBold,
                        color = textPrimary(),
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f),
                    )
                    Text(
                        text = LmsDates.relative(message.sentAt),
                        fontSize = 11.sp,
                        color = textSecondary(),
                    )
                }
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                    if (message.starred) {
                        Icon(
                            Icons.Default.Star,
                            contentDescription = null,
                            tint = Color(0xFFF59E0B),
                            modifier = Modifier.size(13.dp),
                        )
                    }
                    if (message.hasAttachment) {
                        Icon(
                            Icons.Default.AttachFile,
                            contentDescription = null,
                            tint = textSecondary(),
                            modifier = Modifier.size(13.dp),
                        )
                    }
                    Text(
                        text = message.subject.ifBlank { "(no subject)" },
                        fontSize = 14.sp,
                        fontWeight = if (message.read) FontWeight.Normal else FontWeight.SemiBold,
                        color = textPrimary(),
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                Text(
                    text = message.snippet,
                    fontSize = 12.sp,
                    color = textSecondary(),
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

private fun initials(name: String): String {
    val parts = name.trim().split(Regex("\\s+"))
    return if (parts.size >= 2) {
        "${parts.first().first()}${parts.last().first()}".uppercase()
    } else {
        name.take(2).uppercase()
    }
}
