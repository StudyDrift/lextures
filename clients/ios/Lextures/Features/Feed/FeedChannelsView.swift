import SwiftUI

/// Course feed channel list (M7.6).
struct FeedChannelsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var channels: [FeedChannel] = []
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var showNewChannel = false
    @State private var newChannelName = ""
    @State private var creating = false
    @State private var openChannel: FeedChannelRoute?
    @State private var socket = FeedSocket()

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if !NetworkMonitor.shared.isOnline {
                OfflineBanner()
            }
            if let cacheLabel {
                StalenessChip(label: cacheLabel)
            }
            if let errorMessage {
                LMSErrorBanner(message: errorMessage)
            }

            if course.viewerIsStaff {
                HStack {
                    Spacer()
                    Button {
                        showNewChannel = true
                    } label: {
                        Label(L.text("mobile.feed.newChannel"), systemImage: "plus")
                            .font(.subheadline.weight(.semibold))
                    }
                }
            }

            if loading && channels.isEmpty {
                LMSSkeletonList(count: 3)
            } else if channels.isEmpty {
                LMSEmptyState(
                    systemImage: "text.bubble",
                    title: L.text("mobile.feed.emptyChannels"),
                    message: ""
                )
            } else {
                ForEach(channels.sorted { $0.sortOrder < $1.sortOrder }) { channel in
                    Button {
                        openChannel = FeedChannelRoute(channelId: channel.id, channelName: channel.name)
                    } label: {
                        channelRow(channel)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .navigationDestination(item: $openChannel) { route in
            FeedChannelView(course: course, channelId: route.channelId, channelName: route.channelName)
        }
        .alert(L.text("mobile.feed.newChannel"), isPresented: $showNewChannel) {
            TextField(L.text("mobile.feed.channelNamePlaceholder"), text: $newChannelName)
            Button(L.text("mobile.feed.create")) {
                Task { await createChannel() }
            }
            Button(L.text("mobile.common.close"), role: .cancel) {
                newChannelName = ""
            }
        }
        .task {
            socket.connect(courseCode: course.courseCode, accessToken: { session.accessToken })
            await load()
        }
        .onDisappear { socket.disconnect() }
        .onChange(of: socket.channelsRevision) { _, _ in
            Task { await load(force: true) }
        }
    }

    @ViewBuilder
    private func channelRow(_ channel: FeedChannel) -> some View {
        LMSCard {
            HStack {
                Image(systemName: "number")
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(channel.name)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func load(force: Bool = false) async {
        guard let token = session.accessToken else { return }
        if !force && !channels.isEmpty { loading = false }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.feedChannels(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchFeedChannels(courseCode: course.courseCode, accessToken: token)
            }
            channels = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.feed.loadError")
        }
    }

    private func createChannel() async {
        guard let token = session.accessToken else { return }
        let name = newChannelName.trimmingCharacters(in: .whitespacesAndNewlines)
        newChannelName = ""
        guard !name.isEmpty, !creating else { return }
        creating = true
        defer { creating = false }
        do {
            _ = try await LMSAPI.createFeedChannel(courseCode: course.courseCode, name: name, accessToken: token)
            await load(force: true)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
        }
    }
}

struct FeedChannelRoute: Hashable {
    var channelId: String
    var channelName: String
}
