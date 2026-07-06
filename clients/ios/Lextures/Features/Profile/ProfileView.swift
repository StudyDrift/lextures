import SwiftUI

/// Profile tab: identity hero, notifications, app info, and sign-out.
struct ProfileView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(LocalePreferences.self) private var localePreferences
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.accessibilityPreferences) private var accessibilityPreferences
    @Environment(UIModeStore.self) private var uiModeStore
    @State private var confirmingSignOut = false
    @State private var confirmingClearCache = false
    @State private var confirmingClearSearchHistory = false
    @State private var navigatedMoreDestination: MoreDestination?
    @State private var billingNav: BillingRoute?
    @State private var localeError: String?

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        identityHero
                        if shell.iaRedesignEnabled {
                            ProfileIaContextCard()
                            ProfileMoreHubCard()
                        }
                        if offline.pendingCount > 0 {
                            ProfileOfflineSyncCard()
                        }
                        ProfilePersonalCard()
                        ProfileDepthCards()
                        offlineStorageCard
                        ProfileAppearanceCard()
                        localeCard
                        if uiModeStore.featureEnabled {
                            uiModeCard
                        }
                        accessibilityCard
                        ProfileSecurityCard()
                        accountCard
                        ProfileNotificationsCard()
                        ProfileLegalCard()
                        aboutCard
                        signOutButton
                    }
                    .padding(16)
                }
                .refreshable {
                    await shell.refresh(accessToken: session.accessToken)
                }
            }
            .navigationTitle(L.text("mobile.profile.title"))
            .navigationBarTitleDisplayMode(.inline)
            .globalDrawerToolbar()
            .navigationDestination(for: NotificationsRoute.self) { _ in
                NotificationsView()
            }
            .navigationDestination(for: DeviceSessionsRoute.self) { _ in
                DeviceSessionsView()
            }
            .navigationDestination(for: EditProfileRoute.self) { _ in
                EditProfileView()
            }
            .navigationDestination(for: MyAccommodationsRoute.self) { _ in
                MyAccommodationsView()
            }
            .navigationDestination(for: ProfilePersonalDetailsRoute.self) { _ in
                ProfilePersonalDetailsView()
            }
            .navigationDestination(for: ResearchStudiesRoute.self) { _ in
                ResearchStudiesView()
            }
            .navigationDestination(for: MoreHubRoute.self) { _ in
                MoreHubView()
            }
            .navigationDestination(item: $billingNav) { _ in
                BillingView()
            }
            .navigationDestination(for: MoreDestination.self) { destination in
                ProfileMoreDestinationScreen(destination: destination)
            }
            .navigationDestination(item: $navigatedMoreDestination) { destination in
                ProfileMoreDestinationScreen(destination: destination)
            }
            .confirmationDialog(
                L.text("mobile.profile.clearCacheConfirm"),
                isPresented: $confirmingClearCache,
                titleVisibility: .visible
            ) {
                Button(L.text("mobile.profile.clearCache"), role: .destructive) {
                    Task { await offline.clearStorage() }
                }
            } message: {
                Text(L.text("mobile.profile.clearCacheMessage"))
            }
            .confirmationDialog(
                L.text("mobile.search.clearHistoryConfirm"),
                isPresented: $confirmingClearSearchHistory,
                titleVisibility: .visible
            ) {
                Button(L.text("mobile.search.clearHistory"), role: .destructive) {
                    SearchRecentsStore.clearAll()
                }
            } message: {
                Text(L.text("mobile.search.clearHistoryMessage"))
            }
            .task { await offline.refreshState() }
            .onAppear {
                openPendingMoreDestinationIfNeeded()
                if shell.consumePendingBilling() {
                    billingNav = BillingRoute()
                }
            }
        }
    }

    private var identityHero: some View {
        ZStack(alignment: .topTrailing) {
            Circle()
                .fill(.white.opacity(0.07))
                .frame(width: 150, height: 150)
                .offset(x: 46, y: -56)

            VStack(spacing: 10) {
                ProfileAvatarView(
                    avatarUrl: shell.accountProfile?.avatarUrl,
                    initials: profileInitials,
                    size: 76,
                    initialsBackground: .white.opacity(0.16),
                    initialsForeground: .white
                )
                Text(displayName)
                    .font(LexturesTheme.displayFont(22))
                    .foregroundStyle(.white)
                Text(shell.profile?.email ?? session.userEmail ?? "")
                    .font(.footnote)
                    .foregroundStyle(.white.opacity(0.8))
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 26)
        }
        .background(LexturesTheme.heroGradient)
        .clipShape(RoundedRectangle(cornerRadius: 24, style: .continuous))
        .shadow(color: LexturesTheme.primaryDeep.opacity(0.25), radius: 14, y: 7)
    }

    private var displayName: String {
        if let account = shell.accountProfile {
            return account.resolvedDisplayName
        }
        let name = shell.profile?.displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !name.isEmpty { return name }
        return shell.profile?.firstName ?? L.text("mobile.profile.welcome")
    }

    private var profileInitials: String {
        shell.accountProfile?.resolvedInitials ?? shell.profile?.initials ?? "··"
    }

    private var localeCard: some View {
        LMSCard {
            Text(L.text("common.locale.label"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(L.text("common.locale.description"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Picker(L.text("common.locale.label"), selection: Binding(
                get: { localePreferences.localeTag },
                set: { newTag in Task { await saveLocale(tag: newTag) } }
            )) {
                ForEach(LocalePreferences.localeOptions, id: \.tag) { option in
                    Text(option.tag == "system" ? L.text("common.locale.systemDefault") : option.label)
                        .tag(option.tag)
                }
            }
            .pickerStyle(.menu)
            if let localeError {
                Text(localeError)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.error)
            }
        }
    }

    private var uiModeCard: some View {
        LMSCard {
            Text(L.text("mobile.uiMode.title"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(L.text("mobile.uiMode.description"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            if uiModeStore.hasAdminOverride {
                Text(L.text("mobile.uiMode.adminOverride"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            Picker(L.text("mobile.uiMode.title"), selection: Binding(
                get: { uiModeStore.localPreference },
                set: { uiModeStore.localPreference = $0 }
            )) {
                ForEach(UIModePreference.allCases) { option in
                    Text(option.label).tag(option)
                }
            }
            .pickerStyle(.menu)
            .disabled(uiModeStore.hasAdminOverride)
        }
    }

    private var accessibilityCard: some View {
        LMSCard {
            Text(L.text("mobile.profile.accessibility"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Toggle(isOn: Binding(
                get: { accessibilityPreferences.dyslexiaDisplayEnabled },
                set: { accessibilityPreferences.dyslexiaDisplayEnabled = $0 }
            )) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(L.text("mobile.profile.dyslexiaFriendly"))
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(L.text("mobile.profile.dyslexiaFriendlyDescription"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    private var offlineStorageCard: some View {
        LMSCard {
            Text(L.text("mobile.profile.offlineStorage"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            infoRow(
                label: L.text("mobile.profile.cacheSize"),
                value: ByteCountFormatter.string(fromByteCount: Int64(offline.storageBytes), countStyle: .file),
                systemImage: "internaldrive"
            )
            Divider()
            Button {
                confirmingClearCache = true
            } label: {
                Label(L.text("mobile.profile.clearCachedData"), systemImage: "trash")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.error)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
            .buttonStyle(.plain)
            if shell.universalSearchEnabled {
                Divider()
                Button {
                    confirmingClearSearchHistory = true
                } label: {
                    Label(L.text("mobile.search.clearHistory"), systemImage: "clock.arrow.circlepath")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.error)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
                .buttonStyle(.plain)
            }
        }
    }

    private func openPendingMoreDestinationIfNeeded() {
        guard let destination = shell.consumePendingMoreDestination() else { return }
        navigatedMoreDestination = destination
    }

    private var accountCard: some View {
        LMSCard {
            Text(L.text("mobile.profile.account"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            infoRow(label: L.text("mobile.profile.displayName"), value: displayName, systemImage: "person")
            Divider()
            infoRow(
                label: L.text("mobile.profile.email"),
                value: shell.profile?.email ?? session.userEmail ?? L.text("mobile.emDash"),
                systemImage: "envelope"
            )
            if BillingLogic.billingEnabled(shell.platformFeatures) {
                Divider()
                Button {
                    billingNav = BillingRoute()
                } label: {
                    Label(L.text("mobile.billing.title"), systemImage: "creditcard")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.primary)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
                .buttonStyle(.plain)
            }
        }
    }

    private var aboutCard: some View {
        LMSCard {
            Text(L.text("mobile.profile.about"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            infoRow(label: L.text("mobile.profile.version"), value: appVersion, systemImage: "app.badge")
            Divider()
            infoRow(label: L.text("mobile.profile.server"), value: AppConfiguration.apiBaseURL.absoluteString, systemImage: "server.rack")
        }
    }

    private var appVersion: String {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0"
        let build = Bundle.main.infoDictionary?["CFBundleVersion"] as? String
        if let build, !build.isEmpty { return "\(version) (\(build))" }
        return version
    }

    private var signOutButton: some View {
        Button {
            confirmingSignOut = true
        } label: {
            Label(L.text("mobile.profile.signOut"), systemImage: "rectangle.portrait.and.arrow.right")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.error)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 14)
                .background(LexturesTheme.error.opacity(0.09))
                .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
        }
        .buttonStyle(.plain)
        .confirmationDialog(
            L.text("mobile.profile.signOutConfirm"),
            isPresented: $confirmingSignOut,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.profile.signOut"), role: .destructive) {
                session.signOut()
            }
        }
    }

    @MainActor
    private func saveLocale(tag: String) async {
        localeError = nil
        let previous = localePreferences.localeTag
        localePreferences.localeTag = tag
        guard let token = session.accessToken else { return }
        let apiTag = tag == "system" ? Locale.current.identifier : tag
        do {
            let saved = try await LocaleAPI.saveLocale(apiTag, accessToken: token)
            localePreferences.applyStoredTag(saved)
        } catch {
            localePreferences.localeTag = previous
            localeError = L.text("common.locale.saveError")
        }
    }

    private func infoRow(label: String, value: String, systemImage: String) -> some View {
        HStack(spacing: 12) {
            Image(systemName: systemImage)
                .font(.footnote)
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                .frame(width: 24)
            VStack(alignment: .leading, spacing: 2) {
                Text(label)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(value)
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .lineLimit(1)
                    .truncationMode(.middle)
            }
            Spacer(minLength: 0)
        }
    }
}
