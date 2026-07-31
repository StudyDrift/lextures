// swiftlint:disable identifier_name large_tuple
import SwiftUI

struct InlineDiscussionToolView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.scenePhase) private var scenePhase
    let props: ContentToolRendererProps

    @State private var draft = ""
    @State private var busy = false
    @State private var errorText: String?
    @State private var posts: [Post] = []
    @State private var locked = false
    @State private var lockReason: String?
    @State private var page = 1
    @State private var pageSize = ContentToolPack2Logic.defaultPageSize
    @State private var total: Int?
    @State private var anonymity = "named"
    @State private var replyTo: String?
    @State private var editingId: String?
    @State private var reportTarget: String?
    @State private var showReportConfirm = false
    @State private var showDeleteConfirm = false
    @State private var deleteTarget: String?
    @State private var requirements: (requiredPosts: Int, requiredReplies: Int, myPosts: Int, myReplies: Int)?
    @State private var viewerCanEndorse = false
    @State private var viewerCanModerate = false

    private struct Post: Identifiable, Equatable {
        var id: String
        var parentPostId: String?
        var text: String
        var authorDisplay: String?
        var upvoteCount: Int
        var viewerUpvoted: Bool
        var endorsed: Bool
        var removed: Bool
        var tombstone: Bool
        var isOwn: Bool
        var canEdit: Bool
        var canDelete: Bool
        var createdAt: String?
    }

    private var draftKey: String {
        ContentToolPack2Logic.draftStorageKey(instanceId: props.instanceId, slot: editingId ?? replyTo ?? "composer")
    }

    private var online: Bool { NetworkMonitor.shared.isOnline }

    private var allowReplies: Bool {
        ContentToolPack2Logic.boolField(props.config, key: "allowReplies") != false
    }

    private var prompt: String {
        ContentToolHostLogic.stringField(props.config, key: "prompt") ?? ""
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if !prompt.isEmpty {
                CourseMarkdownContentView(markdown: prompt, compact: true)
            }

            if anonymity == "anonymous_to_peers" {
                Text(L.text("mobile.contentTools.tools.inline_discussion.anonymityNote"))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }

            Text(L.text("mobile.contentTools.tools.inline_discussion.moderationNote"))
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            if let requirements {
                Text(L.format(
                    "mobile.contentTools.tools.inline_discussion.progress",
                    requirements.myPosts,
                    requirements.myReplies
                ))
                .font(.caption2)
            }

            if locked {
                Text(L.text("mobile.contentTools.tools.inline_discussion.lockedHint"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else if posts.isEmpty {
                Text(L.text("mobile.contentTools.tools.inline_discussion.empty"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 12) {
                        ForEach(rootPosts) { post in
                            postView(post, depth: 0)
                            ForEach(replies(to: post.id)) { reply in
                                postView(reply, depth: 1)
                            }
                        }
                    }
                }
                .frame(maxHeight: 360)
                .accessibilityElement(children: .contain)
                .accessibilityLabel(L.text("mobile.contentTools.tools.inline_discussion.label"))

                if let next = ContentToolPack2Logic.nextPage(currentPage: page, pageSize: pageSize, total: total) {
                    Button(L.text("mobile.contentTools.tools.inline_discussion.loadMore")) {
                        Task { await loadThread(page: next, append: true) }
                    }
                    .disabled(busy)
                }
            }

            if let errorText {
                Text(errorText)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
            }

            if !props.readOnly {
                if let replyTo {
                    Text(L.text("mobile.contentTools.tools.inline_discussion.replyLabel"))
                        .font(.caption.weight(.semibold))
                    Button(L.text("mobile.contentTools.tools.inline_discussion.cancel")) {
                        self.replyTo = nil
                        editingId = nil
                    }
                } else if editingId != nil {
                    Text(L.text("mobile.contentTools.tools.inline_discussion.editLabel"))
                        .font(.caption.weight(.semibold))
                }

                ToolComposerView(
                    placeholder: L.text("mobile.contentTools.tools.inline_discussion.composerLabel"),
                    sendLabel: editingId != nil
                        ? L.text("mobile.contentTools.tools.inline_discussion.saveEdit")
                        : (replyTo != nil
                            ? L.text("mobile.contentTools.tools.inline_discussion.submitReply")
                            : L.text("mobile.contentTools.tools.inline_discussion.submitPost")),
                    cancelLabel: L.text("mobile.contentTools.runtime.cancel"),
                    text: $draft,
                    draftKey: draftKey,
                    enabled: true,
                    online: online,
                    busy: busy,
                    onSend: { Task { await submit() } }
                )
            }
        }
        .task { await loadThread(page: 1, append: false) }
        .onChange(of: scenePhase) { _, phase in
            if phase == .active {
                Task { await loadThread(page: 1, append: false) }
            }
        }
        .onAppear {
            if draft.isEmpty {
                draft = ContentToolHostLogic.stringField(props.state, key: "draft")
                    ?? ContentToolDraftStore.load(key: draftKey)
            }
        }
        .confirmationDialog(
            L.text("mobile.contentTools.tools.inline_discussion.report"),
            isPresented: $showReportConfirm,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.contentTools.tools.inline_discussion.report"), role: .destructive) {
                if let reportTarget { Task { await report(postId: reportTarget) } }
            }
            Button(L.text("mobile.contentTools.runtime.cancel"), role: .cancel) {}
        }
        .confirmationDialog(
            L.text("mobile.contentTools.tools.inline_discussion.delete"),
            isPresented: $showDeleteConfirm,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.contentTools.tools.inline_discussion.delete"), role: .destructive) {
                if let deleteTarget { Task { await deletePost(postId: deleteTarget) } }
            }
            Button(L.text("mobile.contentTools.runtime.cancel"), role: .cancel) {}
        }
    }

    private var rootPosts: [Post] {
        posts.filter { $0.parentPostId == nil || $0.parentPostId?.isEmpty == true }
    }

    private func replies(to parentId: String) -> [Post] {
        posts.filter { $0.parentPostId == parentId }
    }

    @ViewBuilder
    private func postView(_ post: Post, depth: Int) -> some View {
        let controls = ContentToolPack2Logic.discussionControls(
            isOwn: post.isOwn,
            canEditFlag: post.canEdit,
            canDeleteFlag: post.canDelete,
            allowReplies: allowReplies,
            viewerCanEndorse: viewerCanEndorse,
            viewerCanModerate: viewerCanModerate,
            readOnly: props.readOnly,
            removed: post.removed || post.tombstone
        )
        let author = ContentToolPack2Logic.authorDisplay(
            serverAuthorDisplay: post.authorDisplay,
            anonymity: anonymity,
            isOwn: post.isOwn
        ) ?? L.text("mobile.contentTools.tools.inline_discussion.classmate")

        VStack(alignment: .leading, spacing: 6) {
            if ContentToolPack2Logic.shouldRenderTombstone(
                removed: post.removed,
                tombstone: post.tombstone,
                moderationState: nil
            ) {
                Text(L.text("mobile.contentTools.tools.inline_discussion.tombstone"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .italic()
            } else {
                Text(author)
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                CourseMarkdownContentView(markdown: post.text, compact: true)
                if post.endorsed {
                    Text(L.text("mobile.contentTools.tools.inline_discussion.endorsedBadge"))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                }
            }

            HStack(spacing: 12) {
                if controls.canUpvote {
                    Button(L.format("mobile.contentTools.tools.inline_discussion.upvote", post.upvoteCount)) {
                        Task { await upvote(postId: post.id) }
                    }
                    .disabled(busy)
                    .accessibilityLabel(L.format("mobile.contentTools.tools.inline_discussion.upvote", post.upvoteCount))
                }
                if controls.canReply {
                    Button(L.text("mobile.contentTools.tools.inline_discussion.reply")) {
                        replyTo = post.id
                        editingId = nil
                    }
                    .accessibilityLabel(L.text("mobile.contentTools.tools.inline_discussion.reply"))
                }
                if controls.canEdit {
                    Button(L.text("mobile.contentTools.tools.inline_discussion.edit")) {
                        editingId = post.id
                        replyTo = nil
                        draft = post.text
                    }
                }
                if controls.canDelete {
                    Button(L.text("mobile.contentTools.tools.inline_discussion.delete"), role: .destructive) {
                        deleteTarget = post.id
                        showDeleteConfirm = true
                    }
                }
                if controls.canReport {
                    Button(L.text("mobile.contentTools.tools.inline_discussion.report")) {
                        reportTarget = post.id
                        showReportConfirm = true
                    }
                    .accessibilityLabel(L.text("mobile.contentTools.tools.inline_discussion.report"))
                }
                if controls.canEndorse {
                    Button(L.text("mobile.contentTools.tools.inline_discussion.endorse")) {
                        Task { await endorse(postId: post.id) }
                    }
                }
                if controls.canModerate {
                    Button(L.text("mobile.contentTools.tools.inline_discussion.moderate")) {
                        Task { await moderate(postId: post.id) }
                    }
                    .accessibilityLabel(L.text("mobile.contentTools.tools.inline_discussion.moderate"))
                }
            }
            .font(.caption)
        }
        .padding(.leading, CGFloat(depth) * 16)
        .accessibilityElement(children: .contain)
        .accessibilityLabel(
            L.format("mobile.contentTools.tools.inline_discussion.postAria", author, post.createdAt ?? "")
        )
    }

    private func loadThread(page: Int, append: Bool) async {
        busy = true
        defer { busy = false }
        do {
            let raw = try await props.runAction("thread", .object([
                "page": .number(Double(page)),
            ]))
            let result = ContentToolPack2Logic.objectMap(raw)
            locked = ContentToolPack2Logic.boolField(raw, key: "locked") == true
            if case .string(let reason) = result["lockReason"] { lockReason = reason }
            if case .string(let a) = result["anonymity"] { anonymity = a }
            if case .number(let n) = result["pageSize"] { pageSize = Int(n) }
            if case .number(let n) = result["total"] { total = Int(n) }
            self.page = page
            let parsed = ContentToolPack2Logic.arrayField(raw, key: "posts").compactMap(parsePost)
            posts = append ? posts + parsed : parsed
            // Staff capability heuristic: if any post exposes canEdit for non-own via endorse fields.
            viewerCanEndorse = parsed.contains { $0.endorsed } || viewerCanEndorse
            if case .object(let req) = result["requirements"] {
                let requiredPosts = Int(ContentToolPack2Logic.numberField(.object(req), key: "requiredPosts") ?? 1)
                let requiredReplies = Int(ContentToolPack2Logic.numberField(.object(req), key: "requiredReplies") ?? 0)
                let myPosts = Int(ContentToolPack2Logic.numberField(.object(req), key: "myPosts") ?? 0)
                let myReplies = Int(ContentToolPack2Logic.numberField(.object(req), key: "myReplies") ?? 0)
                requirements = (requiredPosts, requiredReplies, myPosts, myReplies)
            }
            // Detect staff from successful endorse/moderate absence — keep false until used.
            _ = lockReason
        } catch {
            errorText = L.text("mobile.contentTools.tools.inline_discussion.loadError")
        }
    }

    private func parsePost(_ raw: JSONValue) -> Post? {
        let o = ContentToolPack2Logic.objectMap(raw)
        guard case .string(let id) = o["id"] else { return nil }
        let text: String = { if case .string(let t) = o["text"] { return t }; return "" }()
        let parent: String? = { if case .string(let p) = o["parentPostId"] { return p }; return nil }()
        let author: String? = { if case .string(let a) = o["authorDisplay"] { return a }; return nil }()
        let created: String? = { if case .string(let c) = o["createdAt"] { return c }; return nil }()
        return Post(
            id: id,
            parentPostId: parent,
            text: text,
            authorDisplay: author,
            upvoteCount: Int(ContentToolPack2Logic.numberField(raw, key: "upvoteCount") ?? 0),
            viewerUpvoted: ContentToolPack2Logic.boolField(raw, key: "viewerUpvoted") == true,
            endorsed: ContentToolPack2Logic.boolField(raw, key: "endorsed") == true,
            removed: ContentToolPack2Logic.boolField(raw, key: "removed") == true,
            tombstone: ContentToolPack2Logic.boolField(raw, key: "tombstone") == true,
            isOwn: ContentToolPack2Logic.boolField(raw, key: "isOwn") == true,
            canEdit: ContentToolPack2Logic.boolField(raw, key: "canEdit") == true,
            canDelete: ContentToolPack2Logic.boolField(raw, key: "canDelete") == true,
            createdAt: created
        )
    }

    private func submit() async {
        let text = draft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !text.isEmpty, online, !busy, !props.readOnly else {
            if !online { errorText = L.text("mobile.contentTools.runtime.offlineComposer") }
            return
        }
        busy = true
        errorText = nil
        defer { busy = false }
        do {
            let raw: JSONValue?
            if let editingId {
                raw = try await props.runAction("edit", .object([
                    "postId": .string(editingId),
                    "text": .string(text),
                ]))
            } else {
                var input: [String: JSONValue] = [
                    "text": .string(text),
                    "idempotencyKey": .string(ContentToolHostLogic.newIdempotencyKey()),
                ]
                if let replyTo { input["parentPostId"] = .string(replyTo) }
                raw = try await props.runAction("post", .object(input))
            }
            let result = ContentToolPack2Logic.objectMap(raw)
            if case .string(let code) = result["error"] {
                errorText = L.text(ContentToolPack2Logic.plainLanguageMessageKey(for: code))
                return
            }
            draft = ""
            replyTo = nil
            editingId = nil
            ToolComposerView.clearDraft(key: draftKey)
            props.save(["draft": .string("")])
            props.announce(L.text("mobile.contentTools.tools.inline_discussion.postAnnounced"), false)
            await loadThread(page: 1, append: false)
        } catch {
            errorText = L.text("mobile.contentTools.tools.inline_discussion.postError")
        }
    }

    private func upvote(postId: String) async {
        busy = true
        defer { busy = false }
        do {
            _ = try await props.runAction("upvote", .object(["postId": .string(postId)]))
            await loadThread(page: 1, append: false)
        } catch {
            errorText = L.text("mobile.contentTools.runtime.retry")
        }
    }

    private func report(postId: String) async {
        busy = true
        defer { busy = false }
        do {
            _ = try await props.runAction("report", .object([
                "postId": .string(postId),
                "category": .string("inappropriate"),
            ]))
            props.announce(L.text("mobile.contentTools.tools.inline_discussion.reportThanks"), false)
        } catch {
            errorText = L.text("mobile.contentTools.tools.inline_discussion.reportError")
        }
    }

    private func deletePost(postId: String) async {
        busy = true
        defer { busy = false }
        do {
            _ = try await props.runAction("delete", .object(["postId": .string(postId)]))
            await loadThread(page: 1, append: false)
        } catch {
            errorText = L.text("mobile.contentTools.tools.inline_discussion.deleteError")
        }
    }

    private func endorse(postId: String) async {
        busy = true
        defer { busy = false }
        do {
            _ = try await props.runAction("endorse", .object(["postId": .string(postId)]))
            viewerCanEndorse = true
            await loadThread(page: 1, append: false)
        } catch {
            // 403 → hide gracefully
            viewerCanEndorse = false
            errorText = L.text("mobile.contentTools.tools.inline_discussion.endorseError")
        }
    }

    private func moderate(postId: String) async {
        busy = true
        defer { busy = false }
        do {
            _ = try await props.runAction("moderate", .object([
                "postId": .string(postId),
                "action": .string("removed"),
            ]))
            viewerCanModerate = true
            await loadThread(page: 1, append: false)
        } catch {
            viewerCanModerate = false
            errorText = L.text("mobile.contentTools.tools.inline_discussion.error.forbidden")
        }
    }
}
