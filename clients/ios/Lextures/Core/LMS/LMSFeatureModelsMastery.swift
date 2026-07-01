import Foundation

struct MasteryConcept: Codable, Identifiable, Hashable {
    var id: String
    var name: String
}

struct MasteryCell: Codable, Hashable {
    var conceptId: String
    var masteryScore: Double?
    var assessed: Bool
    var updatedAt: String?
}

struct StudentMasteryRow: Codable {
    var enrollmentId: String
    var userId: String
    var concepts: [MasteryConcept]
    var cells: [MasteryCell]
}

struct ReportCardSummary: Codable, Identifiable, Hashable {
    var id: String
    var studentId: String
    var courseId: String
    var gradingPeriod: String
    var status: String
    var finalGradePct: Double?
    var letterGrade: String?
    var comment: String?
    var pdfUrl: String?
    var generatedAt: String?
    var releasedAt: String?
    var createdAt: String?
}

struct MyReportCardsResponse: Decodable {
    var reportCards: [ReportCardSummary]
}
