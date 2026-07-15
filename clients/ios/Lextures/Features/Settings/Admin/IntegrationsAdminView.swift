import SwiftUI

struct IntegrationsAdminRoute: Hashable {}
struct IntegrationsLtiRoute: Hashable {}
struct IntegrationsScimRoute: Hashable {}
struct IntegrationsCloudRoute: Hashable {}
struct IntegrationsLrsRoute: Hashable {}
struct IntegrationsOerRoute: Hashable {}

/// Hub for LTI, SCIM, cloud, LRS, and OER admin status (M14.8).
struct IntegrationsAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var scimEnabled = false
    @State private var scimFlagLoaded = false
    @State private var errorMessage: String?

    private var canView: Bool {
        IntegrationsAdminLogic.canView(features: shell.platformFeatures, permissions: shell.permissions)
    }

    private var sections: [IntegrationsAdminLogic.Section] {
        IntegrationsAdminLogic.visibleSections(
            features: shell.platformFeatures,
            scimEnabled: scimEnabled
        )
    }

    var body: some View {
        Group {
            if canView {
                content
            } else {
                accessDenied
            }
        }
        .navigationTitle(L.text("mobile.admin.integrations.hub.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task {
            guard canView else { return }
            await loadScimFlag()
        }
        .navigationDestination(for: IntegrationsLtiRoute.self) { _ in
            LtiIntegrationsAdminView()
        }
        .navigationDestination(for: IntegrationsScimRoute.self) { _ in
            ScimIntegrationsAdminView()
        }
        .navigationDestination(for: IntegrationsCloudRoute.self) { _ in
            CloudProvidersAdminView()
        }
        .navigationDestination(for: IntegrationsLrsRoute.self) { _ in
            LrsIntegrationsAdminView()
        }
        .navigationDestination(for: IntegrationsOerRoute.self) { _ in
            OerProvidersAdminView()
        }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.integrations.accessDenied.title"),
            message: L.text("mobile.admin.integrations.accessDenied.message")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.integrations.hub.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if !scimFlagLoaded {
                        LMSSkeletonList(count: 3)
                    } else if sections.isEmpty {
                        LMSEmptyState(
                            systemImage: "link",
                            title: L.text("mobile.admin.integrations.emptyTitle"),
                            message: L.text("mobile.admin.integrations.emptyMessage")
                        )
                    } else {
                        LMSCard {
                            VStack(spacing: 0) {
                                ForEach(Array(sections.enumerated()), id: \.element.id) { index, section in
                                    if index > 0 {
                                        Divider().padding(.leading, 44)
                                    }
                                    sectionRow(section)
                                }
                            }
                        }
                    }
                }
                .padding(16)
            }
        }
    }

    @ViewBuilder
    private func sectionRow(_ section: IntegrationsAdminLogic.Section) -> some View {
        switch section {
        case .lti:
            SettingsNavigationRow(
                route: IntegrationsLtiRoute(),
                systemImage: section.systemImage,
                title: L.text(section.titleKey),
                subtitle: L.text(section.subtitleKey)
            )
        case .scim:
            SettingsNavigationRow(
                route: IntegrationsScimRoute(),
                systemImage: section.systemImage,
                title: L.text(section.titleKey),
                subtitle: L.text(section.subtitleKey)
            )
        case .cloud:
            SettingsNavigationRow(
                route: IntegrationsCloudRoute(),
                systemImage: section.systemImage,
                title: L.text(section.titleKey),
                subtitle: L.text(section.subtitleKey)
            )
        case .lrs:
            SettingsNavigationRow(
                route: IntegrationsLrsRoute(),
                systemImage: section.systemImage,
                title: L.text(section.titleKey),
                subtitle: L.text(section.subtitleKey)
            )
        case .oer:
            SettingsNavigationRow(
                route: IntegrationsOerRoute(),
                systemImage: section.systemImage,
                title: L.text(section.titleKey),
                subtitle: L.text(section.subtitleKey)
            )
        }
    }

    private func loadScimFlag() async {
        guard let token = session.accessToken else {
            scimFlagLoaded = true
            return
        }
        errorMessage = nil
        do {
            scimEnabled = try await LMSAPI.fetchPlatformScimEnabled(accessToken: token)
        } catch {
            // Hide SCIM section when flag cannot be confirmed; still show other sections.
            scimEnabled = false
            errorMessage = nil
        }
        scimFlagLoaded = true
    }
}

