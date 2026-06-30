import SwiftUI

/// Sectioned module list with type icons, completion, and lock indicators (M3.1).
struct ModuleListView: View {
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let groups: [ModuleGroup]
    let progress: ModulesProgressSnapshot?
    let onSelectItem: (CourseStructureItem) -> Void
    let onLockedItem: (CourseStructureItem, LockReason?) -> Void

    var body: some View {
        ForEach(Array(groups.enumerated()), id: \.element.id) { index, group in
            moduleCard(group, number: index + 1)
        }
    }

    private func moduleCard(_ group: ModuleGroup, number: Int) -> some View {
        let moduleState = ModuleContentLogic.moduleLockState(in: progress, moduleId: group.id)

        return LMSCard {
            HStack(spacing: 10) {
                Text("\(number)")
                    .font(LexturesTheme.displayFont(14, weight: .bold))
                    .foregroundStyle(.white)
                    .frame(width: 26, height: 26)
                    .background(LexturesTheme.coverGradient(for: course.courseCode))
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                VStack(alignment: .leading, spacing: 2) {
                    Text(group.title)
                        .font(LexturesTheme.displayFont(17))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    if moduleState?.locked == true, let reason = moduleState?.reason?.message, !reason.isEmpty {
                        Label(reason, systemImage: "lock.fill")
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.coral)
                    }
                }
            }

            if group.items.isEmpty {
                Text(L.text("mobile.modules.empty"))
                    .font(.caption)
                    .italic()
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(Array(group.items.enumerated()), id: \.element.id) { index, item in
                    if index > 0 { Divider() }
                    moduleItemRow(item)
                }
            }
        }
    }

    private func moduleItemRow(_ item: CourseStructureItem) -> some View {
        let navigable = ModuleContentLogic.isNavigable(item.kind)
        let locked = ModuleContentLogic.isLocked(in: progress, itemId: item.id)
        let complete = ModuleContentLogic.isComplete(in: progress, itemId: item.id)

        return Button {
            if locked {
                onLockedItem(item, ModuleContentLogic.itemLockState(in: progress, itemId: item.id)?.reason)
            } else if navigable {
                onSelectItem(item)
            }
        } label: {
            HStack(spacing: 12) {
                ZStack(alignment: .bottomTrailing) {
                    Image(systemName: locked ? "lock.fill" : ItemKind.icon(for: item.kind))
                        .font(.footnote.weight(.semibold))
                        .foregroundStyle(locked ? LexturesTheme.textSecondary(for: colorScheme) : LexturesTheme.accent(for: colorScheme))
                        .frame(width: 32, height: 32)
                        .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.16 : 0.13))
                        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                    if complete {
                        Image(systemName: "checkmark.circle.fill")
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.primary)
                            .offset(x: 4, y: 4)
                    }
                }

                VStack(alignment: .leading, spacing: 3) {
                    Text(item.title)
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(locked ? LexturesTheme.textSecondary(for: colorScheme) : LexturesTheme.textPrimary(for: colorScheme))
                    HStack(spacing: 6) {
                        Text(ItemKind.label(for: item.kind))
                            .font(.caption2.weight(.medium))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        if let due = LMSDates.parse(item.dueAt) {
                            Text("·")
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            Text("Due \(due.formatted(date: .abbreviated, time: .shortened))")
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(LexturesTheme.coral)
                        }
                    }
                }

                Spacer(minLength: 0)

                if let points = item.pointsWorth ?? item.pointsPossible {
                    Text("\(points.formatted()) pts")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.amber)
                        .padding(.horizontal, 7)
                        .padding(.vertical, 3)
                        .background(LexturesTheme.amber.opacity(0.13))
                        .clipShape(Capsule())
                }
                if navigable && !locked {
                    Image(systemName: "chevron.right")
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
                }
            }
            .padding(.vertical, 4)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .disabled(!navigable && !locked)
        .accessibilityLabel(ModuleContentLogic.accessibilityLabel(for: item, progress: progress))
    }
}
