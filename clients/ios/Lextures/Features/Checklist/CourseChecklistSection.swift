import SwiftUI

/// Course checklist workspace section (CC.9).
struct CourseChecklistSection: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    let course: CourseSummary
    var initialFocus: String?

    @State private var checklist: CourseChecklist?
    @State private var loading = false
    @State private var errorMessage: String?
    @State private var rateLimitMessage: String?
    @State private var showCompleted = false
    @State private var expandedCategories: Set<String> = []
    @State private var expandedItems: Set<String> = []
    @State private var dismissTarget: ChecklistItem?
    @State private var actionError: String?
    @State private var highlightAnchor: String?
    @State private var highlightClearTask: Task<Void, Never>?

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var summary: CourseChecklistSummary? {
        checklist?.summary ?? CourseChecklistSummaryStore.shared.cached(courseCode: course.courseCode)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            header

            if !isOnline {
                offlinePanel
            } else if let errorMessage {
                errorPanel(errorMessage)
            } else if loading && checklist == nil {
                LMSSkeletonList(count: 4)
            } else if let checklist {
                content(checklist)
            } else {
                LMSSkeletonList(count: 3)
            }

            if let rateLimitMessage {
                Text(rateLimitMessage)
                    .font(.footnote)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            if let actionError {
                Text(actionError)
                    .font(.footnote)
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
        .task { await loadFull() }
        .sheet(item: $dismissTarget) { item in
            ChecklistDismissSheet(
                item: item,
                isOnline: isOnline,
                onCancel: { dismissTarget = nil },
                onConfirm: { reason, note in
                    dismissTarget = nil
                    Task { await dismiss(item: item, reason: reason, note: note) }
                }
            )
            .presentationDetents([.medium, .large])
        }
    }

    // MARK: - Header

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(L.text("mobile.checklist.pageTitle"))
                    .font(LexturesTheme.displayFont(18))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Spacer()
                Button {
                    Task { await refresh() }
                } label: {
                    Image(systemName: "arrow.clockwise")
                }
                .disabled(!isOnline || loading)
                .accessibilityLabel(L.text("mobile.checklist.recheck"))
            }

            if let summary {
                let progress = CourseChecklistLogic.progressLabel(done: summary.done, total: summary.total)
                Text(progress)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .accessibilityLabel(progress)
                    .accessibilityValue("\(Int(CourseChecklistLogic.progressFraction(done: summary.done, total: summary.total) * 100)) percent")

                ProgressView(value: CourseChecklistLogic.progressFraction(done: summary.done, total: summary.total))
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
                Task { await loadFull(force: true) }
            }
            .buttonStyle(.bordered)
        }
    }

    // MARK: - Content

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
            // Offer show/hide whenever any items are done (not only when everything is done).
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
                dismissedBlock(checklist.dismissed)
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
        let expanded = expandedCategories.contains(category.id) || outstanding > 0
        let items = CourseChecklistLogic.visibleItems(in: category, showCompleted: showCompleted)

        if items.isEmpty {
            EmptyView()
        } else {
            categoryBlockBody(category, outstanding: outstanding, expanded: expanded, items: items)
        }
    }

    private func categoryBlockBody(
        _ category: ChecklistCategory,
        outstanding: Int,
        expanded: Bool,
        items: [ChecklistItem]
    ) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Button {
                if expandedCategories.contains(category.id) {
                    expandedCategories.remove(category.id)
                } else {
                    expandedCategories.insert(category.id)
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
                    itemRow(item)
                }
            }
        }
        .padding(.vertical, 4)
    }

    private func itemRow(_ item: ChecklistItem) -> some View {
        let done = CourseChecklistLogic.isDone(item.status)
        let evidenceCount = item.evidence?.rows.count ?? 0
        let expanded = expandedItems.contains(item.id)

        return VStack(alignment: .leading, spacing: 6) {
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
                        Button(L.text("mobile.checklist.open")) {
                            openTarget(item.target)
                        }
                        Button(L.text("mobile.checklist.dismiss")) {
                            dismissTarget = item
                        }
                        .disabled(!isOnline)
                    }
                    Button(L.text("mobile.checklist.recheckItem")) {
                        Task { await recheck(item) }
                    }
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
                    toggleItem(item.id)
                } else if !done {
                    openTarget(item.target)
                }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel(item.title)
            .accessibilityValue(CourseChecklistLogic.accessibilityStatusValue(item.status))
            .accessibilityHint(done ? "" : L.text("mobile.checklist.open"))

            if evidenceCount > 0 {
                Button {
                    toggleItem(item.id)
                } label: {
                    Text(expanded
                          ? L.text("mobile.checklist.hideEvidence")
                          : L.format("mobile.checklist.showEvidence", evidenceCount))
                        .font(.footnote.weight(.semibold))
                }
                .buttonStyle(.plain)
            }

            if expanded, let evidence = item.evidence {
                evidenceList(evidence)
            }
        }
        .padding(10)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 12, style: .continuous)
                .stroke(
                    highlightAnchor != nil && item.target?.anchor == highlightAnchor
                        ? LexturesTheme.accent(for: colorScheme)
                        : .clear,
                    lineWidth: 2
                )
        )
    }

    private func evidenceList(_ evidence: ChecklistEvidence) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            if let truncated = evidence.truncatedAt, truncated > 0, evidence.rows.count < truncated {
                Text(L.format("mobile.checklist.evidenceTruncated", evidence.rows.count, truncated))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            ForEach(evidence.rows) { row in
                Button {
                    openTarget(row.target)
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

    private func dismissedBlock(_ items: [ChecklistItem]) -> some View {
        DisclosureGroup {
            ForEach(items) { item in
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(item.title)
                            .font(.subheadline)
                        if let dismissal = item.dismissal {
                            Text(L.format("mobile.checklist.dismissedBy", dismissal.byDisplayName, dismissal.reason))
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }
                    Spacer()
                    Button(L.text("mobile.checklist.restore")) {
                        Task { await restore(item) }
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

    private func toggleItem(_ id: String) {
        if expandedItems.contains(id) {
            expandedItems.remove(id)
        } else {
            expandedItems.insert(id)
        }
    }

    // MARK: - Navigation

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
                    applyHighlight(anchor: anchor)
                }
            }
        case .web:
            if let path = resolved.webPath {
                let url = AppConfiguration.webURL(path: path)
                LinkOpener.open(
                    LinkOpener.Request(urlString: url.absoluteString, source: "checklist"),
                    shell: shell
                )
            }
        case .unresolved:
            if let section = resolved.workspaceSection {
                shell.activeCourseSection = section
            } else if let path = resolved.webPath {
                let url = AppConfiguration.webURL(path: path)
                LinkOpener.open(
                    LinkOpener.Request(urlString: url.absoluteString, source: "checklist"),
                    shell: shell
                )
            }
        }
    }

    private func applyHighlight(anchor: String) {
        highlightClearTask?.cancel()
        highlightAnchor = anchor
        let delay = reduceMotion ? 0.05 : CourseChecklistLogic.highlightDurationSeconds
        highlightClearTask = Task {
            try? await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))
            if !Task.isCancelled {
                highlightAnchor = nil
            }
        }
    }

    // MARK: - Networking

    private func loadFull(force: Bool = false) async {
        guard isOnline else { return }
        guard let token = session.accessToken else { return }
        if !force, checklist != nil { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let result = try await LMSAPI.fetchCourseChecklist(
                courseCode: course.courseCode,
                accessToken: token
            )
            checklist = result
            CourseChecklistSummaryStore.shared.applyChecklist(result)
            for cat in result.categories where CourseChecklistLogic.outstandingCount(in: cat) > 0 {
                expandedCategories.insert(cat.id)
            }
            if let focus = initialFocus {
                applyHighlight(anchor: focus)
            }
        } catch let APIError.httpStatus(code, _) where code == 403 {
            CourseChecklistSummaryStore.shared.markForbidden(courseCode: course.courseCode)
            errorMessage = nil
            checklist = nil
        } catch {
            errorMessage = Self.errorText(error)
        }
    }

    private func refresh() async {
        guard isOnline, let token = session.accessToken else { return }
        loading = true
        rateLimitMessage = nil
        defer { loading = false }
        do {
            let result = try await LMSAPI.refreshCourseChecklist(
                courseCode: course.courseCode,
                accessToken: token
            )
            checklist = result
            CourseChecklistSummaryStore.shared.applyChecklist(result)
        } catch let APIError.httpStatus(code, _) where code == 429 {
            rateLimitMessage = CourseChecklistLogic.rateLimitedMessage
        } catch {
            errorMessage = Self.errorText(error)
        }
    }

    private func dismiss(item: ChecklistItem, reason: ChecklistDismissReason, note: String?) async {
        guard isOnline, let token = session.accessToken, var current = checklist else { return }
        // Optimistic move
        let snapshot = current
        removeItem(item.id, from: &current)
        current.dismissed.insert(item, at: 0)
        current.summary.dismissed += 1
        if item.tier == .essential, CourseChecklistLogic.isOutstanding(item.status) {
            current.summary.outstandingEssential = max(0, current.summary.outstandingEssential - 1)
        }
        if CourseChecklistLogic.isOutstanding(item.status) {
            current.summary.outstandingTotal = max(0, current.summary.outstandingTotal - 1)
        }
        checklist = current
        CourseChecklistSummaryStore.shared.applyChecklist(current)
        do {
            _ = try await LMSAPI.dismissChecklistItem(
                courseCode: course.courseCode,
                itemID: item.id,
                reason: reason,
                note: note,
                accessToken: token
            )
            await loadFull(force: true)
        } catch {
            checklist = snapshot
            CourseChecklistSummaryStore.shared.applyChecklist(snapshot)
            actionError = Self.errorText(error)
        }
    }

    private func restore(_ item: ChecklistItem) async {
        guard isOnline, let token = session.accessToken else { return }
        do {
            _ = try await LMSAPI.restoreChecklistItem(
                courseCode: course.courseCode,
                itemID: item.id,
                accessToken: token
            )
            await loadFull(force: true)
        } catch {
            actionError = Self.errorText(error)
        }
    }

    private func recheck(_ item: ChecklistItem) async {
        guard isOnline, let token = session.accessToken else { return }
        do {
            _ = try await LMSAPI.recheckChecklistItem(
                courseCode: course.courseCode,
                itemID: item.id,
                accessToken: token
            )
            await loadFull(force: true)
        } catch {
            actionError = Self.errorText(error)
        }
    }

    private static func errorText(_ error: Error) -> String {
        (error as? LocalizedError)?.errorDescription ?? L.text("mobile.checklist.loadError")
    }

    private func removeItem(_ id: String, from checklist: inout CourseChecklist) {
        checklist.categories = checklist.categories.map { cat in
            var c = cat
            c.items = c.items.filter { $0.id != id }
            return c
        }
    }
}

