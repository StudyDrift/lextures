import Foundation

// MARK: - Role & shell

/// Effective navigation persona derived from server-authoritative signals.
enum MobileRoleKind: String, CaseIterable, Equatable {
    case student
    case instructor
    case parent
    case selfLearner
}

/// Persisted active context for multi-role users (Learning vs Teaching vs Parent).
enum MobileRoleContext: String, CaseIterable, Equatable {
    case learning
    case teaching
    case parent

    var label: String {
        switch self {
        case .learning: return L.text("mobile.ia.context.learning")
        case .teaching: return L.text("mobile.ia.context.teaching")
        case .parent: return L.text("mobile.ia.context.parent")
        }
    }
}

/// Primary bottom-bar destinations (≤5 per shell).
enum ShellTab: String, CaseIterable, Equatable, Identifiable {
    case home
    case courses
    case notebooks
    case inbox
    case profile
    case teach
    case children
    case calendar

    var id: String { rawValue }

    var label: String {
        switch self {
        case .home: return L.text("tabs.home")
        case .courses: return L.text("tabs.courses")
        case .notebooks: return L.text("tabs.notebooks")
        case .inbox: return L.text("tabs.inbox")
        case .profile: return L.text("tabs.profile")
        case .teach: return L.text("mobile.ia.tabs.teach")
        case .children: return L.text("mobile.ia.tabs.children")
        case .calendar: return L.text("mobile.ia.tabs.calendar")
        }
    }

    var systemImage: String {
        switch self {
        case .home: return "house.fill"
        case .courses: return "books.vertical.fill"
        case .notebooks: return "square.and.pencil"
        case .inbox: return "tray.fill"
        case .profile: return "person.fill"
        case .teach: return "checkmark.circle.fill"
        case .children: return "figure.2.and.child.holdinghands"
        case .calendar: return "calendar"
        }
    }
}

// MARK: - Drawer navigation (web-parity sidebar)

/// Two-level left-drawer state machine shared by the shell.
/// `.none` = no drawer; `.course` = course-scoped menu; `.global` = app-wide menu.
enum DrawerState: Equatable {
    case none
    case course
    case global
}

/// Top-level app destinations reachable from the global drawer.
/// Replaces the former bottom-bar `ShellTab` selection.
enum RootDestination: String, CaseIterable, Equatable, Identifiable {
    case dashboard
    case courses
    case calendar
    case todos
    case review
    case insights
    case notebooks
    case globalNotebook
    case accommodations
    case inbox
    case settings
    case profile
    case teach
    case children

    var id: String { rawValue }

    var label: String {
        switch self {
        case .dashboard: return L.text("mobile.drawer.dashboard")
        case .courses: return L.text("tabs.courses")
        case .calendar: return L.text("mobile.ia.tabs.calendar")
        case .todos: return L.text("mobile.drawer.todos")
        case .review: return L.text("mobile.drawer.review")
        case .insights: return L.text("mobile.drawer.insights")
        case .notebooks: return L.text("mobile.drawer.notebooks")
        case .globalNotebook: return L.text("mobile.drawer.globalNotebook")
        case .accommodations: return L.text("mobile.drawer.accommodations")
        case .inbox: return L.text("tabs.inbox")
        case .settings: return L.text("mobile.ia.more.settings")
        case .profile: return L.text("tabs.profile")
        case .teach: return L.text("mobile.ia.tabs.teach")
        case .children: return L.text("mobile.ia.tabs.children")
        }
    }

    var systemImage: String {
        switch self {
        case .dashboard: return "square.grid.2x2.fill"
        case .courses: return "books.vertical.fill"
        case .calendar: return "calendar"
        case .todos: return "checklist"
        case .review: return "arrow.triangle.2.circlepath"
        case .insights: return "chart.line.uptrend.xyaxis"
        case .notebooks: return "book.closed.fill"
        case .globalNotebook: return "globe"
        case .accommodations: return "accessibility"
        case .inbox: return "tray.fill"
        case .settings: return "gearshape.fill"
        case .profile: return "person.fill"
        case .teach: return "checkmark.circle.fill"
        case .children: return "figure.2.and.child.holdinghands"
        }
    }

