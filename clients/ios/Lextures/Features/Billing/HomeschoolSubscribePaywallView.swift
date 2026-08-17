import StoreKit
import SwiftUI
import UIKit

/// Full-screen subscribe prompt for self.lextures.com when the learner has no active subscription.
/// Includes Guideline 3.1.2(c) disclosures: title, length, price (plus per-week), and
/// functional Privacy Policy + Terms of Use (EULA) links in a sticky footer.
struct HomeschoolSubscribePaywallView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    var onSubscribed: (() -> Void)?
    var onDismiss: (() -> Void)?

    @State private var monthly: Product?
    @State private var annual: Product?
    @State private var selected: HomeschoolPaywallLogic.PlanKind?
    @State private var loading = true
    @State private var purchasing = false
    @State private var errorMessage: String?
    @State private var requestedProductIDs: [String] = []

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            VStack(spacing: 0) {
                chrome
                PaywallPlansPage(
                    colorScheme: colorScheme,
                    loading: loading,
                    purchasing: purchasing,
                    errorMessage: errorMessage,
                    monthly: monthly,
                    annual: annual,
                    selected: selected,
                    requestedProductIDs: requestedProductIDs,
                    selectedProduct: selectedProduct,
                    ctaTitle: ctaTitle,
                    onSelect: selectPlan,
                    onPurchase: {
                        guard let product = selectedProduct else { return }
                        Task { await purchase(product) }
                    },
                    onRestore: { Task { await restore() } },
                    onRetry: { Task { await loadProducts() } },
                    restoreDisabled: purchasing || session.accessToken == nil
                )
                PaywallLegalFooter(
                    selectedProduct: selectedProduct,
                    colorScheme: colorScheme
                )
            }
        }
        .interactiveDismissDisabled(false)
        .task { await loadProducts() }
    }

    private var chrome: some View {
        HStack {
            Spacer()
            Button(L.text("mobile.common.close")) {
                onDismiss?()
            }
            .font(.subheadline.weight(.semibold))
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            .frame(minHeight: 44)
            .accessibilityIdentifier("paywall.close")
        }
        .padding(.horizontal, 16)
        .padding(.top, 8)
    }

    private var selectedProduct: Product? {
        switch selected {
        case .monthly: return monthly
        case .annual: return annual
        case nil: return nil
        }
    }

    private var ctaTitle: String {
        if purchasing { return L.text("mobile.billing.startingCheckout") }
        if let name = selectedProduct?.displayName, !name.isEmpty {
            return L.format("mobile.billing.paywall.ctaPlan", name)
        }
        return L.text("mobile.billing.paywall.cta")
    }

    private func selectPlan(_ kind: HomeschoolPaywallLogic.PlanKind) {
        Haptics.trigger(.selection)
        selected = kind
    }

    private func applyLoaded(_ loaded: [Product]) {
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
        selected = HomeschoolPaywallLogic.defaultPlan(
            hasMonthly: monthly != nil,
            hasAnnual: annual != nil
        )
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
            applyLoaded(loaded)
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
            Haptics.trigger(.success)
            onSubscribed?()
        } catch let err as StoreKitPurchaseService.PurchaseError {
            if case .userCancelled = err { return }
            Haptics.trigger(.error)
            errorMessage = err.errorDescription ?? L.text("mobile.billing.checkoutError")
        } catch {
            Haptics.trigger(.error)
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
                Haptics.trigger(.success)
                onSubscribed?()
            } else {
                errorMessage = L.text("mobile.billing.iap.restoreFailed")
            }
        } catch {
            errorMessage = L.text("mobile.billing.iap.restoreFailed")
        }
    }
}

// MARK: - Pages

private struct PaywallValueHeader: View {
    let colorScheme: ColorScheme

    private let benefits: [(icon: String, key: String)] = [
        ("sparkles", "mobile.billing.paywall.benefit.adaptive"),
        ("clock.arrow.2.circlepath", "mobile.billing.paywall.benefit.review"),
        ("books.vertical", "mobile.billing.paywall.benefit.courses"),
        ("iphone", "mobile.billing.paywall.benefit.devices"),
    ]

