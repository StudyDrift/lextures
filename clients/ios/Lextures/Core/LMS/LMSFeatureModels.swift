import Foundation

// MARK: - Profile

/// GET `/api/v1/me`.
struct MeProfile: Decodable {
    var id: String
    var email: String
    var displayName: String?

    /// First name for greetings; falls back to the email local part.
    var firstName: String {
        let name = displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !name.isEmpty {
            return String(name.split(separator: " ").first ?? Substring(name))
        }
        return String(email.split(separator: "@").first ?? "there")
    }

    /// Two-letter initials for the avatar chip.
    var initials: String {
        let name = displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let source = name.isEmpty ? String(email.split(separator: "@").first ?? "?") : name
        let parts = source.split(separator: " ")
        if parts.count >= 2,
           let firstInitial = parts.first?.first,
           let lastInitial = parts.last?.first {
            return String([firstInitial, lastInitial]).uppercased()
        }
        return String(source.prefix(2)).uppercased()
    }
}

// MARK: - Notifications

/// Row from GET `/api/v1/me/notifications`.
struct AppNotification: Codable, Identifiable, Hashable {
    var id: String
    var eventType: String
    var title: String
    var body: String
    var actionUrl: String?
    var isRead: Bool
    var createdAt: String

    init(
        id: String,
        eventType: String,
        title: String,
        body: String,
        actionUrl: String? = nil,
        isRead: Bool = false,
        createdAt: String = ""
    ) {
        self.id = id
        self.eventType = eventType
        self.title = title
        self.body = body
        self.actionUrl = actionUrl
        self.isRead = isRead
        self.createdAt = createdAt
    }
}

struct NotificationsPage: Codable {
    var notifications: [AppNotification]
    var unreadCount: Int

    enum CodingKeys: String, CodingKey { case notifications, unreadCount }

    init(notifications: [AppNotification] = [], unreadCount: Int = 0) {
        self.notifications = notifications
        self.unreadCount = unreadCount
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        notifications = try container.decodeIfPresent([AppNotification].self, forKey: .notifications) ?? []
        unreadCount = try container.decodeIfPresent(Int.self, forKey: .unreadCount) ?? 0
    }
}

/// Row from GET `/api/v1/me/notification-preferences`.
struct NotificationPreference: Codable, Identifiable, Hashable {
    var id: String { eventType }
    var eventType: String
    var emailEnabled: Bool
    var pushEnabled: Bool
    var smsEnabled: Bool
    var digestMode: String

    enum CodingKeys: String, CodingKey {
        case eventType, emailEnabled, pushEnabled, smsEnabled, digestMode
    }

    init(
        eventType: String,
        emailEnabled: Bool = true,
        pushEnabled: Bool = true,
        smsEnabled: Bool = false,
        digestMode: String = "instant"
    ) {
        self.eventType = eventType
        self.emailEnabled = emailEnabled
        self.pushEnabled = pushEnabled
        self.smsEnabled = smsEnabled
        self.digestMode = digestMode
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        eventType = try container.decode(String.self, forKey: .eventType)
        emailEnabled = try container.decodeIfPresent(Bool.self, forKey: .emailEnabled) ?? true
        pushEnabled = try container.decodeIfPresent(Bool.self, forKey: .pushEnabled) ?? true
        smsEnabled = try container.decodeIfPresent(Bool.self, forKey: .smsEnabled) ?? false
        digestMode = try container.decodeIfPresent(String.self, forKey: .digestMode) ?? "instant"
    }
}

struct NotificationPreferencesResponse: Decodable {
    var preferences: [NotificationPreference]

    enum CodingKeys: String, CodingKey { case preferences }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        preferences = try container.decodeIfPresent([NotificationPreference].self, forKey: .preferences) ?? []
    }
}

struct NotificationPreferencePatch: Encodable {
    var eventType: String
    var emailEnabled: Bool?
    var pushEnabled: Bool?
    var smsEnabled: Bool?
    var digestMode: String?
}

struct NotificationPreferencesUpdate: Encodable {
    var preferences: [NotificationPreferencePatch]
}

struct DeviceTokenRegistration: Encodable {
    var token: String
    var platform: String
    var appBundleId: String?
    var appVersion: String?
}

struct DeviceTokenResponse: Decodable {
    var id: String
}

struct DeviceTokensPage: Decodable {
    var tokens: [DevicePushToken]
}

