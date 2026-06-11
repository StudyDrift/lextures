import SwiftUI

/// Post-auth shell: Dashboard, Courses, Notebooks, Inbox tabs.
struct MainTabView: View {
    @Environment(AuthSession.self) private var session
    @State private var unreadInbox = 0

    var body: some View {
        TabView {
            DashboardView(unreadInbox: $unreadInbox)
                .tabItem { Label("Dashboard", systemImage: "rectangle.grid.2x2") }

            CoursesListView()
                .tabItem { Label("Courses", systemImage: "book") }

            NotebooksListView()
                .tabItem { Label("Notebooks", systemImage: "note.text") }

            InboxView(unreadInbox: $unreadInbox)
                .tabItem { Label("Inbox", systemImage: "tray") }
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

struct LMSCard<Content: View>: View {
    @Environment(\.colorScheme) private var colorScheme
    @ViewBuilder var content: () -> Content

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            content()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(16)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 12, style: .continuous)
                .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.9), lineWidth: 1)
        )
    }
}

struct LMSSectionHeader: View {
    @Environment(\.colorScheme) private var colorScheme
    let title: String
    var systemImage: String?

    var body: some View {
        HStack(spacing: 6) {
            if let systemImage {
                Image(systemName: systemImage)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.primary)
            }
            Text(title)
                .font(.headline)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
    }
}

struct LMSErrorBanner: View {
    let message: String

    var body: some View {
        Label(message, systemImage: "exclamationmark.triangle")
            .font(.subheadline)
            .foregroundStyle(LexturesTheme.error)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(12)
            .background(LexturesTheme.error.opacity(0.08))
            .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
    }
}

struct LMSEmptyState: View {
    @Environment(\.colorScheme) private var colorScheme
    let systemImage: String
    let title: String
    let message: String

    var body: some View {
        VStack(spacing: 10) {
            Image(systemName: systemImage)
                .font(.system(size: 34))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Text(title)
                .font(.headline)
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
