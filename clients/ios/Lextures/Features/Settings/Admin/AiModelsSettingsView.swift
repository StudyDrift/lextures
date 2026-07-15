import SwiftUI

/// Platform AI model pickers and OpenRouter API key (M14.7).
struct AiModelsSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var imageModelId = ""
    @State private var courseSetupModelId = ""
    @State private var notebookFlashcardsModelId = ""
    @State private var vibeActivityModelId = ""
    @State private var graderAgentModelId = ""
    @State private var openRouterApiKey = ""
    @State private var openRouterApiKeyBaseline = ""

    @State private var textModels: [AiModelOption] = AiModelsAdminLogic.fallbackTextModels
    @State private var imageModels: [AiModelOption] = AiModelsAdminLogic.fallbackImageModels
    @State private var modelsConfigured = false
    @State private var modelsError: String?
    @State private var modelsRefreshing = false

    @State private var loading = true
    @State private var saveStatus: AiModelsAdminLogic.SaveStatus = .idle

    private var canView: Bool {
        AiModelsAdminLogic.canView(features: shell.platformFeatures, permissions: shell.permissions)
    }

    private var saveDisabled: Bool {
        AiModelsAdminLogic.isSaveDisabled(
            saving: saveStatus == .saving,
            imageModelId: imageModelId,
            courseSetupModelId: courseSetupModelId,
            notebookFlashcardsModelId: notebookFlashcardsModelId,
            vibeActivityModelId: vibeActivityModelId
        )
    }

    var body: some View {
        Group {
            if canView { content } else { accessDenied }
        }
        .navigationTitle(L.text("mobile.admin.ai.models.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task { if canView { await load() } }
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
                    Text(L.text("mobile.admin.ai.models.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if case let .error(message) = saveStatus {
                        LMSErrorBanner(message: message)
                    }
                    if let modelsError {
                        LMSErrorBanner(message: modelsError)
                    }

                    if loading {
                        LMSSkeletonList(count: 5)
                    } else {
                        if !modelsConfigured {
                            Text(L.text("mobile.admin.ai.models.keyRequired"))
                                .font(.caption)
                                .foregroundStyle(.orange)
                                .padding(12)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(Color.orange.opacity(0.12))
                                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                        }

                        apiKeyCard
                        modelPickers
                        saveSection
                    }
                }
                .padding(16)
            }
        }
    }

    private var apiKeyCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.admin.ai.models.apiKey"))
                    .font(.subheadline.weight(.semibold))
                SecureField(AiModelsAdminLogic.platformSecretPlaceholder, text: $openRouterApiKey)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .textFieldStyle(.roundedBorder)
                    .frame(minHeight: 44)
                Text(L.text("mobile.admin.ai.models.apiKeyHint"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var modelPickers: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text(L.text("mobile.admin.ai.models.pickersTitle"))
                    .font(.title3.bold())
                Spacer()
                Button {
                    Task { await refreshModels() }
                } label: {
                    if modelsRefreshing {
                        ProgressView()
                    } else {
                        Label(L.text("mobile.admin.ai.models.refresh"), systemImage: "arrow.clockwise")
                    }
                }
                .buttonStyle(.bordered)
                .disabled(modelsRefreshing || saveStatus == .saving)
                .frame(minHeight: 44)
            }

            modelPicker(
                titleKey: "mobile.admin.ai.models.courseSetup",
                hintKey: "mobile.admin.ai.models.courseSetupHint",
                selection: $courseSetupModelId,
                models: AiModelsAdminLogic.modelsWithSelection(textModels, selectedId: courseSetupModelId)
            )
            modelPicker(
                titleKey: "mobile.admin.ai.models.flashcards",
                hintKey: "mobile.admin.ai.models.flashcardsHint",
                selection: $notebookFlashcardsModelId,
                models: AiModelsAdminLogic.modelsWithSelection(textModels, selectedId: notebookFlashcardsModelId)
            )
            modelPicker(
                titleKey: "mobile.admin.ai.models.vibe",
                hintKey: "mobile.admin.ai.models.vibeHint",
                selection: $vibeActivityModelId,
                models: AiModelsAdminLogic.modelsWithSelection(textModels, selectedId: vibeActivityModelId)
            )
            modelPicker(
                titleKey: "mobile.admin.ai.models.grader",
                hintKey: "mobile.admin.ai.models.graderHint",
                selection: $graderAgentModelId,
                models: AiModelsAdminLogic.modelsWithSelection(textModels, selectedId: graderAgentModelId)
            )
            modelPicker(
                titleKey: "mobile.admin.ai.models.image",
                hintKey: "mobile.admin.ai.models.imageHint",
                selection: $imageModelId,
                models: AiModelsAdminLogic.modelsWithSelection(imageModels, selectedId: imageModelId)
            )
        }
    }

    private func modelPicker(
        titleKey: String.LocalizationValue,
        hintKey: String.LocalizationValue,
        selection: Binding<String>,
        models: [AiModelOption]
    ) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text(titleKey))
                    .font(.subheadline.weight(.semibold))
                Picker("", selection: selection) {
                    ForEach(models) { model in
                        Text(AiModelsAdminLogic.modelDisplayLabel(model))
                            .tag(model.id)
                    }
                }
                .pickerStyle(.menu)
                .frame(minHeight: 44)
                Text(L.text(hintKey))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var saveSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Button {
                Task { await save() }
            } label: {
                if saveStatus == .saving {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                } else {
                    Text(L.text("mobile.admin.ai.models.save"))
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.brandTeal)
            .disabled(saveDisabled)
            .frame(minHeight: 44)

            if case .saved = saveStatus {
                Text(L.text("mobile.admin.ai.models.saved"))
                    .font(.caption)
                    .foregroundStyle(.green)
            }
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        if case .error = saveStatus { saveStatus = .idle }
        defer { loading = false }
        do {
            let settings = try await LMSAPI.fetchAiSettings(accessToken: token)
            applySettings(settings)
            await loadModels(token: token)
        } catch {
            saveStatus = .error(AiModelsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.ai.models.loadError"
            ))
            textModels = AiModelsAdminLogic.fallbackTextModels
            imageModels = AiModelsAdminLogic.fallbackImageModels
            modelsConfigured = false
        }
    }

    private func applySettings(_ settings: AiSettingsResponse) {
        imageModelId = settings.imageModelId
        courseSetupModelId = settings.courseSetupModelId
        notebookFlashcardsModelId = settings.notebookFlashcardsModelId
        vibeActivityModelId = settings.vibeActivityModelId
        graderAgentModelId = settings.graderAgentModelId
        let key = settings.openRouterApiKey ?? ""
        openRouterApiKey = key
        openRouterApiKeyBaseline = key
    }

    private func loadModels(token: String) async {
        modelsError = nil
        do {
            async let text = LMSAPI.fetchAiModels(kind: "text", accessToken: token)
            async let image = LMSAPI.fetchAiModels(kind: "image", accessToken: token)
            let (textRes, imageRes) = try await (text, image)
            modelsConfigured = textRes.configured || imageRes.configured
            textModels = textRes.models.isEmpty ? AiModelsAdminLogic.fallbackTextModels : textRes.models
            imageModels = imageRes.models.isEmpty ? AiModelsAdminLogic.fallbackImageModels : imageRes.models
        } catch {
            modelsError = AiModelsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.ai.models.modelsError"
            )
            textModels = AiModelsAdminLogic.fallbackTextModels
            imageModels = AiModelsAdminLogic.fallbackImageModels
            modelsConfigured = false
        }
    }

    private func refreshModels() async {
        guard let token = session.accessToken else { return }
        modelsRefreshing = true
        defer { modelsRefreshing = false }
        await loadModels(token: token)
    }

    private func save() async {
        guard let token = session.accessToken else { return }
        saveStatus = .saving
        let body = AiModelsAdminLogic.buildAiSettingsSaveRequest(
            imageModelId: imageModelId,
            courseSetupModelId: courseSetupModelId,
            notebookFlashcardsModelId: notebookFlashcardsModelId,
            vibeActivityModelId: vibeActivityModelId,
            graderAgentModelId: graderAgentModelId,
            openRouterApiKey: openRouterApiKey,
            openRouterApiKeyBaseline: openRouterApiKeyBaseline
        )
        do {
            let updated = try await LMSAPI.putAiSettings(body: body, accessToken: token)
            applySettings(updated)
            saveStatus = .saved
            await loadModels(token: token)
        } catch {
            saveStatus = .error(AiModelsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.ai.models.saveError"
            ))
        }
    }
}
