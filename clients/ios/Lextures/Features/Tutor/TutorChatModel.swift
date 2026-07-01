import Foundation

@MainActor
@Observable
final class TutorChatModel {
    var messages: [TutorDisplayMessage] = []
    var sessions: [TutorSessionSummary] = []
    var activeSessionId: String?
    var studyBuddySessionId: String?
    var input = ""
    var streamingText = ""
    var streaming = false
    var loading = true
    var errorMessage: String?
    var showDisclosure = false
    var tokensUsed = 0
    var tokenLimit = 0
    var selectedCourse: CourseSummary?
    var courses: [CourseSummary] = []
    var sentContext = false

    private var sendTask: Task<Void, Never>?
    private let streamClient = TutorStreamClient()

    func cancelStreams() {
        sendTask?.cancel()
    }

    func load(
        mode: TutorChatMode,
        accessToken: String?,
        platform: MobilePlatformFeatures
    ) async {
        loading = true
        errorMessage = nil
        defer { loading = false }
        guard let token = accessToken else { return }

        showDisclosure = !TutorLogic.hasAcceptedDisclosure(courseCode: courseCode(for: mode))

        switch mode {
        case .askAi:
            do {
                courses = try await LMSAPI.fetchCourses(accessToken: token)
                if platform.ragNotebookEnabled, !notebookInputs(token: token).isEmpty {
                    selectedCourse = nil
                } else if let first = courses.first(where: { $0.isAiTutorEnabled }) {
                    selectedCourse = first
                    await loadCourseTutor(course: first, token: token, persistent: platform.ffPersistentTutor)
                }
            } catch {
                errorMessage = mapError(error)
            }
        case .course(let course, _):
            await loadCourseTutor(course: course, token: token, persistent: platform.ffPersistentTutor)
        }
    }

    func loadCourseChat(accessToken: String?, persistent: Bool) async {
        guard let token = accessToken, let course = selectedCourse else { return }
        loading = true
        defer { loading = false }
        await loadCourseTutor(course: course, token: token, persistent: persistent)
    }

    func sendMessage(
        mode: TutorChatMode,
        accessToken: String?,
        platform: MobilePlatformFeatures
    ) async {
        let raw = input.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !raw.isEmpty, let token = accessToken else { return }
        guard NetworkMonitor.shared.isOnline else {
            errorMessage = L.text("mobile.tutor.offline")
            return
        }

        if case .askAi = mode, platform.ragNotebookEnabled, selectedCourse == nil {
            await sendNotebookQuery(raw, token: token)
            return
        }

        guard let code = courseCode(for: mode) else { return }

        let includeContext: Bool
        let itemTitle: String?
        let itemKind: String?
        if case .course(_, let item) = mode {
            includeContext = !sentContext
            itemTitle = item?.title
            itemKind = item?.kind
            sentContext = true
        } else {
            includeContext = false
            itemTitle = nil
            itemKind = nil
        }

        let text = TutorLogic.messageWithContext(
            raw,
            itemTitle: itemTitle,
            itemKind: itemKind,
            includeContext: includeContext
        )

        input = ""
        errorMessage = nil
        streaming = true
        streamingText = ""
        messages.append(TutorDisplayMessage(role: "user", content: raw))

        sendTask = Task {
            var fullText = ""
            var citations: [TutorCitation] = []
            do {
                let stream: AsyncThrowingStream<TutorStreamEvent, Error>
                if platform.aiStudyBuddyEnabled, case .askAi = mode {
                    stream = LMSAPI.studyBuddyMessageStream(
                        courseCode: code,
                        message: text,
                        sessionId: studyBuddySessionId,
                        accessToken: token
                    )
                } else if platform.ffPersistentTutor, let sessionId = activeSessionId {
                    stream = LMSAPI.tutorSessionMessageStream(
                        courseCode: code,
                        sessionId: sessionId,
                        content: text,
                        accessToken: token
                    )
                } else {
                    stream = LMSAPI.tutorMessageStream(courseCode: code, message: text, accessToken: token)
                }

                for try await event in stream {
                    guard !Task.isCancelled else { break }
                    switch event {
                    case .content(let chunk):
                        fullText += chunk
                        streamingText = fullText
                    case .error(let message):
                        errorMessage = message
                    case .done(_, _, let sessionId, let doneCitations):
                        citations = doneCitations
                        if let sessionId { studyBuddySessionId = sessionId }
                    }
                }

                appendAssistantMessage(fullText: fullText, citations: citations)
            } catch {
                if !streamingText.isEmpty {
                    messages.append(TutorDisplayMessage(role: "assistant", content: streamingText))
                }
                errorMessage = mapError(error)
            }
            streamingText = ""
            streaming = false
        }
    }

