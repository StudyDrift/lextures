import SwiftUI

/// Credential detail with verify, share, LinkedIn, and Open Badge export (M9.3).
struct CredentialDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let credential: IssuedCredentialSummary

    @State private var linkedInLoading = false
    @State private var badgeExportLoading = false
    @State private var actionError: String?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                LMSCard {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(credential.title)
                            .font(LexturesTheme.displayFont(20))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(CredentialsLogic.sourceTypeLabel(credential.sourceType))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        Text(L.format(
                            "mobile.credentials.issued",
                            CredentialsLogic.issuedDateLabel(iso: credential.issuedAt)
                        ))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        if credential.revoked {
                            Text(L.text("mobile.credentials.revoked"))
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(LexturesTheme.coral)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }

                if let actionError {
                    LMSErrorBanner(message: actionError)
                }

                LMSSectionHeader(title: L.text("mobile.credentials.shareTitle"), systemImage: "square.and.arrow.up")
                LMSCard {
                    VStack(spacing: 10) {
                        ShareLink(
                            item: credential.verificationUrl,
                            subject: Text(credential.title),
                            message: Text(L.format("mobile.credentials.shareText", credential.title, credential.verificationUrl))
                        ) {
                            actionRow(
                                title: L.text("mobile.credentials.shareVerify"),
                                systemImage: "link"
                            )
                        }

                        Button {
                            openURL(URL(string: credential.verificationUrl)!)
                        } label: {
                            actionRow(
                                title: L.text("mobile.credentials.openVerify"),
                                systemImage: "checkmark.seal"
                            )
                        }
                        .buttonStyle(.plain)

                        Button {
                            Task { await shareLinkedIn() }
                        } label: {
                            actionRow(
                                title: linkedInLoading
                                    ? L.text("mobile.credentials.openingLinkedIn")
                                    : L.text("mobile.credentials.addLinkedIn"),
                                systemImage: "link.badge.plus"
                            )
                        }
                        .buttonStyle(.plain)
                        .disabled(linkedInLoading || credential.revoked)

                        Button {
                            Task { await exportOpenBadge() }
                        } label: {
                            actionRow(
                                title: badgeExportLoading
                                    ? L.text("mobile.credentials.exportingBadge")
                                    : L.text("mobile.credentials.exportBadge"),
                                systemImage: "doc.badge.gearshape"
                            )
                        }
                        .buttonStyle(.plain)
                        .disabled(badgeExportLoading || credential.revoked)
                    }
                }
            }
            .padding(16)
        }
        .navigationTitle(L.text("mobile.credentials.detailTitle"))
        .navigationBarTitleDisplayMode(.inline)
    }

    @ViewBuilder
    private func actionRow(title: String, systemImage: String) -> some View {
        HStack(spacing: 12) {
            Image(systemName: systemImage)
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                .frame(width: 28)
            Text(title)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Spacer(minLength: 0)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .contentShape(Rectangle())
    }

    private func shareLinkedIn() async {
        guard let token = session.accessToken else { return }
        linkedInLoading = true
        actionError = nil
        defer { linkedInLoading = false }
        do {
            let params = try await LMSAPI.fetchCredentialLinkedInParams(
                credentialId: credential.id,
                accessToken: token
            )
            try await LMSAPI.recordCredentialShare(
                credentialId: credential.id,
                channel: "linkedin",
                accessToken: token
            )
            if let url = URL(string: params.url) {
                openURL(url)
            }
        } catch {
            actionError = L.text("mobile.credentials.linkedInError")
        }
    }

    private func exportOpenBadge() async {
        guard let token = session.accessToken else { return }
        badgeExportLoading = true
        actionError = nil
        defer { badgeExportLoading = false }
        do {
            let export = try await LMSAPI.fetchCredentialBadgeExportUrl(
                credentialId: credential.id,
                accessToken: token
            )
            try await LMSAPI.recordCredentialShare(
                credentialId: credential.id,
                channel: "badge_export",
                accessToken: token
            )
            if let url = URL(string: export.downloadUrl) {
                openURL(url)
            }
        } catch {
            actionError = L.text("mobile.credentials.badgeExportError")
        }
    }
}