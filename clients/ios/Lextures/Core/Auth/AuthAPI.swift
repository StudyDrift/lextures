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
        let c = try decoder.container(keyedBy: CodingKeys.self)
        minLength = try c.decodeIfPresent(Int.self, forKey: .minLength) ?? 8
        requireUpper = try c.decodeIfPresent(Bool.self, forKey: .requireUpper) ?? false
        requireLower = try c.decodeIfPresent(Bool.self, forKey: .requireLower) ?? false
        requireDigit = try c.decodeIfPresent(Bool.self, forKey: .requireDigit) ?? false
        requireSpecial = try c.decodeIfPresent(Bool.self, forKey: .requireSpecial) ?? false
        checkHibp = try c.decodeIfPresent(Bool.self, forKey: .checkHibp) ?? true
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

enum AuthAPI {
    private static let client = APIClient()

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
