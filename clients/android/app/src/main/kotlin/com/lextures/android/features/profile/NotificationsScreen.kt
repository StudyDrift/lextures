package com.lextures.android.features.profile

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Campaign
import androidx.compose.material.icons.filled.Email
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material.icons.filled.Verified
import androidx.compose.material.icons.filled.Forum
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.AppNotification
import com.lextures.android.core.lms.Broadcast
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.NotificationCategory
import com.lextures.android.core.lms.NotificationFilter
import com.lextures.android.core.lms.NotificationLogic
import com.lextures.android.core.lms.NotificationsPage
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.core.routing.DeepLinkRouter
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSegmentedChips
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.serializer

/** In-app notification inbox: category filters, mark-read on tap, mark-all-read, preferences. */
@Composable
fun NotificationsScreen(
    session: AuthSession,
    shell: HomeShellState,
    onBack: () -> Unit,
    onOpenPreferences: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }

    var filter by remember { mutableStateOf(NotificationFilter.All.id) }
    var notifications by remember { mutableStateOf<List<AppNotification>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var staleLabel by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    BackHandler(onBack = onBack)

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            val (page, cached) = offline.cachedFetch(
                key = OfflineCacheKey.notificationsPage(),
                accessToken = token,
                serializer = serializer<NotificationsPage>(),
            ) {
                LmsApi.fetchNotifications(token)
            }
            notifications = page.notifications
            shell.unreadNotifications = page.unreadCount
            staleLabel = if (cached?.isStale(offline.networkMonitor.isOnline.value) == true) {
                cached.lastUpdatedLabel()
            } else {
                null
            }
        } catch (e: Exception) {
            if (notifications.isEmpty()) {
                errorMessage = session.mapError(e)
            } else {
                staleLabel = staleLabel ?: L.text("mobile.notifications.stale.offline")
            }
        } finally {
            loading = false
        }
    }

    val activeFilter = NotificationFilter.entries.firstOrNull { it.id == filter } ?: NotificationFilter.All
    val visible = NotificationLogic.filter(notifications, activeFilter)

    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp, end = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = L.text("mobile.notifications.title"),
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                modifier = Modifier.weight(1f),
            )
            IconButton(onClick = onOpenPreferences) {
                Icon(
                    Icons.Default.Settings,
                    contentDescription = L.text("mobile.notifications.preferences.title"),
                    tint = textPrimary(),
                )
            }
            if (notifications.any { !it.isRead }) {
                Text(
                    text = L.text("mobile.notifications.markAllRead"),
                    fontSize = 13.sp,
                    fontWeight = FontWeight.Medium,
                    color = accentColor(),
                    modifier = Modifier
                        .clip(RoundedCornerShape(50))
                        .clickable {
                            val token = accessToken ?: return@clickable
                            scope.launch {
                                runCatching {
                                    offline.enqueueMutation(
                                        method = "POST",
                                        path = "/api/v1/me/notifications/read-all",
                                        bodyJson = "{}",
                                        label = L.text("mobile.notifications.markAllReadLabel"),
                                        accessToken = token,
                                    )
                                }.onSuccess {
                                    notifications = notifications.map { it.copy(isRead = true) }
                                    shell.unreadNotifications = 0
                                }.onFailure {
                                    errorMessage = session.mapError(it)
                                }
                            }
                        }
                        .padding(horizontal = 8.dp, vertical = 6.dp),
                )
            }
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                LmsSegmentedChips(
                    options = NotificationFilter.entries.map { it.id to L.text(it.labelKey) },
                    selectedId = filter,
                    onSelect = { filter = it },
                )
            }

            staleLabel?.let { label ->
                item {
                    Text(text = label, fontSize = 12.sp, color = textSecondary())
                }
            }

            errorMessage?.let { message ->
                item { LmsErrorBanner(message) }
            }

            if (loading && notifications.isEmpty()) {
                item { LmsSkeletonList(count = 5) }
            } else if (visible.isEmpty()) {
                item {
                    LmsEmptyState(
                        icon = Icons.Default.Notifications,
                        title = when (activeFilter) {
                            NotificationFilter.Unread -> L.text("mobile.notifications.empty.unread")
                            else -> L.text("mobile.notifications.empty.all")
                        },
                        message = L.text("mobile.notifications.empty.message"),
                    )
                }
            } else {
                items(visible, key = { it.id }) { notification ->
                    val readLabel = if (notification.isRead) {
                        L.text("mobile.notifications.accessibility.read")
                    } else {
                        L.text("mobile.notifications.accessibility.unread")
                    }
                    LmsCard(
                        accent = if (notification.isRead) null else LexturesColors.BrandTeal,
                        onClick = {
                            val token = accessToken ?: return@LmsCard
                            if (!notification.isRead) {
                                notifications = notifications.map {
                                    if (it.id == notification.id) it.copy(isRead = true) else it
                                }
                                shell.unreadNotifications = (shell.unreadNotifications - 1).coerceAtLeast(0)
                                scope.launch {
                                    runCatching {
                                        offline.enqueueMutation(
                                            method = "POST",
                                            path = "/api/v1/me/notifications/${notification.id}/read",
                                            bodyJson = "{}",
                                            label = L.text("mobile.notifications.markReadLabel"),
                                            accessToken = token,
                                        )
                                    }
                                }
                            }
                            notification.actionUrl?.let { url ->
                                shell.openDeepLink(DeepLinkRouter.resolve(url))
                            }
                        },
                        modifier = Modifier.semantics {
                            contentDescription = "$readLabel. ${notification.title}. ${notification.body}"
                        },
                    ) {
                        Row(
                            verticalAlignment = Alignment.Top,
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                        ) {
                            Box(
                                modifier = Modifier
                                    .size(32.dp)
                                    .clip(RoundedCornerShape(10.dp))
                                    .background(
                                        LexturesColors.BrandTeal.copy(alpha = if (isDarkTheme()) 0.18f else 0.14f),
                                    ),
                                contentAlignment = Alignment.Center,
                            ) {
                                Icon(
                                    iconFor(notification.eventType),
                                    contentDescription = null,
                                    tint = accentColor(),
                                    modifier = Modifier.size(16.dp),
                                )
                            }
                            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(3.dp)) {
                                Row(verticalAlignment = Alignment.Top) {
                                    Text(
                                        text = notification.title,
                                        fontSize = 14.sp,
                                        fontWeight = if (notification.isRead) FontWeight.Normal else FontWeight.SemiBold,
                                        color = textPrimary(),
                                        modifier = Modifier.weight(1f),
                                    )
                                    Text(
                                        text = LmsDates.relative(notification.createdAt),
                                        fontSize = 11.sp,
                                        color = textSecondary(),
                                    )
                                }
                                Text(
                                    text = eventTypeLabel(notification.eventType),
                                    fontSize = 11.sp,
                                    fontWeight = FontWeight.Medium,
                                    color = accentColor(),
                                )
                                if (notification.body.isNotEmpty()) {
                                    Text(
                                        text = notification.body,
                                        fontSize = 12.sp,
                                        color = textSecondary(),
                                        maxLines = 3,
                                        overflow = TextOverflow.Ellipsis,
                                    )
                                }
                            }
                            if (!notification.isRead) {
                                Box(
                                    modifier = Modifier
                                        .padding(top = 6.dp)
                                        .size(8.dp)
                                        .clip(CircleShape)
                                        .background(LexturesColors.Coral),
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

private fun eventTypeLabel(eventType: String): String {
    val key = NotificationLogic.eventLabelKey(eventType)
    val localized = L.text(key)
    return if (localized == key) eventType.replace('_', ' ') else localized
}

private fun iconFor(eventType: String): ImageVector = when (NotificationLogic.category(eventType)) {
    NotificationCategory.Grades -> Icons.Default.Verified
    NotificationCategory.Messages -> Icons.Default.Email
    NotificationCategory.Assignments, NotificationCategory.Reminders -> Icons.Default.Schedule
    NotificationCategory.Announcements -> Icons.Default.Campaign
    NotificationCategory.Discussions -> Icons.Default.Forum
    else -> Icons.Default.Notifications
}

/** Full announcement history ("See all" from the dashboard banner). */
@Composable
fun AnnouncementsScreen(
    session: AuthSession,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()

    var broadcasts by remember { mutableStateOf<List<Broadcast>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    BackHandler(onBack = onBack)

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            broadcasts = LmsApi.fetchMyBroadcasts(token)
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp, end = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = "Announcements",
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            errorMessage?.let { message ->
                item { LmsErrorBanner(message) }
            }

            if (loading && broadcasts.isEmpty()) {
                item { LmsSkeletonList(count = 4) }
            } else if (broadcasts.isEmpty()) {
                item {
                    LmsEmptyState(
                        icon = Icons.Default.Campaign,
                        title = "No announcements",
                        message = "School-wide announcements will appear here.",
                    )
                }
            } else {
                items(broadcasts, key = { it.id }) { broadcast ->
                    LmsCard(accent = if (broadcast.isEmergency) LexturesColors.Coral else null) {
                        Row(verticalAlignment = Alignment.Top) {
                            Text(
                                text = broadcast.subject,
                                fontSize = 14.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = textPrimary(),
                                modifier = Modifier.weight(1f),
                            )
                            Text(
                                text = LmsDates.relative(broadcast.sentAt ?: broadcast.createdAt),
                                fontSize = 11.sp,
                                color = textSecondary(),
                            )
                        }
                        Text(
                            text = broadcast.body,
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    }
                }
            }
        }
    }
}
