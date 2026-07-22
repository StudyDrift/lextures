import AuthenticationServices
import CryptoKit
import Foundation
import UIKit

enum AppleSignInError: LocalizedError {
    case cancelled
    case missingIdentityToken
    case failed

    var errorDescription: String? {
        switch self {
        case .cancelled:
            return nil
        case .missingIdentityToken, .failed:
            return String(localized: "auth.social.appleFailed")
        }
    }
}

struct AppleSignInResult {
    var idToken: String
    var rawNonce: String
    var authorizationCode: String?
    var fullName: String?
    var email: String?
}

/// Native Sign in with Apple via AuthenticationServices (MOB.9 / App Store Guideline 4.8).
enum AppleSignInController {
    @MainActor
    static func signIn() async throws -> AppleSignInResult {
        let rawNonce = randomNonceString()
        let hashed = sha256Hex(rawNonce)
        return try await withCheckedThrowingContinuation { continuation in
            let delegate = AppleSignInDelegate(rawNonce: rawNonce, continuation: continuation)
            AppleSignInDelegate.current = delegate
            let provider = ASAuthorizationAppleIDProvider()
            let request = provider.createRequest()
            request.requestedScopes = [.fullName, .email]
            request.nonce = hashed
            let controller = ASAuthorizationController(authorizationRequests: [request])
            controller.delegate = delegate
            controller.presentationContextProvider = PresentationContext.shared
            controller.performRequests()
        }
    }

    /// Cryptographically random raw nonce (sent to the server; SHA256 is set on the Apple request).
    static func randomNonceString(length: Int = 32) -> String {
        precondition(length > 0)
        let charset = Array("0123456789ABCDEFGHIJKLMNOPQRSTUVXYZabcdefghijklmnopqrstuvwxyz-._")
        var result = ""
        var remaining = length
        while remaining > 0 {
            var randoms = [UInt8](repeating: 0, count: 16)
            let status = SecRandomCopyBytes(kSecRandomDefault, randoms.count, &randoms)
            if status != errSecSuccess {
                // Fall back to UUID entropy if the secure RNG fails (should be rare).
                return UUID().uuidString.replacingOccurrences(of: "-", with: "")
            }
            for random in randoms where remaining > 0 {
                if random < charset.count {
                    result.append(charset[Int(random)])
                    remaining -= 1
                }
            }
        }
        return result
    }

    static func sha256Hex(_ input: String) -> String {
        let data = Data(input.utf8)
        let hash = SHA256.hash(data: data)
        return hash.map { String(format: "%02x", $0) }.joined()
    }
}

private final class AppleSignInDelegate: NSObject, ASAuthorizationControllerDelegate {
    static var current: AppleSignInDelegate?

    private let rawNonce: String
    private let continuation: CheckedContinuation<AppleSignInResult, Error>
    private var resumed = false

    init(rawNonce: String, continuation: CheckedContinuation<AppleSignInResult, Error>) {
        self.rawNonce = rawNonce
        self.continuation = continuation
    }

    func authorizationController(
        controller: ASAuthorizationController,
        didCompleteWithAuthorization authorization: ASAuthorization
    ) {
        defer { AppleSignInDelegate.current = nil }
        guard !resumed else { return }
        resumed = true
        guard let credential = authorization.credential as? ASAuthorizationAppleIDCredential,
              let tokenData = credential.identityToken,
              let idToken = String(data: tokenData, encoding: .utf8),
              !idToken.isEmpty
        else {
            continuation.resume(throwing: AppleSignInError.missingIdentityToken)
            return
        }
        var authCode: String?
        if let codeData = credential.authorizationCode {
            authCode = String(data: codeData, encoding: .utf8)
        }
        var fullName: String?
        if let name = credential.fullName {
            let parts = [name.givenName, name.familyName].compactMap { $0 }.filter { !$0.isEmpty }
            if !parts.isEmpty {
                fullName = parts.joined(separator: " ")
            }
        }
        continuation.resume(returning: AppleSignInResult(
            idToken: idToken,
            rawNonce: rawNonce,
            authorizationCode: authCode,
            fullName: fullName,
            email: credential.email
        ))
    }

    func authorizationController(controller: ASAuthorizationController, didCompleteWithError error: Error) {
        defer { AppleSignInDelegate.current = nil }
        guard !resumed else { return }
        resumed = true
        if let authError = error as? ASAuthorizationError, authError.code == .canceled {
            continuation.resume(throwing: AppleSignInError.cancelled)
            return
        }
        continuation.resume(throwing: AppleSignInError.failed)
    }
}

private final class PresentationContext: NSObject, ASAuthorizationControllerPresentationContextProviding {
    static let shared = PresentationContext()

    func presentationAnchor(for controller: ASAuthorizationController) -> ASPresentationAnchor {
        let scenes = UIApplication.shared.connectedScenes.compactMap { $0 as? UIWindowScene }
        return scenes.flatMap(\.windows).first { $0.isKeyWindow } ?? ASPresentationAnchor()
    }
}
