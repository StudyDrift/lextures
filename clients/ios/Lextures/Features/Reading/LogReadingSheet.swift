import SwiftUI

struct LogReadingDraft: Equatable {
    var bookId: String?
    var bookTitle: String = ""
    var logDate: String = ReadingLogic.todayISO()
    var pagesRead: String = ""
    var reflection: String = ""
}

struct LogReadingSheet: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let initial: LogReadingDraft
    let saving: Bool
    let errorMessage: String?
    let onSave: (LogReadingDraft) -> Void

    @State private var draft: LogReadingDraft

    init(
        initial: LogReadingDraft = LogReadingDraft(),
        saving: Bool,
        errorMessage: String?,
        onSave: @escaping (LogReadingDraft) -> Void
    ) {
        self.initial = initial
        self.saving = saving
        self.errorMessage = errorMessage
        self.onSave = onSave
        _draft = State(initialValue: initial)
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }

                        VStack(alignment: .leading, spacing: 6) {
                            Text(L.text("mobile.reading.logBookTitle"))
                                .font(.subheadline.weight(.semibold))
                            TextField(L.text("mobile.reading.logBookPlaceholder"), text: $draft.bookTitle)
                                .textFieldStyle(.roundedBorder)
                                .disabled(draft.bookId != nil)
                                .accessibilityLabel(L.text("mobile.reading.logBookTitle"))
                        }

                        VStack(alignment: .leading, spacing: 6) {
                            Text(L.text("mobile.reading.logDate"))
                                .font(.subheadline.weight(.semibold))
                            TextField(L.text("mobile.reading.logDate"), text: $draft.logDate)
                                .textFieldStyle(.roundedBorder)
                                .keyboardType(.numbersAndPunctuation)
                                .accessibilityLabel(L.text("mobile.reading.logDate"))
                        }

                        VStack(alignment: .leading, spacing: 6) {
                            Text(L.text("mobile.reading.logPages"))
                                .font(.subheadline.weight(.semibold))
                            TextField(L.text("mobile.reading.logPagesPlaceholder"), text: $draft.pagesRead)
                                .textFieldStyle(.roundedBorder)
                                .keyboardType(.numberPad)
                                .accessibilityLabel(L.text("mobile.reading.logPages"))
                        }

                        VStack(alignment: .leading, spacing: 6) {
                            Text(L.text("mobile.reading.logReflection"))
                                .font(.subheadline.weight(.semibold))
                            TextField(
                                L.text("mobile.reading.logReflectionPlaceholder"),
                                text: $draft.reflection,
                                axis: .vertical
                            )
                            .lineLimit(2 ... 4)
                            .textFieldStyle(.roundedBorder)
                            .accessibilityLabel(L.text("mobile.reading.logReflection"))
                        }

                        Button {
                            onSave(draft)
                        } label: {
                            Text(saving ? L.text("mobile.reading.saving") : L.text("mobile.reading.logSave"))
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.borderedProminent)
                        .disabled(saving || !ReadingLogic.logEntryValid(
                            bookTitle: draft.bookTitle,
                            bookId: draft.bookId,
                            logDate: draft.logDate
                        ))
                    }
                    .padding(16)
                }
            }
            .navigationTitle(L.text("mobile.reading.logTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.ia.close")) { dismiss() }
                }
            }
        }
    }
}