import SwiftUI

/// Shared header affordances for IA redesign root tabs: universal search + notifications.
struct ShellHeaderBar: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    var onSearch: () -> Void

    var body: some View {
        HStack(spacing: 10) {
            Spacer(minLength: 0)
            if shell.universalSearchEnabled {
                searchButton
            }
            notificationsButton
        }
    }

    private var searchButton: some View {
        Button(action: onSearch) {
                Image(systemName: "magnifyingglass")
                    .font(.body.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .frame(width: 44, height: 44)
            }
            .buttonStyle(.plain)
            .accessibilityLabel(L.text("mobile.ia.search"))
    }

    private var notificationsButton: some View {
        NavigationLink(value: NotificationsRoute()) {
                ZStack(alignment: .topTrailing) {
                    Image(systemName: "bell.fill")
                        .font(.body.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .frame(width: 44, height: 44)
                    if shell.unreadNotifications > 0 {
                        Circle()
                            .fill(LexturesTheme.coral)
                            .frame(width: 8, height: 8)
                            .offset(x: 2, y: 2)
                    }
                }
            }
            .buttonStyle(.plain)
            .accessibilityLabel(L.text("mobile.profile.notifications"))
    }
}