struct DevicePushToken: Decodable, Identifiable {
    var id: String
    var platform: String
    var appBundleId: String?
    var appVersion: String?
    var isActive: Bool
    var createdAt: String
}

// MARK: - Announcements (org broadcasts)

/// Row from GET `/api/v1/me/broadcasts`.
struct Broadcast: Decodable, Identifiable, Hashable {
    var id: String
    var type: String // "announcement" | "emergency"
    var subject: String
    var body: String
    var sentAt: String?
    var createdAt: String

    var isEmergency: Bool { type == "emergency" }
}

struct BroadcastsResponse: Decodable {
    var broadcasts: [Broadcast]
}

// MARK: - My grades

/// Column from `/my-grades` / gradebook grid (subset used on mobile).
struct GradeColumn: Codable, Identifiable, Hashable {
    var id: String
    var kind: String
    var title: String
    var maxPoints: Double?
    var dueAt: String?
    var assignmentGroupId: String?
    var neverDrop: Bool
    var replaceWithFinal: Bool
    var rubric: RubricDefinition?

    enum CodingKeys: String, CodingKey {
        case id, kind, title, maxPoints, dueAt, assignmentGroupId
        case neverDrop, replaceWithFinal, rubric
    }

    init(
        id: String,
        kind: String,
        title: String,
        maxPoints: Double? = nil,
        dueAt: String? = nil,
        assignmentGroupId: String? = nil,
        neverDrop: Bool = false,
        replaceWithFinal: Bool = false,
        rubric: RubricDefinition? = nil
    ) {
        self.id = id
        self.kind = kind
        self.title = title
        self.maxPoints = maxPoints
        self.dueAt = dueAt
        self.assignmentGroupId = assignmentGroupId
        self.neverDrop = neverDrop
        self.replaceWithFinal = replaceWithFinal
        self.rubric = rubric
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        kind = try container.decodeIfPresent(String.self, forKey: .kind) ?? ""
        title = try container.decodeIfPresent(String.self, forKey: .title) ?? ""
        maxPoints = try container.decodeIfPresent(Double.self, forKey: .maxPoints)
        dueAt = try container.decodeIfPresent(String.self, forKey: .dueAt)
        assignmentGroupId = try container.decodeIfPresent(String.self, forKey: .assignmentGroupId)
        neverDrop = try container.decodeIfPresent(Bool.self, forKey: .neverDrop) ?? false
        replaceWithFinal = try container.decodeIfPresent(Bool.self, forKey: .replaceWithFinal) ?? false
        rubric = try container.decodeIfPresent(RubricDefinition.self, forKey: .rubric)
    }
}

struct AssignmentGroup: Codable, Hashable {
    var id: String
    var name: String
    var weightPercent: Double
    var dropLowest: Int
    var dropHighest: Int
    var replaceLowestWithFinal: Bool

    enum CodingKeys: String, CodingKey {
        case id, name, weightPercent, dropLowest, dropHighest, replaceLowestWithFinal
    }

    init(
        id: String,
        name: String,
        weightPercent: Double,
        dropLowest: Int = 0,
        dropHighest: Int = 0,
        replaceLowestWithFinal: Bool = false
    ) {
        self.id = id
        self.name = name
        self.weightPercent = weightPercent
        self.dropLowest = dropLowest
        self.dropHighest = dropHighest
        self.replaceLowestWithFinal = replaceLowestWithFinal
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        name = try container.decodeIfPresent(String.self, forKey: .name) ?? ""
        weightPercent = try container.decodeIfPresent(Double.self, forKey: .weightPercent) ?? 0
        dropLowest = try container.decodeIfPresent(Int.self, forKey: .dropLowest) ?? 0
        dropHighest = try container.decodeIfPresent(Int.self, forKey: .dropHighest) ?? 0
        replaceLowestWithFinal = try container.decodeIfPresent(Bool.self, forKey: .replaceLowestWithFinal) ?? false
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(name, forKey: .name)
        try container.encode(weightPercent, forKey: .weightPercent)
        try container.encode(dropLowest, forKey: .dropLowest)
        try container.encode(dropHighest, forKey: .dropHighest)
        try container.encode(replaceLowestWithFinal, forKey: .replaceLowestWithFinal)
    }
}

struct RubricLevel: Codable, Hashable {
    var label: String
    var points: Double
    var description: String?
}

