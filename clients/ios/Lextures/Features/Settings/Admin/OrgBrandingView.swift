import PhotosUI
import SwiftUI
import UIKit

/// Branding card: logo, colors, email sender, custom domain read-only (M14.5).
struct OrgBrandingView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let orgId: String

    @State private var branding: OrgBrandingResponse?
    @State private var primaryColor = OrgBrandingAdminLogic.defaultPrimaryColor
    @State private var secondaryColor = OrgBrandingAdminLogic.defaultSecondaryColor
    @State private var emailDisplayName = ""
    @State private var previewLogoUrl: URL?
    @State private var loading = true
    @State private var saveStatus: OrgBrandingAdminLogic.SaveStatus = .idle
    @State private var uploadingLogo = false
    @State private var photoItem: PhotosPickerItem?

    var body: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 16) {
                Text(L.text("mobile.admin.orgBranding.branding.title"))
                    .font(.headline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                } else {
                    logoSection
                    colorSection
                    contrastWarning
                    emailSenderSection
                    customDomainSection
                    previewSection
                    saveSection
                }
            }
        }
        .task(id: orgId) { await load() }
        .onChange(of: photoItem) { _, item in
            Task { await uploadLogo(item) }
        }
    }

    private var logoSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.admin.orgBranding.branding.logo"))
                .font(.subheadline.weight(.semibold))

            HStack(spacing: 12) {
                if let previewLogoUrl {
                    AsyncImage(url: previewLogoUrl) { phase in
                        switch phase {
                        case .success(let image):
                            image.resizable().scaledToFit()
                        default:
                            ProgressView()
                        }
                    }
                    .frame(width: 72, height: 72)
                    .accessibilityHidden(true)
                }

                PhotosPicker(selection: $photoItem, matching: .images) {
                    Label(
                        uploadingLogo
                            ? L.text("mobile.admin.orgBranding.branding.uploading")
                            : L.text("mobile.admin.orgBranding.branding.uploadLogo"),
                        systemImage: "photo"
                    )
                }
                .disabled(uploadingLogo || saveStatus == .saving)
            }

            Text(L.text("mobile.admin.orgBranding.branding.logoHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private var colorSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            colorField(
                title: L.text("mobile.admin.orgBranding.branding.primaryColor"),
                hex: $primaryColor
            )
            colorField(
                title: L.text("mobile.admin.orgBranding.branding.secondaryColor"),
                hex: $secondaryColor
            )
        }
    }

    private func colorField(title: String, hex: Binding<String>) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title)
                .font(.subheadline.weight(.semibold))
            HStack(spacing: 10) {
                ColorPicker("", selection: Binding(
                    get: { OrgBrandingAdminLogic.color(from: hex.wrappedValue) },
                    set: { newColor in
                        if let converted = newColor.toHex() {
                            hex.wrappedValue = converted
                        }
                    }
                ))
                .labelsHidden()
                .frame(width: 44, height: 44)
                .accessibilityLabel(title)

                TextField("#RRGGBB", text: hex)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .font(.system(.body, design: .monospaced))
                    .textFieldStyle(.roundedBorder)
            }
        }
    }

    @ViewBuilder
    private var contrastWarning: some View {
        let warning = OrgBrandingAdminLogic.showsContrastWarning(
            primaryColor: primaryColor,
            serverWarning: branding?.contrastWarningPrimary == true,
            serverRatio: branding?.contrastRatioPrimary
        )
        if warning {
            Text(L.text("mobile.admin.orgBranding.branding.contrastWarning"))
                .font(.caption)
                .foregroundStyle(.orange)
                .accessibilityLabel(L.text("mobile.admin.orgBranding.branding.contrastWarning"))
        }
    }

    private var emailSenderSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(L.text("mobile.admin.orgBranding.branding.emailSender"))
                .font(.subheadline.weight(.semibold))
            TextField(
                L.text("mobile.admin.orgBranding.branding.emailSenderPlaceholder"),
                text: $emailDisplayName
            )
            .textFieldStyle(.roundedBorder)
            Text(L.text("mobile.admin.orgBranding.branding.emailSenderHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private var customDomainSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.admin.orgBranding.branding.customDomain"))
                .font(.subheadline.weight(.semibold))

            if let domain = branding?.customDomain, !domain.isEmpty {
                Text(domain)
                    .font(.body.monospaced())
            } else {
                Text(L.text("mobile.admin.orgBranding.branding.customDomainNone"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }

            Button {
                openURL(AppConfiguration.webURL(path: OrgBrandingAdminLogic.webOrgBrandingPath()))
            } label: {
                Label(L.text("mobile.admin.orgBranding.branding.configureOnWeb"), systemImage: "safari")
                    .font(.subheadline.weight(.medium))
            }
            .buttonStyle(.bordered)
        }
    }

    private var previewSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.admin.orgBranding.branding.preview"))
                .font(.subheadline.weight(.semibold))

            VStack(spacing: 12) {
                if let previewLogoUrl {
                    AsyncImage(url: previewLogoUrl) { phase in
                        if case .success(let image) = phase {
                            image.resizable().scaledToFit()
                        }
                    }
                    .frame(maxHeight: 64)
                }
                RoundedRectangle(cornerRadius: 4)
                    .fill(OrgBrandingAdminLogic.color(from: primaryColor))
                    .frame(height: 8)
                Text(L.text("mobile.admin.orgBranding.branding.previewSignIn"))
                    .font(.subheadline.weight(.medium))
                Button(L.text("mobile.admin.orgBranding.branding.previewContinue")) {}
                    .buttonStyle(.borderedProminent)
                    .tint(OrgBrandingAdminLogic.color(from: primaryColor))
                    .allowsHitTesting(false)
            }
            .padding()
            .frame(maxWidth: .infinity)
            .background(LexturesTheme.cardBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 12))
        }
    }

    private var saveSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Button {
                Task { await save() }
            } label: {
                if case .saving = saveStatus {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                } else {
                    Text(L.text("mobile.admin.orgBranding.branding.save"))
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.brandTeal)
            .disabled(!canSave)

            switch saveStatus {
            case .saved:
                Text(L.text("mobile.admin.orgBranding.branding.saved"))
                    .font(.caption)
                    .foregroundStyle(.green)
            case .error(let message):
                Text(message)
                    .font(.caption)
                    .foregroundStyle(.red)
            default:
                EmptyView()
            }
        }
    }

    private var canSave: Bool {
        if case .saving = saveStatus { return false }
        return !loading
            && OrgBrandingAdminLogic.isValidHexColor(primaryColor)
            && OrgBrandingAdminLogic.isValidHexColor(secondaryColor)
    }

    private func load() async {
        guard let token = session.accessToken, !orgId.isEmpty else {
            loading = false
            return
        }
        loading = true
        defer { loading = false }
        do {
            let response = try await LMSAPI.fetchOrgBranding(orgId: orgId, accessToken: token)
            branding = response
            primaryColor = response.primaryColor
            secondaryColor = response.secondaryColor
            emailDisplayName = response.customEmailDisplayName ?? ""
            previewLogoUrl = OrgBrandingAdminLogic.resolveBrandAssetUrl(response.logoUrl)
            saveStatus = .idle
        } catch {
            saveStatus = .error(OrgBrandingAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.orgBranding.branding.loadError"
            ))
        }
    }

    private func save() async {
        guard let token = session.accessToken,
              OrgBrandingAdminLogic.isValidHexColor(primaryColor),
              OrgBrandingAdminLogic.isValidHexColor(secondaryColor) else { return }
        saveStatus = .saving
        let trimmedEmail = emailDisplayName.trimmingCharacters(in: .whitespacesAndNewlines)
        let request = PutOrgBrandingRequest(
            logoUrl: branding?.logoUrl,
            faviconUrl: branding?.faviconUrl,
            primaryColor: OrgBrandingAdminLogic.normalizedHexColor(primaryColor) ?? primaryColor,
            secondaryColor: OrgBrandingAdminLogic.normalizedHexColor(secondaryColor) ?? secondaryColor,
            customDomain: branding?.customDomain,
            customEmailDisplayName: trimmedEmail.isEmpty ? nil : trimmedEmail
        )
        do {
            let response = try await LMSAPI.putOrgBranding(orgId: orgId, body: request, accessToken: token)
            branding = response
            primaryColor = response.primaryColor
            secondaryColor = response.secondaryColor
            emailDisplayName = response.customEmailDisplayName ?? ""
            previewLogoUrl = OrgBrandingAdminLogic.resolveBrandAssetUrl(response.logoUrl)
            saveStatus = .saved
        } catch {
            saveStatus = .error(OrgBrandingAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.orgBranding.branding.saveError"
            ))
        }
    }

    private func uploadLogo(_ item: PhotosPickerItem?) async {
        guard let item, let token = session.accessToken else { return }
        uploadingLogo = true
        defer {
            uploadingLogo = false
            photoItem = nil
        }
        do {
            guard let data = try await item.loadTransferable(type: Data.self) else { return }
            if data.count > OrgBrandingAdminLogic.maxLogoUploadBytes {
                saveStatus = .error(L.text("mobile.admin.orgBranding.branding.fileTooLarge"))
                return
            }
            let upload = try await LMSAPI.uploadOrgBrandingLogo(
                orgId: orgId,
                fileName: "logo.jpg",
                mimeType: "image/jpeg",
                fileData: data,
                accessToken: token
            )
            if let url = upload.url {
                branding?.logoUrl = url
                previewLogoUrl = OrgBrandingAdminLogic.resolveBrandAssetUrl(url)
            }
            saveStatus = .saved
        } catch {
            saveStatus = .error(OrgBrandingAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.orgBranding.branding.uploadError"
            ))
        }
    }
}

private extension Color {
    func toHex() -> String? {
        guard let components = UIColor(self).cgColor.components, components.count >= 3 else {
            return nil
        }
        let red = Int(components[0] * 255)
        let green = Int(components[1] * 255)
        let blue = Int(components[2] * 255)
        return String(format: "#%02X%02X%02X", red, green, blue)
    }
}
