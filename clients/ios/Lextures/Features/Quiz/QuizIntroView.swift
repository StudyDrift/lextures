import SwiftUI

/// Quiz intro: rules, attempts, and Start / Resume (M4.1).
struct QuizIntroView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let item: CourseStructureItem
    var onProgressChanged: (() async -> Void)?

    @State private var detail: ModuleItemDetail?
    @State private var quizPayload: ModuleQuizPayload?
    @State private var pastAttempts: [QuizAttemptSummary] = []
    @State private var accessCode = ""
    @State private var loading = true
    @State private var starting = false
    @State private var errorMessage: String?
    @State private var startResponse: QuizStartResponse?
    @State private var showTaker = false
    @State private var showPreview = false

    private var courseCode: String { course.courseCode }
    private var isStaff: Bool { course.viewerIsStaff }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 40)
                    } else {
                        introHeader
                        if let markdown = detail?.markdown ?? quizPayload?.markdown,
                           !markdown.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                            LMSCard {
                                CourseMarkdownContentView(markdown: markdown)
                                    .lexturesReadableText()
                            }
                        }
                        rulesCard
                        if !pastAttempts.isEmpty {
                            previousAttemptsCard
                        }
                        startSection
                    }
                }
                .padding(16)
            }
            .refreshable { await load() }
        }
        .navigationTitle(item.title)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .navigationDestination(isPresented: $showTaker) {
            if let start = startResponse {
                QuizTakerView(
                    course: course,
                    item: item,
                    quiz: quizPayload,
                    detail: detail,
                    start: start,
                    onFinished: {
                        showTaker = false
                        Task {
                            await load()
                            await onProgressChanged?()
                        }
                    }
                )
            }
        }
        .navigationDestination(isPresented: $showPreview) {
            if let payload = quizPayload {
                QuizPreviewView(title: item.title, quiz: payload)
            }
        }
    }

    private var introHeader: some View {
        VStack(alignment: .leading, spacing: 8) {
            Label(L.text("mobile.quiz.label"), systemImage: "checkmark.circle.fill")
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            Text(detail?.title ?? quizPayload?.title ?? item.title)
                .font(LexturesTheme.displayFont(24))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            if let count = quizPayload?.questionCount ?? detail?.questionCount, count > 0 {
                Text(L.format("mobile.quiz.questionCount", count))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private var rulesCard: some View {
        LMSCard {
            Text(L.text("mobile.quiz.rulesTitle"))
                .font(LexturesTheme.displayFont(18))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            let rows = ItemDetailRows.rows(for: item, detail: detail, pointsValue: pointsValue)
            if !rows.isEmpty {
                Divider().padding(.vertical, 2)
                ForEach(rows, id: \.0) { label, value in
                    HStack {
                        Text(label)
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        Spacer(minLength: 12)
                        Text(value)
                            .font(.subheadline.weight(.semibold))
                            .multilineTextAlignment(.trailing)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    }
                }
            }

            if let policy = detail?.gradeAttemptPolicy ?? quizPayload?.gradeAttemptPolicy {
                Text(QuizLogic.retakePolicyNotice(policy))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .padding(.top, 4)
            }
        }
    }

    private var previousAttemptsCard: some View {
        LMSCard(accent: LexturesTheme.brandTeal) {
            Text(L.text("mobile.quiz.previousAttempts"))
                .font(LexturesTheme.displayFont(18))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            ForEach(pastAttempts) { attempt in
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L.format("mobile.quiz.attemptNumber", attempt.attemptNumber))
                            .font(.subheadline.weight(.medium))
                        if !attempt.submittedAt.isEmpty {
                            Text(LMSDates.shortDateTime(attempt.submittedAt))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }
                    Spacer()
                    if let score = attempt.scorePercent {
                        Text("\(Int(score.rounded()))%")
                            .font(.subheadline.weight(.bold))
                            .foregroundStyle(LexturesTheme.primary)
                    } else if attempt.needsManualGrading == true {
                        Text(L.text("mobile.quiz.pendingReview"))
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(LexturesTheme.amber)
                    }
                }
                .padding(.vertical, 4)
            }
        }
    }

    @ViewBuilder
    private var startSection: some View {
        if quizPayload?.isAdaptive == true {
            LMSCard {
                Label(L.text("mobile.quiz.adaptiveWebOnly"), systemImage: "safari")
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        } else if isStaff {
            Button {
                showPreview = true
            } label: {
                Text(L.text("mobile.quiz.previewQuiz"))
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(AuthPrimaryButtonStyle())
        } else if !canStartAttempt {
            LMSCard {
                Label(L.text("mobile.quiz.noAttemptsRemaining"), systemImage: "exclamationmark.circle")
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(LexturesTheme.coral)
            }
        } else {
            if detail?.requiresQuizAccessCode == true || quizPayload?.requiresQuizAccessCode == true {
                LMSCard {
                    Text(L.text("mobile.quiz.accessCode"))
                        .font(.subheadline.weight(.semibold))
                    SecureField(L.text("mobile.quiz.accessCodePlaceholder"), text: $accessCode)
                        .textContentType(.password)
                }
            }
            Button {
                Task { await startAttempt() }
            } label: {
                HStack {
                    if starting {
                        ProgressView()
                    } else {
                        Text(resumeLabel)
                    }
                }
                .frame(maxWidth: .infinity)
            }
            .buttonStyle(AuthPrimaryButtonStyle())
            .disabled(starting)
        }
    }

    private var resumeLabel: String {
        if pastAttempts.contains(where: { $0.submittedAt.isEmpty }) {
            return L.text("mobile.quiz.resumeAttempt")
        }
        return L.text("mobile.quiz.startAttempt")
    }

    private var canStartAttempt: Bool {
        if quizPayload?.isAdaptive == true { return false }
        if detail?.unlimitedAttempts == true || quizPayload?.unlimitedAttempts == true { return true }
        let max = detail?.maxAttempts ?? quizPayload?.maxAttempts ?? 1
        return pastAttempts.count < max
    }

    private var pointsValue: Int? {
        if let pts = detail?.pointsWorth { return pts }
        if let pts = quizPayload?.pointsWorth { return pts }
        if let pts = item.pointsWorth { return Int(pts) }
        return nil
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            detail = try await LMSAPI.fetchItemDetail(courseCode: courseCode, item: item, accessToken: token)
            quizPayload = try await LMSAPI.fetchModuleQuiz(
                courseCode: courseCode,
                itemId: item.id,
                attemptId: nil,
                accessToken: token
            )
            pastAttempts = try await LMSAPI.fetchQuizAttempts(
                courseCode: courseCode,
                itemId: item.id,
                accessToken: token
            )
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.modules.loadError")
        }
    }

    private func startAttempt() async {
        guard let token = session.accessToken else { return }
        starting = true
        errorMessage = nil
        defer { starting = false }
        do {
            let start = try await LMSAPI.startQuiz(
                courseCode: courseCode,
                itemId: item.id,
                accessCode: accessCode.isEmpty ? nil : accessCode,
                accessToken: token
            )
            quizPayload = try await LMSAPI.fetchModuleQuiz(
                courseCode: courseCode,
                itemId: item.id,
                attemptId: start.attemptId,
                accessToken: token
            )
            startResponse = start
            showTaker = true
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.quiz.startError")
        }
    }
}

/// Read-only teacher preview of a quiz. Renders every question with the shared
/// question view using ephemeral local answers; nothing is saved and no attempt
/// is started. Mirrors the web "Student preview" modal.
private struct QuizPreviewView: View {
    @Environment(\.colorScheme) private var colorScheme

    let title: String
    let quiz: ModuleQuizPayload

    @State private var answers: [String: QuizAnswerState] = [:]

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    LMSCard {
                        Text(L.text("mobile.quiz.previewNote"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    let questions = quiz.questions ?? []
                    if questions.isEmpty {
                        LMSCard {
                            Text(L.text("mobile.quiz.previewEmpty"))
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    } else {
                        ForEach(Array(questions.enumerated()), id: \.offset) { _, question in
                            QuizQuestionView(
                                question: question,
                                answer: answers[question.id] ?? QuizAnswerState(),
                                saveState: .idle,
                                onChange: { answers[question.id] = $0 },
                                isFlagged: false,
                                onToggleFlag: {}
                            )
                        }
                    }
                }
                .padding(16)
            }
        }
        .navigationTitle(title)
        .navigationBarTitleDisplayMode(.inline)
    }
}
