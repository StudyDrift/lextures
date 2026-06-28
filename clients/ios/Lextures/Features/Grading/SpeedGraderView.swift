import SwiftUI

/// SpeedGrader-style flow: page through an assignment's submissions, read each
/// one, and enter a score/feedback without returning to the list. Seeded with the
/// already-loaded submissions snapshot and the index of the tapped student.
struct SpeedGraderView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let course: CourseSummary
    let assignmentId: String
    let submissions: [AssignmentSubmission]
    var onSaved: () -> Void = {}

    @State private var index: Int
    @State private var graded: Set<String>
    @State private var pointsText = ""
    @State private var comment = ""
    @State private var maxPoints: Double?
    @State private var loading = true
    @State private var saving = false
    @State private var errorMessage: String?

    init(
        course: CourseSummary,
        assignmentId: String,
        submissions: [AssignmentSubmission],
        startIndex: Int,
        onSaved: @escaping () -> Void = {}
    ) {
        self.course = course
        self.assignmentId = assignmentId
        self.submissions = submissions
        self.onSaved = onSaved
        _index = State(initialValue: min(max(startIndex, 0), max(submissions.count - 1, 0)))
        _graded = State(initialValue: Set(submissions.filter { $0.isGraded == true }.map(\.id)))
    }

    private var current: AssignmentSubmission? {
        submissions.indices.contains(index) ? submissions[index] : nil
    }

    private var remainingUngraded: Int {
        submissions.filter { !graded.contains($0.id) }.count
    }

    private var pointsValue: Double? {
        Double(pointsText.replacingOccurrences(of: ",", with: "."))
    }

    private var pointsValid: Bool {
        guard let value = pointsValue else { return false }
        if let max = maxPoints { return value >= 0 && value <= max }
        return value >= 0
    }

    /// First ungraded index strictly after the current one; nil when none remain.
    private func nextUngradedIndex(after start: Int) -> Int? {
        guard !submissions.isEmpty else { return nil }
        for offset in 1...submissions.count {
            let candidate = (start + offset) % submissions.count
            if candidate == start { break }
            if !graded.contains(submissions[candidate].id) { return candidate }
        }
        return nil
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                if let submission = current {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 14) {
                            progressCard
                            if let errorMessage {
                                LMSErrorBanner(message: errorMessage)
                            }
                            studentCard(submission)
                            submissionCard(submission)
                            scoreCard
                            feedbackCard
                            actionBar
                        }
                        .padding(16)
                    }
                } else {
                    LMSEmptyState(
                        systemImage: "checkmark.seal.fill",
                        title: "Nothing to grade",
                        message: "There are no submissions in this view."
                    )
                    .padding(24)
                }
            }
            .navigationTitle("Speed grader")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Done") { dismiss() }
                }
            }
            .task(id: index) { await loadGrade() }
        }
    }

    // MARK: - Cards

    private var progressCard: some View {
        HStack(spacing: 10) {
            Text("\(index + 1) of \(submissions.count)")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Spacer(minLength: 0)
            Text(remainingUngraded == 0 ? "All graded" : "\(remainingUngraded) ungraded")
                .font(.caption.weight(.semibold))
                .foregroundStyle(remainingUngraded == 0 ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.amber)
                .padding(.horizontal, 9)
                .padding(.vertical, 3)
                .background((remainingUngraded == 0 ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.amber).opacity(0.14))
                .clipShape(Capsule())
        }
    }

    private func studentCard(_ submission: AssignmentSubmission) -> some View {
        LMSCard {
            HStack(spacing: 12) {
                Circle()
                    .fill(LexturesTheme.coverGradient(for: submission.displayName))
                    .frame(width: 42, height: 42)
                    .overlay(
                        Text(String(submission.displayName.prefix(2)).uppercased())
                            .font(.subheadline.weight(.bold))
                            .foregroundStyle(.white)
                    )
                VStack(alignment: .leading, spacing: 3) {
                    Text(submission.displayName)
                        .font(LexturesTheme.displayFont(18))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    HStack(spacing: 6) {
                        Text("Submitted \(LMSDates.shortDateTime(submission.submittedAt))")
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        if let version = submission.versionNumber, version > 1 {
                            Text("v\(version)")
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                        }
                    }
                }
                Spacer(minLength: 0)
                if graded.contains(submission.id) {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                }
            }
        }
    }

    @ViewBuilder
    private func submissionCard(_ submission: AssignmentSubmission) -> some View {
        let body = submission.bodyText?.trimmingCharacters(in: .whitespacesAndNewlines)
        let filename = submission.attachmentFilename
        if (body?.isEmpty == false) || (filename?.isEmpty == false) {
            LMSCard {
                Text("Submission")
                    .font(LexturesTheme.displayFont(17))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                if let body, !body.isEmpty {
                    Text(body)
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .textSelection(.enabled)
                }
                if let filename, !filename.isEmpty {
                    Label(filename, systemImage: "paperclip")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text("Open the web app to review file submissions in full.")
                        .font(.caption2)
                        .italic()
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        } else {
            LMSCard {
                Text("Submission")
                    .font(LexturesTheme.displayFont(17))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text("No text or attachment was submitted.")
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private var scoreCard: some View {
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
    }

    private var feedbackCard: some View {
        LMSCard {
            Text("Feedback")
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            TextField("Comment for the student (optional)", text: $comment, axis: .vertical)
                .lineLimit(3 ... 6)
                .padding(12)
                .background(LexturesTheme.sceneBackground(for: colorScheme).opacity(0.7))
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
                )
        }
    }

    private var actionBar: some View {
        VStack(spacing: 10) {
            Button {
                Task { await save() }
            } label: {
                if saving {
                    ProgressView().frame(maxWidth: .infinity)
                } else {
                    Text(remainingUngraded <= 1 && pointsValid ? "Save grade" : "Save & next")
                }
            }
            .buttonStyle(AuthPrimaryButtonStyle())
            .disabled(!pointsValid || saving || loading)

            HStack(spacing: 12) {
                navButton(title: "Previous", systemImage: "chevron.left", disabled: index == 0 || saving) {
                    index = max(index - 1, 0)
                }
                navButton(title: "Next", systemImage: "chevron.right", disabled: index >= submissions.count - 1 || saving) {
                    index = min(index + 1, submissions.count - 1)
                }
            }
        }
    }

    private func navButton(title: String, systemImage: String, disabled: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Label(title, systemImage: systemImage)
                .font(.subheadline.weight(.semibold))
                .frame(maxWidth: .infinity)
                .padding(.vertical, 12)
                .foregroundStyle(disabled ? LexturesTheme.textSecondary(for: colorScheme).opacity(0.5) : LexturesTheme.accent(for: colorScheme))
                .background(LexturesTheme.cardBackground(for: colorScheme))
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
                )
        }
        .buttonStyle(.plain)
        .disabled(disabled)
    }

    // MARK: - Data

    private func loadGrade() async {
        guard let token = session.accessToken, let submission = current else {
            loading = false
            return
        }
        loading = true
        errorMessage = nil
        pointsText = ""
        comment = ""
        maxPoints = nil
        defer { loading = false }
        if let grade = try? await LMSAPI.fetchSubmissionGrade(
            courseCode: course.courseCode,
            itemId: assignmentId,
            submissionId: submission.id,
            accessToken: token
        ) {
            maxPoints = grade.maxPoints
            if let earned = grade.pointsEarned {
                pointsText = earned.formatted()
                graded.insert(submission.id)
            }
            comment = grade.instructorComment ?? ""
        }
    }

    private func save() async {
        guard let token = session.accessToken, let points = pointsValue, let submission = current else { return }
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
            graded.insert(submission.id)
            onSaved()
            if let next = nextUngradedIndex(after: index) {
                index = next
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not save the grade."
        }
    }
}
