import SwiftUI

// MARK: - Shell state

enum AppTab: String, CaseIterable, Identifiable {
    case home, courses, notebooks, inbox, profile

    var id: String { rawValue }

    var label: String {
        switch self {
        case .home: return L.text("tabs.home")
        case .courses: return L.text("tabs.courses")
        case .notebooks: return L.text("tabs.notebooks")
        case .inbox: return L.text("tabs.inbox")
        case .profile: return L.text("tabs.profile")
        }
    }

    var systemImage: String {
        switch self {
        case .home: return "house.fill"
        case .courses: return "books.vertical.fill"
        case .notebooks: return "square.and.pencil"
        case .inbox: return "tray.fill"
        case .profile: return "person.fill"
        }
    }
}

/// Cross-tab state: selected tab, viewer profile, and unread counters.
/// Single source for the tab badge, Home stat card, and notification bell dot.
@MainActor
@Observable
final class AppShellModel {
    var selectedTab: AppTab = .home
    var selectedShellTab: ShellTab = .home
    var profile: MeProfile?
    var accountProfile: AccountProfile?
    var unreadInbox = 0
    var unreadNotifications = 0
    var pendingDeepLink: DeepLinkDestination?
    var roleSnapshot = RoleSnapshot()
    var activeRoleContext: MobileRoleContext = .learning
    var platformFeatures = MobilePlatformFeatures()
    var showUniversalSearch = false
    var iaRedesignEnabled = MobileIaPreferences.isRedesignEnabled
    var universalSearchEnabled = MobileIaPreferences.isUniversalSearchEnabled
    var profileDepthEnabled = MobileProfileDepthPreferences.isEnabled
    var pendingMoreDestination: MoreDestination?
    var pendingReview = false
    var pendingInsights = false
    var pendingCheckout: PendingCheckoutContext?
    var checkoutReturnPhase: CheckoutReturnPhase?

    // MARK: Drawer navigation

    /// Current left-drawer state (two-level: course menu → global menu).
    var drawer: DrawerState = .none
    /// Selected top-level destination shown behind the drawer.
    var rootDestination: RootDestination = .dashboard
    /// The course currently being viewed, if any. Enables the course-scoped drawer
    /// and the "swipe once → course menu, swipe again → global menu" behavior.
    var activeCourse: CourseSummary?
    /// The top-level pane the active course was opened under (courses can open from
    /// Dashboard or Courses). The course drawer is only offered while that pane is shown.
    var activeCourseRoot: RootDestination = .courses
    /// Selected section within the active course (driven by the course drawer).
    var activeCourseSection: CourseWorkspaceSection = .modules
    /// Sections available for the active course (published by `CourseDetailView`).
    var activeCourseSections: [CourseWorkspaceSection] = []

    /// Courses the user pinned on web/mobile — surfaced in the global drawer.
    var pinnedCourses: [CourseSummary] = []

    /// Active root-pane push (outgoing screen + whether progress tracks drawer close).
    var rootNavigationTransition: RootNavigationTransition?

    var globalDrawerGroups: [DrawerGroup] {
        let uiMode = UIModeStore.shared.effectiveMode(roleContext: activeRoleContext)
        return MobileDestinations.globalDrawerGroups(
            context: activeRoleContext,
            platform: platformFeatures,
            uiMode: uiMode
        )
    }

    var effectiveUIMode: UIMode {
        UIModeStore.shared.effectiveMode(roleContext: activeRoleContext)
    }

    /// Selects a top-level destination and dismisses any open drawer.
    func select(_ destination: RootDestination) {
        let closingDrawer = drawer != .none
        if sharesRootPane(destination, rootDestination) {
            if closingDrawer {
                withAnimation(Self.drawerAnimation) { drawer = .none }
            }
            return
        }
        if closingDrawer {
            rootNavigationTransition = RootNavigationTransition(
                outgoing: rootDestination,
                drivenByDrawer: true
            )
            withAnimation(Self.drawerAnimation) { drawer = .none }
            rootDestination = destination
            return
        }
        rootNavigationTransition = RootNavigationTransition(
            outgoing: rootDestination,
            drivenByDrawer: false
        )
        rootDestination = destination
    }

