package com.lextures.android.features.home

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.MenuBook
import androidx.compose.material.icons.filled.EditNote
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Inbox
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MeProfile
import com.lextures.android.core.push.PushManager
import com.lextures.android.core.routing.DeepLinkDestination
import com.lextures.android.features.courses.CoursesTab
import com.lextures.android.features.dashboard.DashboardTab
import com.lextures.android.features.inbox.InboxTab
import com.lextures.android.features.notebooks.NotebooksTab
import com.lextures.android.features.profile.ProfileTab

enum class HomeTab(val labelRes: Int, val icon: ImageVector) {
    Dashboard(R.string.tabs_home, Icons.Default.Home),
    Courses(R.string.tabs_courses, Icons.AutoMirrored.Filled.MenuBook),
    Notebooks(R.string.tabs_notebooks, Icons.Default.EditNote),
    Inbox(R.string.tabs_inbox, Icons.Default.Inbox),
    Profile(R.string.tabs_profile, Icons.Default.Person),
}

/**
 * Cross-tab state: viewer profile and unread counters. Single source for the
 * tab badge, Home stat card, and notification bell dot.
 */
class HomeShellState {
    var profile by mutableStateOf<MeProfile?>(null)
    var unreadInbox by mutableIntStateOf(0)
    var unreadNotifications by mutableIntStateOf(0)
    var pendingDeepLink by mutableStateOf<DeepLinkDestination?>(null)

    fun openDeepLink(destination: DeepLinkDestination) {
        pendingDeepLink = destination
        selectedTabOverride = when (destination) {
            DeepLinkDestination.Home -> HomeTab.Dashboard.name
            DeepLinkDestination.Inbox -> HomeTab.Inbox.name
            is DeepLinkDestination.Course -> HomeTab.Courses.name
        }
    }

    var selectedTabOverride by mutableStateOf<String?>(null)

    suspend fun refresh(accessToken: String?) {
        val token = accessToken ?: return
        runCatching { LmsApi.fetchMe(token) }.getOrNull()?.let { profile = it }
        runCatching { LmsApi.fetchUnreadInboxCount(token) }.getOrNull()?.let { unreadInbox = it }
        runCatching { LmsApi.fetchNotifications(token) }.getOrNull()?.let {
            unreadNotifications = it.unreadCount
        }
    }
}

/** Post-auth shell: Home, Courses, Notebooks, Inbox, Profile behind a floating pill tab bar. */
@Composable
fun HomeScreen(session: AuthSession, modifier: Modifier = Modifier) {
    var selectedTab by rememberSaveable { mutableStateOf(HomeTab.Dashboard.name) }
    val shell = remember { HomeShellState() }
    val accessToken by session.accessToken.collectAsState()
    val context = androidx.compose.ui.platform.LocalContext.current
    val pushManager = remember { PushManager.getInstance(context) }
    val externalDeepLink by pushManager.pendingDeepLink.collectAsState()

    LaunchedEffect(accessToken) {
        shell.refresh(accessToken)
        if (accessToken != null) {
            pushManager.requestTokenSync()
        }
    }

    LaunchedEffect(shell.selectedTabOverride) {
        shell.selectedTabOverride?.let {
            selectedTab = it
            shell.selectedTabOverride = null
        }
    }

    LaunchedEffect(externalDeepLink) {
        externalDeepLink?.let { destination ->
            shell.openDeepLink(destination)
            pushManager.consumePendingDeepLink()
        }
    }

    Box(
        modifier = modifier
            .fillMaxSize()
            .background(sceneBackground()),
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .navigationBarsPadding(),
        ) {
            Box(modifier = Modifier.weight(1f)) {
                when (selectedTab) {
                    HomeTab.Dashboard.name -> DashboardTab(
                        session = session,
                        shell = shell,
                        onOpenProfile = { selectedTab = HomeTab.Profile.name },
                        modifier = Modifier.fillMaxSize(),
                    )
                    HomeTab.Courses.name -> CoursesTab(
                        session = session,
                        shell = shell,
                        modifier = Modifier.fillMaxSize(),
                    )
                    HomeTab.Notebooks.name -> NotebooksTab(session = session, modifier = Modifier.fillMaxSize())
                    HomeTab.Inbox.name -> InboxTab(
                        session = session,
                        onUnreadChanged = { shell.unreadInbox = it },
                        modifier = Modifier.fillMaxSize(),
                    )
                    HomeTab.Profile.name -> ProfileTab(
                        session = session,
                        shell = shell,
                        modifier = Modifier.fillMaxSize(),
                    )
                }
            }

            LexturesTabBar(
                selected = selectedTab,
                unreadInbox = shell.unreadInbox,
                onSelect = { selectedTab = it },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(start = 24.dp, end = 24.dp, top = 8.dp, bottom = 6.dp),
            )
        }
    }
}

/** Deep-teal floating capsule: selected tab gets a cream circular "puck". */
@Composable
fun LexturesTabBar(
    selected: String,
    unreadInbox: Int,
    onSelect: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val dark = isDarkTheme()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val shape = RoundedCornerShape(50)
    Row(
        modifier = modifier
            .fillMaxWidth()
            .shadow(
                elevation = if (dark) 0.dp else 12.dp,
                shape = shape,
                clip = false,
                ambientColor = LexturesColors.PrimaryDeep.copy(alpha = 0.6f),
                spotColor = LexturesColors.PrimaryDeep.copy(alpha = 0.6f),
            )
            .clip(shape)
            .background(if (dark) LexturesColors.CardBackgroundDark else LexturesColors.PrimaryDeep)
            .border(
                1.dp,
                if (dark) LexturesColors.FieldBorderDark else Color.White.copy(alpha = 0.08f),
                shape,
            )
            .padding(horizontal = 10.dp, vertical = 9.dp),
        horizontalArrangement = Arrangement.SpaceEvenly,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        HomeTab.entries.forEach { tab ->
            val isSelected = selected == tab.name
            Box(
                modifier = Modifier
                    .size(44.dp)
                    .clip(CircleShape)
                    .background(if (isSelected) LexturesColors.BrandCream else Color.Transparent)
                    .clickable { onSelect(tab.name) }
                    .semantics { contentDescription = L.text(context, localePrefs, tab.labelRes) },
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    tab.icon,
                    contentDescription = null,
                    tint = when {
                        isSelected -> LexturesColors.PrimaryDeep
                        dark -> LexturesColors.TextSecondaryDark
                        else -> Color.White.copy(alpha = 0.72f)
                    },
                    modifier = Modifier.size(21.dp),
                )
                if (tab == HomeTab.Inbox && unreadInbox > 0) {
                    Text(
                        text = if (unreadInbox > 99) "99+" else "$unreadInbox",
                        fontSize = 9.sp,
                        color = Color.White,
                        modifier = Modifier
                            .align(Alignment.TopEnd)
                            .offset(x = 2.dp, y = (-2).dp)
                            .clip(RoundedCornerShape(50))
                            .background(LexturesColors.Coral)
                            .padding(horizontal = 4.dp, vertical = 1.dp),
                    )
                }
            }
        }
    }
}
