import SwiftUI

/// Neutral placeholder naming why a tool did not mount (CT.M9 policy-blocked states).
struct PolicyBlockedPlaceholder: View {
    let decision: ContentToolGovernanceLogic.MountDecision
    var toolName: String?
    var onRefresh: (() -> Void)?

    var body: some View {
        ToolPlaceholderView(
            reason: .unavailable,
            toolName: toolName,
            message: L.text(String.LocalizationValue(ContentToolGovernanceLogic.reasonMessageKey(decision))),
            onOpenInBrowser: nil
        )
        .overlay(alignment: .bottomTrailing) {
            if decision == .blockStalePolicy, let onRefresh {
                Button(L.text("mobile.contentTools.governance.refreshPolicy"), action: onRefresh)
                    .buttonStyle(.bordered)
                    .padding(8)
            }
        }
    }
}
