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
        accessToken: String? = nil
    ) async throws -> (Data, HTTPURLResponse) {
        var request = URLRequest(url: AppConfiguration.apiURL(path: path))
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Accept")

        if let body {
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONEncoder().encode(AnyEncodable(body))
        }

        if authorized, let accessToken {
            request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
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
