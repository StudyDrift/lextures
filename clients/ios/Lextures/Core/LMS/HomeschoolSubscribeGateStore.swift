import Foundation
import Observation

/// Drives the self.lextures.com subscribe paywall after a short delay when the
/// signed-in learner has no active subscription.
@MainActor
@Observable
final class HomeschoolSubscribeGateStore {
    private(set) var hasActiveSubscription = false
    private(set) var subscriptionChecked = false
    /// When true, paywall stays hidden for the rest of this app session.
    private(set) var dismissedForCurrentPrompt = false
    var showPaywall = false

    private var userKey: String?
    private var paywallTask: Task<Void, Never>?

    init() {}

    func configure(userId: String?) {
        let key = userId.flatMap { $0.isEmpty ? nil : $0 } ?? "anonymous"
        if userKey == key { return }
        userKey = key
        cancelScheduledPaywall()
        dismissedForCurrentPrompt = false
        showPaywall = false
        hasActiveSubscription = false
        subscriptionChecked = false
    }

    func setHasActiveSubscription(_ active: Bool) {
        hasActiveSubscription = active
        subscriptionChecked = true
        if active {
            cancelScheduledPaywall()
            showPaywall = false
            dismissedForCurrentPrompt = false
        }
    }

    /// Start or cancel the delayed paywall based on host + billing + subscription state.
    func considerSchedulingPaywall(
        environmentKind: EnvironmentStore.Kind?,
        apiBaseURLString: String?,
        billingEnabled: Bool
    ) {
        let host = HomeschoolSubscribeGateLogic.isSelfLearnerHost(
            kind: environmentKind,
            apiBaseURLString: apiBaseURLString
        )
        let eligible = HomeschoolSubscribeGateLogic.isEligible(
            isSelfLearnerHost: host,
            billingEnabled: billingEnabled,
            hasActiveSubscription: hasActiveSubscription
        )

        guard subscriptionChecked else { return }

        if !eligible {
            cancelScheduledPaywall()
            showPaywall = false
            return
        }

        if dismissedForCurrentPrompt || showPaywall || paywallTask != nil {
            return
        }

        scheduleDelayedPaywall()
    }

    func dismissPaywallForNow() {
        dismissedForCurrentPrompt = true
        showPaywall = false
        cancelScheduledPaywall()
    }

    func markSubscribed() {
        hasActiveSubscription = true
        subscriptionChecked = true
        showPaywall = false
        dismissedForCurrentPrompt = false
        cancelScheduledPaywall()
    }

    private func scheduleDelayedPaywall() {
        cancelScheduledPaywall()
        let delay = HomeschoolSubscribeGateLogic.presentationDelaySeconds
        paywallTask = Task { @MainActor [weak self] in
            let nanos = UInt64(max(delay, 0) * 1_000_000_000)
            try? await Task.sleep(nanoseconds: nanos)
            guard let self, !Task.isCancelled else { return }
            // Re-check subscription / dismiss; host + billing were required to schedule.
            if HomeschoolSubscribeGateLogic.shouldPresentPaywall(
                eligible: !self.hasActiveSubscription,
                delayElapsed: true,
                dismissedForSession: self.dismissedForCurrentPrompt
            ) {
                self.showPaywall = true
            }
        }
    }

    private func cancelScheduledPaywall() {
        paywallTask?.cancel()
        paywallTask = nil
    }
}