// MARK: - LTI

struct LtiIntegrationsAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var platforms: [LtiParentPlatform] = []
    @State private var tools: [LtiExternalTool] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var statusMessage: String?
    @State private var busyId: String?
    @State private var pendingPlatform: LtiParentPlatform?
    @State private var pendingTool: LtiExternalTool?

    var body: some View {
        detailScaffold(
            state: DetailScaffoldState(
                title: L.text("mobile.admin.integrations.lti.title"),
                description: L.text("mobile.admin.integrations.lti.description"),
                webPath: IntegrationsAdminLogic.Section.lti.webPath,
                loading: loading && platforms.isEmpty && tools.isEmpty,
                empty: DetailScaffoldEmpty(
                    isEmpty: platforms.isEmpty && tools.isEmpty && !loading,
                    title: L.text("mobile.admin.integrations.lti.emptyTitle"),
                    message: L.text("mobile.admin.integrations.lti.emptyMessage")
                ),
                errorMessage: errorMessage,
                statusMessage: statusMessage
            ),
            onRefresh: { await load() }
        ) {
            if !platforms.isEmpty {
                sectionHeader(L.text("mobile.admin.integrations.lti.platforms"))
                ForEach(platforms) { row in
                    statusToggleCard(
                        title: row.name.isEmpty ? row.platformIss : row.name,
                        subtitle: row.clientId,
                        enabled: row.active,
                        busy: busyId == row.id
                    ) {
                        pendingPlatform = row
                    }
                }
            }
            if !tools.isEmpty {
                sectionHeader(L.text("mobile.admin.integrations.lti.tools"))
                ForEach(tools) { row in
                    statusToggleCard(
                        title: row.name.isEmpty ? row.toolIssuer : row.name,
                        subtitle: row.clientId,
                        enabled: row.active,
                        busy: busyId == row.id
                    ) {
                        pendingTool = row
                    }
                }
            }
        }
        .task { await load() }
        .confirmationDialog(
            confirmTitle(active: pendingPlatform.map { !$0.active } ?? pendingTool.map { !$0.active }),
            isPresented: Binding(
                get: { pendingPlatform != nil || pendingTool != nil },
                set: { if !$0 { pendingPlatform = nil; pendingTool = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.admin.integrations.confirm"), role: .destructive) {
                if let row = pendingPlatform {
                    Task { await setPlatformActive(row, active: !row.active) }
                } else if let row = pendingTool {
                    Task { await setToolActive(row, active: !row.active) }
                }
            }
            Button(L.text("mobile.cancel"), role: .cancel) {}
        } message: {
            Text(L.text("mobile.admin.integrations.confirm.message"))
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        do {
            let data = try await LMSAPI.fetchLtiRegistrations(accessToken: token)
            platforms = data.parentPlatforms
            tools = data.externalTools
        } catch {
            errorMessage = IntegrationsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.integrations.error"
            )
        }
        loading = false
    }

    private func setPlatformActive(_ row: LtiParentPlatform, active: Bool) async {
        pendingPlatform = nil
        guard let token = session.accessToken else { return }
        busyId = row.id
        errorMessage = nil
        statusMessage = nil
        do {
            try await LMSAPI.setLtiParentPlatformActive(id: row.id, active: active, accessToken: token)
            platforms = IntegrationsAdminLogic.applyingLtiPlatformActive(platforms, id: row.id, active: active)
            statusMessage = L.text("mobile.admin.integrations.saved")
        } catch {
            errorMessage = IntegrationsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.integrations.toggleError"
            )
            await load()
        }
        busyId = nil
    }

    private func setToolActive(_ row: LtiExternalTool, active: Bool) async {
        pendingTool = nil
        guard let token = session.accessToken else { return }
        busyId = row.id
        errorMessage = nil
        statusMessage = nil
        do {
            try await LMSAPI.setLtiExternalToolActive(id: row.id, active: active, accessToken: token)
            tools = IntegrationsAdminLogic.applyingLtiToolActive(tools, id: row.id, active: active)
            statusMessage = L.text("mobile.admin.integrations.saved")
        } catch {
            errorMessage = IntegrationsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.integrations.toggleError"
            )
            await load()
        }
        busyId = nil
    }
}

