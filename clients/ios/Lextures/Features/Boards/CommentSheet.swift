import SwiftUI

/// Nested comment thread for a board card (VC.M5).
struct CommentSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let courseCode: String
    let boardId: String
    let postId: String
    var canInteract: Bool
    var canManageBoard: Bool
    var currentUserId: String?
    var onCountChange: (Int) -> Void

    @State private var comments: [BoardComment] = []
    @State private var loading = true
    @State private var draft = ""
    @State private var replyTo: String?
    @State private var busy = false
    @State private var errorMessage: String?
    @State private var reportCommentId: String?

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let errorMessage, comments.isEmpty {
                    VStack(spacing: 12) {
                        LMSErrorBanner(message: errorMessage)
                        Button(L.text("mobile.common.retry")) {
                            Task { await load() }
                        }
                    }
                    .padding(16)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    ScrollView {
                        LazyVStack(alignment: .leading, spacing: 12) {
                            Text(L.text("mobile.boards.comment.threadHeading"))
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                .textCase(.uppercase)
                                .accessibilityAddTraits(.isHeader)

                            let nested = BoardsLogic.nestComments(
                                BoardsLogic.visibleComments(comments, canManageBoard: canManageBoard)
                            )
                            if nested.isEmpty {
                                Text(L.text("mobile.boards.comment.empty"))
                                    .font(.subheadline)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            } else {
                                ForEach(nested, id: \.comment.id) { row in
                                    commentBlock(row.comment)
                                    ForEach(row.children) { child in
                                        commentBlock(child)
                                            .padding(.leading, 16)
                                    }
                                }
                            }
                        }
                        .padding(16)
                        .accessibilityElement(children: .contain)
                        .accessibilityLabel(L.text("mobile.boards.comment.threadAria"))
                    }

                    if canInteract {
                        composer
                    }
                }
            }
            .background(LexturesTheme.sceneBackground(for: colorScheme))
            .navigationTitle(L.text("mobile.boards.comment.toggle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) { dismiss() }
                }
            }
            .task { await load() }
            .sheet(isPresented: Binding(
                get: { reportCommentId != nil },
                set: { if !$0 { reportCommentId = nil } }
            )) {
                if let reportCommentId {
                    ReportDialog(
                        courseCode: courseCode,
                        boardId: boardId,
                        commentId: reportCommentId
                    )
                    .presentationDetents([.medium])
                }
            }
        }
    }

    private var composer: some View {
        VStack(alignment: .leading, spacing: 8) {
            if replyTo != nil {
                HStack {
                    Text(L.text("mobile.boards.comment.replying"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Button(L.text("mobile.boards.comment.cancelReply")) {
                        replyTo = nil
                    }
                    .font(.caption.weight(.semibold))
                }
            }
            if let errorMessage {
                Text(errorMessage)
                    .font(.caption)
                    .foregroundStyle(.red)
            }
            HStack(alignment: .bottom, spacing: 8) {
                TextField(L.text("mobile.boards.comment.placeholder"), text: $draft, axis: .vertical)
                    .lineLimit(2 ... 5)
                    .textFieldStyle(.roundedBorder)
                    .accessibilityLabel(L.text("mobile.boards.comment.add"))
                Button(L.text("mobile.boards.comment.submit")) {
                    Task { await submit() }
                }
                .font(.subheadline.weight(.semibold))
                .disabled(busy || draft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
        .padding(12)
        .background(.ultraThinMaterial)
    }

    @ViewBuilder
    private func commentBlock(_ comment: BoardComment) -> some View {
        let isAuthor =
            currentUserId != nil
            && comment.authorId != nil
            && currentUserId!.caseInsensitiveCompare(comment.authorId!) == .orderedSame

        if comment.hidden && canManageBoard {
            Text(L.text("mobile.boards.comment.hiddenPlaceholder"))
                .font(.caption)
                .italic()
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .padding(10)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(Color.secondary.opacity(0.08), in: RoundedRectangle(cornerRadius: 8))
        } else {
            VStack(alignment: .leading, spacing: 6) {
                Text(BoardsLogic.commentPlainText(comment))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .frame(maxWidth: .infinity, alignment: .leading)
                HStack(spacing: 12) {
                    if canInteract {
                        Button(L.text("mobile.boards.comment.reply")) {
                            replyTo = comment.id
                        }
                        .font(.caption.weight(.semibold))
                    }
                    Button(L.text("mobile.boards.report.action")) {
                        reportCommentId = comment.id
                    }
                    .font(.caption.weight(.semibold))
                    if isAuthor {
                        Button(L.text("mobile.boards.comment.delete"), role: .destructive) {
                            Task { await remove(comment.id) }
                        }
                        .font(.caption.weight(.semibold))
                    }
                    if canManageBoard, !comment.hidden {
                        Button(L.text("mobile.boards.comment.hide")) {
                            Task { await hide(comment.id) }
                        }
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.orange)
                    }
                }
            }
            .padding(10)
            .background(Color.secondary.opacity(0.08), in: RoundedRectangle(cornerRadius: 8))
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        do {
            comments = try await LMSAPI.fetchBoardPostComments(
                courseCode: courseCode,
                boardId: boardId,
                postId: postId,
                accessToken: token
            )
        } catch {
            errorMessage = L.text("mobile.boards.comment.error")
        }
        loading = false
    }

    private func submit() async {
        let text = draft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard canInteract, !text.isEmpty, !busy, let token = session.accessToken else { return }
        busy = true
        errorMessage = nil
        do {
            let created = try await LMSAPI.createBoardPostComment(
                courseCode: courseCode,
                boardId: boardId,
                postId: postId,
                body: BoardsLogic.makeTextBody(text),
                parentId: replyTo,
                accessToken: token
            )
            comments.append(created)
            draft = ""
            replyTo = nil
            onCountChange(1)
        } catch let error as APIError {
            if case let .httpStatus(code, _) = error, code == 403 {
                errorMessage = L.text("mobile.boards.react.forbidden")
            } else {
                errorMessage = L.text("mobile.boards.comment.error")
            }
        } catch {
            errorMessage = L.text("mobile.boards.comment.error")
        }
        busy = false
    }

    private func hide(_ id: String) async {
        guard canManageBoard, !busy, let token = session.accessToken else { return }
        busy = true
        do {
            let updated = try await LMSAPI.patchBoardPostComment(
                courseCode: courseCode,
                boardId: boardId,
                postId: postId,
                commentId: id,
                hidden: true,
                accessToken: token
            )
            comments = comments.map { $0.id == id ? updated : $0 }
            onCountChange(-1)
        } catch {
            errorMessage = L.text("mobile.boards.comment.error")
        }
        busy = false
    }

    private func remove(_ id: String) async {
        guard !busy, let token = session.accessToken else { return }
        busy = true
        do {
            try await LMSAPI.deleteBoardPostComment(
                courseCode: courseCode,
                boardId: boardId,
                postId: postId,
                commentId: id,
                accessToken: token
            )
            comments = comments.map { $0.id == id ? BoardComment(
                id: $0.id,
                postId: $0.postId,
                parentId: $0.parentId,
                authorId: $0.authorId,
                body: $0.body,
                hidden: true,
                createdAt: $0.createdAt,
                updatedAt: $0.updatedAt
            ) : $0 }
            onCountChange(-1)
        } catch {
            errorMessage = L.text("mobile.boards.comment.error")
        }
        busy = false
    }
}
