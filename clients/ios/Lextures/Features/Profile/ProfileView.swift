import SwiftUI

/// Profile tab: identity hero, notifications, app info, and sign-out.
struct ProfileView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @State private var confirmingSignOut = false

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        identityHero
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
            .navigationTitle("Profile")
            .navigationBarTitleDisplayMode(.inline)
            .navigationDestination(for: NotificationsRoute.self) { _ in
                NotificationsView()
            }
            .confirmationDialog(
                "Sign out of Lextures?",
                isPresented: $confirmingSignOut,
                titleVisibility: .visible
            ) {
                Button("Sign out", role: .destructive) {
                    session.signOut()
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
        return shell.profile?.firstName ?? "Welcome"
    }

    private var accountCard: some View {
        LMSCard {
            Text("Account")
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            infoRow(label: "Display name", value: displayName, systemImage: "person")
            Divider()
            infoRow(
                label: "Email",
                value: shell.profile?.email ?? session.userEmail ?? "—",
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
                        Text("Notifications")
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(
                            shell.unreadNotifications > 0
                                ? "\(shell.unreadNotifications) unread"
                                : "You're all caught up"
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
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
        }
    }

    private var aboutCard: some View {
        LMSCard {
            Text("About")
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            infoRow(label: "Version", value: appVersion, systemImage: "app.badge")
            Divider()
            infoRow(label: "Server", value: AppConfiguration.apiBaseURL.absoluteString, systemImage: "server.rack")
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
            Label("Sign out", systemImage: "rectangle.portrait.and.arrow.right")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.error)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 14)
                .background(LexturesTheme.error.opacity(0.09))
                .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
        }
        .buttonStyle(.plain)
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
