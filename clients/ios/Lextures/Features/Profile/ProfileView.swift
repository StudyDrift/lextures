import SwiftUI

/// Profile tab: identity hero, notifications, app info, and sign-out.
struct ProfileView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(LocalePreferences.self) private var localePreferences
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.accessibilityPreferences) private var accessibilityPreferences
    @State private var confirmingSignOut = false
    @State private var confirmingClearCache = false
    @State private var localeError: String?
    @Environment(BiometricGate.self) private var biometricGate

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        identityHero
                        if offline.pendingCount > 0 {
                            offlineSyncCard
                        }
                        offlineStorageCard
                        localeCard
                        accessibilityCard
                        securityCard
                        accountCard
                        notificationsCard
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
            .navigationDestination(for: NotificationsRoute.self) { _ in
                NotificationsView()
            }
            .navigationDestination(for: DeviceSessionsRoute.self) { _ in
                DeviceSessionsView()
            }
            .confirmationDialog(
                L.text("mobile.profile.signOutConfirm"),
                isPresented: $confirmingSignOut,
                titleVisibility: .visible
            ) {
                Button(L.text("mobile.profile.signOut"), role: .destructive) {
                    session.signOut()
                }
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
            .task { await offline.refreshState() }
        }
    }

    private var identityHero: some View {
        ZStack(alignment: .topTrailing) {
            Circle()
                .fill(.white.opacity(0.07))
                .frame(width: 150, height: 150)
                .offset(x: 46, y: -56)

            VStack(spacing: 10) {
                Circle()
                    .fill(.white.opacity(0.16))
                    .frame(width: 76, height: 76)
                    .overlay(
                        Text(shell.profile?.initials ?? "··")
                            .font(LexturesTheme.displayFont(28, weight: .bold))
                            .foregroundStyle(.white)
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
        let name = shell.profile?.displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !name.isEmpty { return name }
        return shell.profile?.firstName ?? L.text("mobile.profile.welcome")
    }

    private var offlineSyncCard: some View {
        LMSCard {
            Text(L.text("mobile.profile.pendingSync"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(L.plural("mobile.pendingSync.waiting", count: offline.pendingCount))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            ForEach(offline.outboxItems.filter {
                $0.status == .queued || $0.status == .failed || $0.status == .conflict
            }) { item in
                Divider()
                VStack(alignment: .leading, spacing: 6) {
                    Text(item.label)
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    OutboxStatusChip(status: item.status)
                    if item.status == .failed || item.status == .conflict {
                        Button(L.text("mobile.profile.retry")) {
                            Task { await offline.retryOutboxItem(id: item.id, accessToken: session.accessToken) }
                        }
                        .font(.caption.weight(.semibold))
                    }
                }
            }
        }
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
        }
    }

    private var securityCard: some View {
        LMSCard {
            Text(L.text("mobile.profile.security"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            if biometricGate.canEnableBiometrics {
                Toggle(isOn: Binding(
                    get: { biometricGate.isEnabled },
                    set: { biometricGate.isEnabled = $0 }
                )) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L.format("mobile.biometric.toggle", biometricGate.biometryLabel))
                            .font(.subheadline.weight(.medium))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(L.text("mobile.biometric.toggleDescription"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
                Divider()
            }

            NavigationLink(value: DeviceSessionsRoute()) {
                HStack(spacing: 12) {
                    Image(systemName: "desktopcomputer")
                        .font(.footnote.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                        .frame(width: 32, height: 32)
                        .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.18 : 0.14))
                        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L.text("mobile.sessions.title"))
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(L.text("mobile.sessions.profileHint"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer(minLength: 0)
                    Image(systemName: "chevron.right")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
                        .flipsForRightToLeftLayoutDirection(true)
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
        }
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
        }
    }

    private var notificationsCard: some View {
        LMSCard {
            NavigationLink(value: NotificationsRoute()) {
                HStack(spacing: 12) {
                    Image(systemName: "bell.fill")
                        .font(.footnote.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                        .frame(width: 32, height: 32)
                        .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.18 : 0.14))
                        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L.text("mobile.profile.notifications"))
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(
                            shell.unreadNotifications > 0
                                ? L.format("mobile.profile.unread", shell.unreadNotifications)
                                : L.text("mobile.dashboard.caughtUp")
                        )
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer(minLength: 0)
                    if shell.unreadNotifications > 0 {
                        Text("\(shell.unreadNotifications)")
                            .font(.caption.weight(.bold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 3)
                            .background(LexturesTheme.coral)
                            .clipShape(Capsule())
                    }
                    Image(systemName: "chevron.right")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
                        .flipsForRightToLeftLayoutDirection(true)
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .simultaneousGesture(TapGesture().onEnded {
                Task { await PushManager.shared.requestPermissionIfNeeded() }
            })
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
