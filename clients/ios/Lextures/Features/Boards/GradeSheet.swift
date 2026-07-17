import SwiftUI

/// Grader sheet to set a card grade and optionally sync to the gradebook (VC.M5).
struct GradeSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let courseCode: String
    let boardId: String
    let post: BoardPost
    var assignmentLinked: Bool
    var onPostUpdate: (BoardPost) -> Void
    var onAnnounce: ((String) -> Void)?

    @State private var gradeText: String = ""
    @State private var busy = false
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField(L.text("mobile.boards.react.gradeInput"), text: $gradeText)
                        .keyboardType(.decimalPad)
                        .accessibilityLabel(L.text("mobile.boards.react.gradeInput"))
                }
                if let errorMessage {
                    Section {
                        Text(errorMessage)
                            .foregroundStyle(.red)
                            .font(.caption)
                    }
                }
                Section {
                    Button(L.text("mobile.boards.grade.save")) {
                        Task { await save() }
                    }
                    .disabled(busy || gradeText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)

                    if assignmentLinked {
                        Button(L.text("mobile.boards.react.sendGradebook")) {
                            Task { await syncGradebook() }
                        }
                        .disabled(busy || BoardsLogic.visibleGrade(for: post) == nil)
                    }
                }
            }
            .navigationTitle(L.text("mobile.boards.grade.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) { dismiss() }
                }
            }
            .onAppear {
                if let grade = BoardsLogic.visibleGrade(for: post) ?? post.myReaction?.value {
                    gradeText = BoardsLogic.formatGrade(grade)
                }
            }
        }
    }

    private func save() async {
        guard !busy, let token = session.accessToken else { return }
        let trimmed = gradeText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let value = Double(trimmed), value.isFinite else {
            errorMessage = L.text("mobile.boards.react.error")
            return
        }
        busy = true
        errorMessage = nil
        do {
            let result = try await LMSAPI.putBoardPostReaction(
                courseCode: courseCode,
                boardId: boardId,
                postId: post.id,
                kind: "grade",
                value: value,
                accessToken: token
            )
            onPostUpdate(BoardsLogic.applyReactionResult(post, result: result))
            onAnnounce?(L.format("mobile.boards.react.gradeSet", BoardsLogic.formatGrade(value)))
            dismiss()
        } catch let error as APIError {
            if case let .httpStatus(code, _) = error, code == 403 {
                errorMessage = L.text("mobile.boards.react.forbidden")
            } else {
                errorMessage = L.text("mobile.boards.react.error")
            }
        } catch {
            errorMessage = L.text("mobile.boards.react.error")
        }
        busy = false
    }

    private func syncGradebook() async {
        guard assignmentLinked, !busy, let token = session.accessToken else { return }
        busy = true
        errorMessage = nil
        do {
            let result = try await LMSAPI.syncBoardPostGrade(
                courseCode: courseCode,
                boardId: boardId,
                postId: post.id,
                accessToken: token
            )
            onAnnounce?(
                L.format(
                    "mobile.boards.react.gradeSynced",
                    BoardsLogic.formatGrade(result.pointsEarned)
                )
            )
            dismiss()
        } catch {
            errorMessage = L.text("mobile.boards.react.error")
        }
        busy = false
    }
}
