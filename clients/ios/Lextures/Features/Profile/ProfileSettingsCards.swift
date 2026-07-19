import SwiftUI

/// Entry points to the editable profile and the student's accommodations.
struct ProfilePersonalCard: View {
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        LMSCard {
            SettingsNavigationRow(
                route: EditProfileRoute(),
                systemImage: "person.text.rectangle",
                title: L.text("mobile.editProfile.title"),
                subtitle: L.text("mobile.editProfile.subtitle")
            )
            Divider()
            SettingsNavigationRow(
                route: MyAccommodationsRoute(),
                systemImage: "checkmark.seal",
                title: L.text("mobile.accommodations.title"),
                subtitle: L.text("mobile.accommodations.subtitle")
            )
        }
    }
}

/// Device override for age-appropriate UI mode (M10.4).
struct ProfileUIModeCard: View {
    @Environment(UIModeStore.self) private var uiModeStore
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        if uiModeStore.featureEnabled {
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
    }
}

/// Theme override (system / light / dark) — a device-only preference.
struct ProfileAppearanceCard: View {
    @Environment(ThemePreference.self) private var themePreference
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        LMSCard {
            Text(L.text("mobile.settings.appearance"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(L.text("mobile.settings.appearance.description"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Picker(L.text("mobile.settings.appearance"), selection: Binding(
                get: { themePreference.appearance },
                set: { themePreference.appearance = $0 }
            )) {
                ForEach(ThemePreference.Appearance.allCases) { option in
                    Text(L.text(option.labelKey)).tag(option)
                }
            }
            .pickerStyle(.segmented)
        }
    }
}

/// Outbound links to the privacy center, trust center, and accessibility statement.
struct ProfileLegalCard: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    var body: some View {
        LMSCard {
            Text(L.text("mobile.settings.privacyTrust"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            linkRow(systemImage: "hand.raised.fill", title: L.text("mobile.settings.privacyCenter"), path: "/privacy")
            Divider()
            linkRow(systemImage: "checkmark.shield.fill", title: L.text("mobile.settings.trustCenter"), path: "/security")
            Divider()
            linkRow(systemImage: "figure.roll", title: L.text("mobile.settings.accessibilityStatement"), path: "/accessibility")
        }
    }

    private func linkRow(systemImage: String, title: String, path: String) -> some View {
        Button {
            openURL(AppConfiguration.webURL(path: path))
        } label: {
            HStack(spacing: 12) {
                Image(systemName: systemImage)
                    .font(.footnote)
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 24)
                Text(title)
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Spacer(minLength: 0)
                Image(systemName: "arrow.up.right.square")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
            }
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}

/// Chevron row that pushes a value-based navigation destination.
struct SettingsNavigationRow<Route: Hashable>: View {
    @Environment(\.colorScheme) private var colorScheme
    let route: Route
    let systemImage: String
    let title: String
    let subtitle: String

    var body: some View {
        NavigationLink(value: route) {
            HStack(spacing: 12) {
                Image(systemName: systemImage)
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 32, height: 32)
                    .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.18 : 0.14))
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(subtitle)
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

/// Profile tab hero with avatar, display name, and email.
struct ProfileIdentityHero: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell

    var body: some View {
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
}

/// Locale picker with server persistence.
struct ProfileLocaleCard: View {
    @Environment(AuthSession.self) private var session
    @Environment(LocalePreferences.self) private var localePreferences
    @Environment(\.colorScheme) private var colorScheme
    @Binding var localeError: String?

    var body: some View {
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
}

/// Offline cache size and clear-data actions.
struct ProfileOfflineStorageCard: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Binding var confirmingClearCache: Bool
    @Binding var confirmingClearSearchHistory: Bool

    var body: some View {
        LMSCard {
            Text(L.text("mobile.profile.offlineStorage"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            ProfileInfoRow(
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
}

/// Account summary and billing entry point.
struct ProfileAccountCard: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Binding var billingNav: BillingRoute?
    @Binding var purchasesNav: MyPurchasesRoute?

    var body: some View {
        LMSCard {
            Text(L.text("mobile.profile.account"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            ProfileInfoRow(label: L.text("mobile.profile.displayName"), value: displayName, systemImage: "person")
            Divider()
            ProfileInfoRow(
                label: L.text("mobile.profile.email"),
                value: shell.profile?.email ?? session.userEmail ?? L.text("mobile.emDash"),
                systemImage: "envelope"
            )
            if MarketplaceLogic.purchaseEnabled(shell.platformFeatures) {
                Divider()
                Button {
                    purchasesNav = MyPurchasesRoute()
                } label: {
                    Label(L.text("mobile.marketplace.purchases.title"), systemImage: "cart")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.primary)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
                .buttonStyle(.plain)
            }
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

    private var displayName: String {
        if let account = shell.accountProfile {
            return account.resolvedDisplayName
        }
        let name = shell.profile?.displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !name.isEmpty { return name }
        return shell.profile?.firstName ?? L.text("mobile.profile.welcome")
    }
}

/// App version and API server details.
struct ProfileAboutCard: View {
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        LMSCard {
            Text(L.text("mobile.profile.about"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            ProfileInfoRow(label: L.text("mobile.profile.version"), value: appVersion, systemImage: "app.badge")
            Divider()
            ProfileInfoRow(
                label: L.text("mobile.profile.server"),
                value: AppConfiguration.apiBaseURL.absoluteString,
                systemImage: "server.rack"
            )
        }
    }

    private var appVersion: String {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0"
        let build = Bundle.main.infoDictionary?["CFBundleVersion"] as? String
        if let build, !build.isEmpty { return "\(version) (\(build))" }
        return version
    }
}

/// Sign-out control with confirmation dialog.
struct ProfileSignOutButton: View {
    @Environment(AuthSession.self) private var session
    @Binding var confirmingSignOut: Bool

    var body: some View {
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
}

/// Label/value row used across profile cards.
struct ProfileInfoRow: View {
    @Environment(\.colorScheme) private var colorScheme
    let label: String
    let value: String
    let systemImage: String

    var body: some View {
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
