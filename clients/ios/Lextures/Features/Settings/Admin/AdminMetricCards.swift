import SwiftUI

struct AdminMetricCardsGrid<Filter: Hashable, Definition: Identifiable>: View where Definition.ID == Filter {
    @Environment(\.colorScheme) private var colorScheme
    let definitions: [Definition]
    let selected: Filter?
    let loading: Bool
    let value: (Definition) -> Int64?
    let title: (Definition) -> String
    let hint: (Definition) -> String?
    let systemImage: (Definition) -> String
    let onSelect: (Filter) -> Void
    let hintLine: String
    private let columns = [GridItem(.flexible(), spacing: 12), GridItem(.flexible(), spacing: 12)]

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(hintLine).font(.caption).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            LazyVGrid(columns: columns, spacing: 12) {
                ForEach(definitions) { def in
                    let isSelected = selected == def.id
                    Button { onSelect(def.id) } label: {
                        metricCard(title: title(def), hint: hint(def), value: value(def), systemImage: systemImage(def), selected: isSelected)
                    }
                    .buttonStyle(.plain)
                    .accessibilityAddTraits(isSelected ? [.isSelected, .isButton] : .isButton)
                }
            }
        }
    }

    private func metricCard(title: String, hint: String?, value: Int64?, systemImage: String, selected: Bool) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .top, spacing: 8) {
                Text(title.uppercased())
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .lineLimit(2)
                    .frame(maxWidth: .infinity, minHeight: 28, alignment: .topLeading)
                Image(systemName: systemImage)
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 32, height: 32)
                    .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.18 : 0.14))
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
            }
            Group {
                if loading, value == nil {
                    RoundedRectangle(cornerRadius: 6, style: .continuous)
                        .fill(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.35))
                        .frame(width: 56, height: 28)
                } else {
                    Text(value.map { NumberFormatter.localizedString(from: NSNumber(value: $0), number: .decimal) } ?? "—")
                        .font(LexturesTheme.displayFont(26, weight: .bold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .monospacedDigit()
                }
            }
            .frame(minHeight: 32, alignment: .leading)
            Text(hint ?? " ").font(.caption2).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme)).lineLimit(2).frame(minHeight: 28, alignment: .topLeading)
            HStack(spacing: 4) {
                Text(selected ? L.text("mobile.admin.metric.hideList") : L.text("mobile.admin.metric.viewList")).font(.caption2.weight(.semibold))
                Image(systemName: "chevron.down").font(.caption2.weight(.semibold)).rotationEffect(.degrees(selected ? 180 : 0))
            }
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .padding(14)
        .frame(maxWidth: .infinity, minHeight: 148, alignment: .topLeading)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .stroke(selected ? LexturesTheme.brandTeal.opacity(0.85) : LexturesTheme.fieldBorder(for: colorScheme).opacity(colorScheme == .dark ? 0.9 : 0.45), lineWidth: selected ? 2 : 1)
        )
        .shadow(color: LexturesTheme.cardShadow(for: colorScheme), radius: selected ? 14 : 10, y: 4)
    }
}
