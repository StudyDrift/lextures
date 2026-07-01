import SwiftUI

struct ResearchStudiesRoute: Hashable {}

/// Research study consent management (M1.5).
struct ResearchStudiesView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    @State private var phase: Phase = .loading
    @State private var pending: [ConsentStudy] = []
    @State private var history: [ConsentHistoryEntry] = []
    @State private var errorMessage: String?
    @State private var busyStudyId: String?
    @State private var selectedStudy: ConsentStudy?

    private enum Phase { case loading, ready, failed }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            switch phase {
            case .loading:
                ProgressView().controlSize(.large)
            case .failed:
                failedState
            case .ready:
                content
            }
        }
        .navigationTitle(L.text("mobile.profileDepth.research.title"))
        .navigationBarTitleDisplayMode(.inline)
        .sheet(item: $selectedStudy) { study in
            consentSheet(study)
        }
        .task { await load() }
    }

    private var failedState: some View {
        VStack(spacing: 12) {
            Text(L.text("mobile.profileDepth.research.loadError"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Button(L.text("mobile.common.retry")) { Task { await load() } }
        }
        .padding(32)
    }

    private var content: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text(L.text("mobile.profileDepth.research.description"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if let errorMessage {
                    Text(errorMessage)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.error)
                        .accessibilityLabel(errorMessage)
                }

                if !pending.isEmpty {
                    LMSCard {
                        Text(L.text("mobile.profileDepth.research.awaiting"))
                            .font(LexturesTheme.displayFont(17))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        ForEach(pending) { study in
                            pendingRow(study)
                            if study.id != pending.last?.id { Divider() }
                        }
                    }
                }

                LMSCard {
                    Text(L.text("mobile.profileDepth.research.decisions"))
                        .font(LexturesTheme.displayFont(17))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    let decisions = ProfileDepthLogic.latestConsentByStudy(history)
                    if decisions.isEmpty {
                        Text(L.text("mobile.profileDepth.research.empty"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    } else {
                        ForEach(decisions, id: \.studyId) { entry in
                            historyRow(entry)
                            if entry.studyId != decisions.last?.studyId { Divider() }
                        }
                    }
                }
            }
            .padding(16)
        }
    }

    private func pendingRow(_ study: ConsentStudy) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(study.title)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(L.format("mobile.profileDepth.research.irb", study.irbProtocol))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            HStack(spacing: 8) {
                Button(L.text("mobile.profileDepth.research.viewConsent")) {
                    selectedStudy = study
                }
                .font(.caption.weight(.semibold))
                Spacer(minLength: 0)
                Button(L.text("mobile.profileDepth.research.decline")) {
                    Task { await respond(studyId: study.id, decision: .declined) }
                }
                .font(.caption.weight(.semibold))
                .disabled(busyStudyId == study.id)
                Button(L.text("mobile.profileDepth.research.consent")) {
                    Task { await respond(studyId: study.id, decision: .granted) }
                }
                .font(.caption.weight(.bold))
                .foregroundStyle(.white)
                .padding(.horizontal, 12)
                .padding(.vertical, 6)
                .background(LexturesTheme.primary)
                .clipShape(Capsule())
                .disabled(busyStudyId == study.id)
            }
        }
        .padding(.vertical, 4)
    }

    private func historyRow(_ entry: ConsentHistoryEntry) -> some View {
        HStack(alignment: .top, spacing: 12) {
            VStack(alignment: .leading, spacing: 4) {
                Text(entry.studyTitle ?? L.text("mobile.profileDepth.research.study"))
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(ProfileDepthLogic.consentDecisionLabel(entry.decision))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            Spacer(minLength: 0)
            if entry.decision == .granted {
                Button(L.text("mobile.profileDepth.research.withdraw")) {
                    Task { await respond(studyId: entry.studyId, decision: .withdrawn) }
                }
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.error)
                .disabled(busyStudyId == entry.studyId)
            }
        }
        .padding(.vertical, 4)
    }

    private func consentSheet(_ study: ConsentStudy) -> some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    Text(study.consentText)
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(L.text("mobile.profileDepth.research.dataUse"))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text(study.dataUseDescription)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                .padding(16)
            }
            .navigationTitle(study.title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) { selectedStudy = nil }
                }
            }
        }
    }

    @MainActor
    private func load() async {
        phase = .loading
        errorMessage = nil
        guard let token = session.accessToken else {
            phase = .failed
            return
        }
        do {
            async let pendingStudies = LMSAPI.fetchPendingConsentStudies(accessToken: token)
            async let historyEntries = LMSAPI.fetchConsentHistory(accessToken: token)
            pending = try await pendingStudies
            history = try await historyEntries
            phase = .ready
        } catch {
            phase = .failed
        }
    }

    @MainActor
    private func respond(studyId: String, decision: ConsentDecision) async {
        guard let token = session.accessToken else { return }
        busyStudyId = studyId
        errorMessage = nil
        defer { busyStudyId = nil }
        do {
            try await LMSAPI.respondToConsentStudy(studyId: studyId, decision: decision, accessToken: token)
            await load()
        } catch {
            errorMessage = L.text("mobile.profileDepth.research.actionError")
        }
    }
}