    /// Whether the inbox unread badge should render on this row.
    var showsInboxBadge: Bool { self == .inbox }
}

/// A titled section of the global drawer. `titleKey == nil` renders header-less
/// (the top block of primary destinations, mirroring the web sidebar).
struct DrawerGroup: Identifiable, Equatable {
    let titleKey: String?
    let items: [RootDestination]

    var id: String { titleKey ?? "primary" }
    var title: String? { titleKey.map { L.text(String.LocalizationValue($0)) } }
}

/// A titled section of the course drawer, grouping existing workspace sections.
struct CourseDrawerGroup: Identifiable, Equatable {
    let titleKey: String
    let sections: [CourseWorkspaceSection]

    var id: String { titleKey }
    var title: String { L.text(String.LocalizationValue(titleKey)) }
}

/// Secondary destinations surfaced from Profile / More hub.
enum MoreDestination: String, CaseIterable, Equatable, Identifiable {
    case calendar
    case planner
    case catalog
    case paths
    case library
    case reading
    case portfolio
    case credentials
    case gamification
    case advising
    case settings
    case askAi
    case peerReviews
    case reportCards
    case insights

    var id: String { rawValue }

    var label: String {
        switch self {
        case .askAi: return L.text("mobile.tutor.askAi")
        case .peerReviews: return L.text("mobile.peerReview.title")
        case .reportCards: return L.text("mobile.mastery.reportCards")
        case .insights: return L.text("mobile.ia.more.insights")
        case .calendar: return L.text("mobile.ia.more.calendar")
        case .planner: return L.text("mobile.ia.more.planner")
        case .catalog: return L.text("mobile.ia.more.catalog")
        case .paths: return L.text("mobile.ia.more.paths")
        case .library: return L.text("mobile.ia.more.library")
        case .reading: return L.text("mobile.ia.more.reading")
        case .portfolio: return L.text("mobile.ia.more.portfolio")
        case .credentials: return L.text("mobile.ia.more.credentials")
        case .gamification: return L.text("mobile.ia.more.gamification")
        case .advising: return L.text("mobile.ia.more.advising")
        case .settings: return L.text("mobile.ia.more.settings")
        }
    }

    var systemImage: String {
        switch self {
        case .askAi: return "sparkles"
        case .peerReviews: return "person.2.wave.2.fill"
        case .reportCards: return "doc.text.fill"
        case .insights: return "chart.line.uptrend.xyaxis"
        case .calendar: return "calendar"
        case .planner: return "list.bullet.rectangle"
        case .catalog: return "books.vertical"
        case .paths: return "point.topleft.down.to.point.bottomright.curvepath"
        case .library: return "books.vertical.fill"
        case .reading: return "book.fill"
        case .portfolio: return "folder.fill"
        case .credentials: return "rosette"
        case .gamification: return "flame.fill"
        case .advising: return "person.2.fill"
        case .settings: return "gearshape.fill"
        }
    }
}

// MARK: - Course workspace

/// Course-scoped workspace chips (registry-driven; not hardcoded to four).
enum CourseWorkspaceSection: String, CaseIterable, Equatable, Hashable {
    case overview
    case modules
    case grades
    case mastery
    case discussions
    case feed
    case live
    case people
    case files
    case attendance
    case evaluations
    case library
    case officeHours
    case groups
    case collabDocs
    case grading

