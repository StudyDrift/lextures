import SwiftUI

/// Keyboard-aware free-text composer with local draft persistence (CT.M6 FR-2 / FR-3).
struct ToolComposerView: View {
    @Environment(\.colorScheme) private var colorScheme

    let placeholder: String
    let sendLabel: String
    let cancelLabel: String?
    @Binding var text: String
    var draftKey: String
    var enabled: Bool
    var online: Bool
    var busy: Bool
    var showCancel: Bool = false
    var onSend: () -> Void
    var onCancel: (() -> Void)? = nil
    var onDraftChange: ((String) -> Void)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            if !online {
                Text(L.text("mobile.contentTools.runtime.offlineComposer"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .accessibilityLabel(L.text("mobile.contentTools.runtime.offlineComposer"))
            }
            HStack(alignment: .bottom, spacing: 8) {
                TextField(placeholder, text: $text, axis: .vertical)
                    .lineLimit(1 ... 5)
                    .disabled(!enabled || busy)
                    .onChange(of: text) { _, next in
                        ContentToolDraftStore.save(key: draftKey, text: next)
                        onDraftChange?(next)
                    }
                    .accessibilityLabel(placeholder)

                if busy, let onCancel, showCancel {
                    Button(cancelLabel ?? L.text("mobile.contentTools.runtime.cancel")) {
                        onCancel()
                    }
                    .accessibilityLabel(cancelLabel ?? L.text("mobile.contentTools.runtime.cancel"))
                } else {
                    Button(sendLabel) {
                        onSend()
                    }
                    .disabled(!ContentToolPack2Logic.composerSendEnabled(
                        text: text,
                        readOnly: !enabled,
                        online: online,
                        busy: busy
                    ))
                    .accessibilityLabel(sendLabel)
                }
            }
        }
        .onAppear {
            if text.isEmpty {
                let restored = ContentToolDraftStore.load(key: draftKey)
                if !restored.isEmpty {
                    text = restored
                }
            }
        }
    }

    static func clearDraft(key: String) {
        ContentToolDraftStore.clear(key: key)
    }
}
