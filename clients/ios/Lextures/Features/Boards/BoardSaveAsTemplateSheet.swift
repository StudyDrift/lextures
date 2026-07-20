import SwiftUI

/// Save current board as a reusable template (MOB.8 / VC.8).
struct BoardSaveAsTemplateSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let courseCode: String
    let boardId: String
    let defaultTitle: String
    var onSaved: ((BoardTemplate) -> Void)?

    @State private var title: String = ""
    @State private var description: String = ""
    @State private var scope: String = "course"
    @State private var includePosts = false
    @State private var saving = false
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField(L.text("mobile.boards.templates.saveTitle"), text: $title)
                    TextField(L.text("mobile.boards.templates.saveDescription"), text: $description, axis: .vertical)
                        .lineLimit(2 ... 4)
                }
                Section {
                    Picker(L.text("mobile.boards.templates.scope"), selection: $scope) {
                        Text(L.text("mobile.boards.templates.scopeCourse")).tag("course")
                        Text(L.text("mobile.boards.templates.scopeOrg")).tag("org")
                    }
                    Toggle(L.text("mobile.boards.templates.includePosts"), isOn: $includePosts)
                } footer: {
                    Text(L.text("mobile.boards.templates.includePostsHint"))
                }
                if let errorMessage {
                    Section {
                        Text(errorMessage).foregroundStyle(.red)
                    }
                }
            }
            .navigationTitle(L.text("mobile.boards.templates.saveAction"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.common.save")) {
                        Task { await save() }
                    }
                    .disabled(title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || saving)
                }
            }
            .onAppear {
                if title.isEmpty { title = defaultTitle }
            }
        }
    }

    private func save() async {
        guard let token = session.accessToken else { return }
        saving = true
        errorMessage = nil
        defer { saving = false }
        do {
            let template = try await LMSAPI.saveBoardAsTemplate(
                courseCode: courseCode,
                boardId: boardId,
                scope: scope,
                title: title.trimmingCharacters(in: .whitespacesAndNewlines),
                description: description.trimmingCharacters(in: .whitespacesAndNewlines),
                includePosts: includePosts,
                accessToken: token
            )
            BoardsAdvancedObservability.record(
                "board_saved_as_template",
                attributes: ["scope": scope, "include_posts": includePosts ? "1" : "0"]
            )
            onSaved?(template)
            dismiss()
        } catch {
            errorMessage = L.text("mobile.boards.templates.saveError")
        }
    }
}
