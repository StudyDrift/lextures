// swiftlint:disable identifier_name large_tuple
import SwiftUI

/// Purpose-built passage selection for highlight_annotate — sentence/paragraph tap
/// by default; avoids the OS copy/paste callout (CT.M7 FR-12 / FR-13).
struct PassageSelectionView: View {
    @Environment(\.colorScheme) private var colorScheme

    let passage: String
    let units: [ContentToolPack3Logic.PassageUnit]
    let annotations: [(id: String, start: Int, end: Int, tagLabel: String, tagColor: String)]
    let selectedUnitIndex: Int?
    let readOnly: Bool
    let onSelectUnit: (Int) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.contentTools.tools.highlight_annotate.sentenceTapHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            VStack(alignment: .leading, spacing: 6) {
                ForEach(units, id: \.index) { unit in
                    let isSelected = selectedUnitIndex == unit.index
                    let covering = annotations.filter { $0.start < unit.end && $0.end > unit.start }
                    Button {
                        guard !readOnly else { return }
                        onSelectUnit(unit.index)
                    } label: {
                        Text(unit.text)
                            .font(.body)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            .multilineTextAlignment(.leading)
                            .padding(10)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .frame(minHeight: 44)
                            .background(
                                RoundedRectangle(cornerRadius: 6)
                                    .fill(isSelected
                                          ? LexturesTheme.accent(for: colorScheme).opacity(0.2)
                                          : covering.isEmpty
                                            ? Color.clear
                                            : LexturesTheme.amber.opacity(0.18))
                            )
                            .overlay(
                                RoundedRectangle(cornerRadius: 6)
                                    .strokeBorder(
                                        isSelected
                                            ? LexturesTheme.accent(for: colorScheme)
                                            : Color.clear,
                                        lineWidth: 2
                                    )
                            )
                    }
                    .buttonStyle(.plain)
                    .disabled(readOnly)
                    .accessibilityLabel(unitAccessibility(unit: unit, covering: covering, selected: isSelected))
                    .accessibilityAddTraits(isSelected ? .isSelected : [])
                }
            }
            .textSelection(.disabled)
        }
    }

    private func unitAccessibility(
        unit: ContentToolPack3Logic.PassageUnit,
        covering: [(id: String, start: Int, end: Int, tagLabel: String, tagColor: String)],
        selected: Bool
    ) -> String {
        var parts: [String] = []
        if selected {
            parts.append(L.text("mobile.contentTools.interaction.selected"))
        }
        parts.append(unit.text)
        if let first = covering.first {
            parts.append(L.format("mobile.contentTools.tools.highlight_annotate.taggedAs", first.tagLabel))
        }
        if !readOnly {
            parts.append(L.text("mobile.contentTools.tools.highlight_annotate.doubleTapToSelect"))
        }
        return parts.joined(separator: ". ")
    }
}
