package com.lextures.android.features.home

import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
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
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.mutableFloatStateOf
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
import androidx.compose.ui.unit.IntOffset
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
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.navigation.CourseWorkspaceSection
import com.lextures.android.core.navigation.DrawerGroup
import com.lextures.android.core.navigation.DrawerState
import com.lextures.android.core.navigation.RootDestination
import com.lextures.android.core.navigation.MobileDestinations
import com.lextures.android.core.navigation.MobileIaPreferences
import com.lextures.android.core.navigation.MobileProfileDepthPreferences
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.navigation.MobileRoleContext
import com.lextures.android.core.navigation.RoleSnapshot
import com.lextures.android.core.navigation.ShellTab
import com.lextures.android.core.push.PushManager
import com.lextures.android.core.realtime.RealtimeManager
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
import kotlin.math.roundToInt

data class RootNavigationTransition(
    val outgoing: RootDestination,
    val drivenByDrawer: Boolean,
)

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
    var pendingReview by mutableStateOf(false)
    var pendingInsights by mutableStateOf(false)
    var pendingCheckout by mutableStateOf<com.lextures.android.core.lms.PendingCheckoutContext?>(null)
    var checkoutReturnPhase by mutableStateOf<com.lextures.android.core.lms.CheckoutReturnPhase?>(null)
    var pendingBilling by mutableStateOf(false)

    // Drawer navigation (web-parity sidebar)
    var drawerState by mutableStateOf(DrawerState.None)
    var rootDestination by mutableStateOf(RootDestination.Dashboard)
    var activeCourse by mutableStateOf<CourseSummary?>(null)
    /** The top-level pane the active course was opened under (Dashboard or Courses). */
    var activeCourseRoot by mutableStateOf(RootDestination.Courses)
    var activeCourseSection by mutableStateOf(CourseWorkspaceSection.Modules)
    var activeCourseSections by mutableStateOf<List<CourseWorkspaceSection>>(emptyList())

    /** Courses the user pinned on web/mobile — surfaced in the global drawer. */
    var pinnedCourses by mutableStateOf<List<CourseSummary>>(emptyList())

    /** Active root-pane push (outgoing screen + whether progress tracks drawer close). */
    var rootNavigationTransition by mutableStateOf<RootNavigationTransition?>(null)

    val globalDrawerGroups: List<DrawerGroup>
        get() = MobileDestinations.globalDrawerGroups(activeRoleContext, platformFeatures)

    /** Whether the course drawer is reachable (course open and its host pane visible). */
    val courseAvailable: Boolean
        get() = activeCourse != null && rootDestination == activeCourseRoot

    fun select(destination: RootDestination) {
        val closingDrawer = drawerState != DrawerState.None
        if (sharesRootPane(destination, rootDestination)) {
            if (closingDrawer) drawerState = DrawerState.None
            return
        }
        if (closingDrawer) {
            rootNavigationTransition = RootNavigationTransition(
                outgoing = rootDestination,
                drivenByDrawer = true,
            )
            drawerState = DrawerState.None
            rootDestination = destination
            return
        }
        rootNavigationTransition = RootNavigationTransition(
            outgoing = rootDestination,
            drivenByDrawer = false,
        )
        rootDestination = destination
    }

    fun completeRootNavigationTransition() {
        rootNavigationTransition = null
    }

    fun openGlobalDrawer() { drawerState = DrawerState.Global }
    fun closeDrawer() { drawerState = DrawerState.None }

    private fun sharesRootPane(lhs: RootDestination, rhs: RootDestination): Boolean {
        val profilePane = setOf(RootDestination.Profile, RootDestination.Settings)
        if (lhs in profilePane && rhs in profilePane) return true
        return lhs == rhs
    }
    fun exitCourseToDashboard() {
        activeCourse = null
        select(RootDestination.Dashboard)
    }

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
        select(
            when (destination) {
                DeepLinkDestination.Home -> RootDestination.Dashboard
                DeepLinkDestination.Inbox -> RootDestination.Inbox
                DeepLinkDestination.Review -> {
                    pendingReview = true
                    RootDestination.Dashboard
                }
                DeepLinkDestination.Insights -> {
                    pendingInsights = true
                    RootDestination.Dashboard
                }
                DeepLinkDestination.Billing -> {
                    pendingBilling = true
                    RootDestination.Profile
                }
                DeepLinkDestination.Credentials -> {
                    pendingMoreDestination = com.lextures.android.core.navigation.MoreDestination.Credentials
                    RootDestination.Profile
                }
                is DeepLinkDestination.CheckoutSuccess -> {
                    checkoutReturnPhase = com.lextures.android.core.lms.CheckoutReturnPhase.Success(destination.courseId)
                    RootDestination.Dashboard
                }
                DeepLinkDestination.CheckoutCancel -> {
                    pendingCheckout = null
                    checkoutReturnPhase = com.lextures.android.core.lms.CheckoutReturnPhase.Cancel
                    RootDestination.Dashboard
                }
                is DeepLinkDestination.Course -> RootDestination.Courses
            },
        )
    }

    fun consumePendingBilling(): Boolean {
        val pending = pendingBilling
        pendingBilling = false
        return pending
    }

    fun consumePendingReview(): Boolean {
        val pending = pendingReview
        pendingReview = false
        return pending
    }

    fun consumePendingInsights(): Boolean {
        val pending = pendingInsights
        pendingInsights = false
        return pending
    }

    fun navigateFromSearch(path: String) {
        val target = com.lextures.android.core.search.SearchPathNavigator.resolve(path) ?: return
        when (target) {
            is com.lextures.android.core.search.SearchNavigationTarget.ShellTabTarget -> {
                select(rootFor(target.tab))
            }
            is com.lextures.android.core.search.SearchNavigationTarget.DeepLinkTarget -> {
                openDeepLink(target.destination)
            }
            is com.lextures.android.core.search.SearchNavigationTarget.MoreTarget -> {
                select(RootDestination.Profile)
                pendingMoreDestination = target.destination
            }
        }
    }

    private fun rootFor(tab: ShellTab): RootDestination = when (tab) {
        ShellTab.Home -> RootDestination.Dashboard
        ShellTab.Courses -> RootDestination.Courses
        ShellTab.Notebooks -> RootDestination.Notebooks
        ShellTab.Inbox -> RootDestination.Inbox
        ShellTab.Profile -> RootDestination.Profile
        ShellTab.Teach -> RootDestination.Teach
        ShellTab.Children -> RootDestination.Children
        ShellTab.Calendar -> RootDestination.Calendar
    }

    fun consumePendingMoreDestination(): com.lextures.android.core.navigation.MoreDestination? {
        val destination = pendingMoreDestination
        pendingMoreDestination = null
        return destination
    }

    fun setRoleContext(context: MobileRoleContext, onTabChanged: (String) -> Unit = {}) {
        activeRoleContext = context
        MobileIaPreferences.saveRoleContext(androidContext, context)
        val available = globalDrawerGroups.flatMap { it.items }
        if (rootDestination !in available) {
            available.firstOrNull()?.let { select(it) }
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
        pinnedCourses = courses.filter { it.isPinned }
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
            RealtimeManager.configure { session.accessToken.value }
        }
    }

    val mailboxRevision by RealtimeManager.mailboxRevision.collectAsState()
    val coursesRevision by RealtimeManager.coursesRevision.collectAsState()
    val enrollmentsRevision by RealtimeManager.enrollmentsRevision.collectAsState()
    val notificationsRevision by RealtimeManager.notificationsRevision.collectAsState()
    val realtimeRevisionSum = mailboxRevision + coursesRevision + enrollmentsRevision + notificationsRevision
    LaunchedEffect(realtimeRevisionSum) {
        if (realtimeRevisionSum > 0) {
            shell.refresh(accessToken)
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
        var drawerOpenProgress by remember { mutableFloatStateOf(0f) }
        com.lextures.android.features.navigation.DrawerScaffold(
            state = shell.drawerState,
            courseAvailable = shell.courseAvailable,
            onStateChange = { shell.drawerState = it },
            onDrawerProgress = { drawerOpenProgress = it },
            globalPanel = { com.lextures.android.features.navigation.GlobalDrawer(shell, accessToken) },
            coursePanel = { com.lextures.android.features.navigation.CourseDrawer(shell) },
        ) {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .navigationBarsPadding(),
            ) {
                RootContent(
                    destination = shell.rootDestination,
                    drawerOpenProgress = drawerOpenProgress,
                    session = session,
                    shell = shell,
                )
            }
        }

        shell.checkoutReturnPhase?.let { phase ->
            val localePrefs = com.lextures.android.core.i18n.LocalLocalePreferences.current
            com.lextures.android.features.billing.CheckoutReturnOverlay(
                session = session,
                shell = shell,
                localePrefs = localePrefs,
                phase = phase,
                onDismiss = { shell.checkoutReturnPhase = null },
            )
        }
    }
}

