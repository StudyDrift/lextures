import SwiftUI

/// Live meetings section embedded in the course workspace (M7.5).
struct CourseLiveSection: View {
    let course: CourseSummary

    var body: some View {
        MeetingListView(course: course)
    }
}

struct MeetingListView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL
    let course: CourseSummary

    @State private var meetings: [VirtualMeeting] = []
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var selectedMeeting: VirtualMeeting?
    @State private var openWhiteboard: CourseWhiteboard?
    @State private var pollTask: Task<Void, Never>?

    private var grouped: LiveMeetingsLogic.GroupedMeetings {
        LiveMeetingsLogic.groupMeetings(meetings)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if shell.platformFeatures.ffMobileLiveMeetings == false {
                CourseDestinationPlaceholder(section: .live)
            } else {
                liveContent
            }
        }
        .task { await load() }
        .onDisappear { pollTask?.cancel() }
        .sheet(item: $selectedMeeting) { meeting in
            MeetingDetailView(
                course: course,
                meeting: meeting,
                onUpdated: { updated in
                    replaceMeeting(updated)
                    if updated.status == "cancelled" {
                        meetings.removeAll { $0.id == updated.id }
                        selectedMeeting = nil
                    }
                },
                onOpenWhiteboard: { board in
                    selectedMeeting = nil
                    openWhiteboard = board
                }
            )
        }
        .sheet(item: $openWhiteboard) { board in
            WhiteboardView(
                course: course,
                board: board,
                canEdit: WhiteboardLogic.canEdit(
                    viewerIsStaff: course.viewerIsStaff,
                    features: shell.platformFeatures
                ),
                onDeleted: { openWhiteboard = nil }
            )
        }
    }

    @ViewBuilder
    private var liveContent: some View {
        if !NetworkMonitor.shared.isOnline {
            OfflineBanner()
        }
        if let cacheLabel {
            StalenessChip(label: cacheLabel)
        }
        if let errorMessage {
            LMSErrorBanner(message: errorMessage)
        }

        if !grouped.live.isEmpty {
            liveBanner
        }

        if loading {
            LMSSkeletonList(count: 3)
        } else if meetings.isEmpty {
            LMSEmptyState(
                systemImage: "video",
                title: L.text("mobile.live.empty.title"),
                message: course.viewerIsStaff
                    ? L.text("mobile.live.empty.staffMessage")
                    : L.text("mobile.live.empty.message")
            )
        } else {
            meetingSections
        }

        if course.viewerIsStaff {
            Text(L.text("mobile.live.manageOnWeb"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private var liveBanner: some View {
        LMSCard(accent: LexturesTheme.brandCoral) {
            HStack {
                HStack(spacing: 8) {
                    Circle()
                        .fill(LexturesTheme.brandCoral)
                        .frame(width: 10, height: 10)
                        .accessibilityHidden(true)
                    Text(
                        grouped.live.count > 1
                            ? L.format("mobile.live.banner.multiple", "\(grouped.live.count)")
                            : L.text("mobile.live.banner.single")
                    )
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                }
                Spacer()
                Button(L.text("mobile.live.joinNow")) {
                    Task { await joinMeeting(grouped.live[0]) }
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.brandCoral)
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(L.text("mobile.live.banner.accessibility"))
    }

    @ViewBuilder
    private var meetingSections: some View {
        if !grouped.live.isEmpty {
            sectionHeader(L.text("mobile.live.section.liveNow"), color: LexturesTheme.brandCoral)
            ForEach(grouped.live) { meeting in
                meetingCard(meeting)
            }
        }
        if !grouped.upcoming.isEmpty {
            sectionHeader(L.text("mobile.live.section.upcoming"), color: LexturesTheme.brandTeal)
            ForEach(grouped.upcoming) { meeting in
                meetingCard(meeting)
            }
        }
        if !grouped.past.isEmpty {
            sectionHeader(L.text("mobile.live.section.past"), color: LexturesTheme.textSecondary(for: colorScheme))
            ForEach(grouped.past) { meeting in
                meetingCard(meeting)
            }
        }
    }

    private func sectionHeader(_ title: String, color: Color) -> some View {
        Text(title.uppercased())
            .font(.caption.weight(.semibold))
            .foregroundStyle(color)
            .padding(.top, 4)
    }

    private func meetingCard(_ meeting: VirtualMeeting) -> some View {
        let isActive = meeting.status == "live"
        let isSoon = LiveMeetingsLogic.isLiveOrSoon(meeting)
        return LMSCard(accent: isActive ? LexturesTheme.brandCoral : nil) {
            VStack(alignment: .leading, spacing: 8) {
                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(meeting.title)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        MeetingStatusChip(status: meeting.status)
                        Text(LiveMeetingsLogic.formatMeetingTime(meeting))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        Text(LiveMeetingsLogic.providerLabel(meeting.provider))
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        if let countdown = meeting.scheduledStart.flatMap({ LiveMeetingsLogic.countdownText(scheduledStart: $0) }) {
                            Text(countdown)
                                .font(.caption2.weight(.medium))
                                .foregroundStyle(LexturesTheme.brandCoral)
                                .accessibilityLabel(countdown)
                        }
                    }
                    Spacer(minLength: 8)
                }
                HStack(spacing: 8) {
                    if LiveMeetingsLogic.canJoin(meeting) {
                        Button(isActive ? L.text("mobile.live.joinNow") : L.text("mobile.live.join")) {
                            Task { await joinMeeting(meeting) }
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(isActive || isSoon ? LexturesTheme.brandCoral : LexturesTheme.brandTeal)
                    }
                    Button {
                        selectedMeeting = meeting
                    } label: {
                        Text(L.text("mobile.live.details"))
                    }
                    .buttonStyle(.bordered)
                    if let icalURL = LiveMeetingsLogic.meetingIcalURL(meetingId: meeting.id) {
                        Button {
                            openURL(icalURL)
                        } label: {
                            Image(systemName: "calendar.badge.plus")
                        }
                        .accessibilityLabel(L.text("mobile.live.addToCalendar"))
                    }
                }
                .font(.caption.weight(.medium))
            }
        }
    }

    private func load(force: Bool = false) async {
        guard let token = session.accessToken else { return }
        if !force && !meetings.isEmpty { return }
        loading = meetings.isEmpty
        errorMessage = nil
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.liveMeetings(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourseMeetings(courseCode: course.courseCode, accessToken: token)
            }
            meetings = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
            schedulePolling()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.live.error.load")
        }
    }

    private func schedulePolling() {
        pollTask?.cancel()
        guard grouped.upcoming.contains(where: { LiveMeetingsLogic.isLiveOrSoon($0) }) || !grouped.live.isEmpty else {
            return
        }
        pollTask = Task {
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(30))
                guard !Task.isCancelled else { return }
                await load(force: true)
            }
        }
    }

    private func joinMeeting(_ meeting: VirtualMeeting) async {
        guard let token = session.accessToken else { return }
        do {
            let info = try await LMSAPI.fetchMeetingJoinInfo(meetingId: meeting.id, accessToken: token)
            let urlString = (course.viewerIsStaff ? info?.hostUrl : nil) ?? info?.joinUrl
            guard let urlString, let url = URL(string: urlString) else {
                errorMessage = L.text("mobile.live.error.join")
                return
            }
            await UIApplication.shared.open(url)
            await load(force: true)
        } catch {
            errorMessage = L.text("mobile.live.error.join")
        }
    }

    private func replaceMeeting(_ updated: VirtualMeeting) {
        guard let index = meetings.firstIndex(where: { $0.id == updated.id }) else { return }
        meetings[index] = updated
    }
}

struct MeetingStatusChip: View {
    let status: String

    var body: some View {
        Text(LiveMeetingsLogic.statusLabel(status))
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(background)
            .foregroundStyle(foreground)
            .clipShape(Capsule())
            .accessibilityLabel(LiveMeetingsLogic.statusLabel(status))
    }

    private var background: Color {
        switch status {
        case "live": return Color.green.opacity(0.15)
        case "scheduled": return Color.blue.opacity(0.15)
        case "ended": return Color.gray.opacity(0.15)
        case "cancelled": return Color.red.opacity(0.15)
        default: return Color.gray.opacity(0.15)
        }
    }

    private var foreground: Color {
        switch status {
        case "live": return .green
        case "scheduled": return .blue
        case "ended", "cancelled": return .secondary
        default: return .secondary
        }
    }
}