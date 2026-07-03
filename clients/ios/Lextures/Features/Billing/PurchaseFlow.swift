import SwiftUI

/// Paid course checkout sheet: tax quote + Stripe web checkout handoff (M9.2).
struct PurchaseFlowSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL
    @Environment(\.dismiss) private var dismiss

    let courseId: String
    let courseCode: String
    let title: String
    let priceCents: Int
    let currency: String

    @State private var quote: CheckoutTaxQuote?
    @State private var loadingQuote = false
    @State private var purchasing = false
    @State private var errorMessage: String?

    private var displayTotalCents: Int {
        quote?.totalCents ?? priceCents
    }

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
                            Text(purchasing ? L.text("mobile.billing.startingCheckout") : L.text("mobile.billing.purchase"))
                                .font(.headline)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 14)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(LexturesTheme.primary)
                        .disabled(purchasing || session.accessToken == nil)

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
            .task { await loadQuote() }
        }
    }

    @ViewBuilder
    private var priceCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                if loadingQuote {
                    ProgressView()
                        .frame(maxWidth: .infinity, alignment: .center)
                } else if let quote {
                    ForEach(Array(BillingLogic.quoteLineItems(quote).enumerated()), id: \.offset) { _, row in
                        HStack {
                            Text(row.label)
                                .font(.subheadline)
                            Spacer()
                            Text(BillingLogic.formatMoney(cents: row.cents, currency: quote.currency))
                                .font(.subheadline.weight(.semibold))
                        }
                    }
                    Divider()
                    HStack {
                        Text(L.text("mobile.billing.total"))
                            .font(.headline)
                        Spacer()
                        Text(BillingLogic.formatMoney(cents: quote.totalCents, currency: quote.currency))
                            .font(.headline)
                    }
                    if let jurisdiction = quote.taxJurisdiction, !jurisdiction.isEmpty {
                        Text(L.format("mobile.billing.taxJurisdiction", jurisdiction))
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                } else {
                    HStack {
                        Text(L.text("mobile.billing.total"))
                            .font(.headline)
                        Spacer()
                        Text(BillingLogic.formatMoney(cents: priceCents, currency: currency))
                            .font(.headline)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func loadQuote() async {
        guard shell.platformFeatures.ffTaxCollection,
              let token = session.accessToken else { return }
        loadingQuote = true
        defer { loadingQuote = false }
        do {
            quote = try await LMSAPI.fetchCheckoutQuote(courseId: courseId, accessToken: token)
        } catch {
            quote = nil
        }
    }

    private func purchase() async {
        guard let token = session.accessToken else { return }
        purchasing = true
        errorMessage = nil
        defer { purchasing = false }
        do {
            shell.pendingCheckout = PendingCheckoutContext(
                courseId: courseId,
                courseCode: courseCode,
                title: title
            )
            let result = try await LMSAPI.startCheckout(
                courseId: courseId,
                successUrl: BillingLogic.checkoutSuccessURL(courseId: courseId).absoluteString,
                cancelUrl: BillingLogic.checkoutCancelURL().absoluteString,
                usePaymentsAbstraction: shell.platformFeatures.ffPaymentsEnabled,
                accessToken: token
            )
            guard let url = URL(string: result.checkoutUrl) else {
                errorMessage = L.text("mobile.billing.checkoutError")
                return
            }
            dismiss()
            openURL(url)
        } catch {
            shell.pendingCheckout = nil
            errorMessage = L.text("mobile.billing.checkoutError")
        }
    }
}