import SwiftUI
import UIKit

/// Platform lockdown for kiosk-mode quizzes: Guided Access guidance, screenshot/recording
/// detection, and focus-loss reporting (M4.2).
@MainActor
@Observable
final class LockdownController {
    var focusLossBanner: String?
    var platformWarning: String?
    var isActive = false

    private var onFocusLoss: ((String) -> Void)?
    private var screenshotObserver: NSObjectProtocol?
    private var captureObserver: NSObjectProtocol?
    private var guidedAccessObserver: NSObjectProtocol?
    private var wasGuidedAccessEnabled = false
    private var skipNextForegroundFocusLoss = true

    func activate(onFocusLoss: @escaping (String) -> Void) {
        guard !isActive else { return }
        isActive = true
        self.onFocusLoss = onFocusLoss
        focusLossBanner = nil
        skipNextForegroundFocusLoss = true

        UIApplication.shared.isIdleTimerDisabled = true
        wasGuidedAccessEnabled = UIAccessibility.isGuidedAccessEnabled
        if !wasGuidedAccessEnabled {
            platformWarning = L.text("mobile.quiz.lockdown.guidedAccessWarning")
        }

        let center = NotificationCenter.default
        screenshotObserver = center.addObserver(
            forName: UIApplication.userDidTakeScreenshotNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor in
                self?.reportIntegrityEvent("screenshot")
            }
        }
        captureObserver = center.addObserver(
            forName: UIScreen.capturedDidChangeNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor in
                if UIScreen.main.isCaptured {
                    self?.reportIntegrityEvent("screen_recording")
                }
            }
        }
        guidedAccessObserver = center.addObserver(
            forName: UIAccessibility.guidedAccessStatusDidChangeNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor in
                self?.handleGuidedAccessChange()
            }
        }
    }

    func deactivate() {
        guard isActive else { return }
        isActive = false
        onFocusLoss = nil
        focusLossBanner = nil
        platformWarning = nil
        skipNextForegroundFocusLoss = true
        UIApplication.shared.isIdleTimerDisabled = false

        let center = NotificationCenter.default
        if let screenshotObserver {
            center.removeObserver(screenshotObserver)
            self.screenshotObserver = nil
        }
        if let captureObserver {
            center.removeObserver(captureObserver)
            self.captureObserver = nil
        }
        if let guidedAccessObserver {
            center.removeObserver(guidedAccessObserver)
            self.guidedAccessObserver = nil
        }
    }

    func handleScenePhaseChange(_ phase: ScenePhase) {
        guard isActive else { return }
        switch phase {
        case .background, .inactive:
            reportIntegrityEvent("app_background", showBanner: false)
        case .active:
            if skipNextForegroundFocusLoss {
                skipNextForegroundFocusLoss = false
                return
            }
            focusLossBanner = L.text("mobile.quiz.lockdown.focusLossBanner")
        @unknown default:
            break
        }
    }

    private func handleGuidedAccessChange() {
        guard isActive else { return }
        let enabled = UIAccessibility.isGuidedAccessEnabled
        if wasGuidedAccessEnabled && !enabled {
            reportIntegrityEvent("guided_access_exit")
        }
        wasGuidedAccessEnabled = enabled
        if enabled {
            platformWarning = nil
        }
    }

    private func reportIntegrityEvent(_ eventType: String, showBanner: Bool = true) {
        onFocusLoss?(eventType)
        if showBanner {
            focusLossBanner = L.text("mobile.quiz.lockdown.focusLossBanner")
        }
    }
}
