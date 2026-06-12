import SwiftUI

/// "Attendance" section of course detail: session list. Students see their own
/// status and can self-report into open self-report sessions; staff see records.
struct CourseAttendanceSection: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    @State private var sessions: [AttendanceSession] = []
    @State private var errorMessage: String?
    @State private var loading = true

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if let errorMessage {
                LMSErrorBanner(message: errorMessage)
            }

            if loading && sessions.isEmpty {
                LMSSkeletonList(count: 3)
            } else if sessions.isEmpty {
                LMSEmptyState(
                    systemImage: "person.crop.circle.badge.checkmark",
                    title: "No attendance sessions",
                    message: "Attendance sessions will appear here when your instructor opens one."
                )
            } else {
                ForEach(sessions) { attendanceSession in
                    NavigationLink(value: attendanceSession) {
                        sessionCard(attendanceSession)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .task { await load() }
    }

    private func sessionCard(_ attendanceSession: AttendanceSession) -> some View {
        LMSCard(accent: attendanceSession.isOpen ? LexturesTheme.brandTeal : nil) {
            HStack(spacing: 12) {
                Image(systemName: attendanceSession.isSelfReport ? "hand.raised.fill" : "list.clipboard.fill")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 32, height: 32)
                    .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.16 : 0.13))
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))

                VStack(alignment: .leading, spacing: 3) {
                    Text(attendanceSession.displayTitle)
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    HStack(spacing: 6) {
                        if let date = attendanceSession.sessionDate, !date.isEmpty {
                            Text(formattedSessionDate(date))
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        Text(attendanceSession.isSelfReport ? "Self report" : "Roll call")
                            .font(.caption2.weight(.medium))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }

                Spacer(minLength: 0)

                Text(attendanceSession.isOpen ? "Open" : "Closed")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(attendanceSession.isOpen ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.textSecondary(for: colorScheme))
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(
                        (attendanceSession.isOpen ? LexturesTheme.brandTeal : LexturesTheme.fieldBorder(for: colorScheme))
                            .opacity(attendanceSession.isOpen ? 0.16 : 0.4)
                    )
                    .clipShape(Capsule())
                Image(systemName: "chevron.right")
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
            }
        }
    }

    private func formattedSessionDate(_ raw: String) -> String {
        if let parsed = LMSDates.parse(raw) {
            return parsed.formatted(date: .abbreviated, time: .omitted)
        }
        return raw
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
            sessions = try await LMSAPI.fetchAttendanceSessions(courseCode: course.courseCode, accessToken: token)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load attendance."
        }
    }
}

/// One attendance session: my status + self-report (students) or records (staff).
struct AttendanceSessionDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary
    let attendanceSession: AttendanceSession

    @State private var detail: AttendanceSessionDetail?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var reporting = false

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && detail == nil {
                        LMSSkeletonList(count: 2)
                    } else if let detail {
                        myStatusCard(detail)
                        if detail.canSelfReport == true && attendanceSession.isOpen {
                            selfReportCard
                        }
                        if let records = detail.records, !records.isEmpty {
                            recordsCard(records)
                        }
                    }
                }
                .padding(16)
            }
            .refreshable { await load() }
        }
        .navigationTitle(attendanceSession.displayTitle)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    @ViewBuilder
    private func myStatusCard(_ detail: AttendanceSessionDetail) -> some View {
        if let record = detail.myRecord {
            LMSCard {
                HStack(spacing: 12) {
                    Image(systemName: statusIcon(record.status))
                        .font(.title3)
                        .foregroundStyle(statusTint(record.status))
                        .frame(width: 44, height: 44)
                        .background(statusTint(record.status).opacity(0.13))
                        .clipShape(Circle())
                    VStack(alignment: .leading, spacing: 2) {
                        Text(AttendanceStatusInfo.label(record.status))
                            .font(LexturesTheme.displayFont(18))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        if let recordedAt = LMSDates.parse(record.recordedAt) {
                            Text("Recorded \(recordedAt.formatted(date: .abbreviated, time: .shortened))")
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }
                    Spacer(minLength: 0)
                }
            }
        }
    }

    private var selfReportCard: some View {
        LMSCard {
            Text("Report your attendance")
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text("This session is open for self-reporting.")
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            HStack(spacing: 10) {
                reportButton("I'm here", status: "present", tint: LexturesTheme.primary)
                reportButton("I'm late", status: "tardy", tint: LexturesTheme.amber)
            }
        }
    }

    private func reportButton(_ title: String, status: String, tint: Color) -> some View {
        Button {
            Task { await selfReport(status) }
        } label: {
            if reporting {
                ProgressView()
                    .controlSize(.small)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 11)
            } else {
                Text(title)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.white)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 11)
            }
        }
        .background(tint)
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        .buttonStyle(.plain)
        .disabled(reporting)
    }

    private func recordsCard(_ records: [AttendanceRecord]) -> some View {
        LMSCard {
            Text("Roster")
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            ForEach(Array(records.enumerated()), id: \.offset) { index, record in
                if index > 0 {
                    Divider()
                }
                HStack {
                    Text(record.displayName ?? "Student")
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Spacer()
                    Text(AttendanceStatusInfo.label(record.status))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(statusTint(record.status))
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .background(statusTint(record.status).opacity(0.12))
                        .clipShape(Capsule())
                }
                .padding(.vertical, 2)
            }
        }
    }

    private func statusIcon(_ status: String) -> String {
        switch status {
        case "present": return "checkmark.circle.fill"
        case "absent": return "xmark.circle.fill"
        case "tardy": return "clock.fill"
        case "excused": return "checkmark.seal.fill"
        default: return "questionmark.circle"
        }
    }

    private func statusTint(_ status: String) -> Color {
        switch status {
        case "present": return LexturesTheme.primary
        case "absent": return LexturesTheme.error
        case "tardy": return LexturesTheme.amber
        case "excused": return LexturesTheme.brandTeal
        default: return LexturesTheme.textSecondaryDark
        }
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
            detail = try await LMSAPI.fetchAttendanceSessionDetail(
                courseCode: course.courseCode,
                sessionId: attendanceSession.id,
                accessToken: token
            )
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load this session."
        }
    }

    private func selfReport(_ status: String) async {
        guard let token = session.accessToken else { return }
        reporting = true
        errorMessage = nil
        defer { reporting = false }
        do {
            try await LMSAPI.selfReportAttendance(
                courseCode: course.courseCode,
                sessionId: attendanceSession.id,
                status: status,
                accessToken: token
            )
            await load()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not report attendance."
        }
    }
}
