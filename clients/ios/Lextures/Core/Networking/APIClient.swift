import Foundation

struct APIClient {
    let session: URLSession

    init(session: URLSession? = nil) {
        self.session = session ?? NetworkBootstrap.makeSession()
    }

    func request(
        path: String,
        method: String = "GET",
        body: (any Encodable)? = nil,
        authorized: Bool = false,
        accessToken: String? = nil,
        idempotencyKey: String? = nil
    ) async throws -> (Data, HTTPURLResponse) {
        let bodyData: Data?
        if let body {
            bodyData = try JSONEncoder().encode(AnyEncodable(body))
        } else {
            bodyData = nil
        }
        return try await requestRaw(
            path: path,
            method: method,
            bodyData: bodyData,
            authorized: authorized,
            accessToken: accessToken,
            idempotencyKey: idempotencyKey
        )
    }

    /// Sends a request with a pre-encoded JSON body (used by the offline outbox replay path).
    func requestRaw(
        path: String,
        method: String = "GET",
        bodyData: Data? = nil,
        authorized: Bool = false,
        accessToken: String? = nil,
        idempotencyKey: String? = nil
    ) async throws -> (Data, HTTPURLResponse) {
        var request = URLRequest(url: AppConfiguration.apiURL(path: path))
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        request.setValue("ios", forHTTPHeaderField: "X-Platform")
        request.setValue(LocalePreferences.acceptLanguageHeaderValue(), forHTTPHeaderField: "Accept-Language")
        if let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String {
            request.setValue(version, forHTTPHeaderField: "X-App-Version")
        }

        if let bodyData {
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = bodyData
        }

        if authorized, let accessToken {
            request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
        }

        if let idempotencyKey, !idempotencyKey.isEmpty {
            request.setValue(idempotencyKey, forHTTPHeaderField: "X-Idempotency-Key")
        }

        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: request)
        } catch {
            throw APIError.transport(error)
        }

        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        guard (200 ... 299).contains(http.statusCode) else {
            let message = parseAPIErrorMessage(from: data)
            throw APIError.httpStatus(http.statusCode, message: message)
        }

        return (data, http)
    }
}

/// Type-erased Encodable wrapper for generic JSON bodies.
private struct AnyEncodable: Encodable {
    private let encode: (Encoder) throws -> Void

    init(_ value: any Encodable) {
        encode = value.encode
    }

    func encode(to encoder: Encoder) throws {
        try encode(encoder)
    }
}
