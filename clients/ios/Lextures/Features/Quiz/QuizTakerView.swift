import SwiftUI

@MainActor
@Observable
final class QuizTakerModel {
    var questions: [QuizQuestion] = []
    var currentIndex = 0
    var answers: [String: QuizAnswerState] = [:]
    var saveStates: [String: QuizSaveState] = [:]
    var flaggedIds: Set<String> = []
    var errorMessage: String?
    var loading = true
    var advancing = false
    var submitting = false
    var showSubmitConfirm = false
    var showResults = false
    var results: QuizResultsResponse?
    var timerSeconds: Int?
    var timerNotice: String?
    var serverLockdown = false
    var serverQuestion: QuizQuestion?
    var serverCompleted = false
    var totalQuestions = 0
    var allowBackNavigation = true
    var oneQuestionAtATime = true

    let courseCode: String
    let itemId: String
    let attemptId: String
    let deadlineAt: String?
    private var submitGuard = false

    init(courseCode: String, itemId: String, start: QuizStartResponse) {
        self.courseCode = courseCode
        self.itemId = itemId
        attemptId = start.attemptId
        deadlineAt = start.deadlineAt
        serverLockdown = QuizLogic.isServerLockdown(start.lockdownMode)
        allowBackNavigation = start.backNavigationAllowed ?? true
        currentIndex = Int(start.currentQuestionIndex)
        timerSeconds = QuizLogic.secondsRemaining(deadlineISO: start.deadlineAt)
    }

    var activeQuestions: [QuizQuestion] {
        if serverLockdown, let serverQuestion {
            return [serverQuestion]
        }
        return questions
    }

    var currentQuestion: QuizQuestion? {
        if serverLockdown { return serverQuestion }
        guard currentIndex >= 0, currentIndex < questions.count else { return nil }
        return questions[currentIndex]
    }

    var totalCount: Int {
        if serverLockdown { return max(Int(serverQuestion != nil ? 1 : 0), currentIndex + 1) }
        return questions.count
    }

    var answeredCount: Int {
        activeQuestionsForPalette.filter { QuizLogic.isAnswered(question: $0, answer: answers[$0.id]) }.count
    }

    var activeQuestionsForPalette: [QuizQuestion] {
        serverLockdown ? questions : questions
    }

