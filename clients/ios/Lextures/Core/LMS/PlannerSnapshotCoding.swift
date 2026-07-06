import Foundation

/// Snapshot encode/decode helpers extracted from PlannerLogic (SwiftLint type_body_length).
enum PlannerSnapshotCoding {
    static func encodeSnapshot(
        todos: [StudentTodoItem],
        events: [PlannerCalendarEvent],
        fetchedAt: Date = Date()
    ) -> PlannerSnapshot {
        PlannerSnapshot(
            fetchedAt: fetchedAt,
            todos: todos.map(cachedTodo),
            events: events.map(cachedEvent)
        )
    }

    static func decodeSnapshot(
        _ snapshot: PlannerSnapshot
    ) -> (todos: [StudentTodoItem], events: [PlannerCalendarEvent]) {
        (
            snapshot.todos.map(decodedTodo),
            snapshot.events.map(decodedEvent)
        )
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
            evaluationWindowId: item.evaluationWindowId,
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
            evaluationWindowId: cached.evaluationWindowId,
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
            meetingId: event.meetingId,
            conferenceSlotId: event.conferenceSlotId,
            videoLink: event.videoLink
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
            meetingId: cached.meetingId,
            conferenceSlotId: cached.conferenceSlotId,
            videoLink: cached.videoLink
        )
    }
}
