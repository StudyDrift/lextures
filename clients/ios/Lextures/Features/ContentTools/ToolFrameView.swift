import SwiftUI

struct ToolFrameView<Content: View>: View {
    @Environment(\.colorScheme) private var colorScheme
    let title: String
    let status: String
    let syncStatus: ContentToolHostLogic.SyncStatus
    let score: ToolScore?
    let readOnly: Bool
    let readOnlyMessage: String?
    let studentResetAllowed: Bool
    var onReset: (() -> Void)?
    @ViewBuilder var content: () -> Content

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    HStack(spacing: 8) {
                        chip(statusLabel)
                        if let sync = syncLabel { chip(sync) }
                        if let score {
                            chip("\(L.text("mobile.contentTools.runtime.score")) \(score.raw)/\(score.max)")
                        }
                    }
                }
                Spacer(minLength: 8)
                Menu {
                    Button(L.text("mobile.contentTools.runtime.help")) {}
                    if studentResetAllowed, let onReset, !readOnly {
                        Button(L.text("mobile.contentTools.runtime.reset"), role: .destructive, action: onReset)
                    }
                    Button(L.text("mobile.contentTools.runtime.report")) {}
                } label: {
                    Image(systemName: "ellipsis.circle")
                        .font(.body)
                        .frame(minWidth: 44, minHeight: 44)
                }
            }
            if let readOnlyMessage, !readOnlyMessage.isEmpty {
                Text(readOnlyMessage)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            content()
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.sceneBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        .opacity(readOnly ? 0.85 : 1)
        .accessibilityElement(children: .contain)
        .accessibilityLabel(ContentToolHostLogic.accessibleName(title: title, status: status))
    }

    private var statusLabel: String {
        switch ContentToolHostLogic.statusChip(status) {
        case "completed": return L.text("mobile.contentTools.runtime.status.completed")
        case "submitted": return L.text("mobile.contentTools.runtime.status.submitted")
        case "in_progress": return L.text("mobile.contentTools.runtime.status.in_progress")
        default: return L.text("mobile.contentTools.runtime.status.not_started")
        }
    }

    private var syncLabel: String? {
        switch syncStatus {
        case .saving: return L.text("mobile.contentTools.runtime.saving")
        case .saved: return L.text("mobile.contentTools.runtime.saved")
        case .unsynced: return L.text("mobile.contentTools.runtime.unsynced")
        case .error: return L.text("mobile.contentTools.runtime.retry")
        case .idle: return nil
        }
    }

    @ViewBuilder
    private func chip(_ label: String) -> some View {
        Text(label)
            .font(.caption2.weight(.semibold))
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(LexturesTheme.textSecondary(for: colorScheme).opacity(0.12))
            .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))
    }
}
