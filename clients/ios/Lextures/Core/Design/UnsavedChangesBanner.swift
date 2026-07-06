import SwiftUI

/// Sticky unsaved-changes banner mirroring web `UnsavedChangesBanner` (M13.1).
struct UnsavedChangesBanner: View {
    @Environment(\.colorScheme) private var colorScheme

    var isSaving: Bool
    var onSave: () -> Void
    var onDiscard: () -> Void

    var body: some View {
        HStack(spacing: 12) {
            Text(L.text("mobile.courseSettings.unsavedChanges"))
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                .frame(maxWidth: .infinity, alignment: .leading)

            Button(L.text("mobile.courseSettings.discard")) {
                onDiscard()
            }
            .font(.subheadline.weight(.semibold))
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            Button {
                onSave()
            } label: {
                if isSaving {
                    ProgressView()
                        .controlSize(.small)
                } else {
                    Text(L.text("mobile.courseSettings.saveChanges"))
                        .font(.subheadline.weight(.semibold))
                }
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.brandTeal)
            .disabled(isSaving)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(.ultraThinMaterial)
        .overlay(alignment: .top) {
            Divider()
        }
        .accessibilityElement(children: .combine)
    }
}
