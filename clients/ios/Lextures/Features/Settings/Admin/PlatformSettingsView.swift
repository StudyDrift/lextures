import SwiftUI

struct PlatformSettingsAdminRoute: Hashable {}

struct PlatformSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var settings: PlatformSettingsSnapshot?
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var savedMessage: String?
    @State private var busyKey: String?
    @State private var pendingFeature: PlatformFeatureDefinition?

    private var canView: Bool {
        PlatformSettingsAdminLogic.canView(features: shell.platformFeatures, permissions: shell.permissions)
    }

    var body: some View {
        Group {
            if canView { content } else { accessDenied }
        }
        .navigationTitle(L.text("mobile.admin.platform.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task { if canView { await load() } }
        .confirmationDialog(
            confirmationTitle,
            isPresented: Binding(
                get: { pendingFeature != nil },
                set: { if !$0 { pendingFeature = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.admin.platform.confirm"), role: .destructive) {
                if let feature = pendingFeature { Task { await toggle(feature) } }
            }
            Button(L.text("mobile.cancel"), role: .cancel) {}
        } message: {
            Text(L.text("mobile.admin.platform.confirm.message"))
        }
    }

    private var confirmationTitle: String {
        guard let feature = pendingFeature, let settings else { return "" }
        let next = !PlatformSettingsAdminLogic.value(for: feature.key, in: settings)
        return L.format(
            next ? "mobile.admin.platform.confirm.enable" : "mobile.admin.platform.confirm.disable",
            L.dynamicText(feature.labelKey)
        )
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.platform.accessDenied.title"),
            message: L.text("mobile.admin.platform.accessDenied.message")
        ).padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.platform.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if let errorMessage { LMSErrorBanner(message: errorMessage) }
                    if let savedMessage {
                        Text(savedMessage).font(.caption).foregroundStyle(LexturesTheme.brandTeal)
                    }
                    if loading && settings == nil {
                        LMSSkeletonList(count: 4)
                    } else if let settings {
                        configCard(settings)
                        featureList(settings)
                        Button {
                            openURL(AppConfiguration.webURL(path: PlatformSettingsAdminLogic.webSettingsPath()))
                        } label: {
                            Label(L.text("mobile.admin.platform.editOnWeb"), systemImage: "arrow.up.right.square")
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.bordered)
                        .frame(minHeight: 44)
                    }
                }.padding(16)
            }
        }
    }

    private func configCard(_ value: PlatformSettingsSnapshot) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(L.text("mobile.admin.platform.config.title")).font(.headline)
                configRow("mobile.admin.platform.config.saml", value.samlSsoEnabled ? L.text("mobile.enabled") : L.text("mobile.disabled"))
                configRow("mobile.admin.platform.config.mfa", value.mfaEnabled ? value.mfaEnforcement : L.text("mobile.disabled"))
                configRow("mobile.admin.platform.config.baseUrl", value.samlPublicBaseUrl)
                configRow("mobile.admin.platform.config.entityId", value.samlSpEntityId)
                configRow("mobile.admin.platform.config.smtp", value.smtpHost.isEmpty ? L.text("mobile.admin.platform.notConfigured") : "\(value.smtpHost):\(value.smtpPort)")
                configRow("mobile.admin.platform.config.from", value.smtpFrom)
                Text(L.text("mobile.admin.platform.config.secretsOmitted"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }.frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func configRow(_ key: String.LocalizationValue, _ value: String) -> some View {
        HStack(alignment: .firstTextBaseline) {
            Text(L.text(key)).font(.subheadline.weight(.semibold))
            Spacer()
            Text(value.isEmpty ? L.text("mobile.admin.platform.notConfigured") : value)
                .font(.subheadline).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .multilineTextAlignment(.trailing)
        }
    }

    private func featureList(_ value: PlatformSettingsSnapshot) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(L.text("mobile.admin.platform.features.title")).font(.title3.bold())
            Text(L.text("mobile.admin.platform.features.allowlist"))
                .font(.caption).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            ForEach(PlatformSettingsAdminLogic.featureDefinitions) { feature in
                LMSCard {
                    Toggle(isOn: Binding(
                        get: { PlatformSettingsAdminLogic.value(for: feature.key, in: value) },
                        set: { _ in pendingFeature = feature }
                    )) {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(L.dynamicText(feature.labelKey)).font(.headline)
                            Text(L.dynamicText(feature.descriptionKey))
                                .font(.caption).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            Text(PlatformSettingsAdminLogic.value(for: feature.key, in: value)
                                ? L.text("mobile.enabled") : L.text("mobile.disabled"))
                                .font(.caption2.weight(.semibold))
                        }
                    }
                    .disabled(busyKey != nil)
                    .accessibilityValue(PlatformSettingsAdminLogic.value(for: feature.key, in: value)
                        ? L.text("mobile.enabled") : L.text("mobile.disabled"))
                    .frame(minHeight: 44)
                }
            }
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        do { settings = try await LMSAPI.fetchPlatformSettings(accessToken: token) }
        catch { errorMessage = L.text("mobile.admin.platform.error") }
        loading = false
    }

    private func toggle(_ feature: PlatformFeatureDefinition) async {
        pendingFeature = nil
        guard let token = session.accessToken, let settings else { return }
        let desired = !PlatformSettingsAdminLogic.value(for: feature.key, in: settings)
        busyKey = feature.key
        errorMessage = nil
        savedMessage = nil
        do {
            let updated = try await LMSAPI.setPlatformFeature(key: feature.key, enabled: desired, accessToken: token)
            guard PlatformSettingsAdminLogic.value(for: feature.key, in: updated) == desired else {
                throw APIError.invalidResponse
            }
            self.settings = updated
            savedMessage = L.text("mobile.admin.platform.saved")
        } catch {
            errorMessage = L.text("mobile.admin.platform.toggleError")
            await load()
        }
        busyKey = nil
    }
}

