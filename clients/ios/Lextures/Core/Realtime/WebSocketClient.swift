import Foundation

/// Shared `URLSession` delegate that signals when a WebSocket handshake completes.
///
/// `URLSessionWebSocketTask` connects asynchronously after `resume()`; sending before
/// `urlSession(_:webSocketTask:didOpenWithProtocol:)` returns POSIX 57 ("Socket is not connected")
/// on device. Android and the web SPA wait for `onOpen` before the auth frame — this does the same.
private final class WebSocketSessionSupport: NSObject, URLSessionWebSocketDelegate, @unchecked Sendable {
    static let shared = WebSocketSessionSupport()

    lazy var session: URLSession = {
        NetworkBootstrap.warmup()
        return URLSession(configuration: .ephemeral, delegate: self, delegateQueue: nil)
    }()

    private let lock = NSLock()
    private var openContinuations: [ObjectIdentifier: CheckedContinuation<Bool, Never>] = [:]
    private var openedTasks: Set<ObjectIdentifier> = []

    /// Suspends until the handshake completes. Returns `true` when `didOpen` fires, `false` when
    /// the task ends before the socket is connected (server down, TLS failure, etc.).
    func waitForOpen(_ task: URLSessionWebSocketTask) async -> Bool {
        await withCheckedContinuation { continuation in
            lock.lock()
            openContinuations[ObjectIdentifier(task)] = continuation
            lock.unlock()
        }
    }

    func urlSession(
        _ session: URLSession,
        webSocketTask: URLSessionWebSocketTask,
        didOpenWithProtocol protocol: String?
    ) {
        lock.lock()
        openedTasks.insert(ObjectIdentifier(webSocketTask))
        lock.unlock()
        resumeOpenWaiter(for: webSocketTask, opened: true)
    }

    func urlSession(_ session: URLSession, task: URLSessionTask, didCompleteWithError error: Error?) {
        guard let webSocketTask = task as? URLSessionWebSocketTask else { return }
        lock.lock()
        let hadOpened = openedTasks.remove(ObjectIdentifier(webSocketTask)) != nil
        lock.unlock()
        guard !hadOpened else { return }
        resumeOpenWaiter(for: webSocketTask, opened: false)
    }

    private func resumeOpenWaiter(for task: URLSessionWebSocketTask, opened: Bool) {
        lock.lock()
        let continuation = openContinuations.removeValue(forKey: ObjectIdentifier(task))
        lock.unlock()
        continuation?.resume(returning: opened)
    }
}

/// Lifecycle hooks for per-screen sockets that need connection-state UX (e.g. BoardSocket).
enum WebSocketLifecycleEvent: Equatable, Sendable {
    case opened
    /// `httpStatus` is set when the HTTP upgrade failed before the socket opened (e.g. 404).
    /// `willReconnect` is false when the client stopped retrying (permanent refusal or disconnect()).
    case closed(httpStatus: Int?, willReconnect: Bool)
}

/// Reconnecting JSON WebSocket client for the server's realtime hubs.
///
/// Mirrors the web client's handshake (`clients/web/src/context/inbox-unread-provider.tsx`):
/// on open, sends `{"authToken":"<jwt>"}` as the first text message (the server reads exactly
/// one auth message before subscribing — see `handleCommWS` in
/// `server/internal/httpserver/communication.go`), then treats every later frame as an event
/// payload. Reconnects after a fixed 2s delay, matching the web app's `scheduleReconnect`.
@MainActor
final class WebSocketClient {
    private let path: String
    private let accessTokenProvider: () -> String?
    private let onMessage: (Data) -> Void
    private let onLifecycle: ((WebSocketLifecycleEvent) -> Void)?
    private let stopOnPermanentRefusal: Bool

    private var task: URLSessionWebSocketTask?
    private var connectedToken: String?
    private var isStopped = true
    private var reconnectTask: Task<Void, Never>?

    init(
        path: String,
        accessTokenProvider: @escaping () -> String?,
        onMessage: @escaping (Data) -> Void,
        onLifecycle: ((WebSocketLifecycleEvent) -> Void)? = nil,
        stopOnPermanentRefusal: Bool = false
    ) {
        self.path = path
        self.accessTokenProvider = accessTokenProvider
        self.onMessage = onMessage
        self.onLifecycle = onLifecycle
        self.stopOnPermanentRefusal = stopOnPermanentRefusal
    }

    /// Connects if not already connected with the current token. Safe to call repeatedly
    /// (e.g. on scenePhase changes or token refresh) — it's a no-op when already connected.
    func connect() {
        isStopped = false
        reconnectTask?.cancel()
        reconnectTask = nil
        guard let token = accessTokenProvider(), !token.isEmpty else { return }
        if task != nil, connectedToken == token { return }
        openConnection(token: token)
    }

    func disconnect() {
        isStopped = true
        reconnectTask?.cancel()
        reconnectTask = nil
        connectedToken = nil
        task?.cancel(with: .normalClosure, reason: nil)
        task = nil
    }

    private func openConnection(token: String) {
        task?.cancel(with: .normalClosure, reason: nil)
        connectedToken = token

        let newTask = WebSocketSessionSupport.shared.session.webSocketTask(
            with: AppConfiguration.webSocketURL(path: path)
        )
        task = newTask
        newTask.resume()

        guard let authData = try? JSONSerialization.data(withJSONObject: ["authToken": token]),
              let authString = String(data: authData, encoding: .utf8) else {
            return
        }

        Task { [weak self] in
            guard let self else { return }
            do {
                let opened = await WebSocketSessionSupport.shared.waitForOpen(newTask)
                guard opened, !self.isStopped, self.task === newTask else {
                    let status = (newTask.response as? HTTPURLResponse)?.statusCode
                    await self.handleDisconnect(after: newTask, httpStatus: status)
                    return
                }
                self.onLifecycle?(.opened)
                try await newTask.send(.string(authString))
                try await self.receiveLoop(task: newTask)
                await self.handleDisconnect(after: newTask, httpStatus: nil)
            } catch {
                let status = (newTask.response as? HTTPURLResponse)?.statusCode
                await self.handleDisconnect(after: newTask, httpStatus: status)
            }
        }
    }

    private func receiveLoop(task: URLSessionWebSocketTask) async throws {
        while true {
            let message = try await task.receive()
            switch message {
            case .data(let data):
                onMessage(data)
            case .string(let text):
                if let data = text.data(using: .utf8) {
                    onMessage(data)
                }
            @unknown default:
                break
            }
        }
    }

    private func handleDisconnect(after finishedTask: URLSessionWebSocketTask, httpStatus: Int?) async {
        guard !isStopped, task === finishedTask else {
            onLifecycle?(.closed(httpStatus: httpStatus, willReconnect: false))
            return
        }
        task = nil
        connectedToken = nil

        let permanent = stopOnPermanentRefusal && Self.isPermanentWsRefusal(httpStatus)
        if permanent {
            isStopped = true
            onLifecycle?(.closed(httpStatus: httpStatus, willReconnect: false))
            return
        }

        onLifecycle?(.closed(httpStatus: httpStatus, willReconnect: true))
        reconnectTask = Task { [weak self] in
            try? await Task.sleep(for: .seconds(2))
            guard let self, !Task.isCancelled else { return }
            self.connect()
        }
    }

    private static func isPermanentWsRefusal(_ statusCode: Int?) -> Bool {
        guard let statusCode else { return false }
        return statusCode == 401 || statusCode == 403 || statusCode == 404
    }
}
