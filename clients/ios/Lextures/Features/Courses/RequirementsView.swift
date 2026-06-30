import SwiftUI

/// Structured lock explanation with progress and deep-link to the next required item (M3.4).
struct RequirementsView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let targetItem: CourseStructureItem
    let groups: [ModuleGroup]
    let progress: ModulesProgressSnapshot?
    let onGoToRequired: (String) -> Void

    private var summary: RequirementsSummary {
        RequirementsLogic.buildRequirements(for: targetItem, groups: groups, progress: progress)
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if summary.totalCount > 0 {
                        progressSection
                    }

                    VStack(alignment: .leading, spacing: 0) {
                        ForEach(Array(summary.rows.enumerated()), id: \.element.id) { index, row in
                            if index > 0 { Divider() }
                            requirementRow(row)
                        }
                    }
                    .accessibilityElement(children: .contain)
                    .accessibilityLabel(L.text("mobile.modules.requirements.listLabel"))

                    if let nextId = summary.nextRequiredItemId {
                        Button {
                            dismiss()
                            onGoToRequired(nextId)
                        } label: {
                            Label(L.text("mobile.modules.requirements.goToNext"), systemImage: "arrow.right.circle.fill")
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.borderedProminent)
                        .accessibilityHint(L.text("mobile.modules.requirements.goToNextHint"))
                    }
                }
                .padding(16)
            }
            .navigationTitle(L.text("mobile.modules.requirements.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.modules.requirements.done")) { dismiss() }
                }
            }
        }
        .onChange(of: progress) { _, updated in
            guard let updated else { return }
            let unlocked = !ModuleContentLogic.isLocked(in: updated, itemId: targetItem.id)
            if unlocked { dismiss() }
        }
    }

    private var progressSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.format("mobile.modules.requirements.progress", summary.metCount, summary.totalCount))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                .accessibilityLabel(
                    L.format("mobile.modules.requirements.progressA11y", summary.metCount, summary.totalCount)
                )

            ProgressView(value: Double(summary.metCount), total: Double(max(summary.totalCount, 1)))
                .tint(LexturesTheme.primary)
                .accessibilityHidden(true)

            Text(targetItem.title)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .padding(14)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.12 : 0.08))
        .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
    }

    private func requirementRow(_ row: RequirementRow) -> some View {
        HStack(alignment: .top, spacing: 12) {
            Image(systemName: row.met ? "checkmark.circle.fill" : "circle")
                .font(.body.weight(.semibold))
                .foregroundStyle(row.met ? LexturesTheme.primary : LexturesTheme.textSecondary(for: colorScheme))
                .accessibilityHidden(true)

            VStack(alignment: .leading, spacing: 3) {
                Text(row.title)
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                if let detail = row.detail, !detail.isEmpty {
                    Text(detail)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Text(row.met ? L.text("mobile.modules.requirements.met") : L.text("mobile.modules.requirements.unmet"))
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(row.met ? LexturesTheme.primary : LexturesTheme.coral)
            }

            Spacer(minLength: 0)
        }
        .padding(.vertical, 10)
        .accessibilityElement(children: .combine)
        .accessibilityLabel(requirementAccessibilityLabel(row))
    }

    private func requirementAccessibilityLabel(_ row: RequirementRow) -> String {
        let status = row.met ? L.text("mobile.modules.requirements.met") : L.text("mobile.modules.requirements.unmet")
        if let detail = row.detail, !detail.isEmpty {
            return "\(row.title), \(detail), \(status)"
        }
        return "\(row.title), \(status)"
    }
}
