import SwiftUI

/// Interactive rubric picker reused for peer review scoring (M5.2 / M6.1 pattern).
struct RubricScorerView: View {
    @Environment(\.colorScheme) private var colorScheme

    let rubric: RubricDefinition
    @Binding var scores: [String: Double]
    var disabled: Bool = false

    private var gradedCount: Int { PeerReviewLogic.rubricGradedCount(rubric, scores: scores) }
    private var totalScore: Double { PeerReviewLogic.rubricTotal(rubric, scores: scores) }
    private var allGraded: Bool { PeerReviewLogic.rubricScoresComplete(rubric, scores: scores) }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            LMSCard(accent: allGraded ? LexturesTheme.brandTeal : LexturesTheme.primary) {
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(L.text("mobile.peerReview.rubricScore"))
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        Text(totalScore.formatted())
                            .font(LexturesTheme.displayFont(24, weight: .bold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    }
                    Spacer(minLength: 0)
                    Text(L.format("mobile.peerReview.criteriaProgress", gradedCount, rubric.criteria.count))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }

            if let title = rubric.title?.nilIfEmpty {
                Text(title)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            }

            ForEach(Array(rubric.criteria.enumerated()), id: \.element.id) { index, criterion in
                criterionCard(criterion, index: index)
            }
        }
    }

    private func criterionCard(_ criterion: RubricCriterion, index: Int) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text("\(index + 1). \(criterion.title)")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                if let description = criterion.description?.nilIfEmpty {
                    Text(description)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                ForEach(Array(criterion.levels.enumerated()), id: \.offset) { _, level in
                    levelButton(criterion: criterion, level: level)
                }
            }
        }
        .accessibilityElement(children: .contain)
    }

    private func levelButton(criterion: RubricCriterion, level: RubricLevel) -> some View {
        let active = scores[criterion.id] == level.points
        return Button {
            guard !disabled else { return }
            scores[criterion.id] = level.points
        } label: {
            HStack(alignment: .top, spacing: 10) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(level.label)
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(
                            active
                                ? LexturesTheme.primary
                                : LexturesTheme.textPrimary(for: colorScheme)
                        )
                    if let description = level.description?.nilIfEmpty {
                        Text(description)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
                Spacer(minLength: 0)
                Text(L.format("mobile.peerReview.points", level.points.formatted()))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .padding(10)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(
                active
                    ? LexturesTheme.primary.opacity(0.12)
                    : LexturesTheme.sceneBackground(for: colorScheme).opacity(0.7)
            )
            .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 10, style: .continuous)
                    .stroke(
                        active ? LexturesTheme.primary : LexturesTheme.fieldBorder(for: colorScheme),
                        lineWidth: active ? 2 : 1
                    )
            )
        }
        .buttonStyle(.plain)
        .disabled(disabled)
        .accessibilityLabel("\(criterion.title), \(level.label), \(level.points) points")
        .accessibilityAddTraits(active ? .isSelected : [])
    }
}

private extension String {
    var nilIfEmpty: String? {
        trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : self
    }
}
