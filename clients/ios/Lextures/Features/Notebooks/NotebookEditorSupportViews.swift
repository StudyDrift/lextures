import SwiftUI

/// One block row in edit mode: styled like the reading view, but editable in place.
struct NotebookEditBlockRow: View {
    @Environment(\.colorScheme) private var colorScheme
    @Binding var block: NotebookEditBlock
    let orderedNumber: Int
    var focus: FocusState<UUID?>.Binding
    var onTextChange: () -> Void
    var onToggleTask: () -> Void
    var onEditTaskDue: () -> Void
    var onEditDrawing: () -> Void
    var onDelete: () -> Void

    var body: some View {
        switch block.kind {
        case .paragraph:
            textField(font: .subheadline)

        case .heading(let level):
            textField(
                font: LexturesTheme.displayFont(level == 1 ? 26 : level == 2 ? 21 : 17),
                prompt: "Heading"
            )

        case .bullet:
            HStack(alignment: .firstTextBaseline, spacing: 10) {
                Circle()
                    .fill(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 5, height: 5)
                    .padding(.top, 6)
                textField(font: .subheadline, prompt: "List item")
            }
            .padding(.leading, 4)

        case .ordered:
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text("\(orderedNumber).")
                    .font(.subheadline.weight(.semibold).monospacedDigit())
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                textField(font: .subheadline, prompt: "List item")
            }
            .padding(.leading, 4)

        case .quote:
            HStack(alignment: .top, spacing: 10) {
                RoundedRectangle(cornerRadius: 2)
                    .fill(LexturesTheme.brandAmber)
                    .frame(width: 3)
                textField(font: .subheadline.italic(), prompt: "Quote")
            }
            .padding(.vertical, 2)

        case .code:
            TextField("Code", text: $block.text, axis: .vertical)
                .font(.caption.monospaced())
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .focused(focus, equals: block.id)
                .padding(12)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(LexturesTheme.sceneBackground(for: colorScheme).opacity(colorScheme == .dark ? 0.6 : 1))
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                        .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
                )

        case .divider:
            nonTextRow {
                Rectangle()
                    .fill(LexturesTheme.fieldBorder(for: colorScheme))
                    .frame(height: 1)
                    .padding(.vertical, 8)
            }

        case .task(_, let checked, let dueAt):
            HStack(alignment: .top, spacing: 10) {
                Button(action: onToggleTask) {
                    Image(systemName: checked ? "checkmark.square.fill" : "square")
                        .font(.system(size: 20))
                        .foregroundStyle(
                            checked
                                ? LexturesTheme.accent(for: colorScheme)
                                : LexturesTheme.textSecondary(for: colorScheme)
                        )
                }
                .buttonStyle(.plain)

                VStack(alignment: .leading, spacing: 3) {
                    textField(font: .subheadline, prompt: "Task")
                    Button(action: onEditTaskDue) {
                        HStack(spacing: 4) {
                            Image(systemName: "calendar")
                                .font(.caption2)
                            Text(dueLabel(dueAt))
                                .font(.caption)
                        }
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(10)
            .background(LexturesTheme.cardBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.9), lineWidth: 1)
            )

        case .image(let alt, let url):
            nonTextRow {
                AuthorizedNotebookImage(urlString: url, alt: alt)
            }

        case .drawing(let elementsJson):
            nonTextRow {
                NotebookDrawingBlockView(elementsJson: elementsJson, onTap: onEditDrawing)
            }
        }
    }

    private func textField(font: Font, prompt: String = "Write something…") -> some View {
        TextField(prompt, text: $block.text, axis: .vertical)
            .font(font)
            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            .focused(focus, equals: block.id)
            .onChange(of: block.text) {
                onTextChange()
            }
    }

    /// Non-text blocks get a small delete affordance since they can't be backspaced away.
    private func nonTextRow(@ViewBuilder content: () -> some View) -> some View {
        HStack(alignment: .top, spacing: 8) {
            content()
            Button(action: onDelete) {
                Image(systemName: "xmark.circle.fill")
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.7))
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Delete block")
        }
    }

    private func dueLabel(_ dueAt: String?) -> String {
        guard let dueAt, let date = LMSDates.parse(dueAt) else { return "Add due date" }
        return "Due \(date.formatted(date: .abbreviated, time: .omitted))"
    }
}

/// Due-date picker for a notebook task (set / change / remove).
struct NotebookDueDateSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme
    let task: ParsedNotebookTask
    var onSave: (Date?) -> Void

    @State private var date: Date = .now

    var body: some View {
        NavigationStack {
            VStack(spacing: 12) {
                if !task.text.isEmpty {
                    Text(task.text)
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .lineLimit(2)
                        .padding(.horizontal, 20)
                }

                DatePicker("Due date", selection: $date, displayedComponents: .date)
                    .datePickerStyle(.graphical)
                    .tint(LexturesTheme.accent(for: colorScheme))
                    .padding(.horizontal, 12)

                if task.dueAt != nil {
                    Button(role: .destructive) {
                        onSave(nil)
                        dismiss()
                    } label: {
                        Label("Remove due date", systemImage: "calendar.badge.minus")
                            .font(.subheadline.weight(.semibold))
                    }
                    .padding(.bottom, 8)
                }
            }
            .navigationTitle("Due date")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Save") {
                        onSave(date)
                        dismiss()
                    }
                    .fontWeight(.semibold)
                    .tint(LexturesTheme.accent(for: colorScheme))
                }
            }
            .onAppear {
                if let dueAt = task.dueAt, let parsed = LMSDates.parse(dueAt) {
                    date = parsed
                }
            }
        }
    }
}