    var body: some View {
        VStack(spacing: 20) {
            VStack(spacing: 14) {
                Image("PaywallLogo")
                    .resizable()
                    .renderingMode(.original)
                    .scaledToFit()
                    .frame(maxHeight: 96)
                    .accessibilityLabel("Lextures")
                    .padding(.top, 4)

                Text(L.text("mobile.billing.paywall.headline"))
                    .font(LexturesTheme.displayFont(28))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: .infinity)

                Text(L.text("mobile.billing.paywall.subtitle"))
                    .font(.body)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .multilineTextAlignment(.center)
                    .fixedSize(horizontal: false, vertical: true)
            }

            VStack(alignment: .leading, spacing: 12) {
                ForEach(benefits, id: \.key) { item in
                    HStack(alignment: .top, spacing: 12) {
                        Image(systemName: item.icon)
                            .font(.body.weight(.semibold))
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                            .frame(width: 28, height: 28)
                        Text(L.text(String.LocalizationValue(stringLiteral: item.key)))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            .fixedSize(horizontal: false, vertical: true)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)

            Text(L.text("mobile.billing.paywall.cancelAnytime"))
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .multilineTextAlignment(.center)
                .frame(maxWidth: .infinity)
        }
    }
}

private struct PaywallPlansPage: View {
    let colorScheme: ColorScheme
    let loading: Bool
    let purchasing: Bool
    let errorMessage: String?
    let monthly: Product?
    let annual: Product?
    let selected: HomeschoolPaywallLogic.PlanKind?
    let requestedProductIDs: [String]
    let selectedProduct: Product?
    let ctaTitle: String
    let onSelect: (HomeschoolPaywallLogic.PlanKind) -> Void
    let onPurchase: () -> Void
    let onRestore: () -> Void
    let onRetry: () -> Void
    let restoreDisabled: Bool

    var body: some View {
        ScrollView {
            VStack(spacing: 20) {
                PaywallValueHeader(colorScheme: colorScheme)

                if let errorMessage {
                    LMSErrorBanner(message: errorMessage)
                }

                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 24)
                } else {
                    planList
                    PaywallCompareTable(
                        colorScheme: colorScheme,
                        savingsPercent: savingsPercent
                    )
                    PaywallPrimaryButton(
                        title: ctaTitle,
                        disabled: purchasing || selectedProduct == nil,
                        action: onPurchase
                    )
                    .accessibilityIdentifier("paywall.subscribe")
                }

                Button(action: onRestore) {
                    Text(L.text("mobile.billing.iap.restore"))
                        .font(.subheadline.weight(.semibold))
                        .frame(maxWidth: .infinity)
                        .frame(minHeight: 44)
                }
                .buttonStyle(.borderless)
                .disabled(restoreDisabled)
            }
            .padding(24)
            .frame(maxWidth: 520)
            .frame(maxWidth: .infinity)
        }
    }

    @ViewBuilder
    private var planList: some View {
        VStack(spacing: 12) {
            if let annual {
                PaywallPlanCard(
                    product: annual,
                    kind: .annual,
                    selected: selected == .annual,
                    savingsPercent: savingsPercent,
                    colorScheme: colorScheme,
                    onSelect: { onSelect(.annual) }
                )
            }
            if let monthly {
                PaywallPlanCard(
                    product: monthly,
                    kind: .monthly,
                    selected: selected == .monthly,
                    savingsPercent: nil,
                    colorScheme: colorScheme,
                    onSelect: { onSelect(.monthly) }
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
                Button(L.text("mobile.billing.paywall.retry"), action: onRetry)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.primary)
            }
        }
    }

    private var savingsPercent: Int? {
        guard let monthly, let annual else { return nil }
        return HomeschoolPaywallLogic.annualSavingsPercent(
            monthlyPrice: monthly.price,
            annualPrice: annual.price
        )
    }
}

