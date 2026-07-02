import SwiftUI

@MainActor
@Observable
final class InsightsModel {
    var stats: StudyStats?
    var journal: [ReflectionJournalEntry] = []
    var tips: [CoachingTip] = []
    var courseProgress: [CourseProgressSummary] = []
    var studentCourses: [CourseSummary] = []
    var weeklyHours: Float = 10
    var optedIn = false
    var remindersEnabled = false
    var journalDraft = ""
    var errorMessage: String?
    var loading = false
    var saving = false

    func load(accessToken: String?) async {
        guard let accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }

        do {
            let goal = try? await LMSAPI.fetchStudyGoal(accessToken: accessToken)
            if let goal {
                optedIn = goal.optedIn
                if goal.weeklyHours > 0 { weeklyHours = goal.weeklyHours }
            } else if let probe = try? await LMSAPI.fetchStudyStats(accessToken: accessToken) {
                optedIn = probe.optedIn
                if let hours = probe.weeklyGoalHours, hours > 0 { weeklyHours = hours }
            }

            guard optedIn else {
                stats = nil
                journal = []
                tips = []
                courseProgress = []
                return
            }

            async let statsTask = OfflineService.shared.cachedFetch(
                key: OfflineCacheKey.studyStats(),
                accessToken: accessToken
            ) { try await LMSAPI.fetchStudyStats(accessToken: accessToken) }
            async let journalTask = OfflineService.shared.cachedFetch(
                key: OfflineCacheKey.reflectionJournal(),
                accessToken: accessToken
            ) { try await LMSAPI.fetchReflectionJournal(accessToken: accessToken) }
            async let tipsTask = OfflineService.shared.cachedFetch(
                key: OfflineCacheKey.coachingTips(),
                accessToken: accessToken
            ) { try await LMSAPI.fetchCoachingTips(accessToken: accessToken) }
            async let coursesTask = LMSAPI.fetchCourses(accessToken: accessToken)
            async let remindersTask = LMSAPI.fetchReminderConfig(accessToken: accessToken)

            stats = try await statsTask.value
            journal = try await journalTask.value
            let tipsResponse = try await tipsTask.value
            tips = tipsResponse.history ?? []
            if let hours = stats?.weeklyGoalHours, hours > 0 { weeklyHours = hours }

            let courses = try await coursesTask
            studentCourses = courses.filter(\.viewerIsStudent)
            var summaries: [CourseProgressSummary] = []
            for course in studentCourses.prefix(8) {
                if let snapshot = try? await LMSAPI.fetchModulesProgress(
                    courseCode: course.courseCode,
                    accessToken: accessToken
                ) {
                    summaries.append(CourseProgressSummary(
                        courseCode: course.courseCode,
                        title: course.displayTitle,
                        percentComplete: InsightsLogic.moduleCompletionPercent(snapshot)
                    ))
                }
            }
            courseProgress = summaries.sorted { $0.percentComplete > $1.percentComplete }

            if let config = try? await remindersTask {
                remindersEnabled = config.enabled
            }
        } catch {
            errorMessage = L.text("mobile.insights.error.load")
        }
    }

    func saveGoals(accessToken: String?) async {
        guard let accessToken else { return }
        saving = true
        defer { saving = false }
        do {
            _ = try await LMSAPI.putStudyGoal(
                body: PutStudyGoalBody(weeklyHours: weeklyHours, optedIn: optedIn),
                accessToken: accessToken
            )
            await load(accessToken: accessToken)
        } catch {
            errorMessage = L.text("mobile.insights.error.saveGoals")
        }
    }

    func addJournalEntry(accessToken: String?, offline: OfflineService) async {
        guard let accessToken, InsightsLogic.journalEntryValid(journalDraft) else { return }
        let text = journalDraft.trimmingCharacters(in: .whitespacesAndNewlines)
        journalDraft = ""
        do {
            if NetworkMonitor.shared.isOnline {
                _ = try await LMSAPI.createReflectionJournalEntry(
                    body: PostReflectionJournalBody(entryText: text),
                    accessToken: accessToken
                )
            } else {
                struct Body: Encodable { var entryText: String }
                _ = try await offline.enqueueMutation(
                    method: "POST",
                    path: "/api/v1/me/reflection-journal",
                    body: Body(entryText: text),
                    label: L.text("mobile.insights.journal.addLabel"),
                    accessToken: accessToken
                )
            }
            await load(accessToken: accessToken)
        } catch {
            errorMessage = L.text("mobile.insights.error.journal")
        }
    }

    func deleteJournalEntry(id: String, accessToken: String?) async {
        guard let accessToken else { return }
        do {
            try await LMSAPI.deleteReflectionJournalEntry(id: id, accessToken: accessToken)
            await load(accessToken: accessToken)
        } catch {
            errorMessage = L.text("mobile.insights.error.journal")
        }
    }

    func rateTip(id: String, rating: Int, accessToken: String?) async {
        guard let accessToken else { return }
        _ = try? await LMSAPI.rateCoachingTip(id: id, rating: rating, accessToken: accessToken)
        await load(accessToken: accessToken)
    }

    func toggleReminders(accessToken: String?, enabled: Bool) async {
        guard let accessToken else { return }
        do {
            let config = try await LMSAPI.patchReminderConfig(enabled: enabled, accessToken: accessToken)
            remindersEnabled = config.enabled
        } catch {
            errorMessage = L.text("mobile.insights.error.reminders")
        }
    }
}

