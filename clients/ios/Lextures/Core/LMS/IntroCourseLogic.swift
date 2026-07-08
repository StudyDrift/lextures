import Foundation

/// Intro course onboarding helpers (IC07; mirrors web `intro-course-api.ts`).
enum IntroCourseLogic {
    static func introCourseEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.introCourseEnabled
    }

    static func cacheKeyProgress() -> String { "intro-course:progress" }

    static func cardState(
        progress: IntroCourseProgress?,
        loading: Bool,
        error: Bool
    ) -> IntroCourseCardState {
        if loading { return .loading }
        if error || progress == nil { return .error }
        if !progress!.enrolled { return .hidden }
        if progress!.completedAt != nil { return .completed }
        if progress!.modulesComplete <= 0 { return .notStarted }
        return .inProgress
    }

    static func shouldShowCelebration(_ progress: IntroCourseProgress?) -> Bool {
        guard let progress, progress.enrolled, progress.completedAt != nil else { return false }
        return progress.celebrationSeen != true
    }

    static func fallbackRoute(courseCode: String = IntroCourseConstants.courseCode) -> String {
        "/courses/\(courseCode)"
    }

    static func ctaRoute(for progress: IntroCourseProgress) -> String {
        progress.nextItem?.route ?? fallbackRoute(courseCode: progress.courseCode ?? IntroCourseConstants.courseCode)
    }

    static func isIntroCourse(_ courseCode: String) -> Bool {
        courseCode == IntroCourseConstants.courseCode
    }
}