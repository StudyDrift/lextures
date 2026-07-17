import SwiftUI

/// Report a board post or comment (VC.M7).
struct ReportDialog: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme

    let courseCode: String
    let boardId: String
    var postId: String?
    var commentId: String?
    var onSubmitted: (() -> Void)?

    private enum ReasonKey: String, CaseIterable, Identifiable {
        case hurtful, inappropriate, spam, other
        var id: String { rawValue }
    }

    @State private var reasonKey: ReasonKey = .hurtful
    @State private var details = ""
    @State private var submitting = false
    @State private var errorMessage: String?
    @State private var alreadyReported = false

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text(L.text("mobile.boards.report.subtitle"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                if alreadyReported {
                    Section {
                        Text(L.text("mobile.boards.report.alreadyReported"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }

                if let errorMessage {
                    Section {
                        Text(errorMessage)
                            .font(.caption)
                            .foregroundStyle(.red)
                    }
                }

                Section {
                    Picker(L.text("mobile.boards.report.reasonLabel"), selection: $reasonKey) {
                        ForEach(ReasonKey.allCases) { key in
                            Text(L.text("mobile.boards.report.reason.\(key.rawValue)")).tag(key)
                        }
                    }
                    .accessibilityLabel(L.text("mobile.boards.report.reasonLabel"))
                    .disabled(alreadyReported)

                    TextField(L.text("mobile.boards.report.detailsLabel"), text: $details, axis: .vertical)
                        .lineLimit(3 ... 6)
                        .disabled(alreadyReported)
                }
            }
            .navigationTitle(L.text("mobile.boards.report.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.boards.report.submit")) {
                        Task { await submit() }
                    }
                    .disabled(submitting || alreadyReported)
                }
            }
            .onAppear {
                alreadyReported = BoardsLogic.hasReported(postId: postId, commentId: commentId)
            }
        }
    }

    private func submit() async {
        guard let token = session.accessToken else { return }
        if BoardsLogic.hasReported(postId: postId, commentId: commentId) {
            alreadyReported = true
            return
        }
        submitting = true
        errorMessage = nil
        defer { submitting = false }
        let reasonLabel = L.text("mobile.boards.report.reason.\(reasonKey.rawValue)")
        let detail = details.trimmingCharacters(in: .whitespacesAndNewlines)
        let reason = detail.isEmpty ? reasonLabel : "\(reasonLabel) — \(detail)"
        do {
            _ = try await LMSAPI.reportBoardContent(
                courseCode: courseCode,
                boardId: boardId,
                postId: postId,
                commentId: commentId,
                reason: reason,
                accessToken: token
            )
            BoardsLogic.markReported(postId: postId, commentId: commentId)
            onSubmitted?()
            dismiss()
        } catch let err as APIError {
            if case let .httpStatus(code, message) = err, code == 429 {
                errorMessage = L.text("mobile.boards.report.rateLimited")
            } else if case let .httpStatus(_, message) = err {
                errorMessage = message ?? L.text("mobile.boards.report.error")
            } else {
                errorMessage = L.text("mobile.boards.report.error")
            }
        } catch {
            errorMessage = L.text("mobile.boards.report.error")
        }
    }
}
