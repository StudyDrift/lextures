import SwiftUI

/// Forum picker + thread list for a course.
struct DiscussionsListView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    var initialThreadId: String?

    @State private var forums: [DiscussionForum] = []
    @State private var selectedForumId: String?
    @State private var threads: [DiscussionThreadSummary] = []
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var showNewThread = false
    @State private var openThread: DiscussionThreadRoute?
    @State private var consumedInitialThread = false

    private var sortedThreads: [DiscussionThreadSummary] {
        DiscussionLogic.sortThreads(threads)
    }

    private var forumChipSelection: Binding<DiscussionForum> {
        Binding(
            get: {
                guard let selectedForumId,
                      let forum = forums.first(where: { $0.id == selectedForumId }) else {
                    return forums[0]
                }
                return forum
            },
            set: { selectedForumId = $0.id }
        )
    }

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

            if forums.count > 1, let selectedForumId {
                LMSSegmentedChips(
                    options: forums,
                    selection: forumChipSelection,
                    label: \.name
                )
            }

            HStack {
                Spacer()
                Button {
                    showNewThread = true
                } label: {
                    Label(L.text("mobile.discussions.newThread"), systemImage: "plus")
                        .font(.subheadline.weight(.semibold))
                }
                .disabled(selectedForumId == nil || forums.isEmpty)
            }

            if loading && threads.isEmpty {
                LMSSkeletonList(count: 4)
            } else if sortedThreads.isEmpty {
                LMSEmptyState(
                    systemImage: "bubble.left.and.bubble.right",
                    title: L.text("mobile.discussions.emptyTitle"),
                    message: L.text("mobile.discussions.emptyMessage")
                )
            } else {
                ForEach(sortedThreads) { thread in
                    Button {
                        openThread = DiscussionThreadRoute(threadId: thread.id)
                    } label: {
                        threadRow(thread)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .navigationDestination(item: $openThread) { route in
            DiscussionThreadView(course: course, threadId: route.threadId)
        }
        .sheet(isPresented: $showNewThread) {
            if let forumId = selectedForumId {
                PostComposerView(
                    mode: .newThread(forumId: forumId),
                    course: course,
                    onPosted: { threadId in
                        showNewThread = false
                        Task {
                            await loadThreads(force: true)
                            openThread = DiscussionThreadRoute(threadId: threadId)
                        }
                    }
                )
            }
        }
        .task { await loadForums() }
        .onChange(of: selectedForumId) { _, _ in
            Task { await loadThreads() }
        }
        .onChange(of: threads) { _, loaded in
            guard !consumedInitialThread,
                  let threadId = initialThreadId,
                  loaded.contains(where: { $0.id == threadId }) else { return }
            consumedInitialThread = true
            openThread = DiscussionThreadRoute(threadId: threadId)
        }
    }

    @ViewBuilder
    private func threadRow(_ thread: DiscussionThreadSummary) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 6) {
                HStack(alignment: .top) {
                    Text(thread.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .multilineTextAlignment(.leading)
                    Spacer(minLength: 8)
                    if thread.isPinned {
                        Image(systemName: "pin.fill")
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            .accessibilityLabel(L.text("mobile.discussions.pinned"))
                    }
                    if thread.isLocked {
                        Image(systemName: "lock.fill")
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            .accessibilityLabel(L.text("mobile.discussions.locked"))
                    }
                }
                HStack(spacing: 8) {
                    Text(L.format("mobile.discussions.replyCount", thread.replyCount))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text("·")
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text(LMSDates.relative(thread.updatedAt))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    private func loadForums() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.discussionForums(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchDiscussionForums(courseCode: course.courseCode, accessToken: token)
            }
            forums = result.value.sorted { $0.position < $1.position }
            if selectedForumId == nil {
                selectedForumId = forums.first?.id
            }
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
            if !forums.isEmpty {
                await loadThreads()
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.discussions.loadError")
        }
    }

    private func loadThreads(force: Bool = false) async {
        guard let token = session.accessToken, let forumId = selectedForumId else { return }
        if !force && !threads.isEmpty { loading = false }
        loading = true
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.discussionThreads(courseCode: course.courseCode, forumId: forumId),
                accessToken: token
            ) {
                try await LMSAPI.fetchDiscussionThreads(
                    courseCode: course.courseCode,
                    forumId: forumId,
                    accessToken: token
                )
            }
            threads = result.value
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.discussions.loadError")
        }
    }
}