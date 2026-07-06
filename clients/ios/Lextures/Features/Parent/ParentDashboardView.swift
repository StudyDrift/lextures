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
                            ParentDashboardHeaderSection()
                            if children.isEmpty {
                                ParentNoChildrenBanner()
                            } else {
                                ParentChildSwitcherView(
                                    children: children,
                                    selectedStudentId: $selectedStudentId
                                )
                                if let selectedChild {
                                    ParentReadOnlyBanner(name: ParentLogic.childLabel(selectedChild))
                                }
                                if let detailError {
                                    LMSErrorBanner(message: detailError)
                                }
                                if detailLoading {
                                    LMSSkeletonList(count: 3)
                                } else if selectedStudentId != nil {
                                    ParentDashboardSummarySection(
                                        grades: grades,
                                        assignments: assignments,
                                        attendance: attendance,
                                        behavior: behavior,
                                        weeklySummary: weeklySummary,
                                        displayName: displayName
                                    )
                                    ParentDashboardActionLinks(
                                        selectedStudentId: selectedStudentId,
                                        conferenceSchedulingEnabled: shell.platformFeatures.ffConferenceScheduling,
                                        onGrades: {
                                            if let studentId = selectedStudentId {
                                                navigationPath.append(ParentRoute.grades(studentId: studentId))
                                            }
                                        },
                                        onAttendance: {
                                            if let studentId = selectedStudentId {
                                                navigationPath.append(ParentRoute.attendance(studentId: studentId))
                                            }
                                        },
                                        onNotificationPrefs: {
                                            navigationPath.append(ParentRoute.notificationPrefs)
                                        },
                                        onConferences: {
                                            if let studentId = selectedStudentId {
                                                navigationPath.append(ParentRoute.conferences(studentId: studentId))
                                            }
                                        }
                                    )
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
