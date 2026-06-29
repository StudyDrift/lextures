import Foundation

enum LocaleAPI {
    struct LocaleResponse: Decodable {
        var locale: String?
    }

    struct LocaleRequest: Encodable {
        var locale: String
    }

    /// Persists the user's locale preference on the server (web parity).
    static func saveLocale(_ tag: String, accessToken: String) async throws -> String {
        let (data, _) = try await APIClient().request(
            path: "/api/v1/settings/locale",
            method: "PUT",
            body: LocaleRequest(locale: tag),
            authorized: true,
            accessToken: accessToken
        )
        let response = try JSONDecoder().decode(LocaleResponse.self, from: data)
        return response.locale?.trimmingCharacters(in: .whitespacesAndNewlines).nonEmpty ?? tag
    }
}

private extension String {
    var nonEmpty: String? {
        isEmpty ? nil : self
    }
}
