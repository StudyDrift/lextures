import SwiftUI

/// Read-only rubric viewer for assignment details (criteria + rating bands).
struct AssignmentRubricViewerView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let rubric: RubricDefinition

    private var totalMax: Double {
        rubric.criteria.reduce(0) { partial, criterion in
            partial + (criterion.levels.map(\.points).max() ?? 0)
        }
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    if let title = rubric.title?.nilIfEmpty {
                        Text(title)
                            .font(LexturesTheme.displayFont(20))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    }
                    Text(summaryLine)
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    ForEach(Array(rubric.criteria.enumerated()), id: \.element.id) { index, criterion in
                        criterionCard(criterion, index: index)
                    }
                }
                .padding(16)
            }
            .background(LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea())
            .navigationTitle(L.text("mobile.assignment.rubricTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) { dismiss() }
                }
            }
        }
    }

    private var summaryLine: String {
        let count = rubric.criteria.count
        let criteriaPart = count == 1
            ? L.format("mobile.assignment.rubricCriterionCount", count)
            : L.format("mobile.assignment.rubricCriteriaCount", count)
        if totalMax > 0 {
            return "\(criteriaPart) · \(L.format("mobile.assignment.rubricPointsTotal", formatPoints(totalMax)))"
        }
        return criteriaPart
    }

    private func criterionCard(_ criterion: RubricCriterion, index: Int) -> some View {
        let maxPts = criterion.levels.map(\.points).max() ?? 0
        return LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                HStack(alignment: .firstTextBaseline) {
                    Text("\(index + 1). \(criterion.title)")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Spacer(minLength: 8)
                    if maxPts > 0 {
                        Text(L.format("mobile.assignment.rubricPoints", formatPoints(maxPts)))
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
                if let description = criterion.description?.nilIfEmpty {
                    Text(description)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                if !criterion.levels.isEmpty {
                    VStack(spacing: 8) {
                        ForEach(Array(criterion.levels.enumerated()), id: \.offset) { _, level in
                            HStack(alignment: .top, spacing: 10) {
                                VStack(alignment: .leading, spacing: 3) {
                                    Text(level.label)
                                        .font(.subheadline.weight(.medium))
                                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                    if let note = level.description?.nilIfEmpty {
                                        Text(note)
                                            .font(.caption)
                                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    }
                                }
                                Spacer(minLength: 0)
                                Text(L.format("mobile.assignment.rubricPoints", formatPoints(level.points)))
                                    .font(.caption.weight(.semibold))
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            .padding(10)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(LexturesTheme.sceneBackground(for: colorScheme).opacity(0.7))
                            .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                        }
                    }
                }
            }
        }
        .accessibilityElement(children: .combine)
    }

    private func formatPoints(_ value: Double) -> String {
        if value.rounded() == value {
            return String(Int(value))
        }
        return value.formatted()
    }
}

private extension String {
    var nilIfEmpty: String? {
        trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : self
    }
}
