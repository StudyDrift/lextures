import StoreKit
import SwiftUI

/// Full-screen subscribe prompt for self.lextures.com when the learner has no active subscription.
struct HomeschoolSubscribePaywallView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    var onSubscribed: (() -> Void)?
    var onDismiss: (() -> Void)?

    @State private var products: [Product] = []
    @State private var monthly: Product?
    @State private var annual: Product?
    @State private var loading = true
    @State private var purchasing = false
    @State private var errorMessage: String?
    @State private var requestedProductIDs: [String] = []

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(spacing: 24) {
                    HStack {
                        Spacer()
                        Button(L.text("mobile.common.close")) {
                            onDismiss?()
                        }
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }

                    VStack(spacing: 16) {
                        Image("PaywallLogo")
                            .resizable()
                            .renderingMode(.original)
                            .scaledToFit()
                            .frame(maxHeight: 120)
                            .accessibilityLabel("Lextures")
                            .padding(.top, 8)

                        Text(L.text("mobile.billing.paywall.title"))
                            .font(LexturesTheme.displayFont(28))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            .multilineTextAlignment(.center)
                            .frame(maxWidth: .infinity)

                        Text(L.text("mobile.billing.paywall.message"))
                            .font(.body)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            .multilineTextAlignment(.center)
                            .fixedSize(horizontal: false, vertical: true)
                    }

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 24)
                    } else {
                        VStack(spacing: 12) {
                            if let monthly {
                                planButton(
                                    product: monthly,
                                    badge: L.text("mobile.billing.paywall.monthlyBadge")
                                )
                            }
                            if let annual {
                                planButton(
                                    product: annual,
                                    badge: L.text("mobile.billing.paywall.annualBadge")
                                )
                            }
                            if monthly == nil, annual == nil {
                                Text(L.text("mobile.billing.iap.notConfigured"))
                                    .font(.subheadline)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    .multilineTextAlignment(.center)
                                #if DEBUG
                                if !requestedProductIDs.isEmpty {
                                    Text(requestedProductIDs.joined(separator: "\n"))
                                        .font(.caption.monospaced())
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                        .multilineTextAlignment(.center)
                                }
                                #endif
                                Button(L.text("mobile.billing.paywall.retry")) {
                                    Task { await loadProducts() }
                                }
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(LexturesTheme.primary)
                            }
                        }
                    }

                    Button {
                        Task { await restore() }
                    } label: {
                        Text(L.text("mobile.billing.iap.restore"))
                            .font(.subheadline.weight(.semibold))
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderless)
                    .disabled(purchasing || session.accessToken == nil)

                    Text(L.text("mobile.billing.paywall.legal"))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .multilineTextAlignment(.center)
                        .frame(maxWidth: .infinity)
                }
                .padding(24)
                .frame(maxWidth: 520)
                .frame(maxWidth: .infinity)
            }
        }
        .interactiveDismissDisabled(false)
        .task { await loadProducts() }
    }

    @ViewBuilder
    private func planButton(product: Product, badge: String) -> some View {
        Button {
            Task { await purchase(product) }
        } label: {
            VStack(alignment: .leading, spacing: 6) {
                Text(badge)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.primary)
                HStack {
                    Text(product.displayName)
                        .font(.headline)
                    Spacer()
                    Text(product.displayPrice)
                        .font(.headline)
                }
                if let period = product.subscription?.subscriptionPeriod {
                    Text(periodLabel(period))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            .padding(16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .fill(LexturesTheme.cardBackground(for: colorScheme))
            )
            .overlay(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .strokeBorder(LexturesTheme.primary.opacity(0.35), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
        .disabled(purchasing)
        .accessibilityLabel("\(badge), \(product.displayName), \(product.displayPrice)")
    }

    private func periodLabel(_ period: Product.SubscriptionPeriod) -> String {
        switch period.unit {
        case .month:
            return L.format("mobile.billing.paywall.everyMonths", period.value)
        case .year:
            return L.format("mobile.billing.paywall.everyYears", period.value)
        case .day:
            return L.format("mobile.billing.paywall.everyDays", period.value)
        case .week:
            return L.format("mobile.billing.paywall.everyWeeks", period.value)
        @unknown default:
            return ""
        }
    }

    private func loadProducts() async {
        guard let token = session.accessToken else {
            loading = false
            errorMessage = L.text("mobile.billing.checkoutError")
            return
        }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let response = try await LMSAPI.fetchAppleIAPProducts(courseId: nil, accessToken: token)
            let infos = response.products ?? []
            let ids = BillingLogic.appleSubscriptionProductIDs(from: infos)
            requestedProductIDs = ids
            let loaded = try await StoreKitPurchaseService.loadProducts(ids: ids)
            #if DEBUG
            print("IAP requested \(ids) loaded \(loaded.map(\.id))")
            #endif
            products = loaded
            monthly = loaded.first {
                BillingLogic.isMonthlySubscriptionProduct(id: $0.id)
                    || $0.subscription?.subscriptionPeriod.unit == .month
            }
            annual = loaded.first {
                BillingLogic.isAnnualSubscriptionProduct(id: $0.id)
                    || $0.subscription?.subscriptionPeriod.unit == .year
            }
            if monthly == nil { monthly = loaded.first }
            if annual == nil, loaded.count > 1 { annual = loaded.dropFirst().first }
        } catch {
            errorMessage = L.text("mobile.billing.iap.productNotFound")
        }
    }

    private func purchase(_ product: Product) async {
        guard let token = session.accessToken else { return }
        purchasing = true
        errorMessage = nil
        defer { purchasing = false }
        let accountToken = UUID(uuidString: shell.profile?.id ?? "")
        do {
            _ = try await StoreKitPurchaseService.purchase(
                product: product,
                appAccountToken: accountToken,
                courseId: nil,
                accessToken: token
            )
            onSubscribed?()
        } catch let err as StoreKitPurchaseService.PurchaseError {
            if case .userCancelled = err { return }
            errorMessage = err.errorDescription ?? L.text("mobile.billing.checkoutError")
        } catch {
            errorMessage = L.text("mobile.billing.checkoutError")
        }
    }

    private func restore() async {
        guard let token = session.accessToken else { return }
        purchasing = true
        errorMessage = nil
        defer { purchasing = false }
        do {
            let count = try await StoreKitPurchaseService.restorePurchases(accessToken: token)
            if count > 0 {
                onSubscribed?()
            } else {
                errorMessage = L.text("mobile.billing.iap.restoreFailed")
            }
        } catch {
            errorMessage = L.text("mobile.billing.iap.restoreFailed")
        }
    }
}
