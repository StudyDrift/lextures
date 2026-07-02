import SwiftUI

struct PathsRoute: Hashable {}

@MainActor
@Observable
final class MyPathsModel {
    var paths: [PathProgress] = []
    var errorMessage: String?
    var loading = false

    func load(accessToken: String?) async {
        guard let accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            paths = try await OfflineService.shared.cachedFetch(
                key: OfflineCacheKey.myPaths(),
                accessToken: accessToken
            ) {
                try await LMSAPI.fetchMyPaths(accessToken: accessToken)
            }.value
        } catch {
            errorMessage = L.text("mobile.paths.error.load")
        }
    }
}

struct MyPathsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @State private var model = MyPathsModel()
    @State private var showCatalog = false

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if model.loading && model.paths.isEmpty {
                LMSSkeletonList(count: 3)
            } else if let error = model.errorMessage, model.paths.isEmpty {
                LMSEmptyState(
                    systemImage: "point.topleft.down.to.point.bottomright.curvepath",
                    title: L.text("mobile.paths.title"),
                    message: error
                )
            } else if model.paths.isEmpty {
                LMSEmptyState(
                    systemImage: "point.topleft.down.to.point.bottomright.curvepath",
                    title: L.text("mobile.paths.emptyTitle"),
                    message: L.text("mobile.paths.emptyMessage")
                )
            } else {
                ScrollView {
                    LazyVStack(spacing: 12) {
                        ForEach(model.paths) { path in
                            NavigationLink(value: path) {
                                pathCard(path)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.paths.title"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button(L.text("mobile.paths.browse")) { showCatalog = true }
            }
        }
        .navigationDestination(for: PathProgress.self) { path in
            PathRunnerView(path: path)
        }
        .navigationDestination(isPresented: $showCatalog) {
            PathsCatalogView()
        }
        .refreshable { await model.load(accessToken: session.accessToken) }
        .task { await model.load(accessToken: session.accessToken) }
    }

    private func pathCard(_ path: PathProgress) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                HStack(alignment: .top) {
                    Text(path.pathTitle)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Spacer(minLength: 0)
                    if path.completedAt != nil {
                        Image(systemName: "checkmark.circle.fill")
                            .foregroundStyle(LexturesTheme.primary)
                    }
                }
                ProgressView(value: Double(path.percent), total: 100)
                    .tint(LexturesTheme.primary)
                Text(path.progressLabel)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                if let next = PathsLogic.nextCourse(in: path) {
                    Label(L.format("mobile.paths.continueCourse", next.title), systemImage: "arrow.right.circle.fill")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}