    func completeRootNavigationTransition() {
        rootNavigationTransition = nil
    }

    private static let drawerAnimation = Animation.easeInOut(duration: 0.35)

    private func sharesRootPane(_ lhs: RootDestination, _ rhs: RootDestination) -> Bool {
        let profilePane: Set<RootDestination> = [.profile, .settings]
        if profilePane.contains(lhs), profilePane.contains(rhs) { return true }
        return lhs == rhs
    }

    /// Leading-edge swipe entry point implementing the two-level state machine.
    func edgeSwipeOpen() {
        switch drawer {
        case .none:
            drawer = activeCourse != nil ? .course : .global
        case .course:
            drawer = .global
        case .global:
            break
        }
    }

    func openGlobalDrawer() {
        withAnimation(Self.drawerAnimation) { drawer = .global }
    }
    func closeDrawer() {
        withAnimation(Self.drawerAnimation) { drawer = .none }
    }

    /// Leaves the active course and returns to the Dashboard (course drawer "Dashboard").
    func exitCourseToDashboard() {
        activeCourse = nil
        select(.dashboard)
    }

    private func rootDestination(for tab: ShellTab) -> RootDestination {
        switch tab {
        case .home: return .dashboard
        case .courses: return .courses
        case .notebooks: return .notebooks
        case .inbox: return .inbox
        case .profile: return .profile
        case .teach: return .teach
        case .children: return .children
        case .calendar: return .calendar
        }
    }

    var shellTabs: [ShellTab] {
        iaRedesignEnabled
            ? MobileDestinations.shellTabs(context: activeRoleContext)
            : AppTab.allCases.map { legacyShellTab(for: $0) }
    }

    var pendingBilling = false
    var pendingParentStudentId: String?
    var pendingParentRoute: ParentRoute?

    func consumePendingParentNavigation() -> (studentId: String?, route: ParentRoute?)? {
        defer {
            pendingParentStudentId = nil
            pendingParentRoute = nil
        }
        guard pendingParentStudentId != nil || pendingParentRoute != nil else { return nil }
        return (pendingParentStudentId, pendingParentRoute)
    }

    func openDeepLink(_ destination: DeepLinkDestination) {
        pendingDeepLink = destination
        switch destination {
        case .home:
            selectShellTab(.home)
        case .inbox:
            selectShellTab(.inbox)
        case .review:
            selectShellTab(.home)
            pendingReview = true
        case .insights:
            selectShellTab(.home)
            pendingInsights = true
        case .billing:
            selectShellTab(.profile)
            pendingBilling = true
        case .credentials:
            selectShellTab(.profile)
            pendingMoreDestination = .credentials
        case let .checkoutSuccess(courseId):
            checkoutReturnPhase = .success(courseId: courseId)
        case .checkoutCancel:
            pendingCheckout = nil
            checkoutReturnPhase = .cancel
        case .course:
            selectShellTab(.courses)
        case let .parent(studentId, section):
            if roleSnapshot.hasParentDashboard {
                setRoleContext(.parent)
            }
            if let studentId {
                MobileIaPreferences.saveSelectedChildId(studentId)
                pendingParentStudentId = studentId
            }
            pendingParentRoute = parentRoute(studentId: studentId, section: section)
            selectShellTab(.children)
        }
    }

    private func parentRoute(studentId: String?, section: ParentDeepLinkSection) -> ParentRoute? {
        let resolvedStudentId = studentId ?? pendingParentStudentId ?? MobileIaPreferences.loadSelectedChildId()
        guard let resolvedStudentId else {
            switch section {
            case .dashboard, .notificationPrefs:
                return section == .notificationPrefs ? .notificationPrefs : nil
            default:
                return nil
            }
        }
        switch section {
        case .dashboard:
            return nil
        case .grades:
            return .grades(studentId: resolvedStudentId)
        case .attendance:
            return .attendance(studentId: resolvedStudentId)
        case .conferences:
            return .conferences(studentId: resolvedStudentId)
        case .notificationPrefs:
            return .notificationPrefs
        }
    }

