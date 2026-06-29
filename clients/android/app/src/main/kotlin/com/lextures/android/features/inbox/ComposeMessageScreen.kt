package com.lextures.android.features.inbox

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
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
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.accessibility.DictationField
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.SendMessageRequest
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch

@Composable
fun ComposeMessageScreen(
    session: AuthSession,
    initialTo: String,
    initialSubject: String,
    onDone: (Boolean) -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()

    var to by remember { mutableStateOf(initialTo) }
    var subject by remember { mutableStateOf(initialSubject) }
    var bodyText by remember { mutableStateOf("") }
    var sending by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    val canSend = to.isNotBlank() && (subject.isNotBlank() || bodyText.isNotBlank()) && !sending

    BackHandler { onDone(false) }

    fun send() {
        val token = accessToken ?: return
        sending = true
        errorMessage = null
        scope.launch {
            try {
                LmsApi.sendMessage(
                    SendMessageRequest(toEmail = to.trim(), subject = subject.trim(), body = bodyText),
                    token,
                )
                onDone(true)
            } catch (e: Exception) {
                errorMessage = session.mapError(e)
            } finally {
                sending = false
            }
        }
    }

    Column(modifier = modifier.padding(bottom = 16.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp, end = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = { onDone(false) }) {
                Icon(Icons.Default.Close, contentDescription = "Cancel", tint = textPrimary())
            }
            Text(
                text = "New message",
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                modifier = Modifier.weight(1f),
            )
            if (sending) {
                CircularProgressIndicator(
                    color = LexturesColors.Primary,
                    modifier = Modifier.height(22.dp),
                    strokeWidth = 2.dp,
                )
            } else {
                TextButton(onClick = { send() }, enabled = canSend) {
                    Text(
                        text = "Send",
                        fontWeight = FontWeight.SemiBold,
                        color = if (canSend) LexturesColors.Primary else textSecondary(),
                    )
                }
            }
        }

        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            errorMessage?.let { LmsErrorBanner(it) }

            OutlinedTextField(
                value = to,
                onValueChange = { to = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text("To") },
                placeholder = { Text("name@school.edu", color = textSecondary()) },
                singleLine = true,
                shape = RoundedCornerShape(10.dp),
                colors = OutlinedTextFieldDefaults.colors(focusedBorderColor = LexturesColors.Primary),
            )

            OutlinedTextField(
                value = subject,
                onValueChange = { subject = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text("Subject") },
                singleLine = true,
                shape = RoundedCornerShape(10.dp),
                colors = OutlinedTextFieldDefaults.colors(focusedBorderColor = LexturesColors.Primary),
            )

            DictationField(
                title = "Message",
                value = bodyText,
                onValueChange = { bodyText = it },
                modifier = Modifier.weight(1f),
            )
        }
    }
}
