import SwiftUI

/// Official transcripts webhook and pickup configuration (M14.9).
struct TranscriptsSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var webhookUrl = ""
    @State private var webhookSecret = ""
    @State private var pickupInstructions = ""
    @State private var hasWebhookSecret = false
    @State private var failures: [AdminTranscriptRequestRow] = []
    @State private var loading = true
    @State private var saveStatus: TranscriptsAdvisingAdminLogic.SaveStatus = .idle

    private var canView: Bool {
        TranscriptsAdvisingAdminLogic.canViewTranscripts(
            features: shell.platformFeatures,
            permissions: shell.permissions
        )
    }

    private var saveDisabled: Bool {
        TranscriptsAdvisingAdminLogic.isTranscriptsSaveDisabled(
            saving: saveStatus == .saving,
            webhookUrl: webhookUrl
        )
    }

    var body: some View {
        Group {
            if canView { content } else { accessDenied }
        }
        .navigationTitle(L.text("mobile.admin.transcripts.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task { if canView { await load() } }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.transcriptsAdvising.accessDenied.title"),
            message: L.text("mobile.admin.transcripts.flagOff")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.transcripts.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if case let .error(message) = saveStatus {
                        LMSErrorBanner(message: message)
                    }
                    if case .saved = saveStatus {
                        Text(L.text("mobile.admin.transcripts.saved"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.brandTeal)
                    }

                    if loading {
                        LMSSkeletonList(count: 4)
                    } else {
                        formCard
                        saveButton
                        failuresSection
                        Button {
                            openURL(AppConfiguration.webURL(path: TranscriptsAdvisingAdminLogic.Section.transcripts.webPath))
                        } label: {
                            Label(L.text("mobile.admin.transcripts.configureOnWeb"), systemImage: "arrow.up.right.square")
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.bordered)
                        .frame(minHeight: 44)
                    }
                }
                .padding(16)
            }
        }
    }

    private var formCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 14) {
                fieldLabel("mobile.admin.transcripts.webhookUrl")
                TextField(
                    L.text("mobile.admin.transcripts.webhookUrl.placeholder"),
                    text: $webhookUrl
                )
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .keyboardType(.URL)
                .textFieldStyle(.roundedBorder)
                .frame(minHeight: 44)
                .accessibilityLabel(L.text("mobile.admin.transcripts.webhookUrl"))
                hint("mobile.admin.transcripts.webhookUrl.hint")

                fieldLabel("mobile.admin.transcripts.pickup")
                TextField(
                    L.text("mobile.admin.transcripts.pickup.placeholder"),
                    text: $pickupInstructions,
                    axis: .vertical
                )
                .lineLimit(3 ... 6)
                .textFieldStyle(.roundedBorder)
                .frame(minHeight: 88)
                .accessibilityLabel(L.text("mobile.admin.transcripts.pickup"))
                hint("mobile.admin.transcripts.pickup.hint")

                fieldLabel("mobile.admin.transcripts.secret")
                SecureField(
                    hasWebhookSecret
                        ? TranscriptsAdvisingAdminLogic.secretPlaceholder
                        : L.text("mobile.admin.transcripts.secret.placeholder"),
                    text: $webhookSecret
                )
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .textFieldStyle(.roundedBorder)
                .frame(minHeight: 44)
                .accessibilityLabel(L.text("mobile.admin.transcripts.secret"))
                hint("mobile.admin.transcripts.secret.hint")
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var saveButton: some View {
        Button {
            Task { await save() }
        } label: {
            if saveStatus == .saving {
                ProgressView()
                    .frame(maxWidth: .infinity)
            } else {
                Text(L.text("mobile.admin.transcripts.save"))
                    .frame(maxWidth: .infinity)
            }
        }
        .buttonStyle(.borderedProminent)
        .disabled(saveDisabled)
        .frame(minHeight: 44)
    }

    @ViewBuilder
    private var failuresSection: some View {
        if !failures.isEmpty {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.admin.transcripts.failures.title"))
                    .font(.title3.bold())
                Text(L.text("mobile.admin.transcripts.failures.description"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                ForEach(failures) { row in
                    LMSCard {
                        VStack(alignment: .leading, spacing: 6) {
                            Text(TranscriptsAdvisingAdminLogic.formatTimestamp(row.requestedAt))
                                .font(.subheadline.weight(.semibold))
                            Text(row.errorMessage?.isEmpty == false
                                ? (row.errorMessage ?? "")
                                : L.text("mobile.emDash"))
                                .font(.caption)
                                .foregroundStyle(.red)
                            Text("\(L.text("mobile.admin.transcripts.failures.http")): \(TranscriptsAdvisingAdminLogic.httpStatusLabel(row.webhookResponseCode))")
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
            }
        }
    }

    private func fieldLabel(_ key: String.LocalizationValue) -> some View {
        Text(L.text(key))
            .font(.subheadline.weight(.semibold))
    }

    private func hint(_ key: String.LocalizationValue) -> some View {
        Text(L.text(key))
            .font(.caption)
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
    }

    private func apply(_ config: AdminTranscriptsConfig) {
        webhookUrl = config.webhookUrl
        hasWebhookSecret = config.hasWebhookSecret
        webhookSecret = TranscriptsAdvisingAdminLogic.webhookSecretField(from: config)
        pickupInstructions = config.pickupInstructions ?? ""
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        saveStatus = .idle
        do {
            async let configTask = LMSAPI.fetchAdminTranscriptsConfig(accessToken: token)
            async let failuresTask = LMSAPI.fetchAdminTranscriptRequests(accessToken: token)
            let (config, failed) = try await (configTask, failuresTask)
            apply(config)
            failures = failed
        } catch {
            saveStatus = .error(
                TranscriptsAdvisingAdminLogic.userFacingError(error, fallbackKey: "mobile.admin.transcripts.loadError")
            )
        }
        loading = false
    }

    private func save() async {
        guard let token = session.accessToken else { return }
        saveStatus = .saving
        let body = TranscriptsAdvisingAdminLogic.buildTranscriptsSaveRequest(
            webhookUrl: webhookUrl,
            webhookSecret: webhookSecret,
            pickupInstructions: pickupInstructions
        )
        do {
            let updated = try await LMSAPI.putAdminTranscriptsConfig(body: body, accessToken: token)
            apply(updated)
            saveStatus = .saved
        } catch {
            saveStatus = .error(
                TranscriptsAdvisingAdminLogic.userFacingError(error, fallbackKey: "mobile.admin.transcripts.saveError")
            )
        }
    }
}
