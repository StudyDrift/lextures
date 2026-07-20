import SwiftUI

/// Course boards list (VC.M1).
struct BoardsListView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    var initialBoardId: String?

    @State private var boards: [Board] = []
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var includeArchived = false
    @State private var showNewBoard = false
    @State private var showTemplatePicker = false
    @State private var newTitle = ""
    @State private var newDescription = ""
    @State private var creating = false
    @State private var openBoard: BoardRoute?
    @State private var featureUnavailable = false
    @State private var renameTarget: Board?
    @State private var renameTitle = ""
    @State private var archiveTarget: Board?
    @State private var duplicateTarget: Board?
    @State private var duplicating = false

    private var canCreate: Bool {
        BoardsLogic.canCreateBoards(courseCode: course.courseCode, permissions: shell.permissions)
    }

    private var advancedEnabled: Bool {
        BoardsAdvancedLogic.isAdvancedEnabled(
            courseEnabled: course.isVisualBoardsEnabled,
            features: shell.platformFeatures
        )
    }

    private var canUseTemplates: Bool {
        BoardsAdvancedLogic.canUseTemplates(
            courseEnabled: course.isVisualBoardsEnabled,
            features: shell.platformFeatures,
            canCreate: canCreate
        )
    }

    private var visibleBoards: [Board] {
        BoardsLogic.sortedBoards(boards, includeArchived: includeArchived)
    }

    var body: some View {
        Group {
            if featureUnavailable || !course.isVisualBoardsEnabled {
                BoardsUnavailableView()
            } else {
                listContent
            }
        }
        .navigationDestination(item: $openBoard) { route in
            BoardDetailView(
                course: course,
                boardId: route.boardId,
                titleHint: route.title,
                canManage: canCreate,
                onBoardChanged: { Task { await load(force: true) } }
            )
        }
        .alert(L.text("mobile.boards.newBoard"), isPresented: $showNewBoard) {
            TextField(L.text("mobile.boards.titlePlaceholder"), text: $newTitle)
            TextField(L.text("mobile.boards.descriptionPlaceholder"), text: $newDescription)
            Button(L.text("mobile.boards.create")) {
                Task { await createBoard() }
            }
            .disabled(newTitle.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || creating)
            Button(L.text("mobile.common.cancel"), role: .cancel) {
                newTitle = ""
                newDescription = ""
            }
        }
        .alert(L.text("mobile.boards.rename"), isPresented: Binding(
            get: { renameTarget != nil },
            set: { if !$0 { renameTarget = nil } }
        )) {
            TextField(L.text("mobile.boards.titlePlaceholder"), text: $renameTitle)
            Button(L.text("mobile.common.save")) {
                Task { await renameBoard() }
            }
            .disabled(renameTitle.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            Button(L.text("mobile.common.cancel"), role: .cancel) {
                renameTarget = nil
            }
        }
        .confirmationDialog(
            L.text("mobile.boards.archiveConfirmTitle"),
            isPresented: Binding(
                get: { archiveTarget != nil },
                set: { if !$0 { archiveTarget = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.boards.archive"), role: .destructive) {
                Task { await archiveBoard() }
            }
            Button(L.text("mobile.common.cancel"), role: .cancel) {
                archiveTarget = nil
            }
        } message: {
            Text(L.text("mobile.boards.archiveConfirmMessage"))
        }
        .confirmationDialog(
            L.text("mobile.boards.templates.duplicateTitle"),
            isPresented: Binding(
                get: { duplicateTarget != nil },
                set: { if !$0 { duplicateTarget = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.boards.templates.duplicateStructure")) {
                Task { await duplicateBoard(mode: .structure) }
            }
            Button(L.text("mobile.boards.templates.duplicateFull")) {
                Task { await duplicateBoard(mode: .full) }
            }
            Button(L.text("mobile.common.cancel"), role: .cancel) {
                duplicateTarget = nil
            }
        } message: {
            Text(L.text("mobile.boards.templates.duplicateMessage"))
        }
        .sheet(isPresented: $showTemplatePicker) {
            BoardTemplatePickerView(courseCode: course.courseCode) { board in
                boards.insert(board, at: 0)
                openBoard = BoardRoute(boardId: board.id, title: board.title)
            }
        }
        .task {
            await load()
            if let initialBoardId, !initialBoardId.isEmpty,
               let board = boards.first(where: { $0.id == initialBoardId }) {
                openBoard = BoardRoute(boardId: board.id, title: board.title)
            } else if let initialBoardId, !initialBoardId.isEmpty, course.isVisualBoardsEnabled {
                openBoard = BoardRoute(boardId: initialBoardId, title: "")
            }
        }
        .refreshable { await load(force: true) }
    }

    @ViewBuilder
    private var listContent: some View {
        VStack(alignment: .leading, spacing: 12) {
            if !NetworkMonitor.shared.isOnline {
                OfflineBanner()
            }
            if let errorMessage {
                VStack(alignment: .leading, spacing: 8) {
                    LMSErrorBanner(message: errorMessage)
                    Button(L.text("mobile.common.retry")) {
                        Task { await load(force: true) }
                    }
                    .font(.subheadline.weight(.semibold))
                }
            }

            HStack {
                Toggle(isOn: $includeArchived) {
                    Text(L.text("mobile.boards.showArchived"))
                        .font(.subheadline)
                }
                .toggleStyle(.switch)
                .onChange(of: includeArchived) { _, _ in
                    Task { await load(force: true) }
                }
                Spacer()
                if canCreate {
                    Menu {
                        Button(L.text("mobile.boards.newBoard")) {
                            showNewBoard = true
                        }
                        if canUseTemplates {
                            Button(L.text("mobile.boards.templates.fromTemplate")) {
                                showTemplatePicker = true
                            }
                        }
                    } label: {
                        Label(L.text("mobile.boards.newBoard"), systemImage: "plus")
                            .font(.subheadline.weight(.semibold))
                    }
                    .accessibilityLabel(L.text("mobile.boards.newBoard"))
                }
            }

            if loading && visibleBoards.isEmpty {
                LMSSkeletonList(count: 3)
            } else if visibleBoards.isEmpty {
                LMSEmptyState(
                    systemImage: "rectangle.3.group",
                    title: L.text("mobile.boards.emptyTitle"),
                    message: L.text("mobile.boards.emptyMessage")
                )
            } else {
                ForEach(visibleBoards) { board in
                    Button {
                        openBoard = BoardRoute(boardId: board.id, title: board.title)
                    } label: {
                        boardRow(board)
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel(boardAccessibilityLabel(board))
                    .contextMenu {
                        if canCreate {
                            Button(L.text("mobile.boards.rename")) {
                                renameTarget = board
                                renameTitle = board.title
                            }
                            if advancedEnabled {
                                Button(L.text("mobile.boards.templates.duplicate")) {
                                    duplicateTarget = board
                                }
                                .disabled(duplicating)
                            }
                            if !board.archived {
                                Button(L.text("mobile.boards.archive"), role: .destructive) {
                                    archiveTarget = board
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func boardRow(_ board: Board) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 6) {
                HStack(alignment: .top) {
                    Text(board.title)
                        .font(LexturesTheme.displayFont(17))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .multilineTextAlignment(.leading)
                    Spacer()
                    if board.archived {
                        Text(L.text("mobile.boards.archivedBadge"))
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Image(systemName: "chevron.right")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                if !board.description.isEmpty {
                    Text(board.description)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .lineLimit(2)
                        .multilineTextAlignment(.leading)
                }
                let updated = BoardsLogic.relativeUpdatedLabel(board)
                if !updated.isEmpty {
                    Text(updated)
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    private func boardAccessibilityLabel(_ board: Board) -> String {
        var parts = [board.title]
        if !board.description.isEmpty { parts.append(board.description) }
        let updated = BoardsLogic.relativeUpdatedLabel(board)
        if !updated.isEmpty { parts.append(updated) }
        return parts.joined(separator: ", ")
    }

    private func load(force: Bool = false) async {
        guard let token = session.accessToken else { return }
        if !force && !boards.isEmpty { loading = false }
        loading = true
        errorMessage = nil
        featureUnavailable = false
        defer { loading = false }
        do {
            boards = try await LMSAPI.fetchBoards(
                courseCode: course.courseCode,
                includeArchived: includeArchived,
                accessToken: token
            )
        } catch let error as APIError {
            if case let .httpStatus(code, _) = error, code == 404 {
                featureUnavailable = true
                boards = []
            } else {
                errorMessage = error.errorDescription ?? L.text("mobile.boards.loadError")
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.loadError")
        }
    }

    private func createBoard() async {
        let title = newTitle.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !title.isEmpty, let token = session.accessToken else { return }
        creating = true
        defer { creating = false }
        do {
            let board = try await LMSAPI.createBoard(
                courseCode: course.courseCode,
                title: title,
                description: newDescription.trimmingCharacters(in: .whitespacesAndNewlines),
                accessToken: token
            )
            newTitle = ""
            newDescription = ""
            await load(force: true)
            openBoard = BoardRoute(boardId: board.id, title: board.title)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.createError")
        }
    }

    private func renameBoard() async {
        guard let target = renameTarget, let token = session.accessToken else { return }
        let title = renameTitle.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !title.isEmpty else { return }
        do {
            _ = try await LMSAPI.patchBoard(
                courseCode: course.courseCode,
                boardId: target.id,
                title: title,
                accessToken: token
            )
            renameTarget = nil
            await load(force: true)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.renameError")
            renameTarget = nil
        }
    }

    private func duplicateBoard(mode: BoardCopyMode) async {
        guard let target = duplicateTarget, let token = session.accessToken else { return }
        duplicating = true
        defer {
            duplicating = false
            duplicateTarget = nil
        }
        do {
            let result = try await LMSAPI.duplicateBoard(
                targetCourseCode: course.courseCode,
                sourceBoardId: target.id,
                mode: mode,
                title: target.title,
                accessToken: token
            )
            switch result {
            case let .board(board):
                await load(force: true)
                openBoard = BoardRoute(boardId: board.id, title: board.title)
            case let .job(job):
                var current = job
                var attempt = 0
                while !BoardsAdvancedLogic.isCopyTerminal(current.status) {
                    let delay = BoardsAdvancedLogic.pollDelaySeconds(attempt: attempt)
                    try await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))
                    current = try await LMSAPI.fetchBoardCopyJob(
                        courseCode: course.courseCode,
                        jobId: current.id,
                        accessToken: token
                    )
                    attempt += 1
                    if attempt > 30 { break }
                }
                await load(force: true)
                if let id = current.resultBoardId, !id.isEmpty {
                    openBoard = BoardRoute(boardId: id, title: current.title)
                }
            }
        } catch {
            errorMessage = L.text("mobile.boards.templates.duplicateError")
        }
    }

    private func archiveBoard() async {
        guard let target = archiveTarget, let token = session.accessToken else { return }
        do {
            _ = try await LMSAPI.patchBoard(
                courseCode: course.courseCode,
                boardId: target.id,
                archived: true,
                accessToken: token
            )
            archiveTarget = nil
            await load(force: true)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.archiveError")
            archiveTarget = nil
        }
    }
}

/// Shown when boards are flag-off or the list endpoint returns 404.
struct BoardsUnavailableView: View {
    var body: some View {
        LMSEmptyState(
            systemImage: "rectangle.3.group.bubble.left",
            title: L.text("mobile.boards.unavailableTitle"),
            message: L.text("mobile.boards.unavailableMessage")
        )
        .frame(maxWidth: .infinity)
        .padding(.vertical, 24)
    }
}
