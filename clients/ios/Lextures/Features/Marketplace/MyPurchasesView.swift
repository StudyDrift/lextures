import SwiftUI

/// Purchased courses library backed by `/api/v1/me/purchases` (MOB.7 / MKT5).
struct MyPurchasesView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var purchases: [CoursePurchase] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var openError: String?

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                LMSSkeletonList(count: 3)
            } else if !shell.platformFeatures.ffCourseMarketplace {
                LMSEmptyState(
                    systemImage: "storefront",
                    title: L.text("mobile.marketplace.unavailable"),
                    message: L.text("mobile.marketplace.purchases.disabledBody")
                )
            } else if let errorMessage, purchases.isEmpty {
                LMSEmptyState(
                    systemImage: "exclamationmark.triangle",
                    title: L.text("mobile.marketplace.purchases.errorTitle"),
                    message: errorMessage
                )
            } else if purchases.isEmpty {
                LMSEmptyState(
                    systemImage: "cart",
                    title: L.text("mobile.marketplace.purchases.emptyTitle"),
                    message: L.text("mobile.marketplace.purchases.emptyMessage")
                )
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 12) {
                        if let openError {
                            LMSErrorBanner(message: openError)
                        }
                        Text(L.text("mobile.marketplace.purchases.subtitle"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                        ForEach(purchases) { purchase in
                            purchaseRow(purchase)
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.marketplace.purchases.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task { await load() }
    }

    @ViewBuilder
    private func purchaseRow(_ purchase: CoursePurchase) -> some View {
        let freeLabel = L.text("mobile.marketplace.free")
        let priceLabel = MarketplaceLogic.formatPrice(
            cents: purchase.priceCents,
            currency: purchase.currency,
            freeLabel: freeLabel
        )
        let sourceKey = MarketplaceLogic.purchaseSourceLabelKey(purchase.source)
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(purchase.title)
                    .font(.headline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                HStack {
                    Text(priceLabel)
                        .font(.subheadline.weight(.semibold))
                    Text("·")
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text(L.text(sourceKey))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Spacer()
                    Text(MarketplaceLogic.formatAcquiredAt(purchase.acquiredAt))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Button(L.text("mobile.marketplace.goToCourse")) {
                    Task { await open(purchase.courseCode) }
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.primary)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel(purchase.title)
    }

    private func load() async {
        guard let token = session.accessToken else {
            errorMessage = L.text("mobile.marketplace.signInRequired")
            loading = false
            return
        }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            purchases = try await LMSAPI.fetchMyPurchases(accessToken: token)
        } catch {
            errorMessage = L.text("mobile.marketplace.purchases.error")
            purchases = []
        }
    }

    private func open(_ courseCode: String) async {
        guard let token = session.accessToken else { return }
        openError = nil
        do {
            let summary = try await LMSAPI.fetchCourse(courseCode: courseCode, accessToken: token)
            shell.activeCourse = summary
            shell.activeCourseRoot = .profile
            shell.activeCourseSection = .modules
            shell.select(.courses)
        } catch {
            openError = L.text("mobile.marketplace.openCourseError")
        }
    }
}

struct MyPurchasesRoute: Hashable, Identifiable {
    var id: String { "my-purchases" }
}
