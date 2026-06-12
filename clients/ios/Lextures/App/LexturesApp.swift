import SwiftUI

@main
struct LexturesApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @State private var authSession = AuthSession()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(authSession)
        }
    }
}
