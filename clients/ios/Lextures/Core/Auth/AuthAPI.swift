import Foundation

struct PasswordPolicy: Decodable {
    var minLength: Int
    var requireUpper: Bool
    var requireLower: Bool
    var requireDigit: Bool
    var requireSpecial: Bool
    var checkHibp: Bool

    enum CodingKeys: String, CodingKey {
        case minLength
        case requireUpper
        case requireLower
        case requireDigit
        case requireSpecial
        case checkHibp
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        minLength = try container.decodeIfPresent(Int.self, forKey: .minLength) ?? 8
        requireUpper = try container.decodeIfPresent(Bool.self, forKey: .requireUpper) ?? false
        requireLower = try container.decodeIfPresent(Bool.self, forKey: .requireLower) ?? false
        requireDigit = try container.decodeIfPresent(Bool.self, forKey: .requireDigit) ?? false
        requireSpecial = try container.decodeIfPresent(Bool.self, forKey: .requireSpecial) ?? false
        checkHibp = try container.decodeIfPresent(Bool.self, forKey: .checkHibp) ?? true
    }

    static let fallback = PasswordPolicy(
        minLength: 8,
        requireUpper: false,
        requireLower: false,
        requireDigit: false,
        requireSpecial: false,
        checkHibp: true
    )

    private init(
        minLength: Int,
        requireUpper: Bool,
        requireLower: Bool,
        requireDigit: Bool,
        requireSpecial: Bool,
        checkHibp: Bool
    ) {
        self.minLength = minLength
        self.requireUpper = requireUpper
        self.requireLower = requireLower
        self.requireDigit = requireDigit
        self.requireSpecial = requireSpecial
        self.checkHibp = checkHibp
    }
}

struct AuthUser: Decodable {
    var email: String?
    var uiTheme: String?
    var locale: String?
    var accountType: String?
}

struct LoginRequest: Encodable {
    var email: String
    var password: String
}

struct SignupRequest: Encodable {
    var email: String
    var password: String
    var displayName: String?
    var accountType: String?
    var timezone: String?

    enum CodingKeys: String, CodingKey {
        case email
        case password
        case displayName = "display_name"
        case accountType = "account_type"
        case timezone
    }
}

struct AuthTokenResponse: Decodable {
    var accessToken: String?
    var refreshToken: String?
    var expiresIn: Int?
    var requiresMFA: Bool?
    var mfaPendingToken: String?
    var mfaSetupRequired: Bool?
    var user: AuthUser?

    enum CodingKeys: String, CodingKey {
        case accessToken = "access_token"
        case refreshToken = "refresh_token"
        case expiresIn = "expires_in"
        case requiresMFA = "requires_mfa"
        case mfaPendingToken = "mfa_pending_token"
        case mfaSetupRequired = "mfa_setup_required"
        case user
    }
}

struct SamlStatusResponse: Decodable {
    var enabled: Bool
    var idp: SamlIdpInfo?

    struct SamlIdpInfo: Decodable {
        var id: String
        var label: String
        var forceSaml: Bool

        enum CodingKeys: String, CodingKey {
            case id
            case label
            case forceSaml
        }
    }
}

struct OidcStatusResponse: Decodable {
    var enabled: Bool
    var cleverEnabled: Bool?
    var classlinkEnabled: Bool?
    var clever: Bool?
    var classlink: Bool?
    var google: Bool?
    var microsoft: Bool?
    var apple: Bool?
    /// MOB.9 — native Sign in with Apple (AuthenticationServices), independent of tenant OIDC.
    var appleNative: Bool?
    /// MOB.9 — native Google Sign-In (Android Credential Manager); informational on iOS.
    var googleNative: Bool?
    var custom: [OidcCustomProvider]?

    struct OidcCustomProvider: Decodable {
        var id: String
        var displayName: String
    }

    var showsClever: Bool { cleverEnabled == true || clever == true }
    var showsClassLink: Bool { classlinkEnabled == true || classlink == true }
    var showsAppleNative: Bool { appleNative == true }
}

struct NativeAppleSignInRequest: Encodable {
    var idToken: String
    var rawNonce: String
    var authorizationCode: String?
    var fullName: String?
    var email: String?

    enum CodingKeys: String, CodingKey {
        case idToken = "id_token"
        case rawNonce = "raw_nonce"
        case authorizationCode = "authorization_code"
        case fullName = "full_name"
        case email
    }
}

struct NativeGoogleSignInRequest: Encodable {
    var idToken: String
    var rawNonce: String?

    enum CodingKeys: String, CodingKey {
        case idToken = "id_token"
        case rawNonce = "raw_nonce"
    }
}

struct MagicLinkRequest: Encodable {
    var email: String
    var redirectTo: String?

    enum CodingKeys: String, CodingKey {
        case email
        case redirectTo = "redirect_to"
    }
}

struct MagicLinkRequestResponse: Decodable {
    var message: String?
}

