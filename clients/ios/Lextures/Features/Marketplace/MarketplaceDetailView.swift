import SwiftUI
import UIKit

/// Marketplace course detail with free claim, paid IAP, and coupon redemption (MKT6 / MOB.7 / MKTC.6).
struct MarketplaceDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    let slug: String

    @State private var detail: MarketplaceCourseDetail?
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var claiming = false
    @State private var claimError: String?
    @State private var showPurchase = false

    // Coupon state (session memory only — FR-5).
    @State private var couponExpanded = false
    @State private var couponInput = ""
    @State private var couponPreview: CouponPreview?
    @State private var couponChecking = false
    @State private var couponError: String?
    @State private var couponAnnouncement = ""

    private var purchaseEnabled: Bool {
        MarketplaceLogic.purchaseEnabled(shell.platformFeatures)
    }

    private var couponsEnabled: Bool {
        MarketplaceLogic.couponsEnabled(shell.platformFeatures)
    }

    private var webRedirectLinkEnabled: Bool {
        shell.platformFeatures.iosCouponWebRedirectEnabled && AppConfiguration.iosCouponWebRedirectEnabled
    }

    private var purchaseRoute: MarketplaceIOSPurchaseRoute {
        MarketplaceLogic.purchaseRoute(preview: couponPreview, features: shell.platformFeatures)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                LMSSkeletonList(count: 3)
            } else if let errorMessage, detail == nil {
                LMSEmptyState(
                    systemImage: "exclamationmark.triangle",
                    title: L.text("mobile.marketplace.landingErrorTitle"),
                    message: errorMessage
                )
            } else if let detail {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        header(detail)
                        aboutSection(detail.course)
                        whatsIncludedSection(detail.whatsIncluded)
                        actionSection(detail)
                        if couponsEnabled,
                           MarketplaceLogic.isPaid(priceCents: detail.priceCents),
                           !detail.course.owned {
                            couponSection(detail)
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(detail?.course.title ?? L.text("mobile.marketplace.landingTitle"))
        .navigationBarTitleDisplayMode(.inline)
        .task {
            MarketplaceObservability.record("marketplace_viewed")
            await load()
            await autoApplyPendingCouponIfNeeded()
        }
        .onChange(of: detail?.course.owned) { _, owned in
            if owned == true {
                clearCouponState()
            }
        }
        .onChange(of: shell.platformFeatures.ffCourseCoupons) { _, enabled in
            if enabled {
                Task { await autoApplyPendingCouponIfNeeded() }
            }
        }
        .sheet(isPresented: $showPurchase) {
            if let detail {
                PurchaseFlowSheet(
                    courseId: detail.course.id,
                    courseCode: detail.course.courseCode,
                    title: detail.course.title,
                    priceCents: detail.priceCents,
                    currency: detail.priceCurrency,
                    marketplaceSlug: slug,
                    onAlreadyOwned: {
                        Task {
                            await reloadOwnedAndOpen(detail.course.courseCode)
                        }
                    },
                    onPurchased: {
                        clearCouponState()
                    }
                )
            }
        }
        .accessibilityElement(children: .contain)
        .overlay(alignment: .bottom) {
            if !couponAnnouncement.isEmpty {
                Text(couponAnnouncement)
                    .font(.caption)
                    .foregroundStyle(.clear)
                    .accessibilityLabel(couponAnnouncement)
                    .accessibilityAddTraits(.updatesFrequently)
                    .accessibilityHidden(false)
                    .frame(width: 1, height: 1)
                    .allowsHitTesting(false)
            }
        }
    }

    @ViewBuilder
    private func header(_ detail: MarketplaceCourseDetail) -> some View {
        let course = detail.course
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                CourseHeroImage(urlString: course.heroImageUrl, fallbackKey: course.courseCode, height: 160)
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))

                HStack(spacing: 8) {
                    if let category = course.category, !category.isEmpty {
                        Text(category)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    if let level = course.level, !level.isEmpty {
                        Text(level.capitalized)
                            .font(.caption.weight(.semibold))
                            .padding(.horizontal, 8)
                            .padding(.vertical, 3)
                            .background(Capsule().fill(LexturesTheme.cardBackground(for: colorScheme)))
                    }
                    if course.owned {
                        Text(L.text("mobile.marketplace.owned"))
                            .font(.caption.weight(.semibold))
                            .padding(.horizontal, 8)
                            .padding(.vertical, 3)
                            .background(Capsule().fill(LexturesTheme.accent(for: colorScheme).opacity(0.15)))
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    }
                }

                if let instructor = course.instructorName, !instructor.isEmpty {
                    Text(L.format("mobile.marketplace.instructor", instructor))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                Text(L.format("mobile.marketplace.enrolledCount", course.enrollmentCount))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private func aboutSection(_ course: MarketplaceCourse) -> some View {
        LMSSectionHeader(title: L.text("mobile.marketplace.aboutTitle"), systemImage: "text.alignleft")
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                let paragraphs = MarketplaceLogic.previewParagraphs(from: course.description)
                if paragraphs.isEmpty {
                    Text(L.text("mobile.marketplace.noDescription"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(Array(paragraphs.enumerated()), id: \.offset) { _, paragraph in
                        Text(paragraph)
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private func whatsIncludedSection(_ included: MarketplaceWhatsIncluded) -> some View {
        LMSSectionHeader(title: L.text("mobile.marketplace.whatsIncluded"), systemImage: "list.bullet")
        LMSCard {
            VStack(alignment: .leading, spacing: 6) {
                Text(L.format("mobile.marketplace.modulesCount", included.moduleCount))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(L.format("mobile.marketplace.itemsCount", included.itemCount))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private func actionSection(_ detail: MarketplaceCourseDetail) -> some View {
        if let claimError {
            LMSErrorBanner(message: claimError)
        }

        let freeLabel = L.text("mobile.marketplace.free")
        let listCents = detail.priceCents
        let currency = detail.priceCurrency
        let applied = couponPreview?.applied == true
        let chargedCents = applied ? (couponPreview?.chargedCents ?? listCents) : listCents
        let chargedLabel = MarketplaceLogic.formatPrice(
            cents: chargedCents,
            currency: currency,
            freeLabel: freeLabel
        )
        let listLabel = MarketplaceLogic.formatPrice(
            cents: listCents,
            currency: currency,
            freeLabel: freeLabel
        )
        let discountCents = applied ? (couponPreview?.discountCents ?? 0) : 0
        let savingsLabel = discountCents > 0
            ? MarketplaceLogic.formatPrice(cents: discountCents, currency: currency, freeLabel: freeLabel)
            : nil

        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                priceRow(
                    chargedLabel: chargedLabel,
                    listLabel: listLabel,
                    showStrike: applied && discountCents > 0 && chargedCents < listCents,
                    savingsLabel: savingsLabel
                )
                HStack {
                    Spacer()
                    actionButton(detail, chargedCents: chargedCents, chargedLabel: chargedLabel, listLabel: listLabel)
                }
                if let routeHint = routeHint(for: detail) {
                    Text(routeHint)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                if purchaseRoute == .webRedirect, let preview = couponPreview, preview.applied {
                    webRedirectNotice(preview: preview, listLabel: listLabel)
                }
            }
        }
    }

    @ViewBuilder
    private func priceRow(
        chargedLabel: String,
        listLabel: String,
        showStrike: Bool,
        savingsLabel: String?
    ) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(chargedLabel)
                    .font(.title3.weight(.bold))
                if showStrike {
                    Text(listLabel)
                        .font(.subheadline)
                        .strikethrough()
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .accessibilityLabel(L.format("mobile.marketplace.coupon.wasPrice", listLabel))
                }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel(
                showStrike
                    ? L.format("mobile.marketplace.coupon.priceNowWas", chargedLabel, listLabel)
                    : L.text("mobile.marketplace.priceLabel")
            )
            .accessibilityValue(chargedLabel)

            if let savingsLabel, showStrike {
                Text(L.format("mobile.marketplace.coupon.youSave", savingsLabel))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            }
        }
    }

    @ViewBuilder
    private func webRedirectNotice(preview: CouponPreview, listLabel: String) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Label {
                Text(L.text("mobile.marketplace.coupon.webOnly"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } icon: {
                Image(systemName: "safari")
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            }
            .accessibilityLabel(L.text("mobile.marketplace.coupon.webOnly"))

            HStack(spacing: 8) {
                Button {
                    UIPasteboard.general.string = preview.code
                    announce(L.text("mobile.marketplace.coupon.codeCopied"))
                } label: {
                    Text(L.text("mobile.marketplace.coupon.copyCode"))
                        .font(.subheadline.weight(.semibold))
                }
                .buttonStyle(.bordered)
                .accessibilityLabel(L.text("mobile.marketplace.coupon.copyCode"))

                if webRedirectLinkEnabled {
                    Button {
                        openCouponInBrowser(code: preview.code)
                    } label: {
                        Text(L.text("mobile.marketplace.coupon.continueInBrowser"))
                            .font(.subheadline.weight(.semibold))
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(LexturesTheme.primary)
                    .accessibilityLabel(L.text("mobile.marketplace.coupon.continueInBrowser"))
                }
            }

            if purchaseEnabled {
                Button {
                    showPurchase = true
                } label: {
                    Text(L.format("mobile.marketplace.coupon.buyFullPrice", listLabel))
                        .font(.subheadline.weight(.semibold))
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered)
                .disabled(session.accessToken == nil)
                .accessibilityLabel(L.format("mobile.marketplace.coupon.buyFullPrice", listLabel))
                .accessibilityHint(L.text("mobile.marketplace.coupon.buyFullPriceHint"))
            }
        }
        .padding(.top, 4)
    }

    private func routeHint(for detail: MarketplaceCourseDetail) -> String? {
        if detail.course.owned { return nil }
        if purchaseRoute == .webRedirect {
            return nil
        }
        if MarketplaceLogic.isPaid(priceCents: detail.priceCents) {
            return purchaseEnabled
                ? L.text("mobile.marketplace.paidCheckoutHint")
                : L.text("mobile.marketplace.paidWebHint")
        }
        return nil
    }

    @ViewBuilder
    private func actionButton(
        _ detail: MarketplaceCourseDetail,
        chargedCents: Int,
        chargedLabel: String,
        listLabel: String
    ) -> some View {
        if detail.course.owned {
            Button(L.text("mobile.marketplace.goToCourse")) {
                Task { await openOwnedCourse(detail.course.courseCode) }
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.primary)
        } else if purchaseRoute == .freeGrant, let preview = couponPreview, preview.applied {
            Button {
                Task { await claimWithCoupon(detail, code: preview.code) }
            } label: {
                Text(claiming
                    ? L.text("mobile.marketplace.claiming")
                    : L.text("mobile.marketplace.coupon.enrollFree"))
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.primary)
            .disabled(claiming || session.accessToken == nil)
        } else if purchaseRoute == .webRedirect {
            // Discounted non-zero: primary StoreKit amount button disabled (FR-13/FR-14).
            Button(L.format("mobile.marketplace.coupon.buyDiscountedDisabled", chargedLabel)) {}
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.primary)
                .disabled(true)
                .accessibilityLabel(L.format("mobile.marketplace.coupon.buyDiscountedDisabled", chargedLabel))
                .accessibilityHint(L.text("mobile.marketplace.coupon.disabledIAPHint"))
        } else if MarketplaceLogic.isPaid(priceCents: detail.priceCents) {
            if purchaseEnabled {
                Button(L.text("mobile.marketplace.buy")) {
                    showPurchase = true
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.primary)
                .disabled(session.accessToken == nil)
            } else {
                Button(L.text("mobile.marketplace.buyOnWeb")) {
                    LinkOpener.open(
                        AppConfiguration.webURL(
                            path: MarketplaceLogic.marketplaceWebPath(slug: slug)
                        ),
                        shell: shell,
                        source: "marketplace"
                    )
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.primary)
            }
        } else {
            Button {
                Task { await claim(detail) }
            } label: {
                Text(claiming ? L.text("mobile.marketplace.claiming") : L.text("mobile.marketplace.enrollFree"))
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.primary)
            .disabled(claiming || session.accessToken == nil)
        }
    }

    @ViewBuilder
    private func couponSection(_ detail: MarketplaceCourseDetail) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                DisclosureGroup(isExpanded: $couponExpanded) {
                    couponContent(detail)
                } label: {
                    Text(L.text("mobile.marketplace.coupon.haveCode"))
                        .font(.subheadline.weight(.semibold))
                }

                if let preview = couponPreview, preview.applied {
                    appliedCouponRow(preview)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private func couponContent(_ detail: MarketplaceCourseDetail) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            if let couponError {
                Text(couponError)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.error)
                    .accessibilityLabel(couponError)
            }

            HStack(spacing: 8) {
                TextField(
                    L.text("mobile.marketplace.coupon.placeholder"),
                    text: $couponInput
                )
                .textInputAutocapitalization(.characters)
                .autocorrectionDisabled()
                .textFieldStyle(.roundedBorder)
                .disabled(couponChecking)
                .accessibilityLabel(L.text("mobile.marketplace.coupon.label"))

                Button {
                    Task { await applyCoupon(detail) }
                } label: {
                    if couponChecking {
                        ProgressView()
                    } else {
                        Text(L.text("mobile.marketplace.coupon.apply"))
                    }
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.primary)
                .disabled(
                    couponChecking
                        || couponInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                        || !NetworkMonitor.shared.isOnline
                        || session.accessToken == nil
                )
            }

            if !NetworkMonitor.shared.isOnline {
                Text(L.text("mobile.marketplace.coupon.offline"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
        .padding(.top, 4)
    }

    @ViewBuilder
    private func appliedCouponRow(_ preview: CouponPreview) -> some View {
        HStack(spacing: 8) {
            Image(systemName: "checkmark.seal.fill")
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            VStack(alignment: .leading, spacing: 2) {
                Text(preview.code)
                    .font(.subheadline.weight(.semibold))
                if preview.discountCents > 0 {
                    let saved = MarketplaceLogic.formatPrice(
                        cents: preview.discountCents,
                        currency: preview.currency,
                        freeLabel: L.text("mobile.marketplace.free")
                    )
                    Text(L.format("mobile.marketplace.coupon.youSave", saved))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            Spacer()
            Button(L.text("mobile.marketplace.coupon.remove")) {
                removeCoupon()
            }
            .font(.caption.weight(.semibold))
            .accessibilityLabel(L.text("mobile.marketplace.coupon.remove"))
        }
        .padding(10)
        .background(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .fill(LexturesTheme.accent(for: colorScheme).opacity(0.12))
        )
        .accessibilityElement(children: .combine)
    }

    // MARK: - Data

    private func load() async {
        loading = true
        errorMessage = nil
        defer { loading = false }
        guard let token = session.accessToken else {
            errorMessage = L.text("mobile.marketplace.signInRequired")
            return
        }
        do {
            let loaded = try await LMSAPI.fetchMarketplaceCourseDetail(slug: slug, accessToken: token)
            if let loaded {
                detail = loaded
                if loaded.course.owned {
                    clearCouponState()
                }
            } else {
                errorMessage = L.text("mobile.marketplace.landingNotFound")
            }
        } catch {
            errorMessage = L.text("mobile.marketplace.landingError")
        }
    }

    private func autoApplyPendingCouponIfNeeded() async {
        guard let code = shell.peekPendingCoupon(for: slug), !code.isEmpty else { return }
        // Wait until coupons are enabled (platform features may still be loading).
        // When the flag is definitively off after marketplace is on, drop without preview (AC-11).
        if !couponsEnabled {
            if shell.platformFeatures.ffCourseMarketplace && !shell.platformFeatures.ffCourseCoupons {
                _ = shell.consumePendingCoupon(for: slug)
            }
            return
        }
        _ = shell.consumePendingCoupon(for: slug)
        couponExpanded = true
        couponInput = code
        guard let detail, !detail.course.owned else { return }
        await applyCoupon(detail, codeOverride: code, fromDeepLink: true)
    }

    private func applyCoupon(
        _ detail: MarketplaceCourseDetail,
        codeOverride: String? = nil,
        fromDeepLink: Bool = false
    ) async {
        guard couponsEnabled else { return }
        guard let token = session.accessToken else { return }
        guard NetworkMonitor.shared.isOnline else {
            couponError = L.text("mobile.marketplace.coupon.offline")
            return
        }
        let raw = codeOverride ?? couponInput
        let code = MarketplaceLogic.normalizeCouponCode(raw)
        guard !code.isEmpty else {
            couponError = L.dynamicText(MarketplaceLogic.couponReasonKey("not_found"))
            return
        }

        couponChecking = true
        couponError = nil
        defer { couponChecking = false }

        do {
            let preview = try await LMSAPI.previewMarketplaceCoupon(
                slug: slug,
                code: code,
                accessToken: token
            )
            couponInput = preview.code.isEmpty ? code : preview.code
            if preview.applied {
                couponPreview = preview
                couponError = nil
                announceAppliedPrice(preview: preview, listCents: detail.priceCents)
            } else {
                couponPreview = nil
                couponError = L.dynamicText(MarketplaceLogic.couponReasonKey(preview.reason))
                announce(couponError ?? "")
            }
            if fromDeepLink {
                MarketplaceObservability.record(
                    "coupon_from_deeplink",
                    attributes: ["result": preview.applied ? "ok" : preview.reason]
                )
            }
        } catch let error as APIError {
            couponPreview = nil
            if case let .httpStatus(status, message: _) = error {
                if status == 429 {
                    couponError = L.text("mobile.marketplace.coupon.cooldown")
                } else {
                    couponError = L.text("mobile.marketplace.coupon.applyError")
                }
            } else {
                couponError = L.text("mobile.marketplace.coupon.applyError")
            }
            announce(couponError ?? "")
        } catch {
            couponPreview = nil
            couponError = L.text("mobile.marketplace.coupon.applyError")
            announce(couponError ?? "")
        }
    }

    private func removeCoupon() {
        clearCouponState(keepExpanded: true)
        announce(L.text("mobile.marketplace.coupon.removed"))
    }

    private func clearCouponState(keepExpanded: Bool = false) {
        couponPreview = nil
        couponError = nil
        couponInput = ""
        if !keepExpanded { couponExpanded = false }
        shell.clearPendingCoupon(for: slug)
    }

    private func claim(_ detail: MarketplaceCourseDetail) async {
        guard let token = session.accessToken else { return }
        claiming = true
        claimError = nil
        defer { claiming = false }
        do {
            let result = try await LMSAPI.claimMarketplaceCourse(slug: slug, accessToken: token)
            clearCouponState()
            let code = result.courseCode.isEmpty ? detail.course.courseCode : result.courseCode
            await openOwnedCourse(code)
        } catch let error as APIError {
            if case let .httpStatus(status, message: _) = error, status == 402 {
                claimError = L.text("mobile.marketplace.claimPaidError")
            } else {
                claimError = L.text("mobile.marketplace.claimError")
            }
        } catch {
            claimError = L.text("mobile.marketplace.claimError")
        }
    }

    private func claimWithCoupon(_ detail: MarketplaceCourseDetail, code: String) async {
        guard let token = session.accessToken else { return }
        claiming = true
        claimError = nil
        defer { claiming = false }
        do {
            let result = try await LMSAPI.claimMarketplaceCourse(
                slug: slug,
                couponCode: code,
                accessToken: token
            )
            clearCouponState()
            let courseCode = result.courseCode.isEmpty ? detail.course.courseCode : result.courseCode
            await openOwnedCourse(courseCode)
        } catch let error as APIError {
            if case let .httpStatus(status, message: _) = error {
                switch status {
                case 422:
                    // Code lapsed between preview and buy (FR-6).
                    clearCouponState(keepExpanded: true)
                    claimError = L.text("mobile.marketplace.coupon.lapsed")
                    couponError = L.text("mobile.marketplace.coupon.lapsed")
                    announce(claimError ?? "")
                case 429:
                    claimError = L.text("mobile.marketplace.coupon.cooldown")
                    announce(claimError ?? "")
                case 402:
                    claimError = L.text("mobile.marketplace.claimPaidError")
                default:
                    claimError = L.text("mobile.marketplace.claimError")
                }
            } else {
                claimError = L.text("mobile.marketplace.claimError")
            }
        } catch {
            claimError = L.text("mobile.marketplace.claimError")
        }
    }

    private func openCouponInBrowser(code: String) {
        let path = MarketplaceLogic.marketplaceWebPath(slug: slug, couponCode: code)
        MarketplaceObservability.record(
            "coupon_web_redirect",
            attributes: ["slug": slug]
        )
        LinkOpener.open(
            LinkOpener.Request(
                urlString: path,
                source: "marketplace_coupon",
                forceSystemBrowser: true
            ),
            shell: shell
        )
    }

    private func reloadOwnedAndOpen(_ courseCode: String) async {
        claimError = nil
        clearCouponState()
        if var current = detail {
            current.course.owned = true
            detail = current
        }
        await openOwnedCourse(courseCode)
    }

    private func openOwnedCourse(_ courseCode: String) async {
        guard let token = session.accessToken else { return }
        do {
            let summary = try await LMSAPI.fetchCourse(courseCode: courseCode, accessToken: token)
            shell.activeCourse = summary
            shell.activeCourseRoot = .profile
            shell.activeCourseSection = .modules
            shell.select(.courses)
        } catch {
            claimError = L.text("mobile.marketplace.openCourseError")
        }
    }

    private func announceAppliedPrice(preview: CouponPreview, listCents: Int) {
        let freeLabel = L.text("mobile.marketplace.free")
        let charged = MarketplaceLogic.formatPrice(
            cents: preview.chargedCents,
            currency: preview.currency,
            freeLabel: freeLabel
        )
        let list = MarketplaceLogic.formatPrice(
            cents: listCents,
            currency: preview.currency,
            freeLabel: freeLabel
        )
        if preview.discountCents > 0 {
            announce(L.format("mobile.marketplace.coupon.priceNowWas", charged, list))
        } else {
            announce(L.format("mobile.marketplace.coupon.applied", preview.code, charged))
        }
    }

    private func announce(_ message: String) {
        guard !message.isEmpty else { return }
        couponAnnouncement = message
        // VoiceOver / accessibility announcement.
        UIAccessibility.post(notification: .announcement, argument: message)
    }
}
