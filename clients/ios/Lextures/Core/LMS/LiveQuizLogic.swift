import Foundation

/// Pure helpers for interactive live quizzes (MOB.5 Phase 1 — student play).
enum LiveQuizLogic {
    static let nicknameMaxLength = 24
    private static let nicknameAllowed = try! NSRegularExpression(
        pattern: #"^[\p{L}\p{N} _.\-'!]+$"#
    )

    enum JoinStep: String, Equatable {
        case code
        case nickname
        case play
    }

    enum NicknameReason: String, Equatable {
        case empty
        case tooLong
        case charset
    }

    enum NicknameValidation: Equatable {
        case ok(String)
        case invalid(NicknameReason)
    }

    enum JoinErrorReason: String, Equatable {
        case notFound
        case rateLimited
        case nicknameTaken
        case nicknameInvalid
        case nicknameDenied
        case lobbyLocked
        case banned
        case oneSession
        case gameEnded
        case authRequired
        case joinFailed
        case unknown
    }

    static func normalizeJoinCode(_ raw: String) -> String {
        raw.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
    }

    static func isValidJoinCode(_ raw: String) -> Bool {
        let code = normalizeJoinCode(raw)
        guard !code.isEmpty, code.count <= 12 else { return false }
        return code.allSatisfy { $0.isLetter || $0.isNumber }
    }

    static func normalizeNickname(_ raw: String) -> String {
        raw.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    static func validateNickname(_ raw: String) -> NicknameValidation {
        let nickname = normalizeNickname(raw)
        if nickname.isEmpty { return .invalid(.empty) }
        if nickname.count > nicknameMaxLength { return .invalid(.tooLong) }
        let range = NSRange(nickname.startIndex..<nickname.endIndex, in: nickname)
        guard nicknameAllowed.firstMatch(in: nickname, options: [], range: range) != nil else {
            return .invalid(.charset)
        }
        return .ok(nickname)
    }

    /// Dual gate: per-course interactive quizzes + mobile rollout kill-switch.
    static func shouldShowWorkspaceSection(
        course: CourseSummary,
        features: MobilePlatformFeatures
    ) -> Bool {
        course.isInteractiveQuizzesEnabled && features.ffMobileLiveQuiz
    }

    static func liveQuizEntryEnabled(
        courseEnabled: Bool,
        features: MobilePlatformFeatures
    ) -> Bool {
        courseEnabled && features.ffMobileLiveQuiz
    }

    static func joinErrorReason(status: Int, body: String) -> JoinErrorReason {
        let lower = body.lowercased()
        switch status {
        case 404:
            return .notFound
        case 429:
            return .rateLimited
        case 409:
            if lower.contains("already connected") { return .oneSession }
            return .nicknameTaken
        case 401, 403:
            if lower.contains("lobby") { return .lobbyLocked }
            if lower.contains("rejoin") || lower.contains("cannot") { return .banned }
            return .authRequired
        case 400:
            if lower.contains("ended") { return .gameEnded }
            if lower.contains("isn’t allowed") || lower.contains("isn't allowed") || lower.contains("not allowed") {
                return .nicknameDenied
            }
            if lower.contains("nickname") { return .nicknameInvalid }
            return .joinFailed
        default:
            return .unknown
        }
    }

    static func joinErrorLocalizationKey(_ reason: JoinErrorReason) -> String {
        switch reason {
        case .notFound: return "mobile.liveQuiz.error.notFound"
        case .rateLimited: return "mobile.liveQuiz.error.rateLimited"
        case .nicknameTaken: return "mobile.liveQuiz.error.nicknameTaken"
        case .nicknameInvalid: return "mobile.liveQuiz.error.nicknameInvalid"
        case .nicknameDenied: return "mobile.liveQuiz.error.nicknameDenied"
        case .lobbyLocked: return "mobile.liveQuiz.error.lobbyLocked"
        case .banned: return "mobile.liveQuiz.error.banned"
        case .oneSession: return "mobile.liveQuiz.error.oneSession"
        case .gameEnded: return "mobile.liveQuiz.error.gameEnded"
        case .authRequired: return "mobile.liveQuiz.error.authRequired"
        case .joinFailed: return "mobile.liveQuiz.error.joinFailed"
        case .unknown: return "mobile.liveQuiz.error.generic"
        }
    }

    static func nicknameReasonLocalizationKey(_ reason: NicknameReason) -> String {
        switch reason {
        case .empty: return "mobile.liveQuiz.nickname.error.empty"
        case .tooLong: return "mobile.liveQuiz.nickname.error.tooLong"
        case .charset: return "mobile.liveQuiz.nickname.error.charset"
        }
    }

    static func webSocketPath(courseCode: String, gameId: String) -> String {
        "/api/v1/courses/\(encodePath(courseCode))/live-quizzes/games/\(encodePath(gameId))/ws"
    }

    private static func encodePath(_ value: String) -> String {
        value.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? value
    }
}

/// Player session persisted for reconnect/resume (mirrors web `live-quiz-player-storage`).
struct LiveQuizPlayerSession: Codable, Equatable {
    var gameId: String
    var courseCode: String
    var playerId: String
    var playerToken: String
    var nickname: String
    var joinCode: String
}

enum LiveQuizPlayerSessionStore {
    private static let prefix = "liveQuiz.playerSession."

    static func save(_ session: LiveQuizPlayerSession) {
        guard let data = try? JSONEncoder().encode(session) else { return }
        UserDefaults.standard.set(data, forKey: prefix + session.gameId)
    }

    static func load(gameId: String) -> LiveQuizPlayerSession? {
        guard let data = UserDefaults.standard.data(forKey: prefix + gameId) else { return nil }
        return try? JSONDecoder().decode(LiveQuizPlayerSession.self, from: data)
    }

    static func clear(gameId: String) {
        UserDefaults.standard.removeObject(forKey: prefix + gameId)
    }
}

enum LiveQuizObservability {
    private static var counters: [String: Int] = [:]
    private static let lock = NSLock()

    static func record(_ event: String, attributes: [String: String] = [:]) {
        lock.lock()
        defer { lock.unlock() }
        let key = attributes.isEmpty
            ? event
            : event + "|" + attributes.keys.sorted().map { "\($0)=\(attributes[$0] ?? "")" }.joined(separator: ",")
        counters[key, default: 0] += 1
    }

    static func count(for event: String) -> Int {
        lock.lock()
        defer { lock.unlock() }
        return counters.filter { $0.key == event || $0.key.hasPrefix(event + "|") }.values.reduce(0, +)
    }

    #if DEBUG
    static func resetForTests() {
        lock.lock()
        counters.removeAll()
        lock.unlock()
    }
    #endif
}
