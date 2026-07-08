import SwiftUI

/// One-time onboarding completion celebration (IC07).
struct IntroCompletionCelebrationSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    @Binding var isPresented: Bool
    let progress: IntroCourseProgress
    var onDismissed: () -> Void = {}

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                VStack(spacing: 16) {
                    if !reduceMotion {
                        celebrationDots
                    }
                    Image(systemName: "party.popper.fill")
                        .font(.system(size: 44))
                        .foregroundStyle(LexturesTheme.primary)
                        .accessibilityHidden(true)
                    Text(L.text("mobile.introCourse.celebration.title"))
                        .font(LexturesTheme.displayFont(22))
                        .multilineTextAlignment(.center)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(credentialAvailable
                         ? L.text("mobile.introCourse.celebration.bodyWithCredential")
                         : L.text("mobile.introCourse.celebration.body"))
                        .font(.subheadline)
                        .multilineTextAlignment(.center)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if credentialAvailable {
                        LMSCard {
                            Label {
                                Text(L.text("mobile.introCourse.celebration.badgeLabel"))
                                    .font(.subheadline.weight(.semibold))
                            } icon: {
                                Image(systemName: "rosette")
                                    .foregroundStyle(LexturesTheme.primary)
                            }
                            Button(L.text("mobile.introCourse.celebration.credentialsLink")) {
                                dismiss(andOpenCredentials: true)
                            }
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.primary)
                        }
                    } else {
                        Button(L.text("mobile.introCourse.celebration.credentialsLink")) {
                            dismiss(andOpenCredentials: true)
                        }
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.primary)
                    }

                    Button(L.text("mobile.introCourse.celebration.close")) {
                        dismiss(andOpenCredentials: false)
                    }
                    .buttonStyle(AuthPrimaryButtonStyle())
                    .padding(.top, 8)
                }
                .padding(24)
            }
            .navigationBarTitleDisplayMode(.inline)
            .accessibilityElement(children: .contain)
            .accessibilityLabel(L.text("mobile.introCourse.celebration.ariaLabel"))
            .onAppear { IntroCourseObservability.recordCelebrationView() }
        }
        .presentationDetents([.medium, .large])
    }

    private var credentialAvailable: Bool {
        shell.platformFeatures.ffCompletionCredentials && progress.credentialId != nil
    }

    private var celebrationDots: some View {
        HStack(spacing: 18) {
            Circle().fill(LexturesTheme.amber).frame(width: 8, height: 8)
            Circle().fill(LexturesTheme.primary).frame(width: 8, height: 8)
            Circle().fill(LexturesTheme.coral).frame(width: 8, height: 8)
        }
        .padding(.bottom, 4)
        .accessibilityHidden(true)
    }

    private func dismiss(andOpenCredentials: Bool) {
        Task {
            if let token = session.accessToken {
                try? await LMSAPI.markIntroCelebrationSeen(accessToken: token)
                _ = try? await offline.cachedFetch(
                    key: OfflineCacheKey.introCourseProgress(),
                    accessToken: token
                ) {
                    try await LMSAPI.fetchIntroCourseProgress(accessToken: token)
                }
            }
            await MainActor.run {
                isPresented = false
                onDismissed()
                if andOpenCredentials {
                    shell.openDeepLink(.credentials)
                }
            }
        }
    }
}