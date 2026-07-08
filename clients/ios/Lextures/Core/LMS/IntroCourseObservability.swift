import Foundation

/// Lightweight client-only intro course funnel counters (IC07; no PII).
enum IntroCourseObservability {
    private static let defaults = UserDefaults.standard

    static func recordCardView() {
        bump("intro_course.card_view")
    }

    static func recordCtaClick() {
        bump("intro_course.cta_click")
    }

    static func recordCelebrationView() {
        bump("intro_course.completed_celebration_view")
    }

    private static func bump(_ key: String) {
        let prev = defaults.integer(forKey: key)
        defaults.set(prev + 1, forKey: key)
    }
}