private struct PaywallPlanCard: View {
    let product: Product
    let kind: HomeschoolPaywallLogic.PlanKind
    let selected: Bool
    let savingsPercent: Int?
    let colorScheme: ColorScheme
    let onSelect: () -> Void

    var body: some View {
        Button(action: onSelect) {
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text(badge)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.primary)
                    if let savingsPercent {
                        Text(L.format("mobile.billing.paywall.savePercent", savingsPercent))
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 3)
                            .background(LexturesTheme.primary)
                            .clipShape(Capsule())
                    }
                    Spacer()
                    Image(systemName: selected ? "checkmark.circle.fill" : "circle")
                        .foregroundStyle(selected ? LexturesTheme.primary : LexturesTheme.textSecondary(for: colorScheme))
                }

                Text(product.displayName)
                    .font(.headline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Text(product.displayPrice)
                        .font(.title3.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    if let period {
                        Text(periodLabel(period))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }

                if let weekly = weeklyPriceLabel {
                    Text(L.format("mobile.billing.paywall.perWeek", weekly))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                Text(L.text("mobile.billing.paywall.autoRenews"))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .padding(16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .fill(LexturesTheme.cardBackground(for: colorScheme))
            )
            .overlay(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .strokeBorder(
                        selected ? LexturesTheme.primary : LexturesTheme.primary.opacity(0.25),
                        lineWidth: selected ? 2 : 1
                    )
            )
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(selected ? .isSelected : [])
        .accessibilityLabel(accessibilityLabel)
        .accessibilityIdentifier(kind == .annual ? "paywall.plan.annual" : "paywall.plan.monthly")
    }

    private var badge: String {
        switch kind {
        case .monthly: return L.text("mobile.billing.paywall.monthlyBadge")
        case .annual: return L.text("mobile.billing.paywall.annualBadge")
        }
    }

    private var period: HomeschoolPaywallLogic.Period? {
        PaywallStoreKit.period(from: product)
    }

    private var weeklyPriceLabel: String? {
        guard let period,
              let weekly = HomeschoolPaywallLogic.weeklyPrice(price: product.price, period: period)
        else { return nil }
        return weekly.formatted(product.priceFormatStyle)
    }

    private var accessibilityLabel: String {
        var parts = [badge, product.displayName, product.displayPrice]
        if let period {
            parts.append(periodLabel(period))
        }
        if let weeklyPriceLabel {
            parts.append(L.format("mobile.billing.paywall.perWeek", weeklyPriceLabel))
        }
        return parts.joined(separator: ", ")
    }

    private func periodLabel(_ period: HomeschoolPaywallLogic.Period) -> String {
        L.format(
            String.LocalizationValue(stringLiteral: HomeschoolPaywallLogic.periodLabelKey(for: period.unit)),
            period.value
        )
    }
}

private struct PaywallCompareTable: View {
    let colorScheme: ColorScheme
    let savingsPercent: Int?

    private var rows: [(feature: String, without: String, with: String)] {
        [
            (
                L.text("mobile.billing.paywall.compare.adaptive"),
                L.text("mobile.billing.paywall.compare.limited"),
                L.text("mobile.billing.paywall.compare.included")
            ),
            (
                L.text("mobile.billing.paywall.compare.courses"),
                L.text("mobile.billing.paywall.compare.limited"),
                L.text("mobile.billing.paywall.compare.included")
            ),
            (
                L.text("mobile.billing.paywall.compare.devices"),
                L.text("mobile.billing.paywall.compare.limited"),
                L.text("mobile.billing.paywall.compare.included")
            ),
        ]
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(L.text("mobile.billing.paywall.compareTitle"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            VStack(spacing: 0) {
                HStack {
                    Text(L.text("mobile.billing.paywall.compare.feature"))
                        .frame(maxWidth: .infinity, alignment: .leading)
                    Text(L.text("mobile.billing.paywall.compare.without"))
                        .frame(width: 88, alignment: .center)
                    Text(L.text("mobile.billing.paywall.compare.with"))
                        .frame(width: 88, alignment: .center)
                }
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .padding(.vertical, 8)

                ForEach(rows, id: \.feature) { row in
                    Divider().opacity(0.5)
                    HStack(alignment: .center) {
                        Text(row.feature)
                            .frame(maxWidth: .infinity, alignment: .leading)
                        Text(row.without)
                            .frame(width: 88, alignment: .center)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        Text(row.with)
                            .frame(width: 88, alignment: .center)
                            .foregroundStyle(LexturesTheme.primary)
                            .fontWeight(.semibold)
                    }
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .padding(.vertical, 8)
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 4)
            .background(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .fill(LexturesTheme.cardBackground(for: colorScheme))
            )

            if savingsPercent != nil {
                Text(L.text("mobile.billing.paywall.cancelAnytime"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
        .accessibilityElement(children: .combine)
    }
}

private struct PaywallPrimaryButton: View {
    let title: String
    let disabled: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 8) {
                Text(title)
                    .font(.headline)
                Image(systemName: "chevron.right")
                    .font(.footnote.weight(.bold))
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .foregroundStyle(.white)
            .background(LexturesTheme.primary.opacity(disabled ? 0.55 : 1))
            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
        }
        .buttonStyle(.plain)
        .disabled(disabled)
        .frame(minHeight: 48)
    }
}

/// Always-visible 3.1.2(c) legal row. Uses Safari so first-party hosts are not
/// swallowed as in-app deep links.
private struct PaywallLegalFooter: View {
    let selectedProduct: Product?
    let colorScheme: ColorScheme

    var body: some View {
        VStack(spacing: 10) {
            HStack(spacing: 16) {
                legalLink(
                    title: L.text("mobile.billing.paywall.privacy"),
                    url: HomeschoolPaywallLogic.privacyPolicyURL,
                    identifier: "paywall.privacyPolicy"
                )
                Text("·")
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                legalLink(
                    title: L.text("mobile.billing.paywall.terms"),
                    url: HomeschoolPaywallLogic.termsOfUseURL,
                    identifier: "paywall.termsOfUse"
                )
            }
            .frame(maxWidth: .infinity)
            .accessibilityElement(children: .contain)
            .accessibilityLabel(L.text("mobile.billing.paywall.legalLinksA11y"))

            Text(renewalText)
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .multilineTextAlignment(.center)
                .fixedSize(horizontal: false, vertical: true)
                .frame(maxWidth: .infinity)
                .accessibilityIdentifier("paywall.renewalDisclosure")
        }
        .padding(.horizontal, 20)
        .padding(.top, 10)
        .padding(.bottom, 16)
        .background(LexturesTheme.sceneBackground(for: colorScheme))
    }

    private var renewalText: String {
        guard let product = selectedProduct,
              let period = PaywallStoreKit.period(from: product)
        else {
            return L.text("mobile.billing.paywall.legal")
        }
        let length = L.format(
            String.LocalizationValue(stringLiteral: HomeschoolPaywallLogic.periodLabelKey(for: period.unit)),
            period.value
        )
        return L.format(
            "mobile.billing.paywall.renewal",
            product.displayName,
            product.displayPrice,
            length
        )
    }

    private func legalLink(title: String, url: URL, identifier: String) -> some View {
        Button {
            UIApplication.shared.open(url)
        } label: {
            Text(title)
                .font(.footnote.weight(.semibold))
                .underline()
                .foregroundStyle(LexturesTheme.primary)
                .frame(minHeight: 44)
        }
        .buttonStyle(.plain)
        .accessibilityIdentifier(identifier)
        .accessibilityHint(url.absoluteString)
    }
}

private enum PaywallStoreKit {
    static func period(from product: Product) -> HomeschoolPaywallLogic.Period? {
        guard let period = product.subscription?.subscriptionPeriod else { return nil }
        let unit: HomeschoolPaywallLogic.PeriodUnit
        switch period.unit {
        case .day: unit = .day
        case .week: unit = .week
        case .month: unit = .month
        case .year: unit = .year
        @unknown default: return nil
        }
        return HomeschoolPaywallLogic.Period(value: period.value, unit: unit)
    }
}
