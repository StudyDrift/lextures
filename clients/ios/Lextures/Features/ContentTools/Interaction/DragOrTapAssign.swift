// swiftlint:disable identifier_name
import SwiftUI

/// Shared select-then-place interaction for CT.M7 (visible by default, not a11y-only).
struct DragOrTapAssignBar: View {
    @Environment(\.colorScheme) private var colorScheme
    var selectedLabel: String?
    var helperText: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(helperText)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            if let selectedLabel, !selectedLabel.isEmpty {
                Text(L.format("mobile.contentTools.interaction.selectedItem", selectedLabel))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .accessibilityLabel(
                        L.format("mobile.contentTools.interaction.selectedA11y", selectedLabel)
                    )
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.vertical, 4)
    }
}

struct PlacementChip: View {
    @Environment(\.colorScheme) private var colorScheme
    var title: String
    var selected: Bool
    var locked: Bool
    var correct: Bool?
    var disabled: Bool
    var action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 6) {
                if let correct {
                    Image(systemName: correct ? "checkmark.circle.fill" : "xmark.circle.fill")
                        .foregroundStyle(correct ? LexturesTheme.primary : LexturesTheme.coral)
                        .accessibilityHidden(true)
                }
                Text(title)
                    .font(.subheadline)
                    .multilineTextAlignment(.leading)
                if locked {
                    Image(systemName: "lock.fill")
                        .font(.caption2)
                        .accessibilityHidden(true)
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .frame(minHeight: 44)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(
                RoundedRectangle(cornerRadius: 8)
                    .fill(selected
                          ? LexturesTheme.accent(for: colorScheme).opacity(0.18)
                          : LexturesTheme.cardBackground(for: colorScheme))
            )
            .overlay(
                RoundedRectangle(cornerRadius: 8)
                    .strokeBorder(
                        selected
                            ? LexturesTheme.accent(for: colorScheme)
                            : LexturesTheme.fieldBorder(for: colorScheme),
                        lineWidth: selected ? 2 : 1
                    )
            )
        }
        .buttonStyle(.plain)
        .disabled(disabled || locked)
        .accessibilityAddTraits(selected ? .isSelected : [])
    }
}
