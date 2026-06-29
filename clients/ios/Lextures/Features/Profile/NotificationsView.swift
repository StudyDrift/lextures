import SwiftUI

/// In-app notification inbox (`/me/notifications`): filter chips, mark-read on tap,
/// mark-all-read in the toolbar.
struct NotificationsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    private enum Filter: String, CaseIterable {
        case all = "All"
        case unread = "Unread"
    }

    @State private var filter: Filter = .all
    @State private var notifications: [AppNotification] = []
    @State private var errorMessage: String?
    @State private var loading = true

    private var visible: [AppNotification] {
        filter == .unread ? notifications.filter { !$0.isRead } : notifications
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    LMSSegmentedChips(
                        options: Filter.allCases,
                        selection: $filter,
                        label: \.rawValue
                    )

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && notifications.isEmpty {
                        LMSSkeletonList(count: 5)
                    } else if visible.isEmpty {
                        LMSEmptyState(
                            systemImage: "bell",
                            title: filter == .unread ? "No unread notifications" : "No notifications",
                            message: "Course activity and updates will appear here."
                        )
                    } else {
                        ForEach(visible) { notification in
                            Button {
                                Task { await openNotification(notification) }
                            } label: {
                                notificationCard(notification)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
                .padding(16)
            }
            .refreshable { await load() }
        }
        .navigationTitle("Notifications")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Mark all read") {
                    Task { await markAllRead() }
                }
                .font(.subheadline.weight(.medium))
                .disabled(notifications.allSatisfy(\.isRead))
            }
        }
        .task { await load() }
    }

    private func notificationCard(_ notification: AppNotification) -> some View {
        LMSCard(accent: notification.isRead ? nil : LexturesTheme.brandTeal) {
            HStack(alignment: .top, spacing: 12) {
                Image(systemName: icon(for: notification.eventType))
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 32, height: 32)
                    .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.18 : 0.14))
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))

                VStack(alignment: .leading, spacing: 3) {
                    HStack(alignment: .top) {
                        Text(notification.title)
                            .font(.subheadline.weight(notification.isRead ? .regular : .semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Spacer(minLength: 8)
                        Text(LMSDates.relative(notification.createdAt))
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    if !notification.body.isEmpty {
                        Text(notification.body)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            .lineLimit(3)
                    }
                }

                if !notification.isRead {
                    Circle()
                        .fill(LexturesTheme.coral)
                        .frame(width: 8, height: 8)
                        .padding(.top, 6)
                }
            }
        }
    }

    private func icon(for eventType: String) -> String {
        switch true {
        case eventType.contains("grade"): return "checkmark.seal.fill"
        case eventType.contains("message"), eventType.contains("inbox"): return "envelope.fill"
        case eventType.contains("due"), eventType.contains("assignment"): return "clock.fill"
        case eventType.contains("announcement"), eventType.contains("broadcast"): return "megaphone.fill"
        default: return "bell.fill"
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let page = try await LMSAPI.fetchNotifications(accessToken: token)
            notifications = page.notifications
            shell.unreadNotifications = page.unreadCount
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load notifications."
        }
    }

    private func openNotification(_ notification: AppNotification) async {
        await markRead(notification)
        if let actionUrl = notification.actionUrl {
            shell.openDeepLink(DeepLinkRouter.resolve(actionUrl))
        }
    }

    private func markRead(_ notification: AppNotification) async {
        guard !notification.isRead, let token = session.accessToken else { return }
        // Optimistic flip; reload only on failure.
        if let index = notifications.firstIndex(where: { $0.id == notification.id }) {
            notifications[index].isRead = true
            shell.unreadNotifications = max(0, shell.unreadNotifications - 1)
        }
        do {
            try await LMSAPI.markNotificationRead(id: notification.id, accessToken: token)
        } catch {
            await load()
        }
    }

    private func markAllRead() async {
        guard let token = session.accessToken else { return }
        do {
            try await LMSAPI.markAllNotificationsRead(accessToken: token)
            for index in notifications.indices {
                notifications[index].isRead = true
            }
            shell.unreadNotifications = 0
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not mark all as read."
        }
    }
}
