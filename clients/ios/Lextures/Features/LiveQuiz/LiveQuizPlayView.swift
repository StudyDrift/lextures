import SwiftUI

/// Student play flow: code → nickname → lobby/play/results (MOB.5 Phase 1).
struct LiveQuizPlayView: View {
    @EnvironmentObject private var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    var initialCode: String?

    @State private var step: LiveQuizLogic.JoinStep = .code
    @State private var code = ""
    @State private var nickname = ""
    @State private var lookup: LiveQuizJoinLookup?
    @State private var playerSession: LiveQuizPlayerSession?
    @State private var busy = false
    @State private var errorMessage: String?

    @State private var gameState: LiveGameStateFrame?
    @State private var conn: LiveGameLogic.ConnStatus = .connecting
    @State private var lastAck: LiveGameAnswerAck?
    @State private var answeredIndex: Int?
    @State private var selectedOptionId: String?
    @State private var selectedOptionIds: Set<String> = []
    @State private var answerText = ""
    @State private var answerNumeric = ""
    @State private var orderIds: [String] = []
    @State private var myResults: LiveQuizMyResults?
    @State private var socket: LiveGameSocket?

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                content
                    .padding(16)
            }
            .navigationTitle(L.text("mobile.liveQuiz.join.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) {
                        teardown()
                        dismiss()
                    }
                }
            }
            .onAppear {
                if let initialCode, !initialCode.isEmpty {
                    code = LiveQuizLogic.normalizeJoinCode(initialCode)
                    Task { await lookupCode() }
                }
            }
            .onDisappear { teardown() }
        }
    }

    @ViewBuilder
    private var content: some View {
        switch step {
        case .code:
            codeStep
        case .nickname:
            nicknameStep
        case .play:
            playStep
        }
    }

    private var codeStep: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text(L.text("mobile.liveQuiz.join.codePrompt"))
                .font(.title3.weight(.semibold))
            if let errorMessage {
                Text(errorMessage).font(.caption).foregroundStyle(.red)
            }
            TextField(L.text("mobile.liveQuiz.join.code"), text: $code)
                .textInputAutocapitalization(.characters)
                .autocorrectionDisabled()
                .textFieldStyle(.roundedBorder)
                .accessibilityIdentifier("liveQuiz.join.code")
            Button(L.text("mobile.liveQuiz.join.continue")) {
                Task { await lookupCode() }
            }
            .buttonStyle(.borderedProminent)
            .disabled(busy || !LiveQuizLogic.isValidJoinCode(code))
            Spacer()
        }
    }

    private var nicknameStep: some View {
        VStack(alignment: .leading, spacing: 16) {
            if let lookup {
                Text(lookup.kitTitle)
                    .font(.headline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            }
            Text(L.text("mobile.liveQuiz.join.nicknamePrompt"))
                .font(.title3.weight(.semibold))
            if let errorMessage {
                Text(errorMessage).font(.caption).foregroundStyle(.red)
            }
            TextField(L.text("mobile.liveQuiz.join.nickname"), text: $nickname)
                .textFieldStyle(.roundedBorder)
                .accessibilityIdentifier("liveQuiz.join.nickname")
            Button(L.text("mobile.liveQuiz.join.submit")) {
                Task { await joinGame() }
            }
            .buttonStyle(.borderedProminent)
            .disabled(busy)
            Button(L.text("mobile.common.cancel")) {
                step = .code
                errorMessage = nil
            }
            .buttonStyle(.borderless)
            Spacer()
        }
    }

    @ViewBuilder
    private var playStep: some View {
        let phase = LiveGameLogic.Phase.parse(gameState?.phase)
        let surface = LiveGameLogic.playSurface(for: phase, conn: conn)
        VStack(alignment: .leading, spacing: 12) {
            LiveQuizConnectionBadge(conn: conn)
            switch surface {
            case .connecting:
                ProgressView(L.text("mobile.liveQuiz.play.connecting"))
            case .lobby, .waitingForHost:
                LiveQuizLobbyView(
                    kitTitle: gameState?.kitTitle ?? lookup?.kitTitle ?? "",
                    playerCount: gameState?.players.count
                )
            case .question:
                if let question = gameState?.question {
                    LiveQuizQuestionView(
                        question: question,
                        phase: phase,
                        conn: conn,
                        hasAnswered: answeredIndex == gameState?.questionIndex,
                        deadline: gameState?.deadline,
                        lastAck: lastAck,
                        questionIndex: gameState?.questionIndex,
                        selectedOptionId: $selectedOptionId,
                        selectedOptionIds: $selectedOptionIds,
                        answerText: $answerText,
                        answerNumeric: $answerNumeric,
                        orderIds: $orderIds,
                        onSubmit: submitCurrentAnswer
                    )
                }
            case .leaderboard, .podium:
                LiveQuizLeaderboardView(gameState: gameState)
            case .ended:
                LiveQuizEndedView(
                    myResults: myResults,
                    gameState: gameState,
                    onLoadResults: { Task { await loadResults() } }
                )
            case .kicked:
                Text(L.text("mobile.liveQuiz.play.kicked"))
                    .foregroundStyle(.red)
            }
            Spacer(minLength: 0)
        }
    }

    private func lookupCode() async {
        busy = true
        errorMessage = nil
        defer { busy = false }
        do {
            let result = try await LMSAPI.lookupJoinCode(code)
            lookup = result
            if let existing = LiveQuizPlayerSessionStore.load(gameId: result.gameId) {
                playerSession = existing
                step = .play
                startSocket(session: existing)
                return
            }
            step = .nickname
            LiveQuizObservability.record("live_quiz_join", attributes: ["step": "lookup"])
        } catch let err as LMSAPI.LiveQuizJoinError {
            errorMessage = err.errorDescription
        } catch {
            errorMessage = L.text("mobile.liveQuiz.error.generic")
        }
    }

    private func joinGame() async {
        busy = true
        errorMessage = nil
        defer { busy = false }
        switch LiveQuizLogic.validateNickname(nickname) {
        case .invalid(let reason):
            errorMessage = L.text(String.LocalizationValue(LiveQuizLogic.nicknameReasonLocalizationKey(reason)))
            return
        case .ok(let nick):
            guard let lookup else { return }
            do {
                let joined: LiveQuizJoinPlayerResult
                if lookup.allowsGuests && session.accessToken == nil {
                    joined = try await LMSAPI.joinLiveGameAsGuest(code: code, nickname: nick)
                } else if let token = session.accessToken {
                    joined = try await LMSAPI.joinLiveGame(
                        courseCode: lookup.courseCode,
                        gameId: lookup.gameId,
                        nickname: nick,
                        accessToken: token
                    )
                } else if lookup.allowsGuests {
                    joined = try await LMSAPI.joinLiveGameAsGuest(code: code, nickname: nick)
                } else {
                    errorMessage = L.text("mobile.liveQuiz.error.authRequired")
                    return
                }
                let sess = LiveQuizPlayerSession(
                    gameId: lookup.gameId,
                    courseCode: lookup.courseCode,
                    playerId: joined.playerId,
                    playerToken: joined.playerToken,
                    nickname: joined.nickname,
                    joinCode: LiveQuizLogic.normalizeJoinCode(code)
                )
                LiveQuizPlayerSessionStore.save(sess)
                playerSession = sess
                step = .play
                LiveQuizObservability.record(
                    "live_quiz_join",
                    attributes: ["rejoined": joined.rejoined == true ? "1" : "0"]
                )
                startSocket(session: sess)
            } catch let err as LMSAPI.LiveQuizJoinError {
                errorMessage = err.errorDescription
            } catch {
                errorMessage = L.text("mobile.liveQuiz.error.generic")
            }
        }
    }

    private func startSocket(session: LiveQuizPlayerSession) {
        socket?.disconnect()
        let gameSocket = LiveGameSocket(
            courseCode: session.courseCode,
            gameId: session.gameId,
            role: .player,
            playerToken: session.playerToken,
            accessTokenProvider: { self.session.accessToken },
            onState: { frame in
                let prevIndex = gameState?.questionIndex
                let nextPhase = LiveGameLogic.Phase.parse(frame.phase) ?? .lobby
                if LiveGameLogic.shouldClearAnsweredIndex(
                    previousQuestionIndex: prevIndex,
                    nextQuestionIndex: frame.questionIndex,
                    nextPhase: nextPhase
                ) {
                    answeredIndex = nil
                    lastAck = nil
                    selectedOptionId = nil
                    selectedOptionIds = []
                    answerText = ""
                    answerNumeric = ""
                    orderIds = frame.question?.options.map(\.id) ?? []
                }
                gameState = frame
                if frame.phase == LiveGameLogic.Phase.ended.rawValue {
                    Task { await loadResults() }
                }
            },
            onAck: { ack in
                lastAck = ack
                if ack.ok, let idx = ack.questionIndex {
                    answeredIndex = idx
                }
                LiveQuizObservability.record(
                    "live_quiz_answer",
                    attributes: [
                        "ok": ack.ok ? "1" : "0",
                        "type": gameState?.question?.questionType ?? "",
                    ]
                )
            },
            onKicked: {
                if let gameId = playerSession?.gameId {
                    LiveQuizPlayerSessionStore.clear(gameId: gameId)
                }
            },
            onConn: { conn = $0 }
        )
        socket = gameSocket
        gameSocket.connect()
    }

    private func submitCurrentAnswer(type: LiveGameLogic.QuestionType, question: LiveGameQuestion) {
        guard let index = gameState?.questionIndex else { return }
        let numeric = Double(answerNumeric.trimmingCharacters(in: .whitespacesAndNewlines))
        guard let payload = LiveGameLogic.buildAnswer(
            questionType: type,
            selectedOptionId: selectedOptionId,
            selectedOptionIds: Array(selectedOptionIds),
            text: answerText,
            numeric: numeric,
            order: orderIds.isEmpty ? question.options.map(\.id) : orderIds
        ) else { return }
        answeredIndex = index
        socket?.submitAnswer(questionIndex: index, answer: payload)
    }

    private func loadResults() async {
        guard let playerSession, let token = session.accessToken else { return }
        do {
            myResults = try await LMSAPI.fetchMyGameResults(
                courseCode: playerSession.courseCode,
                gameId: playerSession.gameId,
                accessToken: token
            )
            LiveQuizObservability.record("live_quiz_end", attributes: [:])
        } catch {
            // Results require auth; guests see leaderboard only.
        }
    }

    private func teardown() {
        socket?.disconnect()
        socket = nil
    }
}
