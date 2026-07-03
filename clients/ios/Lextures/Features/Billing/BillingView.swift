import SwiftUI

/// Purchase history, entitlements, and subscription management (M9.2).
struct BillingView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var entitlements: [BillingEntitlement] = []
    @State private var transactions: [BillingTransaction] = []
    @State private var loading = true
    @State private var portalLoading = false
    @State private var errorMessage: String?

    private var activeSubscription: BillingEntitlement? {
        BillingLogic.activeSubscription(entitlements)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                LMSSkeletonList(count: 3)
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }

                        subscriptionSection
                        historySection

                        if let email = shell.profile?.email {
                            Text(L.format("mobile.billing.signedInAs", email))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.billing.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task { await load() }
    }

    @ViewBuilder
    private var subscriptionSection: some View {
        LMSSectionHeader(title: L.text("mobile.billing.subscription"), systemImage: "creditcard")
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                if let activeSubscription {
                    Text(L.format(
                        "mobile.billing.subscriptionActive",
                        BillingLogic.entitlementLabel(activeSubscription.entitlementType)
                    ))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.primary)
                } else {
                    Text(L.text("mobile.billing.noSubscription"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                Button {
                    Task { await openPortal() }
                } label: {
                    Label(
                        portalLoading
                            ? L.text("mobile.billing.openingPortal")
                            : L.text("mobile.billing.manageSubscription"),
                        systemImage: "arrow.up.right.square"
                    )
                    .font(.subheadline.weight(.semibold))
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
                .buttonStyle(.plain)
                .disabled(portalLoading || session.accessToken == nil)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private var historySection: some View {
        LMSSectionHeader(title: L.text("mobile.billing.purchaseHistory"), systemImage: "list.bullet.rectangle")
        if transactions.isEmpty && entitlements.isEmpty {
            LMSEmptyState(
                systemImage: "cart",
                title: L.text("mobile.billing.noPurchasesTitle"),
                message: L.text("mobile.billing.noPurchasesMessage")
            )
        } else if !transactions.isEmpty {
            LMSCard {
                VStack(alignment: .leading, spacing: 12) {
                    ForEach(transactions) { tx in
                        transactionRow(tx)
                        if tx.id != transactions.last?.id { Divider() }
                    }
                }
            }
        } else {
            LMSCard {
                VStack(alignment: .leading, spacing: 12) {
                    ForEach(entitlements) { item in
                        entitlementRow(item)
                        if item.id != entitlements.last?.id { Divider() }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func transactionRow(_ tx: BillingTransaction) -> some View {
        HStack(alignment: .top) {
            VStack(alignment: .leading, spacing: 4) {
                Text(tx.provider.capitalized)
                    .font(.subheadline.weight(.semibold))
                Text(tx.createdAt.prefix(10))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            Spacer()
            VStack(alignment: .trailing, spacing: 4) {
                Text(BillingLogic.formatMoney(cents: tx.amountCents, currency: tx.currency))
                    .font(.subheadline.weight(.semibold))
                Text(tx.status.capitalized)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    @ViewBuilder
    private func entitlementRow(_ item: BillingEntitlement) -> some View {
        HStack(alignment: .top) {
            VStack(alignment: .leading, spacing: 4) {
                Text(BillingLogic.entitlementLabel(item.entitlementType))
                    .font(.subheadline.weight(.semibold))
                Text(item.validFrom.prefix(10))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            Spacer()
            VStack(alignment: .trailing, spacing: 4) {
                Text(BillingLogic.formatMoney(cents: item.amountPaidCents, currency: item.currency))
                    .font(.subheadline.weight(.semibold))
                if let tax = item.taxAmountCents, tax > 0 {
                    Text(L.format(
                        "mobile.billing.taxLine",
                        BillingLogic.formatMoney(cents: tax, currency: item.currency)
                    ))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            async let entitlementsTask = LMSAPI.fetchMyEntitlements(accessToken: token)
            async let transactionsTask = shell.platformFeatures.ffPaymentsEnabled
                ? LMSAPI.fetchMyTransactions(accessToken: token)
                : []
            entitlements = try await entitlementsTask
            transactions = try await transactionsTask
        } catch {
            errorMessage = L.text("mobile.billing.loadError")
        }
    }

    private func openPortal() async {
        guard let token = session.accessToken else { return }
        portalLoading = true
        errorMessage = nil
        defer { portalLoading = false }
        do {
            let urlString = try await LMSAPI.openBillingPortal(
                returnUrl: BillingLogic.billingReturnURL().absoluteString,
                accessToken: token
            )
            guard let url = URL(string: urlString) else { return }
            openURL(url)
        } catch {
            errorMessage = L.text("mobile.billing.portalError")
        }
    }
}

struct BillingRoute: Hashable, Identifiable {
    var id: String { "billing" }
}