// MARK: - SCIM

struct ScimIntegrationsAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var institutions: [AdminOrgRow] = []
    @State private var selectedInstitutionId: String?
    @State private var tokens: [ScimTokenRow] = []
    @State private var events: [ScimEventRow] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var statusMessage: String?

    var body: some View {
        detailScaffold(
            state: DetailScaffoldState(
                title: L.text("mobile.admin.integrations.scim.title"),
                description: L.text("mobile.admin.integrations.scim.description"),
                webPath: IntegrationsAdminLogic.Section.scim.webPath,
                loading: loading && tokens.isEmpty && events.isEmpty,
                empty: DetailScaffoldEmpty(
                    isEmpty: !loading && selectedInstitutionId != nil && tokens.isEmpty && events.isEmpty,
                    title: L.text("mobile.admin.integrations.scim.emptyTitle"),
                    message: L.text("mobile.admin.integrations.scim.emptyMessage")
                ),
                errorMessage: errorMessage,
                statusMessage: statusMessage
            ),
            onRefresh: { await loadStatus() }
        ) {
            if institutions.count > 1 {
                Picker(
                    L.text("mobile.admin.integrations.scim.institution"),
                    selection: Binding(
                        get: { selectedInstitutionId ?? institutions.first?.id ?? "" },
                        set: { newValue in
                            selectedInstitutionId = newValue
                            Task { await loadStatus() }
                        }
                    )
                ) {
                    ForEach(institutions) { org in
                        Text(org.name.isEmpty ? org.id : org.name).tag(org.id)
                    }
                }
                .pickerStyle(.menu)
            } else if let org = institutions.first {
                Text("\(L.text("mobile.admin.integrations.scim.institution")): \(org.name.isEmpty ? org.id : org.name)")
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }

            LMSCard {
                VStack(alignment: .leading, spacing: 8) {
                    Text(L.text("mobile.admin.integrations.scim.summary"))
                        .font(.headline)
                    labeledRow(
                        L.text("mobile.admin.integrations.scim.activeTokens"),
                        "\(IntegrationsAdminLogic.activeTokenCount(tokens))"
                    )
                    labeledRow(
                        L.text("mobile.admin.integrations.scim.lastEvent"),
                        IntegrationsAdminLogic.formatTimestamp(IntegrationsAdminLogic.lastEventAt(events))
                    )
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }

            if !events.isEmpty {
                sectionHeader(L.text("mobile.admin.integrations.scim.recentEvents"))
                ForEach(events.prefix(20)) { event in
                    LMSCard {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("\(event.operation) · \(event.scimResource)")
                                .font(.subheadline.weight(.semibold))
                            if let email = event.userEmail, !email.isEmpty {
                                Text(email)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            Text(IntegrationsAdminLogic.formatTimestamp(event.createdAt))
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
            }

            Text(L.text("mobile.admin.integrations.scim.tokensNote"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .task { await bootstrap() }
    }

    private func bootstrap() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        do {
            institutions = try await LMSAPI.fetchAdminOrganizations(accessToken: token)
            if selectedInstitutionId == nil {
                selectedInstitutionId = institutions.first?.id
            }
        } catch {
            errorMessage = IntegrationsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.integrations.error"
            )
        }
        await loadStatus()
    }

    private func loadStatus() async {
        guard let token = session.accessToken, let institutionId = selectedInstitutionId, !institutionId.isEmpty else {
            loading = false
            return
        }
        loading = true
        errorMessage = nil
        do {
            async let tokensTask = LMSAPI.fetchScimTokens(institutionId: institutionId, accessToken: token)
            async let eventsTask = LMSAPI.fetchScimEvents(institutionId: institutionId, accessToken: token)
            tokens = try await tokensTask
            events = try await eventsTask
        } catch {
            errorMessage = IntegrationsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.integrations.error"
            )
        }
        loading = false
    }
}

// MARK: - Cloud

struct CloudProvidersAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var providers: [CloudProviderStatus] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var statusMessage: String?
    @State private var busyProvider: String?
    @State private var pending: CloudProviderStatus?

    var body: some View {
        detailScaffold(
            state: DetailScaffoldState(
                title: L.text("mobile.admin.integrations.cloud.title"),
                description: L.text("mobile.admin.integrations.cloud.description"),
                webPath: IntegrationsAdminLogic.Section.cloud.webPath,
                loading: loading && providers.isEmpty,
                empty: DetailScaffoldEmpty(
                    isEmpty: providers.isEmpty && !loading,
                    title: L.text("mobile.admin.integrations.cloud.emptyTitle"),
                    message: L.text("mobile.admin.integrations.cloud.emptyMessage")
                ),
                errorMessage: errorMessage,
                statusMessage: statusMessage
            ),
            onRefresh: { await load() }
        ) {
            Text(L.text("mobile.admin.integrations.secretsOmitted"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            ForEach(providers) { row in
                statusToggleCard(
                    title: IntegrationsAdminLogic.cloudProviderLabel(row.provider),
                    subtitle: IntegrationsAdminLogic.formatTimestamp(row.updatedAt),
                    enabled: row.enabled,
                    busy: busyProvider == row.provider
                ) {
                    pending = row
                }
            }
        }
        .task { await load() }
        .confirmationDialog(
            confirmTitle(active: pending.map { !$0.enabled }),
            isPresented: Binding(
                get: { pending != nil },
                set: { if !$0 { pending = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.admin.integrations.confirm"), role: .destructive) {
                if let row = pending {
                    Task { await toggle(row) }
                }
            }
            Button(L.text("mobile.cancel"), role: .cancel) {}
        } message: {
            Text(L.text("mobile.admin.integrations.confirm.message"))
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        do {
            providers = try await LMSAPI.fetchAdminCloudProviders(accessToken: token)
        } catch {
            errorMessage = IntegrationsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.integrations.error"
            )
        }
        loading = false
    }

    private func toggle(_ row: CloudProviderStatus) async {
        pending = nil
        guard let token = session.accessToken else { return }
        let desired = !row.enabled
        busyProvider = row.provider
        errorMessage = nil
        statusMessage = nil
        do {
            try await LMSAPI.setCloudProviderEnabled(
                provider: row.provider,
                enabled: desired,
                accessToken: token
            )
            providers = IntegrationsAdminLogic.applyingCloudEnabled(
                providers,
                provider: row.provider,
                enabled: desired
            )
            statusMessage = L.text("mobile.admin.integrations.saved")
        } catch {
            errorMessage = IntegrationsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.integrations.toggleError"
            )
            await load()
        }
        busyProvider = nil
    }
}

// MARK: - LRS

struct LrsIntegrationsAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var endpoints: [LrsEndpointStatus] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var statusMessage: String?
    @State private var busyId: String?
    @State private var pending: LrsEndpointStatus?

    var body: some View {
        detailScaffold(
            state: DetailScaffoldState(
                title: L.text("mobile.admin.integrations.lrs.title"),
                description: L.text("mobile.admin.integrations.lrs.description"),
                webPath: IntegrationsAdminLogic.Section.lrs.webPath,
                loading: loading && endpoints.isEmpty,
                empty: DetailScaffoldEmpty(
                    isEmpty: endpoints.isEmpty && !loading,
                    title: L.text("mobile.admin.integrations.lrs.emptyTitle"),
                    message: L.text("mobile.admin.integrations.lrs.emptyMessage")
                ),
                errorMessage: errorMessage,
                statusMessage: statusMessage
            ),
            onRefresh: { await load() }
        ) {
            Text(L.text("mobile.admin.integrations.secretsOmitted"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            ForEach(endpoints) { row in
                statusToggleCard(
                    title: row.label.isEmpty ? row.endpointUrl : row.label,
                    subtitle: row.endpointUrl,
                    enabled: row.enabled,
                    busy: busyId == row.id
                ) {
                    pending = row
                }
            }
        }
        .task { await load() }
        .confirmationDialog(
            confirmTitle(active: pending.map { !$0.enabled }),
            isPresented: Binding(
                get: { pending != nil },
                set: { if !$0 { pending = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.admin.integrations.confirm"), role: .destructive) {
                if let row = pending {
                    Task { await toggle(row) }
                }
            }
            Button(L.text("mobile.cancel"), role: .cancel) {}
        } message: {
            Text(L.text("mobile.admin.integrations.confirm.message"))
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        do {
            endpoints = try await LMSAPI.fetchAdminLrsEndpoints(accessToken: token)
        } catch {
            errorMessage = IntegrationsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.integrations.error"
            )
        }
        loading = false
    }

    private func toggle(_ row: LrsEndpointStatus) async {
        pending = nil
        guard let token = session.accessToken else { return }
        let desired = !row.enabled
        busyId = row.id
        errorMessage = nil
        statusMessage = nil
        do {
            try await LMSAPI.setLrsEndpointEnabled(id: row.id, enabled: desired, accessToken: token)
            endpoints = IntegrationsAdminLogic.applyingLrsEnabled(endpoints, id: row.id, enabled: desired)
            statusMessage = L.text("mobile.admin.integrations.saved")
        } catch {
            errorMessage = IntegrationsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.integrations.toggleError"
            )
            await load()
        }
        busyId = nil
    }
}

// MARK: - OER

struct OerProvidersAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var providers: [OerProviderStatus] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var statusMessage: String?
    @State private var busyProvider: String?
    @State private var pending: OerProviderStatus?

    var body: some View {
        detailScaffold(
            state: DetailScaffoldState(
                title: L.text("mobile.admin.integrations.oer.title"),
                description: L.text("mobile.admin.integrations.oer.description"),
                webPath: IntegrationsAdminLogic.Section.oer.webPath,
                loading: loading && providers.isEmpty,
                empty: DetailScaffoldEmpty(
                    isEmpty: providers.isEmpty && !loading,
                    title: L.text("mobile.admin.integrations.oer.emptyTitle"),
                    message: L.text("mobile.admin.integrations.oer.emptyMessage")
                ),
                errorMessage: errorMessage,
                statusMessage: statusMessage
            ),
            onRefresh: { await load() }
        ) {
            ForEach(providers) { row in
                statusToggleCard(
                    title: IntegrationsAdminLogic.oerProviderLabel(row.provider),
                    subtitle: IntegrationsAdminLogic.formatTimestamp(row.updatedAt),
                    enabled: row.enabled,
                    busy: busyProvider == row.provider
                ) {
                    pending = row
                }
            }
        }
        .task { await load() }
        .confirmationDialog(
            confirmTitle(active: pending.map { !$0.enabled }),
            isPresented: Binding(
                get: { pending != nil },
                set: { if !$0 { pending = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.admin.integrations.confirm"), role: .destructive) {
                if let row = pending {
                    Task { await toggle(row) }
                }
            }
            Button(L.text("mobile.cancel"), role: .cancel) {}
        } message: {
            Text(L.text("mobile.admin.integrations.confirm.message"))
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        do {
            providers = try await LMSAPI.fetchAdminOerProviders(accessToken: token)
        } catch {
            errorMessage = IntegrationsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.integrations.error"
            )
        }
        loading = false
    }

    private func toggle(_ row: OerProviderStatus) async {
        pending = nil
        guard let token = session.accessToken else { return }
        let desired = !row.enabled
        busyProvider = row.provider
        errorMessage = nil
        statusMessage = nil
        do {
            try await LMSAPI.setOerProviderEnabled(
                provider: row.provider,
                enabled: desired,
                accessToken: token
            )
            providers = IntegrationsAdminLogic.applyingOerEnabled(
                providers,
                provider: row.provider,
                enabled: desired
            )
            statusMessage = L.text("mobile.admin.integrations.saved")
        } catch {
            errorMessage = IntegrationsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.integrations.toggleError"
            )
            await load()
        }
        busyProvider = nil
    }
}

// MARK: - Shared detail chrome

private func confirmTitle(active: Bool?) -> String {
    guard let active else { return "" }
    return L.text(active ? "mobile.admin.integrations.confirm.enable" : "mobile.admin.integrations.confirm.disable")
}

private func sectionHeader(_ title: String) -> some View {
    Text(title)
        .font(.title3.bold())
        .frame(maxWidth: .infinity, alignment: .leading)
}

private func labeledRow(_ label: String, _ value: String) -> some View {
    HStack {
        Text(label).font(.subheadline.weight(.semibold))
        Spacer()
        Text(value)
            .font(.subheadline)
            .foregroundStyle(.secondary)
            .multilineTextAlignment(.trailing)
    }
}

@ViewBuilder
private func statusToggleCard(
    title: String,
    subtitle: String,
    enabled: Bool,
    busy: Bool,
    onToggle: @escaping () -> Void
) -> some View {
    LMSCard {
        Toggle(isOn: Binding(
            get: { enabled },
            set: { _ in onToggle() }
        )) {
            VStack(alignment: .leading, spacing: 4) {
                Text(title).font(.headline)
                if !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Text(enabled ? L.text("mobile.enabled") : L.text("mobile.disabled"))
                    .font(.caption2.weight(.semibold))
            }
        }
        .disabled(busy)
        .accessibilityValue(enabled ? L.text("mobile.enabled") : L.text("mobile.disabled"))
        .frame(minHeight: 44)
    }
}

private struct DetailScaffoldEmpty {
    let isEmpty: Bool
    let title: String
    let message: String
}

private struct DetailScaffoldState {
    let title: String
    let description: String
    let webPath: String
    let loading: Bool
    let empty: DetailScaffoldEmpty
    let errorMessage: String?
    let statusMessage: String?
}

private struct DetailScaffold<Content: View>: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let state: DetailScaffoldState
    let onRefresh: () async -> Void
    @ViewBuilder let content: () -> Content

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(state.description)
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if let errorMessage = state.errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }
                    if let statusMessage = state.statusMessage {
                        Text(statusMessage)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.brandTeal)
                    }
                    if state.loading {
                        LMSSkeletonList(count: 3)
                    } else if state.empty.isEmpty {
                        LMSEmptyState(
                            systemImage: "tray",
                            title: state.empty.title,
                            message: state.empty.message
                        )
                    } else {
                        content()
                    }
                    Button {
                        openURL(AppConfiguration.webURL(path: state.webPath))
                    } label: {
                        Label(L.text("mobile.admin.integrations.configureOnWeb"), systemImage: "arrow.up.right.square")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .frame(minHeight: 44)
                }
                .padding(16)
            }
        }
        .navigationTitle(state.title)
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await onRefresh() }
    }
}

@ViewBuilder
private func detailScaffold<Content: View>(
    state: DetailScaffoldState,
    onRefresh: @escaping () async -> Void,
    @ViewBuilder content: @escaping () -> Content
) -> some View {
    DetailScaffold(state: state, onRefresh: onRefresh, content: content)
}
