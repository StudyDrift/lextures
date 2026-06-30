import Foundation

// MARK: - Models

enum StudentTodoKind: String, Hashable {
    case dueItem
    case notebookTask
}

enum StudentTodoCompletion: String, Hashable {
    case open
    case submitted
    case completed
}

enum StudentTodoBucket: String, CaseIterable, Identifiable {
    case overdue
    case today
    case thisWeek
    case later

    var id: String { rawValue }
}

enum PlannerCalendarEventKind: String, Hashable {
    case assignment
    case quiz
    case contentPage
    case notebookTask
    case academic
    case officeHours
}

struct StudentTodoItem: Identifiable, Hashable {
    var id: String { key }
    let key: String
    let kind: StudentTodoKind
    let title: String
    let courseCode: String
    let courseTitle: String
    let dueAt: Date?
    let structureKind: String?
    let structureItemId: String?
    let notebookPageId: String?
    let notebookTaskId: String?
    var completion: StudentTodoCompletion

    var isCompleted: Bool { completion == .completed || completion == .submitted }
}

struct PlannerCalendarEvent: Identifiable, Hashable {
    var id: String
    let title: String
    let courseCode: String?
    let courseTitle: String?
    let startsAt: Date
    let endsAt: Date?
    let allDay: Bool
    let kind: PlannerCalendarEventKind
    let structureKind: String?
    let structureItemId: String?
    let notebookPageId: String?
    let officeHoursSlotId: String?
    let meetingId: String?

    init(
        id: String,
        title: String,
        courseCode: String?,
        courseTitle: String?,
        startsAt: Date,
        endsAt: Date?,
        allDay: Bool,
        kind: PlannerCalendarEventKind,
        structureKind: String?,
        structureItemId: String?,
        notebookPageId: String?,
        officeHoursSlotId: String? = nil,
        meetingId: String? = nil
    ) {
        self.id = id
        self.title = title
        self.courseCode = courseCode
        self.courseTitle = courseTitle
        self.startsAt = startsAt
        self.endsAt = endsAt
        self.allDay = allDay
        self.kind = kind
        self.structureKind = structureKind
        self.structureItemId = structureItemId
        self.notebookPageId = notebookPageId
        self.officeHoursSlotId = officeHoursSlotId
        self.meetingId = meetingId
    }
}

struct PlannerCourseFilter: Hashable, Identifiable {
    var id: String { courseCode }
    let courseCode: String
    let title: String
}

/// Cached planner payload for offline reads.
struct PlannerSnapshot: Codable {
    var fetchedAt: Date
    var todos: [CachedStudentTodoItem]
    var events: [CachedPlannerCalendarEvent]
}

struct CachedStudentTodoItem: Codable, Hashable {
    var key: String
    var kind: String
    var title: String
    var courseCode: String
    var courseTitle: String
    var dueAt: String?
    var structureKind: String?
    var structureItemId: String?
    var notebookPageId: String?
    var notebookTaskId: String?
    var completion: String
}

struct CachedPlannerCalendarEvent: Codable, Hashable {
    var id: String
    var title: String
    var courseCode: String?
    var courseTitle: String?
    var startsAt: String
    var endsAt: String?
    var allDay: Bool
    var kind: String
    var structureKind: String?
    var structureItemId: String?
    var notebookPageId: String?
    var officeHoursSlotId: String?
    var meetingId: String?
}

enum DueReminderLeadTime: Int, CaseIterable, Identifiable {
    case none = 0
    case fifteenMinutes = 15
    case oneHour = 60
    case oneDay = 1440

    var id: Int { rawValue }

    var labelKey: String {
        switch self {
        case .none: return "mobile.planner.reminder.none"
        case .fifteenMinutes: return "mobile.planner.reminder.15m"
        case .oneHour: return "mobile.planner.reminder.1h"
        case .oneDay: return "mobile.planner.reminder.1d"
        }
    }
}

// MARK: - Collection

enum PlannerLogic {
    static func dueItemKey(courseCode: String, itemId: String) -> String {
        "due:\(courseCode):\(itemId)"
    }

    static func notebookTaskKey(taskId: String) -> String {
        "notebook:\(taskId)"
    }

