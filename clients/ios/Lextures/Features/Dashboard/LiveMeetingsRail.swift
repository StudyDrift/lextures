import SwiftUI

/// Home/Calendar "Live & upcoming" rail (M7.5).
struct LiveMeetingsRail: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL
    let items: [LiveMeetingsLogic.LiveUpcomingItem]
    let courses: [CourseSummary]

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(L.text("mobile.live.rail.title"))
                .font(.headline)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 12) {
                    ForEach(items) { item in
                        railCard(item)
                    }
                }
            }
        }
    }

    private func railCard(_ item: LiveMeetingsLogic.LiveUpcomingItem) -> some View {
        LMSCard(accent: item.meeting.status == "live" ? LexturesTheme.brandCoral : nil) {
            VStack(alignment: .leading, spacing: 8) {
                if item.meeting.status == "live" {
                    Text(L.text("mobile.live.status.live"))
                        .font(.caption2.weight(.bold))
                        .foregroundStyle(LexturesTheme.brandCoral)
                }
                Text(item.meeting.title)
                    .font(.subheadline.weight(.semibold))
                    .lineLimit(2)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(item.courseTitle)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(LiveMeetingsLogic.formatMeetingTime(item.meeting))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                HStack(spacing: 8) {
                    Button(L.text("mobile.live.join")) {
                        Task { await join(item) }
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(LexturesTheme.brandCoral)
                    Button(L.text("mobile.live.openCourse")) {
                        openCourse(item)
                    }
                    .buttonStyle(.bordered)
                }
                .font(.caption.weight(.medium))
            }
            .frame(width: 240, alignment: .leading)
        }
    }

    private func openCourse(_ item: LiveMeetingsLogic.LiveUpcomingItem) {
        guard let course = courses.first(where: { $0.courseCode == item.courseCode }) else { return }
        shell.activeCourse = course
        shell.activeCourseRoot = .courses
        shell.activeCourseSection = .live
        shell.select(.courses)
    }

    private func join(_ item: LiveMeetingsLogic.LiveUpcomingItem) async {
        guard let token = session.accessToken else { return }
        let info = try? await LMSAPI.fetchMeetingJoinInfo(meetingId: item.meeting.id, accessToken: token)
        guard let urlString = info?.joinUrl, let url = URL(string: urlString) else { return }
        await UIApplication.shared.open(url)
    }
}