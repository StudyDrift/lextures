package com.lextures.android.features.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material.icons.filled.Apps
import androidx.compose.material.icons.filled.Dns
import androidx.compose.material.icons.filled.Email
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Storage
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import com.lextures.android.core.design.OutboxStatusChip
import com.lextures.android.core.offline.OutboxStatus
import com.lextures.android.core.offline.OfflineService
import kotlinx.coroutines.launch
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.BuildConfig
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.HeroBrush
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard

/** Profile tab: identity hero, notifications, app info, and sign-out. */
@Composable
fun ProfileTab(
    session: AuthSession,
    shell: HomeShellState,
    modifier: Modifier = Modifier,
) {
    var confirmingSignOut by remember { mutableStateOf(false) }
    var confirmingClearCache by remember { mutableStateOf(false) }
    var showNotifications by remember { mutableStateOf(false) }
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val pendingCount by offline.pendingCount.collectAsState()
    val storageBytes by offline.storageBytes.collectAsState()
    val outboxItems by offline.outboxItems.collectAsState()
    val scope = rememberCoroutineScope()

    if (showNotifications) {
        NotificationsScreen(
            session = session,
            shell = shell,
            onBack = { showNotifications = false },
            modifier = modifier,
        )
        return
    }

    val displayName = shell.profile?.displayName?.trim().orEmpty()
        .ifEmpty { shell.profile?.firstName ?: "Welcome" }
    val email = shell.profile?.email ?: ""

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        // Identity hero
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(24.dp))
                .background(HeroBrush),
        ) {
            Box(
                modifier = Modifier
                    .size(150.dp)
                    .offset(x = 250.dp, y = (-56).dp)
                    .clip(CircleShape)
                    .background(Color.White.copy(alpha = 0.07f)),
            )
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 26.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Box(
                    modifier = Modifier
                        .size(76.dp)
                        .clip(CircleShape)
                        .background(Color.White.copy(alpha = 0.16f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = shell.profile?.initials ?: "··",
                        style = LexturesType.display(28, FontWeight.Bold),
                        color = Color.White,
                    )
                }
                Text(text = displayName, style = LexturesType.display(22), color = Color.White)
                if (email.isNotEmpty()) {
                    Text(text = email, fontSize = 13.sp, color = Color.White.copy(alpha = 0.8f))
                }
            }
        }

        if (pendingCount > 0) {
            LmsCard {
                Text(text = "Pending sync", style = LexturesType.display(17), color = textPrimary())
                Text(
                    text = "$pendingCount change${if (pendingCount == 1) "" else "s"} waiting to upload",
                    fontSize = 12.sp,
                    color = textSecondary(),
                )
                outboxItems.filter {
                    val status = it.outboxStatus()
                    status == OutboxStatus.Queued || status == OutboxStatus.Failed || status == OutboxStatus.Conflict
                }.forEach { item ->
                    Column(modifier = Modifier.padding(top = 8.dp)) {
                        Text(text = item.label, fontSize = 14.sp, fontWeight = FontWeight.Medium, color = textPrimary())
                        OutboxStatusChip(status = item.outboxStatus())
                        if (item.outboxStatus() == OutboxStatus.Failed || item.outboxStatus() == OutboxStatus.Conflict) {
                            TextButton(onClick = {
                                scope.launch {
                                    offline.retryOutboxItem(item.id, accessToken)
                                }
                            }) {
                                Text("Retry")
                            }
                        }
                    }
                }
            }
        }

        LmsCard {
            Text(text = "Offline storage", style = LexturesType.display(17), color = textPrimary())
            InfoRow(
                Icons.Default.Storage,
                "Cache size",
                android.text.format.Formatter.formatFileSize(context, storageBytes),
            )
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(10.dp))
                    .clickable { confirmingClearCache = true }
                    .padding(vertical = 8.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(Icons.Default.Delete, contentDescription = null, tint = LexturesColors.Error)
                Text(
                    text = "Clear cached data",
                    fontSize = 14.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = LexturesColors.Error,
                )
            }
        }

        // Account
        LmsCard {
            Text(text = "Account", style = LexturesType.display(17), color = textPrimary())
            InfoRow(Icons.Default.Person, "Display name", displayName)
            InfoRow(Icons.Default.Email, "Email", email.ifEmpty { "—" })
        }

        // Notifications
        LmsCard(onClick = { showNotifications = true }) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(
                    modifier = Modifier
                        .size(32.dp)
                        .clip(RoundedCornerShape(10.dp))
                        .background(LexturesColors.BrandTeal.copy(alpha = if (isDarkTheme()) 0.18f else 0.14f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        Icons.Default.Notifications,
                        contentDescription = null,
                        tint = accentColor(),
                        modifier = Modifier.size(16.dp),
                    )
                }
                Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Text(
                        text = "Notifications",
                        fontSize = 15.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    Text(
                        text = if (shell.unreadNotifications > 0) {
                            "${shell.unreadNotifications} unread"
                        } else {
                            "You're all caught up"
                        },
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
                if (shell.unreadNotifications > 0) {
                    Text(
                        text = "${shell.unreadNotifications}",
                        fontSize = 12.sp,
                        fontWeight = FontWeight.Bold,
                        color = Color.White,
                        modifier = Modifier
                            .clip(RoundedCornerShape(50))
                            .background(LexturesColors.Coral)
                            .padding(horizontal = 8.dp, vertical = 3.dp),
                    )
                }
                Icon(
                    Icons.AutoMirrored.Filled.KeyboardArrowRight,
                    contentDescription = null,
                    tint = textSecondary().copy(alpha = 0.6f),
                    modifier = Modifier.size(16.dp),
                )
            }
        }

        // About
        LmsCard {
            Text(text = "About", style = LexturesType.display(17), color = textPrimary())
            InfoRow(Icons.Default.Apps, "Version", BuildConfig.VERSION_NAME)
            InfoRow(Icons.Default.Dns, "Server", AppConfiguration.apiBaseUrl)
        }

        // Sign out
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(14.dp))
                .background(LexturesColors.Error.copy(alpha = 0.09f))
                .clickable { confirmingSignOut = true }
                .padding(vertical = 14.dp),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                Icons.AutoMirrored.Filled.Logout,
                contentDescription = null,
                tint = LexturesColors.Error,
                modifier = Modifier.size(17.dp),
            )
            Box(modifier = Modifier.width(8.dp))
            Text(
                text = "Sign out",
                fontSize = 15.sp,
                fontWeight = FontWeight.SemiBold,
                color = LexturesColors.Error,
            )
        }
    }

    if (confirmingClearCache) {
        AlertDialog(
            onDismissRequest = { confirmingClearCache = false },
            title = { Text("Clear offline storage?") },
            text = {
                Text("Removes cached reads and downloads from this device. Queued changes are kept until they sync.")
            },
            confirmButton = {
                TextButton(onClick = {
                    confirmingClearCache = false
                    offline.clearStorage()
                }) {
                    Text("Clear cache", color = LexturesColors.Error)
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmingClearCache = false }) { Text("Cancel") }
            },
        )
    }

    if (confirmingSignOut) {
        AlertDialog(
            onDismissRequest = { confirmingSignOut = false },
            title = { Text("Sign out of Lextures?") },
            confirmButton = {
                TextButton(onClick = {
                    confirmingSignOut = false
                    session.signOut()
                }) {
                    Text("Sign out", color = LexturesColors.Error)
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmingSignOut = false }) { Text("Cancel") }
            },
        )
    }
}

@Composable
private fun InfoRow(icon: ImageVector, label: String, value: String) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, contentDescription = null, tint = accentColor(), modifier = Modifier.size(17.dp))
        Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(text = label, fontSize = 11.sp, color = textSecondary())
            Text(
                text = value,
                fontSize = 14.sp,
                fontWeight = FontWeight.Medium,
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}
