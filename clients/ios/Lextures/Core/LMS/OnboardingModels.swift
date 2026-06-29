import Foundation

enum PriorKnowledgeLevel: String, CaseIterable, Codable {
    case beginner
    case intermediate
    case advanced
}

struct OnboardingStatus: Decodable {
    var completed: Bool
    var step: Int
    var shouldShowFlow: Bool
}

struct LearnerGoals: Decodable {
    var id: String
    var userId: String
    var topic: String
    var goalText: String?
    var targetDate: String?
    var dailyMinutes: Int
    var priorKnowledgeLevel: PriorKnowledgeLevel
    var diagnosticScore: Double?
    var diagnosticSkipped: Bool
    var onboardingStep: Int
    var onboardingCompleted: Bool
    var reminderOptIn: Bool
    var reminderTime: String?
    var recommendedCourseCode: String?
    var recommendedCourseTitle: String?
}

struct DiagnosticQuestion: Decodable, Identifiable, Hashable {
    var id: String
    var prompt: String
    var choices: [String]
}

struct OnboardingTopic: Identifiable {
    var id: String
    var labelKey: String.LocalizationValue

    static let all: [OnboardingTopic] = [
        OnboardingTopic(id: "python", labelKey: "mobile.onboarding.topic.python"),
        OnboardingTopic(id: "javascript", labelKey: "mobile.onboarding.topic.javascript"),
        OnboardingTopic(id: "data-science", labelKey: "mobile.onboarding.topic.dataScience"),
        OnboardingTopic(id: "math", labelKey: "mobile.onboarding.topic.math"),
        OnboardingTopic(id: "business", labelKey: "mobile.onboarding.topic.business"),
        OnboardingTopic(id: "design", labelKey: "mobile.onboarding.topic.design"),
    ]
}

struct GoalsEnvelope: Decodable {
    var goals: LearnerGoals
}

struct DiagnosticQuestionsResponse: Decodable {
    var questions: [DiagnosticQuestion]
}

enum OnboardingStep: Int, CaseIterable {
    case welcome = 0
    case topic = 1
    case experience = 2
    case diagnostic = 3
    case habits = 4
    case consent = 5
    case complete = 6
}
