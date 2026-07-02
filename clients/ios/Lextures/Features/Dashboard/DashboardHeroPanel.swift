import SwiftUI

/// Deep-teal gradient greeting panel with drawer, search, bell, and avatar.
struct DashboardHeroPanel: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(AuthSession.self) private var session
    let dueThisWeekCount: Int
    let loading: Bool

    var body: some View {
        ZStack(alignment: .topTrailing) {
            Circle()
                .fill(.white.opacity(0.07))
                .frame(width: 160, height: 160)
                .offset(x: 50, y: -60)
            Circle()
                .fill(LexturesTheme.brandCoral.opacity(0.35))
                .frame(width: 56, height: 56)
                .offset(x: -28, y: 26)

            VStack(alignment: .leading, spacing: 6) {
                HStack(alignment: .top, spacing: 12) {
                    Button { shell.openGlobalDrawer() } label: {
                        Image(systemName: "line.3.horizontal")
                            .font(.title3.weight(.semibold))
                            .foregroundStyle(.white)
                    }
                    .accessibilityLabel(L.text("mobile.drawer.menu"))
                    .padding(.top, 2)
                    VStack(alignment: .leading, spacing: 3) {
                        Text(greetingText + ",")
                            .font(LexturesTheme.displayFont(26))
                            .foregroundStyle(.white)
                        Text(greetingFirstName)
                            .font(LexturesTheme.displayFont(26))
                            .foregroundStyle(LexturesTheme.brandCream)
                            .lineLimit(1)
                    }
                    Spacer(minLength: 8)
                    HStack(spacing: 10) {
                        if shell.iaRedesignEnabled && shell.universalSearchEnabled {
                            searchButton
                        }
                        bellButton
                        LMSAvatarButton()
                    }
                }

                if dueThisWeekCount > 0 {
                    Text(L.plural("mobile.dashboard.dueThisWeek.count", count: dueThisWeekCount))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.primaryDeep)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(LexturesTheme.brandCream)
                        .clipShape(Capsule())
                        .padding(.top, 8)
                } else if !loading {
                    Text(L.text("mobile.dashboard.caughtUp"))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.white.opacity(0.9))
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(.white.opacity(0.16))
                        .clipShape(Capsule())
                        .padding(.top, 8)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(20)
        }
        .background(LexturesTheme.heroGradient)
        .clipShape(RoundedRectangle(cornerRadius: 24, style: .continuous))
        .shadow(color: LexturesTheme.primaryDeep.opacity(0.25), radius: 14, y: 7)
    }

    private var searchButton: some View {
        Button {
            shell.showUniversalSearch = true
        } label: {
            Image(systemName: "magnifyingglass")
                .font(.subheadline)
                .foregroundStyle(.white)
                .frame(width: 34, height: 34)
                .background(.white.opacity(0.16))
                .clipShape(Circle())
        }
        .buttonStyle(.plain)
        .accessibilityLabel(L.text("mobile.ia.search"))
    }

    private var bellButton: some View {
        NavigationLink(value: NotificationsRoute()) {
            ZStack(alignment: .topTrailing) {
                Image(systemName: "bell.fill")
                    .font(.subheadline)
                    .foregroundStyle(.white)
                    .frame(width: 34, height: 34)
                    .background(.white.opacity(0.16))
                    .clipShape(Circle())
                if shell.unreadNotifications > 0 {
                    Circle()
                        .fill(LexturesTheme.coral)
                        .frame(width: 9, height: 9)
                        .offset(x: -2, y: 2)
                }
            }
        }
        .buttonStyle(.plain)
        .accessibilityLabel(L.text("mobile.profile.notifications"))
    }

    private var greetingFirstName: String {
        if let account = shell.accountProfile {
            let first = account.resolvedNameFields.firstName
            if !first.isEmpty { return first }
        }
        if let first = shell.profile?.firstName, !first.isEmpty { return first }
        return session.userEmail ?? ""
    }

    private var greetingText: String {
        let hour = Calendar.current.component(.hour, from: Date())
        switch hour {
        case ..<12: return L.text("mobile.dashboard.greeting.morning")
        case ..<17: return L.text("mobile.dashboard.greeting.afternoon")
        default: return L.text("mobile.dashboard.greeting.evening")
        }
    }
}