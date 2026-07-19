import Foundation

/// Canvas course import API (MOB.2).
/// The Canvas access token is sent once per request and is never persisted by this client.
extension LMSAPI {
    static func fetchCanvasCourses(
        canvasBaseUrl: String,
        accessToken: String,
        sessionAccessToken: String
    ) async throws -> [CanvasCourseListItem] {
        let body = CanvasListCoursesRequest(
            canvasBaseUrl: CanvasImportLogic.normalizeBaseURL(canvasBaseUrl),
            accessToken: accessToken.trimmingCharacters(in: .whitespacesAndNewlines)
        )
        let (data, _) = try await client.request(
            path: "/api/v1/integrations/canvas/courses",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: sessionAccessToken
        )
        return try decode(CanvasCoursesResponse.self, from: data).courses ?? []
    }

    /// Queues a Canvas import and streams progress over the job WebSocket until complete/error/cancel.
    static func postCourseImportCanvas(
        courseCode: String,
        body: PostCourseImportCanvasRequest,
        sessionAccessToken: String,
        onProgress: @escaping @MainActor (String) -> Void,
        isCancelled: @escaping () -> Bool
    ) async throws {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/import/canvas",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: sessionAccessToken
        )
        let queued = try decode(CanvasImportQueuedResponse.self, from: data)
        guard let jobId = queued.jobId?.trimmingCharacters(in: .whitespacesAndNewlines), !jobId.isEmpty else {
            throw CanvasImportLogic.CanvasImportError.missingJobId
        }
        let queuedMessage = queued.message?.trimmingCharacters(in: .whitespacesAndNewlines)
        await MainActor.run {
            onProgress(
                (queuedMessage?.isEmpty == false ? queuedMessage : nil)
                    ?? L.text("mobile.canvasImport.progress.queued")
            )
        }
        try await waitForCanvasImportJob(
            jobId: jobId,
            sessionAccessToken: sessionAccessToken,
            onProgress: onProgress,
            isCancelled: isCancelled
        )
    }

    @MainActor
    private static func waitForCanvasImportJob(
        jobId: String,
        sessionAccessToken: String,
        onProgress: @escaping @MainActor (String) -> Void,
        isCancelled: @escaping () -> Bool
    ) async throws {
        let path = CanvasImportLogic.jobWebSocketPath(jobId: jobId)
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
            final class Box: @unchecked Sendable {
                var settled = false
                var client: WebSocketClient?
            }
            let box = Box()

            let finish: (Result<Void, Error>) -> Void = { result in
                guard !box.settled else { return }
                box.settled = true
                box.client?.disconnect()
                box.client = nil
                continuation.resume(with: result)
            }

            let client = WebSocketClient(
                path: path,
                accessTokenProvider: { sessionAccessToken },
                onMessage: { data in
                    Task { @MainActor in
                        if isCancelled() {
                            finish(.failure(CanvasImportLogic.CanvasImportError.cancelled))
                            return
                        }
                        guard let message = CanvasImportLogic.parseWSMessage(from: data) else { return }
                        switch message.type {
                        case .progress:
                            if let text = message.message?.trimmingCharacters(in: .whitespacesAndNewlines), !text.isEmpty {
                                onProgress(text)
                            }
                        case .complete, .coursesUpdated:
                            finish(.success(()))
                        case .error:
                            let detail = message.message?.trimmingCharacters(in: .whitespacesAndNewlines)
                            finish(.failure(CanvasImportLogic.CanvasImportError.server(
                                (detail?.isEmpty == false ? detail : nil)
                                    ?? L.text("mobile.canvasImport.error.failed")
                            )))
                        case .unknown:
                            break
                        }
                    }
                },
                onLifecycle: { event in
                    Task { @MainActor in
                        if isCancelled() {
                            finish(.failure(CanvasImportLogic.CanvasImportError.cancelled))
                            return
                        }
                        switch event {
                        case .opened:
                            break
                        case .closed(_, let willReconnect):
                            if !willReconnect && !box.settled {
                                finish(.failure(CanvasImportLogic.CanvasImportError.connectionClosed))
                            }
                        }
                    }
                },
                stopOnPermanentRefusal: true
            )
            box.client = client
            client.connect()

            Task { @MainActor in
                while !box.settled {
                    if isCancelled() {
                        finish(.failure(CanvasImportLogic.CanvasImportError.cancelled))
                        return
                    }
                    try? await Task.sleep(for: .milliseconds(200))
                }
            }
        }
    }
}
