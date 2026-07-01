import SwiftUI

/// Cross-course list of peer reviews the student must complete (M5.2).
struct PeerReviewListView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    @State private var allocations: [PeerReviewAllocation] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var cacheLabel: String?
    @State private var openDetail: PeerReviewDetailRoute?

    private var pending: [PeerReviewAllocation] { PeerReviewLogic.pending(allocations) }

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

                    progressCard

                    if loading && allocations.isEmpty {
                        LMSSkeletonList(count: 3)
                    } else if pending.isEmpty {
                        LMSEmptyState(
                            systemImage: "checkmark.seal.fill",
                            title: L.text("mobile.peerReview.allDoneTitle"),
                            message: L.text("mobile.peerReview.allDoneMessage")
                        )
                    } else {
                        ForEach(pending) { allocation in
                            Button {
                                openDetail = PeerReviewDetailRoute(allocationId: allocation.id)
                            } label: {
                                allocationRow(allocation)
                            }
                            .buttonStyle(.plain)
                        }
                    }

                    if !allocations.filter(PeerReviewLogic.isComplete).isEmpty {
                        completedSection
                    }
                }
                .padding(16)
            }
            .refreshable { await load(force: true) }
        }
        .navigationTitle(L.text("mobile.peerReview.title"))
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $openDetail) { route in
            PeerReviewDetailView(allocationId: route.allocationId) {
                Task { await load(force: true) }
            }
        }
        .task { await load(force: false) }
    }

    private var progressCard: some View {
        LMSCard {
            Text(L.text("mobile.peerReview.progress"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(
                L.format(
                    "mobile.peerReview.progressSummary",
                    PeerReviewLogic.completedCount(allocations),
                    allocations.count
                )
            )
            .font(.caption)
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private func allocationRow(_ allocation: PeerReviewAllocation) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.format("mobile.peerReview.reviewTarget", PeerReviewLogic.targetLabel(allocation)))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(allocation.courseCode)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                HStack {
                    Text(L.dynamicText(PeerReviewLogic.statusLabelKey(allocation.status)))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    Spacer(minLength: 0)
                    if let closesAt = allocation.closesAt, let date = LMSDates.parse(closesAt) {
                        Text(L.format("mobile.peerReview.due", date.formatted(date: .abbreviated, time: .shortened)))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.coral)
                    }
                }
            }
        }
    }

    private var completedSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(L.text("mobile.peerReview.completed"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            ForEach(allocations.filter(PeerReviewLogic.isComplete)) { allocation in
                Button {
                    openDetail = PeerReviewDetailRoute(allocationId: allocation.id)
                } label: {
                    LMSCard {
                        HStack {
                            Text(PeerReviewLogic.targetLabel(allocation))
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Spacer(minLength: 0)
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundStyle(LexturesTheme.brandTeal)
                        }
                    }
                }
                .buttonStyle(.plain)
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
                key: PeerReviewLogic.cacheKeyAssigned(),
                accessToken: token
            ) {
                try await LMSAPI.fetchPeerReviewAssigned(accessToken: token)
            }
            allocations = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = L.text("mobile.peerReview.loadError")
        }
    }
}