    func stopStreaming() {
        sendTask?.cancel()

        if !streamingText.isEmpty {
            messages.append(TutorDisplayMessage(role: "assistant", content: streamingText))
        }
        streamingText = ""
        streaming = false
    }

    func startNewConversation(courseCode: String, accessToken: String?) async {
        guard let token = accessToken else { return }
        loading = true
        defer { loading = false }
        do {
            let created = try await LMSAPI.createTutorSession(courseCode: courseCode, accessToken: token)
            sessions.insert(created, at: 0)
            activeSessionId = created.id
            messages = []
            sentContext = false
            errorMessage = nil
        } catch {
            errorMessage = mapError(error)
        }
    }

    func resetConversation(courseCode: String, accessToken: String?, persistent: Bool) async {
        guard let token = accessToken else { return }
        do {
            if persistent, let sessionId = activeSessionId {
                try await LMSAPI.deleteTutorSession(courseCode: courseCode, sessionId: sessionId, accessToken: token)
                await startNewConversation(courseCode: courseCode, accessToken: token)
            } else {
                try await LMSAPI.resetTutorConversation(courseCode: courseCode, accessToken: token)
                messages = []
                sentContext = false
            }
            streamingText = ""
            errorMessage = nil
        } catch {
            errorMessage = mapError(error)
        }
    }

    func acceptDisclosure(courseCode: String?) {
        TutorLogic.acceptDisclosure(courseCode: courseCode)
        showDisclosure = false
    }

    private func appendAssistantMessage(fullText: String, citations: [TutorCitation]) {
        let answer = fullText.isEmpty ? streamingText : fullText
        guard !answer.isEmpty else { return }
        messages.append(TutorDisplayMessage(role: "assistant", content: answer, citations: citations))
    }

    private func loadCourseTutor(course: CourseSummary, token: String, persistent: Bool) async {
        sentContext = false
        do {
            if persistent {
                var list = try await LMSAPI.fetchTutorSessions(courseCode: course.courseCode, accessToken: token)
                if list.isEmpty {
                    list = [try await LMSAPI.createTutorSession(courseCode: course.courseCode, accessToken: token)]
                }
                sessions = list
                activeSessionId = list.first?.id
                if let sessionId = activeSessionId {
                    let detail = try await LMSAPI.fetchTutorSession(
                        courseCode: course.courseCode,
                        sessionId: sessionId,
                        accessToken: token
                    )
                    messages = detail.messages
                        .filter { $0.role != "system" }
                        .map {
                            TutorDisplayMessage(
                                id: $0.id ?? UUID().uuidString.lowercased(),
                                role: $0.role,
                                content: $0.content,
                                citations: $0.citations ?? []
                            )
                        }
                }
            } else {
                let conv = try await LMSAPI.fetchTutorConversation(courseCode: course.courseCode, accessToken: token)
                tokensUsed = conv.tokensUsed
                tokenLimit = conv.tokenLimit
                messages = conv.messages.map {
                    TutorDisplayMessage(role: $0.role, content: $0.content, citations: $0.citations ?? [])
                }
            }
        } catch {
            errorMessage = mapError(error)
        }
    }

    private func sendNotebookQuery(_ question: String, token: String) async {
        let notebooks = notebookInputs(token: token)
        guard !notebooks.isEmpty else {
            errorMessage = L.text("mobile.tutor.noNotebooks")
            return
        }
        input = ""
        loading = true
        messages.append(TutorDisplayMessage(role: "user", content: question))
        defer { loading = false }
        do {
            let response = try await LMSAPI.queryNotebooks(
                NotebookRagQueryRequest(question: question, notebooks: notebooks),
                accessToken: token
            )
            messages.append(TutorDisplayMessage(role: "assistant", content: response.answerMarkdown))
        } catch {
            errorMessage = mapError(error)
        }
    }

    private func notebookInputs(token: String) -> [NotebookRagNotebookInput] {
        let store = NotebookStore(accessToken: token)
        return store.allCourseCodes().compactMap { code in
            let notebook = store.load(courseCode: code)
            let body = notebook.previewText
            guard !body.isEmpty else { return nil }
            let title = notebook.courseTitle ?? (code == NotebookStore.globalKey ? NotebookStore.globalTitle : code)
            return NotebookRagNotebookInput(courseCode: code, courseTitle: title, markdown: body)
        }
    }

    private func courseCode(for mode: TutorChatMode) -> String? {
        switch mode {
        case .course(let course, _): return course.courseCode
        case .askAi: return selectedCourse?.courseCode
        }
    }

    private func mapError(_ error: Error) -> String {
        if let api = error as? APIError, case .httpStatus(let code, let message) = api {
            return TutorLogic.gracefulHttpMessage(statusCode: code, body: message)
        }
        return (error as? LocalizedError)?.errorDescription ?? L.text("mobile.tutor.sendError")
    }
}