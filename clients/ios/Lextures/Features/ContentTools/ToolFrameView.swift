import SwiftUI

struct ToolFrameView<Content: View, Disclosure: View>: View {
    @Environment(\.colorScheme) private var colorScheme
    let title: String
    let status: String
    let syncStatus: ContentToolHostLogic.SyncStatus
    let score: ToolScore?
    let readOnly: Bool
    let readOnlyMessage: String?
    let studentResetAllowed: Bool
    var showSandboxBadge: Bool = false
    var showNonConformantNote: Bool = false
    var canReport: Bool = true
    var canModerate: Bool = false
    var onReset: (() -> Void)?
    var onReport: (() -> Void)?
    var onModerate: (() -> Void)?
    @ViewBuilder var disclosure: () -> Disclosure
    @ViewBuilder var content: () -> Content

    init(
        title: String,
        status: String,
        syncStatus: ContentToolHostLogic.SyncStatus,
        score: ToolScore?,
        readOnly: Bool,
        readOnlyMessage: String?,
        studentResetAllowed: Bool,
        showSandboxBadge: Bool = false,
        showNonConformantNote: Bool = false,
        canReport: Bool = true,
        canModerate: Bool = false,
        onReset: (() -> Void)? = nil,
        onReport: (() -> Void)? = nil,
        onModerate: (() -> Void)? = nil,
        @ViewBuilder disclosure: @escaping () -> Disclosure = { EmptyView() },
        @ViewBuilder content: @escaping () -> Content
    ) {
        self.title = title
        self.status = status
        self.syncStatus = syncStatus
        self.score = score
        self.readOnly = readOnly
        self.readOnlyMessage = readOnlyMessage
        self.studentResetAllowed = studentResetAllowed
        self.showSandboxBadge = showSandboxBadge
        self.showNonConformantNote = showNonConformantNote
        self.canReport = canReport
        self.canModerate = canModerate
        self.onReset = onReset
        self.onReport = onReport
        self.onModerate = onModerate
        self.disclosure = disclosure
        self.content = content
    }

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
                        if showSandboxBadge {
                            chip(L.text("mobile.contentTools.sandbox.badge"))
                        }
                    }
                }
                Spacer(minLength: 8)
                Menu {
                    Button(L.text("mobile.contentTools.runtime.help")) {}
                    if ContentToolGovernanceLogic.studentResetVisible(
                        studentResetAllowed: studentResetAllowed,
                        readOnly: readOnly
                    ), let onReset {
                        Button(L.text("mobile.contentTools.runtime.reset"), role: .destructive, action: onReset)
                    }
                    if canReport, let onReport {
                        Button(L.text("mobile.contentTools.governance.reportTitle"), action: onReport)
                    }
                    if canModerate, let onModerate {
                        Button(L.text("mobile.contentTools.governance.moderateTitle"), action: onModerate)
                    }
                } label: {
                    Image(systemName: "ellipsis.circle")
                        .font(.body)
                        .accessibilityLabel(L.text("mobile.contentTools.runtime.help"))
                        .frame(minWidth: 44, minHeight: 44)
                }
            }
            if let readOnlyMessage, !readOnlyMessage.isEmpty {
                Text(readOnlyMessage)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            if showNonConformantNote {
                Text(L.text("mobile.contentTools.governance.nonConformant"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .accessibilityLabel(L.text("mobile.contentTools.governance.nonConformant"))
            }
            // FR-6: disclosure is native frame chrome above tool content.
            disclosure()
            content()
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.sceneBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        .opacity(readOnly ? 0.85 : 1)
        .accessibilityElement(children: .contain)
        .accessibilityLabel(accessibilityName)
    }

    private var accessibilityName: String {
        var parts = [ContentToolHostLogic.accessibleName(title: title, status: status)]
        if showSandboxBadge {
            parts.append(L.text("mobile.contentTools.sandbox.badge"))
        }
        if showNonConformantNote {
            parts.append(L.text("mobile.contentTools.governance.nonConformant"))
        }
        return parts.joined(separator: ", ")
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
