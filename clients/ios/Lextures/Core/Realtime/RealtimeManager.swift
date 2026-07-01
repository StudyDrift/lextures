import Foundation
import Observation

/// Owns the app-wide realtime sockets (mailbox/courses/enrollments + notifications) for the
/// lifetime of an authenticated session. Feature screens observe the revision counters and
/// re-run their existing REST fetches when they bump — mirroring
/// `clients/web/src/context/inbox-unread-provider.tsx`, which does the same with local state
/// instead of a shared cache.
@MainActor
@Observable
final class RealtimeManager {
    static let shared = RealtimeManager()

    private(set) var mailboxRevision = 0
    private(set) var coursesRevision = 0
    private(set) var enrollmentsRevision = 0
    private(set) var lastEnrollmentCourseCode: String?
    private(set) var notificationsRevision = 0

    private var commSocket: WebSocketClient?
    private var notificationsSocket: WebSocketClient?
    private var accessTokenProvider: (() -> String?)?

    private init() {}

    /// Connects (or reconnects, if the token changed) the app-wide sockets. Call whenever the
    /// session becomes authenticated or the access token rotates.
    func configure(accessToken: @escaping () -> String?) {
        accessTokenProvider = accessToken

        if commSocket == nil {
            commSocket = WebSocketClient(
                path: "/api/v1/communication/ws",
                accessTokenProvider: accessToken,
                onMessage: { [weak self] data in
                    Task { @MainActor in self?.handleCommMessage(data) }
                }
            )
        }
        if notificationsSocket == nil {
            notificationsSocket = WebSocketClient(
                path: "/api/v1/ws/notifications",
                accessTokenProvider: accessToken,
                onMessage: { [weak self] data in
                    Task { @MainActor in self?.handleNotificationsMessage(data) }
                }
            )
        }
        commSocket?.connect()
        notificationsSocket?.connect()
    }

    /// Called on sign-out (and can be called on scenePhase background if desired) to stop
    /// reconnect attempts and close both sockets.
    func disconnect() {
        commSocket?.disconnect()
        notificationsSocket?.disconnect()
        accessTokenProvider = nil
    }

    private func handleCommMessage(_ data: Data) {
        guard let event = try? JSONDecoder().decode(CommWSEvent.self, from: data) else { return }
        switch event.type {
        case "mailbox_updated":
            mailboxRevision += 1
        case "courses_updated":
            coursesRevision += 1
        case "enrollments_updated":
            lastEnrollmentCourseCode = event.courseCode
            enrollmentsRevision += 1
        default:
            break
        }
    }

    private func handleNotificationsMessage(_ data: Data) {
        guard let event = try? JSONDecoder().decode(CommWSEvent.self, from: data),
              event.type == "notification_updated" else { return }
        notificationsRevision += 1
    }
}

private struct CommWSEvent: Decodable {
    let type: String
    let courseCode: String?

    enum CodingKeys: String, CodingKey {
        case type
        case courseCode = "courseCode"
        case courseCodeSnake = "course_code"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        type = try container.decode(String.self, forKey: .type)
        courseCode = try container.decodeIfPresent(String.self, forKey: .courseCode)
            ?? container.decodeIfPresent(String.self, forKey: .courseCodeSnake)
    }
}
