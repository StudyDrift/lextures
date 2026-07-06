import SwiftUI

/// Student portfolio list (M12.1).
struct PortfolioView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    @State private var portfolios: [PortfolioSummary] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var cacheLabel: String?
    @State private var selectedPortfolio: PortfolioSummary?
    @State private var showCreate = false
    @State private var newTitle = ""
    @State private var newIntro = ""
    @State private var creating = false
    @State private var createError: String?

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                LMSSkeletonList(count: 3)
            } else if let errorMessage, portfolios.isEmpty {
                LMSEmptyState(
                    systemImage: "folder.fill",
                    title: L.text("mobile.portfolio.errorTitle"),
                    message: errorMessage
                )
            } else if portfolios.isEmpty {
                LMSEmptyState(
                    systemImage: "folder.fill",
                    title: L.text("mobile.portfolio.emptyTitle"),
                    message: L.text("mobile.portfolio.emptyMessage")
                )
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 12) {
                        if let cacheLabel {
                            StalenessChip(label: cacheLabel)
                        }
                        ForEach(portfolios) { portfolio in
                            Button {
                                selectedPortfolio = portfolio
                            } label: {
                                portfolioRow(portfolio)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.portfolio.title"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    showCreate = true
                } label: {
                    Image(systemName: "plus")
                }
                .accessibilityLabel(L.text("mobile.portfolio.create"))
            }
        }
        .task { await load() }
        .refreshable { await load() }
        .navigationDestination(item: $selectedPortfolio) { portfolio in
            PortfolioDetailView(portfolioId: portfolio.id, initialTitle: portfolio.title)
        }
        .sheet(isPresented: $showCreate) {
            createSheet
        }
    }

    @ViewBuilder
    private func portfolioRow(_ portfolio: PortfolioSummary) -> some View {
        LMSCard {
            HStack {
                Image(systemName: portfolio.isPublic ? "eye.fill" : "folder.fill")
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                VStack(alignment: .leading, spacing: 4) {
                    Text(portfolio.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    if !portfolio.introText.isEmpty {
                        Text(portfolio.introText)
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
            .accessibilityLabel(portfolio.title)
        }
    }

    private var createSheet: some View {
        NavigationStack {
            Form {
                if let createError {
                    Text(createError)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.coral)
                }
                TextField(L.text("mobile.portfolio.fieldTitle"), text: $newTitle)
                TextField(L.text("mobile.portfolio.fieldIntro"), text: $newIntro, axis: .vertical)
                    .lineLimit(3 ... 6)
            }
            .navigationTitle(L.text("mobile.portfolio.create"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("common.cancel")) { showCreate = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("common.save")) {
                        Task { await createPortfolio() }
                    }
                    .disabled(creating || newTitle.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
        }
        .presentationDetents([.medium])
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = portfolios.isEmpty
        errorMessage = nil
        defer { loading = false }

        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.portfolioList(),
                accessToken: token
            ) {
                try await LMSAPI.fetchMyPortfolios(accessToken: token)
            }
            portfolios = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = L.text("mobile.portfolio.loadError")
        }
    }

    private func createPortfolio() async {
        guard let token = session.accessToken else { return }
        let title = newTitle.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !title.isEmpty else { return }
        creating = true
        createError = nil
        defer { creating = false }
        do {
            let created = try await LMSAPI.createPortfolio(
                title: title,
                introText: newIntro.trimmingCharacters(in: .whitespacesAndNewlines),
                accessToken: token
            )
            portfolios.insert(created, at: 0)
            showCreate = false
            newTitle = ""
            newIntro = ""
            selectedPortfolio = created
        } catch {
            createError = L.text("mobile.portfolio.createError")
        }
    }
}
