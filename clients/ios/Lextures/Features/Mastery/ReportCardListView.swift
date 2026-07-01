import SwiftUI
import UIKit

/// Cross-course list of the student's released report cards (M6.2).
struct ReportCardListView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    @State private var cards: [ReportCardSummary] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var cacheLabel: String?
    @State private var shareFile: ReportCardShareFile?

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    if !NetworkMonitor.shared.isOnline {
                        OfflineBanner()
                    }
                    if let cacheLabel {
                        StalenessChip(label: cacheLabel)
                    }
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && cards.isEmpty {
                        LMSSkeletonList(count: 3)
                    } else if cards.isEmpty {
                        LMSEmptyState(
                            systemImage: "doc.text.fill",
                            title: L.text("mobile.mastery.reportCardEmptyTitle"),
                            message: L.text("mobile.mastery.reportCardEmptyMessage")
                        )
                    } else {
                        ForEach(cards) { card in
                            cardRow(card)
                        }
                    }
                }
                .padding(16)
            }
            .refreshable { await load() }
        }
        .navigationTitle(L.text("mobile.mastery.reportCards"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .sheet(item: $shareFile) { file in
            ReportCardShareSheet(items: [file.url])
        }
    }

    private func cardRow(_ card: ReportCardSummary) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.format("mobile.mastery.reportCardPeriod", card.gradingPeriod))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                if let letterGrade = card.letterGrade {
                    Text(letterGrade)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Button {
                    Task { await downloadPDF(card) }
                } label: {
                    Text(L.text("mobile.mastery.viewPdf"))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                }
            }
        }
    }

    private func downloadPDF(_ card: ReportCardSummary) async {
        guard let token = session.accessToken else { return }
        do {
            let data = try await LMSAPI.fetchReportCardPDF(cardId: card.id, accessToken: token)
            let url = FileManager.default.temporaryDirectory
                .appendingPathComponent("report-card-\(card.gradingPeriod).pdf")
            try data.write(to: url, options: .atomic)
            shareFile = ReportCardShareFile(url: url)
        } catch {
            errorMessage = L.text("mobile.mastery.reportCardLoadError")
        }
    }

    private func load() async {
        guard let token = session.accessToken else {
            loading = false
            return
        }
        loading = true
        errorMessage = nil
        defer { loading = false }

        do {
            let result = try await offline.cachedFetch(
                key: MasteryLogic.cacheKeyMyReportCards(),
                accessToken: token
            ) {
                try await LMSAPI.fetchMyReportCards(accessToken: token)
            }
            cards = MasteryLogic.releasedReportCards(result.value)
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = L.text("mobile.mastery.reportCardLoadError")
        }
    }
}

private struct ReportCardShareFile: Identifiable {
    let id = UUID()
    let url: URL
}

private struct ReportCardShareSheet: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}
