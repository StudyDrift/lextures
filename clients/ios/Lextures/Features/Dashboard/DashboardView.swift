import SwiftUI

struct DueItem: Identifiable, Hashable {
    var id: String { "\(course.courseCode)/\(item.id)" }
    let course: CourseSummary
    let item: CourseStructureItem
    let dueDate: Date
}

/// Per-staff-course ungraded totals for the teacher snapshot card.
struct StaffBacklog: Identifiable, Hashable {
    let course: CourseSummary
    let items: [GradingBacklogItem]

    var id: String { course.id }
    var total: Int { items.reduce(0) { $0 + $1.ungradedCount } }
}

@MainActor
@Observable
final class DashboardModel {
    var courses: [CourseSummary] = []
    var dueThisWeek: [DueItem] = []
    var courseItemCounts: [String: (modules: Int, items: Int)] = [:]
    var staffBacklogs: [StaffBacklog] = []
    var announcements: [Broadcast] = []
    var reviewStats: ReviewStats?
    var errorMessage: String?
    var loading = false
    private var loadedOnce = false

    var staffCourses: [CourseSummary] { courses.filter(\.viewerIsStaff) }
    var ungradedTotal: Int { staffBacklogs.reduce(0) { $0 + $1.total } }

    func load(accessToken: String?, force: Bool = false) async {
        guard let accessToken else { return }
        if loadedOnce && !force { return }
        loading = true
        errorMessage = nil
        defer {
            loading = false
            loadedOnce = true
        }

        do {
            async let broadcastsTask = (try? LMSAPI.fetchMyBroadcasts(accessToken: accessToken)) ?? []

            let listResult = try await OfflineService.shared.cachedFetch(
                key: OfflineCacheKey.courses(),
                accessToken: accessToken
            ) {
                try await LMSAPI.fetchCourses(accessToken: accessToken)
            }
            let list = listResult.value
            // The list GET omits viewer roles; enrich from the single-course GET.
            let enriched = await withTaskGroup(of: CourseSummary.self) { group in
                for course in list {
                    group.addTask {
                        (try? await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: accessToken)) ?? course
                    }
                }
                var out: [CourseSummary] = []
                for await course in group { out.append(course) }
                return out
            }
            // Preserve catalog order from the list response.
            let order = Dictionary(uniqueKeysWithValues: list.enumerated().map { ($1.id, $0) })
            courses = enriched.sorted { (order[$0.id] ?? 0) < (order[$1.id] ?? 0) }

            await loadStructures(accessToken: accessToken)
            announcements = await broadcastsTask
            await loadStaffBacklogs(accessToken: accessToken)
            await loadReviewStats(accessToken: accessToken)
        } catch {
            errorMessage = L.text("mobile.dashboard.error.load")
        }
    }

    /// One structure fetch per course feeds both the due-this-week rail and the course card counts.
    private func loadStructures(accessToken: String) async {
        let (weekStart, weekEnd) = DashboardModel.currentWeek()
        var due: [DueItem] = []
        var counts: [String: (Int, Int)] = [:]
        await withTaskGroup(of: (CourseSummary, [CourseStructureItem]).self) { group in
            for course in courses {
                group.addTask {
                    let items = (try? await LMSAPI.fetchCourseStructure(
                        courseCode: course.courseCode,
                        accessToken: accessToken
                    )) ?? []
                    return (course, items)
                }
            }
            for await (course, items) in group {
                counts[course.courseCode] = (
                    items.filter(\.isModule).count,
                    items.filter { !$0.isModule && $0.kind != "heading" }.count
                )
                guard course.viewerIsStudent else { continue }
                due.append(contentsOf: items.compactMap { item in
                    guard item.isGradable, let dueAt = LMSDates.parse(item.dueAt) else { return nil }
                    guard dueAt >= weekStart, dueAt <= weekEnd else { return nil }
                    return DueItem(course: course, item: item, dueDate: dueAt)
                })
            }
        }
        courseItemCounts = counts
        dueThisWeek = due.sorted { $0.dueDate < $1.dueDate }
    }

    private func loadStaffBacklogs(accessToken: String) async {
        let staff = staffCourses
        guard !staff.isEmpty else {
            staffBacklogs = []
            return
        }
        var out: [StaffBacklog] = []
        await withTaskGroup(of: StaffBacklog?.self) { group in
            for course in staff {
                group.addTask {
                    guard let items = try? await LMSAPI.fetchGradingBacklog(
                        courseCode: course.courseCode,
                        accessToken: accessToken
                    ) else { return nil }
                    return StaffBacklog(course: course, items: items)
                }
            }
            for await backlog in group {
                if let backlog { out.append(backlog) }
            }
        }
        staffBacklogs = out
            .filter { $0.total > 0 }
            .sorted { $0.total > $1.total }
    }

    private func loadReviewStats(accessToken: String) async {
        guard let userId = NotebookStore.jwtSubject(from: accessToken) else {
            reviewStats = nil
            return
        }
        reviewStats = try? await OfflineService.shared.cachedFetch(
            key: OfflineCacheKey.reviewStats(),
            accessToken: accessToken
        ) {
            try await LMSAPI.fetchLearnerReviewStats(userId: userId, accessToken: accessToken)
        }.value
    }

    /// Monday 00:00 through Sunday 23:59 of the current week (parity with web dashboard).
    static func currentWeek(now: Date = Date()) -> (Date, Date) {
        var calendar = Calendar.current
        calendar.firstWeekday = 2 // Monday
        let start = calendar.dateInterval(of: .weekOfYear, for: now)?.start ?? now
        let end = calendar.date(byAdding: DateComponents(day: 7, second: -1), to: start) ?? now
        return (start, end)
    }
}

