import XCTest
@testable import Lextures

final class PlannerModelsTests: XCTestCase {
    func testBucketsOverdueTodayAndLater() {
        let calendar = Calendar.current
        let now = calendar.date(from: DateComponents(year: 2026, month: 6, day: 30, hour: 12))!
        let yesterday = calendar.date(byAdding: .day, value: -1, to: now)!
        let later = calendar.date(byAdding: .day, value: 10, to: now)!

        let items = [
            makeTodo(key: "a", due: yesterday),
            makeTodo(key: "b", due: now),
            makeTodo(key: "c", due: later),
        ]
        let buckets = PlannerLogic.bucketTodos(items, now: now)
        XCTAssertEqual(buckets[.overdue]?.map(\.key), ["a"])
        XCTAssertEqual(buckets[.today]?.map(\.key), ["b"])
        XCTAssertEqual(buckets[.later]?.map(\.key), ["c"])
    }

    func testCollectTodosFromStructure() {
        let course = CourseSummary(
            id: "1",
            courseCode: "BIO101",
            title: "Biology",
            description: "",
            heroImageUrl: nil,
            startsAt: nil,
            endsAt: nil,
            published: true,
            catalogNickname: nil,
            notebookEnabled: true,
            calendarEnabled: true,
            orgId: nil,
            termId: nil,
            viewerEnrollmentRoles: ["student"]
        )
        let structure = [
            CourseStructureItem(
                id: "q1",
                sortOrder: 1,
                kind: "quiz",
                title: "Quiz 1",
                parentId: nil,
                published: true,
                dueAt: "2026-07-01T23:59:00Z",
                pointsWorth: 10,
                pointsPossible: nil
            ),
        ]
        let todos = PlannerLogic.collectTodos(
            studentCourses: [course],
            structureByCourseCode: ["BIO101": structure],
            notebookTasks: [],
            gradesByCourseCode: [:]
        )
        XCTAssertEqual(todos.count, 1)
        XCTAssertEqual(todos[0].title, "Quiz 1")
        XCTAssertEqual(todos[0].structureKind, "quiz")
    }

    func testMonthGridHas42Cells() {
        let anchor = Calendar.current.date(from: DateComponents(year: 2026, month: 6, day: 1))!
        XCTAssertEqual(PlannerLogic.monthGridCells(monthAnchor: anchor).count, 42)
    }

    func testSnapshotRoundTrip() {
        let todo = makeTodo(key: "due:BIO101:q1", due: Date())
        let event = PlannerCalendarEvent(
            id: "due:BIO101:q1",
            title: "Quiz 1",
            courseCode: "BIO101",
            courseTitle: "Biology",
            startsAt: Date(),
            endsAt: nil,
            allDay: false,
            kind: .quiz,
            structureKind: "quiz",
            structureItemId: "q1",
            notebookPageId: nil
        )
        let snapshot = PlannerLogic.encodeSnapshot(todos: [todo], events: [event])
        let decoded = PlannerLogic.decodeSnapshot(snapshot)
        XCTAssertEqual(decoded.todos.count, 1)
        XCTAssertEqual(decoded.events.count, 1)
        XCTAssertEqual(decoded.todos[0].key, todo.key)
    }

    private func makeTodo(key: String, due: Date) -> StudentTodoItem {
        StudentTodoItem(
            key: key,
            kind: .dueItem,
            title: "Item",
            courseCode: "BIO101",
            courseTitle: "Biology",
            dueAt: due,
            structureKind: "assignment",
            structureItemId: "a1",
            notebookPageId: nil,
            notebookTaskId: nil,
            completion: .open
        )
    }
}
