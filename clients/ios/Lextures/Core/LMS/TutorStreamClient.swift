import Foundation

/// Reads tutor/study-buddy SSE streams from POST endpoints.
struct TutorStreamClient {
    private let session: URLSession

    init(session: URLSession = NetworkBootstrap.makeSession()) {
        self.session = session
    }

    func stream(
        path: String,
        method: String = "POST",
        body: (any Encodable)?,
        accessToken: String
    ) -> AsyncThrowingStream<TutorStreamEvent, Error> {
        AsyncThrowingStream { continuation in
            let task = Task {
                do {
                    var request = URLRequest(url: AppConfiguration.apiURL(path: path))
                    request.httpMethod = method
                    request.setValue("text/event-stream", forHTTPHeaderField: "Accept")
                    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
                    request.setValue("ios", forHTTPHeaderField: "X-Platform")
                    request.setValue(LocalePreferences.acceptLanguageHeaderValue(), forHTTPHeaderField: "Accept-Language")
                    if let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String {
                        request.setValue(version, forHTTPHeaderField: "X-App-Version")
                    }
                    request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
                    if let body {
                        request.httpBody = try JSONEncoder().encode(TutorRequestBody(body))
                    }

                    let (bytes, response) = try await session.bytes(for: request)
                    guard let http = response as? HTTPURLResponse else {
                        throw APIError.invalidResponse
                    }
                    if !(200 ... 299).contains(http.statusCode) {
                        var data = Data()
                        for try await byte in bytes {
                            data.append(byte)
                        }
                        let message = TutorLogic.gracefulHttpMessage(
                            statusCode: http.statusCode,
                            body: parseAPIErrorMessage(from: data)
                        )
                        throw APIError.httpStatus(http.statusCode, message: message)
                    }

                    for try await line in bytes.lines {
                        try Task.checkCancellation()
                        if let event = TutorLogic.parseSSELine(line) {
                            continuation.yield(event)
                            switch event {
                            case .done, .error:
                                continuation.finish()
                                return
                            case .content:
                                continue
                            }
                        }
                    }
                    continuation.finish()
                } catch is CancellationError {
                    continuation.finish()
                } catch {
                    continuation.finish(throwing: error)
                }
            }
            continuation.onTermination = { _ in task.cancel() }
        }
    }
}

private struct TutorRequestBody: Encodable {
    private let value: any Encodable

    init(_ value: any Encodable) { self.value = value }

    func encode(to encoder: Encoder) throws {
        try value.encode(to: encoder)
    }
}