struct RubricCriterion: Codable, Hashable, Identifiable {
    var id: String
    var title: String
    var description: String?
    var levels: [RubricLevel]
}

struct RubricDefinition: Codable, Hashable {
    var title: String?
    var criteria: [RubricCriterion]
}

struct GradeComment: Decodable, Identifiable, Hashable {
    var id: String?
    var displayName: String?
    var body: String
    var createdAt: String?

    var resolvedId: String { id ?? body.hashValue.description }
}

/// GET `/courses/{code}/my-grades` (student only).
struct MyGradesResponse: Codable {
    var columns: [GradeColumn]
    var grades: [String: String]
    var displayGrades: [String: String]
    var assignmentGroups: [AssignmentGroup]
    var heldGradeItemIds: [String]
    var droppedGrades: [String: Bool]
    var gradeStatuses: [String: String]

    enum CodingKeys: String, CodingKey {
        case columns, grades, displayGrades, assignmentGroups
        case heldGradeItemIds, droppedGrades, gradeStatuses
    }

    init(
        columns: [GradeColumn] = [],
        grades: [String: String] = [:],
        displayGrades: [String: String] = [:],
        assignmentGroups: [AssignmentGroup] = [],
        heldGradeItemIds: [String] = [],
        droppedGrades: [String: Bool] = [:],
        gradeStatuses: [String: String] = [:]
    ) {
        self.columns = columns
        self.grades = grades
        self.displayGrades = displayGrades
        self.assignmentGroups = assignmentGroups
        self.heldGradeItemIds = heldGradeItemIds
        self.droppedGrades = droppedGrades
        self.gradeStatuses = gradeStatuses
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        columns = try container.decodeIfPresent([GradeColumn].self, forKey: .columns) ?? []
        grades = try container.decodeIfPresent([String: String].self, forKey: .grades) ?? [:]
        displayGrades = try container.decodeIfPresent([String: String].self, forKey: .displayGrades) ?? [:]
        assignmentGroups = (try? container.decodeIfPresent([AssignmentGroup].self, forKey: .assignmentGroups)) ?? []
        heldGradeItemIds = try container.decodeIfPresent([String].self, forKey: .heldGradeItemIds) ?? []
        droppedGrades = try container.decodeIfPresent([String: Bool].self, forKey: .droppedGrades) ?? [:]
        gradeStatuses = try container.decodeIfPresent([String: String].self, forKey: .gradeStatuses) ?? [:]
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(columns, forKey: .columns)
        try container.encode(grades, forKey: .grades)
        try container.encode(displayGrades, forKey: .displayGrades)
        try container.encode(assignmentGroups, forKey: .assignmentGroups)
        try container.encode(heldGradeItemIds, forKey: .heldGradeItemIds)
        try container.encode(droppedGrades, forKey: .droppedGrades)
        try container.encode(gradeStatuses, forKey: .gradeStatuses)
    }
}

// MARK: - Syllabus

struct SyllabusSection: Decodable, Identifiable, Hashable {
    var id: String
    var heading: String
    var markdown: String
}

/// GET `/courses/{code}/syllabus`.
struct SyllabusPayload: Decodable {
    var sections: [SyllabusSection]
    var updatedAt: String?
    var requireSyllabusAcceptance: Bool?
    var syllabusAcceptancePending: Bool?

    var hasContent: Bool {
        sections.contains { !$0.markdown.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }
    }
}

// MARK: - Assignment submissions

/// Row from `/assignments/{item}/submissions` and `/submissions/mine`.
struct AssignmentSubmission: Decodable, Identifiable, Hashable {
    var id: String
    var submittedBy: String?
    var submittedByDisplayName: String?
    var blindLabel: String?
    var attachmentFilename: String?
    var attachmentMimeType: String?
    var attachmentContentPath: String?
    var bodyText: String?
    var submittedAt: String
    var updatedAt: String?
    var versionNumber: Int?
    var resubmissionRequested: Bool?
    var revisionDueAt: String?
    var revisionFeedback: String?
    var isGraded: Bool?

    /// Name shown in staff lists; respects blind grading.
    var displayName: String {
        if let blind = blindLabel, !blind.isEmpty { return blind }
        if let name = submittedByDisplayName, !name.isEmpty { return name }
        return "Student"
    }
}

