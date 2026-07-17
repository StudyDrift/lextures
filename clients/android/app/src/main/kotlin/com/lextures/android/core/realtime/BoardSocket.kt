package com.lextures.android.core.realtime

import com.lextures.android.core.lms.BoardChangedEvent
import com.lextures.android.core.lms.BoardRealtimeLogic
import com.lextures.android.core.lms.BoardRefetchPlan
import com.lextures.android.core.lms.BoardSyncState
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * Per-screen board WebSocket (VC.M4). Connects to the same relay as web
 * (`GET /api/v1/courses/{code}/boards/{board_id}/ws`), listens for JSON `board.changed`
 * text frames, and ignores binary Y.js replay/sync/awareness frames.
 *
 * Exposes a coalesced revision counter so the board detail screen can refetch without
 * stampeding on bursty boards. Lifecycle mirrors [FeedSocket] (connect on open, tear down
 * on leave).
 */
class BoardSocket {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)

    private val _connectionState = MutableStateFlow(BoardSyncState.Connecting)
    val connectionState: StateFlow<BoardSyncState> = _connectionState.asStateFlow()

    private val _revision = MutableStateFlow(0)
    val revision: StateFlow<Int> = _revision.asStateFlow()

    private val _lastRefetchPlan = MutableStateFlow(BoardRefetchPlan(full = true))
    val lastRefetchPlan: StateFlow<BoardRefetchPlan> = _lastRefetchPlan.asStateFlow()

    /** Bumped on every successful (re)connect so the screen can full-refetch (FR-7). */
    private val _connectRevision = MutableStateFlow(0)
    val connectRevision: StateFlow<Int> = _connectRevision.asStateFlow()

    /** Set when the server pushes `board_locked_or_frozen`; clear after the UI shows it. */
    private val _lockedOrFrozenNotice = MutableStateFlow(false)
    val lockedOrFrozenNotice: StateFlow<Boolean> = _lockedOrFrozenNotice.asStateFlow()

    private var socket: WebSocketClient? = null
    private var coalesceJob: Job? = null
    private val pendingEvents = mutableListOf<BoardChangedEvent>()
    private var consecutiveFailures = 0

    /** Connects to the board relay. Safe to call from a composable effect. */
    fun connect(courseCode: String, boardId: String, accessTokenProvider: () -> String?) {
        if (socket != null) {
            socket?.connect()
            return
        }
        _connectionState.value = BoardSyncState.Connecting
        consecutiveFailures = 0
        socket = WebSocketClient(
            path = "/api/v1/courses/$courseCode/boards/$boardId/ws",
            accessTokenProvider = accessTokenProvider,
            onMessage = ::handleMessage,
            onLifecycle = ::handleLifecycle,
            stopOnPermanentRefusal = true,
        )
        socket?.connect()
    }

    /** Call from `DisposableEffect.onDispose` so the socket doesn't outlive the screen. */
    fun disconnect() {
        coalesceJob?.cancel()
        coalesceJob = null
        pendingEvents.clear()
        socket?.disconnect()
        socket = null
        _connectionState.value = BoardSyncState.Offline
    }

    fun clearLockedOrFrozenNotice() {
        _lockedOrFrozenNotice.value = false
    }

    private fun handleLifecycle(event: WebSocketLifecycleEvent) {
        scope.launch {
            when (event) {
                WebSocketLifecycleEvent.Opened -> {
                    consecutiveFailures = 0
                    _connectionState.value = BoardSyncState.Live
                    _connectRevision.value += 1
                }
                is WebSocketLifecycleEvent.Closed -> {
                    val nextFailures = consecutiveFailures + 1
                    if (!event.willReconnect ||
                        BoardRealtimeLogic.shouldStopRetrying(nextFailures, event.httpStatus)
                    ) {
                        consecutiveFailures = 0
                        _connectionState.value = BoardSyncState.Offline
                        if (event.willReconnect) {
                            socket?.disconnect()
                            socket = null
                        }
                        return@launch
                    }
                    consecutiveFailures = nextFailures
                    _connectionState.value = BoardSyncState.Reconnecting
                }
            }
        }
    }

    private fun handleMessage(text: String) {
        if (BoardRealtimeLogic.isBoardLockedOrFrozenError(text)) {
            _lockedOrFrozenNotice.value = true
            return
        }
        val event = BoardRealtimeLogic.parseBoardChangedEvent(text) ?: return
        scope.launch {
            pendingEvents.add(event)
            coalesceJob?.cancel()
            coalesceJob = scope.launch {
                delay(BoardRealtimeLogic.REFETCH_COALESCE_MS)
                flushPendingEvents()
            }
        }
    }

    private fun flushPendingEvents() {
        if (pendingEvents.isEmpty()) return
        val plan = BoardRealtimeLogic.coalesceRefetchPlan(pendingEvents.toList())
        pendingEvents.clear()
        _lastRefetchPlan.value = plan
        _revision.value += 1
    }
}
