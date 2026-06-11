import SwiftUI

/// Notebook page editor with a page tree, a rendered reading view (interactive tasks),
/// and a WYSIWYG block edit mode with `/` commands + insert toolbar (parity with the web
/// block editor — blocks stay rendered while editing, markdown is only the storage format).
struct NotebookEditorView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let courseCode: String
    let title: String

    @State private var notebook: CourseNotebook = .empty()
    @State private var blocks: [NotebookEditBlock] = []
    @State private var editing = false
    @State private var showPages = false
    @State private var renamingPage = false
    @State private var renameText = ""
    @State private var dueTask: ParsedNotebookTask?
    @State private var loaded = false
    @State private var pushTask: Task<Void, Never>?
    @State private var editingDrawing: EditingDrawing?
    @FocusState private var focusedBlock: UUID?

    /// Drawing being edited — from the reading view (by document index) or the
    /// block editor (by block id).
    private struct EditingDrawing: Identifiable {
        let id = UUID()
        let elementsJson: String
        var readIndex: Int?
        var blockId: UUID?
    }

    private var store: NotebookStore {
        NotebookStore(accessToken: session.accessToken)
    }

    private var activePage: NotebookPage? {
        notebook.pages.first { $0.id == notebook.activePageId } ?? notebook.pages.first { $0.kind != "group" }
    }

    private var activeIsGroup: Bool {
        activePage.map { NotebookTree.isGroup($0) } == true
    }

    /// `/query` typed at the start of the focused block opens the command menu (web parity).
    private var slashQuery: String? {
        guard editing, let focused = focusedBlock,
              let block = blocks.first(where: { $0.id == focused }), block.isTextual,
              block.text.hasPrefix("/")
        else { return nil }
        let query = String(block.text.dropFirst())
        guard !query.contains(" "), !query.contains("\n"), query.count <= 24 else { return nil }
        return query
    }

    private var slashCommands: [NotebookSlashCommand] {
        guard let slashQuery else { return [] }
        return NotebookMarkdown.filterCommands(query: slashQuery)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            VStack(spacing: 0) {
                pageHeader

                if activeIsGroup {
                    groupPanel
                } else if editing {
                    blockEditor
                } else {
                    readingView
                }
            }
        }
        .navigationTitle(title)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar { toolbarContent }
        .sheet(isPresented: $showPages) { pagesSheet }
        .sheet(item: $editingDrawing) { drawing in
            NotebookDrawingEditorView(initialElementsJson: drawing.elementsJson) { json in
                saveDrawing(drawing, elementsJson: json)
            }
        }
        .sheet(item: $dueTask) { task in
            NotebookDueDateSheet(
                task: task,
                onSave: { date in setDueDate(taskId: task.id, date: date) }
            )
            .presentationDetents([.medium])
        }
        .alert("Rename page", isPresented: $renamingPage) {
            TextField("Page title", text: $renameText)
            Button("Save") { renameActivePage() }
            Button("Cancel", role: .cancel) {}
        }
        .onAppear(perform: loadOnce)
        .onChange(of: blocks) {
            if editing { saveDraft() }
        }
        .onDisappear {
            saveDraft()
            if editing { syncAllTasks() }
            // Leave the screen with the server current — skip the debounce.
            pushTask?.cancel()
            NotebookSync.push(store: store, courseCode: courseCode, accessToken: session.accessToken)
        }
    }

    // MARK: - Header (current page → pages sheet)

    private var pageHeader: some View {
        HStack(spacing: 10) {
            Button {
                saveDraft()
                showPages = true
            } label: {
                HStack(spacing: 6) {
                    Image(systemName: activeIsGroup ? "folder.fill" : "doc.text")
                        .font(.caption)
                        .foregroundStyle(activeIsGroup ? LexturesTheme.brandAmber : LexturesTheme.accent(for: colorScheme))
                    Text(headerTitle)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .lineLimit(1)
                    Image(systemName: "chevron.up.chevron.down")
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                .padding(.horizontal, 14)
                .padding(.vertical, 9)
                .background(LexturesTheme.cardBackground(for: colorScheme))
                .clipShape(Capsule())
                .overlay(Capsule().stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1))
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Show pages")

            Spacer()

            if !editing, !activeIsGroup {
                Button {
                    startEditing()
                } label: {
                    HStack(spacing: 5) {
                        Image(systemName: "pencil")
                        Text("Edit")
                    }
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 16)
                    .padding(.vertical, 9)
                    .background(
                        LinearGradient(
                            colors: [LexturesTheme.primary, Color(hex: 0x17897B)],
                            startPoint: .leading,
                            endPoint: .trailing
                        )
                    )
                    .clipShape(Capsule())
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }

    private var headerTitle: String {
        guard let page = activePage else { return "Untitled" }
        let path = NotebookTree.pathLabel(notebook.pages, pageId: page.id)
        return path.isEmpty ? "Untitled" : path
    }

    // MARK: - Reading view

    private var readingView: some View {
        ScrollView {
            if (activePage?.contentMd ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                emptyPageState
            } else {
                NotebookContentView(
                    markdown: activePage?.contentMd ?? "",
                    onToggleTask: { task in toggleTask(task) },
                    onEditTaskDue: { task in dueTask = task },
                    onEditDrawing: { index, elementsJson in
                        editingDrawing = EditingDrawing(elementsJson: elementsJson, readIndex: index, blockId: nil)
                    }
                )
                .padding(16)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(LexturesTheme.cardBackground(for: colorScheme))
                .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 16, style: .continuous)
                        .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.9), lineWidth: 1)
                )
                .padding(.horizontal, 16)
                .padding(.bottom, 24)
            }
        }
    }

    private var emptyPageState: some View {
        VStack(spacing: 14) {
            Image(systemName: "square.and.pencil")
                .font(.system(size: 34))
                .foregroundStyle(LexturesTheme.accent(for: colorScheme).opacity(0.7))
            Text("This page is empty")
                .font(LexturesTheme.displayFont(19))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text("Tap Edit to start writing. Type / on a new line for headings, tasks, drawings, and more.")
                .font(.footnote)
                .multilineTextAlignment(.center)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .padding(.horizontal, 40)
        .padding(.top, 80)
        .frame(maxWidth: .infinity)
    }

    // MARK: - Group panel (active "page" is a group)

    private var groupPanel: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 10) {
                let children = NotebookTree.sortedChildren(notebook.pages, parentId: activePage?.id)
                if children.isEmpty {
                    Text("No pages in this group yet.")
                        .font(.footnote)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .padding(.top, 20)
                        .frame(maxWidth: .infinity)
                }
                ForEach(children) { child in
                    Button {
                        selectPage(child.id)
                    } label: {
                        HStack(spacing: 10) {
                            Image(systemName: NotebookTree.isGroup(child) ? "folder.fill" : "doc.text")
                                .foregroundStyle(NotebookTree.isGroup(child) ? LexturesTheme.brandAmber : LexturesTheme.accent(for: colorScheme))
                            Text(child.title.isEmpty ? "Untitled" : child.title)
                                .font(.subheadline.weight(.medium))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Spacer()
                            Image(systemName: "chevron.right")
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        .padding(14)
                        .background(LexturesTheme.cardBackground(for: colorScheme))
                        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                        .overlay(
                            RoundedRectangle(cornerRadius: 12, style: .continuous)
                                .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
                        )
                    }
                    .buttonStyle(.plain)
                }

                Button {
                    createPage(parentId: activePage?.id)
                } label: {
                    Label("New page in group", systemImage: "plus")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                }
                .padding(.top, 4)
            }
            .padding(16)
        }
    }

    // MARK: - Block editor

    private var blockEditor: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 10) {
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
                                editingDrawing = EditingDrawing(elementsJson: json, readIndex: nil, blockId: block.id)
                            }
                        },
                        onDelete: { deleteBlock(block.id) }
                    )
                }
            }
            .padding(16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(LexturesTheme.cardBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 16, style: .continuous)
                    .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.9), lineWidth: 1)
            )
            .padding(.horizontal, 16)
            .padding(.bottom, 24)
            .onTapGesture {
                // Tap below the last block: focus it (or add a trailing paragraph).
                if let last = blocks.last, last.isTextual {
                    focusedBlock = last.id
                } else {
                    let block = NotebookEditBlock(kind: .paragraph)
                    blocks.append(block)
                    focusedBlock = block.id
                }
            }
        }
        .safeAreaInset(edge: .bottom, spacing: 0) {
            VStack(spacing: 0) {
                if !slashCommands.isEmpty {
                    slashMenu
                }
                insertToolbar
            }
        }
    }

    /// Display number for an ordered block: its position within the current run.
    private func orderedNumber(for blockId: UUID) -> Int {
        guard let idx = blocks.firstIndex(where: { $0.id == blockId }) else { return 1 }
        var number = 1
        var i = idx - 1
        while i >= 0, blocks[i].kind.isOrdered {
            number += 1
            i -= 1
        }
        return number
    }

    private var slashMenu: some View {
        ScrollView {
            VStack(spacing: 0) {
                ForEach(slashCommands) { command in
                    Button {
                        applySlashCommand(command)
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
                toolbarButton("heading1", label: "H1")
                toolbarButton("heading2", label: "H2")
                toolbarButton("heading3", label: "H3")
                toolbarDividerLine
                toolbarButton("task")
                toolbarButton("drawing")
                toolbarButton("bulletList")
                toolbarButton("orderedList")
                toolbarDividerLine
                toolbarButton("blockquote")
                toolbarButton("codeBlock")
                toolbarButton("horizontalRule")
                toolbarDividerLine
                Button {
                    if let focused = focusedBlock { deleteBlock(focused) }
                } label: {
                    Image(systemName: "trash")
                        .font(.subheadline)
                        .foregroundStyle(
                            focusedBlock == nil
                                ? LexturesTheme.textSecondary(for: colorScheme).opacity(0.5)
                                : LexturesTheme.coral
                        )
                        .frame(width: 38, height: 34)
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .disabled(focusedBlock == nil)
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
        if let command = NotebookMarkdown.slashCommands.first(where: { $0.id == commandId }) {
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

    // MARK: - Toolbar (navigation bar)

    @ToolbarContentBuilder
    private var toolbarContent: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            if editing {
                Button("Done") { finishEditing() }
                    .fontWeight(.semibold)
                    .tint(LexturesTheme.accent(for: colorScheme))
            } else {
                Menu {
                    Button {
                        renameText = activePage?.title ?? ""
                        renamingPage = true
                    } label: {
                        Label("Rename page", systemImage: "pencil")
                    }
                    Button {
                        createPage(parentId: nil)
                    } label: {
                        Label("New page", systemImage: "plus")
                    }
                    Button {
                        createGroup()
                    } label: {
                        Label("New group", systemImage: "folder.badge.plus")
                    }
                } label: {
                    Image(systemName: "ellipsis.circle")
                }
            }
        }
    }

    // MARK: - Pages sheet wiring

    private var pagesSheet: some View {
        NotebookPagesSheet(
            pages: notebook.pages,
            activePageId: activePage?.id,
            onSelect: { selectPage($0) },
            onCreatePage: { parentId in createPage(parentId: parentId) },
            onCreateGroup: { createGroup() },
            onRename: { pageId, title in
                notebook.pages = NotebookTree.rename(notebook.pages, pageId: pageId, title: title)
                persist()
            },
            onMove: { pageId, newParentId in
                if let moved = NotebookTree.moveToParent(notebook.pages, pageId: pageId, newParentId: newParentId) {
                    notebook.pages = moved
                    persist()
                }
            },
            onDelete: { pageId in deletePage(pageId) }
        )
        .presentationDetents([.medium, .large])
    }

    // MARK: - Page operations

    private func loadOnce() {
        guard !loaded else { return }
        loaded = true
        loadFromStore()
        // Merge any newer server copy (e.g. written on web), then refresh once.
        Task {
            if await NotebookSync.pull(store: store, accessToken: session.accessToken) {
                loadFromStore()
            }
        }
    }

    private func loadFromStore() {
        var data = store.load(courseCode: courseCode)
        if courseCode != NotebookStore.globalKey {
            data.courseTitle = title
        }
        notebook = data
        loadBlocks()
        // Empty page → straight into edit mode so writing is one tap away.
        if !activeIsGroup, pageIsEmpty {
            editing = true
        }
    }

    private var pageIsEmpty: Bool {
        (activePage?.contentMd ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    private func loadBlocks() {
        blocks = NotebookMarkdown.editBlocks(from: activePage?.contentMd ?? "")
    }

    private func selectPage(_ pageId: String) {
        saveDraft()
        editing = false
        notebook.activePageId = pageId
        loadBlocks()
        if !activeIsGroup, pageIsEmpty {
            editing = true
        }
        persist()
    }

    private func createPage(parentId: String?) {
        saveDraft()
        let (pages, newId) = NotebookTree.addPage(notebook.pages, parentId: parentId)
        notebook.pages = pages
        notebook.activePageId = newId
        loadBlocks()
        editing = true
        focusedBlock = blocks.first?.id
        persist()
    }

    private func createGroup() {
        saveDraft()
        let (pages, _) = NotebookTree.addGroup(notebook.pages, parentId: nil, title: "New group")
        notebook.pages = pages
        persist()
        showPages = true
    }

    private func deletePage(_ pageId: String) {
        var pages = NotebookTree.delete(notebook.pages, pageId: pageId)
        if !pages.contains(where: { !NotebookTree.isGroup($0) }) {
            let (withPage, newId) = NotebookTree.addPage(pages, parentId: nil)
            pages = withPage
            notebook.activePageId = newId
        }
        notebook.pages = pages
        if !notebook.pages.contains(where: { $0.id == notebook.activePageId }) {
            notebook.activePageId = pages.first { !NotebookTree.isGroup($0) }?.id ?? pages.first?.id
        }
        loadBlocks()
        editing = false
        persist()
    }

    private func renameActivePage() {
        guard let active = activePage else { return }
        let name = renameText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !name.isEmpty else { return }
        notebook.pages = NotebookTree.rename(notebook.pages, pageId: active.id, title: name)
        persist()
    }

    // MARK: - Editing & block operations

    private func startEditing() {
        loadBlocks()
        editing = true
        focusedBlock = blocks.last(where: { $0.isTextual })?.id
    }

    private func finishEditing() {
        saveDraft()
        editing = false
        syncAllTasks()
        focusedBlock = nil
        UIApplication.shared.sendAction(#selector(UIResponder.resignFirstResponder), to: nil, from: nil, for: nil)
    }

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

    private func applySlashCommand(_ command: NotebookSlashCommand) {
        applyCommand(command, clearSlash: true)
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

    // MARK: - Tasks (reading view + due sheet)

    private func toggleTask(_ task: ParsedNotebookTask) {
        guard let active = activePage else { return }
        let next = NotebookMarkdown.setTaskChecked(in: active.contentMd, taskId: task.id, checked: !task.checked)
        updateActiveContent(next)
        syncTask(ParsedNotebookTask(id: task.id, text: task.text, checked: !task.checked, dueAt: task.dueAt))
    }

    private func setDueDate(taskId: String, date: Date?) {
        saveDraft()
        guard let active = activePage else { return }
        let dueAt = date.map { endOfDayISO($0) }
        let next = NotebookMarkdown.setTaskDueAt(in: active.contentMd, taskId: taskId, dueAt: dueAt)
        updateActiveContent(next)
        if let task = NotebookMarkdown.parseTasks(in: next).first(where: { $0.id == taskId }) {
            syncTask(task)
        }
    }

    private func endOfDayISO(_ date: Date) -> String {
        let start = Calendar.current.startOfDay(for: date)
        let end = Calendar.current.date(byAdding: DateComponents(day: 1, second: -1), to: start) ?? date
        return ISO8601DateFormatter().string(from: end)
    }

    private func updateActiveContent(_ contentMd: String) {
        guard let active = activePage else { return }
        notebook.pages = NotebookTree.updateContent(notebook.pages, pageId: active.id, contentMd: contentMd)
        loadBlocks()
        persist()
    }

    // MARK: - Drawings

    private func saveDrawing(_ drawing: EditingDrawing, elementsJson: String) {
        if let blockId = drawing.blockId,
           let idx = blocks.firstIndex(where: { $0.id == blockId }) {
            blocks[idx].kind = .drawing(elementsJson: elementsJson)
            saveDraft()
            return
        }
        if let index = drawing.readIndex, let active = activePage {
            let next = NotebookMarkdown.replaceDrawing(in: active.contentMd, index: index, elementsJson: elementsJson)
            updateActiveContent(next)
        }
    }

    /// Fire-and-forget dashboard sync (web parity: tasks also live server-side).
    private func syncTask(_ task: ParsedNotebookTask) {
        guard let token = session.accessToken, let pageId = activePage?.id else { return }
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
        guard let active = activePage else { return }
        for task in NotebookMarkdown.parseTasks(in: active.contentMd) {
            syncTask(task)
        }
    }

    // MARK: - Persistence

    private func saveDraft() {
        guard editing, let active = activePage, !NotebookTree.isGroup(active) else {
            persist()
            return
        }
        notebook.pages = NotebookTree.updateContent(
            notebook.pages,
            pageId: active.id,
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

/// One block row in edit mode: styled like the reading view, but editable in place.
private struct NotebookEditBlockRow: View {
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
private struct NotebookDueDateSheet: View {
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