struct MySubmissionResponse: Decodable {
    var submission: AssignmentSubmission?
}

struct SubmissionsListResponse: Decodable {
    var submissions: [AssignmentSubmission]

    enum CodingKeys: String, CodingKey { case submissions }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        // The list endpoint returns a roster row for every enrolled student, including
        // non-submitters that lack `id`/`submittedAt`. Decode leniently and keep only
        // real submissions so one placeholder doesn't fail the whole list.
        let rows = try container.decodeIfPresent([LossyDecodable<AssignmentSubmission>].self, forKey: .submissions) ?? []
        submissions = rows.compactMap(\.value)
    }
}

/// Decodes `T` if possible, otherwise yields `nil` without throwing — used to skip
/// malformed/placeholder elements inside an array without failing the whole decode.
struct LossyDecodable<T: Decodable>: Decodable {
    let value: T?

    init(from decoder: Decoder) throws {
        value = try? T(from: decoder)
    }
}

/// GET/PUT `.../submissions/{id}/grade`.
struct SubmissionGrade: Decodable {
    var submissionId: String?
    var pointsEarned: Double?
    var maxPoints: Double?
    var rubricScores: [String: Double]?
    var instructorComment: String?
    var comments: [GradeComment]?
    var posted: Bool?
    var excused: Bool?
    var gradedByAi: Bool?
}

struct SubmissionAnnotation: Decodable, Identifiable, Hashable {
    var id: String
    var submissionId: String?
    var page: Int
    var toolType: String
    var colour: String
    var coordsJson: AnnotationCoords?
    var body: String?
    var createdAt: String?
}

/// Loose JSON holder for annotation geometry (highlight rects, draw paths, pins).
struct AnnotationCoords: Decodable, Hashable {
    var x1: Double?
    var y1: Double?
    var x2: Double?
    var y2: Double?
    var x: Double?
    var y: Double?
    var points: [AnnotationPoint]?
    var rects: [AnnotationRect]?
}

struct AnnotationPoint: Decodable, Hashable {
    var x: Double
    var y: Double
}

struct AnnotationRect: Decodable, Hashable {
    var x1: Double
    var y1: Double
    var x2: Double
    var y2: Double
}

struct SubmissionFeedbackMedia: Decodable, Identifiable, Hashable {
    var id: String
    var mediaType: String
    var mimeType: String
    var durationSecs: Double?
    var contentPath: String
    var createdAt: String?
}

struct SubmissionAnnotationsResponse: Decodable {
    var annotations: [SubmissionAnnotation]
}

struct SubmissionFeedbackMediaResponse: Decodable {
    var items: [SubmissionFeedbackMedia]
}

struct FeedbackPlaybackInfo: Decodable {
    var contentPath: String
    var captionPath: String?
    var expiresAt: String?
}

/// GET `/api/v1/platform/features` (subset used on mobile).
struct PlatformFeatures: Decodable {
    var ffWhatifGrades: Bool?
    var feedbackMediaEnabled: Bool?
}

/// Navigation target for grade feedback detail (M6.1).
struct GradeFeedbackRoute: Hashable {
    var column: GradeColumn
}

// MARK: - Quiz attempts (staff)

/// Row from GET `/quizzes/{item}/attempts`.
struct QuizAttemptSummary: Decodable, Identifiable, Hashable {
    var id: String
    var studentUserId: String?
    var attemptNumber: Int
    var submittedAt: String
    var scorePercent: Double?
    var pointsEarned: Double
    var pointsPossible: Double
    var studentName: String?
    var needsManualGrading: Bool?
}

struct QuizAttemptsListResponse: Decodable {
    var attempts: [QuizAttemptSummary]
}

// MARK: - Grading backlog (staff)

/// Row from GET `/courses/{code}/grading-backlog`.
struct GradingBacklogItem: Decodable, Identifiable, Hashable {
    var itemId: String?
    var itemType: String? // "assignment" | "quiz"
    var assignmentId: String
    var assignmentTitle: String
    var ungradedCount: Int

    var resolvedItemId: String { itemId ?? assignmentId }
    var isQuiz: Bool { itemType == "quiz" }
    var id: String { "\(itemType ?? "assignment")-\(resolvedItemId)" }
}

