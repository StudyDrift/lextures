import SwiftUI

/// Engagement insights summarized for mobile ("what's working") (M11.3).
struct WhatsWorkingView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL
    let course: CourseSummary
    let features: MobilePlatformFeatures

    @State private var workingWell: [InstructorSignalItem] = []
    @State private var needsAttention: [InstructorSignalItem] = []
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true

    var body: some View {
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

                if loading && workingWell.isEmpty && needsAttention.isEmpty {
                    LMSSkeletonList(count: 3)
                } else if workingWell.isEmpty && needsAttention.isEmpty {
                    LMSEmptyState(
                        systemImage: "chart.bar.doc.horizontal",
                        title: L.text("mobile.instructorInsights.whatsWorking.empty"),
                        message: L.text("mobile.instructorInsights.whatsWorking.emptyHint")
                    )
                } else {
                    if !workingWell.isEmpty {
                        sectionHeader(L.text("mobile.instructorInsights.whatsWorking.working"))
                        ForEach(workingWell) { item in
                            signalCard(item, positive: true)
                        }
                    }
                    if !needsAttention.isEmpty {
                        sectionHeader(L.text("mobile.instructorInsights.whatsWorking.attention"))
                        ForEach(needsAttention) { item in
                            signalCard(item, positive: false)
                        }
                    }
                }

                redesignSection

                Button {
                    openURL(AppConfiguration.webURL(path: InstructorInsightsLogic.webWhatsWorkingPath(courseCode: course.courseCode)))
                } label: {
                    Text(L.text("mobile.instructorInsights.webReports"))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.brandTeal)
                }
                .buttonStyle(.plain)
            }
            .padding(16)
        }
        .navigationTitle(L.text("mobile.instructorInsights.whatsWorking.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.courseInstructorInsights(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchInstructorInsights(courseCode: course.courseCode, accessToken: token)
            }
            workingWell = result.value.workingWell
            needsAttention = result.value.needsAttention
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
            errorMessage = nil
        } catch {
            errorMessage = L.text("mobile.instructorInsights.error.insights")
        }
    }

    @ViewBuilder
    private var redesignSection: some View {
        EmptyView()
    }

    private func sectionHeader(_ title: String) -> some View {
        Text(title)
            .font(LexturesTheme.displayFont(17))
            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
    }

    private func signalCard(_ item: InstructorSignalItem, positive: Bool) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                HStack(spacing: 8) {
                    Image(systemName: positive ? "hand.thumbsup.fill" : "exclamationmark.circle.fill")
                        .foregroundStyle(positive ? LexturesTheme.brandTeal : LexturesTheme.amber)
                        .accessibilityHidden(true)
                    Text(item.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                }
                Text(item.narrative)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(signalMetrics(item))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .accessibilityLabel(signalMetricsAccessibility(item))
            }
        }
    }

    private func signalMetrics(_ item: InstructorSignalItem) -> String {
        var parts = [L.format("mobile.instructorInsights.whatsWorking.completion", InstructorInsightsLogic.completionPercentText(item.completionRate))]
        if let avg = InstructorInsightsLogic.optionalPercentText(item.avgScore) {
            parts.append(L.format("mobile.instructorInsights.whatsWorking.avgScore", avg))
        }
        parts.append(L.format("mobile.instructorInsights.whatsWorking.engagement", Int(item.engagement.rounded())))
        return parts.joined(separator: " · ")
    }

    private func signalMetricsAccessibility(_ item: InstructorSignalItem) -> String {
        signalMetrics(item)
    }
}
