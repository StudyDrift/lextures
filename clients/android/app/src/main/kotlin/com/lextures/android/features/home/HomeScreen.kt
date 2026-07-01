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
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.EditNote
import androidx.compose.material.icons.filled.FamilyRestroom
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
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.AccountProfile
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MeProfile
import com.lextures.android.core.navigation.MobileDestinations
import com.lextures.android.core.navigation.MobileIaPreferences
import com.lextures.android.core.navigation.MobileProfileDepthPreferences
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.navigation.MobileRoleContext
import com.lextures.android.core.navigation.RoleSnapshot
import com.lextures.android.core.navigation.ShellTab
import com.lextures.android.core.push.PushManager
import com.lextures.android.core.routing.DeepLinkDestination
import com.lextures.android.features.courses.CoursesTab
import com.lextures.android.features.dashboard.DashboardTab
import com.lextures.android.features.inbox.InboxTab
import com.lextures.android.features.notebooks.NotebooksTab
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.planner.PlannerScreen
import com.lextures.android.features.planner.PlannerTab
import com.lextures.android.features.profile.ProfileTab
import com.lextures.android.features.search.UniversalSearchScreen

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
    var accountProfile by mutableStateOf<AccountProfile?>(null)
    var unreadInbox by mutableIntStateOf(0)
    var unreadNotifications by mutableIntStateOf(0)
    var pendingDeepLink by mutableStateOf<DeepLinkDestination?>(null)
    var roleSnapshot by mutableStateOf(RoleSnapshot())
    var activeRoleContext by mutableStateOf(MobileRoleContext.Learning)
    var platformFeatures by mutableStateOf(MobilePlatformFeatures())
    var iaRedesignEnabled by mutableStateOf(false)
    var showUniversalSearch by mutableStateOf(false)
    var universalSearchEnabled by mutableStateOf(false)
    var profileDepthEnabled by mutableStateOf(false)
    var pendingMoreDestination by mutableStateOf<com.lextures.android.core.navigation.MoreDestination?>(null)

    val shellTabs: List<ShellTab>
        get() = if (iaRedesignEnabled) {
            MobileDestinations.shellTabs(activeRoleContext)
        } else {
            listOf(
                ShellTab.Home,
                ShellTab.Courses,
                ShellTab.Notebooks,
                ShellTab.Inbox,
                ShellTab.Profile,
            )
        }

    fun openDeepLink(destination: DeepLinkDestination) {
        pendingDeepLink = destination
        selectedTabOverride = when (destination) {
            DeepLinkDestination.Home -> shellTabKey(ShellTab.Home)
            DeepLinkDestination.Inbox -> shellTabKey(ShellTab.Inbox)
            is DeepLinkDestination.Course -> shellTabKey(ShellTab.Courses)
        }
    }

    fun navigateFromSearch(path: String) {
        val target = com.lextures.android.core.search.SearchPathNavigator.resolve(path) ?: return
        when (target) {
            is com.lextures.android.core.search.SearchNavigationTarget.ShellTabTarget -> {
                selectedTabOverride = shellTabKey(target.tab)
            }
            is com.lextures.android.core.search.SearchNavigationTarget.DeepLinkTarget -> {
                openDeepLink(target.destination)
            }
            is com.lextures.android.core.search.SearchNavigationTarget.MoreTarget -> {
                selectedTabOverride = shellTabKey(ShellTab.Profile)
                pendingMoreDestination = target.destination
            }
        }
    }

    fun consumePendingMoreDestination(): com.lextures.android.core.navigation.MoreDestination? {
        val destination = pendingMoreDestination
        pendingMoreDestination = null
        return destination
    }

    fun setRoleContext(context: MobileRoleContext, onTabChanged: (String) -> Unit) {
        activeRoleContext = context
        MobileIaPreferences.saveRoleContext(androidContext, context)
        if (!shellTabs.any { shellTabKey(it) == selectedTabOverride }) {
            shellTabs.firstOrNull()?.let { onTabChanged(shellTabKey(it)) }
        }
    }

    var selectedTabOverride by mutableStateOf<String?>(null)
    lateinit var androidContext: android.content.Context

    suspend fun refresh(accessToken: String?) {
        val token = accessToken ?: return
        runCatching { LmsApi.fetchMe(token) }.getOrNull()?.let { profile = it }
        runCatching { LmsApi.fetchAccountProfile(token) }.getOrNull()?.let { accountProfile = it }
        runCatching { LmsApi.fetchUnreadInboxCount(token) }.getOrNull()?.let { unreadInbox = it }
        runCatching { LmsApi.fetchNotifications(token) }.getOrNull()?.let {
            unreadNotifications = it.unreadCount
        }
        val features = MobilePlatformFeatures.from(runCatching { LmsApi.fetchPlatformFeatures(token) }.getOrNull())
        platformFeatures = features
        if (features.ffMobileIaRedesign) {
            iaRedesignEnabled = true
            MobileIaPreferences.setRedesignEnabled(androidContext, true)
        } else {
            iaRedesignEnabled = MobileIaPreferences.isRedesignEnabled(androidContext)
        }
        if (features.ffMobileUniversalSearch) {
            universalSearchEnabled = true
            MobileIaPreferences.setUniversalSearchEnabled(androidContext, true)
        } else {
            universalSearchEnabled = MobileIaPreferences.isUniversalSearchEnabled(androidContext)
        }
        if (features.ffMobileProfileDepth) {
            profileDepthEnabled = true
            MobileProfileDepthPreferences.setEnabled(androidContext, true)
        } else {
            profileDepthEnabled = MobileProfileDepthPreferences.isEnabled(androidContext)
        }
        val courses = runCatching { LmsApi.fetchCourses(token) }.getOrDefault(emptyList())
        val permissions = runCatching { LmsApi.fetchMyPermissions(token) }.getOrDefault(emptyList())
        roleSnapshot = MobileDestinations.buildRoleSnapshot(permissions, courses)
        activeRoleContext = roleSnapshot.resolvedContext(MobileIaPreferences.loadRoleContext(androidContext))
    }
}