    func consumePendingBilling() -> Bool {
        defer { pendingBilling = false }
        return pendingBilling
    }

    func selectShellTab(_ tab: ShellTab) {
        selectedShellTab = tab
        if let legacy = MobileDestinations.legacyTab(from: tab) {
            selectedTab = legacy
        }
        select(rootDestination(for: tab))
    }

    func selectLegacyTab(_ tab: AppTab) {
        selectedTab = tab
        selectedShellTab = legacyShellTab(for: tab)
    }

    func setRoleContext(_ context: MobileRoleContext) {
        activeRoleContext = context
        MobileIaPreferences.saveRoleContext(context)
        let tabs = MobileDestinations.shellTabs(context: context)
        if !tabs.contains(selectedShellTab) {
            selectShellTab(tabs.first ?? .home)
        }
    }

    func consumePendingDeepLink() -> DeepLinkDestination? {
        defer { pendingDeepLink = nil }
        return pendingDeepLink
    }

    func consumePendingMoreDestination() -> MoreDestination? {
        defer { pendingMoreDestination = nil }
        return pendingMoreDestination
    }

    func consumePendingReview() -> Bool {
        defer { pendingReview = false }
        return pendingReview
    }

    func consumePendingInsights() -> Bool {
        defer { pendingInsights = false }
        return pendingInsights
    }

    func navigateFromSearch(path: String) {
        guard let target = SearchPathNavigator.resolve(path) else { return }
        switch target {
        case .shellTab(let tab):
            selectShellTab(tab)
        case .deepLink(let destination):
            openDeepLink(destination)
        case .more(let destination):
            selectShellTab(.profile)
            pendingMoreDestination = destination
        }
    }

    func refresh(accessToken: String?) async {
        guard let token = accessToken else { return }
        async let me = try? LMSAPI.fetchMe(accessToken: token)
        async let account = try? LMSAPI.fetchAccountProfile(accessToken: token)
        async let inbox = try? LMSAPI.fetchUnreadInboxCount(accessToken: token)
        async let notifications = try? LMSAPI.fetchNotifications(accessToken: token)
        async let permissions = try? LMSAPI.fetchMyPermissions(accessToken: token)
        async let platform = try? LMSAPI.fetchPlatformFeatures(accessToken: token)
        async let courses = try? LMSAPI.fetchCourses(accessToken: token)
        if let me = await me { profile = me }
        if let account = await account { accountProfile = account }
        if let inbox = await inbox { unreadInbox = inbox }
        if let page = await notifications { unreadNotifications = page.unreadCount }
        let features = MobilePlatformFeatures.from(await platform)
        platformFeatures = features
        UIModeStore.shared.updatePlatform(featureEnabled: features.ffUiMode)
        let readingApiEnabled = features.ffReadingPreferences
            || (features.readAloudEnabled && features.ffReadAloud)
        if features.ffUiMode || readingApiEnabled {
            await ReadingPreferencesStore.shared.loadFromServer(
                accessToken: token,
                apiEnabled: readingApiEnabled || features.ffUiMode,
                uiModeEnabled: features.ffUiMode
            )
        }
        if features.ffMobileIaRedesign {
            iaRedesignEnabled = true
            MobileIaPreferences.isRedesignEnabled = true
        }
        if features.ffMobileUniversalSearch {
            universalSearchEnabled = true
            MobileIaPreferences.isUniversalSearchEnabled = true
        }
        if features.ffMobileProfileDepth {
            profileDepthEnabled = true
            MobileProfileDepthPreferences.isEnabled = true
        } else {
            profileDepthEnabled = MobileProfileDepthPreferences.isEnabled
        }
        let courseList = await courses ?? []
        pinnedCourses = courseList.filter(\.isPinned)
        roleSnapshot = MobileDestinations.buildRoleSnapshot(
            permissions: await permissions ?? [],
            courses: courseList
        )
        activeRoleContext = roleSnapshot.resolvedContext(stored: MobileIaPreferences.loadRoleContext())
        _ = UIModeStore.shared.effectiveMode(roleContext: activeRoleContext)
        if iaRedesignEnabled {
            let tabs = MobileDestinations.shellTabs(context: activeRoleContext)
            if !tabs.contains(selectedShellTab) {
                selectShellTab(tabs.first ?? .home)
            }
        }
    }