    static func collectTodos(
        studentCourses: [CourseSummary],
        structureByCourseCode: [String: [CourseStructureItem]],
        notebookTasks: [NotebookTask],
        gradesByCourseCode: [String: MyGradesResponse]
    ) -> [StudentTodoItem] {
        let courseTitles = Dictionary(uniqueKeysWithValues: studentCourses.map { ($0.courseCode, $0.displayTitle) })
        let studentCodes = Set(studentCourses.map(\.courseCode))
        var items: [StudentTodoItem] = []

        for task in notebookTasks where !task.completed {
            if task.courseCode != "__global__", !studentCodes.contains(task.courseCode) { continue }
            let title = task.taskText.trimmingCharacters(in: .whitespacesAndNewlines)
            items.append(
                StudentTodoItem(
                    key: notebookTaskKey(taskId: task.id),
                    kind: .notebookTask,
                    title: title.isEmpty ? "Untitled task" : title,
                    courseCode: task.courseCode,
                    courseTitle: courseTitle(for: task.courseCode, titles: courseTitles),
                    dueAt: LMSDates.parse(task.dueAt),
                    structureKind: nil,
                    structureItemId: nil,
                    notebookPageId: task.notebookPageId,
                    notebookTaskId: task.id,
                    completion: task.completed ? .completed : .open
                )
            )
        }

        for course in studentCourses where course.isCalendarEnabled {
            let structure = structureByCourseCode[course.courseCode] ?? []
            let grades = gradesByCourseCode[course.courseCode]
            for row in structure where isDueStructureItem(row) {
                let completion = completionStatus(
                    itemId: row.id,
                    kind: row.kind,
                    grades: grades
                )
                items.append(
                    StudentTodoItem(
                        key: dueItemKey(courseCode: course.courseCode, itemId: row.id),
                        kind: .dueItem,
                        title: row.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                            ? "Untitled"
                            : row.title.trimmingCharacters(in: .whitespacesAndNewlines),
                        courseCode: course.courseCode,
                        courseTitle: course.displayTitle,
                        dueAt: LMSDates.parse(row.dueAt),
                        structureKind: row.kind,
                        structureItemId: row.id,
                        notebookPageId: nil,
                        notebookTaskId: nil,
                        completion: completion
                    )
                )
            }
        }

        return items.sorted { lhs, rhs in
            switch (lhs.dueAt, rhs.dueAt) {
            case let (left?, right?): return left < right
            case (nil, _?): return false
            case (_?, nil): return true
            case (nil, nil): return lhs.title.localizedCaseInsensitiveCompare(rhs.title) == .orderedAscending
            }
        }
    }

    static func bucketTodos(_ items: [StudentTodoItem], now: Date = Date()) -> [StudentTodoBucket: [StudentTodoItem]] {
        var calendar = Calendar.current
        calendar.firstWeekday = 2
        let startOfToday = calendar.startOfDay(for: now)
        let endOfToday = calendar.date(byAdding: DateComponents(day: 1, second: -1), to: startOfToday) ?? now
        let weekStart = calendar.dateInterval(of: .weekOfYear, for: now)?.start ?? startOfToday
        let weekEnd = calendar.date(byAdding: DateComponents(day: 7, second: -1), to: weekStart) ?? now

        var buckets = Dictionary(uniqueKeysWithValues: StudentTodoBucket.allCases.map { ($0, [StudentTodoItem]()) })
        for item in items {
            guard let due = item.dueAt else {
                buckets[.later, default: []].append(item)
                continue
            }
            if due < startOfToday {
                buckets[.overdue, default: []].append(item)
            } else if due <= endOfToday {
                buckets[.today, default: []].append(item)
            } else if due <= weekEnd {
                buckets[.thisWeek, default: []].append(item)
            } else {
                buckets[.later, default: []].append(item)
            }
        }
        return buckets
    }

