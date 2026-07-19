import Foundation

/// Lightweight client-only course-create funnel counters (MOB.1; no PII).
enum CourseCreateObservability {
    private static let defaults = UserDefaults.standard

    static func recordStarted(mode: String, templateId: String) {
        bump("course_create_started")
        defaults.set(mode, forKey: "course_create_started.last_mode")
        defaults.set(templateId, forKey: "course_create_started.last_template")
    }

    static func recordStepCompleted(step: Int) {
        bump("course_create_step_completed")
        defaults.set(step, forKey: "course_create_step_completed.last_step")
    }

    static func recordFinished(mode: String, templateId: String) {
        bump("course_create_finished")
        defaults.set(mode, forKey: "course_create_finished.last_mode")
        defaults.set(templateId, forKey: "course_create_finished.last_template")
    }

    private static func bump(_ key: String) {
        let prev = defaults.integer(forKey: key)
        defaults.set(prev + 1, forKey: key)
    }
}
