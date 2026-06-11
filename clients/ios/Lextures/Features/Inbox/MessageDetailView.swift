import SwiftUI

struct MessageDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    let message: MailboxMessage
    var onChanged: () -> Void

    @State private var starred: Bool
    @State private var errorMessage: String?
    @State private var replying = false

    init(message: MailboxMessage, onChanged: @escaping () -> Void) {
        self.message = message
        self.onChanged = onChanged
        _starred = State(initialValue: message.starred)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    Text(message.subject.isEmpty ? "(no subject)" : message.subject)
                        .font(.title3.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

                    LMSCard {
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(message.from.name.isEmpty ? message.from.email : message.from.name)
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                Text(message.from.email)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                if !message.to.isEmpty {
                                    Text("To: \(message.to)")
                                        .font(.caption)
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                }
                            }
                            Spacer()
                            Text(LMSDates.shortDateTime(message.sentAt))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }

                    LMSCard {
                        Text(message.body.isEmpty ? message.snippet : message.body)
                            .font(.body)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            .textSelection(.enabled)
                    }
                }
                .padding(16)
            }
        }
        .navigationTitle("Message")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItemGroup(placement: .topBarTrailing) {
                Button {
                    Task { await toggleStar() }
                } label: {
                    Image(systemName: starred ? "star.fill" : "star")
                        .foregroundStyle(starred ? .yellow : LexturesTheme.primary)
                }
                Button {
                    replying = true
                } label: {
                    Image(systemName: "arrowshape.turn.up.left")
                }
                Button(role: .destructive) {
                    Task { await moveToTrash() }
                } label: {
                    Image(systemName: "trash")
                }
            }
        }
        .sheet(isPresented: $replying) {
            ComposeMessageView(
                initialTo: message.from.email,
                initialSubject: message.subject.hasPrefix("Re:") ? message.subject : "Re: \(message.subject)"
            ) { sent in
                if sent { onChanged() }
            }
        }
        .task {
            await markRead()
        }
    }

    private func markRead() async {
        guard !message.read, let token = session.accessToken else { return }
        try? await LMSAPI.patchMailbox(messageId: message.id, patch: .init(read: true), accessToken: token)
        onChanged()
    }

    private func toggleStar() async {
        guard let token = session.accessToken else { return }
        do {
            try await LMSAPI.patchMailbox(messageId: message.id, patch: .init(starred: !starred), accessToken: token)
            starred.toggle()
            onChanged()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not update the message."
        }
    }

    private func moveToTrash() async {
        guard let token = session.accessToken else { return }
        do {
            try await LMSAPI.patchMailbox(messageId: message.id, patch: .init(folder: "trash"), accessToken: token)
            onChanged()
            dismiss()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not move the message to trash."
        }
    }
}
