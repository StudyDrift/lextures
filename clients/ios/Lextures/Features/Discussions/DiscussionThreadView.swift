import SwiftUI

/// Thread detail with nested replies, upvotes, and compose actions.
struct DiscussionThreadView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let threadId: String

    @State private var thread: DiscussionThreadDetail?
    @State private var posts: [DiscussionPost] = []
    @State private var hiddenUntilFirstPost = false
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var composerMode: PostComposerMode?
    @State private var pendingUpvotes: Set<String> = []

    private var viewerId: String? {
        NotebookStore.jwtSubject(from: session.accessToken)
    }

    private var nestedPosts: [DiscussionNestedPost] {
        DiscussionLogic.nestPosts(posts)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    if !NetworkMonitor.shared.isOnline {
                        OfflineBanner()
                    }
                    if let cacheLabel {
                        StalenessChip(label: cacheLabel)
                    }
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && thread == nil {
                        LMSSkeletonList(count: 3)
                    } else if let thread {
                        threadHeader(thread)
                        if hiddenUntilFirstPost {
                            LMSCard {
                                Text(L.text("mobile.discussions.postFirstHint"))
                                    .font(.subheadline)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        } else {
                            ForEach(nestedPosts) { nested in
                                postCard(nested)
                            }
                        }
                    }
                }
                .padding(16)
            }
            .refreshable { await load(force: true) }
        }
        .navigationTitle(thread?.title ?? L.text("mobile.discussions.thread"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            if let thread, DiscussionLogic.canReply(thread: thread, viewerIsStaff: course.viewerIsStaff) {
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        composerMode = .reply(parentPostId: nil)
                    } label: {
                        Image(systemName: "square.and.pencil")
                    }
                    .accessibilityLabel(L.text("mobile.discussions.reply"))
                }
            }
        }
        .sheet(item: $composerMode) { mode in
            PostComposerView(
                mode: mode,
                course: course,
                threadId: threadId,
                onPosted: { _ in
                    composerMode = nil
                    Task { await load(force: true) }
                }
            )
        }
        .task { await load() }
    }

    @ViewBuilder
    private func threadHeader(_ thread: DiscussionThreadDetail) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text(DiscussionLogic.authorLabel(authorId: thread.authorId, viewerId: viewerId))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Spacer()
                    Text(LMSDates.shortDateTime(thread.createdAt))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                if !thread.bodyPlainText.isEmpty {
                    discussionReaderToolbar(
                        text: thread.bodyPlainText,
                        contentId: thread.id,
                        contentType: "discussion_post"
                    )
                    Text(thread.bodyPlainText)
                        .font(.body)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .textSelection(.enabled)
                        .lexturesReadableText()
                }
            }
        }
    }

    @ViewBuilder
    private func postCard(_ nested: DiscussionNestedPost) -> some View {
        let post = nested.post
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text(DiscussionLogic.authorLabel(authorId: post.authorId, viewerId: viewerId))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Spacer()
                    Text(LMSDates.shortDateTime(post.createdAt))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                discussionReaderToolbar(
                    text: post.bodyPlainText,
                    contentId: post.id,
                    contentType: "discussion_post"
                )
                Text(post.bodyPlainText)
                    .font(.body)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .textSelection(.enabled)
                    .lexturesReadableText()
                    .padding(.leading, CGFloat(min(nested.depth, 6)) * 12)

                HStack(spacing: 12) {
                    Button {
                        Task { await toggleUpvote(post) }
                    } label: {
                        Label(
                            "\(post.upvoteCount)",
                            systemImage: post.viewerUpvoted ? "hand.thumbsup.fill" : "hand.thumbsup"
                        )
                        .font(.caption.weight(.semibold))
                    }
                    .disabled(pendingUpvotes.contains(post.id))
                    .accessibilityLabel(L.text("mobile.discussions.upvote"))

                    if let thread, DiscussionLogic.canReply(thread: thread, viewerIsStaff: course.viewerIsStaff) {
                        Button(L.text("mobile.discussions.reply")) {
                            composerMode = .reply(parentPostId: post.id)
                        }
                        .font(.caption.weight(.semibold))
                    }

                    if DiscussionLogic.canDeletePost(post: post, viewerId: viewerId) {
                        Button(L.text("mobile.discussions.delete"), role: .destructive) {
                            Task { await deletePost(post) }
                        }
                        .font(.caption.weight(.semibold))
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func discussionReaderToolbar(text: String, contentId: String, contentType: String) -> some View {
        let caps = shell.platformFeatures.immersiveReader
        if caps.toolbarEnabled, !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            let targetLang = Locale.current.language.languageCode?.identifier ?? "en"
            ReaderToolbar(
                text: text,
                ugcTranslation: .ugc(
                    contentType: contentType,
                    contentId: contentId,
                    text: text,
                    targetLang: targetLang
                ),
                readAloudEnabled: caps.readAloudEnabled,
                translationEnabled: caps.translationEnabled,
                preferencesEnabled: caps.preferencesEnabled
            )
        }
    }

    private func load(force: Bool = false) async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            async let threadTask = offline.cachedFetch(
                key: OfflineCacheKey.discussionThread(courseCode: course.courseCode, threadId: threadId),
                accessToken: token
            ) {
                try await LMSAPI.fetchDiscussionThread(
                    courseCode: course.courseCode,
                    threadId: threadId,
                    accessToken: token
                )
            }
            async let postsTask = offline.cachedFetch(
                key: OfflineCacheKey.discussionPosts(courseCode: course.courseCode, threadId: threadId),
                accessToken: token
            ) {
                try await LMSAPI.fetchDiscussionPosts(
                    courseCode: course.courseCode,
                    threadId: threadId,
                    accessToken: token
                )
            }
            let threadResult = try await threadTask
            let postsResult = try await postsTask
            thread = threadResult.value
            posts = postsResult.value.posts ?? []
            hiddenUntilFirstPost = postsResult.value.hiddenUntilFirstPost ?? false
            if let cached = threadResult.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else if let cached = postsResult.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.discussions.loadError")
        }
    }

    private func toggleUpvote(_ post: DiscussionPost) async {
        guard let token = session.accessToken else { return }
        pendingUpvotes.insert(post.id)
        defer { pendingUpvotes.remove(post.id) }

        let optimistic = !post.viewerUpvoted
        let delta = optimistic ? 1 : -1
        posts = posts.map {
            guard $0.id == post.id else { return $0 }
            var copy = $0
            copy.viewerUpvoted = optimistic
            copy.upvoteCount = max(0, copy.upvoteCount + delta)
            return copy
        }

        do {
            let response = try await LMSAPI.upvoteDiscussionPost(
                courseCode: course.courseCode,
                postId: post.id,
                accessToken: token
            )
            posts = posts.map {
                guard $0.id == post.id else { return $0 }
                var copy = $0
                copy.viewerUpvoted = response.wasAdded
                copy.upvoteCount = response.upvoteCount
                return copy
            }
        } catch {
            posts = posts.map {
                guard $0.id == post.id else { return $0 }
                var copy = $0
                copy.viewerUpvoted = post.viewerUpvoted
                copy.upvoteCount = post.upvoteCount
                return copy
            }
            errorMessage = (error as? LocalizedError)?.errorDescription
        }
    }

    private func deletePost(_ post: DiscussionPost) async {
        guard let token = session.accessToken else { return }
        do {
            try await LMSAPI.deleteDiscussionPost(
                courseCode: course.courseCode,
                postId: post.id,
                accessToken: token
            )
            posts.removeAll { $0.id == post.id }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
        }
    }
}

enum PostComposerMode: Identifiable {
    case newThread(forumId: String)
    case reply(parentPostId: String?)

    var id: String {
        switch self {
        case let .newThread(forumId): return "new-\(forumId)"
        case let .reply(parentPostId): return "reply-\(parentPostId ?? "root")"
        }
    }
}
