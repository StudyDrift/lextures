import SwiftUI

/// Post-auth shell: Dashboard, Courses, Notebooks, Inbox tabs.
struct MainTabView: View {
    @Environment(AuthSession.self) private var session
    @State private var unreadInbox = 0

    var body: some View {
        TabView {
            DashboardView(unreadInbox: $unreadInbox)
                .tabItem { Label("Home", systemImage: "house.fill") }

            CoursesListView()
                .tabItem { Label("Courses", systemImage: "books.vertical.fill") }

            NotebooksListView()
                .tabItem { Label("Notebooks", systemImage: "square.and.pencil") }

            InboxView(unreadInbox: $unreadInbox)
                .tabItem { Label("Inbox", systemImage: "tray.fill") }
                .badge(unreadInbox)
        }
        .tint(LexturesTheme.primary)
        .task {
            await refreshUnread()
        }
    }

    private func refreshUnread() async {
        guard let token = session.accessToken else { return }
        unreadInbox = (try? await LMSAPI.fetchUnreadInboxCount(accessToken: token)) ?? unreadInbox
    }
}

// MARK: - Shared LMS UI helpers

/// Floating card: generous radius, soft warm shadow, hairline border in dark mode only.
struct LMSCard<Content: View>: View {
    @Environment(\.colorScheme) private var colorScheme
    var accent: Color? = nil
    @ViewBuilder var content: () -> Content

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            content()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(16)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(alignment: .leading) {
            if let accent {
                UnevenRoundedRectangle(
                    topLeadingRadius: 18,
                    bottomLeadingRadius: 18,
                    bottomTrailingRadius: 0,
                    topTrailingRadius: 0,
                    style: .continuous
                )
                .fill(accent)
                .frame(width: 4)
            }
        }
        .overlay(
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .stroke(
                    LexturesTheme.fieldBorder(for: colorScheme).opacity(colorScheme == .dark ? 0.9 : 0.45),
                    lineWidth: 1
                )
        )
        .shadow(color: LexturesTheme.cardShadow(for: colorScheme), radius: 12, y: 5)
    }
}

/// Serif section header — editorial, like a textbook chapter heading.
struct LMSSectionHeader: View {
    @Environment(\.colorScheme) private var colorScheme
    let title: String
    var systemImage: String?

    var body: some View {
        HStack(spacing: 8) {
            if let systemImage {
                Image(systemName: systemImage)
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 26, height: 26)
                    .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.18 : 0.16))
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
            }
            Text(title)
                .font(LexturesTheme.displayFont(19))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
        .padding(.top, 6)
    }
}

struct LMSErrorBanner: View {
    let message: String

    var body: some View {
        Label(message, systemImage: "exclamationmark.triangle.fill")
            .font(.subheadline)
            .foregroundStyle(LexturesTheme.error)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(14)
            .background(LexturesTheme.error.opacity(0.09))
            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
    }
}

struct LMSEmptyState: View {
    @Environment(\.colorScheme) private var colorScheme
    let systemImage: String
    let title: String
    let message: String

    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: systemImage)
                .font(.system(size: 28))
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                .frame(width: 72, height: 72)
                .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.16 : 0.14))
                .clipShape(Circle())
            Text(title)
                .font(LexturesTheme.displayFont(18))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(message)
                .font(.subheadline)
                .multilineTextAlignment(.center)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 36)
        .padding(.horizontal, 24)
    }
}

/// Rounded gradient tile used as a course "cover" thumbnail.
struct LMSCoverTile: View {
    let key: String
    let systemImage: String
    var size: CGFloat = 48

    var body: some View {
        RoundedRectangle(cornerRadius: size * 0.28, style: .continuous)
            .fill(LexturesTheme.coverGradient(for: key))
            .frame(width: size, height: size)
            .overlay(
                Image(systemName: systemImage)
                    .font(.system(size: size * 0.4, weight: .medium))
                    .foregroundStyle(.white)
            )
    }
}
