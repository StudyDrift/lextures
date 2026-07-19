import SwiftUI

/// Play-phase UI extracted from `LiveQuizPlayView` (SwiftLint type_body_length).
struct LiveQuizConnectionBadge: View {
    @Environment(\.colorScheme) private var colorScheme
    let conn: LiveGameLogic.ConnStatus

    var body: some View {
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
}

struct LiveQuizLobbyView: View {
    @Environment(\.colorScheme) private var colorScheme
    let kitTitle: String
    let playerCount: Int?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(kitTitle)
                .font(.title3.weight(.semibold))
            Text(L.text("mobile.liveQuiz.play.lobbyWaiting"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            if let playerCount {
                Text(L.format("mobile.liveQuiz.play.playerCount", playerCount))
                    .font(.caption)
            }
        }
    }
}

struct LiveQuizQuestionView: View {
    @Environment(\.colorScheme) private var colorScheme
    let question: LiveGameQuestion
    let phase: LiveGameLogic.Phase?
    let conn: LiveGameLogic.ConnStatus
    let hasAnswered: Bool
    let deadline: String?
    let lastAck: LiveGameAnswerAck?
    let questionIndex: Int?
    @Binding var selectedOptionId: String?
    @Binding var selectedOptionIds: Set<String>
    @Binding var answerText: String
    @Binding var answerNumeric: String
    @Binding var orderIds: [String]
    let onSubmit: (LiveGameLogic.QuestionType, LiveGameQuestion) -> Void

    var body: some View {
        let questionType = LiveGameLogic.QuestionType.parse(question.questionType) ?? .mcSingle
        VStack(alignment: .leading, spacing: 12) {
            Text(question.prompt)
                .font(.headline)
                .accessibilityAddTraits(.isHeader)
            if let deadline {
                Text(L.format("mobile.liveQuiz.play.deadline", deadline))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            LiveQuizAnswerSurface(
                type: questionType,
                question: question,
                hasAnswered: hasAnswered,
                selectedOptionId: $selectedOptionId,
                selectedOptionIds: $selectedOptionIds,
                answerText: $answerText,
                answerNumeric: $answerNumeric,
                orderIds: $orderIds
            )
            if let lastAck, lastAck.questionIndex == questionIndex {
                LiveQuizResultCard(ack: lastAck)
            }
            if LiveGameLogic.canSubmitAnswer(phase: phase, hasAnswered: hasAnswered, conn: conn) {
                Button(L.text("mobile.liveQuiz.play.submit")) {
                    onSubmit(questionType, question)
                }
                .buttonStyle(.borderedProminent)
                .accessibilityIdentifier("liveQuiz.play.submit")
            }
        }
    }
}

struct LiveQuizAnswerSurface: View {
    @Environment(\.colorScheme) private var colorScheme

    let type: LiveGameLogic.QuestionType
    let question: LiveGameQuestion
    let hasAnswered: Bool
    @Binding var selectedOptionId: String?
    @Binding var selectedOptionIds: Set<String>
    @Binding var answerText: String
    @Binding var answerNumeric: String
    @Binding var orderIds: [String]

    var body: some View {
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
            ForEach(orderIds.isEmpty ? question.options.map(\.id) : orderIds, id: \.self) { optionId in
                let text = question.options.first(where: { $0.id == optionId })?.text ?? optionId
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
}

struct LiveQuizResultCard: View {
    @Environment(\.colorScheme) private var colorScheme
    let ack: LiveGameAnswerAck

    var body: some View {
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
}

struct LiveQuizLeaderboardView: View {
    let gameState: LiveGameStateFrame?

    var body: some View {
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
}

struct LiveQuizEndedView: View {
    let myResults: LiveQuizMyResults?
    let gameState: LiveGameStateFrame?
    let onLoadResults: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(L.text("mobile.liveQuiz.play.ended"))
                .font(.title3.weight(.semibold))
            if let myResults {
                Text(L.format(
                    "mobile.liveQuiz.results.summary",
                    myResults.rank,
                    myResults.totalScore,
                    myResults.correct,
                    myResults.answered
                ))
                .font(.subheadline)
            } else {
                Button(L.text("mobile.liveQuiz.results.load"), action: onLoadResults)
                    .buttonStyle(.borderedProminent)
            }
            LiveQuizLeaderboardView(gameState: gameState)
        }
    }
}
