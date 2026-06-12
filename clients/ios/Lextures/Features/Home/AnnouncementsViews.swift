import SwiftUI

/// Dashboard banner for the newest org announcement; coral treatment for emergencies.
struct AnnouncementCard: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let broadcast: Broadcast
    var showSeeAll = false
    var onAcknowledged: () -> Void = {}

    @State private var acknowledging = false

    var body: some View {
        LMSCard(accent: broadcast.isEmergency ? LexturesTheme.coral : LexturesTheme.amber) {
            HStack(alignment: .top, spacing: 12) {
                Image(systemName: broadcast.isEmergency ? "exclamationmark.bubble.fill" : "megaphone.fill")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(broadcast.isEmergency ? LexturesTheme.coral : LexturesTheme.amber)
                    .frame(width: 32, height: 32)
                    .background((broadcast.isEmergency ? LexturesTheme.coral : LexturesTheme.amber).opacity(0.13))
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))

                VStack(alignment: .leading, spacing: 4) {
                    HStack(alignment: .top) {
                        Text(broadcast.subject)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Spacer(minLength: 8)
                        Text(LMSDates.relative(broadcast.sentAt ?? broadcast.createdAt))
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Text(broadcast.body)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .lineLimit(3)
                }
            }

            HStack {
                Button {
                    Task { await acknowledge() }
                } label: {
                    if acknowledging {
                        ProgressView()
                            .controlSize(.small)
                    } else {
                        Text("Got it")
                            .font(.caption.weight(.semibold))
                    }
                }
                .buttonStyle(.bordered)
                .tint(broadcast.isEmergency ? LexturesTheme.coral : LexturesTheme.primary)

                Spacer()

                if showSeeAll {
                    NavigationLink(value: BroadcastsListRoute()) {
                        Text("See all")
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private func acknowledge() async {
        guard let token = session.accessToken else { return }
        acknowledging = true
        defer { acknowledging = false }
        // Best-effort: dismiss locally even if the POST fails; the next refresh re-syncs.
        try? await LMSAPI.acknowledgeBroadcast(id: broadcast.id, accessToken: token)
        onAcknowledged()
    }
}

/// Full announcement history ("See all" from the dashboard banner).
struct AnnouncementsListView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    @State private var broadcasts: [Broadcast] = []
    @State private var errorMessage: String?
    @State private var loading = true

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && broadcasts.isEmpty {
                        LMSSkeletonList(count: 4)
                    } else if broadcasts.isEmpty {
                        LMSEmptyState(
                            systemImage: "megaphone",
                            title: "No announcements",
                            message: "School-wide announcements will appear here."
                        )
                    } else {
                        ForEach(broadcasts) { broadcast in
                            LMSCard(accent: broadcast.isEmergency ? LexturesTheme.coral : nil) {
                                HStack(alignment: .top) {
                                    Text(broadcast.subject)
                                        .font(.subheadline.weight(.semibold))
                                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                    Spacer(minLength: 8)
                                    Text(LMSDates.relative(broadcast.sentAt ?? broadcast.createdAt))
                                        .font(.caption2)
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                }
                                Text(broadcast.body)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                    }
                }
                .padding(16)
            }
            .refreshable { await load() }
        }
        .navigationTitle("Announcements")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            broadcasts = try await LMSAPI.fetchMyBroadcasts(accessToken: token)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load announcements."
        }
    }
}
