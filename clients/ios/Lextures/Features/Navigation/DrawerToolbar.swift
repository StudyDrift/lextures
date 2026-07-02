import SwiftUI

/// Adds a leading hamburger button that opens the global drawer. Apply inside a
/// top-level screen's `NavigationStack` (after `.navigationTitle`).
struct GlobalDrawerToolbar: ViewModifier {
    @Environment(AppShellModel.self) private var shell

    func body(content: Content) -> some View {
        content.toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Button {
                    shell.openGlobalDrawer()
                } label: {
                    Image(systemName: "line.3.horizontal")
                }
                .accessibilityLabel(L.text("mobile.drawer.menu"))
            }
        }
    }
}

extension View {
    /// Leading hamburger toolbar button that opens the app-wide drawer.
    func globalDrawerToolbar() -> some View {
        modifier(GlobalDrawerToolbar())
    }
}
