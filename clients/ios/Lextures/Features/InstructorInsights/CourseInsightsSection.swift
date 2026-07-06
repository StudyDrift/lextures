import SwiftUI

/// Staff course health snapshot: at-risk count, ungraded backlog, and engagement highlights (M11.3).
struct CourseInsightsSection: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL
    let course: CourseSummary

    @State private var snapshot: CourseHealthSnapshot?
    @State private var workingWell: [InstructorSignalItem] = []
    @State private var needsAttention: [InstructorSignalItem] = []
    @State private var cacheLabel: String?
    @State private var atRiskError: String?
    @State private var insightsError: String?
    @State private var backlogError: String?
    @State private var loading = true

    private var features: MobilePlatformFeatures { shell.platformFeatures }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if !NetworkMonitor.shared.isOnline {
                OfflineBanner()
            }
            if let cacheLabel {
                StalenessChip(label: cacheLabel)
            }

            Text(L.text("mobile.instructorInsights.predictedNotice"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            if loading && snapshot == nil {
                LMSSkeletonList(count: 3)
            } else {
                snapshotCards
                navigationCards
                webLinkCard
            }
        }
        .task { await load() }
    }

    private var snapshotCards: some View {
        VStack(spacing: 10) {
            if features.atRiskAlertsEnabled {
                snapshotCard(
                    title: L.text("mobile.instructorInsights.snapshot.atRisk"),
                    value: snapshot.map { L.plural("mobile.instructorInsights.snapshot.atRiskCount", count: $0.atRiskCount) }
                        ?? "—",
                    systemImage: "exclamationmark.triangle",
                    error: atRiskError
                )
            }

            if course.viewerIsStaff {
                Button {
                    shell.activeCourseSection = .grading
                } label: {
                    snapshotCard(
                        title: L.text("mobile.instructorInsights.snapshot.ungraded"),
                        value: snapshot.map { L.plural("mobile.instructorInsights.snapshot.ungradedCount", count: $0.ungradedCount) }
                            ?? "—",
                        systemImage: "tray",
                        error: backlogError,
                        showsChevron: true
                    )
                }
                .buttonStyle(.plain)
            }

            if features.instructorInsightsEnabled {
                snapshotCard(
                    title: L.text("mobile.instructorInsights.snapshot.engagement"),
                    value: snapshot.map {
                        L.plural("mobile.instructorInsights.snapshot.engagementCount", count: $0.engagementHighlightCount)
                    } ?? "—",
                    systemImage: "chart.line.uptrend.xyaxis",
                    error: insightsError
                )
            }
        }
    }

    private var navigationCards: some View {
        VStack(spacing: 10) {
            if features.atRiskAlertsEnabled {
                NavigationLink(value: InstructorInsightsRoute.atRiskList) {
                    actionRow(
                        title: L.text("mobile.instructorInsights.atRisk.title"),
                        subtitle: L.text("mobile.instructorInsights.atRisk.subtitle"),
                        systemImage: "person.crop.circle.badge.exclamationmark"
                    )
                }
                .buttonStyle(.plain)
            }

            if features.instructorInsightsEnabled {
                NavigationLink(value: InstructorInsightsRoute.whatsWorking) {
                    actionRow(
                        title: L.text("mobile.instructorInsights.whatsWorking.title"),
                        subtitle: L.text("mobile.instructorInsights.whatsWorking.subtitle"),
                        systemImage: "lightbulb"
                    )
                }
                .buttonStyle(.plain)
            }
        }
    }

    private var webLinkCard: some View {
        Button {
            let path = InstructorInsightsLogic.webReportsPath(courseCode: course.courseCode)
            openURL(AppConfiguration.webURL(path: path))
        } label: {
            LMSCard {
                HStack(spacing: 10) {
                    Image(systemName: "safari")
                        .foregroundStyle(LexturesTheme.brandTeal)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L.text("mobile.instructorInsights.webReports"))
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(L.text("mobile.instructorInsights.webReportsHint"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer(minLength: 0)
                    Image(systemName: "arrow.up.right")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
        .buttonStyle(.plain)
    }

    private func snapshotCard(
        title: String,
        value: String?,
        systemImage: String,
        error: String?,
        showsChevron: Bool = false
    ) -> some View {
        LMSCard {
            HStack(spacing: 12) {
                Image(systemName: systemImage)
                    .font(.title3)
                    .foregroundStyle(LexturesTheme.brandTeal)
                    .accessibilityHidden(true)
                VStack(alignment: .leading, spacing: 4) {
                    Text(title)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text(value ?? "—")
                        .font(LexturesTheme.displayFont(22))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    if let error {
                        Text(error)
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.error)
                    }
                }
                Spacer(minLength: 0)
                if showsChevron {
                    Image(systemName: "chevron.right")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    private func actionRow(title: String, subtitle: String, systemImage: String) -> some View {
        LMSCard {
            HStack(spacing: 12) {
                Image(systemName: systemImage)
                    .foregroundStyle(LexturesTheme.brandTeal)
                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(subtitle)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer(minLength: 0)
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        defer { loading = false }

        var atRiskCount = 0
        var ungradedCount = 0
        var loadedWorking: [InstructorSignalItem] = []
        var loadedAttention: [InstructorSignalItem] = []
        var staleLabel: String?

        if features.atRiskAlertsEnabled {
            do {
                let result = try await offline.cachedFetch(
                    key: OfflineCacheKey.courseAtRisk(course.courseCode),
                    accessToken: token
                ) {
                    try await LMSAPI.fetchCourseAtRisk(courseCode: course.courseCode, accessToken: token)
                }
                atRiskCount = result.value.alerts.count
                if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                    staleLabel = cached.lastUpdatedLabel
                }
                atRiskError = nil
            } catch {
                atRiskError = L.text("mobile.instructorInsights.error.atRisk")
            }
        }

        if course.viewerIsStaff {
            do {
                let items = try await LMSAPI.fetchGradingBacklog(courseCode: course.courseCode, accessToken: token)
                ungradedCount = items.count
                backlogError = nil
            } catch {
                backlogError = L.text("mobile.instructorInsights.error.backlog")
            }
        }

        if features.instructorInsightsEnabled {
            do {
                let result = try await offline.cachedFetch(
                    key: OfflineCacheKey.courseInstructorInsights(course.courseCode),
                    accessToken: token
                ) {
                    try await LMSAPI.fetchInstructorInsights(courseCode: course.courseCode, accessToken: token)
                }
                loadedWorking = result.value.workingWell
                loadedAttention = result.value.needsAttention
                if staleLabel == nil,
                   let cached = result.cached,
                   cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                    staleLabel = cached.lastUpdatedLabel
                }
                insightsError = nil
            } catch {
                insightsError = L.text("mobile.instructorInsights.error.insights")
            }
        }

        snapshot = InstructorInsightsLogic.snapshot(
            atRiskCount: atRiskCount,
            ungradedCount: ungradedCount,
            workingWell: loadedWorking,
            needsAttention: loadedAttention
        )
        workingWell = loadedWorking
        needsAttention = loadedAttention
        cacheLabel = staleLabel
    }
}
