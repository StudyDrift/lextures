import SwiftUI

/// Mailbox folders, search, message list, and compose (parity with web inbox).
struct InboxView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var folder: MailboxFolder = .inbox
    @State private var messages: [MailboxMessage] = []
    @State private var searchText = ""
    @State private var errorMessage: String?
    @State private var loading = false
    @State private var composing = false

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                VStack(spacing: 0) {
                    folderPicker

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                            .padding(.horizontal, 16)
                            .padding(.top, 8)
                    }

                    if loading && messages.isEmpty {
                        Spacer()
                        ProgressView()
                        Spacer()
                    } else if messages.isEmpty {
                        Spacer()
                        LMSEmptyState(
                            systemImage: folder.systemImage,
                            title: searchText.isEmpty ? "Nothing in \(folder.label.lowercased())" : "No matching messages",
                            message: searchText.isEmpty
                                ? "Messages will appear here."
                                : "Try different keywords, or clear search."
                        )
                        Spacer()
                    } else {
                        messageList
                    }
                }
            }
            .navigationTitle("Inbox")
            .navigationBarTitleDisplayMode(.inline)
            .globalDrawerToolbar()
            .searchable(text: $searchText, prompt: "Search mail")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        composing = true
                    } label: {
                        Image(systemName: "square.and.pencil")
                    }
                }
            }
            .sheet(isPresented: $composing) {
                ComposeMessageView { sent in
                    if sent {
                        Task { await load() }
                    }
                }
            }
            .navigationDestination(for: MailboxMessage.self) { message in
                MessageDetailView(message: message) {
                    Task {
                        await load()
                        await refreshUnread()
                    }
                }
            }
            .task { await load() }
            .task(id: searchText) {
                try? await Task.sleep(for: .milliseconds(300))
                guard !Task.isCancelled else { return }
                await load()
            }
            .onChange(of: folder) {
                Task { await load() }
            }
        }
    }

    private var folderPicker: some View {
        LMSSegmentedChips(
            options: MailboxFolder.allCases,
            selection: $folder,
            label: \.label
        )
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }

    private var messageList: some View {
        List {
            ForEach(messages) { message in
                NavigationLink(value: message) {
                    MessageRow(message: message)
                }
                .listRowBackground(LexturesTheme.cardBackground(for: colorScheme))
                .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                    if folder != .trash {
                        Button(role: .destructive) {
                            Task { await patch(message, .init(folder: "trash")) }
                        } label: {
                            Label("Trash", systemImage: "trash")
                        }
                    } else {
                        Button {
                            Task { await patch(message, .init(folder: "inbox")) }
                        } label: {
                            Label("Restore", systemImage: "tray.and.arrow.down")
                        }
                    }
                    Button {
                        Task { await patch(message, .init(starred: !message.starred)) }
                    } label: {
                        Label(message.starred ? "Unstar" : "Star", systemImage: "star")
                    }
                    .tint(.yellow)
                }
            }
        }
        .listStyle(.plain)
        .scrollContentBackground(.hidden)
        .refreshable {
            await load()
            await refreshUnread()
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            messages = try await LMSAPI.fetchMailboxMessages(folder: folder, query: searchText, accessToken: token)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load messages."
        }
    }

    private func patch(_ message: MailboxMessage, _ patch: LMSAPI.MailboxPatch) async {
        guard let token = session.accessToken else { return }
        do {
            try await LMSAPI.patchMailbox(messageId: message.id, patch: patch, accessToken: token)
            await load()
            await refreshUnread()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not update the message."
        }
    }

    private func refreshUnread() async {
        guard let token = session.accessToken else { return }
        shell.unreadInbox = (try? await LMSAPI.fetchUnreadInboxCount(accessToken: token)) ?? shell.unreadInbox
    }
}

struct MessageRow: View {
    @Environment(\.colorScheme) private var colorScheme
    let message: MailboxMessage

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            Circle()
                .fill(LexturesTheme.coverGradient(for: message.from.email))
                .frame(width: 40, height: 40)
                .overlay(
                    Text(initials(message.from.name.isEmpty ? message.from.email : message.from.name))
                        .font(.caption.weight(.bold))
                        .foregroundStyle(.white)
                )

            VStack(alignment: .leading, spacing: 3) {
                HStack {
                    Text(message.from.name.isEmpty ? message.from.email : message.from.name)
                        .font(.subheadline.weight(message.read ? .regular : .semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .lineLimit(1)
                    Spacer()
                    Text(LMSDates.relative(message.sentAt))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                HStack(spacing: 4) {
                    if message.starred {
                        Image(systemName: "star.fill")
                            .font(.caption2)
                            .foregroundStyle(.yellow)
                    }
                    if message.hasAttachment {
                        Image(systemName: "paperclip")
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Text(message.subject.isEmpty ? "(no subject)" : message.subject)
                        .font(.subheadline.weight(message.read ? .regular : .semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .lineLimit(1)
                }
                Text(message.snippet)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .lineLimit(2)
            }
        }
        .padding(.vertical, 4)
    }

    private func initials(_ name: String) -> String {
        let parts = name.split(separator: " ")
        if parts.count >= 2, let firstChar = parts.first?.first, let lastChar = parts.last?.first {
            return String([firstChar, lastChar]).uppercased()
        }
        return String(name.prefix(2)).uppercased()
    }
}
