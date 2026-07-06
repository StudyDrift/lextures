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
