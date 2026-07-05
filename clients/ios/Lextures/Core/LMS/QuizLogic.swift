import Foundation

// MARK: - Quiz delivery models (M4.1)

struct ModuleQuizPayload: Codable {
    var itemId: String?
    var title: String?
    var markdown: String?
    var dueAt: String?
    var unlimitedAttempts: Bool?
    var maxAttempts: Int?
    var gradeAttemptPolicy: String?
    var passingScorePercent: Int?
    var pointsWorth: Int?
    var timeLimitMinutes: Int?
    var oneQuestionAtATime: Bool?
    var lockdownMode: String?
    var shuffleQuestions: Bool?
    var shuffleChoices: Bool?
    var allowBackNavigation: Bool?
    var requiresQuizAccessCode: Bool?
    var randomQuestionPoolCount: Int?
    var questions: [QuizQuestion]?
    var usesServerQuestionSampling: Bool?
    var isAdaptive: Bool?
    var showScoreTiming: String?
    var reviewVisibility: String?
    var reviewWhen: String?

    var questionCount: Int { questions?.count ?? 0 }
}

struct QuizQuestion: Codable, Identifiable, Hashable {
    var id: String
    var prompt: String
    var questionType: String
    var choices: [String]?
    var choiceIds: [String]?
    var typeConfig: QuizTypeConfig?
    var correctChoiceIndex: Int?
    var multipleAnswer: Bool?
    var answerWithImage: Bool?
    var required: Bool?
    var points: Int?
    var estimatedMinutes: Int?
}

struct QuizTypeConfig: Codable, Hashable {
    var items: [String]?
    var pairs: [QuizMatchingPairConfig]?
    var starterCode: String?
    var language: String?
    var languageId: Int?
    var multiFile: Bool?
    var files: [QuizCodeFileConfig]?
    var testCases: [QuizCodeTestCaseConfig]?
}

struct QuizCodeFileConfig: Codable, Hashable {
    var path: String?
    var content: String?
}

struct QuizCodeTestCaseConfig: Codable, Hashable {
    var id: String?
    var input: String?
    var expectedOutput: String?
    var isHidden: Bool?
}

struct QuizMatchingPairConfig: Codable, Hashable {
    var leftId: String?
    var rightId: String?
    var left: String?
    var right: String?
}

struct QuizStartRequest: Encodable {
    var quizAccessCode: String?
}

struct QuizStartResponse: Decodable {
    var attemptId: String
    var attemptNumber: Int
    var startedAt: String
    var lockdownMode: String
    var hintsDisabled: Bool?
    var backNavigationAllowed: Bool?
    var currentQuestionIndex: Int
    var deadlineAt: String?
    var maxAttempts: Int?
    var remainingAttempts: Int?
    var retakePolicy: String?
}

struct QuizCurrentQuestionResponse: Decodable {
    var question: QuizQuestion?
    var questionIndex: Int
    var totalQuestions: Int
    var completed: Bool
}

struct QuizAdvanceResponse: Decodable {
    var locked: Bool
    var currentQuestionIndex: Int
    var completed: Bool
}

struct QuizMatchingPairResponse: Codable, Hashable {
    var leftId: String
    var rightId: String
}

struct QuizCodeSubmission: Codable, Hashable {
    var language: String
    var code: String
}

struct QuizHotspotClick: Codable, Hashable {
    var normalizedX: Double
    var normalizedY: Double

    enum CodingKeys: String, CodingKey {
        case normalizedX = "x"
        case normalizedY = "y"
    }
}

struct QuizQuestionResponseItem: Codable {
    var questionId: String
    var selectedChoiceIndex: Int?
    var selectedChoiceIndices: [Int]?
    var textAnswer: String?
    var matchingPairs: [QuizMatchingPairResponse]?
    var orderingSequence: [String]?
    var hotspotClick: QuizHotspotClick?
    var numericValue: Double?
    var formulaLatex: String?
    var codeSubmission: QuizCodeSubmission?
    var fileKey: String?
}

