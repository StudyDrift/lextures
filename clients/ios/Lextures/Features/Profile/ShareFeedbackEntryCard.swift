import SwiftUI

/// Share Feedback entry in profile settings (FB3).
struct ShareFeedbackEntryCard: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    let onTap: () -> Void

    var body: some View {
        if FeedbackLogic.feedbackEnabled(shell.platformFeatures) {
            LMSCard {
                Button(action: onTap) {
                    HStack(spacing: 12) {
                        Image(systemName: "megaphone.fill")
                            .font(.footnote.weight(.semibold))
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                            .frame(width: 32, height: 32)
                            .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.18 : 0.14))
                            .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                        VStack(alignment: .leading, spacing: 2) {
                            Text(L.text("mobile.feedback.entry"))
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Text(L.text("mobile.feedback.entrySubtitle"))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        Spacer(minLength: 0)
                        Image(systemName: "chevron.right")
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
                            .flipsForRightToLeftLayoutDirection(true)
                    }
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityLabel(L.text("mobile.feedback.entry"))
                .accessibilityHint(L.text("mobile.feedback.entrySubtitle"))
            }
        }
    }
}
