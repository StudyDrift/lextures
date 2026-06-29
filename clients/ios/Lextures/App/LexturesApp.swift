import SwiftUI

@main
struct LexturesApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @State private var authSession = AuthSession()
    @State private var localePreferences = LocalePreferences.shared
    @State private var biometricGate = BiometricGate.shared

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(authSession)
                .environment(biometricGate)
                .environment(AccessibilityPreferences.shared)
                .environment(localePreferences)
                .environment(\.locale, localePreferences.effectiveLocale)
                .environment(\.layoutDirection, localePreferences.layoutDirection)
        }
    }
}
