import SwiftUI

/// Post-submit results per quiz reveal policy (M4.1).
struct QuizResultsView: View {
    @Environment(\.colorScheme) private var colorScheme

    let title: String
    let results: QuizResultsResponse
    var onDone: () -> Void

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    header
                    if results.academicIntegrityFlag == true {
                        integrityBanner
                    }
                    if let score = results.score {
                        scoreCard(score)
                    } else {
                        pendingCard
                    }
                    if let questions = results.questions, !questions.isEmpty {
                        feedbackSection(questions)
                    }
                    Button(L.text("mobile.quiz.done")) { onDone() }
                        .buttonStyle(AuthPrimaryButtonStyle())
                }
                .padding(16)
            }
        }
        .navigationTitle(title)
        .navigationBarTitleDisplayMode(.inline)
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 6) {
            Label(L.text("mobile.quiz.submitted"), systemImage: "checkmark.seal.fill")
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.primary)
            Text(L.format("mobile.quiz.attemptNumber", results.attemptNumber))
                .font(LexturesTheme.displayFont(22))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
    }

    private var integrityBanner: some View {
        LMSCard(accent: LexturesTheme.coral) {
            Label(L.text("mobile.quiz.integrityFlag"), systemImage: "exclamationmark.shield.fill")
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.coral)
        }
    }

    private func scoreCard(_ score: QuizResultsScoreSummary) -> some View {
        LMSCard(accent: LexturesTheme.primary) {
            Text(L.text("mobile.quiz.yourScore"))
                .font(LexturesTheme.displayFont(18))
            HStack(alignment: .firstTextBaseline) {
                Text("\(score.pointsEarned.formatted()) / \(score.pointsPossible.formatted())")
                    .font(LexturesTheme.displayFont(28, weight: .bold))
                    .foregroundStyle(LexturesTheme.primary)
                Spacer()
                Text("\(Int(score.scorePercent.rounded()))%")
                    .font(LexturesTheme.displayFont(22, weight: .bold))
            }
        }
    }

    private var pendingCard: some View {
        LMSCard(accent: LexturesTheme.amber) {
            Label(L.text("mobile.quiz.pendingReview"), systemImage: "clock.fill")
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(L.text("mobile.quiz.pendingReviewHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private func feedbackSection(_ questions: [QuizResultsQuestionResult]) -> some View {
        LMSCard {
            Text(L.text("mobile.quiz.feedback"))
                .font(LexturesTheme.displayFont(18))
            ForEach(questions) { question in
                Divider().padding(.vertical, 4)
                VStack(alignment: .leading, spacing: 6) {
                    Text(L.format("mobile.quiz.questionNumber", question.questionIndex + 1))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if let prompt = question.promptSnapshot, !prompt.isEmpty {
                        Text(prompt)
                            .font(.subheadline)
                            .lineLimit(4)
                    }
                    if let awarded = question.pointsAwarded {
                        HStack {
                            if let correct = question.isCorrect {
                                Image(systemName: correct ? "checkmark.circle.fill" : "xmark.circle.fill")
                                    .foregroundStyle(correct ? LexturesTheme.primary : LexturesTheme.coral)
                            }
                            Text("\(awarded.formatted()) / \(question.maxPoints.formatted()) pts")
                                .font(.caption.weight(.semibold))
                        }
                    }
                }
            }
        }
    }
}
