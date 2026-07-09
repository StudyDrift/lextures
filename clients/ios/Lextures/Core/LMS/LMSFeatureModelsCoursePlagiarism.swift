import Foundation

/// Course plagiarism settings models (M13.7).
struct CoursePlagiarismSettings: Codable, Hashable {
    var plagiarismChecksEnabled: Bool
    var plagiarismProvider: String?
    var plagiarismAlertThresholdPct: Double
}

struct PatchCoursePlagiarismBody: Encodable {
    var plagiarismChecksEnabled: Bool
    var plagiarismProvider: String?
    var plagiarismAlertThresholdPct: Double
}
