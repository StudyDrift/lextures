import SwiftUI

/// Advising appointment link and degree-audit provider configuration (M14.9).
struct AdvisingSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var appointmentUrl = ""
    @State private var provider: TranscriptsAdvisingAdminLogic.DegreeAuditProvider = .none
    @State private var baseUrl = ""
    @State private var credentialsRef = ""
    @State private var atRiskBanner = false
    @State private var loading = true
    @State private var saveStatus: TranscriptsAdvisingAdminLogic.SaveStatus = .idle

    private var canView: Bool {
        TranscriptsAdvisingAdminLogic.canViewAdvising(
            features: shell.platformFeatures,
            permissions: shell.permissions
        )
    }

    private var saveDisabled: Bool {
        TranscriptsAdvisingAdminLogic.isAdvisingSaveDisabled(
            saving: saveStatus == .saving,
            appointmentUrl: appointmentUrl
        )
    }

    var body: some View {
        Group {
            if canView { content } else { accessDenied }
        }
        .navigationTitle(L.text("mobile.admin.advising.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task { if canView { await load() } }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.transcriptsAdvising.accessDenied.title"),
            message: L.text("mobile.admin.advising.flagOff")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.advising.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if case let .error(message) = saveStatus {
                        LMSErrorBanner(message: message)
                    }
                    if case .saved = saveStatus {
                        Text(L.text("mobile.admin.advising.saved"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.brandTeal)
                    }

                    if loading {
                        LMSSkeletonList(count: 4)
                    } else {
                        formCard
                        saveButton
                        Button {
                            openURL(AppConfiguration.webURL(path: TranscriptsAdvisingAdminLogic.Section.advising.webPath))
                        } label: {
                            Label(L.text("mobile.admin.advising.configureOnWeb"), systemImage: "arrow.up.right.square")
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
                fieldLabel("mobile.admin.advising.appointmentUrl")
                TextField(
                    L.text("mobile.admin.advising.appointmentUrl.placeholder"),
                    text: $appointmentUrl
                )
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .keyboardType(.URL)
                .textFieldStyle(.roundedBorder)
                .frame(minHeight: 44)
                .accessibilityLabel(L.text("mobile.admin.advising.appointmentUrl"))
                hint("mobile.admin.advising.appointmentUrl.hint")

                fieldLabel("mobile.admin.advising.provider")
                Picker("", selection: $provider) {
                    ForEach(TranscriptsAdvisingAdminLogic.DegreeAuditProvider.allCases) { option in
                        Text(L.text(option.labelKey)).tag(option)
                    }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: .infinity, alignment: .leading)
                .frame(minHeight: 44)
                .accessibilityLabel(L.text("mobile.admin.advising.provider"))

                if provider != .none {
                    fieldLabel("mobile.admin.advising.baseUrl")
                    TextField(
                        L.text("mobile.admin.advising.baseUrl.placeholder"),
                        text: $baseUrl
                    )
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .keyboardType(.URL)
                    .textFieldStyle(.roundedBorder)
                    .frame(minHeight: 44)
                    .accessibilityLabel(L.text("mobile.admin.advising.baseUrl"))

                    fieldLabel("mobile.admin.advising.credentialsRef")
                    TextField(
                        L.text("mobile.admin.advising.credentialsRef.placeholder"),
                        text: $credentialsRef
                    )
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .textFieldStyle(.roundedBorder)
                    .frame(minHeight: 44)
                    .accessibilityLabel(L.text("mobile.admin.advising.credentialsRef"))

                    Toggle(isOn: $atRiskBanner) {
                        Text(L.text("mobile.admin.advising.atRiskBanner"))
                            .font(.subheadline)
                    }
                    .frame(minHeight: 44)
                }
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
                Text(L.text("mobile.admin.advising.save"))
                    .frame(maxWidth: .infinity)
            }
        }
        .buttonStyle(.borderedProminent)
        .disabled(saveDisabled)
        .frame(minHeight: 44)
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

    private func apply(_ config: AdminAdvisingConfig) {
        appointmentUrl = config.appointmentUrl
        provider = TranscriptsAdvisingAdminLogic.DegreeAuditProvider.normalized(config.degreeAuditProvider)
        baseUrl = config.degreeAuditBaseUrl
        credentialsRef = config.apiCredentialsRef
        atRiskBanner = config.atRiskBannerEnabled
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        saveStatus = .idle
        do {
            let config = try await LMSAPI.fetchAdminAdvisingConfig(accessToken: token)
            apply(config)
        } catch {
            saveStatus = .error(
                TranscriptsAdvisingAdminLogic.userFacingError(error, fallbackKey: "mobile.admin.advising.loadError")
            )
        }
        loading = false
    }

    private func save() async {
        guard let token = session.accessToken else { return }
        saveStatus = .saving
        let body = TranscriptsAdvisingAdminLogic.buildAdvisingSaveRequest(
            appointmentUrl: appointmentUrl,
            provider: provider,
            baseUrl: baseUrl,
            credentialsRef: credentialsRef,
            atRiskBannerEnabled: atRiskBanner
        )
        do {
            let updated = try await LMSAPI.postAdminAdvisingConfig(body: body, accessToken: token)
            apply(updated)
            saveStatus = .saved
        } catch {
            saveStatus = .error(
                TranscriptsAdvisingAdminLogic.userFacingError(error, fallbackKey: "mobile.admin.advising.saveError")
            )
        }
    }
}
