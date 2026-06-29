import LocalAuthentication
import SwiftUI

/// Full-screen overlay shown when biometric lock is active.
struct BiometricLockView: View {
    @Environment(BiometricGate.self) private var biometricGate
    @Environment(\.colorScheme) private var colorScheme
    @State private var isUnlocking = false

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            VStack(spacing: 24) {
                Image(systemName: biometricIcon)
                    .font(.system(size: 56))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .accessibilityHidden(true)

                Text(L.text("mobile.biometric.lockedTitle"))
                    .font(LexturesTheme.displayFont(24, weight: .semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .multilineTextAlignment(.center)

                Text(L.format("mobile.biometric.lockedSubtitle", biometricGate.biometryLabel))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .multilineTextAlignment(.center)

                Button {
                    Task { await attemptUnlock() }
                } label: {
                    Group {
                        if isUnlocking {
                            ProgressView()
                                .tint(.white)
                        } else {
                            Text(L.text("mobile.biometric.unlock"))
                        }
                    }
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.white)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(LexturesTheme.primaryDeep)
                    .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
                }
                .buttonStyle(.plain)
                .disabled(isUnlocking)
                .accessibilityLabel(L.text("mobile.biometric.unlock"))
            }
            .padding(32)
        }
        .task {
            await attemptUnlock()
        }
    }

    private var biometricIcon: String {
        switch LAContext().biometryType {
        case .faceID:
            return "faceid"
        case .touchID:
            return "touchid"
        default:
            return "lock.fill"
        }
    }

    @MainActor
    private func attemptUnlock() async {
        guard !isUnlocking else { return }
        isUnlocking = true
        defer { isUnlocking = false }
        _ = await biometricGate.unlock()
    }
}