    private func legacyShellTab(for tab: AppTab) -> ShellTab {
        switch tab {
        case .home: return .home
        case .courses: return .courses
        case .notebooks: return .notebooks
        case .inbox: return .inbox
        case .profile: return .profile
        }
    }
}

/// Tracks an in-flight root destination change (drawer-driven or tab-bar-driven).
struct RootNavigationTransition: Equatable {
    var outgoing: RootDestination
    var drivenByDrawer: Bool
}

/// Post-auth shell: Home, Courses, Notebooks, Inbox, Profile behind a floating pill tab bar.
struct MainTabView: View {
    @Environment(AuthSession.self) private var session
    var initialDeepLink: DeepLinkDestination?
    @State private var shell = AppShellModel()
    @Bindable private var realtime = RealtimeManager.shared
    /// 0 = drawer fully open, 1 = fully closed — reported by `DrawerScaffold`.
    @State private var drawerOpenProgress: CGFloat = 0
    /// 0 = transition start, 1 = settled — used for non-drawer root changes.
    @State private var paneTransitionProgress: CGFloat = 1

    var body: some View {
        DrawerScaffold(
            state: Bindable(shell).drawer,
            openProgress: $drawerOpenProgress,
            courseAvailable: shell.activeCourse != nil && shell.rootDestination == shell.activeCourseRoot,
            main: {
                tabContent
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            },
            globalPanel: { GlobalDrawer() },
            coursePanel: { CourseDrawer() }
        )
        .environment(shell)
        .overlay {
            if let phase = shell.checkoutReturnPhase {
                CheckoutReturnOverlay(phase: phase)
            }
        }
        .sheet(isPresented: Bindable(shell).showUniversalSearch) {
            if shell.universalSearchEnabled {
                UniversalSearchView()
            } else {
                UniversalSearchPlaceholder()
            }
        }
        .task {
            await shell.refresh(accessToken: session.accessToken)
        }
        .onOpenURL { url in
            shell.openDeepLink(DeepLinkRouter.resolve(url.absoluteString))
        }
        .onAppear {
            PushManager.shared.configure(accessToken: { session.accessToken }) { destination in
                shell.openDeepLink(destination)
            }
            RealtimeManager.shared.configure(accessToken: { session.accessToken })
            if let initialDeepLink {
                shell.openDeepLink(initialDeepLink)
            }
        }
        .onChange(of: session.accessToken) { _, token in
            if token != nil {
                Task { await PushManager.shared.syncTokenWithBackend() }
                RealtimeManager.shared.configure(accessToken: { session.accessToken })
            }
        }
        .onChange(of: realtime.mailboxRevision) { _, _ in
            Task { await shell.refresh(accessToken: session.accessToken) }
        }
        .onChange(of: realtime.coursesRevision) { _, _ in
            Task { await shell.refresh(accessToken: session.accessToken) }
        }
        .onChange(of: realtime.enrollmentsRevision) { _, _ in
            Task { await shell.refresh(accessToken: session.accessToken) }
        }
        .onChange(of: realtime.notificationsRevision) { _, _ in
            Task { await shell.refresh(accessToken: session.accessToken) }
        }
        .onChange(of: shell.rootNavigationTransition) { _, transition in
            guard let transition, !transition.drivenByDrawer else { return }
            paneTransitionProgress = 0
            withAnimation(.easeInOut(duration: 0.35)) {
                paneTransitionProgress = 1
            }
            Task {
                try? await Task.sleep(for: .milliseconds(360))
                if paneTransitionProgress >= 0.99 {
                    shell.completeRootNavigationTransition()
                }
            }
        }
        .onChange(of: drawerOpenProgress) { _, progress in
            guard shell.rootNavigationTransition?.drivenByDrawer == true else { return }
            if progress <= 0.01 {
                shell.completeRootNavigationTransition()
            }
        }
    }

