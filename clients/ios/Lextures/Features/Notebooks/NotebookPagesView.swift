import SwiftUI

/// Pages of one notebook: an Evernote-style list with collapsible groups, search, and a
/// floating new-page button. All page management (create / rename / move / delete) lives
/// here via context menus — replaces the old in-editor pages capsule + bottom sheet.
struct NotebookPagesView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let courseCode: String
    let title: String

    @State private var notebook: CourseNotebook = .empty()
    @State private var collapsed: Set<String> = []
    @State private var searchText = ""
    @State private var renameTarget: NotebookPage?
    @State private var renameText = ""
    @State private var deleteTarget: NotebookPage?
    @State private var openedPage: NotebookPageRoute?
    @State private var pulledOnce = false

    private var store: NotebookStore {
        NotebookStore(accessToken: session.accessToken)
    }

    private struct Row: Identifiable {
        let page: NotebookPage
        let depth: Int
        var id: String { page.id }
    }

    /// Depth-first rows, skipping the contents of collapsed groups.
    private var treeRows: [Row] {
        var rows: [Row] = []
        func walk(parentId: String?, depth: Int) {
            for page in NotebookTree.sortedChildren(notebook.pages, parentId: parentId) {
                rows.append(Row(page: page, depth: depth))
                if NotebookTree.isGroup(page), !collapsed.contains(page.id) {
                    walk(parentId: page.id, depth: depth + 1)
                }
            }
        }
        walk(parentId: nil, depth: 0)
        return rows
    }

    /// While searching: flat list of pages whose title or content matches.
    private var searchRows: [Row] {
        let query = searchText.trimmingCharacters(in: .whitespaces).lowercased()
        return notebook.pages
            .filter { !NotebookTree.isGroup($0) }
            .filter {
                $0.title.lowercased().contains(query)
                    || NotebookMarkdown.previewText($0.contentMd).lowercased().contains(query)
            }
            .map { Row(page: $0, depth: 0) }
    }

    private var isSearching: Bool {
        !searchText.trimmingCharacters(in: .whitespaces).isEmpty
    }

    var body: some View {
        ZStack(alignment: .bottomTrailing) {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                LazyVStack(alignment: .leading, spacing: 8) {
                    let rows = isSearching ? searchRows : treeRows
                    if rows.isEmpty {
                        LMSEmptyState(
                            systemImage: isSearching ? "magnifyingglass" : "square.and.pencil",
                            title: isSearching ? "No matches" : "No pages yet",
                            message: isSearching
                                ? "No pages match your search."
                                : "Tap + to create your first page."
                        )
                    }
                    ForEach(rows) { row in
                        if NotebookTree.isGroup(row.page) {
                            groupRow(row)
                        } else {
                            pageRow(row)
                        }
                    }
                }
                .padding(16)
                .padding(.bottom, 96)
            }

            newPageButton
        }
        .navigationTitle(title)
        .navigationBarTitleDisplayMode(.inline)
        .searchable(text: $searchText, placement: .navigationBarDrawer(displayMode: .automatic), prompt: "Search pages")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    createGroup()
                } label: {
                    Image(systemName: "folder.badge.plus")
                }
                .accessibilityLabel("New group")
            }
        }
        .navigationDestination(item: $openedPage) { route in
            NotebookEditorView(courseCode: route.courseCode, notebookTitle: route.notebookTitle, pageId: route.pageId)
        }
        .onAppear(perform: refresh)
        .alert("Rename", isPresented: Binding(
            get: { renameTarget != nil },
            set: { if !$0 { renameTarget = nil } }
        )) {
            TextField("Title", text: $renameText)
            Button("Save") {
                if let target = renameTarget {
                    let name = renameText.trimmingCharacters(in: .whitespacesAndNewlines)
                    if !name.isEmpty {
                        notebook.pages = NotebookTree.rename(notebook.pages, pageId: target.id, title: name)
                        persist()
                    }
                }
                renameTarget = nil
            }
            Button("Cancel", role: .cancel) { renameTarget = nil }
        }
        .alert("Delete \"\(deleteTarget.map { $0.title.isEmpty ? "Untitled" : $0.title } ?? "")\"?", isPresented: Binding(
            get: { deleteTarget != nil },
            set: { if !$0 { deleteTarget = nil } }
        )) {
            Button("Delete", role: .destructive) {
                if let target = deleteTarget { deletePage(target.id) }
                deleteTarget = nil
            }
            Button("Cancel", role: .cancel) { deleteTarget = nil }
        } message: {
            Text(deleteTarget.map { NotebookTree.isGroup($0) } == true
                ? "The group and everything inside it will be deleted."
                : "This page and its notes will be deleted.")
        }
    }

    // MARK: - Rows

    private func pageRow(_ row: Row) -> some View {
        let page = row.page
        let snippet = NotebookMarkdown.previewText(page.contentMd)

        return Button {
            openPage(page)
        } label: {
            HStack(spacing: 12) {
                Image(systemName: "doc.text")
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 24)

                VStack(alignment: .leading, spacing: 3) {
                    Text(page.title.isEmpty ? "Untitled" : page.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .lineLimit(1)
                    if !snippet.isEmpty {
                        Text(snippet)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            .lineLimit(1)
                    }
                }

                Spacer(minLength: 0)

                Image(systemName: "chevron.right")
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 12)
            .background(LexturesTheme.cardBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.9), lineWidth: 1)
            )
            .padding(.leading, CGFloat(row.depth) * 18)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .contextMenu {
            Button {
                renameTarget = page
                renameText = page.title
            } label: {
                Label("Rename", systemImage: "pencil")
            }
            moveMenu(for: page)
            if canDelete(page) {
                Button(role: .destructive) {
                    deleteTarget = page
                } label: {
                    Label("Delete", systemImage: "trash")
                }
            }
        }
    }

    private func groupRow(_ row: Row) -> some View {
        let group = row.page
        let count = NotebookTree.sortedChildren(notebook.pages, parentId: group.id).count
        let isCollapsed = collapsed.contains(group.id)

        return Button {
            withAnimation(.easeOut(duration: 0.18)) {
                if isCollapsed {
                    collapsed.remove(group.id)
                } else {
                    collapsed.insert(group.id)
                }
            }
        } label: {
            HStack(spacing: 12) {
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .rotationEffect(.degrees(isCollapsed ? 0 : 90))
                    .frame(width: 16)

                Image(systemName: "folder.fill")
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.brandAmber)

                Text(group.title.isEmpty ? "Untitled group" : group.title)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .lineLimit(1)

                Spacer(minLength: 0)

                Text(count == 1 ? "1 item" : "\(count) items")
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 10)
            .padding(.leading, CGFloat(row.depth) * 18)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .contextMenu {
            Button {
                createPage(parentId: group.id)
            } label: {
                Label("New page inside", systemImage: "plus")
            }
            Button {
                renameTarget = group
                renameText = group.title
            } label: {
                Label("Rename", systemImage: "pencil")
            }
            moveMenu(for: group)
            if canDelete(group) {
                Button(role: .destructive) {
                    deleteTarget = group
                } label: {
                    Label("Delete", systemImage: "trash")
                }
            }
        }
    }

    @ViewBuilder
    private func moveMenu(for page: NotebookPage) -> some View {
        let targets = NotebookTree.groupMoveTargets(notebook.pages, pageId: page.id).filter { $0.id != page.parentId }
        if !targets.isEmpty || page.parentId != nil {
            Menu {
                if page.parentId != nil {
                    Button {
                        move(page.id, to: nil)
                    } label: {
                        Label("Top level", systemImage: "arrow.up.to.line")
                    }
                }
                ForEach(targets) { group in
                    Button {
                        move(page.id, to: group.id)
                    } label: {
                        Label(NotebookTree.pathLabel(notebook.pages, pageId: group.id), systemImage: "folder")
                    }
                }
            } label: {
                Label("Move to…", systemImage: "arrow.turn.down.right")
            }
        }
    }

    // MARK: - Floating new-page button

    private var newPageButton: some View {
        Button {
            createPage(parentId: nil)
        } label: {
            Image(systemName: "plus")
                .font(.title3.weight(.semibold))
                .foregroundStyle(.white)
                .frame(width: 56, height: 56)
                .background(
                    LinearGradient(
                        colors: [LexturesTheme.primary, Color(hex: 0x17897B)],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                )
                .clipShape(Circle())
                .shadow(color: LexturesTheme.primary.opacity(0.35), radius: 10, y: 5)
        }
        .buttonStyle(.plain)
        .padding(.horizontal, 20)
        .padding(.top, 20)
        .padding(.bottom, 20)
        .accessibilityLabel("New page")
    }

    // MARK: - Operations

    private func refresh() {
        notebook = store.load(courseCode: courseCode)
        guard !pulledOnce else { return }
        pulledOnce = true
        // Merge any newer server copy (e.g. written on web) once, then refresh.
        Task {
            if await NotebookSync.pull(store: store, accessToken: session.accessToken) {
                notebook = store.load(courseCode: courseCode)
            }
        }
    }

    private func openPage(_ page: NotebookPage) {
        openedPage = NotebookPageRoute(courseCode: courseCode, notebookTitle: title, pageId: page.id)
    }

    private func createPage(parentId: String?) {
        let (pages, newId) = NotebookTree.addPage(notebook.pages, parentId: parentId)
        notebook.pages = pages
        notebook.activePageId = newId
        if let parentId { collapsed.remove(parentId) }
        persist()
        openedPage = NotebookPageRoute(courseCode: courseCode, notebookTitle: title, pageId: newId)
    }

    private func createGroup() {
        let (pages, newId) = NotebookTree.addGroup(notebook.pages, parentId: nil, title: "New group")
        notebook.pages = pages
        persist()
        renameTarget = pages.first { $0.id == newId }
        renameText = "New group"
    }

    private func deletePage(_ pageId: String) {
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
        persist()
    }

    private func move(_ pageId: String, to newParentId: String?) {
        if let moved = NotebookTree.moveToParent(notebook.pages, pageId: pageId, newParentId: newParentId) {
            notebook.pages = moved
            persist()
        }
    }

    /// Keep at least one real page in the notebook.
    private func canDelete(_ page: NotebookPage) -> Bool {
        let remaining = NotebookTree.delete(notebook.pages, pageId: page.id)
        return remaining.contains { !NotebookTree.isGroup($0) }
    }

    private func persist() {
        var data = notebook
        if courseCode != NotebookStore.globalKey {
            data.courseTitle = title
        }
        store.save(courseCode: courseCode, notebook: data)
        notebook = data
        NotebookSync.push(store: store, courseCode: courseCode, accessToken: session.accessToken)
    }
}

/// Route to one page's editor.
struct NotebookPageRoute: Hashable, Identifiable {
    let courseCode: String
    let notebookTitle: String
    let pageId: String
    var id: String { pageId }
}