// MARK: - Dismiss sheet

private struct ChecklistDismissSheet: View {
    let item: ChecklistItem
    let isOnline: Bool
    let onCancel: () -> Void
    let onConfirm: (ChecklistDismissReason, String?) -> Void

    @State private var reason: ChecklistDismissReason = .notApplicable
    @State private var note = ""
    @State private var showNote = false

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text(item.title)
                        .font(.body.weight(.medium))
                    Text(L.text("mobile.checklist.dismissDialogHelp"))
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                Section(L.text("mobile.checklist.dismissReasonLabel")) {
                    Picker(L.text("mobile.checklist.dismissReasonLabel"), selection: $reason) {
                        ForEach(ChecklistDismissReason.allCases) { r in
                            Text(L.text(String.LocalizationValue(r.labelKey))).tag(r)
                        }
                    }
                    .pickerStyle(.inline)
                    .labelsHidden()
                }
                Section {
                    if showNote {
                        TextField(
                            L.text("mobile.checklist.dismissNotePlaceholder"),
                            text: $note,
                            axis: .vertical
                        )
                        .lineLimit(3 ... 6)
                    } else {
                        Button(L.text("mobile.checklist.addNote")) {
                            showNote = true
                        }
                    }
                }
                if !isOnline {
                    Text(L.text("mobile.checklist.offlineMutations"))
                        .foregroundStyle(.secondary)
                }
            }
            .navigationTitle(L.text("mobile.checklist.dismissDialogTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.checklist.dismissCancel"), action: onCancel)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.checklist.dismissConfirm")) {
                        onConfirm(reason, showNote ? note : nil)
                    }
                    .disabled(!isOnline)
                }
            }
        }
    }
}
