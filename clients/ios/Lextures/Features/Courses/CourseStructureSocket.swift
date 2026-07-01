import Foundation
import Observation

/// Course-scoped realtime socket for `structure_changed` events (module/item edits, imports
/// finishing, etc). Unlike `RealtimeManager`'s app-wide hubs, this is created per screen and
/// torn down when the course is no longer on screen — mirrors the web app's
/// `use-course-structure-ws.ts` hook.
@MainActor
@Observable
final class CourseStructureSocket {
    private(set) var revision = 0

    private var socket: WebSocketClient?

    /// Connects to `/api/v1/courses/{code}/structure/ws`. Safe to call from `.task {}`.
    func connect(courseCode: String, accessToken: @escaping () -> String?) {
        guard socket == nil else { return }
        socket = WebSocketClient(
            path: "/api/v1/courses/\(courseCode)/structure/ws",
            accessTokenProvider: accessToken,
            onMessage: { [weak self] data in
                Task { @MainActor in self?.handleMessage(data) }
            }
        )
        socket?.connect()
    }

    /// Call from `.onDisappear` so the socket doesn't outlive the screen.
    func disconnect() {
        socket?.disconnect()
        socket = nil
    }

    private func handleMessage(_ data: Data) {
        guard let event = try? JSONDecoder().decode(StructureWSEvent.self, from: data),
              event.type == "structure_changed" else { return }
        revision += 1
    }
}

private struct StructureWSEvent: Decodable {
    let type: String
}
