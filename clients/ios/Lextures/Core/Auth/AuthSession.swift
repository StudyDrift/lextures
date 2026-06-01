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

    var isSignedIn: Bool {
        accessToken != nil
    }

    init() {
        accessToken = KeychainStore.read(key: KeychainStore.Keys.accessToken)
    }

    func finishSplash() {
        phase = isSignedIn ? .authenticated : .unauthenticated
    }

    func applyTokenResponse(_ response: AuthTokenResponse) throws {
        if response.requiresMFA == true, let _ = response.mfaPendingToken {
            mfaRequired = response.mfaSetupRequired == true ? .setup : .challenge
            throw AuthSessionError.mfaRequired
        }

        guard let token = response.accessToken, !token.isEmpty else {
            throw AuthSessionError.missingAccessToken
        }

        try KeychainStore.save(key: KeychainStore.Keys.accessToken, value: token)
        if let refresh = response.refreshToken {
            try? KeychainStore.save(key: KeychainStore.Keys.refreshToken, value: refresh)
        }

        accessToken = token
        userEmail = response.user?.email
        mfaRequired = nil
        phase = .authenticated
    }

    func signOut() {
        KeychainStore.deleteAll()
        accessToken = nil
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
