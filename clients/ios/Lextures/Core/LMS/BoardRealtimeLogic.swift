import Foundation

/// Connection chip states for board realtime (VC.M4 FR-8).
enum BoardSyncState: String, Equatable, Sendable {
    case connecting
    case live
    case reconnecting
    case offline
}

/// Parsed `board.changed` text frame from the board WebSocket relay.
struct BoardChangedEvent: Equatable, Sendable {
    var reason: String
    var postId: String?
}

/// Pure helpers for board realtime (VC.M4) — unit-tested without a live socket.
enum BoardRealtimeLogic {
    /// HTTP statuses that mean the upgrade was refused permanently (flag off, no access, missing board).
    static func isPermanentWsRefusal(_ statusCode: Int?) -> Bool {
        guard let statusCode else { return false }
        return statusCode == 401 || statusCode == 403 || statusCode == 404
    }

    /// After this many consecutive failures without a successful open, stop retrying (silent post-upgrade closes).
    static let maxTransientFailuresBeforeOffline = 8

    static func shouldStopRetrying(consecutiveFailures: Int, lastHttpStatus: Int?) -> Bool {
        if isPermanentWsRefusal(lastHttpStatus) { return true }
        return consecutiveFailures >= maxTransientFailuresBeforeOffline
    }

    /// Parses a JSON text frame into a `board.changed` event. Binary / non-JSON / other types → nil.
    static func parseBoardChangedEvent(from data: Data) -> BoardChangedEvent? {
        guard let msg = try? JSONDecoder().decode(BoardChangedFrame.self, from: data),
              msg.type == "board.changed",
              let reason = msg.reason, !reason.isEmpty else {
            return nil
        }
        let postId = msg.postId?.trimmingCharacters(in: .whitespacesAndNewlines)
        return BoardChangedEvent(
            reason: reason,
            postId: (postId?.isEmpty == false) ? postId : nil
        )
    }

    static func parseBoardChangedEvent(from text: String) -> BoardChangedEvent? {
        guard let data = text.data(using: .utf8) else { return nil }
        return parseBoardChangedEvent(from: data)
    }

    /// Detects `{"error":"board_locked_or_frozen"}` notice frames.
    static func isBoardLockedOrFrozenError(from data: Data) -> Bool {
        guard let msg = try? JSONDecoder().decode(BoardErrorFrame.self, from: data) else { return false }
        return msg.error == "board_locked_or_frozen"
    }

    static func isBoardLockedOrFrozenError(from text: String) -> Bool {
        guard let data = text.data(using: .utf8) else { return false }
        return isBoardLockedOrFrozenError(from: data)
    }

    /// Coalesces a burst of change events into one refetch plan (FR-5 / AC-7).
    /// A general bump (no postId) or multiple distinct postIds → full list refetch.
    static func coalesceRefetchPlan(events: [BoardChangedEvent]) -> BoardRefetchPlan {
        var postIds: [String] = []
        var seen = Set<String>()
        var hasGeneral = false
        for event in events {
            if let postId = event.postId, !postId.isEmpty {
                if seen.insert(postId).inserted {
                    postIds.append(postId)
                }
            } else {
                hasGeneral = true
            }
        }
        if hasGeneral || postIds.count != 1 {
            return BoardRefetchPlan(full: true, postId: nil, createdCount: events.filter { $0.reason == "post.created" }.count)
        }
        return BoardRefetchPlan(
            full: false,
            postId: postIds.first,
            createdCount: events.filter { $0.reason == "post.created" }.count
        )
    }

    /// Debounce window for coalescing `board.changed` bursts (milliseconds).
    static let refetchCoalesceMs: UInt64 = 250
}

struct BoardRefetchPlan: Equatable, Sendable {
    var full: Bool
    var postId: String?
    var createdCount: Int
}

private struct BoardChangedFrame: Decodable {
    let type: String?
    let reason: String?
    let postId: String?
}

private struct BoardErrorFrame: Decodable {
    let error: String?
}
