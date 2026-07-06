import Foundation

/// Parent portal display helpers and summaries (M10.1).
enum ParentLogic {
    static func childLabel(_ child: ParentChildSummary) -> String {
        let name = child.displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !name.isEmpty { return name }
        return child.email
    }

    static func teacherLabel(_ teacher: ConferenceTeacher) -> String {
        let name = teacher.displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !name.isEmpty { return name }
        return L.text("mobile.parent.conferences.teacherFallback")
    }

    static func resolveSelectedChildId(
        children: [ParentChildSummary],
        storedId: String?
    ) -> String? {
        guard !children.isEmpty else { return nil }
        if let storedId, children.contains(where: { $0.studentUserId == storedId }) {
            return storedId
        }
        return children.first?.studentUserId
    }

    static func attendanceSummary(_ records: [ParentAttendanceRecord]) -> (present: Int, absent: Int, tardy: Int) {
        var present = 0
        var absent = 0
        var tardy = 0
        for record in records {
            let category = (record.category ?? record.code ?? "").lowercased()
            if category.contains("absent") || category == "a" {
                absent += 1
            } else if category.contains("tardy") || category == "t" {
                tardy += 1
            } else {
                present += 1
            }
        }
        return (present, absent, tardy)
    }

    static func recentAttendance(_ records: [ParentAttendanceRecord], limit: Int = 5) -> [ParentAttendanceRecord] {
        records.sorted { ($0.date, $0.recordedAt ?? "") > ($1.date, $1.recordedAt ?? "") }
            .prefix(limit)
            .map { $0 }
    }

    static func recentGrades(_ courses: [ParentCourseGradesRow], limit: Int = 6) -> [(course: ParentCourseGradesRow, itemId: String, score: String)] {
        var rows: [(course: ParentCourseGradesRow, itemId: String, score: String)] = []
        for course in courses {
            for (itemId, score) in course.grades {
                rows.append((course, itemId, score))
            }
        }
        return Array(rows.prefix(limit))
    }

    static func upcomingAssignments(_ assignments: [ParentAssignmentRow], limit: Int = 8) -> [ParentAssignmentRow] {
        assignments
            .sorted { ($0.dueAt ?? "") < ($1.dueAt ?? "") }
            .prefix(limit)
            .map { $0 }
    }

    static func weeklyItemsForChild(
        _ items: [ParentWeeklySummaryItem],
        childName: String
    ) -> [ParentWeeklySummaryItem] {
        items.filter { $0.childName == childName }
    }

    static func attendanceLabel(_ record: ParentAttendanceRecord) -> String {
        if let label = record.codeLabel?.trimmingCharacters(in: .whitespacesAndNewlines), !label.isEmpty {
            return label
        }
        if let code = record.code?.trimmingCharacters(in: .whitespacesAndNewlines), !code.isEmpty {
            return code
        }
        return record.category ?? L.text("mobile.parent.attendance.unknown")
    }
}
