import SwiftUI

struct TodosView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Bindable var model: PlannerModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                if let error = model.errorMessage {
                    LMSErrorBanner(message: error)
                }

                filterBar

                if model.loading && model.todos.isEmpty {
                    LMSSkeletonList(count: 4)
                } else if model.filteredTodos.isEmpty {
                    LMSEmptyState(
                        systemImage: "checkmark.circle",
                        title: L.text("mobile.planner.todos.empty.title"),
                        message: L.text("mobile.planner.todos.empty.message")
                    )
                } else {
                    ForEach(StudentTodoBucket.allCases) { bucket in
                        let items = model.bucketedTodos[bucket] ?? []
                        if !items.isEmpty {
                            todoSection(bucket: bucket, items: items)
                        }
                    }
                }
            }
            .padding(16)
        }
    }

    private var filterBar: some View {
        VStack(alignment: .leading, spacing: 10) {
            LMSSegmentedChips(
                options: [""] + model.courseFilters.map(\.courseCode),
                selection: Binding(
                    get: { model.selectedCourseCode ?? "" },
                    set: { model.selectedCourseCode = $0.isEmpty ? nil : $0 }
                )
            ) { code in
                if code.isEmpty {
                    L.text("mobile.planner.filter.allCourses")
                } else {
                    model.courseFilters.first(where: { $0.courseCode == code })?.title ?? code
                }
            }
            Toggle(isOn: $model.showCompleted) {
                Text(L.text("mobile.planner.filter.showCompleted"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .toggleStyle(.switch)
        }
    }

    private func todoSection(bucket: StudentTodoBucket, items: [StudentTodoItem]) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            LMSSectionHeader(title: bucketLabel(bucket), systemImage: bucketIcon(bucket))
            ForEach(items) { item in
                NavigationLink(value: plannerDestination(for: item)) {
                    todoRow(item)
                }
                .buttonStyle(.plain)
            }
        }
    }

    private func todoRow(_ item: StudentTodoItem) -> some View {
        LMSCard(accent: item.isCompleted ? LexturesTheme.brandTeal : bucketTint(item)) {
            HStack(spacing: 12) {
                Image(systemName: rowIcon(item))
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(item.isCompleted ? LexturesTheme.brandTeal : bucketTint(item))
                    .frame(width: 34, height: 34)
                    .background((item.isCompleted ? LexturesTheme.brandTeal : bucketTint(item)).opacity(0.12))
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                VStack(alignment: .leading, spacing: 3) {
                    Text(item.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .strikethrough(item.isCompleted)
                        .lineLimit(2)
                    Text(item.courseTitle)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text(statusLabel(item))
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer(minLength: 0)
                if let due = item.dueAt {
                    VStack(alignment: .trailing, spacing: 2) {
                        Text(due.formatted(.dateTime.weekday(.abbreviated)))
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        Text(due.formatted(date: .omitted, time: .shortened))
                            .font(.caption.weight(.bold))
                            .foregroundStyle(bucketTint(item))
                    }
                }
            }
        }
        .task {
            await DueReminderScheduler.scheduleReminder(for: item)
        }
    }

    private func plannerDestination(for item: StudentTodoItem) -> PlannerItemRoute {
        if item.kind == .evaluation, let windowId = item.evaluationWindowId,
           let course = model.courses.first(where: { $0.courseCode == item.courseCode }) {
            return .evaluation(course: course, windowId: windowId)
        }
        if item.kind == .notebookTask, let pageId = item.notebookPageId {
            return .notebook(
                NotebookPageRoute(
                    courseCode: item.courseCode == "__global__" ? "" : item.courseCode,
                    notebookTitle: item.courseTitle,
                    pageId: pageId
                )
            )
        }
        if let course = model.courses.first(where: { $0.courseCode == item.courseCode }),
           let structureId = item.structureItemId,
           let structureKind = item.structureKind {
            let stub = CourseStructureItem(
                id: structureId,
                sortOrder: 0,
                kind: structureKind,
                title: item.title,
                parentId: nil,
                published: true,
                dueAt: item.dueAt.map { ISO8601DateFormatter().string(from: $0) },
                pointsWorth: nil,
                pointsPossible: nil
            )
            return .courseItem(course: course, item: stub)
        }
        return .courseOnly(model.courses.first { $0.courseCode == item.courseCode })
    }

    private func bucketLabel(_ bucket: StudentTodoBucket) -> String {
        switch bucket {
        case .overdue: return L.text("mobile.planner.bucket.overdue")
        case .today: return L.text("mobile.planner.bucket.today")
        case .thisWeek: return L.text("mobile.planner.bucket.thisWeek")
        case .later: return L.text("mobile.planner.bucket.later")
        }
    }

    private func bucketIcon(_ bucket: StudentTodoBucket) -> String {
        switch bucket {
        case .overdue: return "exclamationmark.triangle.fill"
        case .today: return "sun.max.fill"
        case .thisWeek: return "calendar"
        case .later: return "ellipsis.circle"
        }
    }

    private func bucketTint(_ item: StudentTodoItem) -> Color {
        guard let due = item.dueAt else { return LexturesTheme.amber }
        let buckets = PlannerLogic.bucketTodos([item])
        if buckets[.overdue]?.isEmpty == false { return LexturesTheme.error }
        if buckets[.today]?.isEmpty == false { return LexturesTheme.coral }
        return LexturesTheme.amber
    }

    private func rowIcon(_ item: StudentTodoItem) -> String {
        if item.kind == .evaluation { return "star.bubble" }
        if item.kind == .notebookTask { return "checklist" }
        return ItemKind.icon(for: item.structureKind ?? "content_page")
    }

    private func statusLabel(_ item: StudentTodoItem) -> String {
        switch item.completion {
        case .open: return L.text("mobile.planner.status.open")
        case .submitted: return L.text("mobile.planner.status.submitted")
        case .completed: return L.text("mobile.planner.status.completed")
        }
    }
}

enum PlannerItemRoute: Hashable {
    case courseItem(course: CourseSummary, item: CourseStructureItem)
    case courseOnly(CourseSummary?)
    case notebook(NotebookPageRoute)
    case evaluation(course: CourseSummary, windowId: String)
}
