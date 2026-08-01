import SwiftUI

/// Blocked AI composer state with reachable consent action (CT.M9 FR-7/FR-8).
struct ConsentGateView: View {
    @Environment(\.colorScheme) private var colorScheme
    let busy: Bool
    var onGrant: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.contentTools.governance.consentRequired"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Button(L.text("mobile.contentTools.governance.consentGrant")) { onGrant() }
                .buttonStyle(.borderedProminent)
                .disabled(busy)
                .frame(minHeight: 44)
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel(L.text("mobile.contentTools.governance.consentRequired"))
    }
}
