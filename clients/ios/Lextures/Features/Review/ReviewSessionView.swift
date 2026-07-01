import SwiftUI

@MainActor
@Observable
final class ReviewSessionModel {
    var queue: [ReviewQueueItem]
    let totalDue: Int
    let initialStreak: Int
    var revealed = false
    var reviewedCount = 0
    var errorMessage: String?
    var submitting = false
    var finished = false
    private var shownAt = Date()

    init(initialQueue: [ReviewQueueItem], totalDue: Int, initialStreak: Int) {
        queue = initialQueue
        self.totalDue = max(totalDue, initialQueue.count)
        self.initialStreak = initialStreak
    }

    var current: ReviewQueueItem? { queue.first }

    var progressLabel: String {
        let currentIndex = reviewedCount + (current == nil ? 0 : 1)
        return L.format("mobile.review.progress", currentIndex, totalDue)
    }

    func reveal() {
        revealed = true
    }

    func submit(
        grade: SrsGrade,
        accessToken: String?,
        userId: String?,
        offline: OfflineService
    ) async {
        guard let accessToken, let userId, let current, !submitting else { return }
        submitting = true
        errorMessage = nil
        defer { submitting = false }

        let ratedAt = Date()
        let responseMs = Int(ratedAt.timeIntervalSince(shownAt) * 1000)
        let body = SrsReviewSubmitBody(
            questionId: current.questionId,
            grade: grade.rawValue,
            responseMs: responseMs
        )
        let path = "/api/v1/learners/\(LMSAPI.encodePath(userId))/review"
        let idempotencyKey = ReviewLogic.idempotencyKey(questionId: current.questionId, ratedAt: ratedAt)

        do {
            _ = try await offline.enqueueMutation(
                method: "POST",
                path: path,
                body: body,
                label: L.text("mobile.review.submitLabel"),
                accessToken: accessToken,
                idempotencyKey: idempotencyKey
            )
            reviewedCount += 1
            queue.removeFirst()
            revealed = false
            shownAt = Date()
            if queue.isEmpty {
                finished = true
            }
        } catch {
            errorMessage = L.text("mobile.review.error.submit")
        }
    }
}

