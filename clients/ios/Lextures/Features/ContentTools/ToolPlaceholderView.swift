import SwiftUI

enum ToolPlaceholderReason {
    case loading
    case unavailable
    case openInBrowser
    case readOnlyArchived
    case readOnlyPastDue
    case readOnlyPreview
    case maintenance
    case updateRequired
}

struct ToolPlaceholderView: View {
    @Environment(\.colorScheme) private var colorScheme
    let reason: ToolPlaceholderReason
    var toolName: String?
    var message: String?
    var onOpenInBrowser: (() -> Void)?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(resolvedMessage)
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            if reason == .openInBrowser, let onOpenInBrowser {
                Button(L.text("mobile.contentTools.runtime.openInBrowserAction"), action: onOpenInBrowser)
                    .buttonStyle(.bordered)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.sceneBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
        .accessibilityElement(children: .combine)
        .accessibilityLabel(resolvedMessage)
    }

    private var resolvedMessage: String {
        if let message, !message.isEmpty { return message }
        switch reason {
        case .loading:
            return L.text("mobile.contentTools.runtime.loading")
        case .unavailable, .maintenance:
            return L.text("mobile.contentTools.runtime.unavailable")
        case .openInBrowser:
            if let toolName, !toolName.isEmpty {
                return L.format("mobile.contentTools.runtime.openInBrowserNamed", toolName)
            }
            return L.text("mobile.contentTools.runtime.openInBrowser")
        case .readOnlyArchived:
            return L.text("mobile.contentTools.runtime.readOnlyArchived")
        case .readOnlyPastDue:
            return L.text("mobile.contentTools.runtime.readOnlyPastDue")
        case .readOnlyPreview:
            return L.text("mobile.contentTools.runtime.readOnlyPreview")
        case .updateRequired:
            return L.text("mobile.contentTools.runtime.openInBrowser")
        }
    }
}