/** Post-auth shell: Home, Courses, Notebooks, Inbox, Profile behind a floating pill tab bar. */
@Composable
fun HomeScreen(
    session: AuthSession,
    modifier: Modifier = Modifier,
    initialDeepLink: DeepLinkDestination? = null,
) {
    var selectedTab by rememberSaveable { mutableStateOf(HomeTab.Dashboard.name) }
    val shell = remember { HomeShellState() }
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val pushManager = remember { PushManager.getInstance(context) }
    val externalDeepLink by pushManager.pendingDeepLink.collectAsState()

    LaunchedEffect(Unit) {
        shell.androidContext = context.applicationContext
    }

    LaunchedEffect(initialDeepLink) {
        initialDeepLink?.let { shell.openDeepLink(it) }
    }

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

    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    if (shell.showUniversalSearch) {
        Dialog(
            onDismissRequest = { shell.showUniversalSearch = false },
            properties = DialogProperties(usePlatformDefaultWidth = false),
        ) {
            if (shell.universalSearchEnabled) {
                UniversalSearchScreen(
                    session = session,
                    shell = shell,
                    onDismiss = { shell.showUniversalSearch = false },
                    isOnline = isOnline,
                )
            } else {
                UniversalSearchPlaceholder(onDismiss = { shell.showUniversalSearch = false })
            }
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
                if (shell.iaRedesignEnabled) {
                    IaTabContent(selectedTab = selectedTab, session = session, shell = shell)
                } else {
                    LegacyTabContent(
                        selectedTab = selectedTab,
                        session = session,
                        shell = shell,
                        onSelectTab = { selectedTab = it },
                    )
                }
            }

            if (shell.iaRedesignEnabled) {
                IaShellTabBar(
                    shell = shell,
                    selected = selectedTab,
                    onSelect = { selectedTab = it },
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(start = 24.dp, end = 24.dp, top = 8.dp, bottom = 6.dp),
                )
            } else {
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
}

@Composable
private fun LegacyTabContent(
    selectedTab: String,
    session: AuthSession,
    shell: HomeShellState,
    onSelectTab: (String) -> Unit,
) {
    when (selectedTab) {
        HomeTab.Dashboard.name -> DashboardTab(
            session = session,
            shell = shell,
            onOpenProfile = { onSelectTab(HomeTab.Profile.name) },
            modifier = Modifier.fillMaxSize(),
        )
        HomeTab.Courses.name -> CoursesTab(session = session, shell = shell, modifier = Modifier.fillMaxSize())
        HomeTab.Notebooks.name -> NotebooksTab(session = session, modifier = Modifier.fillMaxSize())
        HomeTab.Inbox.name -> InboxTab(
            session = session,
            onUnreadChanged = { shell.unreadInbox = it },
            modifier = Modifier.fillMaxSize(),
        )
        HomeTab.Profile.name -> ProfileTab(session = session, shell = shell, modifier = Modifier.fillMaxSize())
    }
}

@Composable
private fun IaTabContent(
    selectedTab: String,
    session: AuthSession,
    shell: HomeShellState,
) {
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    when (selectedTab) {
        shellTabKey(ShellTab.Home) -> DashboardTab(
            session = session,
            shell = shell,
            onOpenProfile = { shell.selectedTabOverride = shellTabKey(ShellTab.Profile) },
            modifier = Modifier.fillMaxSize(),
        )
        shellTabKey(ShellTab.Courses) -> CoursesTab(session = session, shell = shell, modifier = Modifier.fillMaxSize())
        shellTabKey(ShellTab.Notebooks) -> NotebooksTab(session = session, modifier = Modifier.fillMaxSize())
        shellTabKey(ShellTab.Inbox) -> InboxTab(
            session = session,
            onUnreadChanged = { shell.unreadInbox = it },
            modifier = Modifier.fillMaxSize(),
        )
        shellTabKey(ShellTab.Profile) -> ProfileTab(session = session, shell = shell, modifier = Modifier.fillMaxSize())
        shellTabKey(ShellTab.Teach) -> TeachHubScreen(session = session, shell = shell, modifier = Modifier.fillMaxSize())
        shellTabKey(ShellTab.Children) -> ChildrenPlaceholderScreen(modifier = Modifier.fillMaxSize())
        shellTabKey(ShellTab.Calendar) -> PlannerScreen(
            session = session,
            offline = offline,
            isOnline = isOnline,
            initialTab = PlannerTab.Calendar,
            modifier = Modifier.fillMaxSize(),
        )
    }
}

fun shellTabKey(tab: ShellTab): String = when (tab) {
    ShellTab.Home -> HomeTab.Dashboard.name
    ShellTab.Courses -> HomeTab.Courses.name
    ShellTab.Notebooks -> HomeTab.Notebooks.name
    ShellTab.Inbox -> HomeTab.Inbox.name
    ShellTab.Profile -> HomeTab.Profile.name
    ShellTab.Teach -> "teach"
    ShellTab.Children -> "children"
    ShellTab.Calendar -> "calendar"
}

@Composable
fun IaShellTabBar(
    shell: HomeShellState,
    selected: String,
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
        shell.shellTabs.forEach { tab ->
            val key = shellTabKey(tab)
            val isSelected = selected == key
            Box(
                modifier = Modifier
                    .size(44.dp)
                    .clip(CircleShape)
                    .background(if (isSelected) LexturesColors.BrandCream else Color.Transparent)
                    .clickable { onSelect(key) }
                    .semantics {
                        contentDescription = L.text(context, localePrefs, shellTabLabelRes(tab))
                    },
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    shellTabIcon(tab),
                    contentDescription = null,
                    tint = when {
                        isSelected -> LexturesColors.PrimaryDeep
                        dark -> LexturesColors.TextSecondaryDark
                        else -> Color.White.copy(alpha = 0.72f)
                    },
                    modifier = Modifier.size(21.dp),
                )
                if (tab == ShellTab.Inbox && shell.unreadInbox > 0) {
                    Text(
                        text = if (shell.unreadInbox > 99) "99+" else "${shell.unreadInbox}",
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

private fun shellTabLabelRes(tab: ShellTab): Int = when (tab) {
    ShellTab.Home -> R.string.tabs_home
    ShellTab.Courses -> R.string.tabs_courses
    ShellTab.Notebooks -> R.string.tabs_notebooks
    ShellTab.Inbox -> R.string.tabs_inbox
    ShellTab.Profile -> R.string.tabs_profile
    ShellTab.Teach -> R.string.mobile_ia_tabs_teach
    ShellTab.Children -> R.string.mobile_ia_tabs_children
    ShellTab.Calendar -> R.string.mobile_ia_tabs_calendar
}

private fun shellTabIcon(tab: ShellTab): ImageVector = when (tab) {
    ShellTab.Home -> Icons.Default.Home
    ShellTab.Courses -> Icons.AutoMirrored.Filled.MenuBook
    ShellTab.Notebooks -> Icons.Default.EditNote
    ShellTab.Inbox -> Icons.Default.Inbox
    ShellTab.Profile -> Icons.Default.Person
    ShellTab.Teach -> Icons.Default.CheckCircle
    ShellTab.Children -> Icons.Default.FamilyRestroom
    ShellTab.Calendar -> Icons.Default.CalendarMonth
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