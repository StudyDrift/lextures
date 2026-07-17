import SwiftUI

/// Manager board moderation controls: mode, filter, lock, freeze (VC.M7).
struct BoardModerationSettings: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme

    let courseCode: String
    let board: Board
    var onBoardUpdated: (Board) -> Void
    var onAnnounce: (String) -> Void

    @State private var moderationMode: BoardModerationMode = .open
    @State private var filterAction: BoardFilterAction = .flag
    @State private var saving = false
    @State private var errorMessage: String?

    private var floorLocked: Bool {
        BoardsLogic.moderationControlsLockedByOrgFloor(board)
    }

    private var frozen: Bool {
        BoardsLogic.isBoardFrozen(board)
    }

    var body: some View {
        NavigationStack {
            Form {
                if let errorMessage {
                    Section {
                        Text(errorMessage)
                            .font(.caption)
                            .foregroundStyle(.red)
                    }
                }

                if floorLocked {
                    Section {
                        Text(L.text("mobile.boards.moderation.minorsFloor"))
                            .font(.caption)
                            .foregroundStyle(.orange)
                            .accessibilityAddTraits(.updatesFrequently)
                    }
                }

                Section {
                    Picker(L.text("mobile.boards.moderation.modeLabel"), selection: $moderationMode) {
                        Text(L.text("mobile.boards.moderation.mode.open")).tag(BoardModerationMode.open)
                        Text(L.text("mobile.boards.moderation.mode.approval")).tag(BoardModerationMode.approval)
                    }
                    .disabled(floorLocked || saving)
                    .onChange(of: moderationMode) { _, next in
                        guard next.rawValue != board.moderationMode.lowercased() else { return }
                        Task { await patch(moderationMode: next.rawValue) }
                    }

                    Picker(L.text("mobile.boards.moderation.filterLabel"), selection: $filterAction) {
                        Text(L.text("mobile.boards.moderation.filter.flag")).tag(BoardFilterAction.flag)
                        Text(L.text("mobile.boards.moderation.filter.block")).tag(BoardFilterAction.block)
                    }
                    .disabled(floorLocked || saving)
                    .onChange(of: filterAction) { _, next in
                        guard next.rawValue != board.filterAction.lowercased() else { return }
                        Task { await patch(filterAction: next.rawValue) }
                    }
                }

                Section {
                    Button {
                        Task { await patch(locked: !board.locked) }
                    } label: {
                        Text(
                            board.locked
                                ? L.text("mobile.boards.moderation.unlock")
                                : L.text("mobile.boards.moderation.lock")
                        )
                    }
                    .disabled(saving)

                    Button {
                        Task { await patch(freezeMinutes: 5) }
                    } label: {
                        Text(L.text("mobile.boards.moderation.freeze5"))
                    }
                    .disabled(saving)

                    if frozen {
                        Button {
                            Task { await patch(frozenUntil: "") }
                        } label: {
                            Text(L.text("mobile.boards.moderation.unfreeze"))
                        }
                        .disabled(saving)
                    }
                }
            }
            .navigationTitle(L.text("mobile.boards.moderation.settingsTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) { dismiss() }
                }
            }
            .onAppear {
                moderationMode = BoardsLogic.moderationMode(for: board)
                filterAction = BoardsLogic.filterAction(for: board)
            }
        }
    }

    private func patch(
        moderationMode: String? = nil,
        filterAction: String? = nil,
        locked: Bool? = nil,
        frozenUntil: String? = nil,
        freezeMinutes: Int? = nil
    ) async {
        guard let token = session.accessToken else { return }
        saving = true
        errorMessage = nil
        defer { saving = false }
        do {
            let updated = try await LMSAPI.patchBoardModeration(
                courseCode: courseCode,
                boardId: board.id,
                moderationMode: moderationMode,
                filterAction: filterAction,
                locked: locked,
                frozenUntil: frozenUntil,
                freezeMinutes: freezeMinutes,
                accessToken: token
            )
            onBoardUpdated(updated)
            if let locked {
                onAnnounce(
                    locked
                        ? L.text("mobile.boards.moderation.lockedAnnounce")
                        : L.text("mobile.boards.moderation.unlockedAnnounce")
                )
            }
            if let freezeMinutes {
                onAnnounce(L.format("mobile.boards.moderation.frozenAnnounce", freezeMinutes))
            }
            if frozenUntil == "" {
                onAnnounce(L.text("mobile.boards.moderation.unfrozenAnnounce"))
            }
        } catch {
            errorMessage = L.text("mobile.boards.moderation.settingsError")
            self.moderationMode = BoardsLogic.moderationMode(for: board)
            self.filterAction = BoardsLogic.filterAction(for: board)
        }
    }
}
