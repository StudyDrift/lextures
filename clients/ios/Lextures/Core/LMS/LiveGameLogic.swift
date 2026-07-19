import Foundation

/// Live-game state machine helpers (MOB.5). Shape mirrors web `live-quiz-realtime.ts`.
enum LiveGameLogic {
    enum Role: String, Equatable {
        case host
        case projector
        case player
    }

    enum Phase: String, Equatable, CaseIterable {
        case lobby
        case questionIntro = "question_intro"
        case questionOpen = "question_open"
        case questionLocked = "question_locked"
        case questionReveal = "question_reveal"
        case leaderboard
        case podium
        case ended
        case waitingForHost = "waiting_for_host"

        static func parse(_ raw: String?) -> Phase? {
            guard let raw else { return nil }
            return Phase(rawValue: raw)
        }
    }

    enum ConnStatus: String, Equatable {
        case connecting
        case connected
        case reconnecting
        case ended
        case kicked
        case disconnected
    }

    enum QuestionType: String, Equatable, CaseIterable {
        case mcSingle = "mc_single"
        case mcMultiple = "mc_multiple"
        case trueFalse = "true_false"
        case typeAnswer = "type_answer"
        case numeric
        case poll
        case ordering
        case wordCloud = "word_cloud"

        static func parse(_ raw: String?) -> QuestionType? {
            guard let raw else { return nil }
            return QuestionType(rawValue: raw)
        }
    }

    enum PointsStyle: String, Equatable {
        case standard
        case double
        case noPoints = "no_points"

        static func parse(_ raw: String?) -> PointsStyle {
            PointsStyle(rawValue: raw ?? "") ?? .standard
        }
    }

    enum PlaySurface: Equatable {
        case lobby
        case waitingForHost
        case question
        case leaderboard
        case podium
        case ended
        case kicked
        case connecting
    }

    /// Answer payload union — serialized to JSON matching the server protocol.
    enum AnswerPayload: Equatable {
        case optionId(String)
        case optionIds([String])
        case text(String)
        case value(Double)
        case order([String])

        func jsonObject() -> [String: Any] {
            switch self {
            case .optionId(let id):
                return ["optionId": id]
            case .optionIds(let ids):
                return ["optionIds": ids]
            case .text(let text):
                return ["text": text]
            case .value(let value):
                return ["value": value]
            case .order(let order):
                return ["order": order]
            }
        }

        func jsonData() throws -> Data {
            try JSONSerialization.data(withJSONObject: jsonObject(), options: [])
        }
    }

    static func playSurface(for phase: Phase?, conn: ConnStatus) -> PlaySurface {
        if conn == .kicked { return .kicked }
        if conn == .connecting || conn == .reconnecting { return .connecting }
        guard let phase else { return .connecting }
        switch phase {
        case .lobby, .questionIntro:
            return .lobby
        case .waitingForHost:
            return .waitingForHost
        case .questionOpen, .questionLocked, .questionReveal:
            return .question
        case .leaderboard:
            return .leaderboard
        case .podium:
            return .podium
        case .ended:
            return .ended
        }
    }

    static func canSubmitAnswer(phase: Phase?, hasAnswered: Bool, conn: ConnStatus) -> Bool {
        guard phase == .questionOpen else { return false }
        guard !hasAnswered else { return false }
        return conn == .connected
    }

    static func buildAnswer(
        questionType: QuestionType,
        selectedOptionId: String?,
        selectedOptionIds: [String],
        text: String?,
        numeric: Double?,
        order: [String]?
    ) -> AnswerPayload? {
        switch questionType {
        case .mcSingle, .trueFalse:
            guard let selectedOptionId, !selectedOptionId.isEmpty else { return nil }
            return .optionId(selectedOptionId)
        case .mcMultiple, .poll:
            guard !selectedOptionIds.isEmpty else { return nil }
            return .optionIds(selectedOptionIds)
        case .typeAnswer, .wordCloud:
            let trimmed = (text ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmed.isEmpty else { return nil }
            return .text(trimmed)
        case .numeric:
            guard let numeric else { return nil }
            return .value(numeric)
        case .ordering:
            guard let order, !order.isEmpty else { return nil }
            return .order(order)
        }
    }

    static func answerMessage(
        questionIndex: Int,
        answer: AnswerPayload,
        clientSentAt: String,
        powerUp: String? = nil
    ) -> [String: Any] {
        var msg: [String: Any] = [
            "type": "answer",
            "questionIndex": questionIndex,
            "answer": answer.jsonObject(),
            "clientSentAt": clientSentAt,
        ]
        if let powerUp { msg["powerUp"] = powerUp }
        return msg
    }

    static func authHandshake(
        authToken: String?,
        role: Role,
        playerToken: String?
    ) -> [String: Any] {
        var msg: [String: Any] = [
            "authToken": authToken ?? "",
            "role": role.rawValue,
        ]
        if let playerToken, !playerToken.isEmpty {
            msg["playerToken"] = playerToken
        }
        return msg
    }

    static func helloMessage(resumeSeq: Int = 0) -> [String: Any] {
        ["type": "hello", "resumeSeq": resumeSeq]
    }

    static func catchupMessage(afterSeq: Int) -> [String: Any] {
        ["type": "catchup", "afterSeq": afterSeq]
    }

    static func reconnectDelayMs(retry: Int) -> Int {
        min(8000, 500 * (1 << min(retry, 4)))
    }

    static func shouldClearAnsweredIndex(
        previousQuestionIndex: Int?,
        nextQuestionIndex: Int,
        nextPhase: Phase
    ) -> Bool {
        guard let previousQuestionIndex else { return false }
        return previousQuestionIndex != nextQuestionIndex && nextPhase == .questionOpen
    }

    /// Color-blind-safe shape labels for answer tiles (parity with web `answer-shape-meta`).
    static func answerShapeLabel(index: Int) -> String {
        let shapes = ["▲", "◆", "●", "■", "★", "✚", "⬡", "▼"]
        guard index >= 0 else { return shapes[0] }
        return shapes[index % shapes.count]
    }

    static func answerShapeName(index: Int) -> String {
        let names = ["triangle", "diamond", "circle", "square", "star", "cross", "hexagon", "invertedTriangle"]
        guard index >= 0 else { return names[0] }
        return names[index % names.count]
    }
}

// MARK: - Wire models

struct LiveQuizJoinLookup: Codable, Equatable {
    var gameId: String
    var courseCode: String
    var kitTitle: String
    var requiresAuth: Bool
    var allowsGuests: Bool
    var phase: String
    var status: String

