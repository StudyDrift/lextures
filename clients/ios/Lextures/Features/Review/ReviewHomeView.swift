import SwiftUI

struct ReviewRoute: Hashable {}

@MainActor
@Observable
final class ReviewHomeModel {
    var stats: ReviewStats?
    var queue: ReviewQueueResponse?
    var recommendations: [LearnerRecommendationItem] = []
    var selectedCourseCode: String?
    var errorMessage: String?
    var loading = false
    var showSession = false

    var filteredItems: [ReviewQueueItem] {
        ReviewLogic.filterQueue(queue?.items ?? [], courseCode: selectedCourseCode)
    }

    var dueCount: Int {
        if let selectedCourseCode, !selectedCourseCode.isEmpty {
            return filteredItems.count
        }
        return stats?.dueToday ?? queue?.totalDue ?? filteredItems.count
    }

    var courseFilters: [ReviewCourseFilter] {
        ReviewLogic.courseFilters(from: queue?.items ?? [])
    }

    func load(accessToken: String?, userId: String?) async {
        guard let accessToken, let userId else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }

        do {
            async let statsTask = OfflineService.shared.cachedFetch(
                key: OfflineCacheKey.reviewStats(),
                accessToken: accessToken
            ) {
                try await LMSAPI.fetchLearnerReviewStats(userId: userId, accessToken: accessToken)
            }
            async let queueTask = OfflineService.shared.cachedFetch(
                key: OfflineCacheKey.reviewQueue(),
                accessToken: accessToken
            ) {
                try await LMSAPI.fetchLearnerReviewQueue(
                    userId: userId,
                    accessToken: accessToken,
                    limit: ReviewLogic.prefetchLimit
                )
            }

            stats = try await statsTask.value
            queue = try await queueTask.value
            await ReviewReminderScheduler.reschedule(dueCount: stats?.dueToday ?? queue?.totalDue ?? 0)
            await loadRecommendations(accessToken: accessToken, userId: userId)
        } catch {
            errorMessage = L.text("mobile.review.error.load")
        }
    }

    private func loadRecommendations(accessToken: String, userId: String) async {
        guard let courseId = queue?.items.first?.courseId else {
            recommendations = []
            return
        }
        recommendations = (try? await LMSAPI.fetchLearnerRecommendations(
            userId: userId,
            courseId: courseId,
            surface: "review",
            accessToken: accessToken
        ).recommendations) ?? []
    }
}

struct ReviewHomeView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @State private var model = ReviewHomeModel()

    private var userId: String? {
        shell.profile?.id ?? NotebookStore.jwtSubject(from: session.accessToken)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    summaryCard
                    reminderCard
                    if !model.courseFilters.isEmpty {
                        courseFilterChips
                    }
                    if !model.recommendations.isEmpty {
                        recommendationsSection
                    }
                    startSection
                }
                .padding(16)
            }
            .refreshable {
                await model.load(accessToken: session.accessToken, userId: userId)
            }
        }
        .navigationTitle(L.text("mobile.review.title"))
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(isPresented: $model.showSession) {
            ReviewSessionView(
                initialQueue: model.filteredItems,
                totalDue: model.dueCount,
                initialStreak: model.stats?.streak ?? 0,
                onFinished: {
                    Task { await model.load(accessToken: session.accessToken, userId: userId) }
                }
            )
        }
        .task {
            await model.load(accessToken: session.accessToken, userId: userId)
        }
    }

    private var summaryCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                if let error = model.errorMessage {
                    Text(error)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.coral)
                } else if model.loading && model.queue == nil {
                    ProgressView()
                } else {
                    Text(L.plural("mobile.review.dueCount", count: model.dueCount))
                        .font(LexturesTheme.displayFont(28, weight: .bold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    if let streak = model.stats?.streak, streak > 0 {
                        Label(
                            L.plural("mobile.review.streak", count: streak),
                            systemImage: "flame.fill"
                        )
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.amber)
                    } else {
                        Text(L.text("mobile.review.subtitle"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var reminderCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Toggle(isOn: Binding(
                    get: { ReviewReminderScheduler.isEnabled },
                    set: { enabled in
                        ReviewReminderScheduler.isEnabled = enabled
                        Task {
                            if enabled {
                                await PushManager.shared.requestPermissionIfNeeded()
                            }
                            await ReviewReminderScheduler.reschedule(dueCount: model.dueCount)
                        }
                    }
                )) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L.text("mobile.review.reminder.label"))
                            .font(.subheadline.weight(.semibold))
                        Text(L.text("mobile.review.reminder.hint"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }

                if ReviewReminderScheduler.isEnabled {
                    DatePicker(
                        L.text("mobile.review.reminder.time"),
                        selection: reminderBinding,
                        displayedComponents: .hourAndMinute
                    )
                    .labelsHidden()
                    .datePickerStyle(.compact)
                }
            }
        }
    }

    private var reminderBinding: Binding<Date> {
        Binding(
            get: {
                var components = DateComponents()
                components.hour = ReviewReminderScheduler.reminderHour
                components.minute = ReviewReminderScheduler.reminderMinute
                return Calendar.current.date(from: components) ?? Date()
            },
            set: { date in
                let components = Calendar.current.dateComponents([.hour, .minute], from: date)
                ReviewReminderScheduler.reminderHour = components.hour ?? 18
                ReviewReminderScheduler.reminderMinute = components.minute ?? 0
                Task { await ReviewReminderScheduler.reschedule(dueCount: model.dueCount) }
            }
        )
    }

    private var courseFilterChips: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                filterChip(title: L.text("mobile.review.filter.all"), selected: model.selectedCourseCode == nil) {
                    model.selectedCourseCode = nil
                }
                ForEach(model.courseFilters) { filter in
                    filterChip(title: filter.courseTitle, selected: model.selectedCourseCode == filter.courseCode) {
                        model.selectedCourseCode = filter.courseCode
                    }
                }
            }
        }
    }

    private func filterChip(title: String, selected: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(.caption.weight(.semibold))
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .background(selected ? LexturesTheme.primary.opacity(0.16) : LexturesTheme.cardBackground(for: colorScheme))
                .foregroundStyle(selected ? LexturesTheme.primary : LexturesTheme.textSecondary(for: colorScheme))
                .clipShape(Capsule())
        }
        .buttonStyle(.plain)
    }

    private var recommendationsSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            LMSSectionHeader(title: L.text("mobile.review.recommendations"), systemImage: "sparkles")
            ForEach(model.recommendations.prefix(3)) { item in
                LMSCard {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(item.title)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(item.reason)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
        }
    }

    private var startSection: some View {
        VStack(spacing: 12) {
            if model.filteredItems.isEmpty && !model.loading {
                LMSEmptyState(
                    systemImage: "checkmark.circle",
                    title: L.text("mobile.review.caughtUpTitle"),
                    message: L.text("mobile.review.caughtUpMessage")
                )
            } else {
                Button {
                    model.showSession = true
                } label: {
                    Text(model.dueCount > 0 ? L.text("mobile.review.start") : L.text("mobile.review.open"))
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.primary)
                .disabled(model.filteredItems.isEmpty || model.loading)
            }
        }
    }
}
