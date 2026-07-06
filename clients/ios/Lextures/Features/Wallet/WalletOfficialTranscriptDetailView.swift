import SwiftUI

/// Official transcript requests with link to web ordering flow (M12.2).
struct WalletOfficialTranscriptDetailView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let requests: [TranscriptRequestSummary]

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                LMSCard {
                    VStack(alignment: .leading, spacing: 10) {
                        Text(L.text("mobile.wallet.officialTranscriptsHint"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        Button {
                            openURL(WalletLogic.officialTranscriptWebURL())
                        } label: {
                            HStack {
                                Image(systemName: "safari")
                                Text(L.text("mobile.wallet.openWebRequest"))
                            }
                            .font(.subheadline.weight(.semibold))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 10)
                        }
                        .buttonStyle(.borderedProminent)
                    }
                }

                LMSSectionHeader(title: L.text("mobile.wallet.requestHistory"), systemImage: "clock.arrow.circlepath")
                if requests.isEmpty {
                    Text(L.text("mobile.wallet.noRequests"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(requests) { request in
                        LMSCard {
                            VStack(alignment: .leading, spacing: 6) {
                                HStack {
                                    Text(WalletLogic.dateLabel(iso: request.requestedAt))
                                        .font(.subheadline.weight(.semibold))
                                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                    Spacer(minLength: 0)
                                    Text(WalletLogic.transcriptStatusLabel(request.status))
                                        .font(.caption.weight(.semibold))
                                        .padding(.horizontal, 8)
                                        .padding(.vertical, 4)
                                        .background(statusColor(request.status).opacity(0.15))
                                        .foregroundStyle(statusColor(request.status))
                                        .clipShape(Capsule())
                                }
                                Text(WalletLogic.deliveryTypeLabel(request.deliveryType))
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                if let error = request.errorMessage, !error.isEmpty, request.status == "failed" {
                                    Text(error)
                                        .font(.caption)
                                        .foregroundStyle(LexturesTheme.coral)
                                }
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }
                    }
                }
            }
            .padding(16)
        }
        .navigationTitle(L.text("mobile.wallet.officialTranscripts.detailTitle"))
        .navigationBarTitleDisplayMode(.inline)
    }

    private func statusColor(_ status: String) -> Color {
        switch status {
        case "submitted": return .green
        case "failed": return LexturesTheme.coral
        default: return .orange
        }
    }
}
