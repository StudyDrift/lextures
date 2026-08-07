import StoreKit
import SwiftUI

/// Paid course checkout sheet: StoreKit 2 In-App Purchase (Path A / App Store 3.1.1).
struct PurchaseFlowSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let courseId: String
    let courseCode: String
    let title: String
    let priceCents: Int
    let currency: String
    /// When set, used for analytics / already-owned handling (marketplace).
    var marketplaceSlug: String? = nil
    var onAlreadyOwned: (() -> Void)? = nil
    /// Called after a successful IAP + server verify (entitlement granted).
    var onPurchased: (() -> Void)? = nil

    @State private var appleProducts: [AppleIAPProductInfo] = []
    @State private var storeProduct: Product?
    @State private var loadingProducts = true
    @State private var purchasing = false
    @State private var errorMessage: String?
    @State private var iapConfigured = false

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }

                        LMSCard {
                            VStack(alignment: .leading, spacing: 8) {
                                Text(title)
                                    .font(.headline)
                                Text(L.text("mobile.billing.purchaseHint"))
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }

                        priceCard

                        Button {
                            Task { await purchase() }
                        } label: {
                            Text(purchaseButtonTitle)
                                .font(.headline)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 14)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(LexturesTheme.primary)
                        .disabled(purchasing || session.accessToken == nil || storeProduct == nil || loadingProducts)
                        .accessibilityLabel(L.text("mobile.billing.purchase"))
                        .accessibilityValue(displayPriceLabel)

                        Text(L.text("mobile.billing.storePolicyNote"))
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    .padding(16)
                }
            }
            .navigationTitle(L.text("mobile.billing.purchaseTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) { dismiss() }
                }
            }
            .task { await loadProducts() }
        }
    }

    private var purchaseButtonTitle: String {
        if loadingProducts {
            return L.text("mobile.billing.iap.loading")
        }
        if purchasing {
            return L.text("mobile.billing.startingCheckout")
        }
        return L.text("mobile.billing.purchase")
    }

    private var displayPriceLabel: String {
        if let storeProduct {
            return storeProduct.displayPrice
        }
        return BillingLogic.formatMoney(cents: priceCents, currency: currency)
    }

    @ViewBuilder
    private var priceCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                if loadingProducts {
                    ProgressView()
                        .frame(maxWidth: .infinity, alignment: .center)
                } else {
                    HStack {
                        Text(L.text("mobile.billing.total"))
                            .font(.headline)
                        Spacer()
                        Text(displayPriceLabel)
                            .font(.headline)
                    }
                    if storeProduct == nil {
                        Text(
                            iapConfigured
                                ? L.text("mobile.billing.iap.productNotFound")
                                : L.text("mobile.billing.iap.notConfigured")
                        )
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func loadProducts() async {
        guard let token = session.accessToken else {
            loadingProducts = false
            errorMessage = L.text("mobile.billing.checkoutError")
            return
        }
        loadingProducts = true
        errorMessage = nil
        defer { loadingProducts = false }
        do {
            let response = try await LMSAPI.fetchAppleIAPProducts(courseId: courseId, accessToken: token)
            iapConfigured = response.configured == true
            appleProducts = response.products ?? []
            guard let preferred = BillingLogic.preferredAppleProduct(from: appleProducts, courseId: courseId)
            else {
                storeProduct = nil
                return
            }
            let loaded = try await StoreKitPurchaseService.loadProducts(ids: [preferred.productId])
            storeProduct = loaded.first
            if storeProduct == nil {
                errorMessage = L.text("mobile.billing.iap.productNotFound")
            }
        } catch {
            errorMessage = L.text("mobile.billing.checkoutError")
            storeProduct = nil
        }
    }

    private func purchase() async {
        guard let token = session.accessToken, let product = storeProduct else { return }
        purchasing = true
        errorMessage = nil
        defer { purchasing = false }

        shell.pendingCheckout = PendingCheckoutContext(
            courseId: courseId,
            courseCode: courseCode,
            title: title
        )

        let accountToken = UUID(uuidString: shell.profile?.id ?? "")
        do {
            let result = try await StoreKitPurchaseService.purchase(
                product: product,
                appAccountToken: accountToken,
                courseId: courseId,
                accessToken: token
            )
            if marketplaceSlug != nil {
                MarketplaceObservability.record(
                    "marketplace_purchase_succeeded",
                    attributes: ["source": "apple_iap", "productId": result.productId]
                )
            }
            if result.alreadyOwned {
                shell.pendingCheckout = nil
                dismiss()
                onAlreadyOwned?()
                return
            }
            shell.pendingCheckout = nil
            onPurchased?()
            dismiss()
            // Route into course via checkout success handler surface.
            shell.checkoutReturnPhase = .success(courseId: courseId)
        } catch let err as StoreKitPurchaseService.PurchaseError {
            shell.pendingCheckout = nil
            if case .userCancelled = err {
                if marketplaceSlug != nil {
                    MarketplaceObservability.record("marketplace_cancelled", attributes: [:])
                }
                return
            }
            errorMessage = err.errorDescription ?? L.text("mobile.billing.checkoutError")
            if marketplaceSlug != nil {
                MarketplaceObservability.record("marketplace_purchase_failed", attributes: ["reason": "iap"])
            }
        } catch {
            shell.pendingCheckout = nil
            errorMessage = L.text("mobile.billing.checkoutError")
            if marketplaceSlug != nil {
                MarketplaceObservability.record("marketplace_purchase_failed", attributes: ["reason": "iap"])
            }
        }
    }
}