struct QuizSubmitRequest: Encodable {
    var attemptId: String
    var responses: [QuizQuestionResponseItem]?
}

struct QuizSubmitResponse: Decodable {
    var attemptId: String
    var pointsEarned: Double
    var pointsPossible: Double
    var scorePercent: Double
}

struct QuizResultsScoreSummary: Decodable {
    var pointsEarned: Double
    var pointsPossible: Double
    var scorePercent: Double
}

struct QuizResultsQuestionResult: Decodable, Identifiable {
    var questionIndex: Int
    var questionId: String?
    var questionType: String
    var promptSnapshot: String?
    var isCorrect: Bool?
    var pointsAwarded: Double?
    var maxPoints: Double

    var id: String { questionId ?? "q-\(questionIndex)" }
}

struct QuizResultsResponse: Decodable {
    var attemptId: String
    var attemptNumber: Int
    var status: String
    var submittedAt: String?
    var academicIntegrityFlag: Bool?
    var needsManualGrading: Bool?
    var score: QuizResultsScoreSummary?
    var questions: [QuizResultsQuestionResult]?
}

struct QuizFocusLossRequest: Encodable {
    var eventType: String
    var durationMs: Int?
}

struct QuizCodeRunRequest: Encodable {
    var code: String
    var languageId: Int?
}

struct QuizCodeRunResult: Codable, Hashable, Identifiable {
    var status: String
    var passed: Bool
    var actualOutput: String
    var expectedOutput: String
    var stderr: String?
    var executionMs: Int?
    var memoryKb: Int?

    var id: String { "\(status)-\(expectedOutput)-\(actualOutput)" }
}

struct QuizCodeRunResponse: Codable, Hashable {
    var questionId: String
    var results: [QuizCodeRunResult]
    var pointsEarned: Double
    var pointsPossible: Double
}

// MARK: - Local answer state

struct QuizAnswerState: Equatable {
    var choice: Int?
    var choices: Set<Int>?
    var text: String?
    var numeric: Double?
    var ordering: [String]?
    var matching: [String: String]?
    var hotspot: QuizHotspotClick?
}

enum QuizSaveState: Equatable {
    case idle
    case saving
    case saved
    case failed
    case queued
}

enum QuizQuestionKind: String {
    case multipleChoice = "multiple_choice"
    case fillInBlank = "fill_in_blank"
    case essay
    case trueFalse = "true_false"
    case shortAnswer = "short_answer"
    case matching
    case ordering
    case hotspot
    case numeric
    case formula
    case code
    case fileUpload = "file_upload"
    case audioResponse = "audio_response"
    case videoResponse = "video_response"
    case unknown

    init(raw: String) {
        self = QuizQuestionKind(rawValue: raw) ?? .unknown
    }

    var supportsMobileInput: Bool {
        switch self {
        case .audioResponse, .videoResponse, .hotspot, .unknown:
            return false
        default:
            return true
        }
    }
}

struct QuizMatchingPairDraft: Equatable {
    var leftId: String
    var rightId: String?
    var left: String
    var right: String?
}

// MARK: - Business logic

enum QuizLogic {
    static func isServerLockdown(_ mode: String?) -> Bool {
        mode == "one_at_a_time" || mode == "kiosk"
    }

    static func visibleChoices(_ question: QuizQuestion) -> [String] {
        (question.choices ?? [])
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
    }

    static func orderingItems(_ question: QuizQuestion) -> [String] {
        if let configured = question.typeConfig?.items {
            let items = configured
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .filter { !$0.isEmpty }
            if !items.isEmpty { return items }
        }
        return visibleChoices(question)
    }

