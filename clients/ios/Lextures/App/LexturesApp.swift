import SwiftUI

@main
struct LexturesApp: App {
    @State private var authSession = AuthSession()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(authSession)
        }
    }
}
