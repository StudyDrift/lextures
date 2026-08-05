import SwiftUI

/// Course checklist workspace section (CC.9).
struct CourseChecklistSection: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    let course: CourseSummary
    var initialFocus: String?

    @State private var controller: CourseChecklistController
    @State private var showCompleted = false
    @State private var expandedItems: Set<String> = []
    @State private var dismissTarget: ChecklistItem?

    init(course: CourseSummary, initialFocus: String? = nil) {
        self.course = course
        self.initialFocus = initialFocus
        _controller = State(initialValue: CourseChecklistController(courseCode: course.courseCode))
    }

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var summary: CourseChecklistSummary? {
        controller.checklist?.summary
            ?? CourseChecklistSummaryStore.shared.cached(courseCode: course.courseCode)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            header

            if !isOnline {
                offlinePanel
            } else if let errorMessage = controller.errorMessage {
                errorPanel(errorMessage)
            } else if controller.loading && controller.checklist == nil {
                LMSSkeletonList(count: 4)
            } else if let checklist = controller.checklist {
                content(checklist)
            } else {
                LMSSkeletonList(count: 3)
            }

            if let rateLimitMessage = controller.rateLimitMessage {
                Text(rateLimitMessage)
                    .font(.footnote)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            if let actionError = controller.actionError {
                Text(actionError)
                    .font(.footnote)
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
        .task {
            await controller.loadFull(
                accessToken: session.accessToken,
                isOnline: isOnline,
                initialFocus: initialFocus,
                reduceMotion: reduceMotion
            )
        }
        .sheet(item: $dismissTarget) { item in
            ChecklistDismissSheet(
                item: item,
                isOnline: isOnline,
                onCancel: { dismissTarget = nil },
                onConfirm: { reason, note in
                    dismissTarget = nil
                    Task {
                        await controller.dismiss(
                            item: item,
                            reason: reason,
                            note: note,
                            accessToken: session.accessToken,
                            isOnline: isOnline
                        )
                    }
                }
            )
            .presentationDetents([.medium, .large])
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(L.text("mobile.checklist.pageTitle"))
                    .font(LexturesTheme.displayFont(18))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Spacer()
                Button {
                    Task {
                        await controller.refresh(
                            accessToken: session.accessToken,
                            isOnline: isOnline
                        )
                    }
                } label: {
                    Image(systemName: "arrow.clockwise")
                }
                .disabled(!isOnline || controller.loading)
                .accessibilityLabel(L.text("mobile.checklist.recheck"))
            }

            if let summary {
                let progress = CourseChecklistLogic.progressLabel(done: summary.done, total: summary.total)
                Text(progress)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .accessibilityLabel(progress)
                    .accessibilityValue(
                        "\(Int(CourseChecklistLogic.progressFraction(done: summary.done, total: summary.total) * 100)) percent"
                    )

                ProgressView(
                    value: CourseChecklistLogic.progressFraction(done: summary.done, total: summary.total)
                )
                .tint(LexturesTheme.accent(for: colorScheme))
            }
        }
    }

    private var offlinePanel: some View {
        LMSEmptyState(
            systemImage: "wifi.slash",
            title: L.text("mobile.checklist.offlineTitle"),
            message: L.text("mobile.checklist.offlineBody")
        )
    }

    private func errorPanel(_ message: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(message)
                .foregroundStyle(LexturesTheme.coral)
            Button(L.text("mobile.checklist.retry")) {
                Task {
                    await controller.loadFull(
                        accessToken: session.accessToken,
                        isOnline: isOnline,
                        force: true,
                        initialFocus: initialFocus,
                        reduceMotion: reduceMotion
                    )
                }
            }
            .buttonStyle(.bordered)
        }
    }

    @ViewBuilder
    private func content(_ checklist: CourseChecklist) -> some View {
        let allDone = checklist.summary.outstandingTotal == 0 && checklist.summary.total > 0
        let hasCompleted = checklist.summary.done > 0
        if allDone && !showCompleted && checklist.dismissed.isEmpty {
            allDonePanel
        } else if checklist.categories.isEmpty && checklist.dismissed.isEmpty {
            Text(L.text("mobile.checklist.catalogEmpty"))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        } else {
            if hasCompleted {
                Toggle(isOn: $showCompleted) {
                    Text(showCompleted
                          ? L.text("mobile.checklist.hideCompleted")
                          : L.text("mobile.checklist.showCompleted"))
                }
                .toggleStyle(.button)
            }

            ForEach(checklist.categories) { category in
                categoryBlock(category)
            }

            if !checklist.dismissed.isEmpty {
                CourseChecklistDismissedBlock(
                    items: checklist.dismissed,
                    isOnline: isOnline,
                    onRestore: { item in
                        Task {
                            await controller.restore(
                                item: item,
                                accessToken: session.accessToken,
                                isOnline: isOnline
                            )
                        }
                    }
                )
            }
        }
    }

    private var allDonePanel: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.checklist.allDoneTitle"))
                .font(.headline)
            Text(L.text("mobile.checklist.allDoneBody"))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Button(L.text("mobile.checklist.showCompleted")) {
                showCompleted = true
            }
            .buttonStyle(.bordered)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
    }

    @ViewBuilder
    private func categoryBlock(_ category: ChecklistCategory) -> some View {
        let outstanding = CourseChecklistLogic.outstandingCount(in: category)
        let expanded = controller.expandedCategories.contains(category.id) || outstanding > 0
        let items = CourseChecklistLogic.visibleItems(in: category, showCompleted: showCompleted)

        if items.isEmpty {
            EmptyView()
        } else {
            VStack(alignment: .leading, spacing: 6) {
                Button {
                    if controller.expandedCategories.contains(category.id) {
                        controller.expandedCategories.remove(category.id)
                    } else {
                        controller.expandedCategories.insert(category.id)
                    }
                } label: {
                    HStack {
                        Image(systemName: expanded ? "chevron.down" : "chevron.right")
                            .font(.caption.weight(.semibold))
                        Text(category.title)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Spacer()
                        if outstanding > 0 {
                            Text(L.format("mobile.checklist.outstandingCount", outstanding))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(.isHeader)

                if expanded {
                    ForEach(items) { item in
                        CourseChecklistItemRow(
                            item: item,
                            isOnline: isOnline,
                            expanded: expandedItems.contains(item.id),
                            isHighlighted: controller.highlightAnchor != nil
                                && item.target?.anchor == controller.highlightAnchor,
                            onToggleEvidence: { toggleItem(item.id) },
                            onOpen: { openTarget(item.target) },
                            onDismiss: { dismissTarget = item },
                            onRecheck: {
                                Task {
                                    await controller.recheck(
                                        item: item,
                                        accessToken: session.accessToken,
                                        isOnline: isOnline
                                    )
                                }
                            },
                            onOpenEvidenceRow: { openTarget($0) }
                        )
                    }
                }
            }
            .padding(.vertical, 4)
        }
    }

    private func toggleItem(_ id: String) {
        if expandedItems.contains(id) {
            expandedItems.remove(id)
        } else {
            expandedItems.insert(id)
        }
    }

    private func openTarget(_ target: ChecklistNavTarget?) {
        let resolved = CourseChecklistLogic.resolveTarget(
            target,
            courseCode: course.courseCode,
            table: CourseChecklistTargetTable.shared
        )
        switch resolved.kind {
        case .native:
            if let section = resolved.workspaceSection {
                shell.activeCourseSection = section
                if let anchor = resolved.focusAnchor {
                    controller.applyHighlight(anchor: anchor, reduceMotion: reduceMotion)
                }
            }
        case .web:
            if let path = resolved.webPath {
                openWebPath(path)
            }
        case .unresolved:
            if let section = resolved.workspaceSection {
                shell.activeCourseSection = section
            } else if let path = resolved.webPath {
                openWebPath(path)
            }
        }
    }

    private func openWebPath(_ path: String) {
        let url = AppConfiguration.webURL(path: path)
        LinkOpener.open(
            LinkOpener.Request(urlString: url.absoluteString, source: "checklist"),
            shell: shell
        )
    }
}