struct DashboardView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @State private var model = DashboardModel()
    @State private var openReview = false
    @Bindable private var realtime = RealtimeManager.shared

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        DashboardHeroPanel(
                            dueThisWeekCount: model.dueThisWeek.count,
                            loading: model.loading
                        )

                        if let error = model.errorMessage {
                            LMSErrorBanner(message: error)
                        }

                        if model.loading && model.courses.isEmpty {
                            LMSSkeletonList(count: 4)
                        } else {
                            announcementCard
                            reviewCard
                            statsRow
                            teacherSnapshot
                            dueSoonSection
                            DashboardCoursesCarousel(
                                courses: model.courses,
                                loading: model.loading,
                                courseItemCounts: model.courseItemCounts,
                                colorScheme: colorScheme
                            )
                        }
                    }
                    .padding(16)
                    .padding(.bottom, 8)
                }
                .refreshable {
                    await model.load(accessToken: session.accessToken, force: true)
                    await shell.refresh(accessToken: session.accessToken)
                }
            }
            .toolbar(.hidden, for: .navigationBar)
            .navigationDestination(for: CourseSummary.self) { course in
                CourseDetailView(course: course)
            }
            .navigationDestination(for: DueItem.self) { due in
                ItemDetailView(course: due.course, item: due.item)
            }
            .navigationDestination(for: StaffBacklog.self) { backlog in
                GradingBacklogView(course: backlog.course)
            }
            .navigationDestination(for: BroadcastsListRoute.self) { _ in
                AnnouncementsListView()
            }
            .navigationDestination(for: NotificationsRoute.self) { _ in
                NotificationsView()
            }
            .navigationDestination(for: PlannerRoute.self) { route in
                PlannerView(initialTab: route.initialTab)
            }
            .navigationDestination(for: ReviewRoute.self) { _ in
                ReviewHomeView()
            }
            .navigationDestination(isPresented: $openReview) {
                ReviewHomeView()
            }
            .task {
                await model.load(accessToken: session.accessToken)
            }
            .onChange(of: realtime.coursesRevision) { _, _ in
                Task { await model.load(accessToken: session.accessToken, force: true) }
            }
            .onChange(of: realtime.enrollmentsRevision) { _, _ in
                Task { await model.load(accessToken: session.accessToken, force: true) }
            }
            .onAppear { openPendingReviewIfNeeded() }
        }
    }

    private func openPendingReviewIfNeeded() {
        guard shell.consumePendingReview() else { return }
        openReview = true
    }

    // MARK: Announcements

    @ViewBuilder
    private var announcementCard: some View {
        if let broadcast = model.announcements.first {
            AnnouncementCard(broadcast: broadcast, showSeeAll: model.announcements.count > 1) {
                model.announcements.removeAll { $0.id == broadcast.id }
            }
        }
    }

    // MARK: Review practice

    @ViewBuilder
    private var reviewCard: some View {
        if let stats = model.reviewStats {
            NavigationLink(value: ReviewRoute()) {
                LMSCard(accent: LexturesTheme.primary) {
                    HStack(spacing: 12) {
                        Image(systemName: "rectangle.stack.fill")
                            .font(.title3.weight(.semibold))
                            .foregroundStyle(LexturesTheme.primary)
                            .frame(width: 40, height: 40)
                            .background(LexturesTheme.primary.opacity(0.12))
                            .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                        VStack(alignment: .leading, spacing: 4) {
                            Text(L.text("mobile.review.dashboardTitle"))
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Text(stats.dueToday > 0
                                 ? L.plural("mobile.review.dueCount", count: stats.dueToday)
                                 : L.text("mobile.review.caughtUpShort"))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            if stats.streak > 0 {
                                Label(L.plural("mobile.review.streak", count: stats.streak), systemImage: "flame.fill")
                                    .font(.caption2.weight(.semibold))
                                    .foregroundStyle(LexturesTheme.amber)
                            }
                        }
                        Spacer(minLength: 0)
                        Text(stats.dueToday > 0 ? L.text("mobile.review.start") : L.text("mobile.review.open"))
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(LexturesTheme.primary)
                    }
                }
            }
            .buttonStyle(.plain)
        }
    }

    // MARK: Stats

    private var statsRow: some View {
        HStack(spacing: 12) {
            statCard(value: "\(model.courses.count)", label: L.text("mobile.dashboard.stat.courses"), systemImage: "books.vertical.fill", tint: LexturesTheme.accent(for: colorScheme))
            statCard(value: "\(model.dueThisWeek.count)", label: L.text("mobile.dashboard.stat.dueThisWeek"), systemImage: "clock.fill", tint: LexturesTheme.coral)
            statCard(value: "\(shell.unreadInbox)", label: L.text("mobile.dashboard.stat.unread"), systemImage: "tray.fill", tint: LexturesTheme.amber)
        }
    }

    private func statCard(value: String, label: String, systemImage: String, tint: Color) -> some View {
        LMSCard {
            Image(systemName: systemImage)
                .font(.footnote.weight(.semibold))
                .foregroundStyle(tint)
                .frame(width: 30, height: 30)
                .background(tint.opacity(0.14))
                .clipShape(RoundedRectangle(cornerRadius: 9, style: .continuous))
            Text(value)
                .font(LexturesTheme.displayFont(24, weight: .bold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(label)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    // MARK: Teacher snapshot

    @ViewBuilder
    private var teacherSnapshot: some View {
        if !model.staffBacklogs.isEmpty {
            LMSSectionHeader(title: L.text("mobile.dashboard.section.needsGrading"), systemImage: "checkmark.rectangle.stack")
            ForEach(model.staffBacklogs) { backlog in
                NavigationLink(value: backlog) {
                    LMSCard(accent: LexturesTheme.amber) {
                        HStack(spacing: 12) {
                            LMSCoverTile(key: backlog.course.courseCode, systemImage: "checkmark.rectangle.stack", size: 44)
                            VStack(alignment: .leading, spacing: 3) {
                                Text(backlog.course.displayTitle)
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                Text(L.plural("mobile.grading.submissionCount", count: backlog.total))
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            Spacer(minLength: 0)
                            Text("\(backlog.total)")
                                .font(LexturesTheme.displayFont(18, weight: .bold))
                                .foregroundStyle(LexturesTheme.amber)
                                .padding(.horizontal, 10)
                                .padding(.vertical, 4)
                                .background(LexturesTheme.amber.opacity(0.14))
                                .clipShape(Capsule())
                        }
                    }
                }
                .buttonStyle(.plain)
            }
        }
    }

    // MARK: Due soon

    @ViewBuilder
    private var dueSoonSection: some View {
        HStack {
            LMSSectionHeader(title: L.text("mobile.dashboard.section.dueThisWeek"), systemImage: "calendar")
            Spacer(minLength: 0)
            NavigationLink(value: PlannerRoute(initialTab: .todos)) {
                Text(L.text("mobile.planner.viewAll"))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            }
            .buttonStyle(.plain)
        }
        if model.dueThisWeek.isEmpty {
            LMSCard {
                Label {
                    Text(L.text("mobile.dashboard.empty.dueThisWeek"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } icon: {
                    Image(systemName: "sparkles")
                        .foregroundStyle(LexturesTheme.amber)
                }
            }
        } else {
            ForEach(model.dueThisWeek) { due in
                NavigationLink(value: due) {
                    LMSCard(accent: LexturesTheme.coral) {
                        HStack(spacing: 12) {
                            Image(systemName: ItemKind.icon(for: due.item.kind))
                                .font(.footnote.weight(.semibold))
                                .foregroundStyle(LexturesTheme.coral)
                                .frame(width: 34, height: 34)
                                .background(LexturesTheme.coral.opacity(0.12))
                                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                            VStack(alignment: .leading, spacing: 3) {
                                Text(due.item.title)
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                    .lineLimit(1)
                                Text(due.course.displayTitle)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    .lineLimit(1)
                            }
                            Spacer(minLength: 0)
                            VStack(alignment: .trailing, spacing: 2) {
                                Text(due.dueDate.formatted(.dateTime.weekday(.abbreviated)))
                                    .font(.caption2.weight(.semibold))
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                Text(due.dueDate.formatted(date: .omitted, time: .shortened))
                                    .font(.caption.weight(.bold))
                                    .foregroundStyle(LexturesTheme.coral)
                            }
                        }
                    }
                }
                .buttonStyle(.plain)
            }
        }
    }

}

/// Value-type routes for dashboard navigation destinations.
struct NotificationsRoute: Hashable {}
struct BroadcastsListRoute: Hashable {}

#Preview {
    DashboardView()
        .environment(AuthSession())
        .environment(AppShellModel())
}
