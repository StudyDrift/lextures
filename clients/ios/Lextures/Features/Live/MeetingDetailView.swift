import SwiftUI

struct MeetingDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL
    @Environment(\.dismiss) private var dismiss
    let course: CourseSummary
    let meeting: VirtualMeeting
    let onUpdated: (VirtualMeeting) -> Void
    let onOpenWhiteboard: (CourseWhiteboard) -> Void

    @State private var currentMeeting: VirtualMeeting
    @State private var attendance: [MeetingAttendanceRecord] = []
    @State private var whiteboards: [CourseWhiteboard] = []
    @State private var loadingAttendance = false
    @State private var loadingWhiteboards = false
    @State private var updatingStatus = false
    @State private var creatingWhiteboard = false
    @State private var errorMessage: String?

    private var canEditWhiteboard: Bool {
        WhiteboardLogic.canEdit(
            viewerIsStaff: course.viewerIsStaff,
            features: shell.platformFeatures
        )
    }

    init(
        course: CourseSummary,
        meeting: VirtualMeeting,
        onUpdated: @escaping (VirtualMeeting) -> Void,
        onOpenWhiteboard: @escaping (CourseWhiteboard) -> Void
    ) {
        self.course = course
        self.meeting = meeting
        self.onUpdated = onUpdated
        self.onOpenWhiteboard = onOpenWhiteboard
        _currentMeeting = State(initialValue: meeting)
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }
                    header
                    actions
                    if course.viewerIsStaff {
                        staffControls
                        attendanceSection
                        whiteboardSection
                    }
                }
                .padding(16)
            }
            .background(LexturesTheme.sceneBackground(for: colorScheme))
            .navigationTitle(L.text("mobile.live.detail.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.live.close")) { dismiss() }
                }
            }
            .task {
                if course.viewerIsStaff {
                    await loadStaffData()
                }
            }
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(currentMeeting.title)
                .font(.title3.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            MeetingStatusChip(status: currentMeeting.status)
            Text(LiveMeetingsLogic.formatMeetingTime(currentMeeting))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Text(LiveMeetingsLogic.providerLabel(currentMeeting.provider))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private var actions: some View {
        VStack(alignment: .leading, spacing: 10) {
            if LiveMeetingsLogic.canJoin(currentMeeting) {
                Button {
                    Task { await joinMeeting() }
                } label: {
                    Label(L.text("mobile.live.joinNow"), systemImage: "video.fill")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.brandCoral)
            }
            if let icalURL = LiveMeetingsLogic.meetingIcalURL(meetingId: currentMeeting.id) {
                Button {
                    openURL(icalURL)
                } label: {
                    Label(L.text("mobile.live.addToCalendar"), systemImage: "calendar.badge.plus")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered)
            }
        }
    }

    @ViewBuilder
    private var staffControls: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(L.text("mobile.live.staffControls"))
                .font(.subheadline.weight(.semibold))
            HStack(spacing: 8) {
                if currentMeeting.status == "scheduled" {
                    Button(L.text("mobile.live.startSession")) {
                        Task { await updateStatus("live") }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(updatingStatus)
                }
                if currentMeeting.status == "live" {
                    Button(L.text("mobile.live.endSession")) {
                        Task { await updateStatus("ended") }
                    }
                    .buttonStyle(.bordered)
                    .disabled(updatingStatus)
                }
                if currentMeeting.status != "cancelled" && currentMeeting.status != "ended" {
                    Button(L.text("mobile.live.cancelSession"), role: .destructive) {
                        Task { await updateStatus("cancelled") }
                    }
                    .disabled(updatingStatus)
                }
            }
            .font(.caption.weight(.medium))
        }
    }

    @ViewBuilder
    private var attendanceSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.live.attendance.title"))
                .font(.subheadline.weight(.semibold))
            if loadingAttendance {
                ProgressView()
            } else if attendance.isEmpty {
                Text(L.text("mobile.live.attendance.empty"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                Text(L.format("mobile.live.attendance.count", "\(attendance.count)"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .accessibilityLabel(L.format("mobile.live.attendance.count", "\(attendance.count)"))
            }
        }
    }

    @ViewBuilder
    private var whiteboardSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(L.text("mobile.live.whiteboard.title"))
                    .font(.subheadline.weight(.semibold))
                Spacer()
                if canEditWhiteboard {
                    Button {
                        Task { await createWhiteboard() }
                    } label: {
                        if creatingWhiteboard {
                            ProgressView()
                        } else {
                            Label(L.text("mobile.whiteboard.create"), systemImage: "plus")
                        }
                    }
                    .disabled(creatingWhiteboard)
                    .accessibilityLabel(L.text("mobile.whiteboard.create"))
                }
            }
            if loadingWhiteboards {
                ProgressView()
            } else if whiteboards.isEmpty {
                Text(L.text("mobile.live.whiteboard.empty"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(whiteboards) { board in
                    Button {
                        onOpenWhiteboard(board)
                    } label: {
                        HStack {
                            Text(board.title)
                            Spacer()
                            Image(systemName: "chevron.right")
                        }
                    }
                    .buttonStyle(.bordered)
                }
            }
            if !canEditWhiteboard {
                Text(L.text("mobile.live.whiteboard.webHint"))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func createWhiteboard() async {
        guard canEditWhiteboard, let token = session.accessToken else { return }
        creatingWhiteboard = true
        defer { creatingWhiteboard = false }
        do {
            let created = try await LMSAPI.createCourseWhiteboard(
                courseCode: course.courseCode,
                title: WhiteboardLogic.defaultTitle(existingCount: whiteboards.count),
                canvasData: [],
                accessToken: token
            )
            whiteboards.insert(created, at: 0)
            onOpenWhiteboard(created)
        } catch {
            errorMessage = L.text("mobile.whiteboard.error.create")
        }
    }

    private func loadStaffData() async {
        guard let token = session.accessToken else { return }
        loadingAttendance = true
        loadingWhiteboards = true
        defer {
            loadingAttendance = false
            loadingWhiteboards = false
        }
        async let attendanceTask = LMSAPI.fetchMeetingAttendance(meetingId: currentMeeting.id, accessToken: token)
        async let boardsTask = LMSAPI.fetchCourseWhiteboards(courseCode: course.courseCode, accessToken: token)
        attendance = (try? await attendanceTask) ?? []
        whiteboards = (try? await boardsTask) ?? []
    }

    private func joinMeeting() async {
        guard let token = session.accessToken else { return }
        do {
            let info = try await LMSAPI.fetchMeetingJoinInfo(meetingId: currentMeeting.id, accessToken: token)
            let urlString = (course.viewerIsStaff ? info?.hostUrl : nil) ?? info?.joinUrl
            guard let urlString, let url = URL(string: urlString) else {
                errorMessage = L.text("mobile.live.error.join")
                return
            }
            await UIApplication.shared.open(url)
            if course.viewerIsStaff {
                await loadStaffData()
            }
        } catch {
            errorMessage = L.text("mobile.live.error.join")
        }
    }

    private func updateStatus(_ status: String) async {
        guard let token = session.accessToken else { return }
        updatingStatus = true
        errorMessage = nil
        defer { updatingStatus = false }
        do {
            let updated = try await LMSAPI.patchMeeting(
                meetingId: currentMeeting.id,
                status: status,
                accessToken: token
            )
            currentMeeting = updated
            onUpdated(updated)
            if status == "live" || status == "ended" {
                await loadStaffData()
            }
            if status == "cancelled" {
                dismiss()
            }
        } catch {
            errorMessage = L.text("mobile.live.error.update")
        }
    }
}