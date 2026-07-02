import Foundation

/// GET `/api/v1/me/permissions`.
struct MyPermissionsResponse: Decodable {
    var permissionStrings: [String]

    enum CodingKeys: String, CodingKey { case permissionStrings }
}

// MARK: - Library & OER (M3.6)

struct LibraryCatalogResult: Decodable, Identifiable, Hashable {
    var mmsId: String?
    var title: String
    var author: String?
    var issn: String?
    var isbn: String?

    var id: String { mmsId ?? "\(title)-\(author ?? "")" }
}

struct LibrarySearchResponse: Decodable {
    var results: [LibraryCatalogResult]
}

struct LibraryResourceMeta: Decodable, Hashable {
    var title: String?
    var author: String?
    var issn: String?
    var isbn: String?
    var source: String?
    var almaMmsId: String?
    var legantoListId: String?
    var ezproxyUrl: String?
}

struct LibraryResourcePayload: Decodable, Hashable {
    var itemId: String
    var resourceType: String
    var metadata: LibraryResourceMeta?
    var ezproxyUrl: String?
    var updatedAt: String?
}

struct OERSearchResult: Decodable, Identifiable, Hashable {
    var id: String
    var title: String
    var description: String?
    var url: String
    var previewUrl: String?
    var provider: String
    var licenseSpdx: String?
    var licenseLabel: String?
    var gradeLevel: String?
    var subject: String?
    var attribution: String?
}

struct OERSearchResponse: Decodable {
    var results: [OERSearchResult]
    var provider: String?
    var fromCache: Bool?
    var cacheAsOf: String?
    var staleCache: Bool?
}

struct OERProviderRow: Decodable {
    var provider: String
}

// MARK: - Universal search (M0.6)

struct SearchCourseItem: Decodable, Hashable {
    var courseCode: String
    var title: String
}

struct SearchPersonItem: Decodable, Hashable {
    var userId: String
    var email: String
    var displayName: String?
    var role: String
    var courseCode: String
    var courseTitle: String
}

struct SearchIndexResponse: Decodable {
    var courses: [SearchCourseItem]
    var people: [SearchPersonItem]
}

struct SearchQueryResultItem: Decodable, Hashable {
    var id: String
    var type: String
    var title: String
    var subtitle: String
    var path: String
    var score: Double?
}

struct SearchQueryGroup: Decodable, Hashable {
    var type: String
    var label: String
    var total: Int
    var items: [SearchQueryResultItem]
}

struct SearchQueryResponse: Decodable {
    var groups: [SearchQueryGroup]
    var tookMs: Int
}

// MARK: - Take attendance (M11.1)

struct CreateAttendanceSessionBody: Encodable {
    var collectionMethod: String
    var title: String?
    var sessionDate: String?
    var sectionId: String?
}

struct SaveAttendanceRecordsBody: Encodable {
    var records: [AttendanceRecordUpsert]
}

struct SaveAttendanceRecordsResponse: Decodable {
    var saved: Int?
    var message: String?
}

struct CourseSection: Decodable, Identifiable, Hashable {
    var id: String
    var sectionCode: String
    var name: String?

    var displayName: String {
        let trimmed = name?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !trimmed.isEmpty { return trimmed }
        return sectionCode
    }
}

struct CourseSectionsResponse: Decodable {
    var sections: [CourseSection]
}

// MARK: - Course roster (M11.4)

struct CourseEnrollment: Codable, Identifiable, Hashable {
    var id: String
    var userId: String
    var displayName: String?
    var avatarUrl: String?
    var role: String
    var roleDisplay: String?
    var lastCourseAccessAt: String?
    var sectionId: String?
    var sectionCode: String?
    var sectionName: String?
    var state: String?
    var invitationPending: Bool?
}

struct CourseEnrollmentsResponse: Codable {
    var enrollments: [CourseEnrollment]
}

struct EnrollmentMessageBody: Encodable {
    var subject: String
    var body: String
}

struct EnrollmentMessageResponse: Decodable {
    var id: String?
}

/// Staff navigation into take-attendance mode (optional existing session).
struct TakeAttendanceRoute: Hashable {
    var sessionId: String?
}