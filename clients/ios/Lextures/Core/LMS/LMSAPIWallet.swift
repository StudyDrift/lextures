import Foundation

/// Credentials wallet: CCR, CE transcript, official transcripts (M12.2).
extension LMSAPI {
    static func fetchMyCCR(accessToken: String) async throws -> CCRSummaryResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/me/ccr",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.wallet.ccr.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CCRSummaryResponse.self, from: data)
    }

    static func generateMyCCR(sharePublicly: Bool, accessToken: String) async throws -> CCRGenerateResponse {
        let body = CCRGenerateRequest(sharePublicly: sharePublicly)
        let (data, response) = try await client.request(
            path: "/api/v1/me/ccr/generate",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CCRGenerateResponse.self, from: data)
    }

    static func ccrDownloadPath(documentId: String, format: String) -> String {
        "/api/v1/me/ccr/\(encodePath(documentId))/download?format=\(format)"
    }

    static func fetchCETranscript(accessToken: String) async throws -> CETranscriptResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/me/ce-transcript",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.wallet.ceTranscript.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CETranscriptResponse.self, from: data)
    }

    static func ceTranscriptPdfPath() -> String {
        "/api/v1/me/ce-transcript?format=pdf"
    }

    static func fetchTranscriptRequests(accessToken: String) async throws -> [TranscriptRequestSummary] {
        let (data, response) = try await client.request(
            path: "/api/v1/transcripts/requests",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.wallet.officialTranscripts.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(TranscriptRequestsResponse.self, from: data).requests ?? []
    }

    static func fetchTranscriptsConfig(accessToken: String) async throws -> TranscriptsStudentConfig {
        let (data, response) = try await client.request(
            path: "/api/v1/transcripts/config",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(TranscriptsStudentConfig.self, from: data)
    }
}