    func load(
        quiz: ModuleQuizPayload?,
        start: QuizStartResponse,
        accessToken: String
    ) async {
        loading = true
        errorMessage = nil
        defer { loading = false }
        oneQuestionAtATime = quiz?.oneQuestionAtATime ?? true
        do {
            if serverLockdown {
                let cur = try await LMSAPI.fetchQuizCurrentQuestion(
                    courseCode: courseCode,
                    itemId: itemId,
                    attemptId: attemptId,
                    accessToken: accessToken
                )
                serverQuestion = cur.question
                serverCompleted = cur.completed
                currentIndex = Int(cur.questionIndex)
                totalQuestions = Int(cur.totalQuestions)
                if let question = cur.question {
                    questions = [question]
                }
            } else {
                let payload: ModuleQuizPayload
                if let quiz {
                    payload = quiz
                } else {
                    payload = try await LMSAPI.fetchModuleQuiz(
                        courseCode: courseCode,
                        itemId: itemId,
                        attemptId: attemptId,
                        accessToken: accessToken
                    )
                }
                questions = payload.questions ?? []
                totalQuestions = questions.count
                if questions.isEmpty {
                    errorMessage = L.text("mobile.quiz.noQuestions")
                }
                currentIndex = min(max(0, currentIndex), max(0, questions.count - 1))
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.modules.loadError")
        }
        _ = start
    }

    func tickTimer() {
        guard let deadlineAt else { return }
        let remaining = QuizLogic.secondsRemaining(deadlineISO: deadlineAt) ?? 0
        timerSeconds = remaining
        if remaining <= 600 && remaining > 60 {
            timerNotice = L.text("mobile.quiz.tenMinutesLeft")
        } else if remaining <= 60 && remaining > 0 {
            timerNotice = L.text("mobile.quiz.oneMinuteLeft")
        } else if remaining == 0 {
            timerNotice = L.text("mobile.quiz.timeUp")
        }
    }

    func setAnswer(_ answer: QuizAnswerState, for questionId: String) {
        answers[questionId] = answer
        saveStates[questionId] = serverLockdown ? .idle : .saved
    }

    func goNext(accessToken: String, offline: OfflineService) async {
        if serverLockdown {
            guard let question = serverQuestion else { return }
            advancing = true
            saveStates[question.id] = .saving
            defer { advancing = false }
            let body = QuizLogic.buildResponseItem(question: question, answer: answers[question.id])
            do {
                if NetworkMonitor.shared.isOnline {
                    let res = try await LMSAPI.advanceQuiz(
                        courseCode: courseCode,
                        itemId: itemId,
                        attemptId: attemptId,
                        responseItem: body,
                        accessToken: accessToken
                    )
                    saveStates[question.id] = .saved
                    serverCompleted = res.completed
                    if res.completed {
                        serverQuestion = nil
                        return
                    }
                } else {
                    let path = "/api/v1/courses/\(LMSAPI.encodePath(courseCode))/quizzes/\(LMSAPI.encodePath(itemId))/attempts/\(LMSAPI.encodePath(attemptId))/advance"
                    _ = try await offline.enqueueMutation(
                        method: "POST",
                        path: path,
                        body: body,
                        label: L.text("mobile.quiz.saveAnswer"),
                        accessToken: accessToken,
                        preferQueue: true
                    )
                    saveStates[question.id] = .queued
                    errorMessage = L.text("mobile.quiz.notYetSaved")
                    return
                }
                await refreshServerQuestion(accessToken: accessToken)
            } catch {
                saveStates[question.id] = .failed
                errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.quiz.saveError")
            }
            return
        }
        if currentIndex < questions.count - 1 {
            currentIndex += 1
        }
    }

    func goPrevious() {
        guard allowBackNavigation, !serverLockdown else { return }
        if currentIndex > 0 { currentIndex -= 1 }
    }

    func jumpTo(index: Int) {
        guard !serverLockdown, index >= 0, index < questions.count else { return }
        currentIndex = index
    }

    private func refreshServerQuestion(accessToken: String) async {
        do {
            let cur = try await LMSAPI.fetchQuizCurrentQuestion(
                courseCode: courseCode,
                itemId: itemId,
                attemptId: attemptId,
                accessToken: accessToken
            )
            serverCompleted = cur.completed
            serverQuestion = cur.question
            currentIndex = Int(cur.questionIndex)
            totalQuestions = Int(cur.totalQuestions)
            if let question = cur.question {
                questions = [question]
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.quiz.saveError")
        }
    }

    func submit(accessToken: String) async {
        guard !submitGuard else { return }
        submitGuard = true
        submitting = true
        errorMessage = nil
        defer {
            submitting = false
            submitGuard = false
        }
        do {
            let responses: [QuizQuestionResponseItem]?
            if serverLockdown {
                responses = nil
            } else {
                responses = questions.map { QuizLogic.buildResponseItem(question: $0, answer: answers[$0.id]) }
            }
            _ = try await LMSAPI.submitQuiz(
                courseCode: courseCode,
                itemId: itemId,
                attemptId: attemptId,
                responses: responses,
                accessToken: accessToken
            )
            results = try await LMSAPI.fetchQuizResults(
                courseCode: courseCode,
                itemId: itemId,
                attemptId: attemptId,
                accessToken: accessToken
            )
            showResults = true
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.quiz.submitError")
            submitGuard = false
        }
    }

    func reportFocusLoss(accessToken: String, eventType: String) {
        Task {
            await LMSAPI.postQuizFocusLoss(
                courseCode: courseCode,
                itemId: itemId,
                attemptId: attemptId,
                eventType: eventType,
                accessToken: accessToken
            )
        }
    }
}

/// Active quiz-taking flow: one question at a time, autosave, timer (M4.1).
struct QuizTakerView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.scenePhase) private var scenePhase

    let course: CourseSummary
    let item: CourseStructureItem
    let quiz: ModuleQuizPayload?
    let detail: ModuleItemDetail?
    let start: QuizStartResponse
    var onFinished: () -> Void

    @State private var model: QuizTakerModel
    @State private var timerTask: Task<Void, Never>?

