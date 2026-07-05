import SwiftUI

/// Instructor evaluation results (M7.7).
struct EvaluationResultsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var results: EvaluationResults?
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var cacheLabel: String?

    var body: some View {
        Group {
            if loading {
                LMSSkeletonList(count: 3)
            } else if let errorMessage, results == nil {
                LMSEmptyState(
                    systemImage: "chart.bar",
                    title: L.text("mobile.evaluations.resultsErrorTitle"),
                    message: errorMessage
                )
            } else if let results {
                resultsContent(results)
            } else {
                LMSEmptyState(
                    systemImage: "chart.bar",
                    title: L.text("mobile.evaluations.noResultsTitle"),
                    message: L.text("mobile.evaluations.noResultsMessage")
                )
            }
        }
        .task { await load() }
        .refreshable { await load(force: true) }
    }

    private func resultsContent(_ results: EvaluationResults) -> some View {
        VStack(alignment: .leading, spacing: 16) {
            if let cacheLabel {
                StalenessChip(label: cacheLabel)
            }

            HStack(spacing: 12) {
                summaryTile(value: "\(results.responseCount)", label: L.text("mobile.evaluations.responses"))
                summaryTile(value: "\(results.enrolledCount)", label: L.text("mobile.evaluations.enrolled"))
                summaryTile(
                    value: "\(Int(results.completionPct.rounded()))%",
                    label: L.text("mobile.evaluations.completion")
                )
            }

            Text(L.format(
                "mobile.evaluations.windowRange",
                EvaluationLogic.formatDeadline(results.opensAt),
                EvaluationLogic.formatDeadline(results.closesAt)
            ))
            .font(.caption)
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            if !results.meetsThreshold {
                LMSCard(accent: LexturesTheme.amber) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(L.text("mobile.evaluations.thresholdTitle"))
                            .font(.subheadline.weight(.semibold))
                        Text(L.text("mobile.evaluations.thresholdMessage"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
            } else {
                ForEach(results.questions) { question in
                    questionResultCard(question)
                }
            }
        }
    }

    private func summaryTile(value: String, label: String) -> some View {
        LMSCard {
            VStack(spacing: 4) {
                Text(value)
                    .font(.title3.weight(.bold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(label)
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .multilineTextAlignment(.center)
            }
            .frame(maxWidth: .infinity)
        }
    }

    @ViewBuilder
    private func questionResultCard(_ question: EvaluationQuestionResult) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text("\(question.index + 1). \(question.text)")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

                switch question.type {
                case .rating:
                    if let average = question.average {
                        Text(L.format("mobile.evaluations.averageRating", String(format: "%.1f", average)))
                            .font(.title3.weight(.bold))
                            .foregroundStyle(LexturesTheme.primary)
                    }
                    ratingDistribution(question.distribution ?? [:])
                case .multipleChoice:
                    choiceDistribution(question.distribution ?? [:])
                case .openText:
                    let texts = question.openTexts ?? []
                    Text(L.format("mobile.evaluations.responseCount", texts.count))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if texts.isEmpty {
                        Text(L.text("mobile.evaluations.noOpenResponses"))
                            .font(.caption)
                            .italic()
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    } else {
                        ForEach(Array(texts.enumerated()), id: \.offset) { _, text in
                            Text(text)
                                .font(.caption)
                                .padding(10)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(LexturesTheme.cardBackground(for: colorScheme))
                                .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        }
                    }
                }
            }
        }
    }

    private func ratingDistribution(_ distribution: [String: Int]) -> some View {
        let max = max(distribution.values.max() ?? 1, 1)
        return VStack(spacing: 6) {
            ForEach(["1", "2", "3", "4", "5"], id: \.self) { rating in
                let count = distribution[rating] ?? 0
                HStack(spacing: 8) {
                    Text(rating)
                        .font(.caption)
                        .frame(width: 16, alignment: .trailing)
                    GeometryReader { geo in
                        ZStack(alignment: .leading) {
                            RoundedRectangle(cornerRadius: 4)
                                .fill(LexturesTheme.cardBackground(for: colorScheme))
                            RoundedRectangle(cornerRadius: 4)
                                .fill(LexturesTheme.primary)
                                .frame(width: geo.size.width * CGFloat(count) / CGFloat(max))
                        }
                    }
                    .frame(height: 12)
                    Text("\(count)")
                        .font(.caption2)
                        .frame(width: 24, alignment: .trailing)
                }
            }
        }
    }

    private func choiceDistribution(_ distribution: [String: Int]) -> some View {
        let total = max(distribution.values.reduce(0, +), 1)
        return VStack(spacing: 6) {
            ForEach(distribution.keys.sorted(), id: \.self) { option in
                let count = distribution[option] ?? 0
                let pct = Int((Double(count) / Double(total) * 100).rounded())
                HStack(spacing: 8) {
                    Text(option)
                        .font(.caption)
                        .lineLimit(1)
                        .frame(minWidth: 72, alignment: .leading)
                    GeometryReader { geo in
                        ZStack(alignment: .leading) {
                            RoundedRectangle(cornerRadius: 4)
                                .fill(LexturesTheme.cardBackground(for: colorScheme))
                            RoundedRectangle(cornerRadius: 4)
                                .fill(LexturesTheme.coral)
                                .frame(width: geo.size.width * CGFloat(count) / CGFloat(total))
                        }
                    }
                    .frame(height: 12)
                    Text("\(count) (\(pct)%)")
                        .font(.caption2)
                }
            }
        }
    }

    private func load(force: Bool = false) async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.evaluationResults(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchEvaluationResults(courseCode: course.courseCode, accessToken: token)
            }
            results = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.evaluations.resultsLoadError")
        }
    }
}