struct InsightsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @State private var model = InsightsModel()
    var onOpenCourse: ((CourseSummary) -> Void)?
    var onOpenReview: (() -> Void)?

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if model.loading && model.stats == nil && model.optedIn {
                LMSSkeletonList(count: 4)
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        if let error = model.errorMessage {
                            LMSErrorBanner(message: error)
                        }
                        goalsCard
                        if model.optedIn {
                            if let stats = model.stats {
                                weekCard(stats)
                                if !model.courseProgress.isEmpty {
                                    courseProgressCard
                                }
                                if !stats.timeAllocation.isEmpty {
                                    allocationCard(stats.timeAllocation)
                                }
                            }
                            journalCard
                            if !model.tips.isEmpty {
                                tipsCard
                            }
                            remindersCard
                            actionsCard
                        }
                    }
                    .padding(16)
                }
                .refreshable { await model.load(accessToken: session.accessToken) }
            }
        }
        .navigationTitle(L.text("mobile.insights.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.load(accessToken: session.accessToken) }
    }

    private var goalsCard: some View {
        LMSCard {
            Text(L.text("mobile.insights.goalsTitle"))
                .font(LexturesTheme.displayFont(17))
            Toggle(L.text("mobile.insights.optIn"), isOn: $model.optedIn)
                .font(.subheadline)
            if model.optedIn {
                Text(L.format("mobile.insights.weeklyGoal", Double(model.weeklyHours)))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Slider(value: Binding(
                    get: { Double(model.weeklyHours) },
                    set: { model.weeklyHours = Float($0) }
                ), in: 0 ... 40, step: 0.5)
                .accessibilityLabel(L.text("mobile.insights.weeklyGoalAccessibility"))
            }
            Button(model.saving ? L.text("mobile.insights.saving") : L.text("mobile.insights.saveGoals")) {
                Task { await model.saveGoals(accessToken: session.accessToken) }
            }
            .font(.caption.weight(.semibold))
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.primary)
            .disabled(model.saving)
        }
    }

    private func weekCard(_ stats: StudyStats) -> some View {
        LMSCard {
            Text(L.text("mobile.insights.thisWeek"))
                .font(LexturesTheme.displayFont(17))
            if stats.loginStreakDays > 0 {
                Label(
                    L.plural("mobile.insights.streak", count: stats.loginStreakDays),
                    systemImage: "flame.fill"
                )
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.amber)
                .accessibilityLabel(L.plural("mobile.insights.streakAccessibility", count: stats.loginStreakDays))
            } else {
                Text(L.text("mobile.insights.streakEmpty"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            let hours = InsightsLogic.hoursFromSeconds(stats.timeOnTaskSecondsThisWeek)
            Text(L.format("mobile.insights.timeOnTask", InsightsLogic.formatHours(hours)))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            if let goal = stats.weeklyGoalHours, goal > 0,
               let pct = InsightsLogic.goalProgressPercent(progressHours: stats.goalProgressHours, goalHours: goal) {
                ProgressView(value: Double(pct), total: 100)
                    .tint(LexturesTheme.primary)
                    .accessibilityLabel(L.format(
                        "mobile.insights.goalProgressAccessibility",
                        InsightsLogic.formatHours(Double(stats.goalProgressHours)),
                        InsightsLogic.formatHours(Double(goal))
                    ))
            }
            if stats.lowStudyEfficiency {
                Text(L.text("mobile.insights.lowEfficiency"))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.amber)
            }
        }
    }

    private var courseProgressCard: some View {
        LMSCard {
            Text(L.text("mobile.insights.courseProgress"))
                .font(LexturesTheme.displayFont(17))
            ForEach(model.courseProgress) { row in
                Button {
                    if let course = model.studentCourses.first(where: { $0.courseCode == row.courseCode }) {
                        onOpenCourse?(course)
                    }
                } label: {
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(row.title)
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Spacer(minLength: 0)
                            Text("\(row.percentComplete)%")
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        ProgressView(value: Double(row.percentComplete), total: 100)
                            .tint(LexturesTheme.primary)
                    }
                }
                .buttonStyle(.plain)
            }
        }
    }

    private func allocationCard(_ rows: [StudyTimeAllocationRow]) -> some View {
        let maxMinutes = InsightsLogic.maxAllocationMinutes(rows)
        return LMSCard {
            Text(L.text("mobile.insights.timeAllocation"))
                .font(LexturesTheme.displayFont(17))
            ForEach(rows) { row in
                VStack(alignment: .leading, spacing: 4) {
                    HStack {
                        Text(row.moduleTitle)
                            .font(.caption)
                            .lineLimit(1)
                        Spacer(minLength: 0)
                        Text(L.format("mobile.insights.minutes", Int(row.minutes.rounded())))
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    GeometryReader { geo in
                        let width = geo.size.width * InsightsLogic.barWidthPercent(
                            minutes: row.minutes,
                            maxMinutes: maxMinutes
                        ) / 100
                        RoundedRectangle(cornerRadius: 4)
                            .fill(LexturesTheme.primary.opacity(0.7))
                            .frame(width: max(0, width), height: 8)
                    }
                    .frame(height: 8)
                    .accessibilityLabel(L.format(
                        "mobile.insights.allocationAccessibility",
                        row.moduleTitle,
                        Int(row.minutes.rounded())
                    ))
                }
            }
        }
    }

    private var journalCard: some View {
        LMSCard {
            Text(L.text("mobile.insights.journalTitle"))
                .font(LexturesTheme.displayFont(17))
            Text(L.text("mobile.insights.journalHint"))
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            TextField(L.text("mobile.insights.journalPlaceholder"), text: $model.journalDraft, axis: .vertical)
                .lineLimit(3 ... 6)
                .textFieldStyle(.roundedBorder)
            Button(L.text("mobile.insights.journalAdd")) {
                Task { await model.addJournalEntry(accessToken: session.accessToken, offline: offline) }
            }
            .font(.caption.weight(.semibold))
            .disabled(!InsightsLogic.journalEntryValid(model.journalDraft))
            ForEach(model.journal) { entry in
                Divider()
                VStack(alignment: .leading, spacing: 4) {
                    Text(InsightsLogic.formatJournalDate(entry.createdAt))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text(entry.entryText)
                        .font(.subheadline)
                    Button(L.text("mobile.insights.journalDelete"), role: .destructive) {
                        Task { await model.deleteJournalEntry(id: entry.id, accessToken: session.accessToken) }
                    }
                    .font(.caption2)
                }
            }
        }
    }

    private var tipsCard: some View {
        LMSCard {
            Text(L.text("mobile.insights.coachingTitle"))
                .font(LexturesTheme.displayFont(17))
            ForEach(model.tips) { tip in
                VStack(alignment: .leading, spacing: 6) {
                    Text(L.format("mobile.insights.coachingWeek", tip.weekOf))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text(tip.tipText)
                        .font(.subheadline)
                    HStack(spacing: 12) {
                        Button(L.text("mobile.insights.coachingHelpful")) {
                            Task { await model.rateTip(id: tip.id, rating: 1, accessToken: session.accessToken) }
                        }
                        Button(L.text("mobile.insights.coachingNotHelpful")) {
                            Task { await model.rateTip(id: tip.id, rating: -1, accessToken: session.accessToken) }
                        }
                    }
                    .font(.caption2)
                }
                .padding(.vertical, 4)
            }
        }
    }

    private var remindersCard: some View {
        LMSCard {
            Text(L.text("mobile.insights.remindersTitle"))
                .font(LexturesTheme.displayFont(17))
            Toggle(L.text("mobile.insights.remindersToggle"), isOn: Binding(
                get: { model.remindersEnabled },
                set: { newValue in Task { await model.toggleReminders(accessToken: session.accessToken, enabled: newValue) } }
            ))
            .font(.subheadline)
        }
    }

    private var actionsCard: some View {
        LMSCard {
            Text(L.text("mobile.insights.actionsTitle"))
                .font(LexturesTheme.displayFont(17))
            if let onOpenReview {
                Button(L.text("mobile.insights.openReview")) { onOpenReview() }
                    .font(.caption.weight(.semibold))
            }
        }
    }
}