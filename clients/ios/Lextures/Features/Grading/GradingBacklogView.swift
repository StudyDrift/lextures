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
                    Text(item.assignmentTitle)
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
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
            GradeSubmissionSheet(
                course: course,
                assignmentId: backlogItem.assignmentId,
                submission: submission
            ) {
                Task { await load() }
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
            submissions = try await LMSAPI.fetchSubmissions(
                courseCode: course.courseCode,
                itemId: backlogItem.assignmentId,
                graded: filter.queryValue,
                accessToken: token
            )
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load submissions."
        }
    }
}

/// Bottom sheet: enter points + comment for one submission.
struct GradeSubmissionSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let course: CourseSummary
    let assignmentId: String
    let submission: AssignmentSubmission
    var onSaved: () -> Void = {}

    @State private var pointsText = ""
    @State private var comment = ""
    @State private var maxPoints: Double?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var saving = false

    private var pointsValue: Double? {
        Double(pointsText.replacingOccurrences(of: ",", with: "."))
    }

    private var pointsValid: Bool {
        guard let value = pointsValue else { return false }
        if let max = maxPoints { return value >= 0 && value <= max }
        return value >= 0
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }

                        LMSCard {
                            Text(submission.displayName)
                                .font(LexturesTheme.displayFont(18))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Text("Submitted \(LMSDates.shortDateTime(submission.submittedAt))")
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            if let filename = submission.attachmentFilename, !filename.isEmpty {
                                Label(filename, systemImage: "paperclip")
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                Text("Open the web app to review file submissions in full.")
                                    .font(.caption2)
                                    .italic()
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }

                        LMSCard {
                            Text("Score")
                                .font(LexturesTheme.displayFont(17))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            HStack(spacing: 10) {
                                TextField("Points", text: $pointsText)
                                    .keyboardType(.decimalPad)
                                    .font(LexturesTheme.displayFont(22, weight: .bold))
                                    .padding(.horizontal, 14)
                                    .padding(.vertical, 10)
                                    .frame(width: 120)
                                    .background(LexturesTheme.sceneBackground(for: colorScheme).opacity(0.7))
                                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                                    .overlay(
                                        RoundedRectangle(cornerRadius: 12, style: .continuous)
                                            .stroke(
                                                pointsText.isEmpty || pointsValid
                                                    ? LexturesTheme.fieldBorder(for: colorScheme)
                                                    : LexturesTheme.error,
                                                lineWidth: 1
                                            )
                                    )
                                if let max = maxPoints {
                                    Text("/ \(max.formatted()) pts")
                                        .font(.subheadline.weight(.semibold))
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                }
                            }
                            if !pointsText.isEmpty && !pointsValid {
                                Text(maxPoints.map { "Enter a number between 0 and \($0.formatted())." } ?? "Enter a valid number.")
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.error)
                            }
                        }

                        LMSCard {
                            Text("Feedback")
                                .font(LexturesTheme.displayFont(17))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            TextField("Comment for the student (optional)", text: $comment, axis: .vertical)
                                .lineLimit(4 ... 8)
                                .padding(12)
                                .background(LexturesTheme.sceneBackground(for: colorScheme).opacity(0.7))
                                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                                .overlay(
                                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                                        .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
                                )
                        }

                        Button {
                            Task { await save() }
                        } label: {
                            if saving {
                                ProgressView()
                                    .frame(maxWidth: .infinity)
                            } else {
                                Text("Save grade")
                            }
                        }
                        .buttonStyle(AuthPrimaryButtonStyle())
                        .disabled(!pointsValid || saving || loading)
                    }
                    .padding(16)
                }
            }
            .navigationTitle("Grade submission")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Cancel") { dismiss() }
                }
            }
            .task { await loadExisting() }
        }
        .presentationDetents([.large])
    }

    private func loadExisting() async {
        guard let token = session.accessToken else {
            loading = false
            return
        }
        loading = true
        defer { loading = false }
        // Pre-fill when a grade already exists; also learns maxPoints for validation.
        if let grade = try? await LMSAPI.fetchSubmissionGrade(
            courseCode: course.courseCode,
            itemId: assignmentId,
            submissionId: submission.id,
            accessToken: token
        ) {
            maxPoints = grade.maxPoints
            if let earned = grade.pointsEarned {
                pointsText = earned.formatted()
            }
            comment = grade.instructorComment ?? ""
        }
    }

    private func save() async {
        guard let token = session.accessToken, let points = pointsValue else { return }
        saving = true
        errorMessage = nil
        defer { saving = false }
        do {
            try await LMSAPI.putSubmissionGrade(
                courseCode: course.courseCode,
                itemId: assignmentId,
                submissionId: submission.id,
                body: .init(
                    pointsEarned: points,
                    instructorComment: comment.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : comment
                ),
                accessToken: token
            )
            onSaved()
            dismiss()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not save the grade."
        }
    }
}
