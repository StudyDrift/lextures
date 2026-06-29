import SwiftUI

struct DeviceSessionsRoute: Hashable {}

/// Lists active sign-in sessions and allows revoking others.
struct DeviceSessionsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    @State private var sessions: [SessionsAPI.ActiveSession] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var confirmingRevokeOthers = false
    @State private var revokingSessionId: String?

    private var otherSessions: [SessionsAPI.ActiveSession] {
        sessions.filter { !$0.isCurrent }
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.sessions.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && sessions.isEmpty {
                        LMSSkeletonList(count: 3)
                    } else if sessions.isEmpty {
                        LMSEmptyState(
                            systemImage: "desktopcomputer",
                            title: L.text("mobile.sessions.emptyTitle"),
                            message: L.text("mobile.sessions.emptyMessage")
                        )
                    } else {
                        ForEach(sessions) { row in
                            sessionRow(row)
                        }

                        if !otherSessions.isEmpty {
                            Button {
                                confirmingRevokeOthers = true
                            } label: {
                                Label(L.text("mobile.sessions.signOutOthers"), systemImage: "rectangle.portrait.and.arrow.right")
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(LexturesTheme.error)
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 14)
                                    .background(LexturesTheme.error.opacity(0.09))
                                    .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
                .padding(16)
            }
        }
        .navigationTitle(L.text("mobile.sessions.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await loadSessions() }
        .task { await loadSessions() }
        .confirmationDialog(
            L.text("mobile.sessions.signOutOthersConfirm"),
            isPresented: $confirmingRevokeOthers,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.sessions.signOutOthers"), role: .destructive) {
                Task { await revokeOtherSessions() }
            }
        }
    }

    private func sessionRow(_ row: SessionsAPI.ActiveSession) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text(row.deviceLabel)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Spacer(minLength: 8)
                    if row.isCurrent {
                        Text(L.text("mobile.sessions.currentDevice"))
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                            .padding(.horizontal, 8)
                            .padding(.vertical, 3)
                            .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.18 : 0.14))
                            .clipShape(Capsule())
                    }
                }
                .accessibilityElement(children: .combine)
                .accessibilityLabel(sessionAccessibilityLabel(row))

                sessionMeta(label: L.text("mobile.sessions.lastActive"), value: DateFormatting.formatAbsoluteShort(row.lastUsedAt))
                sessionMeta(label: L.text("mobile.sessions.location"), value: row.location)
                sessionMeta(label: L.text("mobile.sessions.authMethod"), value: row.authMethod)

                if !row.isCurrent {
                    Button {
                        Task { await revokeSession(id: row.id) }
                    } label: {
                        Group {
                            if revokingSessionId == row.id {
                                ProgressView()
                            } else {
                                Text(L.text("mobile.sessions.signOutDevice"))
                            }
                        }
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.error)
                    }
                    .buttonStyle(.plain)
                    .disabled(revokingSessionId != nil)
                }
            }
        }
    }

    private func sessionMeta(label: String, value: String) -> some View {
        HStack(alignment: .top, spacing: 8) {
            Text(label)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .frame(width: 88, alignment: .leading)
            Text(value)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Spacer(minLength: 0)
        }
    }

    private func sessionAccessibilityLabel(_ row: SessionsAPI.ActiveSession) -> String {
        let current = row.isCurrent ? L.text("mobile.sessions.currentDevice") : ""
        return "\(row.deviceLabel). \(current) \(L.text("mobile.sessions.lastActive")) \(DateFormatting.formatAbsoluteShort(row.lastUsedAt))"
    }
    @MainActor
    private func loadSessions() async {
        errorMessage = nil
        loading = true
        defer { loading = false }
        guard let token = session.accessToken else { return }
        do {
            sessions = try await SessionsAPI.fetchSessions(accessToken: token)
        } catch {
            errorMessage = L.text("mobile.sessions.loadError")
        }
    }

    @MainActor
    private func revokeSession(id: String) async {
        guard let token = session.accessToken else { return }
        revokingSessionId = id
        defer { revokingSessionId = nil }
        do {
            try await SessionsAPI.revokeSession(id: id, accessToken: token)
            sessions.removeAll { $0.id == id }
        } catch {
            errorMessage = L.text("mobile.sessions.revokeError")
        }
    }

    @MainActor
    private func revokeOtherSessions() async {
        guard let token = session.accessToken else { return }
        do {
            try await SessionsAPI.revokeOtherSessions(accessToken: token)
            sessions = sessions.filter(\.isCurrent)
        } catch {
            errorMessage = L.text("mobile.sessions.revokeOthersError")
        }
    }
}