    static func collectCalendarEvents(
        studentCourses: [CourseSummary],
        structureByCourseCode: [String: [CourseStructureItem]],
        notebookTasks: [NotebookTask],
        academicEvents: [AcademicCalendarEvent],
        officeHoursByCourseCode: [String: OfficeHoursAvailability] = [:]
    ) -> [PlannerCalendarEvent] {
        var events: [PlannerCalendarEvent] = []
        let courseTitles = Dictionary(uniqueKeysWithValues: studentCourses.map { ($0.courseCode, $0.displayTitle) })

        for course in studentCourses where course.isCalendarEnabled {
            let structure = structureByCourseCode[course.courseCode] ?? []
            for row in structure where isDueStructureItem(row) {
                guard let due = LMSDates.parse(row.dueAt) else { continue }
                events.append(
                    PlannerCalendarEvent(
                        id: "due:\(course.courseCode):\(row.id)",
                        title: row.title,
                        courseCode: course.courseCode,
                        courseTitle: course.displayTitle,
                        startsAt: due,
                        endsAt: nil,
                        allDay: false,
                        kind: calendarKind(for: row.kind),
                        structureKind: row.kind,
                        structureItemId: row.id,
                        notebookPageId: nil
                    )
                )
            }
        }

        for task in notebookTasks where !task.completed {
            guard let dueRaw = task.dueAt, let due = LMSDates.parse(dueRaw) else { continue }
            events.append(
                PlannerCalendarEvent(
                    id: "notebook:\(task.id)",
                    title: task.taskText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                        ? "Notebook task"
                        : task.taskText.trimmingCharacters(in: .whitespacesAndNewlines),
                    courseCode: task.courseCode == "__global__" ? nil : task.courseCode,
                    courseTitle: task.courseCode == "__global__"
                        ? nil
                        : courseTitle(for: task.courseCode, titles: courseTitles),
                    startsAt: due,
                    endsAt: nil,
                    allDay: false,
                    kind: .notebookTask,
                    structureKind: nil,
                    structureItemId: nil,
                    notebookPageId: task.notebookPageId
                )
            )
        }

        for event in academicEvents {
            guard let start = LMSDates.parse(event.startDate) else { continue }
            let end = event.endDate.flatMap { LMSDates.parse($0) }
            events.append(
                PlannerCalendarEvent(
                    id: "academic:\(event.id)",
                    title: event.eventName,
                    courseCode: nil,
                    courseTitle: nil,
                    startsAt: start,
                    endsAt: end,
                    allDay: event.allDay,
                    kind: .academic,
                    structureKind: nil,
                    structureItemId: nil,
                    notebookPageId: nil
                )
            )
        }

        events.append(contentsOf: OfficeHoursLogic.collectCalendarEvents(
            studentCourses: studentCourses,
            availabilityByCourseCode: officeHoursByCourseCode
        ))

        return events.sorted { $0.startsAt < $1.startsAt }
    }

    static func monthGridCells(monthAnchor: Date) -> [Date] {
        var calendar = Calendar.current
        calendar.firstWeekday = 2
        let start = calendar.date(from: calendar.dateComponents([.year, .month], from: monthAnchor)) ?? monthAnchor
        let mondayOffset = (calendar.component(.weekday, from: start) + 5) % 7
        let gridStart = calendar.date(byAdding: .day, value: -mondayOffset, to: start) ?? start
        return (0 ..< 42).compactMap { calendar.date(byAdding: .day, value: $0, to: gridStart) }
    }

    static func dateKeyLocal(_ date: Date) -> String {
        let calendar = Calendar.current
        let year = calendar.component(.year, from: date)
        let month = calendar.component(.month, from: date)
        let day = calendar.component(.day, from: date)
        return String(format: "%04d-%02d-%02d", year, month, day)
    }

    static func events(on day: Date, events: [PlannerCalendarEvent]) -> [PlannerCalendarEvent] {
        let key = dateKeyLocal(day)
        return events.filter { dateKeyLocal($0.startsAt) == key }
    }

    static func eventCountsByDay(events: [PlannerCalendarEvent]) -> [String: Int] {
        var counts: [String: Int] = [:]
        for event in events {
            let key = dateKeyLocal(event.startsAt)
            counts[key, default: 0] += 1
        }
        return counts
    }

    static func dueSoonItems(from items: [StudentTodoItem], limit: Int = 5, now: Date = Date()) -> [StudentTodoItem] {
        items
            .filter { !$0.isCompleted }
            .filter { item in
                guard let due = item.dueAt else { return false }
                let buckets = bucketTodos([item], now: now)
                return buckets[.overdue]?.isEmpty == false
                    || buckets[.today]?.isEmpty == false
                    || buckets[.thisWeek]?.isEmpty == false
            }
            .prefix(limit)
            .map { $0 }
    }

    static func encodeSnapshot(todos: [StudentTodoItem], events: [PlannerCalendarEvent], fetchedAt: Date = Date()) -> PlannerSnapshot {
        PlannerSnapshot(
            fetchedAt: fetchedAt,
            todos: todos.map(cachedTodo),
            events: events.map(cachedEvent)
        )
    }

