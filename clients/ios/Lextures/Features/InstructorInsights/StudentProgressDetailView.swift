import SwiftUI

/// Read-only per-student progress with message action (M11.3).
struct StudentProgressDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary
    let enrollmentId: String
    let displayName: String

    @State private var progress: StudentProgressResponse?
    @State private var activity: [StudentProgressActivityEvent] = []
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var composeMode = false
    @State private var messageSubject = ""
    @State private var messageBody = ""
    @State private var messageBusy = false
    @State private var messageSuccess: String?

    var body: some View {
        ScrollView {
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
                if let messageSuccess {
                    Label(messageSuccess, systemImage: "checkmark.circle.fill")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.brandTeal)
                }

                if loading && progress == nil {
                    LMSSkeletonList(count: 4)
                } else if let progress {
                    summaryCard(progress.summary)
                    if !progress.missing.isEmpty {
                        missingSection(progress.missing)
                    }
                    if !progress.assignments.isEmpty {
                        assignmentsSection(progress.assignments)
                    }
                    if !activity.isEmpty {
                        activitySection
                    }
                    messageButton(reason: progress.summary.missingCount > 0
                        ? L.text("mobile.instructorInsights.progress.missingWork")
                        : L.text("mobile.instructorInsights.progress.checkIn"))
                }
            }
            .padding(16)
        }
        .navigationTitle(displayName)
        .navigationBarTitleDisplayMode(.inline)
        .sheet(isPresented: $composeMode) {
            messageSheet
        }
        .task { await load() }
    }

    private func summaryCard(_ summary: StudentProgressSummary) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.instructorInsights.progress.summary"))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                metricRow(
                    L.text("mobile.instructorInsights.progress.assignments"),
                    "\(Int(summary.assignmentsSubmittedPct.rounded()))%"
                )
                metricRow(
                    L.text("mobile.instructorInsights.progress.modules"),
                    "\(Int(summary.modulesViewedPct.rounded()))%"
                )
                if let grade = InstructorInsightsLogic.optionalPercentText(summary.avgGradePercent) {
                    metricRow(L.text("mobile.instructorInsights.progress.avgGrade"), grade)
                }
                if summary.missingCount > 0 {
                    Text(L.plural("mobile.instructorInsights.progress.missingCount", count: summary.missingCount))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.amber)
                }
            }
        }
    }

    private func metricRow(_ label: String, _ value: String) -> some View {
        HStack {
            Text(label)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Spacer(minLength: 0)
            Text(value)
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
    }

    private func missingSection(_ items: [StudentProgressMissingItem]) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.instructorInsights.progress.missingTitle"))
                    .font(.subheadline.weight(.semibold))
                ForEach(items.prefix(5)) { item in
                    Text(item.title)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                }
            }
        }
    }

    private func assignmentsSection(_ rows: [StudentProgressAssignmentRow]) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.instructorInsights.progress.assignmentsTitle"))
                    .font(.subheadline.weight(.semibold))
                ForEach(rows.prefix(5)) { row in
                    HStack {
                        Text(row.title)
                            .font(.caption)
                            .lineLimit(1)
                        Spacer(minLength: 0)
                        Text(row.grade)
                            .font(.caption.weight(.semibold))
                    }
                }
            }
        }
    }

    private var activitySection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.instructorInsights.progress.activityTitle"))
                    .font(.subheadline.weight(.semibold))
                ForEach(activity.prefix(8)) { event in
                    VStack(alignment: .leading, spacing: 2) {
                        Text(event.label)
                            .font(.caption.weight(.medium))
                        if let detail = event.detail, !detail.isEmpty {
                            Text(detail)
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }
                }
            }
        }
    }

    private func messageButton(reason: String) -> some View {
        Button {
            messageSubject = InstructorInsightsLogic.messageSubject(displayName: displayName)
            messageBody = InstructorInsightsLogic.messageBody(displayName: displayName, reason: reason)
            composeMode = true
        } label: {
            Text(L.text("mobile.instructorInsights.message.action"))
                .font(.subheadline.weight(.semibold))
                .frame(maxWidth: .infinity)
                .padding(.vertical, 12)
                .background(LexturesTheme.brandTeal)
                .foregroundStyle(.white)
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        }
        .buttonStyle(.plain)
    }

    private var messageSheet: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                VStack(alignment: .leading, spacing: 12) {
                    TextField(L.text("mobile.people.message.subject"), text: $messageSubject)
                        .textFieldStyle(.roundedBorder)
                    TextField(L.text("mobile.people.message.body"), text: $messageBody, axis: .vertical)
                        .lineLimit(4 ... 8)
                        .textFieldStyle(.roundedBorder)
                    Button(messageBusy ? L.text("mobile.people.message.sending") : L.text("mobile.people.message.send")) {
                        Task { await sendMessage() }
                    }
                    .disabled(messageBusy || messageSubject.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || messageBody.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                    Spacer(minLength: 0)
                }
                .padding(16)
            }
            .navigationTitle(L.text("mobile.instructorInsights.message.action"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.people.detail.done")) { composeMode = false }
                }
            }
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.studentProgress(courseCode: course.courseCode, enrollmentId: enrollmentId),
                accessToken: token
            ) {
                try await LMSAPI.fetchStudentProgress(
                    courseCode: course.courseCode,
                    enrollmentId: enrollmentId,
                    accessToken: token
                )
            }
            progress = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
            errorMessage = nil

            if NetworkMonitor.shared.isOnline {
                let activityResult = try await LMSAPI.fetchStudentProgressActivity(
                    courseCode: course.courseCode,
                    enrollmentId: enrollmentId,
                    cursor: nil,
                    accessToken: token
                )
                activity = activityResult.events
            }
        } catch {
            errorMessage = L.text("mobile.instructorInsights.error.progress")
        }
    }

    private func sendMessage() async {
        guard let token = session.accessToken else { return }
        guard NetworkMonitor.shared.isOnline else {
            errorMessage = L.text("mobile.people.message.offline")
            return
        }
        messageBusy = true
        defer { messageBusy = false }
        do {
            _ = try await LMSAPI.sendEnrollmentMessage(
                courseCode: course.courseCode,
                enrollmentId: enrollmentId,
                body: EnrollmentMessageBody(subject: messageSubject, body: messageBody),
                accessToken: token
            )
            messageSuccess = L.text("mobile.people.message.success")
            composeMode = false
        } catch {
            errorMessage = L.text("mobile.people.message.error")
        }
    }
}