@Composable
private fun RootContent(
    destination: RootDestination,
    drawerOpenProgress: Float,
    session: AuthSession,
    shell: HomeShellState,
) {
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val transition = shell.rootNavigationTransition
    val paneTransitionTarget = if (transition != null && !transition.drivenByDrawer) 1f else 0f
    val paneTransitionProgress by animateFloatAsState(
        targetValue = paneTransitionTarget,
        animationSpec = tween(durationMillis = 350, easing = FastOutSlowInEasing),
        label = "paneTransition",
    )
    val navigationCompletion = when {
        transition?.drivenByDrawer == true -> 1f - drawerOpenProgress
        transition != null -> paneTransitionProgress
        else -> 1f
    }

    LaunchedEffect(transition?.drivenByDrawer, drawerOpenProgress) {
        if (transition?.drivenByDrawer == true && drawerOpenProgress <= 0.01f) {
            shell.completeRootNavigationTransition()
        }
    }
    LaunchedEffect(transition, paneTransitionProgress) {
        if (transition != null && !transition.drivenByDrawer && paneTransitionProgress >= 0.99f) {
            shell.completeRootNavigationTransition()
        }
    }

    if (transition != null) {
        BoxWithConstraints(Modifier.fillMaxSize()) {
            val widthPx = constraints.maxWidth.toFloat()
            val incomingOffset = (widthPx * (1f - navigationCompletion)).roundToInt()
            val outgoingOffset = (-widthPx * navigationCompletion).roundToInt()
            Box(
                Modifier
                    .fillMaxSize()
                    .offset { IntOffset(outgoingOffset, 0) },
            ) {
                RootPane(
                    destination = transition.outgoing,
                    session = session,
                    shell = shell,
                    offline = offline,
                    isOnline = isOnline,
                    modifier = Modifier.fillMaxSize(),
                )
            }
            Box(
                Modifier
                    .fillMaxSize()
                    .offset { IntOffset(incomingOffset, 0) },
            ) {
                RootPane(
                    destination = destination,
                    session = session,
                    shell = shell,
                    offline = offline,
                    isOnline = isOnline,
                    modifier = Modifier.fillMaxSize(),
                )
            }
        }
    } else {
        RootPane(
            destination = destination,
            session = session,
            shell = shell,
            offline = offline,
            isOnline = isOnline,
            modifier = Modifier.fillMaxSize(),
        )
    }
}

