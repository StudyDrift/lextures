import Foundation

/// Freemium gate for the hosted self-learner environment (`self.lextures.com`).
/// When a signed-in learner has no active subscription, the app presents a full-screen
/// subscribe prompt after a short delay (StoreKit).
enum HomeschoolSubscribeGateLogic {
    /// Delay after eligibility is confirmed before presenting the paywall.
    static let presentationDelaySeconds: TimeInterval = 5

    /// True when the selected environment is the public self-learner host.
    static func isSelfLearnerHost(
        kind: EnvironmentStore.Kind?,
        apiBaseURLString: String?
    ) -> Bool {
        if kind == .homeschool { return true }
        guard let raw = apiBaseURLString?.lowercased() else { return false }
        return raw.contains("self.lextures.com")
    }

    /// Gate only applies on self.lextures.com when billing is enabled and no sub is active.
    static func isEligible(
        isSelfLearnerHost: Bool,
        billingEnabled: Bool,
        hasActiveSubscription: Bool
    ) -> Bool {
        isSelfLearnerHost && billingEnabled && !hasActiveSubscription
    }

    /// Whether the full-screen subscribe UI should appear.
    static func shouldPresentPaywall(
        eligible: Bool,
        delayElapsed: Bool,
        dismissedForSession: Bool
    ) -> Bool {
        eligible && delayElapsed && !dismissedForSession
    }
}
