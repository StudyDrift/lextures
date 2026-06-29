import SwiftUI

struct ProfileSecurityCard: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(BiometricGate.self) private var biometricGate

    var body: some View {
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
}
