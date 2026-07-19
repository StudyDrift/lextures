import SwiftUI

struct AuditLogAdminRoute: Hashable {}

/// Read-only admin audit log (MOB.3 Phase 1) — parity with web AdminAuditLog.
struct AuditLogAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var actionFilter = ""
    @State private var events: [AdminAuditEvent] = []
    @State private var loading = true
    @State private var errorMessage: String?

    private var canView: Bool {
        AuditLogAdminLogic.canView(features: shell.platformFeatures, permissions: shell.permissions)
    }

    var body: some View {
        Group {
            if canView { content } else { accessDenied }
        }
        .navigationTitle(L.text("mobile.admin.auditLog.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task { if canView { await load() } }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.auditLog.accessDenied.title"),
            message: L.text("mobile.admin.auditLog.accessDenied.message")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.auditLog.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    LMSCard {
                        VStack(alignment: .leading, spacing: 8) {
                            Text(L.text("mobile.admin.auditLog.filter.label"))
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            TextField(
                                L.text("mobile.admin.auditLog.filter.placeholder"),
                                text: $actionFilter
                            )
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                            .accessibilityLabel(L.text("mobile.admin.auditLog.filter.label"))
                            .onSubmit { Task { await load() } }
                            Button(L.text("mobile.admin.auditLog.filter.apply")) {
                                Task { await load() }
                            }
                            .buttonStyle(.bordered)
                            .frame(minHeight: 44)
                        }
                    }

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && events.isEmpty {
                        LMSSkeletonList(count: 4)
                    } else if events.isEmpty {
                        LMSEmptyState(
                            systemImage: "scroll",
                            title: L.text("mobile.admin.auditLog.empty.title"),
                            message: L.text("mobile.admin.auditLog.empty.message")
                        )
                    } else {
                        LazyVStack(spacing: 10) {
                            ForEach(events) { event in
                                eventCard(event)
                            }
                        }
                    }

                    Button {
                        openURL(AppConfiguration.webURL(path: AuditLogAdminLogic.webPath()))
                    } label: {
                        Label(L.text("mobile.admin.auditLog.openOnWeb"), systemImage: "arrow.up.right.square")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .frame(minHeight: 44)
                }
                .padding(16)
            }
        }
    }

    private func eventCard(_ event: AdminAuditEvent) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 6) {
                Text(formatTimestamp(event.timestamp))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(event.eventType)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(L.format("mobile.admin.auditLog.actor", event.actorId))
                    .font(.caption.monospaced())
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(
                    L.format(
                        "mobile.admin.auditLog.target",
                        AuditLogAdminLogic.targetLabel(type: event.targetType, id: event.targetId)
                    )
                )
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .accessibilityElement(children: .combine)
        }
    }

    private func formatTimestamp(_ raw: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: raw) ?? ISO8601DateFormatter().date(from: raw) {
            return date.formatted(date: .abbreviated, time: .shortened)
        }
        return raw
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            events = try await LMSAPI.fetchAdminAuditLog(
                action: actionFilter,
                accessToken: token
            )
        } catch {
            errorMessage = L.text("mobile.admin.auditLog.error")
            events = []
        }
    }
}
