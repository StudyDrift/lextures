import SwiftUI

/// Per-card reaction control for the board's `reactionMode` (VC.M5).
struct ReactionControl: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let courseCode: String
    let boardId: String
    let post: BoardPost
    let reactionMode: BoardReactionMode
    var canInteract: Bool = true
    var canGrade: Bool = false
    var assignmentLinked: Bool = false
    var onPostUpdate: (BoardPost) -> Void
    var onAnnounce: ((String) -> Void)?
    var onOpenGradeSheet: (() -> Void)?

    @State private var busy = false
    @State private var errorMessage: String?

    var body: some View {
        Group {
            switch reactionMode {
            case .none:
                EmptyView()
            case .like:
                toggleButton(
                    kind: "like",
                    systemImage: post.myReaction != nil ? "heart.fill" : "heart",
                    pressedColor: .pink,
                    labelOn: L.text("mobile.boards.react.unlike"),
                    labelOff: L.text("mobile.boards.react.like"),
                    announceOn: L.text("mobile.boards.react.likeOn"),
                    announceOff: L.text("mobile.boards.react.likeOff")
                )
            case .vote:
                toggleButton(
                    kind: "vote",
                    systemImage: post.myReaction != nil ? "arrow.up.circle.fill" : "arrow.up.circle",
                    pressedColor: .indigo,
                    labelOn: L.text("mobile.boards.react.unvote"),
                    labelOff: L.text("mobile.boards.react.vote"),
                    announceOn: L.text("mobile.boards.react.voteOn"),
                    announceOff: L.text("mobile.boards.react.voteOff")
                )
            case .star:
                starRow
            case .grade:
                gradeChip
            }
        }
    }

    private var pressed: Bool { post.myReaction != nil }
    private var count: Int { post.reactionCount ?? 0 }

    private func toggleButton(
        kind: String,
        systemImage: String,
        pressedColor: Color,
        labelOn: String,
        labelOff: String,
        announceOn: String,
        announceOff: String
    ) -> some View {
        Button {
            Task { await toggleLikeOrVote(kind: kind, announceOn: announceOn, announceOff: announceOff) }
        } label: {
            HStack(spacing: 4) {
                Image(systemName: systemImage)
                    .foregroundStyle(pressed ? pressedColor : LexturesTheme.textSecondary(for: colorScheme))
                if count > 0 || kind == "vote" {
                    Text("\(count)")
                        .font(.caption.weight(.medium))
                        .monospacedDigit()
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            .frame(minHeight: 36)
            .padding(.horizontal, 8)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .disabled(!canInteract || busy)
        .opacity(canInteract ? 1 : 0.5)
        .accessibilityLabel(pressed ? labelOn : labelOff)
        .accessibilityAddTraits(pressed ? .isSelected : [])
    }

    private var starRow: some View {
        let mine = Int(post.myReaction?.value ?? 0)
        return HStack(spacing: 2) {
            ForEach(1 ... 5, id: \.self) { n in
                Button {
                    Task { await setStars(n) }
                } label: {
                    Image(systemName: mine >= n ? "star.fill" : "star")
                        .foregroundStyle(.orange)
                        .frame(minWidth: 36, minHeight: 36)
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .disabled(!canInteract || busy)
                .accessibilityLabel(L.format("mobile.boards.react.starN", n))
                .accessibilityAddTraits(mine == n ? .isSelected : [])
            }
            if let avg = post.avgStars {
                Text(
                    L.format(
                        "mobile.boards.react.avgStars",
                        BoardsLogic.formatAvgStars(avg),
                        count
                    )
                )
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .padding(.leading, 4)
            }
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel(L.text("mobile.boards.react.starLabel"))
        .opacity(canInteract ? 1 : 0.5)
    }

    @ViewBuilder
    private var gradeChip: some View {
        if canGrade {
            Button {
                onOpenGradeSheet?()
            } label: {
                HStack(spacing: 4) {
                    Image(systemName: "graduationcap")
                    if let grade = BoardsLogic.visibleGrade(for: post) {
                        Text(BoardsLogic.formatGrade(grade))
                            .monospacedDigit()
                    } else {
                        Text(L.text("mobile.boards.react.grade"))
                    }
                }
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                .frame(minHeight: 36)
                .padding(.horizontal, 8)
            }
            .buttonStyle(.plain)
            .accessibilityLabel(L.text("mobile.boards.react.gradeInput"))
        } else if let grade = BoardsLogic.visibleGrade(for: post) {
            Text(L.format("mobile.boards.react.yourGrade", BoardsLogic.formatGrade(grade)))
                .font(.caption.weight(.medium))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                .frame(minHeight: 36)
        }
    }

    private func toggleLikeOrVote(kind: String, announceOn: String, announceOff: String) async {
        guard canInteract, !busy, let token = session.accessToken else { return }
        busy = true
        errorMessage = nil
        let previous = post
        let optimistic = BoardsLogic.optimisticToggleReaction(post, kind: kind)
        onPostUpdate(optimistic)
        do {
            let result = try await LMSAPI.putBoardPostReaction(
                courseCode: courseCode,
                boardId: boardId,
                postId: post.id,
                kind: kind,
                accessToken: token
            )
            onPostUpdate(BoardsLogic.applyReactionResult(previous, result: result))
            onAnnounce?(result.active ? announceOn : announceOff)
        } catch let error as APIError {
            onPostUpdate(previous)
            if case let .httpStatus(code, _) = error, code == 403 {
                errorMessage = L.text("mobile.boards.react.forbidden")
            } else {
                errorMessage = L.text("mobile.boards.react.error")
            }
            onAnnounce?(errorMessage ?? "")
        } catch {
            onPostUpdate(previous)
            errorMessage = L.text("mobile.boards.react.error")
            onAnnounce?(errorMessage ?? "")
        }
        busy = false
    }

    private func setStars(_ value: Int) async {
        guard canInteract, !busy, let token = session.accessToken else { return }
        busy = true
        let previous = post
        onPostUpdate(BoardsLogic.optimisticSetStars(post, value: value))
        do {
            let result = try await LMSAPI.putBoardPostReaction(
                courseCode: courseCode,
                boardId: boardId,
                postId: post.id,
                kind: "star",
                value: Double(value),
                accessToken: token
            )
            onPostUpdate(BoardsLogic.applyReactionResult(previous, result: result))
            onAnnounce?(L.format("mobile.boards.react.starSet", value))
        } catch let error as APIError {
            onPostUpdate(previous)
            if case let .httpStatus(code, _) = error, code == 403 {
                onAnnounce?(L.text("mobile.boards.react.forbidden"))
            } else {
                onAnnounce?(L.text("mobile.boards.react.error"))
            }
        } catch {
            onPostUpdate(previous)
            onAnnounce?(L.text("mobile.boards.react.error"))
        }
        busy = false
    }
}
