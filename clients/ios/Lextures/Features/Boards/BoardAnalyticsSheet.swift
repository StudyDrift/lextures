import SwiftUI

/// Per-board analytics panel (MOB.8 / VC.10).
struct BoardAnalyticsSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let courseCode: String
    let boardId: String

    @State private var summary: BoardAnalyticsSummary?
    @State private var loading = true
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                Group {
                    if loading && summary == nil {
                        ProgressView()
                    } else if let errorMessage, summary == nil {
                        LMSErrorBanner(message: errorMessage).padding(16)
                    } else if let summary {
                        ScrollView {
                            VStack(alignment: .leading, spacing: 16) {
                                Text(L.text("mobile.boards.analytics.subtitle"))
                                    .font(.subheadline)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                statsGrid(summary)
                                contributors(summary)
                            }
                            .padding(16)
                        }
                    } else {
                        LMSEmptyState(
                            systemImage: "chart.bar",
                            title: L.text("mobile.boards.analytics.empty"),
                            message: ""
                        )
                    }
                }
            }
            .navigationTitle(L.text("mobile.boards.analytics.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) { dismiss() }
                }
            }
            .task {
                BoardsAdvancedObservability.record("board_admin_analytics_viewed")
                await load()
            }
        }
    }

    private func statsGrid(_ summary: BoardAnalyticsSummary) -> some View {
        LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
            stat(L.text("mobile.boards.analytics.cards"), "\(summary.cardCount)")
            stat(L.text("mobile.boards.analytics.contributors"), "\(summary.uniqueContributors)")
            stat(L.text("mobile.boards.analytics.reactions"), "\(summary.reactionCount)")
            stat(L.text("mobile.boards.analytics.comments"), "\(summary.commentCount)")
        }
    }

    private func stat(_ label: String, _ value: String) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 4) {
                Text(label)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(value)
                    .font(.title2.weight(.semibold))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(4)
        }
    }

    @ViewBuilder
    private func contributors(_ summary: BoardAnalyticsSummary) -> some View {
        Text(L.text("mobile.boards.analytics.contributorList"))
            .font(.headline)
        if summary.contributors.isEmpty {
            Text(L.text("mobile.boards.analytics.noContributors"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        } else {
            ForEach(summary.contributors, id: \.userId) { c in
                HStack {
                    Text(String(c.userId.prefix(8)))
                        .font(.subheadline.monospaced())
                    Spacer()
                    Text("\(c.contributionTotal)")
                        .font(.subheadline.weight(.semibold))
                }
                .padding(.vertical, 4)
            }
        }
    }

    private func load() async {
        loading = true
        errorMessage = nil
        defer { loading = false }
        guard let token = session.accessToken else { return }
        do {
            summary = try await LMSAPI.fetchBoardAnalytics(
                courseCode: courseCode,
                boardId: boardId,
                accessToken: token
            )
        } catch {
            errorMessage = L.text("mobile.boards.analytics.loadError")
        }
    }
}
