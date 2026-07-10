import SwiftUI

/// Course marketplace listing + fee settings (MKT6 / MKT2 parity).
struct CourseMarketplaceSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var listing: CourseCatalogListing?
    @State private var marketplaceListed = false
    @State private var amount = ""
    @State private var currency = "usd"
    @State private var loading = true
    @State private var saving = false
    @State private var errorMessage: String?
    @State private var amountError: String?
    @State private var savedMessage: String?

    private var isDraft: Bool { listing?.publishState == "draft" }

    var body: some View {
        Group {
            if loading && listing == nil {
                ProgressView(L.text("mobile.courseSettings.marketplace.loading"))
                    .padding(24)
            } else if listing == nil {
                if let errorMessage {
                    LMSErrorBanner(message: errorMessage)
                        .padding(16)
                }
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        Text(L.text("mobile.courseSettings.marketplace.title"))
                            .font(.headline)
                        Text(L.text("mobile.courseSettings.marketplace.description"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }
                        if let savedMessage {
                            Text(savedMessage)
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }

                        LMSCard {
                            Toggle(isOn: $marketplaceListed) {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(L.text("mobile.courseSettings.marketplace.listToggle"))
                                        .font(.subheadline.weight(.semibold))
                                    Text(
                                        isDraft
                                            ? L.text("mobile.courseSettings.marketplace.publishFirst")
                                            : L.text("mobile.courseSettings.marketplace.listHelp")
                                    )
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                }
                            }
                            .disabled(saving || isDraft)
                        }

                        LMSCard {
                            VStack(alignment: .leading, spacing: 10) {
                                Text(L.text("mobile.courseSettings.marketplace.fee"))
                                    .font(.subheadline.weight(.semibold))
                                TextField(
                                    L.text("mobile.courseSettings.marketplace.feePlaceholder"),
                                    text: $amount
                                )
                                .keyboardType(.decimalPad)
                                .disabled(saving)
                                if let amountError {
                                    Text(amountError)
                                        .font(.caption)
                                        .foregroundStyle(.red)
                                }
                                Picker(L.text("mobile.courseSettings.marketplace.currency"), selection: $currency) {
                                    ForEach(MarketplaceLogic.currencies, id: \.self) { code in
                                        Text(code.uppercased()).tag(code)
                                    }
                                }
                                .disabled(saving)

                                let previewCents: Int = {
                                    if amount.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                                        return 0
                                    }
                                    return MarketplaceLogic.majorUnitsToPriceCents(amount, currency: currency) ?? listing?.priceCents ?? 0
                                }()
                                Text(MarketplaceLogic.formatPrice(cents: previewCents, currency: currency))
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }

                        Button {
                            Task { await save() }
                        } label: {
                            Text(saving
                                ? L.text("mobile.courseSettings.marketplace.saving")
                                : L.text("mobile.courseSettings.marketplace.save"))
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(LexturesTheme.primary)
                        .disabled(saving)
                    }
                    .padding(16)
                }
            }
        }
        .task { await reload() }
    }

    private func reload() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let data = try await LMSAPI.fetchCourseCatalogListing(courseCode: course.courseCode, accessToken: token)
            listing = data
            marketplaceListed = data.marketplaceListed
            amount = MarketplaceLogic.priceCentsToMajorUnits(data.priceCents, currency: data.priceCurrency.isEmpty ? "usd" : data.priceCurrency)
            currency = data.priceCurrency.isEmpty ? "usd" : data.priceCurrency
            amountError = nil
        } catch {
            errorMessage = L.text("mobile.courseSettings.marketplace.loadError")
            listing = nil
        }
    }

    private func save() async {
        guard let token = session.accessToken, let listing else { return }
        if let validation = MarketplaceLogic.validateAmount(amount, currency: currency) {
            amountError = switch validation {
            case "min": L.text("mobile.courseSettings.marketplace.amountMin")
            case "max": L.text("mobile.courseSettings.marketplace.amountMax")
            default: L.text("mobile.courseSettings.marketplace.amountInvalid")
            }
            return
        }
        amountError = nil
        let nextCents: Int
        if amount.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            nextCents = 0
        } else {
            nextCents = MarketplaceLogic.majorUnitsToPriceCents(amount, currency: currency) ?? listing.priceCents
        }
        saving = true
        savedMessage = nil
        errorMessage = nil
        defer { saving = false }
        do {
            let updated = try await LMSAPI.putCourseCatalogListing(
                courseCode: course.courseCode,
                body: MarketplaceLogic.buildListingPutBody(
                    listing: listing,
                    marketplaceListed: marketplaceListed,
                    priceCents: nextCents,
                    priceCurrency: currency
                ),
                accessToken: token
            )
            self.listing = updated
            marketplaceListed = updated.marketplaceListed
            amount = MarketplaceLogic.priceCentsToMajorUnits(updated.priceCents, currency: updated.priceCurrency.isEmpty ? "usd" : updated.priceCurrency)
            currency = updated.priceCurrency.isEmpty ? "usd" : updated.priceCurrency
            savedMessage = L.text("mobile.courseSettings.marketplace.saved")
        } catch {
            errorMessage = L.text("mobile.courseSettings.marketplace.saveError")
            await reload()
        }
    }
}
