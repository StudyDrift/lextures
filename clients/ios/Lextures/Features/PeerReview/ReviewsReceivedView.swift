import SwiftUI

/// Feedback peers left on the student's own work (M5.2).
struct ReviewsReceivedView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let courseCode: String
    let assignmentId: String
    let assignmentTitle: String

    @State private var reviews: [PeerReviewReceivedItem] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var cacheLabel: String?

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

                    if loading && reviews.isEmpty {
                        LMSSkeletonList(count: 2)
                    } else if reviews.isEmpty {
                        LMSEmptyState(
                            systemImage: "text.bubble",
                            title: L.text("mobile.peerReview.receivedEmptyTitle"),
                            message: L.text("mobile.peerReview.receivedEmptyMessage")
                        )
                    } else {
                        ForEach(reviews) { review in
                            reviewCard(review)
                        }
                    }
                }
                .padding(16)
            }
            .refreshable { await load(force: true) }
        }
        .navigationTitle(L.text("mobile.peerReview.receivedTitle"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load(force: false) }
    }

    private func reviewCard(_ review: PeerReviewReceivedItem) -> some View {
        LMSCard {
            HStack {
                Text(review.reviewerLabel ?? L.text("mobile.peerReview.anonymousReviewer"))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Spacer(minLength: 0)
                if let score = review.score {
                    Text(L.format("mobile.peerReview.scoreValue", score.formatted()))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                }
            }
            if let submittedAt = LMSDates.parse(review.submittedAt) {
                Text(submittedAt.formatted(date: .abbreviated, time: .shortened))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            if let comments = review.comments?.trimmingCharacters(in: .whitespacesAndNewlines), !comments.isEmpty {
                Text(comments)
                    .font(.body)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .padding(.top, 6)
            }
        }
    }

    private func load(force: Bool) async {
        guard let token = session.accessToken else {
            loading = false
            return
        }
        loading = true
        errorMessage = nil
        defer { loading = false }

        do {
            let result = try await offline.cachedFetch(
                key: PeerReviewLogic.cacheKeyReceived(courseCode: courseCode, assignmentId: assignmentId),
                accessToken: token
            ) {
                try await LMSAPI.fetchPeerReviewReceived(
                    courseCode: courseCode,
                    assignmentId: assignmentId,
                    accessToken: token
                )
            }
            reviews = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = L.text("mobile.peerReview.receivedLoadError")
        }
    }
}