    static func matchingPairs(_ question: QuizQuestion) -> [QuizMatchingPairDraft] {
        guard let configured = question.typeConfig?.pairs else { return [] }
        return configured.enumerated().compactMap { index, pair in
            let left = (pair.left ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
            let right = (pair.right ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
            guard !left.isEmpty || !right.isEmpty else { return nil }
            let leftId = pair.leftId ?? "left-\(index)"
            return QuizMatchingPairDraft(leftId: leftId, rightId: pair.rightId, left: left, right: pair.right)
        }
    }

    static func sortedRightOptions(for pairs: [QuizMatchingPairDraft]) -> [String] {
        Array(Set(pairs.compactMap { pair in
            let value = (pair.right ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
            return value.isEmpty ? nil : value
        })).sorted()
    }

    static func buildMatchingPairsPayload(
        question: QuizQuestion,
        answer: QuizAnswerState?
    ) -> [QuizMatchingPairResponse] {
        let pairs = matchingPairs(question)
        var out: [QuizMatchingPairResponse] = []
        for (index, pair) in pairs.enumerated() {
            let key = pair.leftId
            let selectedRight = (answer?.matching?[key] ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
            guard !selectedRight.isEmpty else { continue }
            let match = pairs.first { ($0.right ?? "").trimmingCharacters(in: .whitespacesAndNewlines) == selectedRight }
            let rightId = match?.rightId ?? match?.leftId
            let leftId = pair.leftId
            if let rightId, !leftId.isEmpty {
                out.append(QuizMatchingPairResponse(leftId: leftId, rightId: rightId))
            } else if !leftId.isEmpty {
                out.append(QuizMatchingPairResponse(leftId: leftId, rightId: "right-\(index)"))
            }
        }
        return out
    }

    private static func baseResponseItem(questionId: String) -> QuizQuestionResponseItem {
        QuizQuestionResponseItem(
            questionId: questionId,
            selectedChoiceIndex: nil,
            selectedChoiceIndices: nil,
            textAnswer: nil,
            matchingPairs: nil,
            orderingSequence: nil,
            hotspotClick: nil,
            numericValue: nil,
            formulaLatex: nil,
            codeSubmission: nil,
            fileKey: nil
        )
    }

    static func buildResponseItem(
        question: QuizQuestion,
        answer: QuizAnswerState?
    ) -> QuizQuestionResponseItem {
        let kind = QuizQuestionKind(raw: question.questionType)
        let questionId = question.id
        switch kind {
        case .multipleChoice, .trueFalse:
            if question.multipleAnswer == true, let selected = answer?.choices {
                var item = baseResponseItem(questionId: questionId)
                item.selectedChoiceIndices = selected.sorted()
                return item
            }
            var item = baseResponseItem(questionId: questionId)
            item.selectedChoiceIndex = answer?.choice
            return item
        case .numeric:
            var item = baseResponseItem(questionId: questionId)
            item.textAnswer = answer?.text
            item.numericValue = answer?.numeric
            return item
        case .formula:
            var item = baseResponseItem(questionId: questionId)
            item.formulaLatex = answer?.text
            return item
        case .ordering:
            var item = baseResponseItem(questionId: questionId)
            item.orderingSequence = answer?.ordering ?? orderingItems(question)
            return item
        case .matching:
            var item = baseResponseItem(questionId: questionId)
            item.matchingPairs = buildMatchingPairsPayload(question: question, answer: answer)
            return item
        case .hotspot:
            var item = baseResponseItem(questionId: questionId)
            item.hotspotClick = answer?.hotspot
            return item
        case .code:
            var item = baseResponseItem(questionId: questionId)
            item.codeSubmission = QuizCodeSubmission(
                language: question.typeConfig?.language ?? "text",
                code: answer?.text ?? ""
            )
            return item
        case .fileUpload, .audioResponse, .videoResponse:
            var item = baseResponseItem(questionId: questionId)
            item.textAnswer = answer?.text
            item.fileKey = answer?.text
            return item
        default:
            var item = baseResponseItem(questionId: questionId)
            item.textAnswer = answer?.text
            return item
        }
    }

    static func isAnswered(question: QuizQuestion, answer: QuizAnswerState?) -> Bool {
        guard let answer else { return false }
        let kind = QuizQuestionKind(raw: question.questionType)
        switch kind {
        case .multipleChoice, .trueFalse:
            if question.multipleAnswer == true {
                return !(answer.choices?.isEmpty ?? true)
            }
            return answer.choice != nil
        case .numeric:
            return answer.numeric != nil || !(answer.text?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ?? true)
        case .formula, .fillInBlank, .shortAnswer, .essay:
            return !(answer.text?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ?? true)
        case .ordering:
            return !(answer.ordering?.isEmpty ?? true)
        case .matching:
            return !(answer.matching?.isEmpty ?? true)
        case .hotspot:
            return answer.hotspot != nil
        case .fileUpload, .audioResponse, .videoResponse:
            return !(answer.text?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ?? true)
        case .code:
            return !(answer.text?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ?? true)
        case .unknown:
            return false
        }
    }

    static func secondsRemaining(deadlineISO: String?, now: Date = Date()) -> Int? {
        guard let deadlineISO, let deadline = LMSDates.parse(deadlineISO) else { return nil }
        return max(0, Int(deadline.timeIntervalSince(now)))
    }

    static func formatTimer(_ seconds: Int) -> String {
        let minutes = seconds / 60
        let secs = seconds % 60
        return String(format: "%d:%02d", minutes, secs)
    }

    static func retakePolicyNotice(_ policy: String?) -> String {
        switch policy {
        case "highest":
            return "Your highest score counts toward the course grade."
        case "latest":
            return "Your most recent attempt counts toward the course grade."
        case "first":
            return "Your first submitted attempt counts toward the course grade."
        case "average":
            return "The average of your attempts counts toward the course grade."
        default:
            return "Your instructor chose how multiple attempts are scored."
        }
    }

    static func attemptsUsedLabel(
        unlimited: Bool,
        maxAttempts: Int?,
        remaining: Int?,
        pastAttempts: Int
    ) -> String {
        if unlimited { return "Unlimited attempts" }
        let limit = maxAttempts ?? 1
        let used = limit - (remaining ?? Swift.max(0, limit - pastAttempts))
        return "\(used) of \(limit) attempts used"
    }

    static func canStart(
        unlimited: Bool,
        remaining: Int?,
        hasInProgress: Bool
    ) -> Bool {
        if hasInProgress { return true }
        if unlimited { return true }
        return (remaining ?? 0) > 0
    }

    // MARK: - Code questions (M5.3)

    static let codeMobileMaxBytes = 16_384

    static let codeSymbolSnippets = [
        "{", "}", "(", ")", "[", "]", ";", ":", ".", ",", "=", "+", "-", "*", "/", "<", ">", "\"", "'",
        "    ", "\n",
    ]

    static func starterCode(for question: QuizQuestion) -> String {
        let starter = question.typeConfig?.starterCode?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return starter
    }

    static func codeLanguageLabel(for question: QuizQuestion) -> String {
        let language = question.typeConfig?.language?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if language.isEmpty { return "text" }
        return language
    }

    static func isCodeQuestionOversized(_ question: QuizQuestion) -> Bool {
        if question.typeConfig?.multiFile == true { return true }
        let files = question.typeConfig?.files ?? []
        if files.count > 1 { return true }
        let starter = starterCode(for: question)
        if starter.utf8.count > codeMobileMaxBytes { return true }
        return false
    }

    static func initialCodeAnswer(for question: QuizQuestion, existing: QuizAnswerState?) -> QuizAnswerState {
        if let text = existing?.text, !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return existing ?? QuizAnswerState()
        }
        var next = existing ?? QuizAnswerState()
        let starter = starterCode(for: question)
        if !starter.isEmpty {
            next.text = starter
        }
        return next
    }

    static func applyAutoIndent(to text: String) -> String {
        guard text.hasSuffix("\n") else { return text }
        let lines = text.split(separator: "\n", omittingEmptySubsequences: false)
        guard let previous = lines.dropLast().last else { return text }
        let trimmed = previous.trimmingCharacters(in: .whitespaces)
        var indent = ""
        for ch in previous {
            if ch == " " || ch == "\t" { indent.append(ch) } else { break }
        }
        if trimmed.hasSuffix("{") || trimmed.hasSuffix(":") {
            indent += "    "
        }
        return text + indent
    }
}
