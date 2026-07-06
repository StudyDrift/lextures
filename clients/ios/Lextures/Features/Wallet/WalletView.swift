import SwiftUI

/// Consolidated credentials wallet: certs, CCR, CE transcript, official transcripts (M12.2).
struct WalletView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    @State private var credentials: [IssuedCredentialSummary] = []
    @State private var ccrAchievements: [CCRAchievement] = []
    @State private var ccrDocuments: [CCRDocument] = []
    @State private var ceAwards: [CETranscriptAward] = []
    @State private var transcriptRequests: [TranscriptRequestSummary] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var cacheLabel: String?
    @State private var selectedCredential: IssuedCredentialSummary?
    @State private var showCCR = false
    @State private var showCETranscript = false
    @State private var showOfficialTranscripts = false

    private var platform: MobilePlatformFeatures { shell.platformFeatures }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                LMSSkeletonList(count: 4)
            } else if let errorMessage, isEmpty {
                LMSEmptyState(
                    systemImage: "wallet.pass.fill",
                    title: L.text("mobile.wallet.errorTitle"),
                    message: errorMessage
                )
            } else if isEmpty {
                LMSEmptyState(
                    systemImage: "wallet.pass.fill",
                    title: L.text("mobile.wallet.emptyTitle"),
                    message: L.text("mobile.wallet.emptyMessage")
                )
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        if let cacheLabel {
                            StalenessChip(label: cacheLabel)
                        }
                        if WalletLogic.credentialsSectionEnabled(platform), !credentials.isEmpty {
                            credentialsSection
                        }
                        if WalletLogic.ccrEnabled(platform) {
                            ccrSection
                        }
                        if WalletLogic.ceTranscriptEnabled(platform) {
                            ceTranscriptSection
                        }
                        if WalletLogic.officialTranscriptsEnabled(platform) {
                            officialTranscriptsSection
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.wallet.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
        .navigationDestination(item: $selectedCredential) { credential in
            CredentialDetailView(credential: credential)
        }
        .navigationDestination(isPresented: $showCCR) {
            WalletCCRDetailView(
                achievements: ccrAchievements,
                documents: ccrDocuments,
                onUpdate: { achievements, documents in
                    ccrAchievements = achievements
                    ccrDocuments = documents
                }
            )
        }
        .navigationDestination(isPresented: $showCETranscript) {
            WalletCETranscriptDetailView(awards: ceAwards)
        }
        .navigationDestination(isPresented: $showOfficialTranscripts) {
            WalletOfficialTranscriptDetailView(requests: transcriptRequests)
        }
    }

    private var isEmpty: Bool {
        credentials.isEmpty && ccrAchievements.isEmpty && ccrDocuments.isEmpty
            && ceAwards.isEmpty && transcriptRequests.isEmpty
    }

    @ViewBuilder
    private var credentialsSection: some View {
        LMSSectionHeader(title: L.text("mobile.wallet.section.credentials"), systemImage: "rosette")
        ForEach(credentials) { credential in
            Button {
                selectedCredential = credential
            } label: {
                walletRow(
                    title: credential.title,
                    subtitle: CredentialsLogic.sourceTypeLabel(credential.sourceType),
                    detail: L.format("mobile.credentials.issued", WalletLogic.dateLabel(iso: credential.issuedAt)),
                    systemImage: "rosette"
                )
            }
            .buttonStyle(.plain)
        }
    }

    @ViewBuilder
    private var ccrSection: some View {
        LMSSectionHeader(title: L.text("mobile.wallet.section.ccr"), systemImage: "person.text.rectangle")
        Button {
            showCCR = true
        } label: {
            walletRow(
                title: L.text("mobile.wallet.ccr.detailTitle"),
                subtitle: L.format(
                    "mobile.wallet.itemCount",
                    ccrAchievements.count
                ),
                detail: ccrDocuments.first.map { WalletLogic.dateLabel(iso: $0.generatedAt) }
                    ?? L.text("mobile.wallet.viewDetails"),
                systemImage: "person.text.rectangle"
            )
        }
        .buttonStyle(.plain)
    }

    @ViewBuilder
    private var ceTranscriptSection: some View {
        LMSSectionHeader(title: L.text("mobile.wallet.section.ceTranscript"), systemImage: "doc.text")
        Button {
            showCETranscript = true
        } label: {
            walletRow(
                title: L.text("mobile.wallet.ceTranscript.detailTitle"),
                subtitle: L.format("mobile.wallet.itemCount", ceAwards.count),
                detail: ceAwards.isEmpty
                    ? L.text("mobile.wallet.viewDetails")
                    : L.text("mobile.wallet.ceTranscript.downloadPdf"),
                systemImage: "doc.text"
            )
        }
        .buttonStyle(.plain)
    }

    @ViewBuilder
    private var officialTranscriptsSection: some View {
        LMSSectionHeader(title: L.text("mobile.wallet.section.officialTranscripts"), systemImage: "envelope.open")
        Button {
            showOfficialTranscripts = true
        } label: {
            walletRow(
                title: L.text("mobile.wallet.officialTranscripts.detailTitle"),
                subtitle: L.format("mobile.wallet.itemCount", transcriptRequests.count),
                detail: L.text("mobile.wallet.openWebRequest"),
                systemImage: "envelope.open"
            )
        }
        .buttonStyle(.plain)
    }

    @ViewBuilder
    private func walletRow(title: String, subtitle: String, detail: String, systemImage: String) -> some View {
        LMSCard {
            HStack(spacing: 12) {
                Image(systemName: systemImage)
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 28)
                VStack(alignment: .leading, spacing: 4) {
                    Text(title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(subtitle)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text(detail)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer(minLength: 0)
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = isEmpty
        errorMessage = nil
        defer { loading = false }

        var sawCache = false
        var loadError = false

        if WalletLogic.credentialsSectionEnabled(platform) {
            do {
                let result = try await offline.cachedFetch(
                    key: OfflineCacheKey.credentialsList(),
                    accessToken: token
                ) {
                    try await LMSAPI.fetchMyCredentials(accessToken: token)
                }
                credentials = result.value
                if result.cached?.isStale(isOnline: NetworkMonitor.shared.isOnline) == true {
                    sawCache = true
                }
            } catch {
                loadError = credentials.isEmpty
            }
        }

        if WalletLogic.ccrEnabled(platform) {
            do {
                let result = try await offline.cachedFetch(
                    key: OfflineCacheKey.walletCCR(),
                    accessToken: token
                ) {
                    try await LMSAPI.fetchMyCCR(accessToken: token)
                }
                ccrAchievements = result.value.achievements ?? []
                ccrDocuments = result.value.documents ?? []
                if result.cached?.isStale(isOnline: NetworkMonitor.shared.isOnline) == true {
                    sawCache = true
                }
            } catch {
                if ccrAchievements.isEmpty && ccrDocuments.isEmpty { loadError = true }
            }
        }

        if WalletLogic.ceTranscriptEnabled(platform) {
            do {
                let result = try await offline.cachedFetch(
                    key: OfflineCacheKey.walletCETranscript(),
                    accessToken: token
                ) {
                    try await LMSAPI.fetchCETranscript(accessToken: token)
                }
                ceAwards = result.value.awards ?? []
                if result.cached?.isStale(isOnline: NetworkMonitor.shared.isOnline) == true {
                    sawCache = true
                }
            } catch {
                if ceAwards.isEmpty { loadError = true }
            }
        }

        if WalletLogic.officialTranscriptsEnabled(platform) {
            do {
                let result = try await offline.cachedFetch(
                    key: OfflineCacheKey.walletTranscriptRequests(),
                    accessToken: token
                ) {
                    try await LMSAPI.fetchTranscriptRequests(accessToken: token)
                }
                transcriptRequests = result.value
                if result.cached?.isStale(isOnline: NetworkMonitor.shared.isOnline) == true {
                    sawCache = true
                }
            } catch {
                if transcriptRequests.isEmpty { loadError = true }
            }
        }

        if loadError && isEmpty {
            errorMessage = L.text("mobile.wallet.loadError")
        }
        cacheLabel = sawCache ? Cached(value: (), fetchedAt: Date()).lastUpdatedLabel : nil
    }
}
