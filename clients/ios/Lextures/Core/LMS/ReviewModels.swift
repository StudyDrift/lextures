import Foundation

// MARK: - Spaced repetition / review session (M8.1)

enum SrsGrade: String, Codable, CaseIterable, Identifiable {
    case again
    case hard
    case good
    case easy

    var id: String { rawValue }

    var labelKey: String {
        switch self {
        case .again: return "mobile.review.grade.again"
        case .hard: return "mobile.review.grade.hard"
        case .good: return "mobile.review.grade.good"
        case .easy: return "mobile.review.grade.easy"
        }
    }

    var label: String {
        switch self {
        case .again: return L.text("mobile.review.grade.again")
        case .hard: return L.text("mobile.review.grade.hard")
        case .good: return L.text("mobile.review.grade.good")
        case .easy: return L.text("mobile.review.grade.easy")
        }
    }
}

struct ReviewQueueItem: Codable, Identifiable, Hashable {
    var stateId: String
    var questionId: String
    var courseId: String
    var courseCode: String
    var courseTitle: String
    var nextReviewAt: String
    var stem: String
    var questionType: String
    var options: JSONValue?
    var correctAnswer: JSONValue?
    var explanation: String?

    var id: String { questionId }
}

struct ReviewQueueResponse: Codable {
    var items: [ReviewQueueItem]
    var totalDue: Int
}

struct ReviewStats: Codable {
    var streak: Int
    var dueToday: Int
    var dueWeek: Int
    var retentionEstimate: Double
}

struct SrsReviewSubmitBody: Encodable {
    var questionId: String
    var grade: String
    var responseMs: Int?
}

struct SrsReviewSubmitResponse: Decodable {
    var nextReviewAt: String
    var intervalDays: Double
}

struct LearnerRecommendationItem: Codable, Identifiable, Hashable {
    var itemId: String
    var itemType: String
    var title: String
    var surface: String
    var reason: String
    var score: Double

    var id: String { itemId }
}

struct LearnerRecommendationsResponse: Decodable {
    var recommendations: [LearnerRecommendationItem]
    var degraded: Bool?
}

struct ReviewCourseFilter: Identifiable, Hashable {
    var courseCode: String
    var courseTitle: String

    var id: String { courseCode }
}

enum ReviewLogic {
    static let prefetchLimit = 50

    static func formatAnswerPreview(_ value: JSONValue?) -> String {
        guard let value else { return "" }
        switch value {
        case .string(let text):
            return text
        case .number(let number):
            return number.truncatingRemainder(dividingBy: 1) == 0
                ? String(format: "%.0f", number)
                : String(number)
        case .bool(let flag):
            return flag ? "true" : "false"
        case .null:
            return ""
        case .array(let items):
            return items.map { formatAnswerPreview($0) }.filter { !$0.isEmpty }.joined(separator: ", ")
        case .object(let object):
            if let pretty = prettyPrintedJSON(object) {
                return pretty
            }
            return object
                .map { "\($0.key): \(formatAnswerPreview($0.value))" }
                .sorted()
                .joined(separator: "\n")
        }
    }

    static func filterQueue(_ items: [ReviewQueueItem], courseCode: String?) -> [ReviewQueueItem] {
        guard let courseCode, !courseCode.isEmpty else { return items }
        return items.filter { $0.courseCode == courseCode }
    }

    static func courseFilters(from items: [ReviewQueueItem]) -> [ReviewCourseFilter] {
        var seen: [String: String] = [:]
        for item in items {
            if seen[item.courseCode] == nil {
                seen[item.courseCode] = item.courseTitle
            }
        }
        return seen
            .map { ReviewCourseFilter(courseCode: $0.key, courseTitle: $0.value) }
            .sorted { $0.courseTitle.localizedCaseInsensitiveCompare($1.courseTitle) == .orderedAscending }
    }

    static func toQuizQuestion(_ item: ReviewQueueItem) -> QuizQuestion? {
        let kind = QuizQuestionKind(raw: item.questionType)
        guard kind.supportsMobileInput else { return nil }
        let parsed = parseQuestionOptions(item.options, questionType: item.questionType)
        return QuizQuestion(
            id: item.questionId,
            prompt: item.stem,
            questionType: item.questionType,
            choices: parsed.choices,
            choiceIds: parsed.choiceIds,
            typeConfig: parsed.typeConfig,
            correctChoiceIndex: nil,
            multipleAnswer: parsed.multipleAnswer,
            answerWithImage: nil,
            required: nil,
            points: nil,
            estimatedMinutes: nil
        )
    }

    static func idempotencyKey(questionId: String, ratedAt: Date) -> String {
        let ms = Int(ratedAt.timeIntervalSince1970 * 1000)
        return "srs-review:\(questionId):\(ms)"
    }

