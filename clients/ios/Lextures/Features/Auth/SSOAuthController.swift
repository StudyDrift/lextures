import AuthenticationServices
import Foundation
import UIKit

enum SSOAuthError: LocalizedError {
    case cancelled
    case noCallback
    case invalidCallback

    var errorDescription: String? {
        switch self {
        case .cancelled:
            return String(localized: "auth.login.ssoCancelled")
        case .noCallback, .invalidCallback:
            return String(localized: "auth.login.ssoFailed")
        }
    }
}

enum SSOProvider: Equatable {
    case saml(idpId: String)
    case oidc(path: String, label: String)
}

@MainActor
enum SSOAuthController {
    static func start(provider: SSOProvider) async throws -> AuthCallbackPayload {
        let startURL = buildStartURL(for: provider)
        return try await withCheckedThrowingContinuation { continuation in
            let session = ASWebAuthenticationSession(
                url: startURL,
                callbackURLScheme: AuthConstants.callbackScheme
            ) { url, error in
                if let error = error as? ASWebAuthenticationSessionError,
                   error.code == .canceledLogin {
                    continuation.resume(throwing: SSOAuthError.cancelled)
                    return
                }
                if let error {
                    continuation.resume(throwing: error)
                    return
                }
                guard let url, let payload = AuthCallbackParser.parseAuthCallback(url.absoluteString) else {
                    continuation.resume(throwing: SSOAuthError.invalidCallback)
                    return
                }
                continuation.resume(returning: payload)
            }
            session.prefersEphemeralWebBrowserSession = false
            session.presentationContextProvider = PresentationContext.shared
            if !session.start() {
                continuation.resume(throwing: SSOAuthError.noCallback)
            }
        }
    }

    static func buildStartURL(for provider: SSOProvider) -> URL {
        let next = AuthConstants.mobileCallbackPath
        let path: String
        switch provider {
        case let .saml(idpId):
            path = "/auth/saml/login?idpId=\(urlEncode(idpId))&RelayState=\(urlEncode(next))"
        case let .oidc(providerPath, _):
            let separator = providerPath.contains("?") ? "&" : "?"
            path = "\(providerPath)\(separator)next=\(urlEncode(next))"
        }
        return AppConfiguration.apiURL(path: path)
    }

    private static func urlEncode(_ value: String) -> String {
        value.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? value
    }
}

private final class PresentationContext: NSObject, ASWebAuthenticationPresentationContextProviding {
    static let shared = PresentationContext()

    func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
        let scenes = UIApplication.shared.connectedScenes.compactMap { $0 as? UIWindowScene }
        return scenes.flatMap(\.windows).first { $0.isKeyWindow } ?? ASPresentationAnchor()
    }
}
