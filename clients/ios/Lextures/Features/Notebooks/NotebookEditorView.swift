import SwiftUI

/// Notion-style page editor: a large editable title plus an always-editable block list
/// (text, tasks, drawings, …). There is no separate read/edit mode — tap any block to
/// edit it, toggle tasks in place, tap drawings to open the whiteboard. The `/` command
/// menu and insert toolbar ride above the keyboard while a block is focused.
struct NotebookEditorView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let courseCode: String
    let notebookTitle: String
    let pageId: String

    @State private var notebook: CourseNotebook = .empty()
    @State private var blocks: [NotebookEditBlock] = []
    @State private var pageTitle = ""
    @State private var dueTask: ParsedNotebookTask?
    @State private var confirmingDelete = false
    @State private var loaded = false
    @State private var pushTask: Task<Void, Never>?
    @State private var editingDrawing: EditingDrawing?
    @FocusState private var focusedBlock: UUID?
    @FocusState private var titleFocused: Bool

    private struct EditingDrawing: Identifiable {
        let id = UUID()
        let elementsJson: String
        let blockId: UUID
    }

    private var store: NotebookStore {
        NotebookStore(accessToken: session.accessToken)
    }

    private var page: NotebookPage? {
        notebook.pages.first { $0.id == pageId }
    }

    /// `/query` typed at the start of the focused block opens the command menu (web parity).
    private var slashQuery: String? {
        guard let focused = focusedBlock,
              let block = blocks.first(where: { $0.id == focused }), block.isTextual,
              block.text.hasPrefix("/")
        else { return nil }
        let query = String(block.text.dropFirst())
        guard !query.contains(" "), !query.contains("\n"), query.count <= 24 else { return nil }
        return query
    }

    private var slashCommands: [NotebookSlashCommand] {
        guard let slashQuery else { return [] }
        return NotebookSlashCommands.filter(query: slashQuery)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            editorScroll
        }
        .navigationTitle(notebookTitle)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar { toolbarContent }
        .sheet(item: $editingDrawing) { drawing in
            NotebookDrawingEditorView(initialElementsJson: drawing.elementsJson) { json in
                saveDrawing(blockId: drawing.blockId, elementsJson: json)
            }
        }
        .sheet(item: $dueTask) { task in
            NotebookDueDateSheet(
                task: task,
                onSave: { date in setDueDate(taskId: task.id, date: date) }
            )
            .presentationDetents([.medium])
        }
        .alert("Delete this page?", isPresented: $confirmingDelete) {
            Button("Delete", role: .destructive) { deletePage() }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("The page and its notes will be deleted.")
        }
        .onAppear(perform: loadOnce)
        .onChange(of: blocks) {
            guard loaded else { return }
            saveDraft()
        }
        .onChange(of: pageTitle) {
            guard loaded else { return }
            commitTitle()
        }
        .onDisappear {
            saveDraft()
            syncAllTasks()
            // Leave the screen with the server current — skip the debounce.
            pushTask?.cancel()
            NotebookSync.push(store: store, courseCode: courseCode, accessToken: session.accessToken)
        }
    }

    // MARK: - Editor body

    private var editorScroll: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 10) {
                titleField

                ForEach($blocks) { $block in
                    NotebookEditBlockRow(
                        block: $block,
                        orderedNumber: orderedNumber(for: block.id),
                        focus: $focusedBlock,
                        onTextChange: { handleTextChange(block.id) },
                        onToggleTask: { toggleTaskBlock(block.id) },
                        onEditTaskDue: { editTaskDue(block.id) },
                        onEditDrawing: {
                            if case .drawing(let json) = block.kind {
                                editingDrawing = EditingDrawing(elementsJson: json, blockId: block.id)
                            }
                        },
                        onDelete: { deleteBlock(block.id) }
                    )
                }

                // Tap the empty space below the page to keep writing.
                Color.clear
                    .frame(height: 220)
                    .frame(maxWidth: .infinity)
                    .contentShape(Rectangle())
                    .onTapGesture { focusTail() }
            }
            .padding(.horizontal, 20)
            .padding(.top, 8)
        }
        .scrollDismissesKeyboard(.interactively)
        .safeAreaInset(edge: .bottom, spacing: 0) {
            if focusedBlock != nil {
                VStack(spacing: 0) {
                    if !slashCommands.isEmpty {
                        slashMenu
                    }
                    insertToolbar
                }
            }
        }
    }

    private var titleField: some View {
        TextField("Untitled", text: $pageTitle)
            .font(LexturesTheme.displayFont(28, weight: .bold))
            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            .focused($titleFocused)
            .submitLabel(.next)
            .onSubmit { focusHead() }
            .padding(.bottom, 2)
    }

    /// Focus the first textual block (creating one if needed) — after finishing the title.
    private func focusHead() {
        if let first = blocks.first(where: { $0.isTextual }) {
            focusedBlock = first.id
        } else {
            let block = NotebookEditBlock(kind: .paragraph)
            blocks.append(block)
            focusedBlock = block.id
        }
    }

    /// Focus the trailing textual block (appending a paragraph if the page ends in a
    /// drawing/divider) — the tap-below-the-page affordance.
    private func focusTail() {
        if let last = blocks.last, last.isTextual {
            focusedBlock = last.id
        } else {
            let block = NotebookEditBlock(kind: .paragraph)
            blocks.append(block)
            focusedBlock = block.id
        }
    }

    /// Display number for an ordered block: its position within the current run.
    private func orderedNumber(for blockId: UUID) -> Int {
        guard let idx = blocks.firstIndex(where: { $0.id == blockId }) else { return 1 }
        var number = 1
        var walkIndex = idx - 1
        while walkIndex >= 0, blocks[walkIndex].kind.isOrdered {
            number += 1
            walkIndex -= 1
        }
        return number
    }

    // MARK: - Slash menu + insert toolbar

    private var slashMenu: some View {
        ScrollView {
            VStack(spacing: 0) {
                ForEach(slashCommands) { command in
                    Button {
                        applyCommand(command, clearSlash: true)
                    } label: {
                        HStack(spacing: 12) {
                            Image(systemName: command.icon)
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                                .frame(width: 26)
                            VStack(alignment: .leading, spacing: 1) {
                                Text(command.label)
                                    .font(.subheadline.weight(.medium))
                                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                Text(command.detail)
                                    .font(.caption2)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            Spacer()
                        }
                        .padding(.horizontal, 14)
                        .padding(.vertical, 8)
                        .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .frame(maxHeight: 220)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
        )
        .shadow(color: LexturesTheme.cardShadow(for: colorScheme), radius: 14, y: 6)
        .padding(.horizontal, 16)
        .padding(.bottom, 8)
    }

    private var insertToolbar: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 4) {
                toolbarButton("task")
                toolbarButton("drawing")
                toolbarDividerLine
                toolbarButton("heading1", label: "H1")
                toolbarButton("heading2", label: "H2")
                toolbarButton("heading3", label: "H3")
                toolbarDividerLine
                toolbarButton("bulletList")
                toolbarButton("orderedList")
                toolbarButton("blockquote")
                toolbarButton("codeBlock")
                toolbarButton("horizontalRule")
                toolbarDividerLine
                Button {
                    if let focused = focusedBlock { deleteBlock(focused) }
                } label: {
                    Image(systemName: "trash")
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.coral)
                        .frame(width: 38, height: 34)
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Delete block")
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 6)
        }
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .overlay(alignment: .top) {
            Rectangle()
                .fill(LexturesTheme.fieldBorder(for: colorScheme))
                .frame(height: 1)
        }
    }

    private var toolbarDividerLine: some View {
        Rectangle()
            .fill(LexturesTheme.fieldBorder(for: colorScheme))
            .frame(width: 1, height: 22)
            .padding(.horizontal, 4)
    }

    @ViewBuilder
    private func toolbarButton(_ commandId: String, label: String? = nil) -> some View {
        if let command = NotebookSlashCommands.all.first(where: { $0.id == commandId }) {
            Button {
                applyCommand(command, clearSlash: false)
            } label: {
                Group {
                    if let label {
                        Text(label)
                            .font(.footnote.weight(.bold))
                    } else {
                        Image(systemName: command.icon)
                            .font(.subheadline)
                    }
                }
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                .frame(width: 38, height: 34)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .accessibilityLabel(command.label)
        }
    }

    // MARK: - Navigation bar

    @ToolbarContentBuilder
    private var toolbarContent: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            if focusedBlock != nil || titleFocused {
                Button("Done") { dismissKeyboard() }
                    .fontWeight(.semibold)
                    .tint(LexturesTheme.accent(for: colorScheme))
            } else {
                Menu {
                    Button(role: .destructive) {
                        confirmingDelete = true
                    } label: {
                        Label("Delete page", systemImage: "trash")
                    }
                } label: {
                    Image(systemName: "ellipsis.circle")
                }
            }
        }
    }

    private func dismissKeyboard() {
        saveDraft()
        focusedBlock = nil
        titleFocused = false
        UIApplication.shared.sendAction(#selector(UIResponder.resignFirstResponder), to: nil, from: nil, for: nil)
    }

    // MARK: - Loading

    private func loadOnce() {
        guard !loaded else { return }
        var data = store.load(courseCode: courseCode)
        if courseCode != NotebookStore.globalKey {
            data.courseTitle = notebookTitle
        }
        // Opening a page makes it the notebook's active page (web sidebar parity).
        data.activePageId = pageId
        notebook = data
        let current = page
        pageTitle = (current?.title == "Untitled") ? "" : (current?.title ?? "")
        blocks = NotebookMarkdown.editBlocks(from: current?.contentMd ?? "")
        loaded = true
        persist()

        // Fresh page: drop straight into the title so writing is one tap away.
        if (current?.contentMd ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.45) {
                if pageTitle.isEmpty {
                    titleFocused = true
                } else {
                    focusHead()
                }
            }
        }
    }

    // MARK: - Block operations

    /// Newlines never live inside a text block (except code): a return splits the block,
    /// continuing the list/task kind like the web editor does.
    private func handleTextChange(_ blockId: UUID) {
        guard let idx = blocks.firstIndex(where: { $0.id == blockId }) else { return }
        let block = blocks[idx]
        if case .code = block.kind { return }
        guard block.text.contains("\n") else { return }

        let parts = block.text.components(separatedBy: "\n")
        let first = parts[0]
        let rest = parts.dropFirst().joined(separator: " ")

        // Return on an empty list/quote/task line exits the list back to a paragraph.
        if first.isEmpty, rest.isEmpty, !isParagraphKind(block.kind) {
            blocks[idx].kind = .paragraph
            blocks[idx].text = ""
            return
        }

        blocks[idx].text = first
        let continuation: NotebookEditBlock.Kind
        switch block.kind {
        case .bullet: continuation = .bullet
        case .ordered: continuation = .ordered
        case .quote: continuation = .quote
        case .task: continuation = .task(taskId: NotebookMarkdown.newTaskId(), checked: false, dueAt: nil)
        default: continuation = .paragraph
        }
        let newBlock = NotebookEditBlock(kind: continuation, text: rest)
        blocks.insert(newBlock, at: idx + 1)
        focusedBlock = newBlock.id
    }

    private func isParagraphKind(_ kind: NotebookEditBlock.Kind) -> Bool {
        if case .paragraph = kind { return true }
        return false
    }

    /// Convert the focused block (text kinds) or insert after it (divider / drawing).
    private func applyCommand(_ command: NotebookSlashCommand, clearSlash: Bool) {
        var idx = blocks.firstIndex { $0.id == focusedBlock }
        if idx == nil {
            let block = NotebookEditBlock(kind: .paragraph)
            blocks.append(block)
            idx = blocks.count - 1
        }
        guard let idx else { return }
        if clearSlash, blocks[idx].isTextual, blocks[idx].text.hasPrefix("/") {
            blocks[idx].text = ""
        }

        switch command.id {
        case "heading1": blocks[idx].kind = .heading(1)
        case "heading2": blocks[idx].kind = .heading(2)
        case "heading3": blocks[idx].kind = .heading(3)
        case "bulletList": blocks[idx].kind = .bullet
        case "orderedList": blocks[idx].kind = .ordered
        case "blockquote": blocks[idx].kind = .quote
        case "codeBlock": blocks[idx].kind = .code
        case "task":
            if case .task = blocks[idx].kind { break }
            blocks[idx].kind = .task(taskId: NotebookMarkdown.newTaskId(), checked: false, dueAt: nil)
        case "horizontalRule":
            insertBlock(NotebookEditBlock(kind: .divider), after: idx)
        case "drawing":
            insertBlock(NotebookEditBlock(kind: .drawing(elementsJson: "[]")), after: idx)
        default:
            break
        }
        focusedBlock = blocks[min(idx, blocks.count - 1)].id
    }

    /// Insert a non-text block, followed by a paragraph to keep typing into.
    private func insertBlock(_ block: NotebookEditBlock, after idx: Int) {
        let paragraph = NotebookEditBlock(kind: .paragraph)
        blocks.insert(block, at: idx + 1)
        blocks.insert(paragraph, at: idx + 2)
        focusedBlock = paragraph.id
    }

    private func deleteBlock(_ blockId: UUID) {
        guard let idx = blocks.firstIndex(where: { $0.id == blockId }) else { return }
        blocks.remove(at: idx)
        if blocks.isEmpty {
            blocks.append(NotebookEditBlock(kind: .paragraph))
        }
        focusedBlock = blocks[max(0, min(idx - 1, blocks.count - 1))].id
    }

    private func toggleTaskBlock(_ blockId: UUID) {
        guard let idx = blocks.firstIndex(where: { $0.id == blockId }),
              case .task(let taskId, let checked, let dueAt) = blocks[idx].kind
        else { return }
        blocks[idx].kind = .task(taskId: taskId, checked: !checked, dueAt: dueAt)
        syncTask(ParsedNotebookTask(id: taskId, text: blocks[idx].text, checked: !checked, dueAt: dueAt))
    }

    private func editTaskDue(_ blockId: UUID) {
        guard let block = blocks.first(where: { $0.id == blockId }),
              case .task(let taskId, let checked, let dueAt) = block.kind
        else { return }
        dueTask = ParsedNotebookTask(id: taskId, text: block.text, checked: checked, dueAt: dueAt)
    }

    private func setDueDate(taskId: String, date: Date?) {
        guard let idx = blocks.firstIndex(where: {
            if case .task(let id, _, _) = $0.kind { return id == taskId }
            return false
        }), case .task(_, let checked, _) = blocks[idx].kind else { return }
        let dueAt = date.map { endOfDayISO($0) }
        blocks[idx].kind = .task(taskId: taskId, checked: checked, dueAt: dueAt)
        syncTask(ParsedNotebookTask(id: taskId, text: blocks[idx].text, checked: checked, dueAt: dueAt))
    }

    private func endOfDayISO(_ date: Date) -> String {
        let start = Calendar.current.startOfDay(for: date)
        let end = Calendar.current.date(byAdding: DateComponents(day: 1, second: -1), to: start) ?? date
        return ISO8601DateFormatter().string(from: end)
    }

    // MARK: - Drawings

    private func saveDrawing(blockId: UUID, elementsJson: String) {
        guard let idx = blocks.firstIndex(where: { $0.id == blockId }) else { return }
        blocks[idx].kind = .drawing(elementsJson: elementsJson)
    }

    // MARK: - Page operations

    private func commitTitle() {
        let name = pageTitle.trimmingCharacters(in: .whitespacesAndNewlines)
        notebook.pages = NotebookTree.rename(notebook.pages, pageId: pageId, title: name.isEmpty ? "Untitled" : name)
        persist()
    }

    private func deletePage() {
        var pages = NotebookTree.delete(notebook.pages, pageId: pageId)
        // Keep at least one real page in the notebook.
        if !pages.contains(where: { !NotebookTree.isGroup($0) }) {
            let (withPage, newId) = NotebookTree.addPage(pages, parentId: nil)
            pages = withPage
            notebook.activePageId = newId
        } else if !pages.contains(where: { $0.id == notebook.activePageId }) {
            notebook.activePageId = pages.first { !NotebookTree.isGroup($0) }?.id
        }
        notebook.pages = pages
        loaded = false
        store.save(courseCode: courseCode, notebook: notebook)
        NotebookSync.push(store: store, courseCode: courseCode, accessToken: session.accessToken)
        dismiss()
    }

    // MARK: - Task sync (dashboard parity: tasks also live server-side)

    private func syncTask(_ task: ParsedNotebookTask) {
        guard let token = session.accessToken else { return }
        let body = LMSAPI.NotebookTaskUpsert(
            id: task.id,
            courseCode: courseCode,
            notebookPageId: pageId,
            taskText: task.text,
            completed: task.checked,
            dueAt: task.dueAt
        )
        Task { try? await LMSAPI.upsertNotebookTask(body, accessToken: token) }
    }

    private func syncAllTasks() {
        guard loaded else { return }
        for task in NotebookMarkdown.parseTasks(in: NotebookMarkdown.markdown(from: blocks)) {
            syncTask(task)
        }
    }

    // MARK: - Persistence

    private func saveDraft() {
        guard loaded else { return }
        notebook.pages = NotebookTree.updateContent(
            notebook.pages,
            pageId: pageId,
            contentMd: NotebookMarkdown.markdown(from: blocks)
        )
        persist()
    }

    private func persist() {
        store.save(courseCode: courseCode, notebook: notebook)
        schedulePush()
    }

    /// Debounced server push so typing doesn't fire a request per keystroke.
    private func schedulePush() {
        pushTask?.cancel()
        pushTask = Task {
            try? await Task.sleep(nanoseconds: 1_500_000_000)
            guard !Task.isCancelled else { return }
            NotebookSync.push(store: store, courseCode: courseCode, accessToken: session.accessToken)
        }
    }
}
