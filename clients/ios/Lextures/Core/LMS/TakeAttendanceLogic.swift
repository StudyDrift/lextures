import Foundation

enum AttendanceMarkStatus: String, CaseIterable, Hashable {
    case present
    case absent
    case tardy
    case excused
}

struct AttendanceSummaryCounts: Equatable {
    var present: Int
    var absent: Int
    var tardy: Int
    var excused: Int
    var notRecorded: Int
}

struct AttendanceRecordUpsert: Encodable {
    var studentUserId: String
    var status: String
    var source: String
}

enum TakeAttendanceLogic {
    static func todayDateString(now: Date = Date()) -> String {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: now)
    }

    static func studentLabel(_ record: AttendanceRecord) -> String {
        let name = record.displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !name.isEmpty { return name }
        return "Student"
    }

    static func buildDraft(from records: [AttendanceRecord]) -> [String: String] {
        var draft: [String: String] = [:]
        for record in records {
            draft[record.studentUserId] = record.status
        }
        return draft
    }

    static func markAllPresent(records: [AttendanceRecord]) -> [String: String] {
        var draft: [String: String] = [:]
        for record in records {
            draft[record.studentUserId] = AttendanceMarkStatus.present.rawValue
        }
        return draft
    }

    static func summaryCounts(records: [AttendanceRecord], draft: [String: String]) -> AttendanceSummaryCounts {
        var present = 0
        var absent = 0
        var tardy = 0
        var excused = 0
        var notRecorded = 0
        for record in records {
            let status = draft[record.studentUserId] ?? record.status
            switch status {
            case AttendanceMarkStatus.present.rawValue: present += 1
            case AttendanceMarkStatus.absent.rawValue: absent += 1
            case AttendanceMarkStatus.tardy.rawValue: tardy += 1
            case AttendanceMarkStatus.excused.rawValue: excused += 1
            default: notRecorded += 1
            }
        }
        return AttendanceSummaryCounts(
            present: present,
            absent: absent,
            tardy: tardy,
            excused: excused,
            notRecorded: notRecorded
        )
    }

    static func recordsPayload(
        records: [AttendanceRecord],
        draft: [String: String]
    ) -> [AttendanceRecordUpsert] {
        records.map { record in
            AttendanceRecordUpsert(
                studentUserId: record.studentUserId,
                status: draft[record.studentUserId] ?? record.status,
                source: "instructor"
            )
        }
    }

    static func findTodaysOpenRollCallSession(
        sessions: [AttendanceSession],
        date: String = todayDateString()
    ) -> AttendanceSession? {
        sessions.first { session in
            session.collectionMethod == "roll_call"
                && session.status == "open"
                && session.sessionDate == date
        }
    }

    static func shouldTakeSession(_ session: AttendanceSession, isStaff: Bool) -> Bool {
        isStaff && session.collectionMethod == "roll_call"
    }

    static func statusLabel(_ status: String) -> String {
        AttendanceStatusInfo.label(status)
    }
}