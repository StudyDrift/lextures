import Foundation

/// Weak link from the networking layer back to the active auth session for 401 refresh/retry.
@MainActor
enum NetworkAuthContext {
    private(set) static weak var session: AuthSession?

    static func configure(session: AuthSession) {
        self.session = session
    }
}
