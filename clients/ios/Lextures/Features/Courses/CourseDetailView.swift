import SwiftUI

/// Course home: gradient hero + registry-driven workspace sections.
struct CourseDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary
    var initialSection: CourseWorkspaceSection?
    var initialItemId: String?

    @State private var items: [CourseStructureItem] = []
    @State private var progress: ModulesProgressSnapshot?
    @State private var cacheLabel: String?
    @State private var hasAttendanceSessions = false
    @State private var errorMessage: String?
    @State private var loading = false
    @State private var linkedItem: CourseStructureItem?
    @State private var lockedItem: CourseStructureItem?
    @State private var showCourseSearch = false
    @State private var takeAttendanceRoute: TakeAttendanceRoute?
    @State private var selectedAttendanceSession: AttendanceSession?
    @State private var structureSocket = CourseStructureSocket()

    init(
        course: CourseSummary,
        initialSection: CourseWorkspaceSection? = nil,
        initialItemId: String? = nil
    ) {
        self.course = course
        self.initialSection = initialSection
        self.initialItemId = initialItemId
    }

    /// Selected section is owned by the shell so the course drawer can drive it.
    private var section: CourseWorkspaceSection { shell.activeCourseSection }

    private var workspaceContext: CourseWorkspaceContext {
        CourseWorkspaceContext(
            course: course,
            hasAttendanceSessions: hasAttendanceSessions,
            hasLibraryResources: LibraryResourceLogic.hasLibraryResources(in: items),
            platformFeatures: shell.platformFeatures
        )
    }

    private var allSections: [CourseWorkspaceSection] {
        MobileDestinations.courseWorkspaceSections(workspaceContext)
    }

    private var moduleGroups: [ModuleGroup] {
        ModuleContentLogic.buildModuleGroups(from: items)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    CourseBanner(course: course)

                    Text(section.label)
                        .font(LexturesTheme.displayFont(20))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if let cacheLabel {
                        StalenessChip(label: cacheLabel)
                    }

                    sectionContent
                }
                .padding(16)
            }
            .refreshable { await load() }
        }
        .navigationTitle(course.displayTitle)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Button { shell.drawer = .course } label: {
                    Image(systemName: "line.3.horizontal")
                }
                .accessibilityLabel(L.text("mobile.drawer.courseMenu"))
            }
            if shell.universalSearchEnabled {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { showCourseSearch = true } label: {
                        Image(systemName: "magnifyingglass")
                    }
                    .accessibilityLabel(L.text("mobile.search.inThisCourse"))
                }
            }
        }
        .sheet(isPresented: $showCourseSearch) {
            UniversalSearchView(courseScope: course.courseCode)
        }
        .navigationDestination(for: CourseStructureItem.self) { item in
            ModuleItemRouteView(course: course, item: item, onProgressChanged: refreshProgress)
        }
        .navigationDestination(item: $selectedAttendanceSession) { attendanceSession in
            AttendanceSessionDetailView(course: course, attendanceSession: attendanceSession)
        }
        .navigationDestination(item: $takeAttendanceRoute) { route in
            TakeAttendanceView(course: course, initialSessionId: route.sessionId)
        }
        .navigationDestination(for: GradingBacklogItem.self) { backlogItem in
            SubmissionsListView(course: course, backlogItem: backlogItem)
        }
        .navigationDestination(item: $linkedItem) { item in
            ModuleItemRouteView(course: course, item: item, onProgressChanged: refreshProgress)
        }
        .sheet(item: $lockedItem) { item in
            RequirementsView(
                targetItem: item,
                groups: moduleGroups,
                progress: progress,
                onGoToRequired: { itemId in
                    if let match = RequirementsLogic.findItem(id: itemId, in: moduleGroups) {
                        linkedItem = match
                    }
                }
            )
        }
        .tutorLauncher(course: course)
        .task { await load() }
        .task {
            structureSocket.connect(courseCode: course.courseCode, accessToken: { session.accessToken })
        }
        .onDisappear {
            structureSocket.disconnect()
            // Leaving the course view (pop to list, or push a child): drop course context.
            shell.activeCourse = nil
        }
        .onChange(of: structureSocket.revision) { _, _ in
            Task { await load() }
        }
        .onChange(of: allSections) { _, sections in
            shell.activeCourseSections = sections
            if !sections.contains(shell.activeCourseSection), let first = sections.first {
                shell.activeCourseSection = first
            }
        }
        .onChange(of: items) { _, loaded in
            guard linkedItem == nil,
                  let itemId = initialItemId,
                  let match = loaded.first(where: { $0.id == itemId }) else { return }
            linkedItem = match
        }
        .onAppear {
            shell.activeCourse = course
            shell.activeCourseRoot = shell.rootDestination
            shell.activeCourseSections = allSections
            if let initialSection, allSections.contains(initialSection) {
                shell.activeCourseSection = initialSection
            } else if !allSections.contains(shell.activeCourseSection) {
                shell.activeCourseSection = allSections.first ?? .modules
            }
        }
    }

    @ViewBuilder
    private var sectionContent: some View {
        switch section {
        case .overview:
            CourseSyllabusSection(course: course)
        case .modules:
            modulesSection
        case .files:
            CourseFilesView(course: course)
        case .grades:
            CourseGradesSection(course: course)
        case .mastery:
            CourseMasterySection(course: course)
        case .officeHours:
            CourseOfficeHoursSection(course: course)
        case .attendance:
            CourseAttendanceSection(
                course: course,
                onTakeAttendance: course.viewerIsStaff ? { takeAttendanceRoute = TakeAttendanceRoute() } : nil,
                onOpenSession: openAttendanceSession
            )
        case .grading:
            GradingBacklogSection(course: course)
        case .library:
            CourseLibraryView(course: course, items: items, onSelectItem: { linkedItem = $0 })
        case .discussions:
            CourseDiscussionsSection(
                course: course,
                initialThreadId: section == .discussions ? initialItemId : nil
            )
        case .feed:
            CourseFeedSection(course: course)
        case .people:
            CoursePeopleSection(course: course)
        case .live, .evaluations:
            CourseDestinationPlaceholder(section: section)
        }
    }

    @ViewBuilder
    private var modulesSection: some View {
        if loading && items.isEmpty {
            LMSSkeletonList(count: 3)
        } else if moduleGroups.isEmpty {
            LMSEmptyState(
                systemImage: "square.stack.3d.up",
                title: L.text("mobile.modules.emptyCourse"),
                message: L.text("mobile.modules.emptyCourseHint")
            )
        } else {
            ModuleListView(
                course: course,
                groups: moduleGroups,
                progress: progress,
                onSelectItem: { linkedItem = $0 },
                onLockedItem: { item, _ in
                    lockedItem = item
                }
            )
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            async let sessionsTask = (try? LMSAPI.fetchAttendanceSessions(
                courseCode: course.courseCode,
                accessToken: token
            )) ?? []
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.courseStructure(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourseStructure(courseCode: course.courseCode, accessToken: token)
            }
            items = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
            hasAttendanceSessions = await !sessionsTask.isEmpty
            await refreshProgress()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load course content."
        }
    }

    private func refreshProgress() async {
        guard let token = session.accessToken, course.viewerIsStudent else { return }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.modulesProgress(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchModulesProgress(courseCode: course.courseCode, accessToken: token)
                    ?? ModulesProgressSnapshot()
            }
            progress = result.value.modules.isEmpty && result.value.enrollmentId.isEmpty ? nil : result.value
        } catch {
            progress = nil
        }
    }

    private func openAttendanceSession(_ attendanceSession: AttendanceSession) {
        if TakeAttendanceLogic.shouldTakeSession(attendanceSession, isStaff: course.viewerIsStaff) {
            takeAttendanceRoute = TakeAttendanceRoute(sessionId: attendanceSession.id)
        } else {
            selectedAttendanceSession = attendanceSession
        }
    }
}
