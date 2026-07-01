import SwiftUI

/// Per-standard proficiency + compact mastery heatmap for a course (M6.2).
struct CourseMasterySection: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    @State private var rows: [MasteryConceptRow] = []
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true

    var body: some View {
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

            if loading && rows.isEmpty {
                LMSSkeletonList(count: 3)
            } else if rows.isEmpty {
                LMSEmptyState(
                    systemImage: "chart.bar.doc.horizontal",
                    title: L.text("mobile.mastery.emptyTitle"),
                    message: L.text("mobile.mastery.emptyMessage")
                )
            } else {
                summaryCard
                ForEach(rows) { row in
                    conceptRow(row)
                }
            }
        }
        .task { await load() }
        .refreshable { await load() }
    }

    private var summaryCard: some View {
        let summary = MasteryLogic.summary(rows)
        return LMSCard {
            Text(L.format("mobile.mastery.summary", summary.mastered, summary.total))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            if summary.atRisk > 0 {
                Text(L.format("mobile.mastery.summaryAtRisk", summary.atRisk))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
    }

    private func conceptRow(_ row: MasteryConceptRow) -> some View {
        LMSCard {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 6) {
                    Text(row.name)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(L.dynamicText(row.level.labelKey))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(color(for: row.level))
                    if !row.assessed {
                        Text(L.text("mobile.mastery.practiceHint"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
                Spacer(minLength: 0)
                Circle()
                    .fill(color(for: row.level))
                    .frame(width: 14, height: 14)
                    .accessibilityHidden(true)
            }
        }
        .accessibilityElement(children: .combine)
    }

    private func color(for level: MasteryLevel) -> Color {
        switch level {
        case .mastered: return LexturesTheme.brandTeal
        case .developing: return LexturesTheme.amber
        case .beginning: return LexturesTheme.coral
        case .atRisk: return LexturesTheme.error
        case .notAssessed: return LexturesTheme.textSecondary(for: colorScheme)
        }
    }

    private func load() async {
        guard let token = session.accessToken,
              let enrollmentId = course.viewerStudentEnrollmentId else {
            loading = false
            return
        }
        loading = true
        errorMessage = nil
        defer { loading = false }

        do {
            let result = try await offline.cachedFetch(
                key: MasteryLogic.cacheKeyMastery(courseCode: course.courseCode, enrollmentId: enrollmentId),
                accessToken: token
            ) {
                try await LMSAPI.fetchStudentMastery(
                    courseCode: course.courseCode,
                    enrollmentId: enrollmentId,
                    accessToken: token
                )
            }
            rows = MasteryLogic.rows(from: result.value)
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = L.text("mobile.mastery.loadError")
        }
    }
}