@Composable
private fun RootPane(
    destination: RootDestination,
    session: AuthSession,
    shell: HomeShellState,
    offline: OfflineService,
    isOnline: Boolean,
    modifier: Modifier = Modifier,
) {
    when (destination) {
        RootDestination.Dashboard -> DashboardTab(
            session = session,
            shell = shell,
            onOpenProfile = { shell.select(RootDestination.Profile) },
            modifier = modifier,
        )
        RootDestination.Courses -> CoursesTab(session = session, shell = shell, modifier = modifier)
        RootDestination.Notebooks -> NotebooksTab(session = session, modifier = modifier)
        RootDestination.GlobalNotebook -> NotebooksTab(
            session = session,
            initialGlobal = true,
            modifier = modifier,
        )
        RootDestination.Inbox -> InboxTab(
            session = session,
            onUnreadChanged = { shell.unreadInbox = it },
            modifier = modifier,
        )
        RootDestination.Profile, RootDestination.Settings ->
            ProfileTab(session = session, shell = shell, modifier = modifier)
        RootDestination.Teach -> TeachHubScreen(session = session, shell = shell, modifier = modifier)
        RootDestination.Children -> ChildrenPlaceholderScreen(modifier = modifier)
        RootDestination.Calendar -> PlannerScreen(
            session = session,
            offline = offline,
            isOnline = isOnline,
            initialTab = PlannerTab.Calendar,
            onBack = { shell.select(RootDestination.Dashboard) },
            modifier = modifier,
        )
        RootDestination.Todos -> PlannerScreen(
            session = session,
            offline = offline,
            isOnline = isOnline,
            initialTab = PlannerTab.Todos,
            onBack = { shell.select(RootDestination.Dashboard) },
            modifier = modifier,
        )
        RootDestination.Review -> com.lextures.android.features.review.ReviewHomeScreen(
            session = session,
            shell = shell,
            onBack = { shell.select(RootDestination.Dashboard) },
            modifier = modifier,
        )
        RootDestination.Insights -> com.lextures.android.features.insights.InsightsScreen(
            session = session,
            onOpenCourse = { course ->
                shell.activeCourse = course
                shell.activeCourseRoot = RootDestination.Dashboard
                shell.activeCourseSection = com.lextures.android.core.navigation.CourseWorkspaceSection.Modules
                shell.select(RootDestination.Courses)
            },
            onOpenReview = { shell.select(RootDestination.Review) },
            modifier = modifier,
        )
        RootDestination.Accommodations -> com.lextures.android.features.profile.MyAccommodationsScreen(
            session = session,
            onBack = { shell.select(RootDestination.Dashboard) },
            modifier = modifier,
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