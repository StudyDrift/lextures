import Foundation
import Observation

/// Per-screen board WebSocket (VC.M4). Connects to the same relay as web
/// (`GET /api/v1/courses/{code}/boards/{board_id}/ws`), listens for JSON `board.changed`
/// text frames, and ignores binary Y.js replay/sync/awareness frames.
///
/// Exposes a coalesced revision counter so the board detail screen can refetch without
/// stampeding on bursty boards. Lifecycle mirrors `FeedSocket` (connect on open, tear down
/// on leave/background).
@MainActor
@Observable
final class BoardSocket {
    private(set) var connectionState: BoardSyncState = .connecting
    private(set) var revision = 0
    private(set) var lastRefetchPlan = BoardRefetchPlan(full: true, postId: nil, createdCount: 0)
    /// Bumped on every successful (re)connect so the screen can full-refetch (FR-7).
    private(set) var connectRevision = 0
    /// Set when the server pushes `board_locked_or_frozen`; clear after the UI shows it.
    private(set) var lockedOrFrozenNotice = false

    private var socket: WebSocketClient?
    private var coalesceTask: Task<Void, Never>?
    private var pendingEvents: [BoardChangedEvent] = []
    private var consecutiveFailures = 0

    /// Connects to the board relay. Safe to call from `.task {}`.
    func connect(courseCode: String, boardId: String, accessToken: @escaping () -> String?) {
        guard socket == nil else {
            socket?.connect()
            return
        }
        connectionState = .connecting
        consecutiveFailures = 0
        let path = "/api/v1/courses/\(courseCode)/boards/\(boardId)/ws"
        socket = WebSocketClient(
            path: path,
            accessTokenProvider: accessToken,
            onMessage: { [weak self] data in
                Task { @MainActor in self?.handleMessage(data) }
            },
            onLifecycle: { [weak self] event in
                Task { @MainActor in self?.handleLifecycle(event) }
            },
            stopOnPermanentRefusal: true
        )
        socket?.connect()
    }

    /// Call from `.onDisappear` / background so the socket doesn't outlive the screen.
    func disconnect() {
        coalesceTask?.cancel()
        coalesceTask = nil
        pendingEvents = []
        socket?.disconnect()
        socket = nil
        connectionState = .offline
    }

    func clearLockedOrFrozenNotice() {
        lockedOrFrozenNotice = false
    }

    private func handleLifecycle(_ event: WebSocketLifecycleEvent) {
        switch event {
        case .opened:
            consecutiveFailures = 0
            connectionState = .live
            connectRevision += 1
        case let .closed(httpStatus, willReconnect):
            if !willReconnect || BoardRealtimeLogic.shouldStopRetrying(
                consecutiveFailures: consecutiveFailures + 1,
                lastHttpStatus: httpStatus
            ) {
                consecutiveFailures = 0
                connectionState = .offline
                if willReconnect {
                    // Cap reached — stop the underlying client's reconnect loop.
                    socket?.disconnect()
                    socket = nil
                }
                return
            }
            consecutiveFailures += 1
            connectionState = .reconnecting
        }
    }

    private func handleMessage(_ data: Data) {
        if BoardRealtimeLogic.isBoardLockedOrFrozenError(from: data) {
            lockedOrFrozenNotice = true
            return
        }
        guard let event = BoardRealtimeLogic.parseBoardChangedEvent(from: data) else {
            // Binary Y.js frames and unrelated JSON are a safe no-op (FR-4).
            return
        }
        pendingEvents.append(event)
        coalesceTask?.cancel()
        coalesceTask = Task { [weak self] in
            try? await Task.sleep(for: .milliseconds(BoardRealtimeLogic.refetchCoalesceMs))
            guard let self, !Task.isCancelled else { return }
            self.flushPendingEvents()
        }
    }

    private func flushPendingEvents() {
        guard !pendingEvents.isEmpty else { return }
        let plan = BoardRealtimeLogic.coalesceRefetchPlan(events: pendingEvents)
        pendingEvents = []
        lastRefetchPlan = plan
        revision += 1
    }
}