    var label: String {
        switch self {
        case .overview: return L.text("mobile.ia.course.overview")
        case .modules: return L.text("mobile.ia.course.modules")
        case .grades: return L.text("mobile.ia.course.grades")
        case .mastery: return L.text("mobile.ia.course.mastery")
        case .discussions: return L.text("mobile.ia.course.discussions")
        case .feed: return L.text("mobile.ia.course.feed")
        case .live: return L.text("mobile.ia.course.live")
        case .people: return L.text("mobile.ia.course.people")
        case .files: return L.text("mobile.ia.course.files")
        case .attendance: return L.text("mobile.ia.course.attendance")
        case .evaluations: return L.text("mobile.ia.course.evaluations")
        case .library: return L.text("mobile.ia.course.library")
        case .officeHours: return L.text("mobile.ia.course.officeHours")
        case .groups: return L.text("mobile.ia.course.groups")
        case .collabDocs: return L.text("mobile.ia.course.collabDocs")
        case .grading: return L.text("mobile.ia.course.grading")
        }
    }

    /// Deep-link segment used by `DeepLinkRouter` / push payloads.
    var deepLinkSegment: String? {
        switch self {
        case .overview: return "overview"
        case .modules: return "modules"
        case .grades: return "grades"
        case .mastery: return "mastery"
        case .discussions: return "discussions"
        case .feed: return "feed"
        case .live: return "live"
        case .people: return "people"
        case .files: return "files"
        case .attendance: return "attendance"
        case .evaluations: return "evaluations"
        case .library: return "library"
        case .officeHours: return "office-hours"
        case .groups: return "groups"
        case .collabDocs: return "collab-docs"
        case .grading: return "grading"
        }
    }

    static func from(deepLink section: CourseDeepLinkSection) -> CourseWorkspaceSection? {
        switch section {
        case .overview: return .overview
        case .modules: return .modules
        case .grades: return .grades
        case .feed: return .feed
        case .discussions: return .discussions
        case .officeHours: return .officeHours
        case .live: return .live
        case .files: return .files
        case .attendance: return .attendance
        case .people: return .people
        case .evaluations: return .evaluations
        case .library: return .library
        case .groups: return .groups
        case .collabDocs: return .collabDocs
        }
    }
}

struct RoleSnapshot: Equatable {
    var hasStudentEnrollment = false
    var hasStaffEnrollment = false
    var hasParentDashboard = false
    var hasSelfPacedEnrollment = false

    var availableContexts: [MobileRoleContext] {
        var out: [MobileRoleContext] = []
        if hasParentDashboard { out.append(.parent) }
        if hasStaffEnrollment { out.append(.teaching) }
        if hasStudentEnrollment || hasSelfPacedEnrollment { out.append(.learning) }
        if out.isEmpty { out.append(.learning) }
        return out
    }

    func defaultContext() -> MobileRoleContext {
        availableContexts.first ?? .learning
    }

    func resolvedContext(stored: MobileRoleContext?) -> MobileRoleContext {
        guard let stored, availableContexts.contains(stored) else {
            return defaultContext()
        }
        return stored
    }
}

struct MobilePlatformFeatures: Equatable {
    var ffLibrary = false
    var ffCourseEvaluations = false
    var ffMobileCourseEvaluations = true
    var ffMobileIaRedesign = false
    var ffMobileVibeActivities = true
    var ffMobileUniversalSearch = false
    var ffMobileProfileDepth = false
    var ffMobileLibraryEreserves = true
    var ffMobileImmersiveReader = true
    var ffMobileLiveMeetings = true
    var readAloudEnabled = false
    var ffReadAloud = false
    var videoCaptionsEnabled = false
    var autoCaptioningEnabled = false
    var translationMemoryEnabled = false
    var ffReadingPreferences = false
    var oerLibraryEnabled = false
    var customFieldsEnabled = false
    var ffDemographics = false
    var ffResearchConsent = false
    var ffPersistentTutor = false
    var ffAiStudyBuddy = false
    var ragNotebookEnabled = false
    var aiStudyBuddyEnabled = false
    var aiDisclosureEnabled = false
    var ffPeerReview = false
    var ffLearningPaths = false
    var selfReflectionEnabled = false
    var ffPublicCatalog = false
    var ffSelfPacedMode = false
    var ffCourseReviews = false
    var ffCompletionCredentials = false
    var ffGamification = false
    var ffStripeBilling = false
    var ffPaymentsEnabled = false
    var ffTaxCollection = false
    var ffAdvisingIntegration = false
    var ffMobileAdvising = true

