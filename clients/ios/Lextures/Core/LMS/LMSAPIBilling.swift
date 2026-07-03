import Foundation

/// Stripe checkout, entitlements, and billing portal (M9.2).
extension LMSAPI {
    static func fetchMyEntitlements(accessToken: String) async throws -> [BillingEntitlement] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/entitlements",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return [] }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BillingEntitlementsResponse.self, from: data).entitlements ?? []
    }

    static func fetchMyTransactions(accessToken: String) async throws -> [BillingTransaction] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/transactions",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return [] }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BillingTransactionsResponse.self, from: data).transactions ?? []
    }

    static func startCheckout(
        courseId: String,
        successUrl: String,
        cancelUrl: String,
        usePaymentsAbstraction: Bool,
        accessToken: String
    ) async throws -> CheckoutSessionResponse {
        let path = BillingLogic.checkoutEndpoint(usePaymentsAbstraction: usePaymentsAbstraction)
        let body = CheckoutSessionRequest(
            courseId: courseId,
            successUrl: successUrl,
            cancelUrl: cancelUrl
        )
        let (data, response) = try await client.request(
            path: path,
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CheckoutSessionResponse.self, from: data)
    }

    static func fetchCheckoutQuote(
        courseId: String,
        accessToken: String
    ) async throws -> CheckoutTaxQuote {
        let body = CheckoutTaxQuoteRequest(courseId: courseId)
        let (data, response) = try await client.request(
            path: "/api/v1/checkout/quote",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CheckoutTaxQuote.self, from: data)
    }

    static func openBillingPortal(returnUrl: String?, accessToken: String) async throws -> String {
        var path = "/api/v1/billing/portal"
        if let returnUrl, !returnUrl.isEmpty,
           let encoded = returnUrl.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) {
            path += "?return_url=\(encoded)"
        }
        let (data, response) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BillingPortalResponse.self, from: data).portalUrl
    }

    static func checkEntitlement(
        userId: String,
        courseId: String,
        accessToken: String
    ) async throws -> Bool {
        let path =
            "/api/v1/internal/entitlements/check?user_id=\(encodePath(userId))&course_id=\(encodePath(courseId))"
        let (data, response) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else { return false }
        return try decode(EntitlementCheckResponse.self, from: data).entitled == true
    }
}