struct ReviewSessionView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Environment(\.dismiss) private var dismiss

    @State private var model: ReviewSessionModel
    let onFinished: () -> Void

    init(
        initialQueue: [ReviewQueueItem],
        totalDue: Int,
        initialStreak: Int,
        onFinished: @escaping () -> Void
    ) {
        _model = State(initialValue: ReviewSessionModel(
            initialQueue: initialQueue,
            totalDue: totalDue,
            initialStreak: initialStreak
        ))
        self.onFinished = onFinished
    }

    private var userId: String? {
        shell.profile?.id ?? NotebookStore.jwtSubject(from: session.accessToken)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if model.finished {
                summaryView
            } else if let current = model.current {
                sessionContent(current)
            } else {
                ProgressView()
            }
        }
        .navigationTitle(L.text("mobile.review.sessionTitle"))
        .navigationBarTitleDisplayMode(.inline)
    }

    @ViewBuilder
    private func sessionContent(_ current: ReviewQueueItem) -> some View {
        VStack(spacing: 16) {
            HStack {
                Text(model.progressLabel)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .accessibilityLabel(model.progressLabel)
                Spacer(minLength: 0)
                Text(current.courseTitle)
                    .font(.caption2.monospaced())
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .lineLimit(1)
            }
            .padding(.horizontal, 16)

            ScrollView {
                VStack(spacing: 16) {
                    promptCard(current)
                    if model.revealed {
                        answerCard(current)
                    }
                    if let error = model.errorMessage {
                        LMSErrorBanner(message: error)
                    }
                }
                .padding(.horizontal, 16)
            }

            actionBar
        }
        .padding(.vertical, 16)
    }

    private func promptCard(_ current: ReviewQueueItem) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.review.question"))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .textCase(.uppercase)

                if let quizQuestion = ReviewLogic.toQuizQuestion(current), !model.revealed {
                    ReviewQuestionPreview(question: quizQuestion)
                } else {
                    CourseMarkdownContentView(markdown: current.stem)
                        .lexturesReadableText()
                }

                if !model.revealed {
                    Button(L.text("mobile.review.reveal")) {
                        if reduceMotion {
                            model.reveal()
                        } else {
                            withAnimation(.easeInOut(duration: 0.2)) {
                                model.reveal()
                            }
                        }
                    }
                    .buttonStyle(.bordered)
                    .frame(minHeight: 44)
                    .accessibilityHint(L.text("mobile.review.revealHint"))
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .accessibilityElement(children: .contain)
    }

    private func answerCard(_ current: ReviewQueueItem) -> some View {
        LMSCard(accent: LexturesTheme.primary) {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.review.answer"))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.primary)
                    .textCase(.uppercase)
                Text(ReviewLogic.formatAnswerPreview(current.correctAnswer))
                    .font(.body)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .textSelection(.enabled)
                if let explanation = current.explanation, !explanation.isEmpty {
                    Text(explanation)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .transition(reduceMotion ? .opacity : .move(edge: .bottom).combined(with: .opacity))
    }

    private var actionBar: some View {
        Group {
            if model.revealed {
                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 10) {
                    ForEach(SrsGrade.allCases) { grade in
                        Button {
                            Task {
                                await model.submit(
                                    grade: grade,
                                    accessToken: session.accessToken,
                                    userId: userId,
                                    offline: offline
                                )
                            }
                        } label: {
                            Text(grade.label)
                                .font(.subheadline.weight(.semibold))
                                .frame(maxWidth: .infinity, minHeight: 48)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(gradeColor(grade))
                        .disabled(model.submitting)
                        .accessibilityLabel(grade.label)
                    }
                }
                .padding(.horizontal, 16)
            }
        }
    }

    private var summaryView: some View {
        VStack(spacing: 20) {
            Spacer(minLength: 0)
            Image(systemName: "checkmark.seal.fill")
                .font(.system(size: 56))
                .foregroundStyle(LexturesTheme.primary)
            Text(L.text("mobile.review.summaryTitle"))
                .font(LexturesTheme.displayFont(24, weight: .bold))
            Text(L.plural("mobile.review.summaryReviewed", count: model.reviewedCount))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            if model.initialStreak > 0 || model.reviewedCount > 0 {
                Label(
                    L.plural("mobile.review.streak", count: max(model.initialStreak, model.reviewedCount > 0 ? model.initialStreak + 1 : model.initialStreak)),
                    systemImage: "flame.fill"
                )
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.amber)
            }
            Button(L.text("mobile.review.done")) {
                onFinished()
                dismiss()
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.primary)
            Spacer(minLength: 0)
        }
        .padding(24)
    }

    private func gradeColor(_ grade: SrsGrade) -> Color {
        switch grade {
        case .again: return LexturesTheme.coral
        case .hard: return LexturesTheme.amber
        case .good: return LexturesTheme.primary
        case .easy: return Color.green
        }
    }
}

/// Read-only quiz prompt for typed SRS items (M4.1 reuse).
private struct ReviewQuestionPreview: View {
    @Environment(\.colorScheme) private var colorScheme
    let question: QuizQuestion

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            CourseMarkdownContentView(markdown: question.prompt)
                .lexturesReadableText()
            switch QuizQuestionKind(raw: question.questionType) {
            case .multipleChoice, .trueFalse:
                ForEach(Array(QuizLogic.visibleChoices(question).enumerated()), id: \.offset) { _, choice in
                    HStack(spacing: 10) {
                        Image(systemName: "circle")
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        Text(choice)
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Spacer(minLength: 0)
                    }
                    .padding(10)
                    .background(LexturesTheme.cardBackground(for: colorScheme))
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                }
            case .ordering:
                ForEach(Array(QuizLogic.orderingItems(question).enumerated()), id: \.offset) { index, item in
                    HStack {
                        Text("\(index + 1).")
                            .font(.caption.monospaced())
                        Text(item)
                            .font(.subheadline)
                        Spacer(minLength: 0)
                    }
                }
            default:
                EmptyView()
            }
        }
    }
}
