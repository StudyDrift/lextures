import SwiftUI

/// Points, streaks, milestone badges, and course leaderboard (M9.3).
struct GamificationView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    @State private var profile: GamificationProfile?
    @State private var studentCourses: [CourseSummary] = []
    @State private var selectedCourseCode: String?
    @State private var leaderboard: CourseLeaderboardResponse?
    @State private var loading = true
    @State private var leaderboardLoading = false
    @State private var freezing = false
    @State private var errorMessage: String?
    @State private var cacheLabel: String?

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                LMSSkeletonList(count: 4)
            } else if let errorMessage, profile == nil {
                LMSEmptyState(
                    systemImage: "flame.fill",
                    title: L.text("mobile.gamification.errorTitle"),
                    message: errorMessage
                )
            } else if let profile {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        if let cacheLabel { StalenessChip(label: cacheLabel) }
                        statsSection(profile)
                        badgesSection(profile)
                        leaderboardSection(profile)
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.gamification.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
        .onChange(of: selectedCourseCode) { _, code in
            guard let code else { return }
            Task { await loadLeaderboard(courseCode: code) }
        }
    }

    @ViewBuilder
    private func statsSection(_ profile: GamificationProfile) -> some View {
        LMSSectionHeader(title: L.text("mobile.gamification.statsTitle"), systemImage: "chart.bar.fill")
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                statRow(
                    label: L.text("mobile.gamification.currentStreak"),
                    value: L.format("mobile.gamification.days", profile.currentStreak),
                    systemImage: "flame.fill"
                )
                if let risk = GamificationLogic.streakRiskMessage(profile: profile) {
                    Text(risk)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.coral)
                }
                statRow(
                    label: L.text("mobile.gamification.longestStreak"),
                    value: L.format("mobile.gamification.days", profile.longestStreak)
                )
                statRow(
                    label: L.text("mobile.gamification.totalXp"),
                    value: "\(profile.xpTotal)"
                )
                statRow(
                    label: L.text("mobile.gamification.level"),
                    value: "\(profile.level)"
                )
                Text(GamificationLogic.levelProgressLabel(profile: profile))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                ProgressView(value: min(max(profile.levelProgressPct / 100, 0), 1))
                    .tint(LexturesTheme.accent(for: colorScheme))
                statRow(
                    label: L.text("mobile.gamification.streakFreezes"),
                    value: "\(profile.streakFreezes)"
                )
                if GamificationLogic.canUseStreakFreeze(profile: profile) {
                    Button(freezing ? L.text("mobile.gamification.freezing") : L.text("mobile.gamification.useFreeze")) {
                        Task { await useFreeze() }
                    }
                    .font(.caption.weight(.semibold))
                    .disabled(freezing)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private func badgesSection(_ profile: GamificationProfile) -> some View {
        LMSSectionHeader(title: L.text("mobile.gamification.badgesTitle"), systemImage: "medal.fill")
        LMSCard {
            let badges = profile.badges ?? []
            if badges.isEmpty {
                Text(L.text("mobile.gamification.noBadges"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(badges) { badge in
                    HStack {
                        Image(systemName: "medal.fill")
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                        VStack(alignment: .leading, spacing: 2) {
                            Text(GamificationLogic.badgeLabel(badge.badgeType))
                                .font(.subheadline.weight(.semibold))
                            Text(CredentialsLogic.issuedDateLabel(iso: badge.awardedAt))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        Spacer(minLength: 0)
                    }
                    .accessibilityLabel(GamificationLogic.badgeLabel(badge.badgeType))
                }
            }
        }
    }

    @ViewBuilder
    private func leaderboardSection(_ profile: GamificationProfile) -> some View {
        LMSSectionHeader(title: L.text("mobile.gamification.leaderboardTitle"), systemImage: "list.number")
        LMSCard {
            if !GamificationLogic.shouldShowLeaderboard(profile: profile) {
                Text(GamificationLogic.leaderboardOptOutMessage())
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else if studentCourses.isEmpty {
                Text(L.text("mobile.gamification.noCoursesForLeaderboard"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                Picker(L.text("mobile.gamification.selectCourse"), selection: Binding(
                    get: { selectedCourseCode ?? studentCourses.first?.courseCode ?? "" },
                    set: { selectedCourseCode = $0 }
                )) {
                    ForEach(studentCourses, id: \.courseCode) { course in
                        Text(course.displayTitle).tag(course.courseCode)
                    }
                }
                .pickerStyle(.menu)

                if leaderboardLoading {
                    ProgressView()
                        .padding(.top, 8)
                } else if let leaderboard {
                    let entries = leaderboard.topEntries ?? []
                    if entries.isEmpty {
                        Text(L.text("mobile.gamification.leaderboardEmpty"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    } else {
                        ForEach(entries) { entry in
                            leaderboardRow(entry)
                        }
                        if let current = leaderboard.currentUser,
                           !(entries.contains { $0.isCurrentUser == true }) {
                            Divider()
                            leaderboardRow(current)
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func statRow(
        label: String,
        value: String,
        systemImage: String? = nil
    ) -> some View {
        HStack {
            if let systemImage {
                Image(systemName: systemImage)
                    .foregroundStyle(LexturesTheme.coral)
            }
            Text(label)
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Spacer(minLength: 0)
            Text(value)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
    }

    @ViewBuilder
    private func leaderboardRow(_ entry: LeaderboardEntry) -> some View {
        HStack {
            Text("#\(entry.rank)")
                .font(.caption.weight(.bold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .frame(width: 28, alignment: .leading)
            Text(entry.displayName)
                .font(.subheadline.weight(entry.isCurrentUser == true ? .bold : .regular))
            Spacer(minLength: 0)
            Text(L.format("mobile.gamification.xpEarned", entry.xpEarned))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .accessibilityLabel(L.format(
            "mobile.gamification.leaderboardRow",
            entry.rank,
            entry.displayName,
            entry.xpEarned
        ))
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = profile == nil
        errorMessage = nil
        defer { loading = false }

        do {
            async let profileTask = offline.cachedFetch(
                key: OfflineCacheKey.gamificationProfile(),
                accessToken: token
            ) {
                try await LMSAPI.fetchGamificationProfile(accessToken: token)
            }
            async let coursesTask = LMSAPI.fetchCourses(accessToken: token)
            let profileResult = try await profileTask
            profile = profileResult.value
            if let cached = profileResult.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
            studentCourses = try await coursesTask.filter(\.viewerIsStudent)
            if selectedCourseCode == nil {
                selectedCourseCode = studentCourses.first?.courseCode
            }
            if let code = selectedCourseCode {
                await loadLeaderboard(courseCode: code)
            }
        } catch {
            errorMessage = L.text("mobile.gamification.loadError")
        }
    }

    private func loadLeaderboard(courseCode: String) async {
        guard let token = session.accessToken else { return }
        leaderboardLoading = true
        defer { leaderboardLoading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.gamificationLeaderboard(courseCode: courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourseLeaderboard(courseCode: courseCode, accessToken: token)
            }
            leaderboard = result.value
        } catch {
            leaderboard = nil
        }
    }

    private func useFreeze() async {
        guard let token = session.accessToken else { return }
        freezing = true
        defer { freezing = false }
        do {
            profile = try await LMSAPI.freezeGamificationStreak(accessToken: token)
        } catch {
            errorMessage = L.text("mobile.gamification.freezeError")
        }
    }
}