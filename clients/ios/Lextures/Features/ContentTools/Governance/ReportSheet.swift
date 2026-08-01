import SwiftUI

/// Student report sheet — category + optional note (CT.M9 FR-10).
struct ReportSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme

    let categories: [String]
    var onSubmit: (String, String?) async -> Bool

    @State private var category: String = "other"
    @State private var note: String = ""
    @State private var busy = false
    @State private var errorText: String?
    @State private var done = false

    var body: some View {
        NavigationStack {
            Form {
                Section(L.text("mobile.contentTools.governance.reportCategory")) {
                    Picker(L.text("mobile.contentTools.governance.reportCategory"), selection: $category) {
                        ForEach(categories, id: \.self) { cat in
                            Text(categoryLabel(cat)).tag(cat)
                        }
                    }
                }
                Section(L.text("mobile.contentTools.governance.reportNote")) {
                    TextField(L.text("mobile.contentTools.governance.reportNotePlaceholder"), text: $note, axis: .vertical)
                        .lineLimit(3 ... 6)
                }
                if let errorText {
                    Text(errorText)
                        .foregroundStyle(LexturesTheme.coral)
                        .font(.caption)
                }
                if done {
                    Text(L.text("mobile.contentTools.governance.reportThanks"))
                        .font(.caption)
                }
            }
            .navigationTitle(L.text("mobile.contentTools.governance.reportTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.contentTools.runtime.cancel")) { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.contentTools.governance.reportSubmit")) {
                        Task { await submit() }
                    }
                    .disabled(busy || done)
                }
            }
        }
        .onAppear {
            if category.isEmpty || !categories.contains(category) {
                category = categories.first ?? "other"
            }
        }
    }

    private func categoryLabel(_ raw: String) -> String {
        let key = "mobile.contentTools.governance.reportCategory.\(raw)"
        let localized = L.text(String.LocalizationValue(key))
        return localized == key ? raw : localized
    }

    private func submit() async {
        busy = true
        errorText = nil
        defer { busy = false }
        let noteTrim = note.trimmingCharacters(in: .whitespacesAndNewlines)
        let succeeded = await onSubmit(category, noteTrim.isEmpty ? nil : noteTrim)
        if succeeded {
            done = true
            try? await Task.sleep(nanoseconds: 600_000_000)
            dismiss()
        } else {
            errorText = L.text("mobile.contentTools.governance.reportError")
        }
    }
}
