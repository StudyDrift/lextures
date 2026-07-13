import PhotosUI
import SwiftUI

/// Branding card: logo upload, colors, email sender, custom domain read + web link-out (M14.5).
struct OrgBrandingView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let orgId: String
    @Binding var branding: OrgBrandingResponse
    @Binding var statusMessage: String?
    @Binding var errorMessage: String?

    @State private var primaryColor = OrgBrandingAdminLogic.defaultPrimaryColor
    @State private var secondaryColor = OrgBrandingAdminLogic.defaultSecondaryColor
    @State private var emailDisplayName = ""
    @State private var logoUrl: String?
    @State private var faviconUrl: String?
    @State private var customDomain: String?
    @State private var contrastWarning = false
    @State private var contrastRatio: Double?
    @State private var saving = false
    @State private var uploading = false
    @State private var photoItem: PhotosPickerItem?

    var body: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 14) {
                Text(L.text("mobile.admin.orgBranding.branding.title"))
                    .font(.headline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(L.text("mobile.admin.orgBranding.branding.intro"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                logoSection
                colorField(
                    title: L.text("mobile.admin.orgBranding.branding.primaryColor"),
                    value: $primaryColor
                )
                colorField(
                    title: L.text("mobile.admin.orgBranding.branding.secondaryColor"),
                    value: $secondaryColor
                )

                if OrgBrandingAdminLogic.hasContrastWarning(
                    primaryColor: primaryColor,
                    serverWarning: contrastWarning,
                    serverRatio: contrastRatio
                ) {
                    Text(contrastWarningText)
                        .font(.caption)
                        .foregroundStyle(.orange)
                        .accessibilityLabel(contrastWarningText)
                }

                previewStrip

                VStack(alignment: .leading, spacing: 6) {
                    Text(L.text("mobile.admin.orgBranding.branding.emailDisplayName"))
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    TextField(
                        L.text("mobile.admin.orgBranding.branding.emailDisplayNamePlaceholder"),
                        text: $emailDisplayName
                    )
                    .textInputAutocapitalization(.words)
                    .padding(12)
                    .background(LexturesTheme.sceneBackground(for: colorScheme), in: RoundedRectangle(cornerRadius: 10))
                    Text(L.text("mobile.admin.orgBranding.branding.emailDisplayNameHint"))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                customDomainSection

                Button {
                    Task { await save() }
                } label: {
                    if saving {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                    } else {
                        Text(L.text("mobile.admin.orgBranding.branding.save"))
                            .frame(maxWidth: .infinity)
                    }
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.brandTeal)
                .disabled(saving || uploading || !colorsValid)
                .frame(minHeight: 44)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .onAppear { apply(branding) }
        .onChange(of: branding) { _, next in apply(next) }
    }

    private var colorsValid: Bool {
        OrgBrandingAdminLogic.isValidHexColor(primaryColor)
            && OrgBrandingAdminLogic.isValidHexColor(secondaryColor)
    }

    private var contrastWarningText: String {
        if let ratio = contrastRatio ?? OrgBrandingAdminLogic.contrastRatioAgainstWhite(primaryColor) {
            return L.format(
                "mobile.admin.orgBranding.branding.contrastWarningWithRatio",
                String(format: "%.2f", ratio)
            )
        }
        return L.text("mobile.admin.orgBranding.branding.contrastWarning")
    }

    private var logoSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.admin.orgBranding.branding.logo"))
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            HStack(spacing: 12) {
                if let url = OrgBrandingAdminLogic.resolveAssetURL(logoUrl) {
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .success(let image):
                            image.resizable().scaledToFit()
                        default:
                            RoundedRectangle(cornerRadius: 8)
                                .fill(LexturesTheme.sceneBackground(for: colorScheme))
                        }
                    }
                    .frame(width: 72, height: 72)
                    .clipShape(RoundedRectangle(cornerRadius: 8))
                    .accessibilityLabel(L.text("mobile.admin.orgBranding.branding.logoPreview"))
                } else {
                    RoundedRectangle(cornerRadius: 8)
                        .strokeBorder(LexturesTheme.textSecondary(for: colorScheme).opacity(0.3))
                        .frame(width: 72, height: 72)
                        .overlay {
                            Image(systemName: "photo")
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        .accessibilityLabel(L.text("mobile.admin.orgBranding.branding.logoPreview"))
                }

                PhotosPicker(selection: $photoItem, matching: .images) {
                    Label(
                        uploading
                            ? L.text("mobile.admin.orgBranding.branding.uploading")
                            : L.text("mobile.admin.orgBranding.branding.uploadLogo"),
                        systemImage: "square.and.arrow.up"
                    )
                    .frame(minHeight: 44)
                }
                .disabled(uploading || orgId.isEmpty)
                .onChange(of: photoItem) { _, item in
                    Task { await uploadLogo(item) }
                }
            }

            Text(L.text("mobile.admin.orgBranding.branding.logoHint"))
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private var previewStrip: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(L.text("mobile.admin.orgBranding.branding.preview"))
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            RoundedRectangle(cornerRadius: 8)
                .fill(OrgBrandingAdminLogic.color(fromHex: primaryColor) ?? Color.indigo)
                .frame(height: 36)
                .overlay {
                    Text(L.text("mobile.admin.orgBranding.branding.previewButton"))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.white)
                }
                .accessibilityLabel(L.text("mobile.admin.orgBranding.branding.preview"))
        }
    }

    private var customDomainSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(L.text("mobile.admin.orgBranding.branding.customDomain"))
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(
                customDomain?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
                    ? (customDomain ?? L.text("mobile.emDash"))
                    : L.text("mobile.admin.orgBranding.branding.customDomainNone")
            )
            .font(.body.monospaced())
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Text(L.text("mobile.admin.orgBranding.branding.customDomainHint"))
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            Button {
                openURL(AppConfiguration.webURL(path: OrgBrandingAdminLogic.webBrandingPath()))
            } label: {
                HStack(spacing: 8) {
                    Image(systemName: "safari")
                    Text(L.text("mobile.admin.orgBranding.webTitle"))
                    Spacer()
                    Image(systemName: "arrow.up.right")
                        .font(.caption.weight(.semibold))
                }
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.brandTeal)
                .frame(minHeight: 44)
            }
            .buttonStyle(.plain)
            .accessibilityHint(L.text("mobile.admin.orgBranding.branding.customDomainHint"))
        }
    }

    private func colorField(title: String, value: Binding<String>) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            HStack(spacing: 10) {
                RoundedRectangle(cornerRadius: 8)
                    .fill(OrgBrandingAdminLogic.color(fromHex: value.wrappedValue) ?? Color.gray.opacity(0.3))
                    .frame(width: 44, height: 44)
                    .overlay(
                        RoundedRectangle(cornerRadius: 8)
                            .strokeBorder(LexturesTheme.textSecondary(for: colorScheme).opacity(0.25))
                    )
                    .accessibilityHidden(true)
                TextField("#4F46E5", text: value)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .font(.body.monospaced())
                    .padding(12)
                    .background(LexturesTheme.sceneBackground(for: colorScheme), in: RoundedRectangle(cornerRadius: 10))
                    .accessibilityLabel(title)
            }
        }
    }

    private func apply(_ data: OrgBrandingResponse) {
        logoUrl = data.logoUrl
        faviconUrl = data.faviconUrl
        primaryColor = data.primaryColor
        secondaryColor = data.secondaryColor
        emailDisplayName = data.customEmailDisplayName ?? ""
        customDomain = data.customDomain
        contrastWarning = data.contrastWarningPrimary ?? false
        contrastRatio = data.contrastRatioPrimary
    }

    @MainActor
    private func save() async {
        guard let token = session.accessToken, !orgId.isEmpty else { return }
        guard colorsValid else {
            errorMessage = L.text("mobile.admin.orgBranding.branding.invalidColor")
            return
        }
        saving = true
        errorMessage = nil
        statusMessage = nil
        defer { saving = false }
        do {
            let body = OrgBrandingAdminLogic.brandingPutBody(
                logoUrl: logoUrl,
                faviconUrl: faviconUrl,
                primaryColor: primaryColor,
                secondaryColor: secondaryColor,
                customEmailDisplayName: emailDisplayName
            )
            let saved = try await LMSAPI.putOrgBranding(orgId: orgId, body: body, accessToken: token)
            branding = saved
            apply(saved)
            statusMessage = L.text("mobile.admin.orgBranding.branding.saved")
        } catch {
            errorMessage = OrgBrandingAdminLogic.userFacingError(error)
        }
    }

    @MainActor
    private func uploadLogo(_ item: PhotosPickerItem?) async {
        guard let item, let token = session.accessToken, !orgId.isEmpty else { return }
        uploading = true
        errorMessage = nil
        statusMessage = nil
        defer {
            uploading = false
            photoItem = nil
        }
        do {
            guard let data = try await item.loadTransferable(type: Data.self) else { return }
            let upload = try await LMSAPI.uploadOrgBrandingLogo(
                orgId: orgId,
                fileData: data,
                fileName: "logo.jpg",
                mimeType: "image/jpeg",
                accessToken: token
            )
            if let url = upload.url {
                logoUrl = url
                statusMessage = L.text("mobile.admin.orgBranding.branding.logoUploaded")
            }
        } catch {
            errorMessage = OrgBrandingAdminLogic.userFacingError(error)
        }
    }
}