    /// 0 = transition just started, 1 = navigation complete.
    private var navigationCompletion: CGFloat {
        if shell.rootNavigationTransition?.drivenByDrawer == true {
            return 1 - drawerOpenProgress
        }
        if shell.rootNavigationTransition != nil {
            return paneTransitionProgress
        }
        return 1
    }

    private var outgoingDestination: RootDestination? {
        shell.rootNavigationTransition?.outgoing
    }

    /// Primary panes stay alive (state preserved); lighter secondary destinations
    /// render lazily. Selection is driven by the global drawer via `rootDestination`.
    private var tabContent: some View {
        GeometryReader { geo in
            let width = geo.size.width
            ZStack {
                iaPane(.dashboard, width: width) {
                    if shell.activeRoleContext == .parent {
                        ParentDashboardView(
                            initialStudentId: shell.pendingParentStudentId,
                            initialRoute: shell.pendingParentRoute
                        )
                    } else {
                        DashboardView()
                    }
                }
                iaPane(.courses, width: width) { CoursesListView() }
                iaPane(.notebooks, width: width) { NotebooksListView() }
                iaPane(.inbox, width: width) { InboxView() }
                profilePane(width: width)
                iaPane(.teach, width: width) { TeachHubView() }
                iaPane(.children, width: width) {
                    ParentDashboardView(
                        initialStudentId: shell.pendingParentStudentId,
                        initialRoute: shell.pendingParentRoute
                    )
                }

                secondaryPane(.calendar, width: width) {
                    NavigationStack { PlannerView(initialTab: .calendar).globalDrawerToolbar() }
                }
                secondaryPane(.todos, width: width) {
                    NavigationStack { PlannerView(initialTab: .todos).globalDrawerToolbar() }
                }
                secondaryPane(.review, width: width) {
                    NavigationStack { ReviewHomeView().globalDrawerToolbar() }
                }
                secondaryPane(.insights, width: width) {
                    NavigationStack {
                        InsightsView(
                            onOpenCourse: { course in
                                shell.activeCourse = course
                                shell.activeCourseRoot = .dashboard
                                shell.activeCourseSection = .modules
                                shell.select(.courses)
                            },
                            onOpenReview: { shell.select(.review) }
                        )
                        .globalDrawerToolbar()
                    }
                }
                secondaryPane(.globalNotebook, width: width) {
                    NavigationStack {
                        NotebookPagesView(
                            courseCode: NotebookStore.globalKey,
                            title: NotebookStore.globalTitle
                        )
                        .globalDrawerToolbar()
                    }
                }
                secondaryPane(.accommodations, width: width) {
                    NavigationStack { MyAccommodationsView().globalDrawerToolbar() }
                }
            }
        }
    }

    /// Profile also backs the "Settings" destination (mobile Profile is the account hub).
    private func profilePane(width: CGFloat) -> some View {
        let active = shell.rootDestination == .profile || shell.rootDestination == .settings
        let outgoing = outgoingDestination == .profile || outgoingDestination == .settings
        let visible = active || outgoing
        return ProfileView()
            .offset(x: paneOffset(for: .profile, width: width, active: active, outgoing: outgoing))
            .opacity(visible ? 1 : 0)
            .allowsHitTesting(active && navigationCompletion >= 0.99)
            .accessibilityHidden(!active)
    }

    private func iaPane<Content: View>(
        _ dest: RootDestination,
        width: CGFloat,
        @ViewBuilder content: () -> Content
    ) -> some View {
        let active = shell.rootDestination == dest
        let outgoing = outgoingDestination == dest
        let visible = active || outgoing
        return content()
            .offset(x: paneOffset(for: dest, width: width, active: active, outgoing: outgoing))
            .opacity(visible ? 1 : 0)
            .allowsHitTesting(active && navigationCompletion >= 0.99)
            .accessibilityHidden(!active)
    }

    @ViewBuilder
    private func secondaryPane<Content: View>(
        _ dest: RootDestination,
        width: CGFloat,
        @ViewBuilder content: () -> Content
    ) -> some View {
        let active = shell.rootDestination == dest
        let outgoing = outgoingDestination == dest
        if active || outgoing {
            content()
                .offset(x: paneOffset(for: dest, width: width, active: active, outgoing: outgoing))
                .allowsHitTesting(active && navigationCompletion >= 0.99)
                .accessibilityHidden(!active)
        }
    }