    // MARK: - Private

    private struct ParsedOptions {
        var choices: [String]?
        var choiceIds: [String]?
        var typeConfig: QuizTypeConfig?
        var multipleAnswer: Bool?
    }

    private static func parseQuestionOptions(_ value: JSONValue?, questionType: String) -> ParsedOptions {
        guard let value else { return ParsedOptions() }
        switch value {
        case .array(let items):
            let choices = items.compactMap { stringValue($0) }
            return ParsedOptions(choices: choices.isEmpty ? nil : choices, choiceIds: nil, typeConfig: nil, multipleAnswer: nil)
        case .object(let object):
            if let choices = jsonArray(object["choices"])?.compactMap({ stringValue($0) }), !choices.isEmpty {
                let ids = jsonArray(object["choiceIds"])?.compactMap { stringValue($0) }
                let multiple = boolValue(object["multipleAnswer"])
                return ParsedOptions(
                    choices: choices,
                    choiceIds: ids?.count == choices.count ? ids : nil,
                    typeConfig: parseTypeConfig(object),
                    multipleAnswer: multiple
                )
            }
            if let items = jsonArray(object["items"])?.compactMap({ stringValue($0) }), !items.isEmpty {
                return ParsedOptions(
                    choices: nil,
                    choiceIds: nil,
                    typeConfig: QuizTypeConfig(items: items, pairs: nil, starterCode: nil, language: nil),
                    multipleAnswer: nil
                )
            }
            if let pairs = parseMatchingPairs(object["pairs"]) {
                return ParsedOptions(
                    choices: nil,
                    choiceIds: nil,
                    typeConfig: QuizTypeConfig(items: nil, pairs: pairs, starterCode: nil, language: nil),
                    multipleAnswer: nil
                )
            }
            return ParsedOptions()
        default:
            if questionType == QuizQuestionKind.trueFalse.rawValue {
                return ParsedOptions(choices: ["True", "False"], choiceIds: nil, typeConfig: nil, multipleAnswer: nil)
            }
            return ParsedOptions()
        }
    }

    private static func parseTypeConfig(_ object: [String: JSONValue]) -> QuizTypeConfig? {
        let items = jsonArray(object["items"])?.compactMap { stringValue($0) }
        let pairs = parseMatchingPairs(object["pairs"])
        let starterCode = stringValue(object["starterCode"])
        let language = stringValue(object["language"])
        if items == nil && pairs == nil && starterCode == nil && language == nil {
            return nil
        }
        return QuizTypeConfig(items: items, pairs: pairs, starterCode: starterCode, language: language)
    }

    private static func parseMatchingPairs(_ value: JSONValue?) -> [QuizMatchingPairConfig]? {
        guard let array = jsonArray(value) else { return nil }
        let pairs = array.compactMap { element -> QuizMatchingPairConfig? in
            guard case .object(let object) = element else { return nil }
            let left = stringValue(object["left"])
            let right = stringValue(object["right"])
            guard left != nil || right != nil else { return nil }
            return QuizMatchingPairConfig(
                leftId: stringValue(object["leftId"]),
                rightId: stringValue(object["rightId"]),
                left: left,
                right: right
            )
        }
        return pairs.isEmpty ? nil : pairs
    }

    private static func jsonArray(_ value: JSONValue?) -> [JSONValue]? {
        guard case .array(let items) = value else { return nil }
        return items
    }

    private static func stringValue(_ value: JSONValue?) -> String? {
        guard let value else { return nil }
        switch value {
        case .string(let text):
            let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
            return trimmed.isEmpty ? nil : trimmed
        case .number(let number):
            return number.truncatingRemainder(dividingBy: 1) == 0
                ? String(format: "%.0f", number)
                : String(number)
        case .bool(let flag):
            return flag ? "true" : "false"
        default:
            return nil
        }
    }

    private static func boolValue(_ value: JSONValue?) -> Bool? {
        guard case .bool(let flag) = value else { return nil }
        return flag
    }

    private static func prettyPrintedJSON(_ object: [String: JSONValue]) -> String? {
        var dict: [String: Any] = [:]
        for (key, value) in object {
            dict[key] = foundationValue(value)
        }
        guard JSONSerialization.isValidJSONObject(dict),
              let data = try? JSONSerialization.data(withJSONObject: dict, options: [.prettyPrinted, .sortedKeys]),
              let text = String(data: data, encoding: .utf8) else {
            return nil
        }
        return text
    }

    private static func foundationValue(_ value: JSONValue) -> Any {
        switch value {
        case .string(let text): return text
        case .number(let number): return number
        case .bool(let flag): return flag
        case .null: return NSNull()
        case .array(let items): return items.map { foundationValue($0) }
        case .object(let object): return object.mapValues { foundationValue($0) }
        }
    }
}