    static func from(_ features: PlatformFeatures?) -> MobilePlatformFeatures {
        MobilePlatformFeatures(
            ffLibrary: features?.ffLibrary == true,
            ffCourseEvaluations: features?.ffCourseEvaluations == true,
            ffMobileCourseEvaluations: features?.ffMobileCourseEvaluations != false,
            ffMobileIaRedesign: features?.ffMobileIaRedesign == true,
            ffMobileVibeActivities: features?.ffMobileVibeActivities != false,
            ffMobileUniversalSearch: features?.ffMobileUniversalSearch == true,
            ffMobileProfileDepth: features?.ffMobileProfileDepth == true,
            ffMobileLibraryEreserves: features?.ffMobileLibraryEreserves != false,
            ffMobileImmersiveReader: features?.ffMobileImmersiveReader != false,
            ffMobileLiveMeetings: features?.ffMobileLiveMeetings != false,
            readAloudEnabled: features?.readAloudEnabled == true,
            ffReadAloud: features?.ffReadAloud == true,
            videoCaptionsEnabled: features?.videoCaptionsEnabled == true || features?.autoCaptioningEnabled == true,
            autoCaptioningEnabled: features?.autoCaptioningEnabled == true,
            translationMemoryEnabled: features?.translationMemoryEnabled == true,
            ffReadingPreferences: features?.ffReadingPreferences == true,
            oerLibraryEnabled: features?.oerLibraryEnabled == true,
            customFieldsEnabled: features?.customFieldsEnabled == true,
            ffDemographics: features?.ffDemographics == true,
            ffResearchConsent: features?.ffResearchConsent == true,
            ffPersistentTutor: features?.ffPersistentTutor == true,
            ffAiStudyBuddy: features?.ffAiStudyBuddy == true,
            ragNotebookEnabled: features?.ragNotebookEnabled == true,
            aiStudyBuddyEnabled: features?.aiStudyBuddyEnabled == true,
            aiDisclosureEnabled: features?.aiDisclosureEnabled == true,
            ffPeerReview: features?.ffPeerReview == true,
            ffLearningPaths: features?.ffLearningPaths == true,
            selfReflectionEnabled: features?.selfReflectionEnabled == true,
            ffPublicCatalog: features?.ffPublicCatalog == true,
            ffSelfPacedMode: features?.ffSelfPacedMode == true,
            ffCourseReviews: features?.ffCourseReviews == true,
            ffCompletionCredentials: features?.ffCompletionCredentials == true,
            ffGamification: features?.ffGamification == true,
            ffStripeBilling: features?.ffStripeBilling == true,
            ffPaymentsEnabled: features?.ffPaymentsEnabled == true,
            ffTaxCollection: features?.ffTaxCollection == true,
            ffAdvisingIntegration: features?.ffAdvisingIntegration == true,
            ffMobileAdvising: features?.ffMobileAdvising != false
        )
    }

    var libraryBrowseEnabled: Bool {
        ffMobileLibraryEreserves && (ffLibrary || oerLibraryEnabled)
    }

    var immersiveReader: ImmersiveReaderCapabilities {
        guard ffMobileImmersiveReader else {
            return ImmersiveReaderCapabilities(toolbarEnabled: false)
        }
        return ImmersiveReaderCapabilities(
            toolbarEnabled: true,
            readAloudEnabled: readAloudEnabled && ffReadAloud,
            translationEnabled: translationMemoryEnabled,
            captionsEnabled: videoCaptionsEnabled,
            preferencesEnabled: ffReadingPreferences || (readAloudEnabled && ffReadAloud)
        )
    }
}

struct CourseWorkspaceContext: Equatable {
    var course: CourseSummary
    var hasAttendanceSessions = false
    var hasLibraryResources = false
    var evaluationStatus: EvaluationStatus?
    var platformFeatures = MobilePlatformFeatures()
}