    private func paneOffset(
        for dest: RootDestination,
        width: CGFloat,
        active: Bool,
        outgoing: Bool
    ) -> CGFloat {
        guard shell.rootNavigationTransition != nil else { return 0 }
        let completion = navigationCompletion
        if active { return width * (1 - completion) }
        if outgoing { return -width * completion }
        return 0
    }
}

// MARK: - Floating pill tab bar

/// Deep-teal floating capsule: selected tab gets a cream circular "puck".
struct IaShellTabBar: View {
    @Environment(\.colorScheme) private var colorScheme
    @Bindable var shell: AppShellModel

    var body: some View {
        HStack(spacing: 0) {
            ForEach(shell.shellTabs, id: \.self) { tab in
                Button {
                    withAnimation(.spring(duration: 0.3)) {
                        shell.selectShellTab(tab)
                    }
                } label: {
                    shellTabIcon(tab)
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.plain)
                .accessibilityLabel(tab.label)
                .accessibilityAddTraits(shell.selectedShellTab == tab ? [.isSelected] : [])
            }
        }
        .padding(.vertical, 9)
        .padding(.horizontal, 10)
        .background(
            Capsule(style: .continuous)
                .fill(colorScheme == .dark ? LexturesTheme.cardBackgroundDark : LexturesTheme.primaryDeep)
        )
        .overlay(
            Capsule(style: .continuous)
                .stroke(
                    colorScheme == .dark ? LexturesTheme.fieldBorderDark : .white.opacity(0.08),
                    lineWidth: 1
                )
        )
        .shadow(color: LexturesTheme.primaryDeep.opacity(colorScheme == .dark ? 0 : 0.35), radius: 16, y: 8)
    }

    @ViewBuilder
    private func shellTabIcon(_ tab: ShellTab) -> some View {
        let selected = shell.selectedShellTab == tab
        ZStack(alignment: .topTrailing) {
            Image(systemName: tab.systemImage)
                .font(.system(size: 17, weight: .semibold))
                .foregroundStyle(
                    selected
                        ? LexturesTheme.primaryDeep
                        : (colorScheme == .dark ? LexturesTheme.textSecondaryDark : .white.opacity(0.72))
                )
                .frame(width: 44, height: 44)
                .background(
                    Circle().fill(selected ? AnyShapeStyle(LexturesTheme.brandCream) : AnyShapeStyle(.clear))
                )

            if tab == .inbox && shell.unreadInbox > 0 {
                Text(shell.unreadInbox > 99 ? "99+" : "\(shell.unreadInbox)")
                    .font(.system(size: 10, weight: .bold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 5)
                    .padding(.vertical, 2)
                    .background(LexturesTheme.coral)
                    .clipShape(Capsule())
                    .offset(x: 4, y: -2)
            }
        }
    }
}

struct LexturesTabBar: View {
    @Environment(\.colorScheme) private var colorScheme
    @Bindable var shell: AppShellModel

    var body: some View {
        HStack(spacing: 0) {
            ForEach(AppTab.allCases) { tab in
                Button {
                    withAnimation(.spring(duration: 0.3)) {
                        shell.selectLegacyTab(tab)
                    }
                } label: {
                    tabIcon(tab)
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.plain)
                .accessibilityLabel(tab.label)
                .accessibilityAddTraits(shell.selectedTab == tab ? [.isSelected] : [])
            }
        }
        .padding(.vertical, 9)
        .padding(.horizontal, 10)
        .background(
            Capsule(style: .continuous)
                .fill(colorScheme == .dark ? LexturesTheme.cardBackgroundDark : LexturesTheme.primaryDeep)
        )
        .overlay(
            Capsule(style: .continuous)
                .stroke(
                    colorScheme == .dark ? LexturesTheme.fieldBorderDark : .white.opacity(0.08),
                    lineWidth: 1
                )
        )
        .shadow(color: LexturesTheme.primaryDeep.opacity(colorScheme == .dark ? 0 : 0.35), radius: 16, y: 8)
    }

