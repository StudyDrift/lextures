import SwiftUI

/// Page tree for a notebook: groups with nested pages, plus create / rename / move / delete
/// (parity with the web notebook sidebar).
struct NotebookPagesSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme

    let pages: [NotebookPage]
    let activePageId: String?
    var onSelect: (String) -> Void
    var onCreatePage: (_ parentId: String?) -> Void
    var onCreateGroup: () -> Void
    var onRename: (_ pageId: String, _ title: String) -> Void
    var onMove: (_ pageId: String, _ newParentId: String?) -> Void
    var onDelete: (_ pageId: String) -> Void

    @State private var renameTarget: NotebookPage?
    @State private var renameText = ""
    @State private var deleteTarget: NotebookPage?

    private var rows: [NotebookTree.FlatRow] {
        NotebookTree.flatten(pages)
    }

    var body: some View {
        NavigationStack {
            List {
                Section {
                    ForEach(rows) { row in
                        pageRow(row)
                    }
                } footer: {
                    Text("Pages live on this device. Use groups to organize related pages.")
                        .font(.caption2)
                }

                Section {
                    Button {
                        onCreatePage(nil)
                        dismiss()
                    } label: {
                        Label("New page", systemImage: "plus")
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    }
                    Button {
                        onCreateGroup()
                    } label: {
                        Label("New group", systemImage: "folder.badge.plus")
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    }
                }
            }
            .scrollContentBackground(.hidden)
            .background(LexturesTheme.sceneBackground(for: colorScheme))
            .navigationTitle("Pages")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Done") { dismiss() }
                        .fontWeight(.semibold)
                        .tint(LexturesTheme.accent(for: colorScheme))
                }
            }
        }
        .alert("Rename", isPresented: Binding(
            get: { renameTarget != nil },
            set: { if !$0 { renameTarget = nil } }
        )) {
            TextField("Title", text: $renameText)
            Button("Save") {
                if let target = renameTarget {
                    let name = renameText.trimmingCharacters(in: .whitespacesAndNewlines)
                    if !name.isEmpty { onRename(target.id, name) }
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
                if let target = deleteTarget { onDelete(target.id) }
                deleteTarget = nil
            }
            Button("Cancel", role: .cancel) { deleteTarget = nil }
        } message: {
            Text(deleteTarget.map { NotebookTree.isGroup($0) } == true
                ? "The group and everything inside it will be deleted."
                : "This page and its notes will be deleted.")
        }
    }

    private func pageRow(_ row: NotebookTree.FlatRow) -> some View {
        let page = row.page
        let isGroup = NotebookTree.isGroup(page)
        let isActive = page.id == activePageId

        return Button {
            onSelect(page.id)
            dismiss()
        } label: {
            HStack(spacing: 10) {
                Image(systemName: isGroup ? "folder.fill" : "doc.text")
                    .font(.subheadline)
                    .foregroundStyle(isGroup ? LexturesTheme.brandAmber : LexturesTheme.accent(for: colorScheme))
                    .frame(width: 22)

                Text(page.title.isEmpty ? "Untitled" : page.title)
                    .font(.subheadline.weight(isActive ? .semibold : .regular))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .lineLimit(1)

                Spacer(minLength: 0)

                if isActive {
                    Image(systemName: "checkmark")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                }
            }
            .padding(.leading, CGFloat(row.depth) * 20)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .listRowBackground(
            isActive
                ? LexturesTheme.accent(for: colorScheme).opacity(0.10)
                : LexturesTheme.cardBackground(for: colorScheme)
        )
        .contextMenu {
            Button {
                renameTarget = page
                renameText = page.title
            } label: {
                Label("Rename", systemImage: "pencil")
            }

            if isGroup {
                Button {
                    onCreatePage(page.id)
                    dismiss()
                } label: {
                    Label("New page inside", systemImage: "plus")
                }
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

    @ViewBuilder
    private func moveMenu(for page: NotebookPage) -> some View {
        let targets = NotebookTree.groupMoveTargets(pages, pageId: page.id)
        Menu {
            if page.parentId != nil {
                Button {
                    onMove(page.id, nil)
                } label: {
                    Label("Top level", systemImage: "arrow.up.to.line")
                }
            }
            ForEach(targets.filter { $0.id != page.parentId }) { group in
                Button {
                    onMove(page.id, group.id)
                } label: {
                    Label(NotebookTree.pathLabel(pages, pageId: group.id), systemImage: "folder")
                }
            }
        } label: {
            Label("Move to…", systemImage: "arrow.turn.down.right")
        }
        .disabled(targets.filter { $0.id != page.parentId }.isEmpty && page.parentId == nil)
    }

    /// Keep at least one real page in the notebook.
    private func canDelete(_ page: NotebookPage) -> Bool {
        let remaining = NotebookTree.delete(pages, pageId: page.id)
        return remaining.contains { !NotebookTree.isGroup($0) }
    }
}
