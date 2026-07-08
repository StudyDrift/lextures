import SwiftUI

struct LearnerProfileRoute: Hashable {}

/// Read-only learner profile with LP08 controls (LP10).
struct LearnerProfileView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    @State private var profile: LearnerProfile?
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var cacheLabel: String?
    @State private var controlError: String?
    @State private var controlBusy = false
    @State private var confirmingPause = false
    @State private var confirmingResume = false
    @State private var confirmingReset = false
    @State private var resetPhrase = ""
    @State private var exportShare: ExportShareFile?

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var controlsDisabled: Bool { !isOnline || controlBusy }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading, profile == nil {
                LMSSkeletonList(count: 3)
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        if let cacheLabel {
                            StalenessChip(label: cacheLabel)
                        }
                        if !isOnline {
                            offlineBanner
                        }
                        introCard
                        if let errorMessage {
                            errorBanner(errorMessage)
                        }
                        if let controlError {
                            errorBanner(controlError)
                        }
                        if let profile, LearnerProfileLogic.isPaused(profile) {
                            pausedBanner
                        }
                        if let profile, LearnerProfileLogic.showEmptyState(profile) {
                            emptyState
                        } else if let profile {
                            ForEach(LearnerProfileLogic.sortFacets(profile.facets)) { facet in
                                LearnerProfileFacetCard(facet: facet, isOnline: isOnline)
                            }
                        }
                        manageCard
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.learnerProfile.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
        .confirmationDialog(
            L.text("mobile.learnerProfile.manage.pauseConfirmTitle"),
            isPresented: $confirmingPause,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.learnerProfile.manage.pause"), role: .destructive) {
                Task { await runControl { try await LMSAPI.pauseLearnerProfile(accessToken: $0) } }
            }
        } message: {
            Text(L.text("mobile.learnerProfile.manage.pauseConfirmBody"))
        }
        .confirmationDialog(
            L.text("mobile.learnerProfile.manage.resumeConfirmTitle"),
            isPresented: $confirmingResume,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.learnerProfile.manage.resume")) {
                Task { await runControl { try await LMSAPI.resumeLearnerProfile(accessToken: $0) } }
            }
        } message: {
            Text(L.text("mobile.learnerProfile.manage.resumeConfirmBody"))
        }
        .sheet(isPresented: $confirmingReset) {
            resetSheet
        }
        .sheet(item: $exportShare) { item in
            LearnerProfileExportShareSheet(items: [item.url])
        }
    }

    private var introCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.learnerProfile.howItWorks.title"))
                    .font(LexturesTheme.displayFont(17))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(L.text("mobile.learnerProfile.description"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(L.text("mobile.learnerProfile.howItWorks.body"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private var offlineBanner: some View {
        LMSCard {
            Text(L.text("mobile.learnerProfile.offline.banner"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .accessibilityLabel(L.text("mobile.learnerProfile.offline.banner"))
        }
    }

    private var pausedBanner: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 6) {
                Text(L.text("mobile.learnerProfile.paused.title"))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(L.text("mobile.learnerProfile.paused.body"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .accessibilityElement(children: .combine)
        }
    }

    private var emptyState: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.learnerProfile.empty.title"))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(L.text("mobile.learnerProfile.empty.body"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .accessibilityElement(children: .combine)
        }
    }

    private var manageCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.learnerProfile.manage.title"))
                    .font(LexturesTheme.displayFont(17))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(L.text("mobile.learnerProfile.manage.description"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                Button {
                    Task { await downloadExport() }
                } label: {
                    controlRow(
                        systemImage: "arrow.down.doc",
                        title: L.text("mobile.learnerProfile.manage.download")
                    )
                }
                .disabled(controlsDisabled)
                .accessibilityHint(L.text("mobile.learnerProfile.manage.downloadHint"))

                if let profile, LearnerProfileLogic.isPaused(profile) {
                    Button { confirmingResume = true } label: {
                        controlRow(
                            systemImage: "play.fill",
                            title: L.text("mobile.learnerProfile.manage.resume")
                        )
                    }
                    .disabled(controlsDisabled)
                } else {
                    Button { confirmingPause = true } label: {
                        controlRow(
                            systemImage: "pause.fill",
                            title: L.text("mobile.learnerProfile.manage.pause")
                        )
                    }
                    .disabled(controlsDisabled)
                }

                Button(role: .destructive) {
                    resetPhrase = ""
                    confirmingReset = true
                } label: {
                    controlRow(
                        systemImage: "arrow.counterclockwise",
                        title: L.text("mobile.learnerProfile.manage.reset"),
                        destructive: true
                    )
                }
                .disabled(controlsDisabled)

                if controlsDisabled, !isOnline {
                    Text(L.text("mobile.learnerProfile.offline.controlsDisabled"))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else if controlBusy {
                    ProgressView()
                        .controlSize(.small)
                }
            }
        }
    }

    private var resetSheet: some View {
        NavigationStack {
            Form {
                Section {
                    Text(L.text("mobile.learnerProfile.manage.resetConfirmBody"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Section(L.text("mobile.learnerProfile.manage.resetConfirmPhraseLabel")) {
                    TextField(
                        L.text("mobile.learnerProfile.manage.resetConfirmPhrase"),
                        text: $resetPhrase
                    )
                    .textInputAutocapitalization(.characters)
                    .autocorrectionDisabled()
                }
            }
            .navigationTitle(L.text("mobile.learnerProfile.manage.resetConfirmTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) { confirmingReset = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.learnerProfile.manage.reset"), role: .destructive) {
                        confirmingReset = false
                        Task {
                            await runControl { try await LMSAPI.resetLearnerProfile(accessToken: $0) }
                        }
                    }
                    .disabled(
                        resetPhrase.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
                            != L.text("mobile.learnerProfile.manage.resetConfirmPhrase").uppercased()
                    )
                }
            }
        }
        .presentationDetents([.medium])
    }

    @ViewBuilder
    private func controlRow(systemImage: String, title: String, destructive: Bool = false) -> some View {
        HStack(spacing: 12) {
            Image(systemName: systemImage)
                .foregroundStyle(destructive ? LexturesTheme.coral : LexturesTheme.accent(for: colorScheme))
                .frame(width: 24)
            Text(title)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(destructive ? LexturesTheme.coral : LexturesTheme.textPrimary(for: colorScheme))
            Spacer(minLength: 0)
        }
        .frame(minHeight: 44)
        .contentShape(Rectangle())
    }

    private func errorBanner(_ message: String) -> some View {
        Text(message)
            .font(.caption)
            .foregroundStyle(LexturesTheme.error)
            .padding(12)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(LexturesTheme.error.opacity(0.08))
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .accessibilityLabel(message)
    }

    @MainActor
    private func load() async {
        guard let token = session.accessToken else { return }
        if profile == nil { loading = true }
        errorMessage = nil
        defer { loading = false }

        do {
            let result = try await offline.cachedFetch(
                key: LearnerProfileLogic.cacheKeyProfile(),
                accessToken: token
            ) {
                try await LMSAPI.fetchLearnerProfile(accessToken: token)
            }
            profile = result.value
            if let cached = result.cached, cached.isStale(isOnline: isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            if profile == nil {
                errorMessage = L.text("mobile.learnerProfile.error.load")
            }
        }
    }

    @MainActor
    private func runControl(_ action: (String) async throws -> String) async {
        guard let token = session.accessToken, isOnline else { return }
        controlBusy = true
        controlError = nil
        defer { controlBusy = false }
        do {
            _ = try await action(token)
            await load()
        } catch {
            controlError = L.text("mobile.learnerProfile.manage.error")
        }
    }

    @MainActor
    private func downloadExport() async {
        guard let token = session.accessToken, isOnline else { return }
        controlBusy = true
        controlError = nil
        defer { controlBusy = false }
        do {
            let data = try await LMSAPI.exportLearnerProfile(accessToken: token)
            let url = FileManager.default.temporaryDirectory
                .appendingPathComponent("learner-profile-export.json")
            try data.write(to: url, options: .atomic)
            exportShare = ExportShareFile(url: url)
        } catch {
            controlError = L.text("mobile.learnerProfile.manage.error")
        }
    }
}

private struct LearnerProfileFacetCard: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline

    let facet: LearnerProfileFacetSummary
    let isOnline: Bool

    @State private var expanded = true
    @State private var insights: [LearnerProfileInsight]?
    @State private var loadingInsights = false
    @State private var insightError: String?

    var body: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Button {
                    withAnimation(.easeInOut(duration: 0.2)) { expanded.toggle() }
                } label: {
                    HStack(alignment: .top) {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(L.dynamicText(LearnerProfileLogic.facetTitleKey(facet.facetKey)))
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Text(L.dynamicText(LearnerProfileLogic.facetDescriptionKey(facet.facetKey)))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                .multilineTextAlignment(.leading)
                        }
                        Spacer(minLength: 8)
                        Image(systemName: expanded ? "chevron.up" : "chevron.down")
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    .frame(minHeight: 44, alignment: .top)
                }
                .buttonStyle(.plain)
                .accessibilityLabel(
                    expanded
                        ? L.text("mobile.learnerProfile.facet.collapse")
                        : L.text("mobile.learnerProfile.facet.expand")
                )

                if expanded {
                    facetBody
                    footer
                }
            }
        }
        .task(id: facet.facetKey) { await loadInsightsIfNeeded() }
    }

    @ViewBuilder
    private var facetBody: some View {
        if facet.state == "insufficient_data" {
            Text(L.text("mobile.learnerProfile.facet.insufficient"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        } else if let insightError {
            Text(insightError)
                .font(.caption)
                .foregroundStyle(LexturesTheme.error)
        } else if loadingInsights {
            ProgressView()
                .controlSize(.small)
        } else {
            if let chart = chartCaption {
                Text(chart)
                    .font(.caption2.monospaced())
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .accessibilityLabel(chart)
            }
            ForEach(insights ?? []) { insight in
                LearnerProfileInsightRow(
                    facetKey: facet.facetKey,
                    insight: insight,
                    isOnline: isOnline
                )
            }
        }
    }

    private var chartCaption: String? {
        switch facet.facetKey {
        case "study_rhythm": return LearnerProfileLogic.rhythmChartCaption(facet.summary)
        case "content_modality": return LearnerProfileLogic.modalityChartCaption(facet.summary)
        default: return nil
        }
    }

    private var footer: some View {
        HStack {
            Text(L.format("mobile.learnerProfile.facet.lastComputed", DateFormatting.formatAbsoluteShort(facet.updatedAt)))
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Spacer(minLength: 8)
            Label(
                L.dynamicText(LearnerProfileLogic.confidenceLabelKey(facet.confidence)),
                systemImage: confidenceIcon
            )
            .font(.caption2.weight(.medium))
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            .accessibilityLabel(L.dynamicText(LearnerProfileLogic.confidenceLabelKey(facet.confidence)))
        }
    }

    private var confidenceIcon: String {
        switch LearnerProfileLogic.confidenceLevel(facet.confidence) {
        case .high: return "checkmark.seal.fill"
        case .medium: return "checkmark.seal"
        case .low: return "questionmark.circle"
        }
    }

    @MainActor
    private func loadInsightsIfNeeded() async {
        guard facet.state == "ok", let token = session.accessToken else { return }
        loadingInsights = true
        insightError = nil
        defer { loadingInsights = false }
        do {
            let detail = try await LMSAPI.fetchLearnerProfileFacet(facetKey: facet.facetKey, accessToken: token)
            insights = detail?.insights ?? []
        } catch {
            insightError = L.text("mobile.learnerProfile.facet.error")
            insights = []
        }
    }
}

private struct LearnerProfileInsightRow: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(AuthSession.self) private var session

    let facetKey: LearnerProfileFacetKey
    let insight: LearnerProfileInsight
    let isOnline: Bool

    @State private var evidenceExpanded = false
    @State private var evidence: [LearnerProfileEvidenceRow]?
    @State private var loadingEvidence = false
    @State private var evidenceError: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.dynamicText(LearnerProfileLogic.insightLabelKey(insight.insightKey)))
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(LearnerProfileLogic.formatInsightValue(insight, facetKey: facetKey))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Text(derivedSummary)
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.9))

            Button {
                evidenceExpanded.toggle()
                if evidenceExpanded { Task { await loadEvidence() } }
            } label: {
                HStack(spacing: 4) {
                    Text(
                        evidenceExpanded
                            ? L.text("mobile.learnerProfile.evidence.collapse")
                            : L.text("mobile.learnerProfile.evidence.expand")
                    )
                    .font(.caption.weight(.semibold))
                    Image(systemName: evidenceExpanded ? "chevron.up" : "chevron.down")
                        .font(.caption2.weight(.semibold))
                }
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            }
            .frame(minHeight: 44, alignment: .leading)
            .accessibilityLabel(
                evidenceExpanded
                    ? L.text("mobile.learnerProfile.evidence.collapse")
                    : L.text("mobile.learnerProfile.evidence.expand")
            )

            if evidenceExpanded {
                evidenceBody
            }
        }
        .padding(12)
        .background(LexturesTheme.textPrimary(for: colorScheme).opacity(0.04))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
    }

    private var derivedSummary: String {
        let rows = evidence ?? insight.evidence ?? []
        return LearnerProfileLogic.derivedFromSummary(
            count: LearnerProfileLogic.totalObservationCount(rows),
            courses: LearnerProfileLogic.uniqueCourseCount(rows)
        )
    }

    @ViewBuilder
    private var evidenceBody: some View {
        if loadingEvidence {
            ProgressView().controlSize(.small)
        } else if let evidenceError {
            Text(evidenceError)
                .font(.caption2)
                .foregroundStyle(LexturesTheme.error)
        } else if let evidence, evidence.isEmpty {
            Text(L.text("mobile.learnerProfile.evidence.empty"))
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        } else if let evidence {
            ForEach(evidence) { row in
                VStack(alignment: .leading, spacing: 2) {
                    Text(LearnerProfileLogic.sourceKindLabel(row.sourceKind))
                        .font(.caption2.weight(.semibold))
                    Text(L.plural("mobile.learnerProfile.evidence.observationCount", count: row.observationCount))
                        .font(.caption2)
                    if let window = formatWindow(row.windowStart, row.windowEnd) {
                        Text(window)
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.vertical, 4)
                if row.id != evidence.last?.id { Divider() }
            }
        }
    }

    private func formatWindow(_ start: String?, _ end: String?) -> String? {
        guard start != nil || end != nil else { return nil }
        let a = start.map { DateFormatting.formatAbsoluteShort($0) } ?? "…"
        let b = end.map { DateFormatting.formatAbsoluteShort($0) } ?? "…"
        return L.format("mobile.learnerProfile.evidence.window", a, b)
    }

    @MainActor
    private func loadEvidence() async {
        guard evidence == nil, !loadingEvidence, let token = session.accessToken else { return }
        if !isOnline, let inline = insight.evidence, !inline.isEmpty {
            evidence = inline
            return
        }
        loadingEvidence = true
        evidenceError = nil
        defer { loadingEvidence = false }
        do {
            let map = try await LMSAPI.fetchLearnerProfileFacetEvidence(facetKey: facetKey, accessToken: token)
            evidence = map[insight.insightKey] ?? []
        } catch {
            if let inline = insight.evidence, !inline.isEmpty {
                evidence = inline
            } else {
                evidenceError = L.text("mobile.learnerProfile.evidence.error")
            }
        }
    }
}

private struct ExportShareFile: Identifiable {
    let id = UUID()
    let url: URL
}

private struct LearnerProfileExportShareSheet: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}