struct MagicLinkConsumeRequest: Encodable {
    var token: String
}

struct MfaTotpChallengeRequest: Encodable {
    var code: String
}

struct MfaTotpEnrolVerifyRequest: Encodable {
    var credentialId: String
    var code: String

    enum CodingKeys: String, CodingKey {
        case credentialId = "credential_id"
        case code
    }
}

struct MfaBackupChallengeRequest: Encodable {
    var code: String
}

struct MfaTotpEnrolResponse: Decodable {
    var credentialId: String?
    var otpauthUri: String?

    enum CodingKeys: String, CodingKey {
        case credentialId = "credential_id"
        case otpauthUri = "otpauth_uri"
    }
}

struct MfaWebAuthnBeginResponse: Decodable {
    var sessionId: String?
    var options: Data?

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case options
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        sessionId = try container.decodeIfPresent(String.self, forKey: .sessionId)
        if let nested = try? container.decode([String: JSONAny].self, forKey: .options) {
            options = try JSONEncoder().encode(nested)
        } else {
            options = nil
        }
    }
}

/// Decodes arbitrary JSON subtrees for WebAuthn option payloads.
private struct JSONAny: Codable {
    let value: Any

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if let value = try? container.decode(Bool.self) {
            self.value = value
        } else if let value = try? container.decode(Int.self) {
            self.value = value
        } else if let value = try? container.decode(Double.self) {
            self.value = value
        } else if let value = try? container.decode(String.self) {
            self.value = value
        } else if let value = try? container.decode([String: JSONAny].self) {
            self.value = value.mapValues(\.value)
        } else if let value = try? container.decode([JSONAny].self) {
            self.value = value.map(\.value)
        } else {
            self.value = NSNull()
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch value {
        case let value as Bool:
            try container.encode(value)
        case let value as Int:
            try container.encode(value)
        case let value as Double:
            try container.encode(value)
        case let value as String:
            try container.encode(value)
        case let value as [String: Any]:
            try container.encode(value.mapValues { JSONAny(value: $0) })
        case let value as [Any]:
            try container.encode(value.map { JSONAny(value: $0) })
        default:
            try container.encodeNil()
        }
    }

    init(value: Any) {
        self.value = value
    }
}

struct RefreshRequest: Encodable {
    var refreshToken: String

    enum CodingKeys: String, CodingKey {
        case refreshToken = "refresh_token"
    }
}

enum AuthAPI {
    private static let client = APIClient()

    static func fetchSamlStatus() async -> SamlStatusResponse {
        do {
            let (data, _) = try await client.request(path: "/api/v1/auth/saml/status")
            return (try? JSONDecoder().decode(SamlStatusResponse.self, from: data))
                ?? SamlStatusResponse(enabled: false, idp: nil)
        } catch {
            return SamlStatusResponse(enabled: false, idp: nil)
        }
    }

    static func fetchOidcStatus() async -> OidcStatusResponse {
        do {
            let (data, _) = try await client.request(path: "/api/v1/auth/oidc/status")
            return (try? JSONDecoder().decode(OidcStatusResponse.self, from: data))
                ?? OidcStatusResponse(
                    enabled: false,
                    cleverEnabled: false,
                    classlinkEnabled: false,
                    clever: false,
                    classlink: false,
                    google: false,
                    microsoft: false,
                    apple: false,
                    appleNative: false,
                    googleNative: false,
                    custom: []
                )
        } catch {
            return OidcStatusResponse(
                enabled: false,
                cleverEnabled: false,
                classlinkEnabled: false,
                clever: false,
                classlink: false,
                google: false,
                microsoft: false,
                apple: false,
                appleNative: false,
                googleNative: false,
                custom: []
            )
        }
    }

