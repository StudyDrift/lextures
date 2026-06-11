package com.lextures.android.features.home

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.MenuBook
import androidx.compose.material.icons.filled.EditNote
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Inbox
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.courses.CoursesTab
import com.lextures.android.features.dashboard.DashboardTab
import com.lextures.android.features.inbox.InboxTab
import com.lextures.android.features.notebooks.NotebooksTab

private enum class HomeTab(val label: String, val icon: ImageVector) {
    Dashboard("Home", Icons.Default.Home),
    Courses("Courses", Icons.AutoMirrored.Filled.MenuBook),
    Notebooks("Notebooks", Icons.Default.EditNote),
    Inbox("Inbox", Icons.Default.Inbox),
}

/** Post-auth shell: Dashboard, Courses, Notebooks, Inbox tabs. */
@Composable
fun HomeScreen(session: AuthSession, modifier: Modifier = Modifier) {
    var selectedTab by rememberSaveable { mutableStateOf(HomeTab.Dashboard.name) }
    var unreadInbox by remember { mutableIntStateOf(0) }
    val accessToken by session.accessToken.collectAsState()

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        unreadInbox = runCatching { LmsApi.fetchUnreadInboxCount(token) }.getOrDefault(unreadInbox)
    }

    Scaffold(
        modifier = modifier,
        containerColor = sceneBackground(),
        bottomBar = {
            NavigationBar(containerColor = cardBackground()) {
                HomeTab.entries.forEach { tab ->
                    NavigationBarItem(
                        selected = selectedTab == tab.name,
                        onClick = { selectedTab = tab.name },
                        colors = NavigationBarItemDefaults.colors(
                            selectedIconColor = accentColor(),
                            selectedTextColor = accentColor(),
                            unselectedIconColor = textSecondary(),
                            unselectedTextColor = textSecondary(),
                            indicatorColor = LexturesColors.BrandTeal.copy(alpha = 0.18f),
                        ),
                        icon = {
                            if (tab == HomeTab.Inbox && unreadInbox > 0) {
                                BadgedBox(
                                    badge = {
                                        Badge(containerColor = LexturesColors.Coral) { Text("$unreadInbox") }
                                    },
                                ) {
                                    Icon(tab.icon, contentDescription = tab.label)
                                }
                            } else {
                                Icon(tab.icon, contentDescription = tab.label)
                            }
                        },
                        label = { Text(tab.label) },
                    )
                }
            }
        },
    ) { padding ->
        val contentModifier = Modifier
            .fillMaxSize()
            .padding(padding)
        when (selectedTab) {
            HomeTab.Dashboard.name -> DashboardTab(
                session = session,
                unreadInbox = unreadInbox,
                modifier = contentModifier,
            )
            HomeTab.Courses.name -> CoursesTab(session = session, modifier = contentModifier)
            HomeTab.Notebooks.name -> NotebooksTab(session = session, modifier = contentModifier)
            HomeTab.Inbox.name -> InboxTab(
                session = session,
                onUnreadChanged = { unreadInbox = it },
                modifier = contentModifier,
            )
        }
    }
}
