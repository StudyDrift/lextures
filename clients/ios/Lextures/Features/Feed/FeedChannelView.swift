import PhotosUI
import SwiftUI

/// Chat-style channel view: message list with pinned banner, likes, staff pin/delete, and a
/// composer that supports text + a single image attachment (M7.6).
struct FeedChannelView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let channelId: String
    let channelName: String
    var groupContext: GroupFeedContext?

    @State private var roots: [FeedMessage] = []
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var composerText = ""
    @State private var photoPickerItem: PhotosPickerItem?
    @State private var pendingImageData: Data?
    @State private var uploading = false
    @State private var pendingLikes: Set<String> = []
    @State private var socket = FeedSocket()

    private var viewerId: String? {
        NotebookStore.jwtSubject(from: session.accessToken)
    }

    private var orderedMessages: [FeedMessage] {
        FeedLogic.orderedMessages(roots)
    }

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: 10) {
                    if !NetworkMonitor.shared.isOnline {
                        OfflineBanner()
                    }
                    if let cacheLabel {
                        StalenessChip(label: cacheLabel)
                    }
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && roots.isEmpty {
                        LMSSkeletonList(count: 4)
                    } else if orderedMessages.isEmpty {
                        LMSEmptyState(
                            systemImage: "text.bubble",
                            title: L.text("mobile.feed.emptyMessages"),
                            message: ""
                        )
                    } else {
                        ForEach(orderedMessages) { message in
                            messageBubble(message)
                        }
                    }
                }
                .padding(16)
            }
            .refreshable { await load(force: true) }

            composer
        }
        .navigationTitle(channelName)
        .navigationBarTitleDisplayMode(.inline)
        .task {
            socket.connect(courseCode: course.courseCode, accessToken: { session.accessToken })
            await load()
        }
        .onDisappear { socket.disconnect() }
        .onChange(of: socket.revision(forChannel: channelId)) { _, _ in
            Task { await load(force: true) }
        }
        .onChange(of: photoPickerItem) { _, item in
            Task { await loadPhotoPickerItem(item) }
        }
    }

    @ViewBuilder
    private func messageBubble(_ message: FeedMessage) -> some View {
        let extracted = FeedLogic.extractImagePath(from: message.body)
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text(message.authorLabel)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if message.pinnedAt != nil {
                        Image(systemName: "pin.fill")
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.amber)
                            .accessibilityLabel(L.text("mobile.feed.pinned"))
                    }
                    Spacer()
                    Text(LMSDates.relative(message.createdAt))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                if !extracted.text.isEmpty {
                    Text(extracted.text)
                        .font(.body)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .textSelection(.enabled)
                }
                if let imagePath = extracted.imagePath {
                    FeedImageView(courseCode: course.courseCode, path: imagePath)
                }
                HStack(spacing: 14) {
                    Button {
                        Task { await toggleLike(message) }
                    } label: {
                        Label(
                            "\(message.likeCount)",
                            systemImage: message.viewerHasLiked ? "hand.thumbsup.fill" : "hand.thumbsup"
                        )
                        .font(.caption.weight(.semibold))
                    }
                    .disabled(pendingLikes.contains(message.id))
                    .accessibilityLabel(L.text("mobile.feed.like"))

                    if FeedLogic.canPin(viewerIsStaff: course.viewerIsStaff, isReply: message.parentMessageId != nil) {
                        Button(message.pinnedAt != nil ? L.text("mobile.feed.unpin") : L.text("mobile.feed.pin")) {
                            Task { await togglePin(message) }
                        }
                        .font(.caption.weight(.semibold))
                    }

                    if FeedLogic.canDelete(message, viewerId: viewerId) {
                        Button(L.text("mobile.feed.delete"), role: .destructive) {
                            Task { await deleteMessage(message) }
                        }
                        .font(.caption.weight(.semibold))
                    }
                }
            }
        }
        .padding(.leading, message.parentMessageId != nil ? 20 : 0)
    }

    private var composer: some View {
        VStack(alignment: .leading, spacing: 6) {
            if pendingImageData != nil {
                HStack {
                    Label(L.text("mobile.feed.attachImage"), systemImage: "photo")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Spacer()
                    Button {
                        pendingImageData = nil
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                    }
                }
            }
            HStack(spacing: 10) {
                PhotosPicker(selection: $photoPickerItem, matching: .images) {
                    Image(systemName: "photo.on.rectangle")
                }
                TextField(L.text("mobile.feed.composerPlaceholder"), text: $composerText, axis: .vertical)
                    .textFieldStyle(.roundedBorder)
                Button {
                    Task { await sendComposed() }
                } label: {
                    Image(systemName: "paperplane.fill")
                }
                .disabled(uploading || (composerText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    && pendingImageData == nil))
            }
        }
        .padding(12)
        .background(LexturesTheme.sceneBackground(for: colorScheme))
    }

    private func load(force: Bool = false) async {
        guard let token = session.accessToken else { return }
        if !force && !roots.isEmpty { loading = false }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let cacheKey: String
            let fetchMessages: () async throws -> [FeedMessage]
            if let groupContext {
                cacheKey = OfflineCacheKey.groupFeedMessages(
                    courseCode: course.courseCode,
                    groupId: groupContext.groupId,
                    channelId: channelId
                )
                fetchMessages = {
                    try await LMSAPI.fetchGroupFeedMessages(
                        courseCode: course.courseCode,
                        groupId: groupContext.groupId,
                        channelId: channelId,
                        accessToken: token
                    )
                }
            } else {
                cacheKey = OfflineCacheKey.feedMessages(courseCode: course.courseCode, channelId: channelId)
                fetchMessages = {
                    try await LMSAPI.fetchFeedMessages(
                        courseCode: course.courseCode,
                        channelId: channelId,
                        accessToken: token
                    )
                }
            }
            let result = try await offline.cachedFetch(key: cacheKey, accessToken: token, fetch: fetchMessages)
            roots = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.feed.loadError")
        }
    }

    private func loadPhotoPickerItem(_ item: PhotosPickerItem?) async {
        guard let item, let data = try? await item.loadTransferable(type: Data.self) else { return }
        await MainActor.run {
            pendingImageData = data
            photoPickerItem = nil
        }
    }

    private func sendComposed() async {
        guard let token = session.accessToken else { return }
        let text = composerText.trimmingCharacters(in: .whitespacesAndNewlines)
        let imageData = pendingImageData
        guard !text.isEmpty || imageData != nil else { return }
        composerText = ""
        pendingImageData = nil
        errorMessage = nil

        var body = text
        do {
            if let imageData {
                guard NetworkMonitor.shared.isOnline else {
                    throw APIError.transport(URLError(.notConnectedToInternet))
                }
                uploading = true
                let upload = try await LMSAPI.uploadFeedImage(
                    courseCode: course.courseCode,
                    imageData: imageData,
                    fileName: "photo.jpg",
                    mimeType: "image/jpeg",
                    accessToken: token
                )
                uploading = false
                let markdown = "![image](\(upload.contentPath))"
                body = body.isEmpty ? markdown : "\(body)\n\n\(markdown)"
            }

            if NetworkMonitor.shared.isOnline {
                if let groupContext {
                    _ = try await LMSAPI.postGroupFeedMessage(
                        courseCode: course.courseCode,
                        groupId: groupContext.groupId,
                        channelId: channelId,
                        body: body,
                        accessToken: token
                    )
                } else {
                    _ = try await LMSAPI.postFeedMessage(
                        courseCode: course.courseCode,
                        channelId: channelId,
                        body: body,
                        accessToken: token
                    )
                }
                await load(force: true)
            } else {
                let request = PostFeedMessageRequest(
                    body: body,
                    parentMessageId: nil,
                    mentionUserIds: [],
                    mentionsEveryone: false
                )
                let path: String
                if let groupContext {
                    path = "/api/v1/courses/\(course.courseCode)/groups/\(groupContext.groupId)"
                        + "/feed/channels/\(channelId)/messages"
                } else {
                    path = "/api/v1/courses/\(course.courseCode)/feed/channels/\(channelId)/messages"
                }
                _ = try await offline.enqueueMutation(
                    method: "POST",
                    path: path,
                    body: request,
                    label: groupContext == nil
                        ? L.text("mobile.feed.title")
                        : L.text("mobile.groups.discussionTitle"),
                    accessToken: token,
                    preferQueue: true
                )
            }
        } catch {
            uploading = false
            composerText = text
            pendingImageData = imageData
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.feed.sendFailed")
        }
    }

    private func toggleLike(_ message: FeedMessage) async {
        guard let token = session.accessToken else { return }
        pendingLikes.insert(message.id)
        defer { pendingLikes.remove(message.id) }
        let optimistic = !message.viewerHasLiked
        updateMessage(message.id) {
            $0.viewerHasLiked = optimistic
            $0.likeCount = max(0, $0.likeCount + (optimistic ? 1 : -1))
        }
        do {
            if optimistic {
                try await LMSAPI.likeFeedMessage(courseCode: course.courseCode, messageId: message.id, accessToken: token)
            } else {
                try await LMSAPI.unlikeFeedMessage(courseCode: course.courseCode, messageId: message.id, accessToken: token)
            }
        } catch {
            updateMessage(message.id) {
                $0.viewerHasLiked = message.viewerHasLiked
                $0.likeCount = message.likeCount
            }
            errorMessage = (error as? LocalizedError)?.errorDescription
        }
    }

    private func togglePin(_ message: FeedMessage) async {
        guard let token = session.accessToken else { return }
        let pin = message.pinnedAt == nil
        do {
            try await LMSAPI.pinFeedMessage(
                courseCode: course.courseCode,
                messageId: message.id,
                pinned: pin,
                accessToken: token
            )
            await load(force: true)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
        }
    }

    private func deleteMessage(_ message: FeedMessage) async {
        guard let token = session.accessToken else { return }
        do {
            try await LMSAPI.deleteFeedMessage(courseCode: course.courseCode, messageId: message.id, accessToken: token)
            await load(force: true)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
        }
    }

    private func updateMessage(_ id: String, _ mutate: (inout FeedMessage) -> Void) {
        for index in roots.indices {
            if roots[index].id == id {
                mutate(&roots[index])
                return
            }
            for replyIndex in roots[index].replies.indices where roots[index].replies[replyIndex].id == id {
                mutate(&roots[index].replies[replyIndex])
                return
            }
        }
    }
}

/// Authenticated image bubble for feed attachments (server content paths require a bearer token).
private struct FeedImageView: View {
    @Environment(AuthSession.self) private var session

    let courseCode: String
    let path: String

    @State private var uiImage: UIImage?

    var body: some View {
        Group {
            if let uiImage {
                Image(uiImage: uiImage)
                    .resizable()
                    .scaledToFit()
                    .frame(maxHeight: 220)
                    .clipShape(RoundedRectangle(cornerRadius: 10))
            } else {
                RoundedRectangle(cornerRadius: 10)
                    .fill(.secondary.opacity(0.15))
                    .frame(height: 120)
                    .overlay(ProgressView())
            }
        }
        .task { await load() }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        let url = AppConfiguration.apiURL(path: path)
        let request = FileDownloadManager.authorizedRequest(url: url, accessToken: token)
        guard let (data, _) = try? await URLSession.shared.data(for: request) else { return }
        uiImage = UIImage(data: data)
    }
}
