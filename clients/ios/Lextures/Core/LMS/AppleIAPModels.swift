import Foundation

// MARK: - Apple IAP API models (Path A / StoreKit 2)

struct AppleIAPProductsResponse: Codable {
    var configured: Bool?
    var bundleId: String?
    var products: [AppleIAPProductInfo]?
}

struct AppleIAPProductInfo: Codable, Hashable, Identifiable {
    var productId: String
    var kind: String
    var courseId: String?
    var displayHint: String?

    var id: String { productId }

    var isCoursePurchase: Bool { kind == "course_purchase" }
    var isSubscription: Bool { kind.hasPrefix("subscription") }
}

struct AppleIAPVerifyRequest: Encodable {
    var signedTransaction: String
    var courseId: String?
}

struct AppleIAPVerifyResponse: Codable {
    var entitlementId: String?
    var entitlementType: String?
    var courseId: String?
    var transactionId: String?
    var productId: String?
    var created: Bool?
    var alreadyOwned: Bool?
}
