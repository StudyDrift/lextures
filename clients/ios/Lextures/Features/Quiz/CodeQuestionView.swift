import SwiftUI

struct CodeQuestionRunContext: Equatable {
    let courseCode: String
    let itemId: String
    let attemptId: String
    let accessToken: String
}

struct CodeQuestionView: View {
    @Environment(\.colorScheme) private var colorScheme

    let question: QuizQuestion
    let answer: QuizAnswerState
    let runContext: CodeQuestionRunContext?
    let onChange: (QuizAnswerState) -> Void

    @State private var running = false
    @State private var runError: String?
    @State private var runResult: QuizCodeRunResponse?
    @State private var seededStarter = false

    private var language: String { QuizLogic.codeLanguageLabel(for: question) }

    var body: some View {
        if QuizLogic.isCodeQuestionOversized(question) {
            oversizedCard
        } else {
            editorBody
        }
    }

    private var oversizedCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Label(L.text("mobile.quiz.code.oversizedTitle"), systemImage: "macwindow")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(L.text("mobile.quiz.code.oversizedHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.amber.opacity(0.1))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
    }

    private var editorBody: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(L.format("mobile.quiz.code.language", language))
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            CodeEditor(
                text: Binding(
                    get: { answer.text ?? "" },
                    set: { value in
                        var next = answer
                        next.text = value
                        onChange(next)
                    }
                ),
                language: language,
                onInsert: insertSnippet
            )

            Button {
                Task { await runTests() }
            } label: {
                HStack {
                    if running {
                        ProgressView()
                            .controlSize(.small)
                    }
                    Text(running ? L.text("mobile.quiz.code.running") : L.text("mobile.quiz.code.run"))
                        .font(.subheadline.weight(.semibold))
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 10)
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.primary)
            .disabled(running || runContext == nil || (answer.text ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)

            if let runError {
                Text(runError)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
            }

            if let runResult {
                resultsPanel(runResult)
            }
        }
        .onAppear {
            guard !seededStarter else { return }
            seededStarter = true
            let initial = QuizLogic.initialCodeAnswer(for: question, existing: answer)
            if initial != answer {
                onChange(initial)
            }
        }
    }

    private func insertSnippet(_ snippet: String) {
        var next = answer
        let current = next.text ?? ""
        next.text = current + snippet
        onChange(next)
    }

    @MainActor
    private func runTests() async {
        guard let runContext else { return }
        running = true
        runError = nil
        defer { running = false }
        do {
            let result = try await LMSAPI.postQuizQuestionRun(
                courseCode: runContext.courseCode,
                itemId: runContext.itemId,
                attemptId: runContext.attemptId,
                questionId: question.id,
                code: answer.text ?? "",
                languageId: question.typeConfig?.languageId,
                accessToken: runContext.accessToken
            )
            runResult = result
        } catch {
            runError = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.quiz.code.runFailed")
        }
    }

    @ViewBuilder
    private func resultsPanel(_ response: QuizCodeRunResponse) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(
                L.format(
                    "mobile.quiz.code.runScore",
                    response.pointsEarned.formatted(),
                    response.pointsPossible.formatted()
                )
            )
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            ForEach(Array(response.results.enumerated()), id: \.offset) { index, result in
                VStack(alignment: .leading, spacing: 4) {
                    HStack {
                        Text(L.format("mobile.quiz.code.testNumber", index + 1))
                            .font(.caption.weight(.semibold))
                        Spacer()
                        Text(codeRunStatusLabel(result.status))
                            .font(.caption.weight(.bold))
                            .foregroundStyle(result.passed ? LexturesTheme.primary : LexturesTheme.coral)
                    }
                    if !result.expectedOutput.isEmpty {
                        Text(L.format("mobile.quiz.code.expected", result.expectedOutput))
                            .font(.caption2.monospaced())
                    }
                    Text(L.format("mobile.quiz.code.actual", result.actualOutput))
                        .font(.caption2.monospaced())
                    if let stderr = result.stderr, !stderr.isEmpty {
                        Text(L.format("mobile.quiz.code.stderr", stderr))
                            .font(.caption2.monospaced())
                            .foregroundStyle(LexturesTheme.coral)
                    }
                }
                .padding(10)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(LexturesTheme.sceneBackground(for: colorScheme))
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
            }
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel(L.text("mobile.quiz.code.resultsA11y"))
    }

    private func codeRunStatusLabel(_ status: String) -> String {
        switch status {
        case "pass": return L.text("mobile.quiz.code.statusPass")
        case "fail": return L.text("mobile.quiz.code.statusFail")
        case "tle": return L.text("mobile.quiz.code.statusTle")
        case "mle": return L.text("mobile.quiz.code.statusMle")
        case "re": return L.text("mobile.quiz.code.statusRe")
        case "ce": return L.text("mobile.quiz.code.statusCe")
        default: return status.uppercased()
        }
    }
}
