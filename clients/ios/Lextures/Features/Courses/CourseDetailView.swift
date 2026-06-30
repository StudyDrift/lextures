import SwiftUI

/// Course home: gradient hero + segmented sections
/// (Overview · Modules · Grades · Attendance · Grading by role).
struct CourseDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary
    var initialSection: Section?
    var initialItemId: String?

    enum Section: String, CaseIterable {
        case overview = "Overview"
        case modules = "Modules"
        case files = "Files"
        case grades = "Grades"
        case attendance = "Attendance"
        case grading = "Grading"
    }

    @State private var section: Section = .modules
    @State private var items: [CourseStructureItem] = []
    @State private var progress: ModulesProgressSnapshot?
    @State private var cacheLabel: String?
    @State private var hasAttendanceSessions = false
    @State private var errorMessage: String?
    @State private var loading = false
    @State private var linkedItem: CourseStructureItem?
    @State private var lockedItem: CourseStructureItem?

    init(course: CourseSummary, initialSection: Section? = nil, initialItemId: String? = nil) {
        self.course = course
        self.initialSection = initialSection
        self.initialItemId = initialItemId
        if let initialSection {
            _section = State(initialValue: initialSection)
        }
    }

    private var sections: [Section] {
        var out: [Section] = [.overview, .modules, .files]
        if course.viewerIsStudent { out.append(.grades) }
        if course.viewerIsStaff || hasAttendanceSessions { out.append(.attendance) }
        if course.viewerIsStaff { out.append(.grading) }
        return out
    }

    private var moduleGroups: [ModuleGroup] {
        ModuleContentLogic.buildModuleGroups(from: items)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    header

                    LMSSegmentedChips(
                        options: sections,
                        selection: $section,
                        label: \.rawValue
                    )

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if let cacheLabel {
                        StalenessChip(label: cacheLabel)
                    }

                    switch section {
                    case .overview:
                        CourseSyllabusSection(course: course)
                    case .modules:
                        modulesSection
                    case .files:
                        CourseFilesView(course: course)
                    case .grades:
                        CourseGradesSection(course: course)
                    case .attendance:
                        CourseAttendanceSection(course: course)
                    case .grading:
                        GradingBacklogSection(course: course)
                    }
                }
                .padding(16)
            }
            .refreshable { await load() }
        }
        .navigationTitle(course.displayTitle)
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(for: CourseStructureItem.self) { item in
            ModuleItemRouteView(course: course, item: item, onProgressChanged: refreshProgress)
        }
        .navigationDestination(for: AttendanceSession.self) { attendanceSession in
            AttendanceSessionDetailView(course: course, attendanceSession: attendanceSession)
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
        .task { await load() }
        .onChange(of: items) { _, loaded in
            guard linkedItem == nil,
                  let itemId = initialItemId,
                  let match = loaded.first(where: { $0.id == itemId }) else { return }
            linkedItem = match
        }
    }

    /// Gradient cover banner — matches the course's tile color across the app.
    private var header: some View {
        ZStack(alignment: .topTrailing) {
            Circle()
                .fill(.white.opacity(0.08))
                .frame(width: 140, height: 140)
                .offset(x: 44, y: -52)

            VStack(alignment: .leading, spacing: 7) {
                Text(course.courseCode.uppercased())
                    .font(.caption2.weight(.semibold))
                    .tracking(1.2)
                    .foregroundStyle(.white.opacity(0.8))
                Text(course.title)
                    .font(LexturesTheme.displayFont(22))
                    .foregroundStyle(.white)
                if !course.description.isEmpty {
                    Text(course.description)
                        .font(.footnote)
                        .foregroundStyle(.white.opacity(0.85))
                        .lineLimit(3)
                }
                HStack(spacing: 6) {
                    if let starts = LMSDates.parse(course.startsAt) {
                        heroChip(starts.formatted(date: .abbreviated, time: .omitted), icon: "calendar")
                    }
                    ForEach(roleBadges, id: \.self) { role in
                        heroChip(role, icon: "person.fill")
                    }
                }
                .padding(.top, 4)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(20)
        }
        .background(LexturesTheme.coverGradient(for: course.courseCode))
        .clipShape(RoundedRectangle(cornerRadius: 24, style: .continuous))
        .shadow(color: LexturesTheme.cardShadow(for: colorScheme), radius: 14, y: 7)
    }

    private var roleBadges: [String] {
        (course.viewerEnrollmentRoles ?? []).map { role in
            role.count <= 2 ? role.uppercased() : role.capitalized
        }
    }

    private func heroChip(_ text: String, icon: String) -> some View {
        Label(text, systemImage: icon)
            .font(.caption.weight(.medium))
            .foregroundStyle(.white)
            .padding(.horizontal, 9)
            .padding(.vertical, 4)
            .background(.white.opacity(0.16))
            .clipShape(Capsule())
    }

    // MARK: Modules

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
}
