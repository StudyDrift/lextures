import Foundation
import Observation

@MainActor
@Observable
final class PlannerModel {
    var todos: [StudentTodoItem] = []
    var events: [PlannerCalendarEvent] = []
    var courses: [CourseSummary] = []
    var courseFilters: [PlannerCourseFilter] = []
    var selectedCourseCode: String?
    var showCompleted = false
    var loading = false
    var errorMessage: String?
    var staleLabel: String?
    private var loadedOnce = false

    var studentCourses: [CourseSummary] {
        courses.filter(\.viewerIsStudent)
    }

    var filteredTodos: [StudentTodoItem] {
        var items = todos
        if let code = selectedCourseCode {
            items = items.filter { $0.courseCode == code }
        }
        if !showCompleted {
            items = items.filter { !$0.isCompleted }
        }
        return items
    }

    var bucketedTodos: [StudentTodoBucket: [StudentTodoItem]] {
        PlannerLogic.bucketTodos(filteredTodos)
    }

    var filteredEvents: [PlannerCalendarEvent] {
        guard let code = selectedCourseCode else { return events }
        return events.filter { $0.courseCode == nil || $0.courseCode == code }
    }

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
            let listResult = try await OfflineService.shared.cachedFetch(
                key: OfflineCacheKey.courses(),
                accessToken: accessToken
            ) {
                try await LMSAPI.fetchCourses(accessToken: accessToken)
            }
            let list = listResult.value
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
            let order = Dictionary(uniqueKeysWithValues: list.enumerated().map { ($1.id, $0) })
            courses = enriched.sorted { (order[$0.id] ?? 0) < (order[$1.id] ?? 0) }
            courseFilters = studentCourses
                .filter(\.isCalendarEnabled)
                .map { PlannerCourseFilter(courseCode: $0.courseCode, title: $0.displayTitle) }

            let snapshotResult = try await OfflineService.shared.cachedFetch(
                key: OfflineCacheKey.plannerSnapshot(),
                accessToken: accessToken
            ) {
                try await Self.fetchPlannerSnapshot(
                    studentCourses: self.studentCourses.filter(\.isCalendarEnabled),
                    accessToken: accessToken
                )
            }
            let decoded = PlannerLogic.decodeSnapshot(snapshotResult.value)
            todos = decoded.todos
            events = decoded.events
            if let cached = snapshotResult.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                staleLabel = cached.lastUpdatedLabel
            } else {
                staleLabel = nil
            }
        } catch {
            if todos.isEmpty {
                errorMessage = L.text("mobile.planner.error.load")
            } else {
                staleLabel = staleLabel ?? L.text("mobile.planner.stale.offline")
            }
        }
    }

    private static func fetchPlannerSnapshot(
        studentCourses: [CourseSummary],
        accessToken: String
    ) async throws -> PlannerSnapshot {
        async let notebookTasksTask = LMSAPI.fetchNotebookTasks(accessToken: accessToken)

        var structures: [String: [CourseStructureItem]] = [:]
        var grades: [String: MyGradesResponse] = [:]
        await withTaskGroup(of: (String, [CourseStructureItem]?, MyGradesResponse?).self) { group in
            for course in studentCourses {
                group.addTask {
                    async let structure = try? LMSAPI.fetchCourseStructure(
                        courseCode: course.courseCode,
                        accessToken: accessToken
                    )
                    async let myGrades = try? LMSAPI.fetchMyGrades(
                        courseCode: course.courseCode,
                        accessToken: accessToken
                    )
                    return (course.courseCode, await structure, await myGrades)
                }
            }
            for await (code, structure, myGrades) in group {
                if let structure { structures[code] = structure }
                if let myGrades { grades[code] = myGrades }
            }
        }

        let notebookTasks = try await notebookTasksTask
        var academic: [AcademicCalendarEvent] = []
        var seenKeys = Set<String>()
        for course in studentCourses {
            guard let orgId = course.orgId else { continue }
            let key = "\(orgId):\(course.termId ?? "")"
            guard seenKeys.insert(key).inserted else { continue }
            let rows = (try? await LMSAPI.fetchAcademicCalendarEvents(
                orgId: orgId,
                termId: course.termId,
                accessToken: accessToken
            )) ?? []
            academic.append(contentsOf: rows)
        }

        let todos = PlannerLogic.collectTodos(
            studentCourses: studentCourses,
            structureByCourseCode: structures,
            notebookTasks: notebookTasks,
            gradesByCourseCode: grades
        )
        let events = PlannerLogic.collectCalendarEvents(
            studentCourses: studentCourses,
            structureByCourseCode: structures,
            notebookTasks: notebookTasks,
            academicEvents: academic
        )
        return PlannerLogic.encodeSnapshot(todos: todos, events: events)
    }
}
