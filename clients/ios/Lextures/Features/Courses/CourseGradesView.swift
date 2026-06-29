import SwiftUI

/// "Grades" section of course detail (students): overall summary plus one row per
/// gradebook column, with held / dropped / excused treatments.
struct CourseGradesSection: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    @State private var grades: MyGradesResponse?
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

            if loading {
                LMSSkeletonList(count: 3)
            } else if let grades {
                if grades.columns.isEmpty {
                    LMSEmptyState(
                        systemImage: "checkmark.seal",
                        title: "No graded work yet",
                        message: "Grades will appear here as assignments are graded."
                    )
                } else {
                    summaryCard(grades)
                    ForEach(grades.columns) { column in
                        gradeRow(column, grades: grades)
                    }
                }
            }
        }
        .task { await load() }
    }

    // MARK: Overall summary

    private func summaryCard(_ grades: MyGradesResponse) -> some View {
        let overall = GradeMath.overallPercent(grades)
        return LMSCard {
            HStack(spacing: 16) {
                if let overall {
                    LMSProgressRing(progress: overall / 100, size: 56)
                } else {
                    Image(systemName: "hourglass")
                        .font(.title3)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .frame(width: 56, height: 56)
                        .background(LexturesTheme.brandTeal.opacity(0.1))
                        .clipShape(Circle())
                }
                VStack(alignment: .leading, spacing: 3) {
                    Text(overall.map { String(format: "%.1f%%", $0) } ?? "Not graded yet")
                        .font(LexturesTheme.displayFont(22, weight: .bold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(overall == nil
                        ? "Your overall grade appears once work is graded."
                        : "Current overall grade")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer(minLength: 0)
            }
        }
    }

    // MARK: Rows

    private func gradeRow(_ column: GradeColumn, grades: MyGradesResponse) -> some View {
        let isDropped = grades.droppedGrades[column.id] == true
        let isHeld = grades.heldGradeItemIds.contains(column.id)
        let isExcused = grades.gradeStatuses[column.id] == "excused"
        let score = grades.grades[column.id]
        let display = grades.displayGrades[column.id]

        return LMSCard {
            HStack(spacing: 12) {
                Image(systemName: ItemKind.icon(for: column.kind))
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 32, height: 32)
                    .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.16 : 0.13))
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))

                VStack(alignment: .leading, spacing: 3) {
                    Text(column.title)
                        .font(.subheadline.weight(.medium))
                        .strikethrough(isDropped)
                        .foregroundStyle(
                            isDropped
                                ? LexturesTheme.textSecondary(for: colorScheme)
                                : LexturesTheme.textPrimary(for: colorScheme)
                        )
                    HStack(spacing: 6) {
                        if let due = LMSDates.parse(column.dueAt) {
                            Text("Due \(due.formatted(date: .abbreviated, time: .omitted))")
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        if isDropped {
                            badge("Dropped", tint: LexturesTheme.textSecondary(for: colorScheme))
                        }
                        if isHeld {
                            badge("Held", tint: LexturesTheme.amber)
                        }
                        if isExcused {
                            badge("Excused", tint: LexturesTheme.accent(for: colorScheme))
                        }
                    }
                }

                Spacer(minLength: 0)

                VStack(alignment: .trailing, spacing: 2) {
                    if isExcused {
                        Text("—")
                            .font(LexturesTheme.displayFont(16, weight: .bold))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    } else if isHeld {
                        Image(systemName: "eye.slash")
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.amber)
                    } else if let score, !score.isEmpty {
                        Text(scoreText(score, maxPoints: column.maxPoints))
                            .font(LexturesTheme.displayFont(16, weight: .bold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        if let display, !display.isEmpty, display != score {
                            Text(display)
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                        }
                    } else {
                        Text("Not graded")
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
            }
        }
    }

    private func badge(_ text: String, tint: Color) -> some View {
        Text(text)
            .font(.caption2.weight(.semibold))
            .foregroundStyle(tint)
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(tint.opacity(0.13))
            .clipShape(Capsule())
    }

    private func scoreText(_ score: String, maxPoints: Double?) -> String {
        if let max = maxPoints {
            return "\(score) / \(max.formatted())"
        }
        return score
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
                key: OfflineCacheKey.myGrades(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchMyGrades(courseCode: course.courseCode, accessToken: token)
            }
            grades = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load grades."
        }
    }
}

/// Weighted-total math for `/my-grades` (simplified port of the web logic:
/// per-group earned/possible, weights renormalized over groups that have grades).
enum GradeMath {
    static func overallPercent(_ response: MyGradesResponse) -> Double? {
        struct Tally {
            var earned = 0.0
            var possible = 0.0
        }

        var groupTallies: [String: Tally] = [:]
        var flat = Tally()

        for column in response.columns {
            guard let maxPoints = column.maxPoints, maxPoints > 0 else { continue }
            guard response.droppedGrades[column.id] != true else { continue }
            guard response.gradeStatuses[column.id] != "excused" else { continue }
            guard let raw = response.grades[column.id], let earned = Double(raw) else { continue }

            flat.earned += earned
            flat.possible += maxPoints
            let groupKey = column.assignmentGroupId ?? ""
            groupTallies[groupKey, default: Tally()].earned += earned
            groupTallies[groupKey, default: Tally()].possible += maxPoints
        }

        guard flat.possible > 0 else { return nil }

        let weightedGroups = response.assignmentGroups.filter { $0.weightPercent > 0 }
        guard !weightedGroups.isEmpty else {
            return flat.earned / flat.possible * 100
        }

        // Weighted: only groups with graded work count; renormalize their weights.
        var weightTotal = 0.0
        var weightedSum = 0.0
        for group in weightedGroups {
            guard let tally = groupTallies[group.id], tally.possible > 0 else { continue }
            weightTotal += group.weightPercent
            weightedSum += (tally.earned / tally.possible) * group.weightPercent
        }
        guard weightTotal > 0 else {
            return flat.earned / flat.possible * 100
        }
        return weightedSum / weightTotal * 100
    }
}
