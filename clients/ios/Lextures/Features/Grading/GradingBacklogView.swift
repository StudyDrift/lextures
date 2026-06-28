import SwiftUI

/// "Grading" section of course detail (staff): assignments with ungraded work.
struct GradingBacklogSection: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    @State private var items: [GradingBacklogItem] = []
    @State private var errorMessage: String?
    @State private var loading = true

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if let errorMessage {
                LMSErrorBanner(message: errorMessage)
            }

            if loading && items.isEmpty {
                LMSSkeletonList(count: 3)
            } else if items.isEmpty {
                LMSEmptyState(
                    systemImage: "checkmark.rectangle.stack",
                    title: "All caught up",
                    message: "Submissions waiting for a grade will appear here."
                )
            } else {
                ForEach(items) { item in
                    NavigationLink(value: item) {
                        backlogCard(item)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .task { await load() }
    }

    private func backlogCard(_ item: GradingBacklogItem) -> some View {
        LMSCard(accent: item.ungradedCount > 0 ? LexturesTheme.amber : nil) {
            HStack(spacing: 12) {
                Image(systemName: "doc.text.fill")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(LexturesTheme.amber)
                    .frame(width: 32, height: 32)
                    .background(LexturesTheme.amber.opacity(0.13))
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))

                VStack(alignment: .leading, spacing: 3) {
                    HStack(spacing: 6) {
                        Text(item.assignmentTitle)
                            .font(.subheadline.weight(.medium))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        if item.isQuiz {
                            Text("Quiz")
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(LexturesTheme.amber)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(LexturesTheme.amber.opacity(0.14))
                                .clipShape(Capsule())
                        }
                    }
                    Text("\(item.ungradedCount) ungraded submission\(item.ungradedCount == 1 ? "" : "s")")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                Spacer(minLength: 0)

                Text("\(item.ungradedCount)")
                    .font(LexturesTheme.displayFont(16, weight: .bold))
                    .foregroundStyle(LexturesTheme.amber)
                    .padding(.horizontal, 9)
                    .padding(.vertical, 3)
                    .background(LexturesTheme.amber.opacity(0.14))
                    .clipShape(Capsule())
                Image(systemName: "chevron.right")
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
            }
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
            items = try await LMSAPI.fetchGradingBacklog(courseCode: course.courseCode, accessToken: token)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load the grading backlog."
        }
    }
}

/// Standalone backlog screen (pushed from the dashboard teacher snapshot).
struct GradingBacklogView: View {
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                GradingBacklogSection(course: course)
                    .padding(16)
            }
        }
        .navigationTitle("Grading · \(course.displayTitle)")
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(for: GradingBacklogItem.self) { item in
            SubmissionsListView(course: course, backlogItem: item)
        }
    }
}

/// Submissions for one assignment, filterable by graded state.
struct SubmissionsListView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary
    let backlogItem: GradingBacklogItem

    private enum Filter: String, CaseIterable {
        case ungraded = "Ungraded"
        case graded = "Graded"
        case all = "All"

        var queryValue: String? {
            switch self {
            case .ungraded: return "ungraded"
            case .graded: return "graded"
            case .all: return nil
            }
        }
    }

    @State private var filter: Filter = .ungraded
    @State private var submissions: [AssignmentSubmission] = []
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var grading: AssignmentSubmission?

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    LMSSegmentedChips(options: Filter.allCases, selection: $filter, label: \.rawValue)

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && submissions.isEmpty {
                        LMSSkeletonList(count: 4)
                    } else if submissions.isEmpty {
                        LMSEmptyState(
                            systemImage: "tray",
                            title: "No submissions",
                            message: filter == .ungraded
                                ? "Nothing waiting for a grade right now."
                                : "No submissions in this view."
                        )
                    } else {
                        ForEach(submissions) { submission in
                            Button {
                                grading = submission
                            } label: {
                                submissionCard(submission)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
                .padding(16)
            }
            .refreshable { await load() }
        }
        .navigationTitle(backlogItem.assignmentTitle)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .task(id: filter) { await load() }
        .sheet(item: $grading) { submission in
            if backlogItem.isQuiz {
                QuizAttemptInfoSheet(
                    backlogItem: backlogItem,
                    submission: submission
                )
            } else {
                SpeedGraderView(
                    course: course,
                    assignmentId: backlogItem.resolvedItemId,
                    submissions: submissions,
                    startIndex: submissions.firstIndex(of: submission) ?? 0
                ) {
                    Task { await load() }
                }
            }
        }
    }

    private func submissionCard(_ submission: AssignmentSubmission) -> some View {
        LMSCard {
            HStack(spacing: 12) {
                Circle()
                    .fill(LexturesTheme.coverGradient(for: submission.displayName))
                    .frame(width: 38, height: 38)
                    .overlay(
                        Text(String(submission.displayName.prefix(2)).uppercased())
                            .font(.caption.weight(.bold))
                            .foregroundStyle(.white)
                    )

                VStack(alignment: .leading, spacing: 3) {
                    Text(submission.displayName)
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    HStack(spacing: 6) {
                        Text("Submitted \(LMSDates.relative(submission.submittedAt))")
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        if let version = submission.versionNumber, version > 1 {
                            Text("v\(version)")
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                        }
                    }
                    if let filename = submission.attachmentFilename, !filename.isEmpty {
                        Label(filename, systemImage: "paperclip")
                            .font(.caption2)
                            .lineLimit(1)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }

                Spacer(minLength: 0)
                Image(systemName: "square.and.pencil")
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            }
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
            submissions = try await LMSAPI.fetchGradingSubmissions(
                courseCode: course.courseCode,
                backlogItem: backlogItem,
                graded: filter.queryValue,
                accessToken: token
            )
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load submissions."
        }
    }
}

/// Quiz attempts must be graded per question on web (mobile v1 is read-only).
struct QuizAttemptInfoSheet: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let backlogItem: GradingBacklogItem
    let submission: AssignmentSubmission

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        LMSCard {
                            Text(submission.displayName)
                                .font(LexturesTheme.displayFont(18))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Text("Submitted \(LMSDates.shortDateTime(submission.submittedAt))")
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            if let version = submission.versionNumber, version > 1 {
                                Text("Attempt \(version)")
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }

                        LMSCard {
                            Text("Grade on web")
                                .font(LexturesTheme.displayFont(17))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Text("Quiz answers are graded question by question. Open the web app to review responses and enter scores for \(backlogItem.assignmentTitle).")
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }
                    .padding(16)
                }
            }
            .navigationTitle("Quiz attempt")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Done") { dismiss() }
                }
            }
        }
        .presentationDetents([.medium, .large])
    }
}
