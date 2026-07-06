import SwiftUI

enum ParentRoute: Hashable {
    case grades(studentId: String)
    case attendance(studentId: String)
    case notificationPrefs
    case conferences(studentId: String)
}

/// Parent portal: child switcher and read-only per-child dashboard (M10.1).
struct ParentDashboardView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    var initialStudentId: String?
    var initialRoute: ParentRoute?

    @State private var children: [ParentChildSummary] = []
    @State private var selectedStudentId: String?
    @State private var grades: [ParentCourseGradesRow] = []
    @State private var assignments: [ParentAssignmentRow] = []
    @State private var attendance: [ParentAttendanceRecord] = []
    @State private var behavior: ParentBehaviorResponse?
    @State private var weeklySummary: ParentWeeklySummaryResponse?
    @State private var loading = true
    @State private var detailLoading = false
    @State private var loadError: String?
    @State private var detailError: String?
    @State private var navigationPath = NavigationPath()

    private var selectedChild: ParentChildSummary? {
        guard let selectedStudentId else { return nil }
        return children.first { $0.studentUserId == selectedStudentId }
    }

    private var displayName: String {
        guard let child = selectedChild else { return "" }
        return ParentLogic.childLabel(child)
    }

    var body: some View {
        NavigationStack(path: $navigationPath) {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                if loading {
                    LMSSkeletonList(count: 4)
                } else if let loadError, children.isEmpty {
                    LMSEmptyState(
                        systemImage: "figure.2.and.child.holdinghands",
                        title: L.text("mobile.parent.title"),
                        message: loadError
                    )
                } else {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 16) {
                            headerSection
                            if children.isEmpty {
                                noChildrenBanner
                            } else {
                                childSwitcher
                                if let selectedChild {
                                    readOnlyBanner(name: ParentLogic.childLabel(selectedChild))
                                }
                                if let detailError {
                                    LMSErrorBanner(message: detailError)
                                }
                                if detailLoading {
                                    LMSSkeletonList(count: 3)
                                } else if selectedStudentId != nil {
                                    summaryCards
                                    actionLinks
                                }
                            }
                        }
                        .padding(16)
                    }
                }
            }
            .navigationTitle(L.text("mobile.parent.title"))
            .navigationBarTitleDisplayMode(.inline)
            .globalDrawerToolbar()
            .navigationDestination(for: ParentRoute.self) { route in
                switch route {
                case let .grades(studentId):
                    ParentGradesDetailView(studentId: studentId, childName: childName(for: studentId))
                case let .attendance(studentId):
                    ParentAttendanceDetailView(studentId: studentId, childName: childName(for: studentId))
                case .notificationPrefs:
                    ParentNotificationPrefsView()
                case let .conferences(studentId):
                    ConferenceBookingView(studentId: studentId, childName: childName(for: studentId))
                }
            }
            .navigationDestination(for: NotificationsRoute.self) { _ in
                NotificationsView()
            }
        }
        .task { await loadChildren() }
        .refreshable { await reloadAll() }
        .onChange(of: selectedStudentId) { _, newId in
            guard let newId else { return }
            MobileIaPreferences.saveSelectedChildId(newId)
            Task { await loadChildDetails(studentId: newId) }
        }
        .onAppear {
            if let route = initialRoute {
                navigationPath.append(route)
            }
        }
    }

    private var headerSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Label(L.text("mobile.parent.badge"), systemImage: "person.2.fill")
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            Text(L.text("mobile.parent.subtitle"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .accessibilityElement(children: .combine)
    }

    private var noChildrenBanner: some View {
        LMSCard {
            Text(L.text("mobile.parent.noChildren"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private var childSwitcher: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(children) { child in
                    let active = child.studentUserId == selectedStudentId
                    Button {
                        selectedStudentId = child.studentUserId
                    } label: {
                        Text(ParentLogic.childLabel(child))
                            .font(.subheadline.weight(.medium))
                            .lineLimit(1)
                            .padding(.horizontal, 14)
                            .padding(.vertical, 8)
                            .background(active ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.cardBackground(for: colorScheme))
                            .foregroundStyle(active ? Color.white : LexturesTheme.textPrimary(for: colorScheme))
                            .clipShape(Capsule())
                            .overlay(
                                Capsule().stroke(
                                    active ? Color.clear : LexturesTheme.fieldBorder(for: colorScheme),
                                    lineWidth: 1
                                )
                            )
                    }
                    .accessibilityLabel(ParentLogic.childLabel(child))
                    .accessibilityAddTraits(active ? .isSelected : [])
                }
            }
        }
        .accessibilityLabel(L.text("mobile.parent.childSwitcher"))
    }

    private func readOnlyBanner(name: String) -> some View {
        LMSCard(accent: LexturesTheme.amber) {
            Text(L.format("mobile.parent.readOnly", name))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
        .accessibilityElement(children: .combine)
    }

    private var summaryCards: some View {
        VStack(alignment: .leading, spacing: 12) {
            gradesSummaryCard
            attendanceSummaryCard
            assignmentsSummaryCard
            behaviorSummaryCard
            weeklySummaryCard
        }
    }

    private var gradesSummaryCard: some View {
        summaryCard(
            title: L.text("mobile.parent.section.grades"),
            empty: L.text("mobile.parent.grades.empty"),
            hasContent: !grades.isEmpty
        ) {
            ForEach(Array(ParentLogic.recentGrades(grades).enumerated()), id: \.offset) { _, row in
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(row.course.title)
                            .font(.subheadline.weight(.medium))
                        Text(row.itemId.prefix(8) + "…")
                            .font(.caption.monospaced())
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer()
                    Text(row.score)
                        .font(.subheadline.weight(.semibold).monospacedDigit())
                }
            }
        }
    }

    private var attendanceSummaryCard: some View {
        let summary = ParentLogic.attendanceSummary(attendance)
        return summaryCard(
            title: L.text("mobile.parent.section.attendance"),
            empty: L.text("mobile.parent.attendance.empty"),
            hasContent: !attendance.isEmpty
        ) {
            Text(L.format("mobile.parent.attendance.summary", summary.present, summary.absent, summary.tardy))
                .font(.subheadline)
            ForEach(ParentLogic.recentAttendance(attendance)) { record in
                HStack {
                    Text(record.date)
                    Spacer()
                    Text(ParentLogic.attendanceLabel(record))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                .font(.caption)
            }
        }
    }

    private var assignmentsSummaryCard: some View {
        summaryCard(
            title: L.text("mobile.parent.section.assignments"),
            empty: L.text("mobile.parent.assignments.empty"),
            hasContent: !assignments.isEmpty
        ) {
            ForEach(ParentLogic.upcomingAssignments(assignments)) { item in
                VStack(alignment: .leading, spacing: 2) {
                    Text(item.title)
                        .font(.subheadline.weight(.medium))
                    HStack {
                        Text("\(item.courseTitle) · \(item.kind)")
                        if let due = item.dueAt {
                            Spacer()
                            Text(DateFormatting.formatDateTime(due))
                        }
                    }
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    private var behaviorSummaryCard: some View {
        let points = behavior?.totalPoints ?? 0
        let referrals = behavior?.referrals?.count ?? 0
        return summaryCard(
            title: L.text("mobile.parent.section.behavior"),
            empty: L.text("mobile.parent.behavior.empty"),
            hasContent: points > 0 || referrals > 0
        ) {
            Text(L.format("mobile.parent.behavior.summary", points, referrals))
                .font(.subheadline)
        }
    }

    private var weeklySummaryCard: some View {
        let items = ParentLogic.weeklyItemsForChild(weeklySummary?.items ?? [], childName: displayName)
        return summaryCard(
            title: L.text("mobile.parent.section.weekly"),
            empty: L.text("mobile.parent.weekly.empty"),
            hasContent: !items.isEmpty
        ) {
            ForEach(items) { item in
                VStack(alignment: .leading, spacing: 2) {
                    Text(item.title)
                        .font(.subheadline.weight(.medium))
                    Text("\(item.courseTitle) · \(item.kind)")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    private var actionLinks: some View {
        VStack(alignment: .leading, spacing: 10) {
            if let studentId = selectedStudentId {
                parentLink(L.text("mobile.parent.viewGrades")) {
                    navigationPath.append(ParentRoute.grades(studentId: studentId))
                }
                parentLink(L.text("mobile.parent.viewAttendance")) {
                    navigationPath.append(ParentRoute.attendance(studentId: studentId))
                }
            }
            parentLink(L.text("mobile.parent.notificationPrefs")) {
                navigationPath.append(ParentRoute.notificationPrefs)
            }
            if shell.platformFeatures.ffConferenceScheduling, let studentId = selectedStudentId {
                parentLink(L.text("mobile.parent.bookConferences")) {
                    navigationPath.append(ParentRoute.conferences(studentId: studentId))
                }
            }
        }
    }

    @ViewBuilder
    private func summaryCard<Content: View>(
        title: String,
        empty: String,
        hasContent: Bool,
        @ViewBuilder content: () -> Content
    ) -> some View {
        LMSCard {
            Text(title)
                .font(.headline)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            if hasContent {
                VStack(alignment: .leading, spacing: 8) {
                    content()
                }
                .padding(.top, 4)
            } else {
                Text(empty)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .padding(.top, 4)
            }
        }
    }

    private func parentLink(_ title: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack {
                Text(title)
                    .font(.subheadline.weight(.medium))
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.semibold))
            }
            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            .padding(.vertical, 10)
            .padding(.horizontal, 14)
            .background(LexturesTheme.cardBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 12))
        }
    }

    private func childName(for studentId: String) -> String {
        children.first { $0.studentUserId == studentId }
            .map(ParentLogic.childLabel) ?? ""
    }

    private func loadChildren() async {
        guard let token = session.accessToken else { return }
        loading = true
        loadError = nil
        defer { loading = false }
        do {
            let list = try await LMSAPI.fetchParentChildren(accessToken: token)
            children = list
            let stored = MobileIaPreferences.loadSelectedChildId()
            let resolved = ParentLogic.resolveSelectedChildId(children: list, storedId: initialStudentId ?? stored)
            selectedStudentId = resolved
            if resolved == nil {
                grades = []
                assignments = []
                attendance = []
                behavior = nil
            }
        } catch {
            loadError = error.localizedDescription
        }
    }

    private func reloadAll() async {
        await loadChildren()
        if let studentId = selectedStudentId {
            await loadChildDetails(studentId: studentId)
        }
        if let token = session.accessToken {
            weeklySummary = try? await LMSAPI.fetchParentWeeklySummary(accessToken: token)
        }
    }

    private func loadChildDetails(studentId: String) async {
        guard let token = session.accessToken else { return }
        detailLoading = true
        detailError = nil
        defer { detailLoading = false }
        do {
            async let gradesTask = LMSAPI.fetchParentStudentGrades(studentId: studentId, accessToken: token)
            async let assignmentsTask = LMSAPI.fetchParentStudentAssignments(studentId: studentId, accessToken: token)
            async let attendanceTask = LMSAPI.fetchParentStudentAttendance(studentId: studentId, accessToken: token)
            async let behaviorTask = LMSAPI.fetchParentStudentBehavior(studentId: studentId, accessToken: token)
            async let weeklyTask = LMSAPI.fetchParentWeeklySummary(accessToken: token)
            grades = try await gradesTask
            assignments = try await assignmentsTask
            attendance = try await attendanceTask
            behavior = try await behaviorTask
            weeklySummary = try await weeklyTask
        } catch {
            detailError = error.localizedDescription
        }
    }
}
