import Foundation

// MARK: - Billing (M9.2)

struct BillingEntitlement: Codable, Identifiable, Hashable {
    var id: String
    var entitlementType: String
    var courseId: String?
    var amountPaidCents: Int
    var subtotalCents: Int?
    var taxAmountCents: Int?
    var taxType: String?
    var taxJurisdiction: String?
    var reverseCharge: Bool?
    var invoiceId: String?
    var currency: String
    var validFrom: String
    var validUntil: String?
    var status: String
}

struct BillingEntitlementsResponse: Codable {
    var entitlements: [BillingEntitlement]?
}

struct BillingTransaction: Codable, Identifiable, Hashable {
    var id: String
    var courseId: String?
    var provider: String
    var providerTxnId: String
    var amountCents: Int
    var currency: String
    var status: String
    var subscriptionId: String?
    var createdAt: String
}

struct BillingTransactionsResponse: Codable {
    var transactions: [BillingTransaction]?
}

struct CheckoutSessionRequest: Encodable {
    var courseId: String?
    var successUrl: String
    var cancelUrl: String
}

struct CheckoutSessionResponse: Decodable {
    var sessionId: String?
    var checkoutUrl: String
    var provider: String?
}

struct CheckoutTaxQuoteRequest: Encodable {
    var courseId: String
}

struct CheckoutTaxQuoteLine: Codable, Hashable {
    var label: String
    var amountCents: Int
}

struct CheckoutTaxQuote: Codable, Hashable {
    var subtotalCents: Int
    var taxAmountCents: Int
    var totalCents: Int
    var currency: String
    var taxType: String?
    var taxJurisdiction: String?
    var reverseCharge: Bool?
    var lines: [CheckoutTaxQuoteLine]
}

struct BillingPortalResponse: Decodable {
    var portalUrl: String
}

struct EntitlementCheckResponse: Decodable {
    var entitled: Bool?
}

struct PendingCheckoutContext: Equatable {
    var courseId: String
    var courseCode: String
    var title: String
}

enum CheckoutReturnPhase: Equatable {
    case success(courseId: String?)
    case cancel
}