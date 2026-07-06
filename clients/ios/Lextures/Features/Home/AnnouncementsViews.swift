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
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var broadcasts: [Broadcast] = []
    @State private var permissions: [String] = []
    @State private var courses: [CourseSummary] = []
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var showBroadcastComposer = false

    private var canComposeBroadcast: Bool {
        AnnouncementLogic.canComposeBroadcast(permissions: permissions, features: shell.platformFeatures)
    }

    private var broadcastOrgId: String? {
        AnnouncementLogic.resolveOrgId(courses: courses)
    }

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
                            title: L.text("mobile.announcements.empty.title"),
                            message: L.text("mobile.announcements.empty.message")
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
        .navigationTitle(L.text("mobile.announcements.listTitle"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            if canComposeBroadcast, broadcastOrgId != nil {
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        showBroadcastComposer = true
                    } label: {
                        Image(systemName: "square.and.pencil")
                    }
                    .accessibilityLabel(L.text("mobile.broadcast.compose.navTitle"))
                }
            }
        }
        .sheet(isPresented: $showBroadcastComposer) {
            if let orgId = broadcastOrgId {
                BroadcastComposerView(orgId: orgId) { created in
                    broadcasts.insert(created, at: 0)
                }
            }
        }
        .task { await load() }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            async let broadcastsTask = LMSAPI.fetchMyBroadcasts(accessToken: token)
            async let permissionsTask = try? LMSAPI.fetchMyPermissions(accessToken: token)
            async let coursesTask = try? LMSAPI.fetchCourses(accessToken: token)
            broadcasts = try await broadcastsTask
            permissions = await permissionsTask ?? permissions
            courses = await coursesTask ?? courses
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.announcements.loadError")
        }
    }
}
