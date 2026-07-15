import SwiftUI

/// Platform system prompt editor (M14.7). Changes are audited server-side.
struct SystemPromptsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var prompts: [SystemPromptItem] = []
    @State private var selectedKey = ""
    @State private var draft = ""
    @State private var loading = true
    @State private var saveStatus: AiModelsAdminLogic.SaveStatus = .idle
    @State private var showEditor = false

    private var canView: Bool {
        AiModelsAdminLogic.canView(features: shell.platformFeatures, permissions: shell.permissions)
    }

    private var selectedPrompt: SystemPromptItem? {
        prompts.first(where: { $0.key == selectedKey })
    }

    var body: some View {
        Group {
            if canView { content } else { accessDenied }
        }
        .navigationTitle(L.text("mobile.admin.ai.prompts.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task { if canView { await load() } }
        .sheet(isPresented: $showEditor) {
            promptEditorSheet
        }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.ai.accessDenied.title"),
            message: L.text("mobile.admin.ai.accessDenied.message")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.ai.prompts.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if case let .error(message) = saveStatus {
                        LMSErrorBanner(message: message)
                    }

                    if loading {
                        LMSSkeletonList(count: 3)
                    } else if prompts.isEmpty {
                        LMSEmptyState(
                            systemImage: "text.alignleft",
                            title: L.text("mobile.admin.ai.prompts.emptyTitle"),
                            message: L.text("mobile.admin.ai.prompts.emptyMessage")
                        )
                    } else {
                        LMSCard {
                            VStack(alignment: .leading, spacing: 10) {
                                Text(L.text("mobile.admin.ai.prompts.select"))
                                    .font(.subheadline.weight(.semibold))
                                Picker("", selection: $selectedKey) {
                                    ForEach(prompts) { prompt in
                                        Text(prompt.label).tag(prompt.key)
                                    }
                                }
                                .pickerStyle(.menu)
                                .frame(minHeight: 44)
                                .onChange(of: selectedKey) { _, newKey in
                                    if let row = prompts.first(where: { $0.key == newKey }) {
                                        draft = row.content
                                    }
                                }

                                if let selectedPrompt {
                                    Text(L.format(
                                        "mobile.admin.ai.prompts.updatedAt",
                                        selectedPrompt.updatedAt ?? "—"
                                    ))
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                }

                                Button {
                                    showEditor = true
                                } label: {
                                    Label(
                                        L.text("mobile.admin.ai.prompts.edit"),
                                        systemImage: "square.and.pencil"
                                    )
                                    .frame(maxWidth: .infinity)
                                }
                                .buttonStyle(.bordered)
                                .frame(minHeight: 44)

                                Text(draft)
                                    .font(.system(.caption, design: .monospaced))
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    .lineLimit(8)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                            }
                        }

                        saveSection
                    }
                }
                .padding(16)
            }
        }
    }

    private var saveSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Button {
                Task { await save() }
            } label: {
                if saveStatus == .saving {
                    ProgressView().frame(maxWidth: .infinity)
                } else {
                    Text(L.text("mobile.admin.ai.prompts.save"))
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.brandTeal)
            .disabled(selectedKey.isEmpty || saveStatus == .saving)
            .frame(minHeight: 44)

            if case .saved = saveStatus {
                Text(L.text("mobile.admin.ai.prompts.saved"))
                    .font(.caption)
                    .foregroundStyle(.green)
            }
        }
    }

    private var promptEditorSheet: some View {
        NavigationStack {
            TextEditor(text: $draft)
                .font(.system(.body, design: .monospaced))
                .padding(8)
                .navigationTitle(selectedPrompt?.label ?? L.text("mobile.admin.ai.prompts.title"))
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button(L.text("mobile.cancel")) {
                            if let original = selectedPrompt {
                                draft = original.content
                            }
                            showEditor = false
                        }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button(L.text("mobile.common.close")) {
                            showEditor = false
                        }
                    }
                }
        }
        .presentationDetents([.large])
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        if case .error = saveStatus { saveStatus = .idle }
        defer { loading = false }
        do {
            let list = try await LMSAPI.fetchSystemPrompts(accessToken: token)
            prompts = list
            if list.isEmpty {
                selectedKey = ""
                draft = ""
            } else if list.contains(where: { $0.key == selectedKey }),
                      let row = list.first(where: { $0.key == selectedKey }) {
                draft = row.content
            } else {
                selectedKey = list[0].key
                draft = list[0].content
            }
        } catch {
            saveStatus = .error(AiModelsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.ai.prompts.loadError"
            ))
        }
    }

    private func save() async {
        guard let token = session.accessToken else { return }
        let key = selectedKey.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !key.isEmpty else { return }
        saveStatus = .saving
        do {
            let row = try await LMSAPI.putSystemPrompt(key: key, content: draft, accessToken: token)
            prompts = prompts.map { existing in
                existing.key == row.key
                    ? SystemPromptItem(
                        key: row.key,
                        label: row.label.isEmpty ? existing.label : row.label,
                        content: row.content,
                        updatedAt: row.updatedAt
                    )
                    : existing
            }
            draft = row.content
            saveStatus = .saved
        } catch {
            saveStatus = .error(AiModelsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.ai.prompts.saveError"
            ))
        }
    }
}
