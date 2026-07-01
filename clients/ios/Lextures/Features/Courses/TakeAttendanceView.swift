import SwiftUI

/// Staff roll-call: start or resume a session, mark the roster, save.
struct TakeAttendanceView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary
    var initialSessionId: String?

    @State private var activeSessionId: String?
    @State private var sessionDetail: AttendanceSessionDetail?
    @State private var draft: [String: String] = [:]
    @State private var sections: [CourseSection] = []
    @State private var selectedSectionId: String = ""
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var creating = false
    @State private var saving = false

    private var records: [AttendanceRecord] { sessionDetail?.records ?? [] }
    private var isOpen: Bool { sessionDetail?.status == "open" }
    private var counts: AttendanceSummaryCounts {
        TakeAttendanceLogic.summaryCounts(records: records, draft: draft)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && sessionDetail == nil {
                        LMSSkeletonList(count: 4)
                    } else if activeSessionId == nil {
                        startSessionCard
                    } else if let sessionDetail {
                        summaryCard
                        if isOpen {
                            actionBar
                        }
                        if records.isEmpty {
                            LMSEmptyState(
                                systemImage: "person.3",
                                title: L.text("mobile.attendance.take.noRoster"),
                                message: ""
                            )
                        } else {
                            rosterCard(sessionDetail)
                        }
                    }
                }
                .padding(16)
            }
            .refreshable { await reload() }
        }
        .navigationTitle(L.text("mobile.attendance.take.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await bootstrap() }
    }

    private var startSessionCard: some View {
        LMSCard {
            Text(L.text("mobile.attendance.take.newSessionHint"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            if course.isSectionsEnabled && !sections.isEmpty {
                VStack(alignment: .leading, spacing: 6) {
                    Text(L.text("mobile.attendance.take.section"))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Picker(L.text("mobile.attendance.take.section"), selection: $selectedSectionId) {
                        ForEach(sections) { section in
                            Text(section.displayName).tag(section.id)
                        }
                    }
                    .pickerStyle(.menu)
                }
            }

            Button {
                Task { await startSession() }
            } label: {
                if creating {
                    ProgressView()
                        .controlSize(.small)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 11)
                } else {
                    Text(L.text("mobile.attendance.take.start"))
                        .font(.subheadline.weight(.semibold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 11)
                }
            }
            .background(LexturesTheme.accent(for: colorScheme))
            .foregroundStyle(colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .buttonStyle(.plain)
            .disabled(creating)
        }
    }

    private var summaryCard: some View {
        LMSCard {
            Text(
                L.format(
                    "mobile.attendance.take.summary",
                    counts.present,
                    counts.absent,
                    counts.tardy,
                    counts.excused
                )
            )
            .font(LexturesTheme.displayFont(16))
            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            .accessibilityLabel(summaryAccessibilityLabel)
        }
    }

    private var summaryAccessibilityLabel: String {
        "\(counts.present) present, \(counts.absent) absent, \(counts.tardy) tardy, \(counts.excused) excused"
    }

    private var actionBar: some View {
        HStack(spacing: 8) {
            Button(L.text("mobile.attendance.take.markAllPresent")) {
                draft = TakeAttendanceLogic.markAllPresent(records: records)
            }
            .font(.caption.weight(.semibold))
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(LexturesTheme.primary.opacity(0.12))
            .foregroundStyle(LexturesTheme.primary)
            .clipShape(Capsule())
            .buttonStyle(.plain)

            Button {
                Task { await saveMarks() }
            } label: {
                if saving {
                    ProgressView().controlSize(.small)
                } else {
                    Text(L.text("mobile.attendance.take.save"))
                }
            }
            .font(.caption.weight(.semibold))
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(LexturesTheme.accent(for: colorScheme))
            .foregroundStyle(colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
            .clipShape(Capsule())
            .buttonStyle(.plain)
            .disabled(saving || records.isEmpty)

            Button(L.text("mobile.attendance.take.close")) {
                Task { await closeSession() }
            }
            .font(.caption.weight(.semibold))
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .overlay(Capsule().stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1))
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            .buttonStyle(.plain)
            .disabled(saving)
        }
    }

    private func rosterCard(_ detail: AttendanceSessionDetail) -> some View {
        LMSCard {
            Text(detail.title ?? L.text("mobile.attendance.take.title"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            ForEach(Array(records.enumerated()), id: \.offset) { index, record in
                if index > 0 { Divider() }
                studentRow(record, editable: isOpen)
            }
        }
    }

    private func studentRow(_ record: AttendanceRecord, editable: Bool) -> some View {
        let currentStatus = draft[record.studentUserId] ?? record.status
        let markStatus = AttendanceMarkStatus(rawValue: currentStatus) ?? .present

        return VStack(alignment: .leading, spacing: 8) {
            Text(TakeAttendanceLogic.studentLabel(record))
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            if editable {
                LMSSegmentedChips(
                    options: AttendanceMarkStatus.allCases,
                    selection: Binding(
                        get: { markStatus },
                        set: { draft[record.studentUserId] = $0.rawValue }
                    ),
                    label: { TakeAttendanceLogic.statusLabel($0.rawValue) }
                )
                .accessibilityLabel(
                    L.format("mobile.attendance.take.a11yStatusFor", TakeAttendanceLogic.studentLabel(record))
                )
            } else {
                Text(TakeAttendanceLogic.statusLabel(currentStatus))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(statusTint(currentStatus))
            }
        }
        .padding(.vertical, 4)
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

    private func bootstrap() async {
        guard course.viewerIsStaff, let token = session.accessToken else {
            loading = false
            return
        }
        loading = true
        errorMessage = nil
        defer { loading = false }

        if course.isSectionsEnabled {
            sections = (try? await LMSAPI.fetchCourseSections(courseCode: course.courseCode, accessToken: token)) ?? []
            if let first = sections.first {
                selectedSectionId = first.id
            }
        }

        if let initialSessionId {
            activeSessionId = initialSessionId
            await loadSession(initialSessionId)
            return
        }

        do {
            let sessions = try await LMSAPI.fetchAttendanceSessions(courseCode: course.courseCode, accessToken: token)
            if let today = TakeAttendanceLogic.findTodaysOpenRollCallSession(sessions: sessions) {
                activeSessionId = today.id
                await loadSession(today.id)
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.attendance.take.loadError")
        }
    }

    private func reload() async {
        guard let sessionId = activeSessionId else {
            await bootstrap()
            return
        }
        await loadSession(sessionId)
    }

    private func loadSession(_ sessionId: String) async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let detail = try await LMSAPI.fetchAttendanceSessionDetail(
                courseCode: course.courseCode,
                sessionId: sessionId,
                accessToken: token
            )
            sessionDetail = detail
            if let records = detail.records {
                draft = TakeAttendanceLogic.buildDraft(from: records)
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.attendance.take.loadError")
        }
    }

    private func startSession() async {
        guard let token = session.accessToken else { return }
        creating = true
        errorMessage = nil
        defer { creating = false }
        let today = TakeAttendanceLogic.todayDateString()
        do {
            let created = try await LMSAPI.createAttendanceSession(
                courseCode: course.courseCode,
                body: CreateAttendanceSessionBody(
                    collectionMethod: "roll_call",
                    title: "Roll call — \(today)",
                    sessionDate: today,
                    sectionId: selectedSectionId.isEmpty ? nil : selectedSectionId
                ),
                accessToken: token
            )
            activeSessionId = created.id
            await loadSession(created.id)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.attendance.take.createError")
        }
    }

    private func saveMarks() async {
        guard let token = session.accessToken, let sessionId = activeSessionId else { return }
        saving = true
        errorMessage = nil
        defer { saving = false }
        let payload = TakeAttendanceLogic.recordsPayload(records: records, draft: draft)
        let body = SaveAttendanceRecordsBody(records: payload)
        do {
            _ = try await offline.enqueueMutation(
                method: "PUT",
                path: "/api/v1/courses/\(LMSAPI.encodePath(course.courseCode))/attendance/sessions/\(LMSAPI.encodePath(sessionId))/records",
                body: body,
                label: L.text("mobile.attendance.take.save"),
                accessToken: token
            )
            await loadSession(sessionId)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.attendance.take.saveError")
        }
    }

    private func closeSession() async {
        guard let token = session.accessToken, let sessionId = activeSessionId else { return }
        saving = true
        errorMessage = nil
        defer { saving = false }
        do {
            _ = try await offline.enqueueMutation(
                method: "POST",
                path: "/api/v1/courses/\(LMSAPI.encodePath(course.courseCode))/attendance/sessions/\(LMSAPI.encodePath(sessionId))/close",
                body: TakeAttendanceCloseBody(),
                label: L.text("mobile.attendance.take.close"),
                accessToken: token
            )
            await loadSession(sessionId)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.attendance.take.closeError")
        }
    }
}

private struct TakeAttendanceCloseBody: Encodable {
    var finalizeMissingAsAbsent = true
}