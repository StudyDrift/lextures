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

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        heroPanel

                        if let error = model.errorMessage {
                            LMSErrorBanner(message: error)
                        }

                        if model.loading && model.courses.isEmpty {
                            LMSSkeletonList(count: 4)
                        } else {
                            announcementCard
                            statsRow
                            teacherSnapshot
                            dueSoonSection
                            coursesCarousel
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
            .task {
                await model.load(accessToken: session.accessToken)
            }
        }
    }

    // MARK: Hero

    /// Deep-teal gradient greeting panel with bell + avatar — the brand statement.
    private var heroPanel: some View {
        ZStack(alignment: .topTrailing) {
            // Decorative drifting circles, echoing the rocket's arc in the logo.
            Circle()
                .fill(.white.opacity(0.07))
                .frame(width: 160, height: 160)
                .offset(x: 50, y: -60)
            Circle()
                .fill(LexturesTheme.brandCoral.opacity(0.35))
                .frame(width: 56, height: 56)
                .offset(x: -28, y: 26)

            VStack(alignment: .leading, spacing: 6) {
                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 3) {
                        Text(greetingText + ",")
                            .font(LexturesTheme.displayFont(26))
                            .foregroundStyle(.white)
                        Text(greetingFirstName)
                            .font(LexturesTheme.displayFont(26))
                            .foregroundStyle(LexturesTheme.brandCream)
                            .lineLimit(1)
                    }
                    Spacer(minLength: 8)
                    HStack(spacing: 10) {
                        if shell.iaRedesignEnabled && shell.universalSearchEnabled {
                            searchButton
                        }
                        bellButton
                        LMSAvatarButton()
                    }
                }

                if !model.dueThisWeek.isEmpty {
                    Text(L.plural("mobile.dashboard.dueThisWeek.count", count: model.dueThisWeek.count))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.primaryDeep)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(LexturesTheme.brandCream)
                        .clipShape(Capsule())
                        .padding(.top, 8)
                } else if !model.loading {
                    Text(L.text("mobile.dashboard.caughtUp"))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.white.opacity(0.9))
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(.white.opacity(0.16))
                        .clipShape(Capsule())
                        .padding(.top, 8)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(20)
        }
        .background(LexturesTheme.heroGradient)
        .clipShape(RoundedRectangle(cornerRadius: 24, style: .continuous))
        .shadow(color: LexturesTheme.primaryDeep.opacity(0.25), radius: 14, y: 7)
    }

    private var searchButton: some View {
        Button {
            shell.showUniversalSearch = true
        } label: {
            Image(systemName: "magnifyingglass")
                .font(.subheadline)
                .foregroundStyle(.white)
                .frame(width: 34, height: 34)
                .background(.white.opacity(0.16))
                .clipShape(Circle())
        }
        .buttonStyle(.plain)
        .accessibilityLabel(L.text("mobile.ia.search"))
    }

    private var bellButton: some View {
        NavigationLink(value: NotificationsRoute()) {
            ZStack(alignment: .topTrailing) {
                Image(systemName: "bell.fill")
                    .font(.subheadline)
                    .foregroundStyle(.white)
                    .frame(width: 34, height: 34)
                    .background(.white.opacity(0.16))
                    .clipShape(Circle())
                if shell.unreadNotifications > 0 {
                    Circle()
                        .fill(LexturesTheme.coral)
                        .frame(width: 9, height: 9)
                        .offset(x: -2, y: 2)
                }
            }
        }
        .buttonStyle(.plain)
        .accessibilityLabel(L.text("mobile.profile.notifications"))
    }

    private var greetingFirstName: String {
        if let account = shell.accountProfile {
            let first = account.resolvedNameFields.firstName
            if !first.isEmpty { return first }
        }
        if let first = shell.profile?.firstName, !first.isEmpty { return first }
        return session.userEmail ?? ""
    }

    private var greetingText: String {
        let hour = Calendar.current.component(.hour, from: Date())
        switch hour {
        case ..<12: return L.text("mobile.dashboard.greeting.morning")
        case ..<17: return L.text("mobile.dashboard.greeting.afternoon")
        default: return L.text("mobile.dashboard.greeting.evening")
        }
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

    // MARK: Courses carousel

    @ViewBuilder
    private var coursesCarousel: some View {
        LMSSectionHeader(title: L.text("mobile.dashboard.section.yourCourses"), systemImage: "book.fill")
        if model.courses.isEmpty && !model.loading {
            LMSEmptyState(
                systemImage: "book",
                title: L.text("mobile.dashboard.empty.courses.title"),
                message: L.text("mobile.dashboard.empty.courses.message")
            )
        } else {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 12) {
                    ForEach(model.courses) { course in
                        NavigationLink(value: course) {
                            courseCarouselCard(course)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.vertical, 2)
                .padding(.horizontal, 2)
            }
            .scrollClipDisabled()
            .padding(.bottom, 4)
        }
    }

    private func courseCarouselCard(_ course: CourseSummary) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            ZStack(alignment: .topTrailing) {
                CourseHeroImage(
                    urlString: course.heroImageUrl,
                    fallbackKey: course.courseCode,
                    height: 84
                )
                Image(systemName: "book.fill")
                    .font(.title3)
                    .foregroundStyle(.white.opacity(0.5))
                    .padding(12)
            }
            VStack(alignment: .leading, spacing: 4) {
                Text(course.displayTitle)
                    .font(LexturesTheme.displayFont(15))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .lineLimit(2, reservesSpace: true)
                    .multilineTextAlignment(.leading)
                Text(courseSubtitle(course))
                    .font(.caption2.weight(.medium))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .padding(12)
        }
        .frame(width: 190, alignment: .leading)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(colorScheme == .dark ? 0.9 : 0.45), lineWidth: 1)
        )
        .shadow(color: LexturesTheme.cardShadow(for: colorScheme), radius: 12, y: 5)
    }

    private func courseSubtitle(_ course: CourseSummary) -> String {
        if let counts = model.courseItemCounts[course.courseCode], counts.items > 0 {
            return "\(counts.modules) module\(counts.modules == 1 ? "" : "s") · \(counts.items) item\(counts.items == 1 ? "" : "s")"
        }
        return course.courseCode.uppercased()
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