    @ViewBuilder
    private func tabIcon(_ tab: AppTab) -> some View {
        let selected = shell.selectedTab == tab
        ZStack(alignment: .topTrailing) {
            Image(systemName: tab.systemImage)
                .font(.system(size: 17, weight: .semibold))
                .foregroundStyle(
                    selected
                        ? LexturesTheme.primaryDeep
                        : (colorScheme == .dark ? LexturesTheme.textSecondaryDark : .white.opacity(0.72))
                )
                .frame(width: 44, height: 44)
                .background(
                    Circle().fill(selected ? AnyShapeStyle(LexturesTheme.brandCream) : AnyShapeStyle(.clear))
                )

            if tab == .inbox && shell.unreadInbox > 0 {
                Text(shell.unreadInbox > 99 ? "99+" : "\(shell.unreadInbox)")
                    .font(.system(size: 10, weight: .bold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 5)
                    .padding(.vertical, 2)
                    .background(LexturesTheme.coral)
                    .clipShape(Capsule())
                    .offset(x: 4, y: -2)
            }
        }
    }
}

// MARK: - Shared LMS UI helpers

/// Floating card: generous radius, soft warm shadow, hairline border in dark mode only.
struct LMSCard<Content: View>: View {
    @Environment(\.colorScheme) private var colorScheme
    var accent: Color? = nil
    @ViewBuilder var content: () -> Content

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            content()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(16)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(alignment: .leading) {
            if let accent {
                UnevenRoundedRectangle(
                    topLeadingRadius: 18,
                    bottomLeadingRadius: 18,
                    bottomTrailingRadius: 0,
                    topTrailingRadius: 0,
                    style: .continuous
                )
                .fill(accent)
                .frame(width: 4)
            }
        }
        .overlay(
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .stroke(
                    LexturesTheme.fieldBorder(for: colorScheme).opacity(colorScheme == .dark ? 0.9 : 0.45),
                    lineWidth: 1
                )
        )
        .shadow(color: LexturesTheme.cardShadow(for: colorScheme), radius: 12, y: 5)
    }
}

/// Serif section header — editorial, like a textbook chapter heading.
struct LMSSectionHeader: View {
    @Environment(\.colorScheme) private var colorScheme
    let title: String
    var systemImage: String?

    var body: some View {
        HStack(spacing: 8) {
            if let systemImage {
                Image(systemName: systemImage)
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 26, height: 26)
                    .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.18 : 0.16))
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
            }
            Text(title)
                .font(LexturesTheme.displayFont(19))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
        .padding(.top, 6)
    }
}

struct LMSErrorBanner: View {
    let message: String

    var body: some View {
        Label(message, systemImage: "exclamationmark.triangle.fill")
            .font(.subheadline)
            .foregroundStyle(LexturesTheme.error)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(14)
            .background(LexturesTheme.error.opacity(0.09))
            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
    }
}

struct LMSEmptyState: View {
    @Environment(\.colorScheme) private var colorScheme
    let systemImage: String
    let title: String
    let message: String

    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: systemImage)
                .font(.system(size: 28))
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                .frame(width: 72, height: 72)
                .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.16 : 0.14))
                .clipShape(Circle())
            Text(title)
                .font(LexturesTheme.displayFont(18))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(message)
                .font(.subheadline)
                .multilineTextAlignment(.center)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 36)
        .padding(.horizontal, 24)
    }
}

/// Rounded gradient tile used as a course "cover" thumbnail.
struct LMSCoverTile: View {
    let key: String
    let systemImage: String
    var size: CGFloat = 48

    var body: some View {
        RoundedRectangle(cornerRadius: size * 0.28, style: .continuous)
            .fill(LexturesTheme.coverGradient(for: key))
            .frame(width: size, height: size)
            .overlay(
                Image(systemName: systemImage)
                    .font(.system(size: size * 0.4, weight: .medium))
                    .foregroundStyle(.white)
            )
    }
}

