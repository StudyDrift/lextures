import AuthenticationServices
import Foundation
import UIKit

enum MfaPasskeyController {
    @MainActor
    static func performCeremony(
        optionsData: Data,
        setup: Bool
    ) async throws -> Data {
        let json = try JSONSerialization.jsonObject(with: optionsData) as? [String: Any]
        let publicKey = (json?["publicKey"] as? [String: Any]) ?? json ?? [:]
        guard
            let challengeString = publicKey["challenge"] as? String,
            let challenge = base64URLDecode(challengeString),
            let rpId = (publicKey["rpId"] as? String) ?? (publicKey["rp"] as? [String: Any])?["id"] as? String
        else {
            throw MFAChallengeError.passkeyFailed
        }

        let provider = ASAuthorizationPlatformPublicKeyCredentialProvider(relyingPartyIdentifier: rpId)
        let request: ASAuthorizationRequest
        if setup {
            let user = publicKey["user"] as? [String: Any]
            let userIdString = user?["id"] as? String ?? ""
            guard let userId = base64URLDecode(userIdString) else {
                throw MFAChallengeError.passkeyFailed
            }
            let name = (user?["name"] as? String) ?? (user?["displayName"] as? String) ?? "Account"
            request = provider.createCredentialRegistrationRequest(
                challenge: challenge,
                name: name,
                userID: userId
            )
        } else {
            let assertion = provider.createCredentialAssertionRequest(challenge: challenge)
            if let allow = publicKey["allowCredentials"] as? [[String: Any]] {
                assertion.allowedCredentials = allow.compactMap { item in
                    guard
                        let idString = item["id"] as? String,
                        let id = base64URLDecode(idString)
                    else { return nil }
                    return ASAuthorizationPlatformPublicKeyCredentialDescriptor(credentialID: id)
                }
            }
            request = assertion
        }

        let controller = ASAuthorizationController(authorizationRequests: [request])
        let delegate = PasskeyDelegate()
        controller.delegate = delegate
        controller.presentationContextProvider = PasskeyPresentationContext.shared
        controller.performRequests()

        let credential = try await delegate.result()
        return try encodeCredential(credential, setup: setup)
    }

    private static func encodeCredential(_ credential: ASAuthorizationCredential, setup: Bool) throws -> Data {
        if setup, let registration = credential as? ASAuthorizationPlatformPublicKeyCredentialRegistration {
            let payload: [String: Any] = [
                "id": base64URLEncode(registration.credentialID),
                "rawId": base64URLEncode(registration.credentialID),
                "type": "public-key",
                "response": [
                    "clientDataJSON": base64URLEncode(registration.rawClientDataJSON),
                    "attestationObject": base64URLEncode(registration.rawAttestationObject ?? Data()),
                ],
            ]
            return try JSONSerialization.data(withJSONObject: payload)
        }
        if !setup, let assertion = credential as? ASAuthorizationPlatformPublicKeyCredentialAssertion {
            let payload: [String: Any] = [
                "id": base64URLEncode(assertion.credentialID),
                "rawId": base64URLEncode(assertion.credentialID),
                "type": "public-key",
                "response": [
                    "clientDataJSON": base64URLEncode(assertion.rawClientDataJSON),
                    "authenticatorData": base64URLEncode(assertion.rawAuthenticatorData),
                    "signature": base64URLEncode(assertion.signature),
                    "userHandle": base64URLEncode(assertion.userID),
                ],
            ]
            return try JSONSerialization.data(withJSONObject: payload)
        }
        throw MFAChallengeError.passkeyFailed
    }

    private static func base64URLDecode(_ value: String) -> Data? {
        var base64 = value
            .replacingOccurrences(of: "-", with: "+")
            .replacingOccurrences(of: "_", with: "/")
        while base64.count % 4 != 0 { base64.append("=") }
        return Data(base64Encoded: base64)
    }

    private static func base64URLEncode(_ data: Data) -> String {
        data.base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
    }
}

enum MFAChallengeError: LocalizedError {
    case passkeyFailed
    case passkeyUnsupported

    var errorDescription: String? {
        switch self {
        case .passkeyFailed:
            return String(localized: "auth.mfa.passkeyFailed")
        case .passkeyUnsupported:
            return String(localized: "auth.mfa.passkeyUnsupported")
        }
    }
}

@MainActor
private final class PasskeyDelegate: NSObject, ASAuthorizationControllerDelegate {
    private var continuation: CheckedContinuation<ASAuthorizationCredential, Error>?

    func result() async throws -> ASAuthorizationCredential {
        try await withCheckedThrowingContinuation { continuation in
            self.continuation = continuation
        }
    }

    func authorizationController(
        controller: ASAuthorizationController,
        didCompleteWithAuthorization authorization: ASAuthorization
    ) {
        continuation?.resume(returning: authorization.credential)
        continuation = nil
    }

    func authorizationController(
        controller: ASAuthorizationController,
        didCompleteWithError error: Error
    ) {
        continuation?.resume(throwing: error)
        continuation = nil
    }
}

private final class PasskeyPresentationContext: NSObject, ASAuthorizationControllerPresentationContextProviding {
    static let shared = PasskeyPresentationContext()

    func presentationAnchor(for controller: ASAuthorizationController) -> ASPresentationAnchor {
        let scenes = UIApplication.shared.connectedScenes.compactMap { $0 as? UIWindowScene }
        return scenes.flatMap(\.windows).first { $0.isKeyWindow } ?? ASPresentationAnchor()
    }
}
