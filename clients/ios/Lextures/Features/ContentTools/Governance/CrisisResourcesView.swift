import SwiftUI

/// Prominent crisis support resources — never treated as a generic retryable error (CT.M9 FR-13).
struct CrisisResourcesView: View {
    @Environment(\.colorScheme) private var colorScheme
    var resources: [String] = []

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.contentTools.governance.crisisTitle"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(L.text("mobile.contentTools.governance.crisisBody"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            if resources.isEmpty {
                Text(L.text("mobile.contentTools.governance.crisisDefaultResource"))
                    .font(.caption.weight(.medium))
            } else {
                ForEach(resources, id: \.self) { line in
                    Text(line)
                        .font(.caption.weight(.medium))
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.coral.opacity(0.12))
        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
        .accessibilityElement(children: .combine)
        .accessibilityLabel(
            "\(L.text("mobile.contentTools.governance.crisisTitle")). \(L.text("mobile.contentTools.governance.crisisBody"))"
        )
        .accessibilityAddTraits(.isStaticText)
    }
}
