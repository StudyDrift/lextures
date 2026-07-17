import SwiftUI

/// Board detail with layout-aware surface (VC.M2 / VC.M3).
struct BoardDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @Environment(\.scenePhase) private var scenePhase
    @AccessibilityFocusState private var announceFocus: Bool

    let course: CourseSummary
    let boardId: String
    var titleHint: String = ""
    var canManage: Bool = false
    var onBoardChanged: (() -> Void)?

    @State private var board: Board?
    @State private var posts: [BoardPost] = []
    @State private var sections: [BoardSection] = []
    @State private var sortMode: BoardSortMode = .newest
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var unavailable = false
    @State private var showRename = false
    @State private var renameTitle = ""
    @State private var showArchiveConfirm = false
    @State private var showComposer = false
    @State private var showShare = false
    @State private var showModerationQueue = false
    @State private var showModerationSettings = false
    @State private var editingPost: BoardPost?
    @State private var editBody = ""
    @State private var editTitle = ""
    @State private var editLink = ""
    @State private var scrollTarget: String?
    @State private var pendingLayout: BoardLayout?
    @State private var announceMessage = ""
    @State private var socket = BoardSocket()
    @State private var knownPostIds: Set<String> = []

    private var title: String {
        board?.title.isEmpty == false ? (board?.title ?? titleHint) : (titleHint.isEmpty ? L.text("mobile.boards.detailTitle") : titleHint)
    }

    private var managesBoard: Bool {
        BoardsLogic.canManageBoard(
            board: board,
            courseCode: course.courseCode,
            permissions: shell.permissions
        ) || canManage
    }

    private var canPost: Bool {
        BoardsLogic.canPost(board: board, courseCode: course.courseCode, permissions: shell.permissions)
            && BoardsLogic.canWritePosts(board: board, canManage: managesBoard)
    }

    private var boardLocked: Bool { BoardsLogic.isBoardLocked(board) }
    private var boardFrozen: Bool { BoardsLogic.isBoardFrozen(board) }

    private var currentUserId: String? { shell.profile?.id }

    private var resolvedLayout: BoardLayout {
        BoardsLogic.resolveLayout(board?.layout)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            Group {
                if unavailable {
                    BoardsUnavailableView()
                        .padding(16)
                } else if loading && board == nil {
                    ProgressView()
                } else if let errorMessage, board == nil {
                    VStack(alignment: .leading, spacing: 12) {
                        LMSErrorBanner(message: errorMessage)
                        Button(L.text("mobile.common.retry")) {
                            Task { await load() }
                        }
                        .font(.subheadline.weight(.semibold))
                    }
                    .padding(16)
                } else {
                    ScrollViewReader { proxy in
                        ScrollView {
                            VStack(alignment: .leading, spacing: 16) {
                                if let errorMessage {
                                    VStack(alignment: .leading, spacing: 8) {
                                        LMSErrorBanner(message: errorMessage)
                                        Button(L.text("mobile.common.retry")) {
                                            Task { await load() }
                                        }
                                        .font(.subheadline.weight(.semibold))
                                    }
                                }
                                if board != nil, !unavailable {
                                    BoardSyncStatusChip(state: socket.connectionState)
                                }
                                if let description = board?.description, !description.isEmpty {
                                    Text(description)
                                        .font(.subheadline)
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                }
                                if boardLocked {
                                    Text(L.text("mobile.boards.moderation.lockedBanner"))
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(.orange)
                                        .accessibilityAddTraits(.updatesFrequently)
                                } else if boardFrozen {
                                    Text(L.text("mobile.boards.moderation.frozenBanner"))
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(.orange)
                                        .accessibilityAddTraits(.updatesFrequently)
                                }
                                if board?.layoutLocked == true {
                                    Text(L.text("mobile.boards.layout.lockedBadge"))
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                }
                                if !BoardsLogic.layoutHidesSortControls(resolvedLayout) {
                                    sortControls
                                }
                                if !announceMessage.isEmpty {
                                    Text(announceMessage)
                                        .font(.caption)
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                        .accessibilityFocused($announceFocus)
                                        .accessibilityAddTraits(.updatesFrequently)
                                }
                                boardSurface
                            }
                            .padding(16)
                        }
                        .refreshable { await load() }
                        .onChange(of: scrollTarget) { _, id in
                            guard let id else { return }
                            withAnimation {
                                proxy.scrollTo(id, anchor: .center)
                            }
                            scrollTarget = nil
                        }
                    }
                }
            }
        }
        .navigationTitle(title)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItemGroup(placement: .topBarTrailing) {
                if canPost, board != nil, !unavailable {
                    Button {
                        showComposer = true
                    } label: {
                        Image(systemName: "plus.circle.fill")
                    }
                    .accessibilityLabel(L.text("mobile.boards.compose.openAria"))
                }
                if managesBoard, board != nil, !unavailable {
                    Menu {
                        Button(L.text("mobile.boards.share.action")) {
                            showShare = true
                        }
                        Button(L.text("mobile.boards.moderation.queueAction")) {
                            showModerationQueue = true
                        }
                        Button(L.text("mobile.boards.moderation.settingsTitle")) {
                            showModerationSettings = true
                        }
                        layoutSwitcherMenu
                        Button(
                            board?.layoutLocked == true
                                ? L.text("mobile.boards.layout.unlock")
                                : L.text("mobile.boards.layout.lock")
                        ) {
                            Task { await toggleLock() }
                        }
                        Button(L.text("mobile.boards.rename")) {
                            renameTitle = board?.title ?? ""
                            showRename = true
                        }
                        if board?.archived != true {
                            Button(L.text("mobile.boards.archive"), role: .destructive) {
                                showArchiveConfirm = true
                            }
                        }
                    } label: {
                        Image(systemName: "ellipsis.circle")
                    }
                    .accessibilityLabel(L.text("mobile.boards.overflowMenu"))
                }
            }
        }
        .sheet(isPresented: $showComposer) {
            BoardComposerView(courseCode: course.courseCode, boardId: boardId) { created in
                posts.insert(created, at: 0)
                scrollTarget = created.id
            }
            .presentationDetents([.medium, .large])
        }
        .sheet(isPresented: $showShare) {
            if let board {
                BoardShareSheet(courseCode: course.courseCode, board: board) { updated in
                    self.board = updated
                    onBoardChanged?()
                }
                .presentationDetents([.medium, .large])
            }
        }
        .sheet(isPresented: $showModerationQueue) {
            ModerationQueueView(
                courseCode: course.courseCode,
                boardId: boardId
            ) {
                Task { await load(quiet: true) }
            }
            .presentationDetents([.medium, .large])
        }
        .sheet(isPresented: $showModerationSettings) {
            if let board {
                BoardModerationSettings(
                    courseCode: course.courseCode,
                    board: board,
                    onBoardUpdated: { updated in
                        self.board = updated
                        onBoardChanged?()
                    },
                    onAnnounce: { announce($0) }
                )
                .presentationDetents([.medium, .large])
            }
        }
        .alert(L.text("mobile.boards.rename"), isPresented: $showRename) {
            TextField(L.text("mobile.boards.titlePlaceholder"), text: $renameTitle)
            Button(L.text("mobile.common.save")) {
                Task { await rename() }
            }
            .disabled(renameTitle.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            Button(L.text("mobile.common.cancel"), role: .cancel) {}
        }
        .alert(L.text("mobile.boards.post.edit"), isPresented: Binding(
            get: { editingPost != nil },
            set: { if !$0 { editingPost = nil } }
        )) {
            TextField(L.text("mobile.boards.compose.titleLabel"), text: $editTitle)
            if editingPost?.contentType == "text" {
                TextField(L.text("mobile.boards.compose.bodyLabel"), text: $editBody)
            }
            if editingPost?.contentType == "link" || editingPost?.contentType == "video" {
                TextField(L.text("mobile.boards.compose.linkLabel"), text: $editLink)
            }
            Button(L.text("mobile.common.save")) {
                Task { await saveEdit() }
            }
            Button(L.text("mobile.common.cancel"), role: .cancel) {
                editingPost = nil
            }
        }
        .confirmationDialog(
            L.text("mobile.boards.archiveConfirmTitle"),
            isPresented: $showArchiveConfirm,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.boards.archive"), role: .destructive) {
                Task { await archive() }
            }
            Button(L.text("mobile.common.cancel"), role: .cancel) {}
        } message: {
            Text(L.text("mobile.boards.archiveConfirmMessage"))
        }
        .confirmationDialog(
            L.text("mobile.boards.layout.switchConfirm"),
            isPresented: Binding(
                get: { pendingLayout != nil },
                set: { if !$0 { pendingLayout = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.boards.layout.switchConfirmLabel")) {
                if let pendingLayout {
                    Task { await changeLayout(pendingLayout) }
                }
                pendingLayout = nil
            }
            Button(L.text("mobile.common.cancel"), role: .cancel) {
                pendingLayout = nil
            }
        }
        .task {
            socket.connect(
                courseCode: course.courseCode,
                boardId: boardId,
                accessToken: { session.accessToken }
            )
            await load()
        }
        .onDisappear { socket.disconnect() }
        .onChange(of: scenePhase) { _, phase in
            switch phase {
            case .background, .inactive:
                socket.disconnect()
            case .active:
                socket.connect(
                    courseCode: course.courseCode,
                    boardId: boardId,
                    accessToken: { session.accessToken }
                )
                Task { await load(quiet: true) }
            @unknown default:
                break
            }
        }
        .onChange(of: socket.revision) { _, rev in
            guard rev > 0 else { return }
            Task { await load(quiet: true, fromSocket: true) }
        }
        .onChange(of: socket.connectRevision) { _, rev in
            guard rev > 0 else { return }
            Task { await load(quiet: true) }
        }
        .onChange(of: socket.lockedOrFrozenNotice) { _, noticed in
            guard noticed else { return }
            announce(L.text("mobile.boards.sync.lockedNotice"))
            socket.clearLockedOrFrozenNotice()
            Task { await load(quiet: true) }
        }
        .task(id: board?.frozenUntil) {
            await waitForFreezeExpiry()
        }
    }

    private func waitForFreezeExpiry() async {
        guard let until = BoardsLogic.parseBoardDate(board?.frozenUntil) else { return }
        let delay = until.timeIntervalSinceNow
        guard delay > 0 else {
            await load(quiet: true)
            return
        }
        try? await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000) + 250_000_000)
        guard !Task.isCancelled else { return }
        await load(quiet: true)
        if !BoardsLogic.isBoardFrozen(board) {
            announce(L.text("mobile.boards.moderation.unfrozenAnnounce"))
        }
    }

    @ViewBuilder
    private var boardSurface: some View {
        if loading && posts.isEmpty {
            ProgressView()
                .frame(maxWidth: .infinity)
                .padding(.vertical, 24)
        } else if let board {
            BoardSurface(
                board: board,
                posts: posts,
                sections: sections,
                sortMode: sortMode,
                canManage: managesBoard,
                currentUserId: currentUserId,
                onEdit: { post in
                    editingPost = post
                    editTitle = post.title
                    editBody = BoardsLogic.bodyPlainText(post)
                    editLink = post.linkUrl ?? ""
                },
                onDelete: { post in
                    Task { await deletePost(post) }
                },
                onArrange: { post, input in
                    Task { await arrange(post: post, input: input) }
                },
                onCreateSection: managesBoard ? { title in
                    Task { await createSection(title) }
                } : nil,
                onDeleteSection: managesBoard ? { id in
                    Task { await deleteSection(id) }
                } : nil
            )
            .environment(\.boardEngagement, BoardEngagementHandlers(
                courseCode: course.courseCode,
                onPostUpdate: { updated in
                    posts = posts.map { $0.id == updated.id ? updated : $0 }
                },
                onAnnounce: { message in
                    announceMessage = message
                    announceFocus = true
                },
                onHidePost: { post in
                    Task { await hidePost(post) }
                },
                onRemovePost: { post in
                    Task { await removeModeratedPost(post) }
                }
            ))
        }
    }

    private var sortControls: some View {
        Menu {
            ForEach(BoardSortMode.allCases, id: \.self) { mode in
                Button {
                    sortMode = mode
                } label: {
                    if sortMode == mode {
                        Label(sortLabel(mode), systemImage: "checkmark")
                    } else {
                        Text(sortLabel(mode))
                    }
                }
            }
        } label: {
            Label(L.text("mobile.boards.sort.label"), systemImage: "arrow.up.arrow.down")
                .font(.subheadline.weight(.semibold))
        }
        .accessibilityLabel(L.text("mobile.boards.sort.label"))
    }

    @ViewBuilder
    private var layoutSwitcherMenu: some View {
        Menu(L.text("mobile.boards.layout.switcherAria")) {
            ForEach(BoardLayout.allCases, id: \.self) { layout in
                Button {
                    requestLayoutChange(layout)
                } label: {
                    if resolvedLayout == layout {
                        Label(L.text("mobile.boards.layout.\(layout.rawValue)"), systemImage: "checkmark")
                    } else {
                        Text(L.text("mobile.boards.layout.\(layout.rawValue)"))
                    }
                }
            }
        }
    }

    private func sortLabel(_ mode: BoardSortMode) -> String {
        switch mode {
        case .newest: return L.text("mobile.boards.sort.newest")
        case .oldest: return L.text("mobile.boards.sort.oldest")
        case .author: return L.text("mobile.boards.sort.author")
        case .mostReacted: return L.text("mobile.boards.sort.mostReacted")
        }
    }

    private func announce(_ message: String) {
        announceMessage = message
        announceFocus = true
    }

    private func requestLayoutChange(_ layout: BoardLayout) {
        guard layout != resolvedLayout else { return }
        if resolvedLayout == .canvas && layout != .canvas {
            pendingLayout = layout
        } else {
            Task { await changeLayout(layout) }
        }
    }

    private func load(quiet: Bool = false, fromSocket: Bool = false) async {
        guard let token = session.accessToken else { return }
        if !quiet { loading = true }
        errorMessage = nil
        unavailable = false
        defer { if !quiet { loading = false } }
        let previousIds = knownPostIds
        do {
            async let boardTask = LMSAPI.fetchBoard(
                courseCode: course.courseCode,
                boardId: boardId,
                accessToken: token
            )
            async let postsTask = LMSAPI.fetchBoardPosts(
                courseCode: course.courseCode,
                boardId: boardId,
                accessToken: token
            )
            async let sectionsTask = LMSAPI.fetchBoardSections(
                courseCode: course.courseCode,
                boardId: boardId,
                accessToken: token
            )
            board = try await boardTask
            posts = try await postsTask
            sections = try await sectionsTask
            let nextIds = Set(posts.map(\.id))
            if fromSocket, !previousIds.isEmpty {
                let added = nextIds.subtracting(previousIds).count
                let planCreated = socket.lastRefetchPlan.createdCount
                let announceCount = max(added, planCreated)
                if announceCount > 1 {
                    announce(L.format("mobile.boards.sync.cardsAdded", announceCount))
                } else if announceCount == 1 {
                    announce(L.text("mobile.boards.sync.cardAdded"))
                } else if socket.lastRefetchPlan.full || socket.lastRefetchPlan.postId != nil {
                    announce(L.text("mobile.boards.sync.boardUpdated"))
                }
            }
            knownPostIds = nextIds
        } catch let error as APIError {
            if case let .httpStatus(code, _) = error, code == 404 || code == 403 {
                unavailable = true
                board = nil
                posts = []
                sections = []
                knownPostIds = []
            } else if !quiet {
                errorMessage = error.errorDescription ?? L.text("mobile.boards.loadError")
            }
        } catch {
            if !quiet {
                errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.loadError")
            }
        }
    }

    private func arrange(post: BoardPost, input: ArrangeBoardPostInput) async {
        guard let token = session.accessToken else { return }
        guard BoardsLogic.canArrangePost(
            post: post,
            board: board,
            currentUserId: currentUserId,
            canManage: managesBoard
        ) else { return }
        let previous = posts
        if let idx = posts.firstIndex(where: { $0.id == post.id }) {
            var optimistic = posts[idx]
            if let sectionId = input.sectionId { optimistic.sectionId = sectionId }
            if let sortIndex = input.sortIndex { optimistic.sortIndex = sortIndex }
            if let position = input.position { optimistic.position = position }
            if let eventDate = input.eventDate {
                optimistic.eventDate = eventDate.isEmpty ? nil : eventDate
            }
            if let lat = input.lat { optimistic.lat = lat }
            if let lng = input.lng { optimistic.lng = lng }
            posts[idx] = optimistic
        }
        do {
            let updated = try await LMSAPI.arrangeBoardPost(
                courseCode: course.courseCode,
                boardId: boardId,
                postId: post.id,
                input: input,
                accessToken: token
            )
            if let idx = posts.firstIndex(where: { $0.id == updated.id }) {
                posts[idx] = updated
            }
            announce(L.text("mobile.boards.arrange.moved"))
        } catch let error as APIError {
            posts = previous
            if case let .httpStatus(code, message) = error, code == 403 {
                errorMessage = BoardsLogic.isLockOrFreezeMessage(message)
                    ? L.text("mobile.boards.sync.lockedNotice")
                    : L.text("mobile.boards.post.forbidden")
            } else {
                errorMessage = error.errorDescription ?? L.text("mobile.boards.arrange.error")
            }
        } catch {
            posts = previous
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.arrange.error")
        }
    }

    private func changeLayout(_ layout: BoardLayout) async {
        guard let token = session.accessToken else { return }
        do {
            board = try await LMSAPI.patchBoard(
                courseCode: course.courseCode,
                boardId: boardId,
                layout: layout.rawValue,
                accessToken: token
            )
            announce(L.format("mobile.boards.layout.changed", L.text("mobile.boards.layout.\(layout.rawValue)")))
            if layout == .columns {
                sections = try await LMSAPI.fetchBoardSections(
                    courseCode: course.courseCode,
                    boardId: boardId,
                    accessToken: token
                )
                posts = try await LMSAPI.fetchBoardPosts(
                    courseCode: course.courseCode,
                    boardId: boardId,
                    accessToken: token
                )
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.renameError")
        }
    }

    private func toggleLock() async {
        guard let token = session.accessToken, let board else { return }
        do {
            let updated = try await LMSAPI.patchBoard(
                courseCode: course.courseCode,
                boardId: boardId,
                layoutLocked: !board.layoutLocked,
                accessToken: token
            )
            self.board = updated
            announce(
                updated.layoutLocked
                    ? L.text("mobile.boards.layout.lockedAnnounce")
                    : L.text("mobile.boards.layout.unlockedAnnounce")
            )
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.renameError")
        }
    }

    private func createSection(_ title: String) async {
        guard let token = session.accessToken else { return }
        do {
            let created = try await LMSAPI.createBoardSection(
                courseCode: course.courseCode,
                boardId: boardId,
                title: title,
                accessToken: token
            )
            sections.append(created)
            announce(L.text("mobile.boards.section.created"))
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.createError")
        }
    }

    private func deleteSection(_ sectionId: String) async {
        guard let token = session.accessToken else { return }
        do {
            try await LMSAPI.deleteBoardSection(
                courseCode: course.courseCode,
                boardId: boardId,
                sectionId: sectionId,
                accessToken: token
            )
            sections.removeAll { $0.id == sectionId }
            posts = try await LMSAPI.fetchBoardPosts(
                courseCode: course.courseCode,
                boardId: boardId,
                accessToken: token
            )
            announce(L.text("mobile.boards.section.deleted"))
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.post.deleteError")
        }
    }

    private func hidePost(_ post: BoardPost) async {
        guard let token = session.accessToken else { return }
        do {
            let updated = try await LMSAPI.hideBoardPost(
                courseCode: course.courseCode,
                boardId: boardId,
                postId: post.id,
                accessToken: token
            )
            if managesBoard {
                posts = posts.map { $0.id == updated.id ? updated : $0 }
            } else {
                posts.removeAll { $0.id == post.id }
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.boards.moderation.actionError")
        }
    }

    private func removeModeratedPost(_ post: BoardPost) async {
        guard let token = session.accessToken else { return }
        do {
            _ = try await LMSAPI.removeBoardPost(
                courseCode: course.courseCode,
                boardId: boardId,
                postId: post.id,
                accessToken: token
            )
            posts.removeAll { $0.id == post.id }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.boards.moderation.actionError")
        }
    }

    private func deletePost(_ post: BoardPost) async {
        guard let token = session.accessToken else { return }
        let previous = posts
        posts.removeAll { $0.id == post.id }
        do {
            try await LMSAPI.deleteBoardPost(
                courseCode: course.courseCode,
                boardId: boardId,
                postId: post.id,
                accessToken: token
            )
        } catch let error as APIError {
            posts = previous
            if case let .httpStatus(code, _) = error, code == 403 {
                errorMessage = L.text("mobile.boards.post.forbidden")
            } else {
                errorMessage = error.errorDescription ?? L.text("mobile.boards.post.deleteError")
            }
        } catch {
            posts = previous
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.post.deleteError")
        }
    }

    private func saveEdit() async {
        guard let token = session.accessToken, let post = editingPost else { return }
        do {
            var body: BoardPostBody?
            var link: String?
            if post.contentType == "text" {
                body = BoardsLogic.makeTextBody(editBody)
            }
            if post.contentType == "link" || post.contentType == "video" {
                link = editLink.trimmingCharacters(in: .whitespacesAndNewlines)
            }
            let updated = try await LMSAPI.patchBoardPost(
                courseCode: course.courseCode,
                boardId: boardId,
                postId: post.id,
                title: editTitle.trimmingCharacters(in: .whitespacesAndNewlines),
                body: body,
                linkUrl: link,
                accessToken: token
            )
            if let idx = posts.firstIndex(where: { $0.id == updated.id }) {
                posts[idx] = updated
            }
            editingPost = nil
        } catch let error as APIError {
            if case let .httpStatus(code, _) = error, code == 403 {
                errorMessage = L.text("mobile.boards.post.forbidden")
            } else {
                errorMessage = error.errorDescription ?? L.text("mobile.boards.post.editError")
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.post.editError")
        }
    }

    private func rename() async {
        guard let token = session.accessToken else { return }
        let title = renameTitle.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !title.isEmpty else { return }
        do {
            board = try await LMSAPI.patchBoard(
                courseCode: course.courseCode,
                boardId: boardId,
                title: title,
                accessToken: token
            )
            onBoardChanged?()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.renameError")
        }
    }

    private func archive() async {
        guard let token = session.accessToken else { return }
        do {
            _ = try await LMSAPI.patchBoard(
                courseCode: course.courseCode,
                boardId: boardId,
                archived: true,
                accessToken: token
            )
            onBoardChanged?()
            dismiss()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.archiveError")
        }
    }
}
