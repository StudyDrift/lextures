import SwiftUI

/// Co-curricular record detail: achievements, generate, verify, share, PDF (M12.2).
struct WalletCCRDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let achievements: [CCRAchievement]
    let documents: [CCRDocument]
    let onUpdate: ([CCRAchievement], [CCRDocument]) -> Void

    @State private var localAchievements: [CCRAchievement] = []
    @State private var localDocuments: [CCRDocument] = []
    @State private var sharePublicly = false
    @State private var generating = false
    @State private var actionError: String?
    @State private var copiedUrl: String?
    @State private var previewTarget: FilePreviewTarget?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                if let actionError {
                    LMSErrorBanner(message: actionError)
                }

                LMSCard {
                    Toggle(isOn: $sharePublicly) {
                        Text(L.text("mobile.wallet.ccr.sharePublicly"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    }
                    Button {
                        Task { await generate() }
                    } label: {
                        HStack {
                            Image(systemName: "sparkles")
                            Text(generating
                                ? L.text("mobile.wallet.ccr.generating")
                                : L.text("mobile.wallet.ccr.generate"))
                        }
                        .font(.subheadline.weight(.semibold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(generating)
                }

                if localAchievements.isEmpty {
                    Text(L.text("mobile.wallet.ccr.emptyAchievements"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    LMSSectionHeader(title: L.text("mobile.wallet.ccr.achievements"), systemImage: "star.fill")
                    ForEach(groupedAchievements.keys.sorted(), id: \.self) { type in
                        Text(WalletLogic.achievementTypeLabel(type))
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        ForEach(groupedAchievements[type] ?? []) { item in
                            LMSCard {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(item.title)
                                        .font(.subheadline.weight(.semibold))
                                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                    if let description = item.description, !description.isEmpty {
                                        Text(description)
                                            .font(.caption)
                                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    }
                                    Text(WalletLogic.dateLabel(iso: item.issuedAt))
                                        .font(.caption)
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                }
                                .frame(maxWidth: .infinity, alignment: .leading)
                            }
                        }
                    }
                }

                if !localDocuments.isEmpty {
                    LMSSectionHeader(title: L.text("mobile.wallet.ccr.documents"), systemImage: "doc.fill")
                    ForEach(localDocuments) { doc in
                        documentCard(doc)
                    }
                }

                if let copiedUrl {
                    Text(L.text("mobile.wallet.copied"))
                        .font(.caption)
                        .foregroundStyle(.green)
                        .accessibilityLabel(copiedUrl)
                }
            }
            .padding(16)
        }
        .navigationTitle(L.text("mobile.wallet.ccr.detailTitle"))
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $previewTarget) { target in
            FilePreviewView(target: target)
        }
        .onAppear {
            localAchievements = achievements
            localDocuments = documents
        }
    }

    private var groupedAchievements: [String: [CCRAchievement]] {
        Dictionary(grouping: localAchievements, by: \.type)
    }

    @ViewBuilder
    private func documentCard(_ doc: CCRDocument) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(WalletLogic.dateLabel(iso: doc.generatedAt))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                if let url = doc.verificationUrl {
                    Text(url)
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .lineLimit(2)
                    HStack(spacing: 8) {
                        ShareLink(item: url) {
                            actionChip(L.text("mobile.wallet.shareVerify"), systemImage: "square.and.arrow.up")
                        }
                        Button {
                            if let link = URL(string: url) { openURL(link) }
                        } label: {
                            actionChip(L.text("mobile.wallet.openVerify"), systemImage: "checkmark.seal")
                        }
                        .buttonStyle(.plain)
                        Button {
                            UIPasteboard.general.string = url
                            copiedUrl = url
                        } label: {
                            actionChip(L.text("mobile.wallet.copyVerifyLink"), systemImage: "link")
                        }
                        .buttonStyle(.plain)
                    }
                } else {
                    Text(L.text("mobile.wallet.verification.private"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                HStack(spacing: 8) {
                    Button {
                        previewTarget = WalletLogic.ccrPdfPreviewTarget(documentId: doc.id)
                    } label: {
                        actionChip(L.text("mobile.wallet.downloadPdf"), systemImage: "doc.richtext")
                    }
                    .buttonStyle(.plain)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private func actionChip(_ title: String, systemImage: String) -> some View {
        Label(title, systemImage: systemImage)
            .font(.caption.weight(.semibold))
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(LexturesTheme.cardBackground(for: colorScheme))
            .clipShape(Capsule())
    }

    private func generate() async {
        guard let token = session.accessToken else { return }
        generating = true
        actionError = nil
        defer { generating = false }
        do {
            let result = try await LMSAPI.generateMyCCR(sharePublicly: sharePublicly, accessToken: token)
            localDocuments = [result.document] + localDocuments.filter { $0.id != result.document.id }
            if let achievements = result.achievements {
                localAchievements = achievements
            }
            copiedUrl = result.verificationUrl
            onUpdate(localAchievements, localDocuments)
        } catch {
            actionError = L.text("mobile.wallet.ccr.generateError")
        }
    }
}