    init(
        course: CourseSummary,
        item: CourseStructureItem,
        quiz: ModuleQuizPayload?,
        detail: ModuleItemDetail?,
        start: QuizStartResponse,
        onFinished: @escaping () -> Void
    ) {
        self.course = course
        self.item = item
        self.quiz = quiz
        self.detail = detail
        self.start = start
        self.onFinished = onFinished
        _model = State(initialValue: QuizTakerModel(
            courseCode: course.courseCode,
            itemId: item.id,
            start: start
        ))
    }

    var body: some View {
        Group {
            if model.showResults, let results = model.results {
                QuizResultsView(
                    title: item.title,
                    results: results,
                    onDone: onFinished
                )
            } else {
                takerBody
            }
        }
        .navigationTitle(item.title)
        .navigationBarTitleDisplayMode(.inline)
        .task {
            guard let token = session.accessToken else { return }
            await model.load(quiz: quiz, start: start, accessToken: token)
            startTimerLoop()
        }
        .onDisappear { timerTask?.cancel() }
        .onChange(of: scenePhase) { _, phase in
            if phase == .background || phase == .inactive {
                model.reportFocusLoss(accessToken: session.accessToken ?? "", eventType: "app_background")
            }
        }
        .confirmationDialog(
            L.text("mobile.quiz.submitConfirmTitle"),
            isPresented: $model.showSubmitConfirm,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.quiz.submit")) {
                Task {
                    guard let token = session.accessToken else { return }
                    await model.submit(accessToken: token)
                }
            }
            Button(L.text("mobile.quiz.cancel"), role: .cancel) {}
        } message: {
            Text(L.format(
                "mobile.quiz.submitConfirmMessage",
                model.answeredCount,
                model.questions.count - model.answeredCount
            ))
        }
    }

    private var takerBody: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            VStack(spacing: 0) {
                timerBar
                if let error = model.errorMessage {
                    LMSErrorBanner(message: error)
                        .padding(.horizontal, 16)
                        .padding(.top, 8)
                }
                if model.loading {
                    Spacer()
                    ProgressView()
                    Spacer()
                } else if model.serverCompleted && model.currentQuestion == nil {
                    submitReadyView
                } else if let question = model.currentQuestion {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 14) {
                            progressHeader
                            if !model.serverLockdown {
                                QuizPaletteView(
                                    questions: model.questions,
                                    currentIndex: model.currentIndex,
                                    answers: model.answers,
                                    flaggedIds: model.flaggedIds,
                                    onSelect: { model.jumpTo(index: $0) }
                                )
                            }
                            QuizQuestionView(
                                question: question,
                                answer: model.answers[question.id] ?? QuizAnswerState(),
                                saveState: model.saveStates[question.id] ?? .idle,
                                codeRunContext: session.accessToken.map { token in
                                    CodeQuestionRunContext(
                                        courseCode: course.courseCode,
                                        itemId: item.id,
                                        attemptId: model.attemptId,
                                        accessToken: token
                                    )
                                },
                                onChange: { newAnswer in
                                    model.setAnswer(newAnswer, for: question.id)
                                },
                                isFlagged: model.flaggedIds.contains(question.id),
                                onToggleFlag: {
                                    if model.flaggedIds.contains(question.id) {
                                        model.flaggedIds.remove(question.id)
                                    } else {
                                        model.flaggedIds.insert(question.id)
                                    }
                                }
                            )
                        }
                        .padding(16)
                    }
                    navigationBar
                } else {
                    submitReadyView
                }
            }
        }
    }

    private var timerBar: some View {
        HStack {
            if let seconds = model.timerSeconds {
                Label(QuizLogic.formatTimer(seconds), systemImage: "clock.fill")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(seconds <= 60 ? LexturesTheme.coral : LexturesTheme.textPrimary(for: colorScheme))
                    .accessibilityLabel(L.format("mobile.quiz.timeRemainingA11y", QuizLogic.formatTimer(seconds)))
            }
            Spacer()
            if let notice = model.timerNotice {
                Text(notice)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .background(LexturesTheme.cardBackground(for: colorScheme))
    }

    private var progressHeader: some View {
        Text(L.format("mobile.quiz.progress", model.currentIndex + 1, max(model.totalQuestions, max(model.questions.count, 1))))
            .font(.caption.weight(.semibold))
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
    }

    private var navigationBar: some View {
        HStack(spacing: 12) {
            if model.allowBackNavigation && !model.serverLockdown {
                Button(L.text("mobile.quiz.previous")) { model.goPrevious() }
                    .disabled(model.currentIndex == 0)
            }
            Spacer()
            if model.serverLockdown {
                if model.serverCompleted {
                    Button(L.text("mobile.quiz.submit")) { model.showSubmitConfirm = true }
                        .buttonStyle(AuthPrimaryButtonStyle())
                        .disabled(model.submitting)
                } else {
                    Button(L.text("mobile.quiz.next")) {
                        Task {
                            guard let token = session.accessToken else { return }
                            await model.goNext(accessToken: token, offline: offline)
                        }
                    }
                    .buttonStyle(AuthPrimaryButtonStyle())
                    .disabled(model.advancing)
                }
            } else if model.currentIndex >= model.questions.count - 1 {
                Button(L.text("mobile.quiz.submit")) { model.showSubmitConfirm = true }
                    .buttonStyle(AuthPrimaryButtonStyle())
                    .disabled(model.submitting)
            } else {
                Button(L.text("mobile.quiz.next")) {
                    Task {
                        guard let token = session.accessToken else { return }
                        await model.goNext(accessToken: token, offline: offline)
                    }
                }
                .buttonStyle(AuthPrimaryButtonStyle())
                .disabled(model.advancing)
            }
        }
        .padding(16)
        .background(LexturesTheme.cardBackground(for: colorScheme))
    }

    private var submitReadyView: some View {
        VStack(spacing: 16) {
            Spacer()
            Text(L.text("mobile.quiz.readyToSubmit"))
                .font(LexturesTheme.displayFont(20))
            Text(L.format("mobile.quiz.answeredSummary", model.answeredCount, model.questions.count))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Button(L.text("mobile.quiz.submit")) { model.showSubmitConfirm = true }
                .buttonStyle(AuthPrimaryButtonStyle())
                .disabled(model.submitting)
            Spacer()
        }
        .padding(24)
    }

    private func startTimerLoop() {
        timerTask?.cancel()
        timerTask = Task {
            while !Task.isCancelled {
                await MainActor.run {
                    model.tickTimer()
                    if model.timerSeconds == 0 {
                        Task {
                            guard let token = session.accessToken else { return }
                            await model.submit(accessToken: token)
                        }
                    }
                }
                try? await Task.sleep(for: .seconds(1))
            }
        }
    }
}

struct QuizPaletteView: View {
    @Environment(\.colorScheme) private var colorScheme
    let questions: [QuizQuestion]
    let currentIndex: Int
    let answers: [String: QuizAnswerState]
    let flaggedIds: Set<String>
    let onSelect: (Int) -> Void

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(Array(questions.enumerated()), id: \.element.id) { index, question in
                    let answered = QuizLogic.isAnswered(question: question, answer: answers[question.id])
                    let flagged = flaggedIds.contains(question.id)
                    Button {
                        onSelect(index)
                    } label: {
                        Text("\(index + 1)")
                            .font(.caption.weight(.bold))
                            .frame(width: 36, height: 36)
                            .background(paletteColor(index: index, answered: answered, flagged: flagged))
                            .foregroundStyle(index == currentIndex ? .white : LexturesTheme.textPrimary(for: colorScheme))
                            .clipShape(Circle())
                    }
                    .accessibilityLabel(paletteA11y(index: index, answered: answered, flagged: flagged))
                }
            }
        }
    }

    private func paletteColor(index: Int, answered: Bool, flagged: Bool) -> Color {
        if index == currentIndex { return LexturesTheme.primary }
        if flagged { return LexturesTheme.amber.opacity(0.35) }
        if answered { return LexturesTheme.primary.opacity(0.2) }
        return LexturesTheme.textSecondary(for: colorScheme).opacity(0.15)
    }

    private func paletteA11y(index: Int, answered: Bool, flagged: Bool) -> String {
        var parts = [L.format("mobile.quiz.questionNumber", index + 1)]
        if answered { parts.append(L.text("mobile.quiz.answered")) }
        if flagged { parts.append(L.text("mobile.quiz.flagged")) }
        return parts.joined(separator: ", ")
    }
}
