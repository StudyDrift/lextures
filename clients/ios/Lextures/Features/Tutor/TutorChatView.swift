import SwiftUI

/// Course-scoped AI tutor, study buddy, or standalone Ask-AI chat (M7.2).
struct TutorChatView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let mode: TutorChatMode

    @State private var model = TutorChatModel()

    private var courseCode: String? {
        switch mode {
        case .course(let course, _): return course.courseCode
        case .askAi: return model.selectedCourse?.courseCode
        }
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                VStack(spacing: 0) {
                    if !NetworkMonitor.shared.isOnline { OfflineBanner() }
                    if model.showDisclosure { disclosureBanner }
                    if model.tokenLimit > 0 {
                        Text(TutorLogic.budgetLabel(used: model.tokensUsed, limit: model.tokenLimit))
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.horizontal, 16)
                            .padding(.top, 8)
                    }
                    if let errorMessage = model.errorMessage {
                        LMSErrorBanner(message: errorMessage)
                            .padding(.horizontal, 16)
                            .padding(.top, 8)
                    }
                    if case .askAi = mode, model.selectedCourse == nil, !model.loading {
                        askAiCoursePicker
                    } else {
                        messageList
                        composer
                    }
                }
            }
            .navigationTitle(mode == .askAi ? L.text("mobile.tutor.askAi") : L.text("mobile.tutor.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar { toolbarContent }
            .task {
                await model.load(
                    mode: mode,
                    accessToken: session.accessToken,
                    platform: shell.platformFeatures
                )
            }
            .onDisappear { model.cancelStreams() }
        }
    }

    @ToolbarContentBuilder
    private var toolbarContent: some ToolbarContent {
        ToolbarItem(placement: .topBarLeading) {
            Button(L.text("mobile.tutor.close")) { dismiss() }
        }
        ToolbarItemGroup(placement: .topBarTrailing) {
            if case .course = mode, shell.platformFeatures.ffPersistentTutor, courseCode != nil {
                Button {
                    Task { await model.startNewConversation(courseCode: courseCode!, accessToken: session.accessToken) }
                } label: {
                    Image(systemName: "plus")
                }
                .accessibilityLabel(L.text("mobile.tutor.newConversation"))
            }
            if courseCode != nil, !ragAskMode {
                Button {
                    Task {
                        await model.resetConversation(
                            courseCode: courseCode!,
                            accessToken: session.accessToken,
                            persistent: shell.platformFeatures.ffPersistentTutor
                        )
                    }
                } label: {
                    Image(systemName: "trash")
                }
                .accessibilityLabel(L.text("mobile.tutor.reset"))
            }
        }
    }

    private var ragAskMode: Bool {
        if case .askAi = mode, shell.platformFeatures.ragNotebookEnabled { return true }
        return false
    }

    @ViewBuilder
    private var disclosureBanner: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.tutor.disclosureTitle"))
                    .font(.subheadline.weight(.semibold))
                Text(L.text("mobile.tutor.disclosureBody"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Button(L.text("mobile.tutor.disclosureAccept")) {
                    model.acceptDisclosure(courseCode: courseCode)
                }
                .font(.subheadline.weight(.semibold))
            }
        }
        .padding(.horizontal, 16)
        .padding(.top, 8)
    }

    @ViewBuilder
    private var askAiCoursePicker: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.tutor.askAiCourseHint"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                if shell.platformFeatures.ragNotebookEnabled {
                    Text(L.text("mobile.tutor.askAiNotebookHint"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                ForEach(model.courses.filter { $0.isAiTutorEnabled || shell.platformFeatures.aiStudyBuddyEnabled }) { course in
                    Button {
                        model.selectedCourse = course
                        Task {
                            await model.loadCourseChat(
                                accessToken: session.accessToken,
                                persistent: shell.platformFeatures.ffPersistentTutor
                            )
                        }
                    } label: {
                        LMSCard {
                            Text(course.displayTitle)
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                .frame(maxWidth: .infinity, alignment: .leading)
                        }
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(16)
        }
    }

    @ViewBuilder
    private var messageList: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 12) {
                    if model.loading && model.messages.isEmpty {
                        LMSSkeletonList(count: 3)
                    }
                    ForEach(model.messages) { message in
                        TutorMessageBubble(message: message)
                            .id(message.id)
                    }
                    if !model.streamingText.isEmpty {
                        TutorMessageBubble(
                            message: TutorDisplayMessage(
                                role: "assistant",
                                content: model.streamingText,
                                isStreaming: true
                            )
                        )
                        .id("streaming")
                    }
                }
                .padding(16)
            }
            .onChange(of: model.messages.count) { _, _ in scrollToBottom(proxy) }
            .onChange(of: model.streamingText) { _, _ in scrollToBottom(proxy) }
        }
    }

    @ViewBuilder
    private var composer: some View {
        HStack(alignment: .bottom, spacing: 10) {
            TextField(L.text("mobile.tutor.placeholder"), text: $model.input, axis: .vertical)
                .lineLimit(1 ... 5)
                .textFieldStyle(.plain)
                .padding(10)
                .background(LexturesTheme.cardBackground(for: colorScheme))
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                .disabled(!NetworkMonitor.shared.isOnline || model.loading)
            if model.streaming {
                Button(L.text("mobile.tutor.stop")) { model.stopStreaming() }
                    .buttonStyle(.bordered)
            } else {
                Button(L.text("mobile.tutor.send")) {
                    Task {
                        await model.sendMessage(
                            mode: mode,
                            accessToken: session.accessToken,
                            platform: shell.platformFeatures
                        )
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(
                    model.input.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                        || !NetworkMonitor.shared.isOnline
                        || model.loading
                )
            }
        }
        .padding(16)
    }

    private func scrollToBottom(_ proxy: ScrollViewProxy) {
        withAnimation(.easeOut(duration: 0.2)) {
            if !model.streamingText.isEmpty {
                proxy.scrollTo("streaming", anchor: .bottom)
            } else if let last = model.messages.last?.id {
                proxy.scrollTo(last, anchor: .bottom)
            }
        }
    }
}

private struct TutorMessageBubble: View {
    @Environment(\.colorScheme) private var colorScheme
    let message: TutorDisplayMessage

    var body: some View {
        let isUser = message.role == "user"
        HStack {
            if isUser { Spacer(minLength: 40) }
            VStack(alignment: .leading, spacing: 6) {
                if isUser {
                    Text(message.content)
                        .font(.body)
                        .foregroundStyle(.white)
                } else {
                    CourseMarkdownContentView(markdown: message.content)
                    if !message.citations.isEmpty {
                        TutorFlowLayout(spacing: 6) {
                            ForEach(message.citations, id: \.chunkId) { citation in
                                Text(citation.title ?? L.text("mobile.tutor.source"))
                                    .font(.caption2.weight(.medium))
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 4)
                                    .background(LexturesTheme.accent(for: colorScheme).opacity(0.15))
                                    .clipShape(Capsule())
                            }
                        }
                    }
                }
            }
            .padding(12)
            .background(isUser ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.cardBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
            if !isUser { Spacer(minLength: 40) }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(isUser ? L.text("mobile.tutor.you") : L.text("mobile.tutor.assistant"))
        .accessibilityValue(message.content)
    }
}