    static func decodeSnapshot(_ snapshot: PlannerSnapshot) -> (todos: [StudentTodoItem], events: [PlannerCalendarEvent]) {
        (
            snapshot.todos.map(decodedTodo),
            snapshot.events.map(decodedEvent)
        )
    }

    // MARK: - Private

    private static func isDueStructureItem(_ item: CourseStructureItem) -> Bool {
        (item.kind == "content_page" || item.kind == "assignment" || item.kind == "quiz")
            && item.dueAt?.isEmpty == false
    }

    private static func calendarKind(for structureKind: String) -> PlannerCalendarEventKind {
        switch structureKind {
        case "assignment": return .assignment
        case "quiz": return .quiz
        default: return .contentPage
        }
    }

    private static func completionStatus(itemId: String, kind: String, grades: MyGradesResponse?) -> StudentTodoCompletion {
        guard let grades else { return .open }
        if grades.grades[itemId] != nil || grades.displayGrades[itemId] != nil {
            return .completed
        }
        let status = grades.gradeStatuses[itemId]?.lowercased() ?? ""
        if status.contains("submit") { return .submitted }
        if status.contains("complete") || status.contains("graded") { return .completed }
        return .open
    }

    private static func courseTitle(for courseCode: String, titles: [String: String]) -> String {
        if courseCode == "__global__" { return "Notebook" }
        return titles[courseCode] ?? courseCode
    }

    private static func cachedTodo(_ item: StudentTodoItem) -> CachedStudentTodoItem {
        CachedStudentTodoItem(
            key: item.key,
            kind: item.kind.rawValue,
            title: item.title,
            courseCode: item.courseCode,
            courseTitle: item.courseTitle,
            dueAt: item.dueAt.map { ISO8601DateFormatter().string(from: $0) },
            structureKind: item.structureKind,
            structureItemId: item.structureItemId,
            notebookPageId: item.notebookPageId,
            notebookTaskId: item.notebookTaskId,
            completion: item.completion.rawValue
        )
    }

    private static func decodedTodo(_ cached: CachedStudentTodoItem) -> StudentTodoItem {
        StudentTodoItem(
            key: cached.key,
            kind: StudentTodoKind(rawValue: cached.kind) ?? .dueItem,
            title: cached.title,
            courseCode: cached.courseCode,
            courseTitle: cached.courseTitle,
            dueAt: LMSDates.parse(cached.dueAt),
            structureKind: cached.structureKind,
            structureItemId: cached.structureItemId,
            notebookPageId: cached.notebookPageId,
            notebookTaskId: cached.notebookTaskId,
            completion: StudentTodoCompletion(rawValue: cached.completion) ?? .open
        )
    }

    private static func cachedEvent(_ event: PlannerCalendarEvent) -> CachedPlannerCalendarEvent {
        CachedPlannerCalendarEvent(
            id: event.id,
            title: event.title,
            courseCode: event.courseCode,
            courseTitle: event.courseTitle,
            startsAt: ISO8601DateFormatter().string(from: event.startsAt),
            endsAt: event.endsAt.map { ISO8601DateFormatter().string(from: $0) },
            allDay: event.allDay,
            kind: event.kind.rawValue,
            structureKind: event.structureKind,
            structureItemId: event.structureItemId,
            notebookPageId: event.notebookPageId,
            officeHoursSlotId: event.officeHoursSlotId,
            meetingId: event.meetingId
        )
    }

    private static func decodedEvent(_ cached: CachedPlannerCalendarEvent) -> PlannerCalendarEvent {
        PlannerCalendarEvent(
            id: cached.id,
            title: cached.title,
            courseCode: cached.courseCode,
            courseTitle: cached.courseTitle,
            startsAt: LMSDates.parse(cached.startsAt) ?? Date(),
            endsAt: cached.endsAt.flatMap { LMSDates.parse($0) },
            allDay: cached.allDay,
            kind: PlannerCalendarEventKind(rawValue: cached.kind) ?? .academic,
            structureKind: cached.structureKind,
            structureItemId: cached.structureItemId,
            notebookPageId: cached.notebookPageId,
            officeHoursSlotId: cached.officeHoursSlotId,
            meetingId: cached.meetingId
        )
    }
}