enum GradingSubmissionMapper {
    static func quizAttemptsToSubmissions(_ attempts: [QuizAttemptSummary]) -> [AssignmentSubmission] {
        var byStudent: [String: QuizAttemptSummary] = [:]
        for attempt in attempts {
            let key = attempt.studentUserId?.trimmingCharacters(in: .whitespacesAndNewlines).nonEmpty ?? attempt.id
            if let existing = byStudent[key] {
                if attempt.attemptNumber >= existing.attemptNumber {
                    byStudent[key] = attempt
                }
            } else {
                byStudent[key] = attempt
            }
        }
        return byStudent.values
            .sorted { ($0.studentName ?? "") < ($1.studentName ?? "") }
            .map { attempt in
                AssignmentSubmission(
                    id: attempt.id,
                    submittedBy: attempt.studentUserId,
                    submittedByDisplayName: attempt.studentName,
                    blindLabel: nil,
                    attachmentFilename: nil,
                    submittedAt: attempt.submittedAt,
                    updatedAt: nil,
                    versionNumber: attempt.attemptNumber > 1 ? attempt.attemptNumber : nil,
                    resubmissionRequested: nil,
                    revisionDueAt: nil,
                    revisionFeedback: nil,
                    isGraded: attempt.needsManualGrading == false
                )
            }
    }

    static func filterSubmissions(_ submissions: [AssignmentSubmission], graded: String?) -> [AssignmentSubmission] {
        guard let graded, graded != "all" else { return submissions }
        if graded == "graded" {
            return submissions.filter { $0.isGraded == true }
        }
        return submissions.filter { $0.isGraded != true }
    }
}

private extension String {
    var nonEmpty: String? {
        isEmpty ? nil : self
    }

    var nilIfEmpty: String? {
        isEmpty ? nil : self
    }
}

struct GradingBacklogResponse: Decodable {
    var items: [GradingBacklogItem]
}

// MARK: - Module progress & conditional release (M3.1)

struct LockReason: Codable, Hashable {
    var code: String
    var message: String
    var itemId: String?
    var title: String?
}

struct ItemLockState: Codable, Hashable {
    var itemId: String
    var locked: Bool
    var complete: Bool
    var reason: LockReason?
}

struct ModuleLockState: Codable, Hashable {
    var moduleId: String
    var title: String
    var sortOrder: Int
    var locked: Bool
    var complete: Bool
    var reason: LockReason?
    var items: [ItemLockState]?
}

struct ModulesProgressSnapshot: Codable, Hashable {
    var enrollmentId: String
    var modules: [ModuleLockState]

    enum CodingKeys: String, CodingKey { case enrollmentId, modules }

    init(enrollmentId: String = "", modules: [ModuleLockState] = []) {
        self.enrollmentId = enrollmentId
        self.modules = modules
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        enrollmentId = try container.decodeIfPresent(String.self, forKey: .enrollmentId) ?? ""
        modules = try container.decodeIfPresent([ModuleLockState].self, forKey: .modules) ?? []
    }
}

struct MarkItemCompleteResponse: Decodable {
    var enrollmentId: String?
    var justComplete: Bool?
}

// MARK: - Attendance

/// Session from GET `/courses/{code}/attendance/sessions`.
struct AttendanceSession: Decodable, Identifiable, Hashable {
    var id: String
    var title: String?
    var collectionMethod: String // "roll_call" | "self_report"
    var sessionDate: String?
    var status: String // "open" | "closed"

    var isOpen: Bool { status == "open" }
    var isSelfReport: Bool { collectionMethod == "self_report" }

    var displayTitle: String {
        if let title, !title.isEmpty { return title }
        return "Attendance"
    }
}

struct AttendanceRecord: Decodable, Hashable {
    var studentUserId: String
    var displayName: String?
    var status: String
    var recordedAt: String?
}

/// GET `.../attendance/sessions/{id}` — session plus viewer-specific fields.
struct AttendanceSessionDetail: Decodable {
    var id: String
    var title: String?
    var collectionMethod: String
    var sessionDate: String?
    var status: String
    var records: [AttendanceRecord]?
    var myRecord: AttendanceRecord?
    var canSelfReport: Bool?
}

struct AttendanceSessionsResponse: Decodable {
    var sessions: [AttendanceSession]
}

// MARK: - Office hours (M7.3)

struct AvailabilityWindow: Codable, Identifiable, Hashable {
    var id: String
    var instructorId: String
    var courseId: String?
    var dayOfWeek: Int?
    var windowDate: String?
    var startTime: String
    var endTime: String
    var slotDurationMinutes: Int
    var location: String?
    var isVirtual: Bool
    var status: String
    var createdAt: String?
}

