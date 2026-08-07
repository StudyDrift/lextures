import Foundation
import StoreKit

/// StoreKit 2 purchase helper for digital courses and subscriptions (App Store 3.1.1 Path A).
@MainActor
enum StoreKitPurchaseService {
    enum PurchaseError: LocalizedError {
        case productNotFound
        case verificationFailed
        case userCancelled
        case pending
        case serverVerifyFailed
        case notConfigured

        var errorDescription: String? {
            switch self {
            case .productNotFound:
                return L.text("mobile.billing.iap.productNotFound")
            case .verificationFailed:
                return L.text("mobile.billing.iap.verificationFailed")
            case .userCancelled:
                return L.text("mobile.billing.iap.cancelled")
            case .pending:
                return L.text("mobile.billing.iap.pending")
            case .serverVerifyFailed:
                return L.text("mobile.billing.iap.serverVerifyFailed")
            case .notConfigured:
                return L.text("mobile.billing.iap.notConfigured")
            }
        }
    }

    struct PurchaseResult: Equatable {
        var productId: String
        var transactionId: String
        var alreadyOwned: Bool
        var courseId: String?
    }

    /// Loads StoreKit products for the given App Store product identifiers.
    static func loadProducts(ids: [String]) async throws -> [Product] {
        let unique = Array(Set(ids.filter { !$0.isEmpty }))
        guard !unique.isEmpty else { return [] }
        return try await Product.products(for: unique)
    }

    /// Purchases a StoreKit product and posts the signed transaction to Lextures for entitlement grant.
    static func purchase(
        product: Product,
        appAccountToken: UUID?,
        courseId: String?,
        accessToken: String
    ) async throws -> PurchaseResult {
        var options: Set<Product.PurchaseOption> = []
        if let appAccountToken {
            options.insert(.appAccountToken(appAccountToken))
        }

        let result = try await product.purchase(options: options)
        switch result {
        case .success(let verification):
            let transaction = try checkVerified(verification)
            let signedJWS = verification.jwsRepresentation
            let verify: AppleIAPVerifyResponse
            do {
                verify = try await LMSAPI.verifyAppleIAPPurchase(
                    signedTransaction: signedJWS,
                    courseId: courseId,
                    accessToken: accessToken
                )
            } catch {
                throw PurchaseError.serverVerifyFailed
            }
            await transaction.finish()
            return PurchaseResult(
                productId: product.id,
                transactionId: String(transaction.id),
                alreadyOwned: verify.alreadyOwned == true || verify.created == false,
                courseId: verify.courseId ?? courseId
            )
        case .userCancelled:
            throw PurchaseError.userCancelled
        case .pending:
            throw PurchaseError.pending
        @unknown default:
            throw PurchaseError.verificationFailed
        }
    }

    /// Restores entitlements for the current Apple ID by re-verifying with the server.
    static func restorePurchases(accessToken: String) async throws -> Int {
        var granted = 0
        for await entitlement in Transaction.currentEntitlements {
            guard case .verified(let transaction) = entitlement else { continue }
            let signedJWS = entitlement.jwsRepresentation
            do {
                _ = try await LMSAPI.verifyAppleIAPPurchase(
                    signedTransaction: signedJWS,
                    courseId: nil,
                    accessToken: accessToken
                )
                granted += 1
                await transaction.finish()
            } catch {
                continue
            }
        }
        return granted
    }

    /// Syncs unfinished transactions to the backend when authenticated, then finishes them.
    static func syncUnfinished(accessToken: String?) async {
        for await result in Transaction.unfinished {
            guard case .verified(let transaction) = result else { continue }
            if let accessToken {
                do {
                    _ = try await LMSAPI.verifyAppleIAPPurchase(
                        signedTransaction: result.jwsRepresentation,
                        courseId: nil,
                        accessToken: accessToken
                    )
                } catch {
                    continue
                }
            }
            await transaction.finish()
        }
    }

    private static func checkVerified<T>(_ result: VerificationResult<T>) throws -> T {
        switch result {
        case .unverified:
            throw PurchaseError.verificationFailed
        case .verified(let safe):
            return safe
        }
    }
}
