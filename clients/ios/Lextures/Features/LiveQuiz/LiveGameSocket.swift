import Foundation

/// Live-quiz WebSocket with role + playerToken handshake (IQ.3 / MOB.5).
/// Dedicated client — shared `WebSocketClient` only sends `{authToken}`.
@MainActor
final class LiveGameSocket {
    private let courseCode: String
    private let gameId: String
    private let role: LiveGameLogic.Role
    private let playerToken: String?
    private let accessTokenProvider: () -> String?
    private let onState: (LiveGameStateFrame) -> Void
    private let onAck: (LiveGameAnswerAck) -> Void
    private let onKicked: () -> Void
    private let onConn: (LiveGameLogic.ConnStatus) -> Void

    private var task: URLSessionWebSocketTask?
    private var isStopped = true
    private var reconnectTask: Task<Void, Never>?
    private var retry = 0
    private var seq = 0
    private var kicked = false

    init(
        courseCode: String,
        gameId: String,
        role: LiveGameLogic.Role = .player,
        playerToken: String?,
        accessTokenProvider: @escaping () -> String?,
        onState: @escaping (LiveGameStateFrame) -> Void,
        onAck: @escaping (LiveGameAnswerAck) -> Void,
        onKicked: @escaping () -> Void,
        onConn: @escaping (LiveGameLogic.ConnStatus) -> Void
    ) {
        self.courseCode = courseCode
        self.gameId = gameId
        self.role = role
        self.playerToken = playerToken
        self.accessTokenProvider = accessTokenProvider
        self.onState = onState
        self.onAck = onAck
        self.onKicked = onKicked
        self.onConn = onConn
    }

    func connect() {
        guard !kicked else { return }
        isStopped = false
        reconnectTask?.cancel()
        reconnectTask = nil
        let auth = accessTokenProvider()
        if (auth == nil || auth?.isEmpty == true) && (playerToken == nil || playerToken?.isEmpty == true) {
            scheduleReconnect()
            return
        }
        openConnection(authToken: auth)
    }

    func disconnect() {
        isStopped = true
        reconnectTask?.cancel()
        reconnectTask = nil
        task?.cancel(with: .normalClosure, reason: nil)
        task = nil
    }

    func send(_ object: [String: Any]) {
        guard let task, let data = try? JSONSerialization.data(withJSONObject: object),
              let text = String(data: data, encoding: .utf8) else { return }
        Task {
            try? await task.send(.string(text))
        }
    }

    func submitAnswer(questionIndex: Int, answer: LiveGameLogic.AnswerPayload, powerUp: String? = nil) {
        let iso = ISO8601DateFormatter().string(from: Date())
        send(LiveGameLogic.answerMessage(
            questionIndex: questionIndex,
            answer: answer,
            clientSentAt: iso,
            powerUp: powerUp
        ))
    }

    private func openConnection(authToken: String?) {
        task?.cancel(with: .normalClosure, reason: nil)
        onConn(retry > 0 ? .reconnecting : .connecting)

        let path = LiveQuizLogic.webSocketPath(courseCode: courseCode, gameId: gameId)
        let newTask = WebSocketSessionSupport.shared.session.webSocketTask(
            with: AppConfiguration.webSocketURL(path: path)
        )
        task = newTask
        newTask.resume()

        Task { [weak self] in
            guard let self else { return }
            let opened = await WebSocketSessionSupport.shared.waitForOpen(newTask)
            guard opened, !self.isStopped, self.task === newTask else {
                await self.handleDisconnect()
                return
            }
            let handshake = LiveGameLogic.authHandshake(
                authToken: authToken,
                role: self.role,
                playerToken: self.playerToken
            )
            if let data = try? JSONSerialization.data(withJSONObject: handshake),
               let text = String(data: data, encoding: .utf8) {
                try? await newTask.send(.string(text))
            }
            if self.seq > 0 {
                self.send(LiveGameLogic.catchupMessage(afterSeq: self.seq))
            } else if self.role == .player {
                self.send(LiveGameLogic.helloMessage(resumeSeq: 0))
            }
            do {
                try await self.receiveLoop(task: newTask)
            } catch {
                await self.handleDisconnect()
            }
        }
    }

    private func receiveLoop(task: URLSessionWebSocketTask) async throws {
        while true {
            let message = try await task.receive()
            let data: Data?
            switch message {
            case .data(let d): data = d
            case .string(let text): data = text.data(using: .utf8)
            @unknown default: data = nil
            }
            guard let data else { continue }
            switch LiveGameLogic.parseInbound(data) {
            case .kicked:
                kicked = true
                isStopped = true
                onKicked()
                onConn(.kicked)
                task.cancel(with: .normalClosure, reason: nil)
                return
            case .answerAck(let ack):
                onAck(ack)
            case .state(let frame):
                seq = frame.seq
                onState(frame)
                onConn(frame.phase == LiveGameLogic.Phase.ended.rawValue ? .ended : .connected)
                retry = 0
                LiveQuizObservability.record("live_quiz_reconnect", attributes: ["ok": "1"])
            case .unknown:
                break
            }
        }
    }

    private func handleDisconnect() async {
        guard !isStopped, !kicked else { return }
        task = nil
        retry += 1
        LiveQuizObservability.record("live_quiz_reconnect", attributes: ["ok": "0"])
        scheduleReconnect()
    }

    private func scheduleReconnect() {
        onConn(.reconnecting)
        let delayMs = LiveGameLogic.reconnectDelayMs(retry: retry)
        reconnectTask = Task { [weak self] in
            try? await Task.sleep(for: .milliseconds(delayMs))
            guard let self, !Task.isCancelled else { return }
            self.connect()
        }
    }
}
