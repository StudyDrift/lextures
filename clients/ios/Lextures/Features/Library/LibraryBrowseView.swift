import SwiftUI

/// More-hub Library / OER search and open surface (M3.6).
struct LibraryBrowseView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var tab: LibraryBrowseTab = .oer
    @State private var query = ""
    @State private var catalogResults: [LibraryCatalogResult] = []
    @State private var oerResults: [OERSearchResult] = []
    @State private var oerProviders: [String] = []
    @State private var selectedProvider: String?
    @State private var selectedOER: OERSearchResult?
    @State private var selectedCatalog: LibraryCatalogResult?
    @State private var loading = false
    @State private var errorMessage: String?
    @State private var hasSearched = false

    private var librarySearchEnabled: Bool {
        shell.platformFeatures.ffLibrary
    }

    private var oerSearchEnabled: Bool {
        shell.platformFeatures.oerLibraryEnabled
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            VStack(alignment: .leading, spacing: 12) {
                if librarySearchEnabled && oerSearchEnabled {
                    LMSSegmentedChips(
                        options: LibraryBrowseTab.allCases,
                        selection: $tab,
                        label: { $0 == .library ? L.text("mobile.library.tab.catalog") : L.text("mobile.library.tab.oer") }
                    )
                }

                if let errorMessage {
                    LMSErrorBanner(message: errorMessage)
                }

                resultsBody
            }
            .padding(16)
        }
        .navigationTitle(L.text("mobile.ia.more.library"))
        .navigationBarTitleDisplayMode(.inline)
        .searchable(text: $query, prompt: searchPrompt)
        .onSubmit(of: .search) { Task { await runSearch() } }
        .task { await bootstrap() }
        .sheet(item: $selectedOER) { hit in
            NavigationStack {
                WebItemView(
                    title: hit.title,
                    urlString: hit.url,
                    provider: LibraryResourceLogic.oerProviderLabel(hit.provider)
                )
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button(L.text("mobile.ia.close")) { selectedOER = nil }
                    }
                }
            }
        }
        .sheet(item: $selectedCatalog) { hit in
            NavigationStack {
                catalogDetail(hit)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button(L.text("mobile.ia.close")) { selectedCatalog = nil }
                        }
                    }
            }
        }
    }

    private var searchPrompt: String {
        tab == .library ? L.text("mobile.library.searchCatalog") : L.text("mobile.library.searchOer")
    }

    @ViewBuilder
    private var resultsBody: some View {
        if loading {
            LMSSkeletonList(count: 4)
        } else if !hasSearched {
            LMSEmptyState(
                systemImage: "magnifyingglass",
                title: L.text("mobile.library.searchPromptTitle"),
                message: L.text("mobile.library.searchPromptMessage")
            )
        } else if activeResultsEmpty {
            LMSEmptyState(
                systemImage: "books.vertical",
                title: L.text("mobile.library.noResultsTitle"),
                message: L.text("mobile.library.noResultsMessage")
            )
        } else if tab == .oer || !librarySearchEnabled {
            oerList
        } else {
            catalogList
        }
    }

    private var activeResultsEmpty: Bool {
        if tab == .library && librarySearchEnabled {
            return catalogResults.isEmpty
        }
        return oerResults.isEmpty
    }

    private var catalogList: some View {
        ScrollView {
            LazyVStack(spacing: 10) {
                ForEach(catalogResults) { hit in
                    Button { selectedCatalog = hit } label: {
                        resultCard(title: hit.title, subtitle: hit.author, detail: hit.isbn ?? hit.issn)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private var oerList: some View {
        ScrollView {
            LazyVStack(spacing: 10) {
                if oerProviders.count > 1, let selectedProvider {
                    Text(LibraryResourceLogic.oerProviderLabel(selectedProvider))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                ForEach(oerResults) { hit in
                    Button { selectedOER = hit } label: {
                        resultCard(
                            title: hit.title,
                            subtitle: hit.licenseLabel ?? hit.licenseSpdx,
                            detail: hit.subject
                        )
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private func resultCard(title: String, subtitle: String?, detail: String?) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 6) {
                Text(title)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .multilineTextAlignment(.leading)
                if let subtitle, !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                if let detail, !detail.isEmpty {
                    Text(detail)
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func catalogDetail(_ hit: LibraryCatalogResult) -> some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    Text(hit.title)
                        .font(.title3.weight(.semibold))
                    if let author = hit.author {
                        Text(author).font(.subheadline)
                    }
                    if let isbn = hit.isbn {
                        Text("ISBN \(isbn)").font(.caption)
                    }
                    LMSCard {
                        Text(L.text("mobile.library.catalogBrowseHint"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
                .padding(16)
            }
        }
        .navigationTitle(hit.title)
        .navigationBarTitleDisplayMode(.inline)
    }

    private func bootstrap() async {
        if librarySearchEnabled && !oerSearchEnabled {
            tab = .library
        } else {
            tab = .oer
        }
        guard let token = session.accessToken, oerSearchEnabled else { return }
        do {
            let providers = try await LMSAPI.fetchOERProviders(accessToken: token)
            oerProviders = providers
            selectedProvider = LibraryResourceLogic.defaultOERProvider(from: providers)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
        }
    }

    private func runSearch() async {
        guard let token = session.accessToken else { return }
        let q = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !q.isEmpty else { return }
        loading = true
        errorMessage = nil
        defer {
            loading = false
            hasSearched = true
        }
        do {
            if tab == .library && librarySearchEnabled {
                catalogResults = try await LMSAPI.searchLibraryCatalog(query: q, accessToken: token)
                oerResults = []
            } else if let provider = selectedProvider ?? LibraryResourceLogic.defaultOERProvider(from: oerProviders) {
                let response = try await LMSAPI.searchOER(provider: provider, query: q, accessToken: token)
                oerResults = response.results
                catalogResults = []
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.library.searchError")
        }
    }
}