    enum CodingKeys: String, CodingKey {
        case gameId, courseCode, kitTitle, requiresAuth, allowsGuests, phase, status
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        gameId = try c.decode(String.self, forKey: .gameId)
        courseCode = try c.decodeIfPresent(String.self, forKey: .courseCode) ?? ""
        kitTitle = try c.decodeIfPresent(String.self, forKey: .kitTitle) ?? ""
        requiresAuth = try c.decodeIfPresent(Bool.self, forKey: .requiresAuth) ?? true
        allowsGuests = try c.decodeIfPresent(Bool.self, forKey: .allowsGuests) ?? false
        phase = try c.decodeIfPresent(String.self, forKey: .phase) ?? ""
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
    }

    init(
        gameId: String,
        courseCode: String,
        kitTitle: String,
        requiresAuth: Bool,
        allowsGuests: Bool,
        phase: String,
        status: String
    ) {
        self.gameId = gameId
        self.courseCode = courseCode
        self.kitTitle = kitTitle
        self.requiresAuth = requiresAuth
        self.allowsGuests = allowsGuests
        self.phase = phase
        self.status = status
    }
}

struct LiveQuizJoinPlayerResult: Codable, Equatable {
    var playerId: String
    var nickname: String
    var playerToken: String
    var totalScore: Int
    var streak: Int?
    var rejoined: Bool?
}

struct LiveQuizMyResults: Codable, Equatable {
    var sessionId: String
    var nickname: String
    var totalScore: Int
    var rank: Int
    var playerCount: Int
    var answered: Int
    var correct: Int
}

struct LiveGameQuestionOption: Codable, Equatable {
    var id: String
    var text: String
}

struct LiveGameQuestion: Codable, Equatable {
    var index: Int
    var questionType: String
    var prompt: String
    var options: [LiveGameQuestionOption]
    var timeLimitSeconds: Int
    var pointsStyle: String
    var correctOptionIds: [String]?
    var explanation: String?
}

struct LiveGameLeaderboardEntry: Codable, Equatable {
    var rank: Int
    var playerId: String
    var nickname: String
    var totalScore: Int
}

struct LiveGameYou: Codable, Equatable {
    var rank: Int
    var totalScore: Int
    var streak: Int
}

struct LiveGamePlayer: Codable, Equatable {
    var id: String
    var nickname: String
    var totalScore: Int
    var streak: Int
    var connected: Bool
    var renamedByHost: Bool?
    var isGuest: Bool?
}

struct LiveGameStateFrame: Codable, Equatable {
    var type: String
    var seq: Int
    var gameId: String
    var phase: String
    var status: String
    var questionIndex: Int
    var joinCode: String
    var kitTitle: String
    var pacing: String
    var players: [LiveGamePlayer]
    var questionCount: Int
    var namesMuted: Bool?
    var lobbyLocked: Bool?
    var allowGuests: Bool?
    var deadline: String?
    var answerCount: Int?
    var leaderboard: [LiveGameLeaderboardEntry]?
    var podium: [LiveGameLeaderboardEntry]?
    var you: LiveGameYou?
    var scoringProfile: String?
    var powerUpsEnabled: Bool?
    var question: LiveGameQuestion?
}

struct LiveGameAnswerAck: Codable, Equatable {
    var type: String
    var ok: Bool
    var questionIndex: Int?
    var isCorrect: Bool?
    var points: Int?
    var streak: Int?
    var totalScore: Int?
    var rank: Int?
    var duplicate: Bool?
    var late: Bool?
    var alreadyAnswered: Bool?
    var error: String?
}

enum LiveGameInboundMessage: Equatable {
    case state(LiveGameStateFrame)
    case answerAck(LiveGameAnswerAck)
    case kicked
    case unknown
}

extension LiveGameLogic {
    static func parseInbound(_ data: Data) -> LiveGameInboundMessage {
        guard
            let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
            let type = obj["type"] as? String
        else { return .unknown }
        switch type {
        case "kicked":
            return .kicked
        case "answer_ack":
            guard let ack = try? JSONDecoder().decode(LiveGameAnswerAck.self, from: data) else {
                return .unknown
            }
            return .answerAck(ack)
        case "state":
            guard let frame = try? JSONDecoder().decode(LiveGameStateFrame.self, from: data) else {
                return .unknown
            }
            return .state(frame)
        default:
            return .unknown
        }
    }
}
