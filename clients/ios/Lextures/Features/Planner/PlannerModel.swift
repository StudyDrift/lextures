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
            let decoded = PlannerSnapshotCoding.decodeSnapshot(snapshotResult.value)
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
        let (structures, grades) = await loadStructuresAndGrades(
            studentCourses: studentCourses,
            accessToken: accessToken
        )
        let notebookTasks = try await notebookTasksTask
        let academic = await loadAcademicEvents(studentCourses: studentCourses, accessToken: accessToken)
        let officeHoursByCourse = await loadOfficeHoursByCourse(
            studentCourses: studentCourses,
            accessToken: accessToken
        )
        let liveMeetingsByCourse = await loadLiveMeetingsByCourse(
            studentCourses: studentCourses,
            accessToken: accessToken
        )
        let evaluationStatusByCourse = await loadEvaluationStatuses(
            studentCourses: studentCourses,
            accessToken: accessToken
        )
        let parentConferenceBookings = await loadParentConferenceBookingsIfNeeded(accessToken: accessToken)
        let todos = PlannerLogic.collectTodos(
            studentCourses: studentCourses,
            structureByCourseCode: structures,
            notebookTasks: notebookTasks,
            gradesByCourseCode: grades,
            evaluationStatusByCourseCode: evaluationStatusByCourse
        )
        let events = PlannerLogic.collectCalendarEvents(
            studentCourses: studentCourses,
            structureByCourseCode: structures,
            notebookTasks: notebookTasks,
            academicEvents: academic,
            officeHoursByCourseCode: officeHoursByCourse,
            liveMeetingsByCourseCode: liveMeetingsByCourse,
            parentConferenceBookings: parentConferenceBookings
        )
        return PlannerSnapshotCoding.encodeSnapshot(todos: todos, events: events)
    }

    private static func loadStructuresAndGrades(
        studentCourses: [CourseSummary],
        accessToken: String
    ) async -> (structures: [String: [CourseStructureItem]], grades: [String: MyGradesResponse]) {
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
        return (structures, grades)
    }

    private static func loadAcademicEvents(
        studentCourses: [CourseSummary],
        accessToken: String
    ) async -> [AcademicCalendarEvent] {
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
        return academic
    }

    private static func loadOfficeHoursByCourse(
        studentCourses: [CourseSummary],
        accessToken: String
    ) async -> [String: OfficeHoursAvailability] {
        var officeHoursByCourse: [String: OfficeHoursAvailability] = [:]
        await withTaskGroup(of: (String, OfficeHoursAvailability?).self) { group in
            for course in studentCourses where course.isOfficeHoursEnabled {
                group.addTask {
                    let availability = try? await LMSAPI.fetchOfficeHoursAvailability(
                        courseCode: course.courseCode,
                        accessToken: accessToken
                    )
                    return (course.courseCode, availability)
                }
            }
            for await (code, availability) in group {
                if let availability { officeHoursByCourse[code] = availability }
            }
        }
        return officeHoursByCourse
    }

    private static func loadLiveMeetingsByCourse(
        studentCourses: [CourseSummary],
        accessToken: String
    ) async -> [String: [VirtualMeeting]] {
        var liveMeetingsByCourse: [String: [VirtualMeeting]] = [:]
        await withTaskGroup(of: (String, [VirtualMeeting]?).self) { group in
            for course in studentCourses where course.isLiveSessionsEnabled {
                group.addTask {
                    let meetings = try? await LMSAPI.fetchCourseMeetings(
                        courseCode: course.courseCode,
                        accessToken: accessToken
                    )
                    return (course.courseCode, meetings)
                }
            }
            for await (code, meetings) in group {
                if let meetings { liveMeetingsByCourse[code] = meetings }
            }
        }
        return liveMeetingsByCourse
    }

    private static func loadEvaluationStatuses(
        studentCourses: [CourseSummary],
        accessToken: String
    ) async -> [String: EvaluationStatus] {
        var evaluationStatusByCourse: [String: EvaluationStatus] = [:]
        await withTaskGroup(of: (String, EvaluationStatus?).self) { group in
            for course in studentCourses {
                group.addTask {
                    let status = try? await LMSAPI.fetchEvaluationStatus(
                        courseCode: course.courseCode,
                        accessToken: accessToken
                    )
                    return (course.courseCode, status)
                }
            }
            for await (code, status) in group {
                if let status, status.windowOpen || status.hasSubmitted {
                    evaluationStatusByCourse[code] = status
                }
            }
        }
        return evaluationStatusByCourse
    }

    private static func loadParentConferenceBookingsIfNeeded(accessToken: String) async -> [ParentConferenceBooking] {
        guard let children = try? await LMSAPI.fetchParentChildren(accessToken: accessToken), !children.isEmpty else {
            return []
        }
        return await loadParentConferenceBookings(children: children, accessToken: accessToken)
    }

    private static func loadParentConferenceBookings(
        children: [ParentChildSummary],
        accessToken: String
    ) async -> [ParentConferenceBooking] {
        let childTuples = children.map { child in
            let name = child.displayName?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
                ? child.displayName!
                : child.email
            return (studentId: child.studentUserId, childName: name)
        }
        return await ConferenceLogic.loadParentBookings(children: childTuples, accessToken: accessToken)
    }
}