struct AppointmentSlot: Codable, Identifiable, Hashable {
    var id: String
    var windowId: String
    var slotStart: String
    var slotEnd: String
    var studentId: String?
    var studentNote: String?
    var meetingId: String?
    var status: String
    var bookedAt: String?
}

struct OfficeHoursAvailability: Codable, Hashable {
    var windows: [AvailabilityWindow]
    var slots: [AppointmentSlot]
}

struct OfficeHoursAvailabilityResponse: Decodable {
    var windows: [AvailabilityWindow]?
    var slots: [AppointmentSlot]?
}

struct MyAppointmentsResponse: Decodable {
    var appointments: [AppointmentSlot]?
}

struct BookOfficeHoursSlotBody: Encodable {
    var note: String?
}

struct MeetingJoinResponse: Decodable {
    var joinUrl: String?
    var hostUrl: String?
}

// MARK: - Planner (todos + calendar, M2.1)

/// Row from GET `/api/v1/me/notebook-tasks`.
struct NotebookTask: Decodable, Identifiable, Hashable {
    var id: String
    var courseCode: String
    var notebookPageId: String
    var taskText: String
    var completed: Bool
    var dueAt: String?
    var createdAt: String?
    var updatedAt: String?
}

struct NotebookTasksResponse: Decodable {
    var tasks: [NotebookTask]
}

struct CalendarCourseFeed: Decodable, Hashable {
    var courseId: String
    var courseCode: String
    var title: String
    var feedUrl: String
}

struct CalendarTokenStatus: Decodable {
    var hasToken: Bool?
}

struct CalendarTokenInfo: Decodable {
    var hasToken: Bool?
    var personalFeedUrl: String?
    var expiresAt: String?
    var courseFeeds: [CalendarCourseFeed]?
}

struct CalendarTokenCreated: Decodable {
    var token: String
    var feedUrl: String?
    var expiresAt: String?
}

struct AcademicCalendarEvent: Decodable, Identifiable, Hashable {
    var id: String
    var orgId: String
    var termId: String?
    var eventType: String
    var eventName: String
    var startDate: String
    var endDate: String?
    var allDay: Bool
    var notes: String?
}

struct AcademicCalendarEventsResponse: Decodable {
    var events: [AcademicCalendarEvent]
}

// MARK: - Course files (M3.2)

struct CourseFileFolder: Codable, Identifiable, Hashable {
    var id: String
    var courseId: String
    var parentId: String?
    var name: String
    var createdAt: String
    var updatedAt: String
}

struct CourseFileItem: Codable, Identifiable, Hashable {
    var id: String
    var courseId: String
    var folderId: String?
    var storageKey: String
    var originalFilename: String
    var displayName: String
    var mimeType: String
    var byteSize: Int64
    var createdAt: String
    var updatedAt: String

    var title: String {
        let name = displayName.trimmingCharacters(in: .whitespacesAndNewlines)
        return name.isEmpty ? originalFilename : name
    }
}

struct CourseFileBreadcrumb: Codable, Identifiable, Hashable {
    var id: String
    var name: String
}

struct CourseFileFolderContents: Codable {
    var folderId: String?
    var breadcrumbs: [CourseFileBreadcrumb]?
    var folders: [CourseFileFolder]
    var files: [CourseFileItem]

    init(
        folderId: String? = nil,
        breadcrumbs: [CourseFileBreadcrumb]? = nil,
        folders: [CourseFileFolder] = [],
        files: [CourseFileItem] = []
    ) {
        self.folderId = folderId
        self.breadcrumbs = breadcrumbs
        self.folders = folders
        self.files = files
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        folderId = try container.decodeIfPresent(String.self, forKey: .folderId)
        breadcrumbs = try container.decodeIfPresent([CourseFileBreadcrumb].self, forKey: .breadcrumbs)
        folders = try container.decodeIfPresent([CourseFileFolder].self, forKey: .folders) ?? []
        files = try container.decodeIfPresent([CourseFileItem].self, forKey: .files) ?? []
    }
}

/// Identifies a previewable file from the file manager or legacy course-files store.
enum CourseFileContentSource: Hashable {
    case fileManager(itemId: String)
    case courseFile(fileId: String)
    /// Absolute API path from submission `attachmentContentPath`.
    case directPath(String)
}

