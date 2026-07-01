import Foundation
import Observation

/// Course-scoped realtime socket for feed events (`{"type":"feed","scope":"channels"|"messages"}`).
/// Created per screen and torn down when the feed leaves the screen — mirrors
/// `CourseStructureSocket`. Exposes a per-channel revision counter so an open channel only
/// re-fetches when its own messages change.
@MainActor
@Observable
final class FeedSocket {
    private(set) var channelsRevision = 0
    private(set) var messagesRevision: [String: Int] = [:]
    private(set) var isConnected = false

    private var socket: WebSocketClient?

    /// Connects to `/api/v1/courses/{code}/feed/ws`. Safe to call from `.task {}`.
    func connect(courseCode: String, accessToken: @escaping () -> String?) {
        guard socket == nil else { return }
        socket = WebSocketClient(
            path: "/api/v1/courses/\(courseCode)/feed/ws",
            accessTokenProvider: accessToken,
            onMessage: { [weak self] data in
                Task { @MainActor in self?.handleMessage(data) }
            }
        )
        socket?.connect()
        isConnected = true
    }

    /// Call from `.onDisappear` so the socket doesn't outlive the screen.
    func disconnect() {
        socket?.disconnect()
        socket = nil
        isConnected = false
    }

    func revision(forChannel channelId: String) -> Int {
        messagesRevision[channelId] ?? 0
    }

    private func handleMessage(_ data: Data) {
        guard let event = try? JSONDecoder().decode(FeedWSEvent.self, from: data),
              event.type == "feed" else { return }
        switch event.scope {
        case "channels":
            channelsRevision += 1
        case "messages":
            guard let channelId = event.channelId else { return }
            messagesRevision[channelId, default: 0] += 1
        default:
            break
        }
    }
}

private struct FeedWSEvent: Decodable {
    let type: String
    let scope: String
    let channelId: String?
}
