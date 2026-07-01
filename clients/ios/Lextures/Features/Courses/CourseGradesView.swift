import SwiftUI

/// "Grades" section of course detail (students): categories, totals, what-if, feedback detail (M6.1).
struct CourseGradesSection: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    @State private var grades: MyGradesResponse?
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var whatIf = WhatIfController()
    @State private var showWhatIfFeature = false
    @State private var feedbackRoute: GradeFeedbackRoute?

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
                    if showWhatIfFeature {
                        whatIfPanel(grades)
                    }
                    ForEach(GradesDisplayLogic.buildSections(from: grades)) { section in
                        categoryHeader(section)
                        ForEach(section.columns) { column in
                            gradeRow(column, grades: grades, section: section)
                        }
                    }
                }
            }
        }
        .navigationDestination(item: $feedbackRoute) { route in
            GradeFeedbackView(course: course, column: route.column)
        }
        .task { await load() }
    }

    // MARK: Summary

    private func summaryCard(_ grades: MyGradesResponse) -> some View {
        let actual = whatIf.actualPercent(for: grades)
        let projected = whatIf.projectedPercent(for: grades)
        let display = whatIf.mode && whatIf.hasOverrides ? projected : actual

        return LMSCard {
            HStack(spacing: 16) {
                if let display {
                    LMSProgressRing(progress: display / 100, size: 56)
                } else {
                    Image(systemName: "hourglass")
                        .font(.title3)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .frame(width: 56, height: 56)
                        .background(LexturesTheme.brandTeal.opacity(0.1))
                        .clipShape(Circle())
                }
                VStack(alignment: .leading, spacing: 3) {
                    if whatIf.mode && whatIf.hasOverrides {
                        Text("Hypothetical: \(GradeCalculator.formatFinalPercent(projected))")
                            .font(LexturesTheme.displayFont(20, weight: .bold))
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                        Text("Actual: \(GradeCalculator.formatFinalPercent(actual))")
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    } else {
                        Text(GradeCalculator.formatFinalPercent(actual))
                            .font(LexturesTheme.displayFont(22, weight: .bold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(actual == nil
                            ? "Your overall grade appears once work is graded."
                            : "Current overall grade")
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
                Spacer(minLength: 0)
            }
        }
    }

    private func whatIfPanel(_ grades: MyGradesResponse) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Toggle(isOn: Binding(
                    get: { whatIf.mode },
                    set: { _ in whatIf.toggleMode() }
                )) {
                    Label("What-if grades", systemImage: "flask")
                        .font(.subheadline.weight(.semibold))
                }
                if whatIf.mode {
                    Text("Enter hypothetical scores below. These projections are not saved and do not change your real grades.")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if whatIf.hasOverrides {
                        Button("Reset hypothetical scores") { whatIf.reset() }
                            .font(.caption.weight(.semibold))
                    }
                }
            }
        }
    }

    private func categoryHeader(_ section: GradesDisplayLogic.Section) -> some View {
        HStack {
            Text(section.title)
                .font(.caption.weight(.bold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            if let weight = section.weightPercent {
                Text("\(Int(weight))%")
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            }
            Spacer()
        }
        .padding(.top, 4)
    }

    // MARK: Rows

    private func gradeRow(
        _ column: GradeColumn,
        grades: MyGradesResponse,
        section: GradesDisplayLogic.Section
    ) -> some View {
        let dropped = whatIf.activeDropped(for: grades)
        let isDropped = dropped[column.id] == true
        let isHeld = grades.heldGradeItemIds.contains(column.id)
        let isExcused = grades.gradeStatuses[column.id] == "excused"
        let score = grades.grades[column.id]
        let display = grades.displayGrades[column.id]
        let hasOverride = !(whatIf.overrides[column.id] ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        let isHypothetical = whatIf.mode && hasOverride
        let badges = GradesDisplayLogic.statusBadges(column: column, response: grades, dropped: dropped)
        let canOpenFeedback = !isHeld && column.kind == "assignment"
            && (!(score ?? "").isEmpty || isExcused || grades.gradeStatuses[column.id] == "graded")

        return Button {
            if canOpenFeedback {
                feedbackRoute = GradeFeedbackRoute(column: column)
            }
        } label: {
            LMSCard {
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
                            ForEach(badges, id: \.self) { badge in
                                gradeBadge(badge, tint: badgeTint(badge))
                            }
                            if isHypothetical {
                                gradeBadge("Hypothetical", tint: LexturesTheme.accent(for: colorScheme))
                            }
                        }
                    }

                    Spacer(minLength: 0)

                    if whatIf.mode && showWhatIfFeature && !isExcused && !isHeld,
                       let max = column.maxPoints, max > 0 {
                        TextField(
                            "Score",
                            text: Binding(
                                get: { whatIf.overrides[column.id] ?? "" },
                                set: { whatIf.setOverride(itemId: column.id, value: $0) }
                            )
                        )
                        .keyboardType(.decimalPad)
                        .multilineTextAlignment(.trailing)
                        .frame(width: 64)
                        .font(LexturesTheme.displayFont(14, weight: .bold))
                        .accessibilityLabel("Hypothetical score for \(column.title)")
                    } else {
                        scoreColumn(
                            isExcused: isExcused,
                            isHeld: isHeld,
                            score: score,
                            display: display,
                            maxPoints: column.maxPoints
                        )
                    }

                    if canOpenFeedback {
                        Image(systemName: "chevron.right")
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
            }
        }
        .buttonStyle(.plain)
        .disabled(!canOpenFeedback && !whatIf.mode)
    }

    @ViewBuilder
    private func scoreColumn(
        isExcused: Bool,
        isHeld: Bool,
        score: String?,
        display: String?,
        maxPoints: Double?
    ) -> some View {
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
                Text(scoreText(score, maxPoints: maxPoints))
                    .font(LexturesTheme.displayFont(16, weight: .bold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                if let display, !display.isEmpty, display != score {
                    Text(display)
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                }
                if let maxPoints, maxPoints > 0, let earned = Double(score.replacingOccurrences(of: ",", with: "")) {
                    Text(String(format: "%.1f%%", earned / maxPoints * 100))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            } else {
                Text("Not graded")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func gradeBadge(_ text: String, tint: Color) -> some View {
        Text(text)
            .font(.caption2.weight(.semibold))
            .foregroundStyle(tint)
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(tint.opacity(0.13))
            .clipShape(Capsule())
    }

    private func badgeTint(_ badge: String) -> Color {
        switch badge {
        case "Dropped": return LexturesTheme.textSecondary(for: colorScheme)
        case "Pending", "Late": return LexturesTheme.amber
        case "Excused": return LexturesTheme.accent(for: colorScheme)
        default: return LexturesTheme.textSecondary(for: colorScheme)
        }
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
            async let featuresTask = try? LMSAPI.fetchPlatformFeatures(accessToken: token)
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.myGrades(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchMyGrades(courseCode: course.courseCode, accessToken: token)
            }
            grades = result.value
            if let features = await featuresTask {
                showWhatIfFeature = features.ffWhatifGrades == true
            }
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