/// Horizontally scrolling pill selector — solid deep teal when selected,
/// white card with hairline border otherwise (inspiration: segmented chips).
struct LMSSegmentedChips<Option: Hashable>: View {
    @Environment(\.colorScheme) private var colorScheme
    let options: [Option]
    @Binding var selection: Option
    let label: (Option) -> String

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(options, id: \.self) { option in
                    let selected = option == selection
                    Button {
                        withAnimation(.easeOut(duration: 0.15)) { selection = option }
                    } label: {
                        Text(label(option))
                            .font(.subheadline.weight(selected ? .semibold : .regular))
                            .padding(.horizontal, 15)
                            .padding(.vertical, 8)
                            .background(
                                selected
                                    ? AnyShapeStyle(LexturesTheme.accent(for: colorScheme))
                                    : AnyShapeStyle(LexturesTheme.cardBackground(for: colorScheme))
                            )
                            .foregroundStyle(
                                selected
                                    ? (colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
                                    : LexturesTheme.textSecondary(for: colorScheme)
                            )
                            .clipShape(Capsule())
                            .overlay(
                                Capsule().stroke(
                                    selected ? .clear : LexturesTheme.fieldBorder(for: colorScheme),
                                    lineWidth: 1
                                )
                            )
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.vertical, 2)
        }
    }
}

/// Initials avatar chip; tapping jumps to the Profile tab.
struct LMSAvatarButton: View {
    @Environment(AppShellModel.self) private var shell
    var size: CGFloat = 34

    var body: some View {
        Button {
            shell.select(.profile)
        } label: {
            ProfileAvatarView(
                avatarUrl: shell.accountProfile?.avatarUrl,
                initials: shell.accountProfile?.resolvedInitials ?? shell.profile?.initials ?? "··",
                size: size,
                initialsBackground: .white.opacity(0.16),
                initialsForeground: .white
            )
            .clipShape(Circle())
            .overlay(
                Circle()
                    .stroke(.white.opacity(0.35), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
        .accessibilityLabel(L.text("tabs.profile"))
    }
}

/// Card-shaped redacted placeholder shown while a list loads.
struct LMSSkeletonCard: View {
    @Environment(\.colorScheme) private var colorScheme
    var lines: Int = 2

    var body: some View {
        LMSCard {
            HStack(spacing: 12) {
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .fill(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.6))
                    .frame(width: 44, height: 44)
                VStack(alignment: .leading, spacing: 7) {
                    ForEach(0 ..< lines, id: \.self) { index in
                        RoundedRectangle(cornerRadius: 4)
                            .fill(LexturesTheme.fieldBorder(for: colorScheme).opacity(index == 0 ? 0.7 : 0.45))
                            .frame(width: index == 0 ? 180 : 120, height: 11)
                    }
                }
                Spacer(minLength: 0)
            }
        }
        .redacted(reason: .placeholder)
        .accessibilityHidden(true)
    }
}

/// Stack of skeleton cards for list screens.
struct LMSSkeletonList: View {
    var count: Int = 4

    var body: some View {
        VStack(spacing: 12) {
            ForEach(0 ..< count, id: \.self) { _ in
                LMSSkeletonCard()
            }
        }
    }
}

/// Small circular progress ring (course completion) — echoes the health-app ring.
struct LMSProgressRing: View {
    @Environment(\.colorScheme) private var colorScheme
    let progress: Double // 0...1
    var size: CGFloat = 38
    var tint: Color?

    var body: some View {
        ZStack {
            Circle()
                .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.8), lineWidth: 4)
            Circle()
                .trim(from: 0, to: max(0.02, min(1, progress)))
                .stroke(
                    tint ?? LexturesTheme.accent(for: colorScheme),
                    style: StrokeStyle(lineWidth: 4, lineCap: .round)
                )
                .rotationEffect(.degrees(-90))
            Text("\(Int((progress * 100).rounded()))")
                .font(.system(size: size * 0.3, weight: .bold, design: .rounded))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
        .frame(width: size, height: size)
        .accessibilityLabel(L.format("mobile.profile.percentComplete", Int((progress * 100).rounded())))
    }
}
