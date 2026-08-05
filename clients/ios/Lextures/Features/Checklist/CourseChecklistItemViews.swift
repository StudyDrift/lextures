import SwiftUI

/// Single checklist item card with evidence expand (CC.9).
struct CourseChecklistItemRow: View {
    @Environment(\.colorScheme) private var colorScheme

    let item: ChecklistItem
    let isOnline: Bool
    let expanded: Bool
    let isHighlighted: Bool
    let onToggleEvidence: () -> Void
    let onOpen: () -> Void
    let onDismiss: () -> Void
    let onRecheck: () -> Void
    let onOpenEvidenceRow: (ChecklistNavTarget?) -> Void

    private var done: Bool { CourseChecklistLogic.isDone(item.status) }
    private var evidenceCount: Int { item.evidence?.rows.count ?? 0 }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .top, spacing: 8) {
                Image(systemName: done ? "checkmark.circle.fill" : statusIcon(item.status))
                    .foregroundStyle(done ? LexturesTheme.brandTeal : LexturesTheme.textSecondary(for: colorScheme))
                VStack(alignment: .leading, spacing: 4) {
                    Text(item.title)
                        .font(.body.weight(.medium))
                        .strikethrough(done)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    if let detail = item.detail, !detail.isEmpty {
                        Text(detail)
                            .font(.footnote)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    if !item.why.isEmpty {
                        Text(item.why)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    HStack(spacing: 6) {
                        tierChip(item.tier)
                        if let progress = item.progress {
                            Text("\(progress.done) / \(progress.total)")
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }
                }
                Spacer(minLength: 0)
                Menu {
                    if !done {
                        Button(L.text("mobile.checklist.open"), action: onOpen)
                        Button(L.text("mobile.checklist.dismiss"), action: onDismiss)
                            .disabled(!isOnline)
                    }
                    Button(L.text("mobile.checklist.recheckItem"), action: onRecheck)
                        .disabled(!isOnline)
                } label: {
                    Image(systemName: "ellipsis")
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel(L.text("mobile.checklist.overflowMenu"))
            }
            .contentShape(Rectangle())
            .onTapGesture {
                if evidenceCount > 0 {
                    onToggleEvidence()
                } else if !done {
                    onOpen()
                }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel(item.title)
            .accessibilityValue(CourseChecklistLogic.accessibilityStatusValue(item.status))
            .accessibilityHint(done ? "" : L.text("mobile.checklist.open"))

            if evidenceCount > 0 {
                Button(action: onToggleEvidence) {
                    Text(expanded
                          ? L.text("mobile.checklist.hideEvidence")
                          : L.format("mobile.checklist.showEvidence", evidenceCount))
                        .font(.footnote.weight(.semibold))
                }
                .buttonStyle(.plain)
            }

            if expanded, let evidence = item.evidence {
                CourseChecklistEvidenceList(evidence: evidence, onOpenRow: onOpenEvidenceRow)
            }
        }
        .padding(10)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 12, style: .continuous)
                .stroke(
                    isHighlighted ? LexturesTheme.accent(for: colorScheme) : .clear,
                    lineWidth: 2
                )
        )
    }

    private func tierChip(_ tier: ChecklistTier) -> some View {
        Text(tier == .essential
              ? L.text("mobile.checklist.essentialTier")
              : L.text("mobile.checklist.recommendedTier"))
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.35))
            .clipShape(Capsule())
    }

    private func statusIcon(_ status: String) -> String {
        switch CourseChecklistLogic.normalizeStatus(status) {
        case .inProgress: return "circle.lefthalf.filled"
        case .unknown: return "questionmark.circle"
        default: return "circle"
        }
    }
}

struct CourseChecklistEvidenceList: View {
    @Environment(\.colorScheme) private var colorScheme

    let evidence: ChecklistEvidence
    let onOpenRow: (ChecklistNavTarget?) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            if let truncated = evidence.truncatedAt, truncated > 0, evidence.rows.count < truncated {
                Text(L.format("mobile.checklist.evidenceTruncated", evidence.rows.count, truncated))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            ForEach(evidence.rows) { row in
                Button {
                    onOpenRow(row.target)
                } label: {
                    HStack {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(row.label)
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            if let sub = row.sublabel, !sub.isEmpty {
                                Text(sub)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                        Spacer()
                        Image(systemName: "chevron.right")
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    .padding(.vertical, 6)
                    .frame(minHeight: 44)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.leading, 28)
    }
}

struct CourseChecklistDismissedBlock: View {
    @Environment(\.colorScheme) private var colorScheme

    let items: [ChecklistItem]
    let isOnline: Bool
    let onRestore: (ChecklistItem) -> Void

    var body: some View {
        DisclosureGroup {
            ForEach(items) { item in
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(item.title)
                            .font(.subheadline)
                        if let dismissal = item.dismissal {
                            Text(L.format(
                                "mobile.checklist.dismissedBy",
                                dismissal.byDisplayName,
                                dismissal.reason
                            ))
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }
                    Spacer()
                    Button(L.text("mobile.checklist.restore")) {
                        onRestore(item)
                    }
                    .disabled(!isOnline)
                    .font(.footnote.weight(.semibold))
                }
                .padding(.vertical, 6)
            }
        } label: {
            Text(L.format("mobile.checklist.dismissedSection", items.count))
                .font(.subheadline.weight(.semibold))
        }
    }
}
