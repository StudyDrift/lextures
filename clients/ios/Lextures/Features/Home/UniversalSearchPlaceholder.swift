import SwiftUI

/// M0.6 entry target — wired from the header search icon until universal search ships.
struct UniversalSearchPlaceholder: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                LMSEmptyState(
                    systemImage: "magnifyingglass",
                    title: L.text("mobile.ia.search.title"),
                    message: L.text("mobile.ia.search.comingSoon")
                )
            }
            .navigationTitle(L.text("mobile.ia.search"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.ia.close")) { dismiss() }
                }
            }
        }
    }
}