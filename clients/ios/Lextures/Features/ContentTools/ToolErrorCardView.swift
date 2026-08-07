import SwiftUI

struct ToolErrorCardView: View {
    @Environment(\.colorScheme) private var colorScheme
    let toolName: String
    var message: String?
    var onRetry: () -> Void

    private var resolvedMessage: String {
        if let message, !message.isEmpty { return message }
        return L.text("mobile.contentTools.runtime.errorTitle")
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(toolName)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(resolvedMessage)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Button(L.text("mobile.contentTools.runtime.retry"), action: onRetry)
                .buttonStyle(.bordered)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.sceneBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(toolName). \(resolvedMessage)")
    }
}
