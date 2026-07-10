import SwiftUI

/// Native share-feedback sheet (FB3).
struct ShareFeedbackView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Binding var isPresented: Bool
    var onSuccess: () -> Void

    @State private var message = ""
    @State private var category = ""
    @State private var submitting = false
    @State private var errorMessage: String?

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var canSend: Bool { FeedbackLogic.messageValid(message) && !submitting && isOnline }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text(L.text("mobile.feedback.privacy"))
                        .font(.footnote)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                Section(L.text("mobile.feedback.message.label")) {
                    TextEditor(text: $message)
                        .frame(minHeight: 120)
                        .accessibilityLabel(L.text("mobile.feedback.message.label"))
                        .accessibilityHint(L.text("mobile.feedback.message.placeholder"))
                    Text(
                        L.format(
                            "mobile.feedback.message.counter",
                            FeedbackLogic.trimmedMessageLength(message),
                            FeedbackLogic.maxMessageLength
                        )
                    )
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .frame(maxWidth: .infinity, alignment: .trailing)
                    .accessibilityLabel(
                        L.format(
                            "mobile.feedback.message.counter",
                            FeedbackLogic.trimmedMessageLength(message),
                            FeedbackLogic.maxMessageLength
                        )
                    )
                }

                Section(L.text("mobile.feedback.category.label")) {
                    Picker(L.text("mobile.feedback.category.label"), selection: $category) {
                        Text(L.text("mobile.feedback.category.none")).tag("")
                        ForEach(FeedbackLogic.categories, id: \.self) { value in
                            Text(L.dynamicText(FeedbackLogic.categoryLabelKey(value))).tag(value)
                        }
                    }
                    .accessibilityLabel(L.text("mobile.feedback.category.label"))
                }

                if let errorMessage {
                    Section {
                        Text(errorMessage)
                            .font(.footnote)
                            .foregroundStyle(LexturesTheme.coral)
                            .accessibilityLabel(errorMessage)
                    }
                } else if !isOnline {
                    Section {
                        Text(L.text("mobile.feedback.offline"))
                            .font(.footnote)
                            .foregroundStyle(LexturesTheme.coral)
                    }
                }
            }
            .navigationTitle(L.text("mobile.feedback.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.feedback.cancel")) {
                        isPresented = false
                    }
                    .disabled(submitting)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button {
                        Task { await submit() }
                    } label: {
                        if submitting {
                            ProgressView()
                        } else {
                            Text(L.text("mobile.feedback.send"))
                        }
                    }
                    .disabled(!canSend)
                    .accessibilityLabel(L.text("mobile.feedback.send"))
                }
            }
        }
        .presentationDetents([.medium, .large])
        .onAppear {
            errorMessage = isOnline ? nil : L.text("mobile.feedback.offline")
        }
    }

    private func submit() async {
        guard let token = session.accessToken else { return }
        guard isOnline else {
            errorMessage = L.text("mobile.feedback.offline")
            return
        }
        submitting = true
        errorMessage = nil
        defer { submitting = false }

        let viewport = await MainActor.run {
            let size = UIScreen.main.bounds.size
            return "\(Int(size.width))x\(Int(size.height))"
        }
        let request = FeedbackLogic.buildSubmitRequest(
            message: message,
            category: category,
            route: "profile",
            locale: LocalePreferences.shared.effectiveTag,
            viewport: viewport
        )

        do {
            _ = try await LMSAPI.submitFeedback(body: request, accessToken: token)
            isPresented = false
            onSuccess()
        } catch {
            let outcome = FeedbackLogic.mapSubmitError(error, isOnline: isOnline)
            errorMessage = L.dynamicText(FeedbackLogic.errorMessageKey(for: outcome))
        }
    }
}