    static func nativeAppleSignIn(
        idToken: String,
        rawNonce: String,
        authorizationCode: String?,
        fullName: String?,
        email: String?
    ) async throws -> AuthTokenResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/auth/oidc/apple/native",
            method: "POST",
            body: NativeAppleSignInRequest(
                idToken: idToken,
                rawNonce: rawNonce,
                authorizationCode: authorizationCode,
                fullName: fullName,
                email: email
            )
        )
        return try JSONDecoder().decode(AuthTokenResponse.self, from: data)
    }

    static func nativeGoogleSignIn(idToken: String, rawNonce: String?) async throws -> AuthTokenResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/auth/oidc/google/native",
            method: "POST",
            body: NativeGoogleSignInRequest(idToken: idToken, rawNonce: rawNonce)
        )
        return try JSONDecoder().decode(AuthTokenResponse.self, from: data)
    }

    static func requestMagicLink(email: String) async throws -> MagicLinkRequestResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/auth/magic-link/request",
            method: "POST",
            body: MagicLinkRequest(email: email, redirectTo: "/")
        )
        return try JSONDecoder().decode(MagicLinkRequestResponse.self, from: data)
    }

    static func consumeMagicLink(token: String) async throws -> AuthTokenResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/auth/magic-link/consume",
            method: "POST",
            body: MagicLinkConsumeRequest(token: token)
        )
        return try JSONDecoder().decode(AuthTokenResponse.self, from: data)
    }

    static func mfaTotpChallenge(code: String, mfaPendingToken: String) async throws -> AuthTokenResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/auth/mfa/totp/challenge",
            method: "POST",
            body: MfaTotpChallengeRequest(code: code),
            authorized: true,
            accessToken: mfaPendingToken
        )
        return try JSONDecoder().decode(AuthTokenResponse.self, from: data)
    }

    static func mfaTotpEnrol(mfaPendingToken: String) async throws -> MfaTotpEnrolResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/auth/mfa/totp/enrol",
            method: "POST",
            authorized: true,
            accessToken: mfaPendingToken
        )
        return try JSONDecoder().decode(MfaTotpEnrolResponse.self, from: data)
    }

    static func mfaTotpVerifyEnrol(credentialId: String, code: String, mfaPendingToken: String) async throws {
        _ = try await client.request(
            path: "/api/v1/auth/mfa/totp/verify-enrol",
            method: "POST",
            body: MfaTotpEnrolVerifyRequest(credentialId: credentialId, code: code),
            authorized: true,
            accessToken: mfaPendingToken
        )
    }

    static func mfaSetupComplete(mfaPendingToken: String) async throws -> AuthTokenResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/auth/mfa/setup/complete",
            method: "POST",
            authorized: true,
            accessToken: mfaPendingToken
        )
        return try JSONDecoder().decode(AuthTokenResponse.self, from: data)
    }

    static func mfaBackupChallenge(code: String, mfaPendingToken: String) async throws -> AuthTokenResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/auth/mfa/backup/challenge",
            method: "POST",
            body: MfaBackupChallengeRequest(code: code),
            authorized: true,
            accessToken: mfaPendingToken
        )
        return try JSONDecoder().decode(AuthTokenResponse.self, from: data)
    }

    static func mfaWebAuthnBegin(setup: Bool, mfaPendingToken: String) async throws -> MfaWebAuthnBeginResponse {
        let path = setup
            ? "/api/v1/auth/mfa/webauthn/register/begin"
            : "/api/v1/auth/mfa/webauthn/authenticate/begin"
        let (data, _) = try await client.request(
            path: path,
            method: "POST",
            authorized: true,
            accessToken: mfaPendingToken
        )
        return try JSONDecoder().decode(MfaWebAuthnBeginResponse.self, from: data)
    }

    static func mfaWebAuthnComplete(
        setup: Bool,
        sessionId: String,
        credentialJSON: Data,
        mfaPendingToken: String
    ) async throws -> AuthTokenResponse? {
        let path = setup
            ? "/api/v1/auth/mfa/webauthn/register/complete"
            : "/api/v1/auth/mfa/webauthn/authenticate/complete"
        var body: [String: Any] = [
            "session_id": sessionId,
            "credential": try JSONSerialization.jsonObject(with: credentialJSON),
        ]
        if setup {
            body["display_name"] = ""
        }
        let bodyData = try JSONSerialization.data(withJSONObject: body)
        let (data, _) = try await client.requestRaw(
            path: path,
            method: "POST",
            bodyData: bodyData,
            authorized: true,
            accessToken: mfaPendingToken
        )
        if setup {
            return nil
        }
        return try JSONDecoder().decode(AuthTokenResponse.self, from: data)
    }

    /// Exchanges a refresh token for a new access token (+ rotated refresh token).
    static func refresh(refreshToken: String) async throws -> AuthTokenResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/auth/refresh",
            method: "POST",
            body: RefreshRequest(refreshToken: refreshToken)
        )
        return try JSONDecoder().decode(AuthTokenResponse.self, from: data)
    }

    static func fetchPasswordPolicy() async -> PasswordPolicy {
        do {
            let (data, _) = try await client.request(path: "/api/v1/auth/password-policy")
            return (try? JSONDecoder().decode(PasswordPolicy.self, from: data)) ?? .fallback
        } catch {
            return .fallback
        }
    }

    static func login(email: String, password: String) async throws -> AuthTokenResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/auth/login",
            method: "POST",
            body: LoginRequest(email: email, password: password)
        )
        return try JSONDecoder().decode(AuthTokenResponse.self, from: data)
    }

    static func signup(
        email: String,
        password: String,
        displayName: String?,
        registerAsParent: Bool,
        timezone: String?
    ) async throws -> AuthTokenResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/auth/signup",
            method: "POST",
            body: SignupRequest(
                email: email,
                password: password,
                displayName: displayName?.isEmpty == true ? nil : displayName,
                accountType: registerAsParent ? "parent" : nil,
                timezone: timezone?.isEmpty == true ? nil : timezone
            )
        )
        return try JSONDecoder().decode(AuthTokenResponse.self, from: data)
    }
}
