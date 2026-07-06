import SwiftUI

/// Portfolio editor: artifacts, reorder, visibility, share (M12.1).
struct PortfolioDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let portfolioId: String
    let initialTitle: String

    @State private var portfolio: PortfolioSummary?
    @State private var artifacts: [PortfolioArtifact] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var showAddArtifact = false
    @State private var selectedArtifact: PortfolioArtifact?
    @State private var togglingPublic = false
    @State private var copiedLink = false

    private var orderedArtifacts: [PortfolioArtifact] {
        PortfolioLogic.orderedArtifacts(artifacts, order: portfolio?.order ?? [])
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                LMSSkeletonList(count: 4)
            } else if let errorMessage, portfolio == nil {
                LMSEmptyState(
                    systemImage: "folder.fill",
                    title: L.text("mobile.portfolio.errorTitle"),
                    message: errorMessage
                )
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        if let portfolio {
                            headerCard(portfolio)
                            shareCard(portfolio)
                        }
                        artifactsSection
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(portfolio?.title ?? initialTitle)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    showAddArtifact = true
                } label: {
                    Image(systemName: "plus")
                }
                .accessibilityLabel(L.text("mobile.portfolio.addArtifact"))
            }
        }
        .task { await load() }
        .refreshable { await load() }
        .navigationDestination(item: $selectedArtifact) { artifact in
            ArtifactDetailView(portfolioId: portfolioId, artifact: artifact)
        }
        .sheet(isPresented: $showAddArtifact) {
            ArtifactEditorView(portfolioId: portfolioId) { created in
                artifacts.append(created)
                showAddArtifact = false
            }
        }
    }

    @ViewBuilder
    private func headerCard(_ portfolio: PortfolioSummary) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                if !portfolio.introText.isEmpty {
                    Text(portfolio.introText)
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Toggle(isOn: Binding(
                    get: { portfolio.isPublic },
                    set: { newValue in Task { await togglePublic(newValue) } }
                )) {
                    Text(L.text("mobile.portfolio.publicToggle"))
                        .font(.subheadline.weight(.medium))
                }
                .disabled(togglingPublic)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private func shareCard(_ portfolio: PortfolioSummary) -> some View {
        if portfolio.isPublic, let slug = portfolio.publicSlug, !slug.isEmpty {
            let url = PortfolioLogic.publicPortfolioURL(slug: slug)
            LMSCard {
                VStack(alignment: .leading, spacing: 10) {
                    Text(L.text("mobile.portfolio.shareTitle"))
                        .font(.subheadline.weight(.semibold))
                    Text(url.absoluteString)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .textSelection(.enabled)
                    HStack(spacing: 12) {
                        ShareLink(
                            item: url,
                            subject: Text(portfolio.title),
                            message: Text(PortfolioLogic.shareText(title: portfolio.title, url: url))
                        ) {
                            Label(L.text("mobile.portfolio.share"), systemImage: "square.and.arrow.up")
                                .font(.caption.weight(.semibold))
                        }
                        Button {
                            UIPasteboard.general.string = url.absoluteString
                            copiedLink = true
                            DispatchQueue.main.asyncAfter(deadline: .now() + 2) { copiedLink = false }
                        } label: {
                            Label(
                                copiedLink ? L.text("mobile.portfolio.copied") : L.text("mobile.portfolio.copyLink"),
                                systemImage: copiedLink ? "checkmark" : "doc.on.doc"
                            )
                            .font(.caption.weight(.semibold))
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private var artifactsSection: some View {
        if orderedArtifacts.isEmpty {
            LMSEmptyState(
                systemImage: "doc.badge.plus",
                title: L.text("mobile.portfolio.noArtifactsTitle"),
                message: L.text("mobile.portfolio.noArtifactsMessage")
            )
        } else {
            LMSSectionHeader(title: L.text("mobile.portfolio.artifacts"), systemImage: "square.stack.3d.up.fill")
            ForEach(orderedArtifacts) { artifact in
                Button {
                    selectedArtifact = artifact
                } label: {
                    artifactRow(artifact)
                }
                .buttonStyle(.plain)
            }
            if orderedArtifacts.count > 1 {
                reorderControls
            }
        }
    }

    @ViewBuilder
    private func artifactRow(_ artifact: PortfolioArtifact) -> some View {
        LMSCard {
            HStack(alignment: .top, spacing: 10) {
                Image(systemName: artifact.isPublic ? "eye.fill" : "doc.fill")
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                VStack(alignment: .leading, spacing: 4) {
                    Text(artifact.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(PortfolioLogic.artifactTypeLabel(artifact.artifactType))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if !artifact.description.isEmpty {
                        Text(artifact.description)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            .lineLimit(2)
                    }
                }
                Spacer(minLength: 0)
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel(artifact.title)
        }
    }

    private var reorderControls: some View {
        LMSCard {
            Text(L.text("mobile.portfolio.reorderHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = portfolio == nil
        errorMessage = nil
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.portfolioDetail(portfolioId: portfolioId),
                accessToken: token
            ) {
                try await LMSAPI.fetchMyPortfolio(portfolioId: portfolioId, accessToken: token)
            }
            portfolio = result.value.portfolio
            artifacts = result.value.artifacts
        } catch {
            errorMessage = L.text("mobile.portfolio.loadError")
        }
    }

    private func togglePublic(_ isPublic: Bool) async {
        guard let token = session.accessToken else { return }
        togglingPublic = true
        defer { togglingPublic = false }
        do {
            let updated = try await LMSAPI.patchPortfolio(
                portfolioId: portfolioId,
                payload: PatchPortfolioRequest(isPublic: isPublic),
                accessToken: token
            )
            portfolio = updated
        } catch {
            errorMessage = L.text("mobile.portfolio.visibilityError")
        }
    }

    func moveArtifact(from source: IndexSet, to destination: Int) async {
        var ordered = orderedArtifacts
        ordered.move(fromOffsets: source, toOffset: destination)
        artifacts = ordered
        guard let token = session.accessToken else { return }
        do {
            let updated = try await LMSAPI.patchPortfolio(
                portfolioId: portfolioId,
                payload: PatchPortfolioRequest(order: ordered.map(\.id)),
                accessToken: token
            )
            portfolio = updated
        } catch {
            await load()
        }
    }
}