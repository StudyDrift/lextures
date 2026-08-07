import Foundation

/// Apple In-App Purchase product catalog + server verification (Path A).
extension LMSAPI {
    static func fetchAppleIAPProducts(
        courseId: String?,
        accessToken: String
    ) async throws -> AppleIAPProductsResponse {
        var path = "/api/v1/billing/apple/products"
        if let courseId, !courseId.isEmpty {
            path += "?courseId=\(encodePath(courseId))"
        }
        let (data, response) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AppleIAPProductsResponse.self, from: data)
    }

    static func verifyAppleIAPPurchase(
        signedTransaction: String,
        courseId: String?,
        accessToken: String
    ) async throws -> AppleIAPVerifyResponse {
        let body = AppleIAPVerifyRequest(
            signedTransaction: signedTransaction,
            courseId: courseId
        )
        let (data, response) = try await client.request(
            path: "/api/v1/billing/apple/verify",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AppleIAPVerifyResponse.self, from: data)
    }
}
