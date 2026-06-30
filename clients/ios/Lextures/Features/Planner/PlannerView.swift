import SwiftUI

enum PlannerTab: String, CaseIterable, Identifiable {
    case todos
    case calendar

    var id: String { rawValue }

    var label: String {
        switch self {
        case .todos: return L.text("mobile.planner.tab.todos")
        case .calendar: return L.text("mobile.planner.tab.calendar")
        }
    }

    var systemImage: String {
        switch self {
        case .todos: return "list.bullet.rectangle"
        case .calendar: return "calendar"
        }
    }
}

struct PlannerRoute: Hashable {
    var initialTab: PlannerTab = .todos
}

struct PlannerView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @State private var model = PlannerModel()
    @State private var tab: PlannerTab
    @State private var showSubscribeSheet = false

    init(initialTab: PlannerTab = .todos) {
        _tab = State(initialValue: initialTab)
    }

    var body: some View {
        VStack(spacing: 0) {
            if !NetworkMonitor.shared.isOnline {
                OfflineBanner()
                    .padding(.horizontal, 16)
                    .padding(.top, 8)
            }
            if let stale = model.staleLabel {
                StalenessChip(label: stale)
                    .padding(.horizontal, 16)
                    .padding(.top, 8)
            }

            Picker("", selection: $tab) {
                ForEach(PlannerTab.allCases) { option in
                    Text(option.label).tag(option)
                }
            }
            .pickerStyle(.segmented)
            .padding(.horizontal, 16)
            .padding(.vertical, 10)

            Group {
                switch tab {
                case .todos:
                    TodosView(model: model)
                case .calendar:
                    CalendarView(model: model, onSubscribe: { showSubscribeSheet = true })
                }
            }
        }
        .background(LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea())
        .navigationTitle(L.text("mobile.planner.title"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Menu {
                    Picker(L.text("mobile.planner.reminder.label"), selection: Binding(
                        get: { DueReminderScheduler.selectedLeadTime },
                        set: { DueReminderScheduler.selectedLeadTime = $0 }
                    )) {
                        ForEach(DueReminderLeadTime.allCases) { lead in
                            Text(leadTimeLabel(lead)).tag(lead)
                        }
                    }
                    Button(L.text("mobile.planner.subscribe.title")) {
                        showSubscribeSheet = true
                    }
                } label: {
                    Image(systemName: "ellipsis.circle")
                }
            }
        }
        .sheet(isPresented: $showSubscribeSheet) {
            CalendarSubscribeSheet(courses: model.courses)
        }
        .navigationDestination(for: PlannerItemRoute.self) { route in
            switch route {
            case let .courseItem(course, item):
                ItemDetailView(course: course, item: item)
            case let .courseOnly(course):
                if let course {
                    CourseDetailView(course: course)
                } else {
                    Text(L.text("mobile.planner.error.load"))
                }
            case let .notebook(pageRoute):
                NotebookEditorView(
                    courseCode: pageRoute.courseCode,
                    notebookTitle: pageRoute.notebookTitle,
                    pageId: pageRoute.pageId
                )
            }
        }
        .refreshable {
            await model.load(accessToken: session.accessToken, force: true)
        }
        .task {
            await model.load(accessToken: session.accessToken)
        }
    }

    private func leadTimeLabel(_ lead: DueReminderLeadTime) -> String {
        switch lead {
        case .none: return L.text("mobile.planner.reminder.none")
        case .fifteenMinutes: return L.text("mobile.planner.reminder.15m")
        case .oneHour: return L.text("mobile.planner.reminder.1h")
        case .oneDay: return L.text("mobile.planner.reminder.1d")
        }
    }
}
