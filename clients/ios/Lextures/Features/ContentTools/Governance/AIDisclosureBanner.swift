import SwiftUI

/// Native AI disclosure chrome above tool content (CT.M9 FR-6) — cannot be covered by sandboxed tools.
struct AIDisclosureBanner: View {
    @Environment(\.colorScheme) private var colorScheme
    let mode: String
    let busy: Bool
    var onAcknowledge: () -> Void
    var onOptOut: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.contentTools.governance.aiDisclosureTitle"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(L.text("mobile.contentTools.governance.aiDisclosureBody"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            HStack(spacing: 8) {
                Button(ackLabel) { onAcknowledge() }
                    .buttonStyle(.borderedProminent)
                    .disabled(busy)
                Button(L.text("mobile.contentTools.governance.consentOptOut")) { onOptOut() }
                    .buttonStyle(.bordered)
                    .disabled(busy)
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.textSecondary(for: colorScheme).opacity(0.08))
        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
        .accessibilityElement(children: .contain)
        .accessibilityLabel(L.text("mobile.contentTools.governance.aiDisclosureTitle"))
    }

    private var ackLabel: String {
        mode.lowercased() == "acknowledge"
            ? L.text("mobile.contentTools.governance.consentGrant")
            : L.text("mobile.contentTools.governance.continueWithAI")
    }
}
