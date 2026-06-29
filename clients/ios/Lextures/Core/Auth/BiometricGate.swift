import Foundation
import LocalAuthentication
import Observation
import SwiftUI

/// Optional Face ID / Touch ID gate after the app has been backgrounded past a timeout.
@MainActor
@Observable
final class BiometricGate {
    static let shared = BiometricGate()

    /// Background duration before the app locks (seconds).
    static let lockTimeoutSeconds: TimeInterval = 60

    private enum Keys {
        static let enabled = "lextures.biometric.enabled"
    }

    private(set) var isLocked = false
    private var backgroundedAt: Date?

    var isEnabled: Bool {
        get { UserDefaults.standard.bool(forKey: Keys.enabled) }
        set {
            UserDefaults.standard.set(newValue, forKey: Keys.enabled)
            if !newValue {
                isLocked = false
                backgroundedAt = nil
            }
        }
    }

    var canEnableBiometrics: Bool {
        var error: NSError?
        return LAContext().canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error)
    }

    var biometryLabel: String {
        switch LAContext().biometryType {
        case .faceID:
            return L.text("mobile.biometric.faceId")
        case .touchID:
            return L.text("mobile.biometric.touchId")
        default:
            return L.text("mobile.biometric.generic")
        }
    }

    private init() {}

    func recordBackground(at date: Date = Date()) {
        guard isEnabled else { return }
        backgroundedAt = date
    }

    func evaluateOnForeground(now: Date = Date()) {
        guard isEnabled, let backgroundedAt else { return }
        self.backgroundedAt = nil
        if Self.shouldLock(afterBackgroundDuration: now.timeIntervalSince(backgroundedAt)) {
            isLocked = true
        }
    }

    nonisolated static func shouldLock(afterBackgroundDuration duration: TimeInterval) -> Bool {
        duration >= lockTimeoutSeconds
    }

    func unlock() async -> Bool {
        let context = LAContext()
        let reason = L.text("mobile.biometric.reason")
        do {
            let success = try await context.evaluatePolicy(
                .deviceOwnerAuthentication,
                localizedReason: reason
            )
            if success {
                isLocked = false
            }
            return success
        } catch {
            return false
        }
    }

    func resetOnSignOut() {
        isLocked = false
        backgroundedAt = nil
    }
}

private struct BiometricGateKey: EnvironmentKey {
    static let defaultValue = BiometricGate.shared
}

extension EnvironmentValues {
    var biometricGate: BiometricGate {
        get { self[BiometricGateKey.self] }
        set { self[BiometricGateKey.self] = newValue }
    }
}