/// Registry: role-aware shell tabs, More hub, and course workspace chips.
enum MobileDestinations {
    static let maxPrimaryChips = 6
    static let parentDashboardPermission = "app:user:account-parent-dashboard"

    // MARK: Shell

    static func shellTabs(context: MobileRoleContext) -> [ShellTab] {
        switch context {
        case .teaching:
            return [.home, .courses, .teach, .inbox, .profile]
        case .parent:
            return [.home, .children, .calendar, .inbox, .profile]
        case .learning:
            return [.home, .courses, .notebooks, .inbox, .profile]
        }
    }

    // MARK: Global drawer

    /// Role-aware grouped destinations for the global drawer, mirroring the web sidebar.
    static func globalDrawerGroups(
        context: MobileRoleContext,
        platform: MobilePlatformFeatures
    ) -> [DrawerGroup] {
        switch context {
        case .learning:
            return [
                DrawerGroup(titleKey: nil, items: [.dashboard, .courses, .calendar, .todos]),
                DrawerGroup(
                    titleKey: "mobile.drawer.group.learning",
                    items: learningDrawerItems(platform: platform)
                ),
                DrawerGroup(titleKey: "mobile.drawer.group.notes", items: [.notebooks, .globalNotebook]),
                DrawerGroup(titleKey: "mobile.drawer.group.administration", items: [.accommodations]),
                DrawerGroup(titleKey: "mobile.drawer.group.account", items: [.inbox, .settings]),
            ]
        case .teaching:
            return [
                DrawerGroup(titleKey: nil, items: [.dashboard, .courses, .calendar]),
                DrawerGroup(titleKey: "mobile.drawer.group.teaching", items: [.teach]),
                DrawerGroup(titleKey: "mobile.drawer.group.notes", items: [.notebooks]),
                DrawerGroup(titleKey: "mobile.drawer.group.account", items: [.inbox, .settings]),
            ]
        case .parent:
            return [
                DrawerGroup(titleKey: nil, items: [.dashboard, .children, .calendar]),
                DrawerGroup(titleKey: "mobile.drawer.group.account", items: [.inbox, .settings]),
            ]
        }
    }

    // MARK: Course drawer

    /// Regroups the existing course workspace sections under web-style headers.
    /// Only sections already available for the viewer (from `courseWorkspaceSections`)
    /// appear, so per-role gating is inherited unchanged.
    static func courseDrawerGroups(_ sections: [CourseWorkspaceSection]) -> [CourseDrawerGroup] {
        let content: [CourseWorkspaceSection] = [.overview, .modules, .files, .library]
        let collaboration: [CourseWorkspaceSection] = [
            .discussions, .feed, .groups, .collabDocs, .live, .officeHours,
        ]
        let grades: [CourseWorkspaceSection] = [.grades, .mastery]
        let people: [CourseWorkspaceSection] = [.people]
        let manage: [CourseWorkspaceSection] = [.grading, .attendance, .evaluations]

        func filtered(_ group: [CourseWorkspaceSection]) -> [CourseWorkspaceSection] {
            group.filter(sections.contains)
        }

        let groups: [(String, [CourseWorkspaceSection])] = [
            ("mobile.drawer.course.content", filtered(content)),
            ("mobile.drawer.course.collaboration", filtered(collaboration)),
            ("mobile.drawer.course.grades", filtered(grades)),
            ("mobile.drawer.course.people", filtered(people)),
            ("mobile.drawer.course.manage", filtered(manage)),
        ]
        return groups.compactMap { key, list in
            list.isEmpty ? nil : CourseDrawerGroup(titleKey: key, sections: list)
        }
    }

    private static func learningDrawerItems(platform: MobilePlatformFeatures) -> [RootDestination] {
        var items: [RootDestination] = [.review]
        if platform.selfReflectionEnabled { items.append(.insights) }
        return items
    }

    static func legacyTab(from shell: ShellTab) -> AppTab? {
        switch shell {
        case .home: return .home
        case .courses: return .courses
        case .notebooks: return .notebooks
        case .inbox: return .inbox
        case .profile: return .profile
        case .teach, .children, .calendar: return nil
        }
    }

