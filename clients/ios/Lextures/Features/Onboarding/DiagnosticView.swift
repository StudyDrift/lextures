import SwiftUI

/// Multiple-choice placement diagnostic (parity with web onboarding step 3).
struct DiagnosticView: View {
    let questions: [DiagnosticQuestion]
    @Binding var questionIndex: Int
    @Binding var answers: [String: Int]
    var submitting: Bool
    var onContinue: () -> Void
    var onSkip: () -> Void

    @Environment(\.colorScheme) private var colorScheme

    private var current: DiagnosticQuestion? {
        guard questions.indices.contains(questionIndex) else { return nil }
        return questions[questionIndex]
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text(L.text("mobile.onboarding.diagnostic.subtitle"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            if let question = current {
                Text(L.format("mobile.onboarding.diagnostic.questionOf", questionIndex + 1, questions.count))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .accessibilityAddTraits(.isHeader)

                Text(question.prompt)
                    .font(.body.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .fixedSize(horizontal: false, vertical: true)

                VStack(spacing: 8) {
                    ForEach(Array(question.choices.enumerated()), id: \.offset) { idx, choice in
                        choiceRow(questionId: question.id, index: idx, label: choice)
                    }
                }

                HStack(spacing: 12) {
                    if questionIndex < questions.count - 1 {
                        Button(L.text("mobile.onboarding.diagnostic.nextQuestion")) {
                            questionIndex += 1
                        }
                        .buttonStyle(AuthPrimaryButtonStyle())
                        .disabled(answers[question.id] == nil || submitting)
                    } else {
                        Button(L.text("mobile.onboarding.continue")) {
                            onContinue()
                        }
                        .buttonStyle(AuthPrimaryButtonStyle())
                        .disabled(answers[question.id] == nil || submitting)
                    }
                }
                .padding(.top, 4)
            } else if questions.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity)
            }

            Button(L.text("mobile.onboarding.skipDiagnostic"), action: onSkip)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .disabled(submitting)
                .frame(maxWidth: .infinity, alignment: .leading)
                .accessibilityHint(L.text("mobile.onboarding.skipForNow"))
        }
    }

    private func choiceRow(questionId: String, index: Int, label: String) -> some View {
        let selected = answers[questionId] == index
        return Button {
            answers[questionId] = index
        } label: {
            HStack(spacing: 10) {
                Image(systemName: selected ? "largecircle.fill.circle" : "circle")
                    .foregroundStyle(selected ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.textSecondary(for: colorScheme))
                Text(label)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .multilineTextAlignment(.leading)
                Spacer(minLength: 0)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .background(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .fill(selected ? LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.15 : 0.12) : LexturesTheme.cardBackground(for: colorScheme))
            )
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(
                        selected ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.fieldBorder(for: colorScheme),
                        lineWidth: 1
                    )
            )
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(selected ? .isSelected : [])
    }
}
