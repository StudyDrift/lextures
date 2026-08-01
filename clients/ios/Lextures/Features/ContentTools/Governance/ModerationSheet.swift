import SwiftUI

/// Staff moderation controls (CT.M9 FR-11). Hidden for non-entitled viewers by the caller.
struct ModerationSheet: View {
    @Environment(\.dismiss) private var dismiss
    let items: [ContentToolModerationAction]
    var onModerate: (String) async -> Bool

    @State private var busy = false
    @State private var errorText: String?
    @State private var forbidden = false

    var body: some View {
        NavigationStack {
            List {
                if items.isEmpty {
                    Text(L.text("mobile.contentTools.governance.moderateEmpty"))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(items, id: \.id) { item in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(item.action)
                                .font(.subheadline.weight(.semibold))
                            if let category = item.category, !category.isEmpty {
                                Text(category).font(.caption).foregroundStyle(.secondary)
                            }
                            Text(item.createdAt).font(.caption2).foregroundStyle(.secondary)
                        }
                    }
                }
                if let errorText {
                    Text(errorText)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.coral)
                }
            }
            .navigationTitle(L.text("mobile.contentTools.governance.moderateTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.contentTools.runtime.cancel")) { dismiss() }
                }
                ToolbarItemGroup(placement: .bottomBar) {
                    Button(L.text("mobile.contentTools.governance.moderateHide")) {
                        Task { await run("hidden") }
                    }
                    .disabled(busy || forbidden)
                    Button(L.text("mobile.contentTools.governance.moderateRemove"), role: .destructive) {
                        Task { await run("removed") }
                    }
                    .disabled(busy || forbidden)
                    Button(L.text("mobile.contentTools.governance.moderateRestore")) {
                        Task { await run("restored") }
                    }
                    .disabled(busy || forbidden)
                }
            }
        }
    }

    private func run(_ action: String) async {
        busy = true
        errorText = nil
        defer { busy = false }
        let ok = await onModerate(action)
        if !ok {
            if forbidden {
                errorText = L.text("mobile.contentTools.governance.moderateForbidden")
            } else {
                errorText = L.text("mobile.contentTools.governance.moderateError")
            }
        } else {
            dismiss()
        }
    }

    func markForbidden() {
        forbidden = true
        errorText = L.text("mobile.contentTools.governance.moderateForbidden")
    }
}
