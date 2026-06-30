import SwiftUI

/// Per-category notification preferences (push + email) persisted via the preferences API.
struct NotificationPreferencesView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    @State private var preferences: [NotificationPreference] = []
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var saveMessage: String?

    private var ownerKey: String {
        NotebookStore.jwtSubject(from: session.accessToken) ?? "anonymous"
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.notifications.preferences.description"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if let saveMessage {
                        Text(saveMessage)
                            .font(.caption.weight(.medium))
                            .foregroundStyle(LexturesTheme.brandTeal)
                    }

                    if loading && preferences.isEmpty {
                        LMSSkeletonList(count: 4)
                    } else {
                        ForEach(NotificationLogic.groupedPreferences(preferences), id: \.0) { category, rows in
                            preferenceSection(category: category, rows: rows)
                        }
                    }
                }
                .padding(16)
            }
            .refreshable { await load(force: true) }
        }
        .navigationTitle(L.text("mobile.notifications.preferences.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load(force: false) }
    }

    private func preferenceSection(
        category: NotificationCategory,
        rows: [NotificationPreference]
    ) -> some View {
        LMSCard {
            Text(NotificationLogic.categoryLabel(for: category))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            ForEach(Array(rows.enumerated()), id: \.element.id) { index, row in
                if index > 0 { Divider() }
                preferenceRow(row)
            }
        }
    }

    private func preferenceRow(_ row: NotificationPreference) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(NotificationLogic.eventLabel(for: row.eventType))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            Toggle(isOn: pushBinding(for: row.eventType)) {
                Text(L.text("mobile.notifications.preferences.push"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .accessibilityLabel("\(NotificationLogic.eventLabel(for: row.eventType)), \(L.text("mobile.notifications.preferences.push"))")

            Toggle(isOn: emailBinding(for: row.eventType)) {
                Text(L.text("mobile.notifications.preferences.email"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .accessibilityLabel("\(NotificationLogic.eventLabel(for: row.eventType)), \(L.text("mobile.notifications.preferences.email"))")
        }
        .padding(.vertical, 2)
    }

    private func pushBinding(for eventType: String) -> Binding<Bool> {
        Binding(
            get: { preferences.first(where: { $0.eventType == eventType })?.pushEnabled ?? true },
            set: { newValue in
                updatePreference(eventType: eventType) { $0.pushEnabled = newValue }
                Task { await persistPreference(eventType: eventType) }
            }
        )
    }

    private func emailBinding(for eventType: String) -> Binding<Bool> {
        Binding(
            get: { preferences.first(where: { $0.eventType == eventType })?.emailEnabled ?? true },
            set: { newValue in
                updatePreference(eventType: eventType) { $0.emailEnabled = newValue }
                Task { await persistPreference(eventType: eventType) }
            }
        )
    }

    private func updatePreference(eventType: String, mutate: (inout NotificationPreference) -> Void) {
        guard let index = preferences.firstIndex(where: { $0.eventType == eventType }) else { return }
        mutate(&preferences[index])
    }

    private func load(force: Bool) async {
        guard let token = session.accessToken else { return }
        if !force, !preferences.isEmpty { return }
        loading = true
        errorMessage = nil
        defer { loading = false }

        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.notificationPreferences(),
                accessToken: token
            ) {
                try await LMSAPI.fetchNotificationPreferences(accessToken: token)
            }
            preferences = result.value
            NotificationPreferencesCache.save(preferences, ownerKey: ownerKey)
        } catch {
            let cached = NotificationPreferencesCache.load(ownerKey: ownerKey)
            if !cached.isEmpty {
                preferences = cached
            } else {
                errorMessage = (error as? LocalizedError)?.errorDescription
                    ?? L.text("mobile.notifications.preferences.error.load")
            }
        }
    }

    private func persistPreference(eventType: String) async {
        guard let token = session.accessToken,
              let row = preferences.first(where: { $0.eventType == eventType }) else { return }
        saveMessage = nil

        let patch = NotificationPreferencesUpdate(
            preferences: [
                NotificationPreferencePatch(
                    eventType: row.eventType,
                    emailEnabled: row.emailEnabled,
                    pushEnabled: row.pushEnabled
                )
            ]
        )

        do {
            _ = try await offline.enqueueMutation(
                method: "PUT",
                path: "/api/v1/me/notification-preferences",
                body: patch,
                label: L.text("mobile.notifications.preferences.saveLabel"),
                accessToken: token
            )
            NotificationPreferencesCache.save(preferences, ownerKey: ownerKey)
            saveMessage = L.text("mobile.notifications.preferences.saved")
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.notifications.preferences.error.save")
            await load(force: true)
        }
    }
}

struct NotificationPreferencesRoute: Hashable {}
