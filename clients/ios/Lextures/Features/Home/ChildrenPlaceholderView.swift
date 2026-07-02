import SwiftUI

/// Parent shell tab placeholder until M10.1 parent portal ships.
struct ChildrenPlaceholderView: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                VStack(spacing: 0) {
                    if shell.iaRedesignEnabled {
                        ShellHeaderBar { shell.showUniversalSearch = true }
                            .padding(.horizontal, 16)
                    }
                    LMSEmptyState(
                        systemImage: "figure.2.and.child.holdinghands",
                        title: L.text("mobile.ia.children.title"),
                        message: L.text("mobile.ia.children.message")
                    )
                }
            }
            .navigationTitle(L.text("mobile.ia.tabs.children"))
            .navigationBarTitleDisplayMode(.inline)
            .globalDrawerToolbar()
            .navigationDestination(for: NotificationsRoute.self) { _ in
                NotificationsView()
            }
        }
    }
}