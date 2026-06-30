import SwiftUI

/// In-app notification inbox (`/me/notifications`): category filter, mark-read on tap,
/// mark-all-read in the toolbar, and deep-link routing.
struct NotificationsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    @State private var filter: NotificationFilter = .all
    @State private var notifications: [AppNotification] = []
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var staleLabel: String?

    private var visible: [AppNotification] {
        NotificationLogic.filter(notifications, by: filter)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    LMSSegmentedChips(
                        options: NotificationFilter.allCases,
                        selection: $filter,
                        label: \.localizedLabel
                    )

                    if let staleLabel {
                        Text(staleLabel)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && notifications.isEmpty {
                        LMSSkeletonList(count: 5)
                    } else if visible.isEmpty {
                        LMSEmptyState(
                            systemImage: "bell",
                            title: emptyTitle,
                            message: L.text("mobile.notifications.empty.message")
                        )
                    } else {
                        ForEach(visible) { notification in
                            Button {
                                Task { await openNotification(notification) }
                            } label: {
                                notificationCard(notification)
                            }
                            .buttonStyle(.plain)
                            .accessibilityLabel(notificationAccessibilityLabel(notification))
                        }
                    }
                }
                .padding(16)
            }
            .refreshable { await load(force: true) }
        }
        .navigationTitle(L.text("mobile.notifications.title"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                NavigationLink(value: NotificationPreferencesRoute()) {
                    Image(systemName: "gearshape")
                        .accessibilityLabel(L.text("mobile.notifications.preferences.title"))
                }
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button(L.text("mobile.notifications.markAllRead")) {
                    Task { await markAllRead() }
                }
                .font(.subheadline.weight(.medium))
                .disabled(notifications.allSatisfy(\.isRead))
            }
        }
        .navigationDestination(for: NotificationPreferencesRoute.self) { _ in
            NotificationPreferencesView()
        }
        .task { await load(force: false) }
    }

    private var emptyTitle: String {
        switch filter {
        case .unread:
            return L.text("mobile.notifications.empty.unread")
        default:
            return L.text("mobile.notifications.empty.all")
        }
    }

    private func notificationAccessibilityLabel(_ notification: AppNotification) -> String {
        let readState = notification.isRead
            ? L.text("mobile.notifications.accessibility.read")
            : L.text("mobile.notifications.accessibility.unread")
        return "\(readState). \(notification.title). \(notification.body)"
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
                    Text(NotificationLogic.eventLabel(for: notification.eventType))
                        .font(.caption2.weight(.medium))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
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
                        .accessibilityHidden(true)
                }
            }
        }
    }

    private func icon(for eventType: String) -> String {
        switch NotificationLogic.category(for: eventType) {
        case .grades: return "checkmark.seal.fill"
        case .messages: return "envelope.fill"
        case .assignments, .reminders: return "clock.fill"
        case .announcements: return "megaphone.fill"
        case .discussions: return "bubble.left.and.bubble.right.fill"
        default: return "bell.fill"
        }
    }

    private func load(force: Bool) async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }

        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.notificationsPage(),
                accessToken: token
            ) {
                try await LMSAPI.fetchNotifications(accessToken: token)
            }
            notifications = result.value.notifications
            shell.unreadNotifications = result.value.unreadCount
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                staleLabel = cached.lastUpdatedLabel
            } else {
                staleLabel = nil
            }
        } catch {
            if notifications.isEmpty {
                errorMessage = (error as? LocalizedError)?.errorDescription
                    ?? L.text("mobile.notifications.error.load")
            } else {
                staleLabel = staleLabel ?? L.text("mobile.notifications.stale.offline")
            }
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
        if let index = notifications.firstIndex(where: { $0.id == notification.id }) {
            notifications[index].isRead = true
            shell.unreadNotifications = max(0, shell.unreadNotifications - 1)
        }
        do {
            _ = try await offline.enqueueMutation(
                method: "POST",
                path: "/api/v1/me/notifications/\(notification.id.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? notification.id)/read",
                body: nil as String?,
                label: L.text("mobile.notifications.markReadLabel"),
                accessToken: token
            )
        } catch {
            await load(force: true)
        }
    }

    private func markAllRead() async {
        guard let token = session.accessToken else { return }
        do {
            _ = try await offline.enqueueMutation(
                method: "POST",
                path: "/api/v1/me/notifications/read-all",
                body: nil as String?,
                label: L.text("mobile.notifications.markAllReadLabel"),
                accessToken: token
            )
            for index in notifications.indices {
                notifications[index].isRead = true
            }
            shell.unreadNotifications = 0
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.notifications.error.markAllRead")
        }
    }
}
