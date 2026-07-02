import Foundation

// MARK: - Reading log & leveled library (M8.4)

struct LibraryBook: Codable, Identifiable, Hashable {
    var id: String
    var orgId: String
    var title: String
    var author: String?
    var coverUrl: String?
    var lexileLevel: Int?
    var fpBand: String?
    var gradeBand: String?
    var summary: String?
}

struct LibraryBooksFilter {
    var lexileMin: Int?
    var lexileMax: Int?
    var gradeBand: String?
}

struct LibraryBooksResponse: Decodable {
    var books: [LibraryBook]?
}

struct ReadingLogEntry: Codable, Identifiable, Hashable {
    var id: String
    var bookId: String?
    var bookTitle: String?
    var logDate: String
    var pagesRead: Int?
    var reflection: String?
    var loggedAt: String?
}

struct ReadingLogListResponse: Decodable {
    var entries: [ReadingLogEntry]?
}

struct PostReadingLogBody: Encodable {
    var bookId: String?
    var bookTitle: String?
    var logDate: String
    var pagesRead: Int?
    var reflection: String?
}

struct PostReadingLogResponse: Decodable {
    var entry: ReadingLogEntry
}

