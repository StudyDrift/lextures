import Foundation
import Observation

/// Course-scoped realtime socket for `files_changed` events (uploads/deletes from web or other
/// devices). Created per screen and torn down when the folder browser leaves the screen —
/// mirrors `CourseStructureSocket`.
@MainActor
@Observable
final class CourseFilesSocket {
    private(set) var revision = 0

    private var socket: WebSocketClient?

    /// Connects to `/api/v1/courses/{code}/files/ws`. Safe to call from `.task {}`.
    func connect(courseCode: String, accessToken: @escaping () -> String?) {
        guard socket == nil else { return }
        socket = WebSocketClient(
            path: "/api/v1/courses/\(courseCode)/files/ws",
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
        guard let event = try? JSONDecoder().decode(FilesWSEvent.self, from: data),
              event.type == "files_changed" else { return }
        revision += 1
    }
}

private struct FilesWSEvent: Decodable {
    let type: String
}