/// Reusable preview target for M3.2, module file items (M3.1), and submission attachments (M5.1).
struct FilePreviewTarget: Hashable, Identifiable {
    var id: String { sourceKey }
    let courseCode: String
    let displayName: String
    let mimeType: String?
    let byteSize: Int64?
    let source: CourseFileContentSource

    var sourceKey: String {
        switch source {
        case .fileManager(let itemId): return "fm:\(itemId)"
        case .courseFile(let fileId): return "cf:\(fileId)"
        case .directPath(let path): return "dp:\(path.hashValue)"
        }
    }

    static func from(file item: CourseFileItem, courseCode: String) -> FilePreviewTarget {
        FilePreviewTarget(
            courseCode: courseCode,
            displayName: item.title,
            mimeType: item.mimeType,
            byteSize: item.byteSize,
            source: .fileManager(itemId: item.id)
        )
    }

    static func from(moduleItem item: CourseStructureItem, courseCode: String) -> FilePreviewTarget {
        FilePreviewTarget(
            courseCode: courseCode,
            displayName: item.title,
            mimeType: CourseFileLogic.guessMimeType(from: item.title),
            byteSize: nil,
            source: .fileManager(itemId: item.id)
        )
    }

    static func submissionAttachment(
        courseCode: String,
        fileId: String,
        fileName: String,
        mimeType: String?
    ) -> FilePreviewTarget {
        FilePreviewTarget(
            courseCode: courseCode,
            displayName: fileName,
            mimeType: mimeType,
            byteSize: nil,
            source: .courseFile(fileId: fileId)
        )
    }

    static func submissionContentPath(
        courseCode: String,
        contentPath: String,
        fileName: String,
        mimeType: String?
    ) -> FilePreviewTarget {
        FilePreviewTarget(
            courseCode: courseCode,
            displayName: fileName,
            mimeType: mimeType,
            byteSize: nil,
            source: .directPath(contentPath)
        )
    }
}

enum FilePreviewKind: Equatable {
    case image
    case pdf
    case audio
    case video
    case downloadOnly
}

// MARK: - Interactive content (M3.3)

struct ModuleH5PPayload: Decodable {
    var packageId: String
    var itemId: String?
    var title: String
    var contentType: String?
    var extractStatus: String
    var assetsBaseUrl: String?
    var downloadUrl: String?
}

struct ModuleScormSco: Decodable {
    var id: String
    var identifier: String?
    var title: String?
    var launchHref: String?
}

struct ModuleScormPayload: Decodable {
    var packageId: String
    var itemId: String?
    var title: String
    var packageType: String?
    var extractStatus: String
    var assetsBaseUrl: String?
    var downloadUrl: String?
    var scos: [ModuleScormSco]
}

struct ScormLaunchResponse: Decodable {
    var registrationId: String
    var launchUrl: String?
    var renderUrl: String
    var initialCmi: [String: String]?

    enum CodingKeys: String, CodingKey {
        case registrationId, launchUrl, renderUrl, initialCmi
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        registrationId = try container.decodeIfPresent(String.self, forKey: .registrationId) ?? ""
        launchUrl = try container.decodeIfPresent(String.self, forKey: .launchUrl)
        renderUrl = try container.decodeIfPresent(String.self, forKey: .renderUrl) ?? ""
        initialCmi = try container.decodeIfPresent([String: String].self, forKey: .initialCmi)
    }
}

struct ModuleLtiLinkPayload: Decodable {
    var itemId: String
    var title: String
    var externalToolId: String?
    var externalToolName: String?
    var resourceLinkId: String?
    var lineItemUrl: String?
}

struct LtiEmbedTicketResponse: Decodable {
    var ticket: String
}

struct ModuleVibeActivityPayload: Decodable {
    var id: String
    var title: String
    var html: String?
    var published: Bool?
    var archived: Bool?
}

/// Human label + tint key for an attendance status string.
enum AttendanceStatusInfo {
    static func label(_ status: String) -> String {
        switch status {
        case "present": return "Present"
        case "absent": return "Absent"
        case "tardy": return "Tardy"
        case "excused": return "Excused"
        case "not_recorded": return "Not recorded"
        default: return status.replacingOccurrences(of: "_", with: " ").capitalized
        }
    }
}
