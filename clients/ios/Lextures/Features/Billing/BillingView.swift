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
    @State private var restoreLoading = false
    @State private var errorMessage: String?
    @State private var statusMessage: String?
    @State private var showSubscribePaywall = false

    private var activeSubscription: BillingEntitlement? {
        BillingLogic.activeSubscription(entitlements)
    }

    private var historyItems: [BillingHistoryItem] {
        if !transactions.isEmpty {
            return transactions.map { .transaction($0) }
        }
        return entitlements.map { .entitlement($0) }
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                LMSSkeletonList(count: 4)
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 20) {
                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }
                        if let statusMessage {
                            statusBanner(statusMessage)
                        }

                        subscriptionStatusCard
                        manageSection
                        historySection

                        if let email = shell.profile?.email {
                            Text(L.format("mobile.billing.signedInAs", email))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                .frame(maxWidth: .infinity, alignment: .center)
                                .padding(.top, 4)
                        }
                    }
                    .padding(16)
                    .padding(.bottom, 24)
                }
            }
        }
        .navigationTitle(L.text("mobile.billing.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task { await load() }
        .fullScreenCover(isPresented: $showSubscribePaywall) {
            HomeschoolSubscribePaywallView(
                onSubscribed: {
                    showSubscribePaywall = false
                    Task { await load() }
                },
                onDismiss: {
                    showSubscribePaywall = false
                }
            )
            .environment(session)
            .environment(shell)
        }
    }

    // MARK: - Status

    private var subscriptionStatusCard: some View {
        LMSCard(accent: activeSubscription != nil ? LexturesTheme.brandTeal : nil) {
            HStack(alignment: .top, spacing: 14) {
                Image(systemName: activeSubscription != nil ? "checkmark.seal.fill" : "creditcard")
                    .font(.title2.weight(.semibold))
                    .foregroundStyle(
                        activeSubscription != nil
                            ? LexturesTheme.brandTeal
                            : LexturesTheme.textSecondary(for: colorScheme)
                    )
                    .frame(width: 44, height: 44)
                    .background(
                        (activeSubscription != nil ? LexturesTheme.brandTeal : LexturesTheme.textSecondary(for: colorScheme))
                            .opacity(colorScheme == .dark ? 0.18 : 0.12)
                    )
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))

                VStack(alignment: .leading, spacing: 6) {
                    Text(L.text("mobile.billing.subscription"))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .textCase(.uppercase)

                    if let activeSubscription {
                        Text(BillingLogic.entitlementLabel(activeSubscription.entitlementType))
                            .font(.title3.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

                        statusChip(
                            title: L.text("mobile.billing.status.active"),
                            tone: .success
                        )

                        if let until = activeSubscription.validUntil, !until.isEmpty {
                            Text(L.format("mobile.billing.subscriptionActive", String(until.prefix(10))))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    } else {
                        Text(L.text("mobile.billing.noSubscription"))
                            .font(.title3.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

                        Text(L.text("mobile.billing.noSubscriptionHint"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            .fixedSize(horizontal: false, vertical: true)
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }

            if activeSubscription == nil {
                Button {
                    showSubscribePaywall = true
                } label: {
                    Text(L.text("mobile.billing.subscribe"))
                        .font(.subheadline.weight(.semibold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                        .foregroundStyle(.white)
                        .background(LexturesTheme.primary)
                        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                }
                .buttonStyle(.plain)
                .padding(.top, 4)
                .accessibilityHint(L.text("mobile.billing.subscribeHint"))
            }
        }
    }

    // MARK: - Manage

    private var manageSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            LMSSectionHeader(title: L.text("mobile.billing.manage"), systemImage: "gearshape")

            LMSCard {
                actionRow(
                    systemImage: "apple.logo",
                    title: L.text("mobile.billing.manageSubscription"),
                    subtitle: L.text("mobile.billing.manageSubscriptionHint"),
                    trailingSystemImage: "arrow.up.right"
                ) {
                    openURL(BillingLogic.appStoreSubscriptionsURL())
                }

                Divider().opacity(0.5)

                actionRow(
                    systemImage: "arrow.clockwise",
                    title: restoreLoading
                        ? L.text("mobile.billing.iap.restoring")
                        : L.text("mobile.billing.iap.restore"),
                    subtitle: L.text("mobile.billing.restoreHint"),
                    trailingSystemImage: restoreLoading ? nil : "chevron.right",
                    disabled: restoreLoading || session.accessToken == nil
                ) {
                    Task { await restorePurchases() }
                }

                Divider().opacity(0.5)

                actionRow(
                    systemImage: "globe",
                    title: portalLoading
                        ? L.text("mobile.billing.openingPortal")
                        : L.text("mobile.billing.manageWebSubscription"),
                    subtitle: L.text("mobile.billing.manageWebSubscriptionHint"),
                    trailingSystemImage: portalLoading ? nil : "arrow.up.right",
                    disabled: portalLoading || session.accessToken == nil
                ) {
                    Task { await openPortal() }
                }
            }
        }
    }

    // MARK: - History

    private var historySection: some View {
        VStack(alignment: .leading, spacing: 10) {
            LMSSectionHeader(title: L.text("mobile.billing.purchaseHistory"), systemImage: "list.bullet.rectangle")

            if historyItems.isEmpty {
                LMSEmptyState(
                    systemImage: "cart",
                    title: L.text("mobile.billing.noPurchasesTitle"),
                    message: L.text("mobile.billing.noPurchasesMessage")
                )
            } else {
                LMSCard {
                    ForEach(Array(historyItems.enumerated()), id: \.element.id) { index, item in
                        switch item {
                        case .transaction(let tx):
                            transactionRow(tx)
                        case .entitlement(let ent):
                            entitlementRow(ent)
                        }
                        if index < historyItems.count - 1 {
                            Divider().opacity(0.5)
                        }
                    }
                }
            }
        }
    }

    // MARK: - Rows

    private func actionRow(
        systemImage: String,
        title: String,
        subtitle: String,
        trailingSystemImage: String?,
        disabled: Bool = false,
        action: @escaping () -> Void
    ) -> some View {
        Button(action: action) {
            HStack(spacing: 12) {
                Image(systemName: systemImage)
                    .font(.body.weight(.semibold))
                    .foregroundStyle(LexturesTheme.primary)
                    .frame(width: 36, height: 36)
                    .background(LexturesTheme.primary.opacity(colorScheme == .dark ? 0.18 : 0.12))
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))

                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(subtitle)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .fixedSize(horizontal: false, vertical: true)
                }
                .frame(maxWidth: .infinity, alignment: .leading)

                if let trailingSystemImage {
                    Image(systemName: trailingSystemImage)
                        .font(.footnote.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else if disabled {
                    ProgressView()
                        .controlSize(.small)
                }
            }
            .padding(.vertical, 4)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .disabled(disabled)
        .opacity(disabled && trailingSystemImage != nil ? 0.55 : 1)
    }

    private func transactionRow(_ tx: BillingTransaction) -> some View {
        HStack(alignment: .center, spacing: 12) {
            VStack(alignment: .leading, spacing: 4) {
                Text(providerLabel(tx.provider))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(String(tx.createdAt.prefix(10)))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            Spacer(minLength: 8)
            VStack(alignment: .trailing, spacing: 6) {
                Text(BillingLogic.formatMoney(cents: tx.amountCents, currency: tx.currency))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                statusChip(title: statusLabel(tx.status), tone: statusTone(tx.status))
            }
        }
        .padding(.vertical, 4)
        .accessibilityElement(children: .combine)
    }

    private func entitlementRow(_ item: BillingEntitlement) -> some View {
        HStack(alignment: .center, spacing: 12) {
            VStack(alignment: .leading, spacing: 4) {
                Text(BillingLogic.entitlementLabel(item.entitlementType))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(String(item.validFrom.prefix(10)))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            Spacer(minLength: 8)
            VStack(alignment: .trailing, spacing: 6) {
                Text(BillingLogic.formatMoney(cents: item.amountPaidCents, currency: item.currency))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                statusChip(title: statusLabel(item.status), tone: statusTone(item.status))
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
        .padding(.vertical, 4)
        .accessibilityElement(children: .combine)
    }

    private func statusBanner(_ message: String) -> some View {
        Label(message, systemImage: "checkmark.circle.fill")
            .font(.subheadline)
            .foregroundStyle(LexturesTheme.brandTeal)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(14)
            .background(LexturesTheme.brandTeal.opacity(0.1))
            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
    }

    private func statusChip(title: String, tone: StatusTone) -> some View {
        Text(title)
            .font(.caption2.weight(.semibold))
            .foregroundStyle(tone.foreground)
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(tone.foreground.opacity(colorScheme == .dark ? 0.2 : 0.12))
            .clipShape(Capsule())
    }

    // MARK: - Formatting

    private enum StatusTone {
        case success, warning, neutral, danger

        var foreground: Color {
            switch self {
            case .success: return LexturesTheme.brandTeal
            case .warning: return LexturesTheme.amber
            case .danger: return LexturesTheme.error
            case .neutral: return .secondary
            }
        }
    }

    private enum BillingHistoryItem: Identifiable {
        case transaction(BillingTransaction)
        case entitlement(BillingEntitlement)

        var id: String {
            switch self {
            case .transaction(let tx): return "tx-\(tx.id)"
            case .entitlement(let ent): return "ent-\(ent.id)"
            }
        }
    }

    private func providerLabel(_ provider: String) -> String {
        let trimmed = provider.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return L.text("mobile.billing.provider.unknown") }
        switch trimmed.lowercased() {
        case "stripe": return L.text("mobile.billing.provider.stripe")
        case "apple", "app_store", "appstore": return L.text("mobile.billing.provider.apple")
        default: return trimmed.capitalized
        }
    }

    private func statusLabel(_ status: String) -> String {
        switch status.lowercased() {
        case "active", "succeeded", "paid", "complete", "completed":
            return L.text("mobile.billing.status.paid")
        case "pending", "processing", "open":
            return L.text("mobile.billing.status.pending")
        case "failed", "canceled", "cancelled":
            return L.text("mobile.billing.status.failed")
        case "refunded":
            return L.text("mobile.billing.status.refunded")
        default:
            return status.capitalized
        }
    }

    private func statusTone(_ status: String) -> StatusTone {
        switch status.lowercased() {
        case "active", "succeeded", "paid", "complete", "completed":
            return .success
        case "pending", "processing", "open":
            return .warning
        case "failed", "canceled", "cancelled":
            return .danger
        default:
            return .neutral
        }
    }

    // MARK: - Networking

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

    private func restorePurchases() async {
        guard let token = session.accessToken else { return }
        restoreLoading = true
        errorMessage = nil
        statusMessage = nil
        defer { restoreLoading = false }
        do {
            let count = try await StoreKitPurchaseService.restorePurchases(accessToken: token)
            statusMessage = L.format("mobile.billing.iap.restoreDone", count)
            await load()
        } catch {
            errorMessage = L.text("mobile.billing.iap.restoreFailed")
        }
    }
}

struct BillingRoute: Hashable, Identifiable {
    var id: String { "billing" }
}
