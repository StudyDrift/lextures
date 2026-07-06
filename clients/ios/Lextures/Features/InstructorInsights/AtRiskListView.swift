import SwiftUI

/// At-risk students list sorted by predicted severity (M11.3).
struct AtRiskListView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary
    let features: MobilePlatformFeatures

    @State private var alerts: [AtRiskAlert] = []
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                if !NetworkMonitor.shared.isOnline {
                    OfflineBanner()
                }
                if let cacheLabel {
                    StalenessChip(label: cacheLabel)
                }
                if let errorMessage {
                    LMSErrorBanner(message: errorMessage)
                }

                Text(L.text("mobile.instructorInsights.predictedNotice"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if loading && alerts.isEmpty {
                    LMSSkeletonList(count: 4)
                } else if alerts.isEmpty {
                    LMSEmptyState(
                        systemImage: "checkmark.circle",
                        title: L.text("mobile.instructorInsights.atRisk.empty"),
                        message: L.text("mobile.instructorInsights.atRisk.emptyHint")
                    )
                } else {
                    ForEach(alerts) { alert in
                        if features.studentProgressEnabled {
                            NavigationLink(
                                value: InstructorInsightsRoute.studentProgress(
                                    enrollmentId: alert.enrollmentId,
                                    displayName: alert.displayName
                                )
                            ) {
                                alertRow(alert)
                            }
                            .buttonStyle(.plain)
                        } else {
                            alertRow(alert)
                        }
                    }
                }
            }
            .padding(16)
        }
        .navigationTitle(L.text("mobile.instructorInsights.atRisk.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func alertRow(_ alert: AtRiskAlert) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text(alert.displayName)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Spacer(minLength: 0)
                    if features.studentProgressEnabled {
                        Image(systemName: "chevron.right")
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
                severityBadge(alert)
                Text(alert.topFactorLabel)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(L.format("mobile.instructorInsights.atRisk.reason", alert.topFactorLabel))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
        .accessibilityElement(children: .combine)
    }

    private func severityBadge(_ alert: AtRiskAlert) -> some View {
        let severity = InstructorInsightsLogic.severity(score: alert.score)
        return Label(
            L.format("mobile.instructorInsights.severity.label", L.dynamicText(severity.labelKey), Int(alert.score.rounded())),
            systemImage: "exclamationmark.triangle.fill"
        )
        .font(.caption.weight(.semibold))
        .foregroundStyle(severity == .high ? LexturesTheme.error : LexturesTheme.amber)
        .accessibilityLabel(
            L.format("mobile.instructorInsights.severity.accessibility", L.dynamicText(severity.labelKey), Int(alert.score.rounded()))
        )
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.courseAtRisk(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourseAtRisk(courseCode: course.courseCode, accessToken: token)
            }
            alerts = InstructorInsightsLogic.sortAlerts(result.value.alerts)
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
            errorMessage = nil
        } catch {
            errorMessage = L.text("mobile.instructorInsights.error.atRisk")
        }
    }
}
