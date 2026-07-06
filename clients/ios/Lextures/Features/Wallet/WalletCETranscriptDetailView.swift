import SwiftUI

/// CE transcript detail with awards and PDF preview (M12.2).
struct WalletCETranscriptDetailView: View {
    @Environment(\.colorScheme) private var colorScheme

    let awards: [CETranscriptAward]

    @State private var previewTarget: FilePreviewTarget?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                LMSCard {
                    Button {
                        previewTarget = WalletLogic.ceTranscriptPdfPreviewTarget()
                    } label: {
                        HStack {
                            Image(systemName: "doc.richtext")
                                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                            Text(L.text("mobile.wallet.ceTranscript.previewPdf"))
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Spacer(minLength: 0)
                            Image(systemName: "chevron.right")
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
                        }
                    }
                    .buttonStyle(.plain)
                }

                if awards.isEmpty {
                    Text(L.text("mobile.wallet.ceTranscript.empty"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    LMSCard {
                        VStack(alignment: .leading, spacing: 0) {
                            HStack {
                                headerCell(L.text("mobile.wallet.ceTranscript.course"))
                                headerCell(L.text("mobile.wallet.ceTranscript.ceu"))
                                headerCell(L.text("mobile.wallet.ceTranscript.contactHours"))
                                headerCell(L.text("mobile.wallet.ceTranscript.completed"))
                            }
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            ForEach(awards) { award in
                                Divider()
                                HStack(alignment: .top) {
                                    bodyCell(award.courseTitle, weight: .semibold)
                                    bodyCell(String(format: "%.2f", award.ceuCredit))
                                    bodyCell(String(format: "%.1f", award.contactHours))
                                    bodyCell(WalletLogic.dateLabel(iso: award.completedAt))
                                }
                                .font(.caption)
                            }
                        }
                    }
                }
            }
            .padding(16)
        }
        .navigationTitle(L.text("mobile.wallet.ceTranscript.detailTitle"))
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $previewTarget) { target in
            FilePreviewView(target: target)
        }
    }

    @ViewBuilder
    private func headerCell(_ text: String) -> some View {
        Text(text)
            .frame(maxWidth: .infinity, alignment: .leading)
    }

    @ViewBuilder
    private func bodyCell(_ text: String, weight: Font.Weight = .regular) -> some View {
        Text(text)
            .fontWeight(weight)
            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            .frame(maxWidth: .infinity, alignment: .leading)
    }
}
