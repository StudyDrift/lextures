import SwiftUI

struct PathsCatalogView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    @State private var query = ""
    @State private var paths: [CatalogPathSummary] = []
    @State private var loading = false
    @State private var errorMessage: String?
    @State private var hasSearched = false

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading && paths.isEmpty {
                LMSSkeletonList(count: 4)
            } else if let errorMessage, paths.isEmpty {
                LMSEmptyState(
                    systemImage: "point.topleft.down.to.point.bottomright.curvepath",
                    title: L.text("mobile.paths.catalogTitle"),
                    message: errorMessage
                )
            } else if paths.isEmpty {
                LMSEmptyState(
                    systemImage: "magnifyingglass",
                    title: L.text("mobile.paths.catalogEmptyTitle"),
                    message: hasSearched
                        ? L.text("mobile.paths.catalogEmptyMessage")
                        : L.text("mobile.paths.catalogPrompt")
                )
            } else {
                ScrollView {
                    LazyVStack(spacing: 12) {
                        ForEach(paths) { path in
                            NavigationLink(value: path.slug) {
                                catalogCard(path)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.paths.catalogTitle"))
        .navigationBarTitleDisplayMode(.inline)
        .searchable(text: $query, prompt: L.text("mobile.paths.catalogSearch"))
        .onSubmit(of: .search) { Task { await search() } }
        .navigationDestination(for: String.self) { slug in
            PathLandingView(slug: slug)
        }
        .task { await search() }
    }

    private func catalogCard(_ path: CatalogPathSummary) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 6) {
                Text(path.title)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                if !path.description.isEmpty {
                    Text(path.description)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .lineLimit(2)
                }
                Text(catalogMeta(path))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func catalogMeta(_ path: CatalogPathSummary) -> String {
        let duration = PathsLogic.formatDuration(minutes: path.totalDurationMinutes)
        let price = PathsLogic.isPaid(bundlePriceCents: path.bundlePriceCents)
            ? PathsLogic.formatPrice(cents: path.bundlePriceCents ?? 0)
            : L.text("mobile.paths.free")
        return L.format("mobile.paths.catalogMeta", path.courseCount, duration, price)
    }

    private func search() async {
        loading = true
        errorMessage = nil
        hasSearched = true
        defer { loading = false }
        do {
            let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
            if let token = session.accessToken {
                paths = try await OfflineService.shared.cachedFetch(
                    key: OfflineCacheKey.catalogPaths(query: trimmed),
                    accessToken: token
                ) {
                    try await LMSAPI.fetchCatalogPaths(query: query, accessToken: token)
                }.value
            } else {
                paths = try await LMSAPI.fetchCatalogPaths(query: query, accessToken: nil)
            }
        } catch {
            errorMessage = L.text("mobile.paths.error.catalog")
            paths = []
        }
    }
}