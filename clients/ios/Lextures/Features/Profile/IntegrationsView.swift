import SwiftUI

struct IntegrationsRoute: Hashable {}

/// Account integrations: access keys, calendar feeds, MCP, and service tokens (M14.1).
struct IntegrationsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var accessKeys: [AccessKeySummary] = []
    @State private var scopes: [AccessKeyScopeDef] = []
    @State private var serviceTokens: [AccessKeySummary] = []
    @State private var serviceTokensForbidden = true
    @State private var mcpConfig: MCPConfigResponse?
    @State private var tokenInfo: CalendarTokenInfo?
    @State private var createdCalendarToken: CalendarTokenCreated?

    @State private var loading = true
    @State private var errorMessage: String?
    @State private var statusMessage: String?

    @State private var revealSecret: OneTimeSecretReveal?
    @State private var showCreateKey = false
    @State private var showCreateServiceToken = false
    @State private var confirmingRevokeKeyId: String?
    @State private var confirmingRotateKeyId: String?
    @State private var confirmingRevokeServiceTokenId: String?
    @State private var confirmingRegenerateCalendar = false

    @State private var createLabel = ""
    @State private var selectedScopes: Set<String> = Set(AccountIntegrationsLogic.defaultCreateScopes)
    @State private var creating = false

    @State private var serviceAccountName = ""
    @State private var serviceTokenLabel = ""
    @State private var serviceTokenScopes: Set<String> = ["enrollments:read"]
    @State private var mcpTokenDraft = ""

    private var features: MobilePlatformFeatures { shell.platformFeatures }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.integrations.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }
                    if let statusMessage {
                        Text(statusMessage)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.brandTeal)
                    }

                    if AccountIntegrationsLogic.accessKeysEnabled(features) {
                        accessKeysCard
                    }
                    if AccountIntegrationsLogic.calendarSubscriptionsEnabled(features) {
                        calendarCard
                    }
                    if mcpConfig != nil || AccountIntegrationsLogic.accessKeysEnabled(features) {
                        mcpCard
                    }
                    if AccountIntegrationsLogic.shouldShowServiceTokensSection(
                        permissions: shell.permissions,
                        adminApiForbidden: serviceTokensForbidden
                    ) {
                        serviceTokensCard
                    }
                }
                .padding(16)
            }
        }
        .navigationTitle(L.text("mobile.integrations.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await loadAll() }
        .task { await loadAll() }
        .sheet(isPresented: $showCreateKey) { createKeySheet }
        .sheet(isPresented: $showCreateServiceToken) { createServiceTokenSheet }
        .sheet(item: $revealSecret) { secret in
            SecretRevealSheet(secret: secret) { revealSecret = nil }
        }
        .confirmationDialog(
            L.text("mobile.integrations.accessKeys.revokeConfirm"),
            isPresented: Binding(
                get: { confirmingRevokeKeyId != nil },
                set: { if !$0 { confirmingRevokeKeyId = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.integrations.accessKeys.revoke"), role: .destructive) {
                if let id = confirmingRevokeKeyId {
                    Task { await revokeKey(id) }
                }
            }
        }
        .confirmationDialog(
            L.text("mobile.integrations.accessKeys.rotateConfirm"),
            isPresented: Binding(
                get: { confirmingRotateKeyId != nil },
                set: { if !$0 { confirmingRotateKeyId = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.integrations.accessKeys.rotate"), role: .destructive) {
                if let id = confirmingRotateKeyId {
                    Task { await rotateKey(id) }
                }
            }
        }
        .confirmationDialog(
            L.text("mobile.integrations.serviceTokens.revokeConfirm"),
            isPresented: Binding(
                get: { confirmingRevokeServiceTokenId != nil },
                set: { if !$0 { confirmingRevokeServiceTokenId = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.integrations.serviceTokens.revoke"), role: .destructive) {
                if let id = confirmingRevokeServiceTokenId {
                    Task { await revokeServiceToken(id) }
                }
            }
        }
        .confirmationDialog(
            L.text("mobile.integrations.calendar.regenerateConfirm"),
            isPresented: $confirmingRegenerateCalendar,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.integrations.calendar.regenerate"), role: .destructive) {
                Task { await regenerateCalendarToken() }
            }
        } message: {
            Text(L.text("mobile.integrations.calendar.regenerateMessage"))
        }
    }

    // MARK: - Access keys

    private var accessKeysCard: some View {
        LMSCard {
            sectionHeader(
                systemImage: "key",
                title: L.text("mobile.integrations.accessKeys.title"),
                subtitle: L.text("mobile.integrations.accessKeys.description")
            )
            if loading && accessKeys.isEmpty {
                ProgressView().padding(.vertical, 8)
            } else {
                let active = AccountIntegrationsLogic.activeAccessKeys(accessKeys)
                if active.isEmpty {
                    Text(L.text("mobile.integrations.accessKeys.empty"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(active) { key in
                        accessKeyRow(key)
                        if key.id != active.last?.id { Divider() }
                    }
                }
                Button {
                    createLabel = ""
                    selectedScopes = Set(AccountIntegrationsLogic.defaultCreateScopes)
                    showCreateKey = true
                } label: {
                    Label(L.text("mobile.integrations.accessKeys.create"), systemImage: "plus")
                        .font(.subheadline.weight(.semibold))
                }
                .buttonStyle(.borderedProminent)
                .padding(.top, 8)
            }
        }
    }

    private func accessKeyRow(_ key: AccessKeySummary) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(key.label)
                .font(.subheadline.weight(.semibold))
            Text(key.tokenMask)
                .font(.caption.monospaced())
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Text(key.scopes.joined(separator: ", "))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            HStack {
                Button(L.text("mobile.integrations.accessKeys.rotate")) {
                    confirmingRotateKeyId = key.id
                }
                .buttonStyle(.bordered)
                Button(L.text("mobile.integrations.accessKeys.revoke")) {
                    confirmingRevokeKeyId = key.id
                }
                .buttonStyle(.bordered)
                .tint(LexturesTheme.error)
            }
        }
        .padding(.vertical, 4)
    }

    // MARK: - Calendar

    private var calendarCard: some View {
        LMSCard {
            sectionHeader(
                systemImage: "calendar",
                title: L.text("mobile.integrations.calendar.title"),
                subtitle: L.text("mobile.integrations.calendar.description")
            )
            if let url = AccountIntegrationsLogic.resolvedPersonalFeedURL(info: tokenInfo, created: createdCalendarToken) {
                feedURLBlock(title: L.text("mobile.integrations.calendar.personalFeed"), url: url)
            } else if tokenInfo?.hasToken == true {
                Text(L.text("mobile.integrations.calendar.activeHint"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                Text(L.text("mobile.integrations.calendar.emptyHint"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            if let token = createdCalendarToken?.token, let feeds = tokenInfo?.courseFeeds {
                ForEach(feeds, id: \.courseCode) { feed in
                    if let url = AccountIntegrationsLogic.resolvedCourseFeedURL(template: feed.feedUrl, token: token) {
                        feedURLBlock(title: feed.title, url: url)
                    }
                }
            }
            Text(L.text("mobile.integrations.calendar.privacyWarning"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.amber)
            Button {
                if createdCalendarToken != nil || tokenInfo?.hasToken == true {
                    confirmingRegenerateCalendar = true
                } else {
                    Task { await regenerateCalendarToken() }
                }
            } label: {
                Label(
                    (createdCalendarToken != nil || tokenInfo?.hasToken == true)
                        ? L.text("mobile.integrations.calendar.regenerate")
                        : L.text("mobile.integrations.calendar.generate"),
                    systemImage: "arrow.clockwise"
                )
            }
            .buttonStyle(.borderedProminent)
        }
    }

    private func feedURLBlock(title: String, url: String) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title)
                .font(.subheadline.weight(.semibold))
            Text(url)
                .font(.caption2.monospaced())
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .textSelection(.enabled)
            Button(L.text("mobile.integrations.calendar.copy")) {
                copyToClipboard(url, statusKey: "mobile.integrations.calendar.copied")
            }
            .buttonStyle(.bordered)
        }
        .padding(.vertical, 4)
    }

    // MARK: - MCP

    private var mcpCard: some View {
        LMSCard {
            sectionHeader(
                systemImage: "cpu",
                title: L.text("mobile.integrations.mcp.title"),
                subtitle: L.text("mobile.integrations.mcp.description")
            )
            if let config = mcpConfig {
                ForEach(config.instructions, id: \.self) { step in
                    Text("• \(step)")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                SecureField(L.text("mobile.integrations.mcp.tokenPlaceholder"), text: $mcpTokenDraft)
                    .textContentType(.password)
                if let json = AccountIntegrationsLogic.mcpConfigJSON(base: config.cursorConfig, token: mcpTokenDraft) {
                    Text(json)
                        .font(.caption2.monospaced())
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .textSelection(.enabled)
                    Button(L.text("mobile.integrations.mcp.copyConfig")) {
                        copyToClipboard(json, statusKey: "mobile.integrations.mcp.copied")
                    }
                    .buttonStyle(.bordered)
                }
                Text(L.text("mobile.integrations.mcp.apiBaseUrl"))
                    .font(.caption.weight(.semibold))
                Text(config.apiBaseUrl)
                    .font(.caption.monospaced())
            } else if loading {
                ProgressView().padding(.vertical, 8)
            }
        }
    }

    // MARK: - Service tokens

    private var serviceTokensCard: some View {
        LMSCard {
            sectionHeader(
                systemImage: "building.2",
                title: L.text("mobile.integrations.serviceTokens.title"),
                subtitle: L.text("mobile.integrations.serviceTokens.description")
            )
            if serviceTokens.isEmpty {
                Text(L.text("mobile.integrations.serviceTokens.empty"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(serviceTokens) { token in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(token.label)
                            .font(.subheadline.weight(.semibold))
                        if let account = token.serviceAccountName {
                            Text(account)
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        Text(token.tokenMask)
                            .font(.caption.monospaced())
                        Button(L.text("mobile.integrations.serviceTokens.revoke")) {
                            confirmingRevokeServiceTokenId = token.id
                        }
                        .buttonStyle(.bordered)
                        .tint(LexturesTheme.error)
                    }
                    if token.id != serviceTokens.last?.id { Divider() }
                }
            }
            Button {
                serviceAccountName = ""
                serviceTokenLabel = ""
                serviceTokenScopes = ["enrollments:read"]
                showCreateServiceToken = true
            } label: {
                Label(L.text("mobile.integrations.serviceTokens.create"), systemImage: "plus")
            }
            .buttonStyle(.borderedProminent)
            .padding(.top, 8)
        }
    }

    // MARK: - Sheets

    private var createKeySheet: some View {
        NavigationStack {
            Form {
                TextField(L.text("mobile.integrations.accessKeys.label"), text: $createLabel)
                Section(L.text("mobile.integrations.accessKeys.scopes")) {
                    ForEach(scopes) { scope in
                        Toggle(scope.label, isOn: Binding(
                            get: { selectedScopes.contains(scope.id) },
                            set: { on in
                                if on { selectedScopes.insert(scope.id) } else { selectedScopes.remove(scope.id) }
                            }
                        ))
                    }
                }
            }
            .navigationTitle(L.text("mobile.integrations.accessKeys.createTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.integrations.cancel")) { showCreateKey = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(creating ? L.text("mobile.integrations.creating") : L.text("mobile.integrations.create")) {
                        Task { await createKey() }
                    }
                    .disabled(creating || createLabel.trimmingCharacters(in: .whitespaces).isEmpty || selectedScopes.isEmpty)
                }
            }
        }
        .presentationDetents([.medium, .large])
    }

    private var createServiceTokenSheet: some View {
        NavigationStack {
            Form {
                TextField(L.text("mobile.integrations.serviceTokens.accountName"), text: $serviceAccountName)
                TextField(L.text("mobile.integrations.accessKeys.label"), text: $serviceTokenLabel)
                Section(L.text("mobile.integrations.accessKeys.scopes")) {
                    ForEach(scopes) { scope in
                        Toggle(scope.label, isOn: Binding(
                            get: { serviceTokenScopes.contains(scope.id) },
                            set: { on in
                                if on { serviceTokenScopes.insert(scope.id) } else { serviceTokenScopes.remove(scope.id) }
                            }
                        ))
                    }
                }
            }
            .navigationTitle(L.text("mobile.integrations.serviceTokens.createTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.integrations.cancel")) { showCreateServiceToken = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(creating ? L.text("mobile.integrations.creating") : L.text("mobile.integrations.create")) {
                        Task { await createServiceToken() }
                    }
                    .disabled(
                        creating
                            || serviceAccountName.trimmingCharacters(in: .whitespaces).isEmpty
                            || serviceTokenScopes.isEmpty
                    )
                }
            }
        }
        .presentationDetents([.medium, .large])
    }

    // MARK: - Helpers

    private func sectionHeader(systemImage: String, title: String, subtitle: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Label(title, systemImage: systemImage)
                .font(LexturesTheme.displayFont(17))
            Text(subtitle)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .padding(.bottom, 4)
    }

    @MainActor
    private func loadAll() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            if AccountIntegrationsLogic.accessKeysEnabled(features) {
                async let keys = LMSAPI.fetchAccessKeys(accessToken: token)
                async let scopeList = LMSAPI.fetchAccessKeyScopes(accessToken: token)
                async let mcp = LMSAPI.fetchMCPConfig(accessToken: token)
                accessKeys = try await keys
                scopes = try await scopeList
                mcpConfig = try await mcp
            }
            if AccountIntegrationsLogic.calendarSubscriptionsEnabled(features) {
                tokenInfo = try await LMSAPI.fetchCalendarTokenInfo(accessToken: token)
            }
            if AccountIntegrationsLogic.canManageServiceTokens(permissions: shell.permissions) {
                if let tokens = try await LMSAPI.fetchServiceTokens(accessToken: token) {
                    serviceTokens = tokens
                    serviceTokensForbidden = false
                } else {
                    serviceTokens = []
                    serviceTokensForbidden = true
                }
            }
        } catch {
            errorMessage = L.text("mobile.integrations.error")
        }
    }

    @MainActor
    private func createKey() async {
        guard let token = session.accessToken else { return }
        creating = true
        defer { creating = false }
        do {
            let label = createLabel.trimmingCharacters(in: .whitespacesAndNewlines)
            let created = try await LMSAPI.createAccessKey(
                label: label,
                scopes: Array(selectedScopes),
                accessToken: token
            )
            showCreateKey = false
            if let secret = created.token {
                revealSecret = OneTimeSecretReveal(token: secret, label: created.label ?? label)
            }
            statusMessage = L.text("mobile.integrations.accessKeys.created")
            accessKeys = try await LMSAPI.fetchAccessKeys(accessToken: token)
        } catch {
            errorMessage = L.text("mobile.integrations.accessKeys.error")
        }
    }

    @MainActor
    private func revokeKey(_ id: String) async {
        guard let token = session.accessToken else { return }
        do {
            try await LMSAPI.revokeAccessKey(id: id, accessToken: token)
            statusMessage = L.text("mobile.integrations.accessKeys.revoked")
            accessKeys = try await LMSAPI.fetchAccessKeys(accessToken: token)
        } catch {
            errorMessage = L.text("mobile.integrations.accessKeys.error")
        }
    }

    @MainActor
    private func rotateKey(_ id: String) async {
        guard let token = session.accessToken else { return }
        do {
            let rotated = try await LMSAPI.rotateAccessKey(id: id, accessToken: token)
            if let secret = rotated.token {
                revealSecret = OneTimeSecretReveal(token: secret, label: rotated.label ?? "Rotated key")
            }
            statusMessage = L.text("mobile.integrations.accessKeys.created")
            accessKeys = try await LMSAPI.fetchAccessKeys(accessToken: token)
        } catch {
            errorMessage = L.text("mobile.integrations.accessKeys.error")
        }
    }

    @MainActor
    private func regenerateCalendarToken() async {
        guard let token = session.accessToken else { return }
        do {
            createdCalendarToken = try await LMSAPI.createCalendarToken(accessToken: token)
            tokenInfo = try await LMSAPI.fetchCalendarTokenInfo(accessToken: token)
            statusMessage = L.text("mobile.integrations.calendar.copied")
        } catch {
            errorMessage = L.text("mobile.integrations.calendar.error")
        }
    }

    @MainActor
    private func createServiceToken() async {
        guard let token = session.accessToken else { return }
        creating = true
        defer { creating = false }
        do {
            let account = serviceAccountName.trimmingCharacters(in: .whitespacesAndNewlines)
            let label = serviceTokenLabel.trimmingCharacters(in: .whitespacesAndNewlines).ifEmpty(account)
            let created = try await LMSAPI.createServiceToken(
                serviceAccountName: account,
                label: label,
                scopes: Array(serviceTokenScopes),
                accessToken: token
            )
            showCreateServiceToken = false
            if let secret = created.token {
                revealSecret = OneTimeSecretReveal(token: secret, label: created.label ?? label)
            }
            statusMessage = L.text("mobile.integrations.serviceTokens.created")
            serviceTokens = try await LMSAPI.fetchServiceTokens(accessToken: token) ?? []
        } catch {
            errorMessage = L.text("mobile.integrations.serviceTokens.error")
        }
    }

    @MainActor
    private func revokeServiceToken(_ id: String) async {
        guard let token = session.accessToken else { return }
        do {
            try await LMSAPI.revokeServiceToken(id: id, accessToken: token)
            statusMessage = L.text("mobile.integrations.serviceTokens.revoked")
            serviceTokens = try await LMSAPI.fetchServiceTokens(accessToken: token) ?? []
        } catch {
            errorMessage = L.text("mobile.integrations.serviceTokens.error")
        }
    }

    private func copyToClipboard(_ text: String, statusKey: String) {
        UIPasteboard.general.string = text
        statusMessage = L.dynamicText(statusKey)
        DispatchQueue.main.asyncAfter(deadline: .now() + 60) {
            if UIPasteboard.general.string == text {
                UIPasteboard.general.string = ""
            }
        }
    }
}

extension OneTimeSecretReveal: Identifiable {
    var id: String { token }
}

private struct SecretRevealSheet: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    let secret: OneTimeSecretReveal
    let onDismiss: () -> Void

    @State private var copied = false

    var body: some View {
        NavigationStack {
            VStack(alignment: .leading, spacing: 16) {
                Text(L.text("mobile.integrations.secret.warning"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.amber)
                Text(secret.label)
                    .font(.headline)
                Text(secret.token)
                    .font(.caption.monospaced())
                    .textSelection(.enabled)
                    .padding(12)
                    .background(LexturesTheme.cardBackground(for: colorScheme))
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                Button(copied ? L.text("mobile.integrations.secret.copied") : L.text("mobile.integrations.secret.copy")) {
                    UIPasteboard.general.string = secret.token
                    copied = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 60) {
                        if UIPasteboard.general.string == secret.token {
                            UIPasteboard.general.string = ""
                        }
                    }
                }
                .buttonStyle(.borderedProminent)
                Spacer()
            }
            .padding(16)
            .navigationTitle(L.text("mobile.integrations.secret.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.integrations.secret.dismiss")) {
                        dismiss()
                        onDismiss()
                    }
                }
            }
        }
        .interactiveDismissDisabled()
    }
}

private extension String {
    func ifEmpty(_ fallback: String) -> String {
        isEmpty ? fallback : self
    }
}