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
            connectionBadge
            switch surface {
            case .connecting:
                ProgressView(L.text("mobile.liveQuiz.play.connecting"))
            case .lobby, .waitingForHost:
                lobbyView
            case .question:
                questionView
            case .leaderboard, .podium:
                leaderboardView
            case .ended:
                endedView
            case .kicked:
                Text(L.text("mobile.liveQuiz.play.kicked"))
                    .foregroundStyle(.red)
            }
            Spacer(minLength: 0)
        }
    }

    private var connectionBadge: some View {
        let label: String = {
            switch conn {
            case .connecting: return L.text("mobile.liveQuiz.conn.connecting")
            case .connected: return L.text("mobile.liveQuiz.conn.connected")
            case .reconnecting: return L.text("mobile.liveQuiz.conn.reconnecting")
            case .ended: return L.text("mobile.liveQuiz.conn.ended")
            case .kicked: return L.text("mobile.liveQuiz.conn.kicked")
            case .disconnected: return L.text("mobile.liveQuiz.conn.disconnected")
            }
        }()
        return Text(label)
            .font(.caption.weight(.medium))
            .foregroundStyle(conn == .connected ? .green : LexturesTheme.textSecondary(for: colorScheme))
            .accessibilityLabel(label)
    }

    private var lobbyView: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(gameState?.kitTitle ?? lookup?.kitTitle ?? "")
                .font(.title3.weight(.semibold))
            Text(L.text("mobile.liveQuiz.play.lobbyWaiting"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            if let players = gameState?.players {
                Text(L.format("mobile.liveQuiz.play.playerCount", players.count))
                    .font(.caption)
            }
        }
    }

    @ViewBuilder
    private var questionView: some View {
        if let q = gameState?.question {
            let qType = LiveGameLogic.QuestionType.parse(q.questionType) ?? .mcSingle
            let hasAnswered = answeredIndex == gameState?.questionIndex
            VStack(alignment: .leading, spacing: 12) {
                Text(q.prompt)
                    .font(.headline)
                    .accessibilityAddTraits(.isHeader)
                if let deadline = gameState?.deadline {
                    Text(L.format("mobile.liveQuiz.play.deadline", deadline))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                answerSurface(type: qType, question: q, hasAnswered: hasAnswered)
                if let lastAck, lastAck.questionIndex == gameState?.questionIndex {
                    resultCard(lastAck)
                }
                if LiveGameLogic.canSubmitAnswer(
                    phase: LiveGameLogic.Phase.parse(gameState?.phase),
                    hasAnswered: hasAnswered,
                    conn: conn
                ) {
                    Button(L.text("mobile.liveQuiz.play.submit")) {
                        submitCurrentAnswer(type: qType, question: q)
                    }
                    .buttonStyle(.borderedProminent)
                    .accessibilityIdentifier("liveQuiz.play.submit")
                }
            }
        }
    }

    @ViewBuilder
    private func answerSurface(
        type: LiveGameLogic.QuestionType,
        question: LiveGameQuestion,
        hasAnswered: Bool
    ) -> some View {
        switch type {
        case .mcSingle, .trueFalse:
            ForEach(Array(question.options.enumerated()), id: \.element.id) { index, opt in
                Button {
                    selectedOptionId = opt.id
                } label: {
                    HStack {
                        Text(LiveGameLogic.answerShapeLabel(index: index))
                        Text(opt.text)
                        Spacer()
                        if selectedOptionId == opt.id {
                            Image(systemName: "checkmark.circle.fill")
                        }
                    }
                    .padding(12)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(
                        RoundedRectangle(cornerRadius: 10)
                            .fill(selectedOptionId == opt.id
                                  ? Color.accentColor.opacity(0.15)
                                  : LexturesTheme.cardBackground(for: colorScheme))
                    )
                }
                .buttonStyle(.plain)
                .disabled(hasAnswered)
                .accessibilityLabel("\(LiveGameLogic.answerShapeName(index: index)), \(opt.text)")
            }
        case .mcMultiple, .poll:
            ForEach(Array(question.options.enumerated()), id: \.element.id) { index, opt in
                Button {
                    if selectedOptionIds.contains(opt.id) {
                        selectedOptionIds.remove(opt.id)
                    } else {
                        selectedOptionIds.insert(opt.id)
                    }
                } label: {
                    HStack {
                        Text(LiveGameLogic.answerShapeLabel(index: index))
                        Text(opt.text)
                        Spacer()
                        Image(systemName: selectedOptionIds.contains(opt.id) ? "checkmark.square.fill" : "square")
                    }
                    .padding(12)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(
                        RoundedRectangle(cornerRadius: 10)
                            .fill(LexturesTheme.cardBackground(for: colorScheme))
                    )
                }
                .buttonStyle(.plain)
                .disabled(hasAnswered)
            }
        case .typeAnswer, .wordCloud:
            TextField(L.text("mobile.liveQuiz.play.typeAnswer"), text: $answerText)
                .textFieldStyle(.roundedBorder)
                .disabled(hasAnswered)
        case .numeric:
            TextField(L.text("mobile.liveQuiz.play.numeric"), text: $answerNumeric)
                .keyboardType(.decimalPad)
                .textFieldStyle(.roundedBorder)
                .disabled(hasAnswered)
        case .ordering:
            ForEach(orderIds.isEmpty ? question.options.map(\.id) : orderIds, id: \.self) { id in
                let text = question.options.first(where: { $0.id == id })?.text ?? id
                Text(text)
                    .padding(10)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(RoundedRectangle(cornerRadius: 8).fill(LexturesTheme.cardBackground(for: colorScheme)))
                    .onAppear {
                        if orderIds.isEmpty { orderIds = question.options.map(\.id) }
                    }
            }
        }
    }

    private func resultCard(_ ack: LiveGameAnswerAck) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            if ack.ok {
                Text(ack.isCorrect == true
                     ? L.text("mobile.liveQuiz.play.correct")
                     : L.text("mobile.liveQuiz.play.incorrect"))
                    .font(.subheadline.weight(.semibold))
                if let points = ack.points {
                    Text(L.format("mobile.liveQuiz.play.points", points))
                        .font(.caption)
                }
            } else if let err = ack.error {
                Text(err).font(.caption).foregroundStyle(.red)
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(RoundedRectangle(cornerRadius: 8).fill(LexturesTheme.cardBackground(for: colorScheme)))
    }

    private var leaderboardView: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.liveQuiz.play.leaderboard"))
                .font(.headline)
            if let you = gameState?.you {
                Text(L.format("mobile.liveQuiz.play.yourRank", you.rank, you.totalScore))
                    .font(.subheadline)
            }
            ForEach(gameState?.leaderboard ?? gameState?.podium ?? [], id: \.playerId) { entry in
                HStack {
                    Text("#\(entry.rank)")
                        .font(.caption.monospacedDigit())
                    Text(entry.nickname)
                    Spacer()
                    Text("\(entry.totalScore)")
                        .font(.caption.monospacedDigit())
                }
            }
        }
    }

    private var endedView: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(L.text("mobile.liveQuiz.play.ended"))
                .font(.title3.weight(.semibold))
            if let myResults {
                Text(L.format("mobile.liveQuiz.results.summary", myResults.rank, myResults.totalScore, myResults.correct, myResults.answered))
                    .font(.subheadline)
            } else {
                Button(L.text("mobile.liveQuiz.results.load")) {
                    Task { await loadResults() }
                }
                .buttonStyle(.borderedProminent)
            }
            leaderboardView
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
        let s = LiveGameSocket(
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
        socket = s
        s.connect()
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
