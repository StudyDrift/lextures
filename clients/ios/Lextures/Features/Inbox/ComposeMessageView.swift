import SwiftUI

struct ComposeMessageView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    var initialTo = ""
    var initialSubject = ""
    var onDone: (Bool) -> Void = { _ in }

    @State private var to = ""
    @State private var subject = ""
    @State private var bodyText = ""
    @State private var sending = false
    @State private var errorMessage: String?
    @State private var seeded = false

    private var canSend: Bool {
        !to.trimmingCharacters(in: .whitespaces).isEmpty
            && !(subject.trimmingCharacters(in: .whitespaces).isEmpty
                && bodyText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            && !sending
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                VStack(spacing: 12) {
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    AuthTextField(
                        title: "To",
                        text: $to,
                        placeholder: "name@school.edu",
                        keyboard: .emailAddress,
                        textContentType: .emailAddress
                    )

                    AuthTextField(
                        title: "Subject",
                        text: $subject,
                        placeholder: "Subject",
                        autocapitalization: .sentences
                    )

                    DictationField(
                        title: "Message",
                        text: $bodyText,
                        placeholder: "Write your message…"
                    )

                    Spacer()
                }
                .padding(16)
            }
            .navigationTitle("New message")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Cancel") {
                        onDone(false)
                        dismiss()
                    }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        Task { await send() }
                    } label: {
                        if sending {
                            ProgressView()
                        } else {
                            Text("Send").fontWeight(.semibold)
                        }
                    }
                    .disabled(!canSend)
                }
            }
            .onAppear {
                guard !seeded else { return }
                seeded = true
                to = initialTo
                subject = initialSubject
            }
        }
    }

    private func send() async {
        guard let token = session.accessToken else { return }
        sending = true
        errorMessage = nil
        defer { sending = false }
        do {
            try await LMSAPI.sendMessage(
                .init(
                    toEmail: to.trimmingCharacters(in: .whitespaces),
                    subject: subject.trimmingCharacters(in: .whitespaces),
                    body: bodyText
                ),
                accessToken: token
            )
            onDone(true)
            dismiss()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not send the message."
        }
    }
}
