import SwiftUI

/// Dismiss reason + optional note sheet for a checklist item (CC.9).
struct ChecklistDismissSheet: View {
    let item: ChecklistItem
    let isOnline: Bool
    let onCancel: () -> Void
    let onConfirm: (ChecklistDismissReason, String?) -> Void

    @State private var reason: ChecklistDismissReason = .notApplicable
    @State private var note = ""
    @State private var showNote = false

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text(item.title)
                        .font(.body.weight(.medium))
                    Text(L.text("mobile.checklist.dismissDialogHelp"))
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                Section(L.text("mobile.checklist.dismissReasonLabel")) {
                    Picker(L.text("mobile.checklist.dismissReasonLabel"), selection: $reason) {
                        ForEach(ChecklistDismissReason.allCases) { dismissReason in
                            Text(L.text(String.LocalizationValue(dismissReason.labelKey))).tag(dismissReason)
                        }
                    }
                    .pickerStyle(.inline)
                    .labelsHidden()
                }
                Section {
                    if showNote {
                        TextField(
                            L.text("mobile.checklist.dismissNotePlaceholder"),
                            text: $note,
                            axis: .vertical
                        )
                        .lineLimit(3 ... 6)
                    } else {
                        Button(L.text("mobile.checklist.addNote")) {
                            showNote = true
                        }
                    }
                }
                if !isOnline {
                    Text(L.text("mobile.checklist.offlineMutations"))
                        .foregroundStyle(.secondary)
                }
            }
            .navigationTitle(L.text("mobile.checklist.dismissDialogTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.checklist.dismissCancel"), action: onCancel)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.checklist.dismissConfirm")) {
                        onConfirm(reason, showNote ? note : nil)
                    }
                    .disabled(!isOnline)
                }
            }
        }
    }
}
