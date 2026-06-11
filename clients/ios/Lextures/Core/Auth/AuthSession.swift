import Foundation
import Observation

enum AuthPhase: Equatable {
    case splash
    case unauthenticated
    case authenticated
}

enum MFARequired: Equatable {
    case challenge
    case setup
}

@MainActor
@Observable
final class AuthSession {
    var phase: AuthPhase = .splash
    var accessToken: String?
    var userEmail: String?
    var mfaRequired: MFARequired?

    private var refreshToken: String?
    private var refreshTask: Task<Void, Never>?

    var isSignedIn: Bool {
        accessToken != nil
    }

    init() {
        accessToken = KeychainStore.read(key: KeychainStore.Keys.accessToken)
        refreshToken = KeychainStore.read(key: KeychainStore.Keys.refreshToken)
    }

    func finishSplash() {
        phase = isSignedIn ? .authenticated : .unauthenticated
    }

    // MARK: Token refresh
    //
    // Backend access tokens last 15 minutes by design; refresh tokens last 30 days
    // and rotate on every exchange. Refreshing keeps the mobile session alive for
    // weeks without weakening the backend's short-lived bearer tokens.

    /// Refreshes the access token when it is missing, expired, or expiring soon.
    /// Coalesces concurrent callers into one in-flight exchange.
    func refreshIfNeeded(force: Bool = false) async {
        if let task = refreshTask {
            await task.value
            return
        }
        guard refreshToken != nil else { return }
        if !force, let token = accessToken, let expiry = Self.jwtExpiry(token),
           expiry.timeIntervalSinceNow > 120 {
            return
        }

        let task = Task { await performRefresh() }
        refreshTask = task
        await task.value
        refreshTask = nil
    }

    private func performRefresh() async {
        guard let refreshToken else { return }
        do {
            let response = try await AuthAPI.refresh(refreshToken: refreshToken)
            guard let token = response.accessToken, !token.isEmpty else { return }
            try? KeychainStore.save(key: KeychainStore.Keys.accessToken, value: token)
            if let rotated = response.refreshToken, !rotated.isEmpty {
                try? KeychainStore.save(key: KeychainStore.Keys.refreshToken, value: rotated)
                self.refreshToken = rotated
            }
            accessToken = token
            if let email = response.user?.email {
                userEmail = email
            }
        } catch {
            // Only an explicit rejection ends the session; network blips keep it.
            if case APIError.httpStatus(let code, _) = error, code == 401 || code == 403 {
                signOut()
            }
        }
    }

    /// Decodes the `exp` claim from a JWT without verifying the signature.
    static func jwtExpiry(_ jwt: String) -> Date? {
        let segments = jwt.split(separator: ".")
        guard segments.count >= 2 else { return nil }
        var base64 = String(segments[1])
            .replacingOccurrences(of: "-", with: "+")
            .replacingOccurrences(of: "_", with: "/")
        while base64.count % 4 != 0 { base64.append("=") }
        guard
            let data = Data(base64Encoded: base64),
            let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
            let exp = json["exp"] as? TimeInterval
        else { return nil }
        return Date(timeIntervalSince1970: exp)
    }

    func applyTokenResponse(_ response: AuthTokenResponse) throws {
        if response.requiresMFA == true, response.mfaPendingToken != nil {
            mfaRequired = response.mfaSetupRequired == true ? .setup : .challenge
            throw AuthSessionError.mfaRequired
        }

        guard let token = response.accessToken, !token.isEmpty else {
            throw AuthSessionError.missingAccessToken
        }

        try KeychainStore.save(key: KeychainStore.Keys.accessToken, value: token)
        if let refresh = response.refreshToken {
            try? KeychainStore.save(key: KeychainStore.Keys.refreshToken, value: refresh)
            refreshToken = refresh
        }

        accessToken = token
        userEmail = response.user?.email
        mfaRequired = nil
        phase = .authenticated
    }

    func signOut() {
        KeychainStore.deleteAll()
        accessToken = nil
        refreshToken = nil
        userEmail = nil
        mfaRequired = nil
        phase = .unauthenticated
    }

    func serverUnreachableMessage() -> String {
        "Could not reach the server. Is the API running?"
    }

    enum AuthSessionError: LocalizedError {
        case mfaRequired
        case missingAccessToken

        var errorDescription: String? {
            switch self {
            case .mfaRequired:
                return "Multi-factor authentication is required. Complete sign-in on the web app for now."
            case .missingAccessToken:
                return "Unexpected sign-in response."
            }
        }
    }
}