    static func moreDestinations(
        context: MobileRoleContext,
        platform: MobilePlatformFeatures
    ) -> [MoreDestination] {
        var out: [MoreDestination] = []
        switch context {
        case .learning:
            if TutorLogic.askAiEnabled(platform: platform) { out.append(.askAi) }
            if platform.ffPeerReview { out.append(.peerReviews) }
            out.append(.reportCards)
            if platform.selfReflectionEnabled { out.append(.insights) }
            out += [.calendar, .planner, .catalog, .paths]
            if platform.ffLibrary { out.append(.reading) }
            if platform.libraryBrowseEnabled { out.append(.library) }
            if platform.ffCompletionCredentials { out.append(.credentials) }
            if platform.ffGamification { out.append(.gamification) }
            out.append(.portfolio)
            if AdvisingLogic.advisingEnabled(platform) { out.append(.advising) }
            out.append(.settings)
        case .teaching:
            out += [.calendar, .planner]
            if platform.libraryBrowseEnabled { out.append(.library) }
            if AdvisingLogic.advisingEnabled(platform) { out.append(.advising) }
            out.append(.settings)
        case .parent:
            out += [.calendar]
            if AdvisingLogic.advisingEnabled(platform) { out.append(.advising) }
            out.append(.settings)
        }
        return out
    }

    // MARK: Course workspace

    static func courseWorkspaceSections(_ ctx: CourseWorkspaceContext) -> [CourseWorkspaceSection] {
        let course = ctx.course
        var out: [CourseWorkspaceSection] = [.overview, .modules]

        if course.isFilesEnabled { out.append(.files) }
        if course.viewerIsStudent { out.append(.grades) }
        if course.viewerIsStudent && course.isMasteryEnabled { out.append(.mastery) }
        if course.isDiscussionsEnabled { out.append(.discussions) }
        if course.isFeedEnabled { out.append(.feed) }
        if course.isLiveSessionsEnabled { out.append(.live) }
        if course.viewerIsStaff && course.isSectionsEnabled { out.append(.people) }
        if course.isOfficeHoursEnabled { out.append(.officeHours) }
        if course.isGroupSpacesEnabled { out.append(.groups) }
        if course.isCollabDocsEnabled { out.append(.collabDocs) }
        if course.isAttendanceEnabled && (course.viewerIsStaff || ctx.hasAttendanceSessions) {
            out.append(.attendance)
        }
        if EvaluationLogic.shouldShowWorkspaceSection(
            course: course,
            status: ctx.evaluationStatus,
            features: ctx.platformFeatures
        ) {
            out.append(.evaluations)
        }
        if course.viewerIsStaff {
            out.append(.grading)
        }
        if ctx.platformFeatures.ffMobileLibraryEreserves,
           ctx.platformFeatures.ffLibrary,
           ctx.hasLibraryResources {
            out.append(.library)
        }
        return out
    }

    static func splitCourseChips(_ sections: [CourseWorkspaceSection]) -> (visible: [CourseWorkspaceSection], overflow: [CourseWorkspaceSection]) {
        guard sections.count > maxPrimaryChips else {
            return (sections, [])
        }
        let visible = Array(sections.prefix(maxPrimaryChips))
        let overflow = Array(sections.dropFirst(maxPrimaryChips))
        return (visible, overflow)
    }

    // MARK: Role derivation

    static func buildRoleSnapshot(
        permissions: [String],
        courses: [CourseSummary],
        selfPacedEnrollmentCount: Int = 0
    ) -> RoleSnapshot {
        RoleSnapshot(
            hasStudentEnrollment: courses.contains(where: \.viewerIsStudent),
            hasStaffEnrollment: courses.contains(where: \.viewerIsStaff),
            hasParentDashboard: permissions.contains(parentDashboardPermission),
            hasSelfPacedEnrollment: selfPacedEnrollmentCount > 0
        )
    }
}
