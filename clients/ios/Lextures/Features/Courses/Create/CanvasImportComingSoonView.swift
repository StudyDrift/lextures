import SwiftUI

/// MOB.1 FR-8 handoff placeholder until MOB.2 Canvas import ships.
struct CanvasImportComingSoonView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    var onBackToSource: (() -> Void)?

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                LMSEmptyState(
                    systemImage: "square.and.arrow.down.on.square",
                    title: L.text("mobile.createCourse.canvas.comingSoon.title"),
                    message: L.text("mobile.createCourse.canvas.comingSoon.body")
                )
            }
            .navigationTitle(L.text("mobile.createCourse.source.canvas.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) {
                        if let onBackToSource {
                            onBackToSource()
                        } else {
                            dismiss()
                        }
                    }
                }
            }
        }
    }
}
