import Foundation

// MARK: - Live meetings (M7.5)

struct MeetingJoinResponse: Decodable {
    var joinUrl: String?
    var hostUrl: String?
    var meetingId: String?
    var status: String?
}

struct VirtualMeeting: Codable, Identifiable, Hashable {
    var id: String
    var courseId: String
    var sectionId: String?
    var provider: String
    var title: String
    var scheduledStart: String?
    var scheduledEnd: String?
    var joinUrl: String?
    var hostUrl: String?
    var externalMeetingId: String?
    var status: String
    var createdBy: String
    var createdAt: String
}

struct CourseMeetingsResponse: Decodable {
    var meetings: [VirtualMeeting]?
}

struct MeetingJoinInfo: Decodable {
    var joinUrl: String
    var hostUrl: String?
    var meetingId: String
    var status: String
}

struct MeetingAttendanceRecord: Codable, Identifiable, Hashable {
    var id: String
    var meetingId: String
    var userId: String
    var joinedAt: String
    var leftAt: String?
    var durationSeconds: Int?
}

struct MeetingAttendanceResponse: Decodable {
    var attendance: [MeetingAttendanceRecord]?
}

struct PatchMeetingBody: Encodable {
    var status: String?
}

struct CourseWhiteboard: Codable, Identifiable, Hashable {
    var id: String
    var courseId: String
    var title: String
    var canvasData: [WhiteboardElement]?
    var createdBy: String?
    var createdAt: String
    var updatedAt: String
}

struct CourseWhiteboardsResponse: Decodable {
    var whiteboards: [CourseWhiteboard]?
}
