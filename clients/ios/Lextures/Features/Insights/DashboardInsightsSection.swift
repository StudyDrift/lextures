import SwiftUI

struct DashboardInsightsSection: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let onOpenInsights: () -> Void

    @State private var stats: StudyStats?
    @State private var latestTip: CoachingTip?
    @State private var tipDismissed = false

    var body: some View {
        Group {
            if let stats, stats.optedIn {
                VStack(spacing: 12) {
                    if let tip = latestTip, !tipDismissed {
                        coachingTipCard(tip)
                    }
                    statsCard(stats)
                }
            }
        }
        .task {
            await load()
        }
    }

    private func coachingTipCard(_ tip: CoachingTip) -> some View {
        LMSCard {
            HStack(alignment: .top, spacing: 10) {
                Image(systemName: "lightbulb.fill")
                    .foregroundStyle(LexturesTheme.amber)
                VStack(alignment: .leading, spacing: 6) {
                    Text(L.text("mobile.insights.dashboardCoachingTitle"))
                        .font(.subheadline.weight(.semibold))
                    Text(tip.tipText)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Button(L.text("mobile.insights.dashboardOpen")) { onOpenInsights() }
                        .font(.caption.weight(.semibold))
                }
                Spacer(minLength: 0)
                Button {
                    tipDismissed = true
                } label: {
                    Image(systemName: "xmark")
                        .font(.caption.weight(.semibold))
                }
                .accessibilityLabel(L.text("mobile.insights.dashboardDismissTip"))
            }
        }
    }

    private func statsCard(_ stats: StudyStats) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text(L.text("mobile.insights.dashboardTitle"))
                        .font(.subheadline.weight(.semibold))
                    Spacer(minLength: 0)
                    Button(L.text("mobile.insights.dashboardOpen")) { onOpenInsights() }
                        .font(.caption.weight(.semibold))
                }
                if stats.loginStreakDays > 0 {
                    Label(
                        L.plural("mobile.insights.streak", count: stats.loginStreakDays),
                        systemImage: "flame.fill"
                    )
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.amber)
                }
                let hours = InsightsLogic.hoursFromSeconds(stats.timeOnTaskSecondsThisWeek)
                Text(L.format("mobile.insights.timeOnTask", InsightsLogic.formatHours(hours)))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                if let goal = stats.weeklyGoalHours, goal > 0,
                   let pct = InsightsLogic.goalProgressPercent(
                       progressHours: stats.goalProgressHours,
                       goalHours: goal
                   ) {
                    ProgressView(value: Double(pct), total: 100)
                        .tint(LexturesTheme.primary)
                }
            }
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        do {
            async let statsTask = OfflineService.shared.cachedFetch(
                key: OfflineCacheKey.studyStats(),
                accessToken: token
            ) { try await LMSAPI.fetchStudyStats(accessToken: token) }
            async let tipsTask = OfflineService.shared.cachedFetch(
                key: OfflineCacheKey.coachingTips(),
                accessToken: token
            ) { try await LMSAPI.fetchCoachingTips(accessToken: token) }
            let loadedStats = try await statsTask.value
            stats = loadedStats.optedIn ? loadedStats : nil
            latestTip = InsightsLogic.latestCoachingTip(from: try await tipsTask.value)
        } catch {
            stats = nil
            latestTip = nil
        }
    }
}