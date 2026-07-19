import SwiftUI

/// Marketplace course detail with free claim and paid Stripe checkout handoff (MKT6 / MOB.7).
struct MarketplaceDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let slug: String

    @State private var detail: MarketplaceCourseDetail?
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var claiming = false
    @State private var claimError: String?
    @State private var showPurchase = false

    private var purchaseEnabled: Bool {
        MarketplaceLogic.purchaseEnabled(shell.platformFeatures)
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
                    }
                )
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
        let priceLabel = MarketplaceLogic.formatPrice(
            cents: detail.priceCents,
            currency: detail.priceCurrency,
            freeLabel: freeLabel
        )

        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                HStack {
                    Text(priceLabel)
                        .font(.title3.weight(.bold))
                        .accessibilityLabel(L.text("mobile.marketplace.priceLabel"))
                        .accessibilityValue(priceLabel)
                    Spacer()
                    actionButton(detail)
                }
                if MarketplaceLogic.isPaid(priceCents: detail.priceCents) && !detail.course.owned {
                    Text(
                        purchaseEnabled
                            ? L.text("mobile.marketplace.paidCheckoutHint")
                            : L.text("mobile.marketplace.paidWebHint")
                    )
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    @ViewBuilder
    private func actionButton(_ detail: MarketplaceCourseDetail) -> some View {
        if detail.course.owned {
            Button(L.text("mobile.marketplace.goToCourse")) {
                Task { await openOwnedCourse(detail.course.courseCode) }
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.primary)
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
                    openURL(AppConfiguration.webURL(path: MarketplaceLogic.marketplaceWebPath(slug: slug)))
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
            } else {
                errorMessage = L.text("mobile.marketplace.landingNotFound")
            }
        } catch {
            errorMessage = L.text("mobile.marketplace.landingError")
        }
    }

    private func claim(_ detail: MarketplaceCourseDetail) async {
        guard let token = session.accessToken else { return }
        claiming = true
        claimError = nil
        defer { claiming = false }
        do {
            let result = try await LMSAPI.claimMarketplaceCourse(slug: slug, accessToken: token)
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

    private func reloadOwnedAndOpen(_ courseCode: String) async {
        claimError = nil
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
}
