import SwiftUI

/// Compose a new thread or reply (text + offline queue).
struct PostComposerView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let mode: PostComposerMode
    let course: CourseSummary
    var threadId: String?
    var onPosted: (String) -> Void

    @State private var title = ""
    @State private var bodyText = ""
    @State private var sending = false
    @State private var errorMessage: String?

    private var isNewThread: Bool {
        if case .newThread = mode { return true }
        return false
    }

    private var canSend: Bool {
        !sending && (isNewThread ? !title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty : true)
            && !DiscussionLogic.isBodyEmpty(bodyText)
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                VStack(spacing: 12) {
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if isNewThread {
                        AuthTextField(
                            title: L.text("mobile.discussions.threadTitle"),
                            text: $title,
                            placeholder: L.text("mobile.discussions.threadTitlePlaceholder"),
                            autocapitalization: .sentences
                        )
                    }

                    DictationField(
                        title: L.text("mobile.discussions.message"),
                        text: $bodyText,
                        placeholder: L.text("mobile.discussions.messagePlaceholder")
                    )

                    Spacer()
                }
                .padding(16)
            }
            .navigationTitle(isNewThread ? L.text("mobile.discussions.newThread") : L.text("mobile.discussions.reply"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button(L.text("mobile.discussions.cancel")) { dismiss() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        Task { await send() }
                    } label: {
                        if sending {
                            ProgressView()
                        } else {
                            Text(L.text("mobile.discussions.post")).fontWeight(.semibold)
                        }
                    }
                    .disabled(!canSend)
                }
            }
        }
    }

    private func send() async {
        guard let token = session.accessToken else { return }
        sending = true
        errorMessage = nil
        defer { sending = false }

        let bodyData: Data
        do {
            bodyData = try DiscussionLogic.encodeBody(text: bodyText)
        } catch {
            errorMessage = L.text("mobile.discussions.postError")
            return
        }

        if !NetworkMonitor.shared.isOnline {
            await sendOffline(bodyData: bodyData, accessToken: token)
            return
        }

        do {
            switch mode {
            case let .newThread(forumId):
                let thread = try await LMSAPI.createDiscussionThread(
                    courseCode: course.courseCode,
                    forumId: forumId,
                    title: title.trimmingCharacters(in: .whitespacesAndNewlines),
                    body: bodyData,
                    accessToken: token
                )
                dismiss()
                onPosted(thread.id)
            case let .reply(parentPostId):
                guard let threadId else { return }
                _ = try await LMSAPI.createDiscussionPost(
                    courseCode: course.courseCode,
                    threadId: threadId,
                    parentPostId: parentPostId,
                    body: bodyData,
                    accessToken: token
                )
                dismiss()
                onPosted(threadId)
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.discussions.postError")
        }
    }

    private func sendOffline(bodyData: Data, accessToken: String) async {
        do {
            switch mode {
            case let .newThread(forumId):
                let payload = CreateDiscussionThreadRequest(
                    title: title.trimmingCharacters(in: .whitespacesAndNewlines),
                    body: bodyData
                )
                let path = "/api/v1/courses/\(course.courseCode)/forums/\(forumId)/threads"
                _ = try await offline.enqueueMutation(
                    method: "POST",
                    path: path,
                    body: payload,
                    label: L.text("mobile.discussions.newThread"),
                    accessToken: accessToken,
                    preferQueue: true
                )
                dismiss()
                onPosted("")
            case let .reply(parentPostId):
                guard let threadId else { return }
                let payload = CreateDiscussionPostRequest(parentPostId: parentPostId, body: bodyData)
                let path = "/api/v1/courses/\(course.courseCode)/discussion-threads/\(threadId)/posts"
                _ = try await offline.enqueueMutation(
                    method: "POST",
                    path: path,
                    body: payload,
                    label: L.text("mobile.discussions.reply"),
                    accessToken: accessToken,
                    preferQueue: true
                )
                dismiss()
                onPosted(threadId)
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.discussions.postError")
        }
    }
}
