import Foundation

// MARK: - Course evaluations (M7.7)

enum EvaluationQuestionType: String, Codable, Equatable {
    case rating
    case multipleChoice = "multiple_choice"
    case openText = "open_text"
}

struct EvaluationQuestion: Codable, Equatable, Identifiable {
    var id: String { "\(type.rawValue)-\(text)" }
    let type: EvaluationQuestionType
    let text: String
    let options: [String]?
    let required: Bool?

    var isRequired: Bool { required == true }
}

struct EvaluationStatus: Codable, Equatable {
    let windowOpen: Bool
    let windowId: String?
    let hasSubmitted: Bool
    let opensAt: String?
    let closesAt: String?
    let questions: [EvaluationQuestion]?
}

struct EvaluationSubmitBody: Encodable {
    let answers: [String: String]
}

struct EvaluationSubmitResponse: Decodable {
    let message: String?
}

struct EvaluationQuestionResult: Codable, Equatable, Identifiable {
    var id: Int { index }
    let index: Int
    let type: EvaluationQuestionType
    let text: String
    let average: Double?
    let distribution: [String: Int]?
    let openTexts: [String]?
}

struct EvaluationResults: Codable, Equatable {
    let windowId: String
    let opensAt: String
    let closesAt: String
    let responseCount: Int
    let enrolledCount: Int
    let completionPct: Double
    let meetsThreshold: Bool
    let questions: [EvaluationQuestionResult]
}
