import SwiftUI

/// Markdown notebook editor with a page picker; saves to device-local storage as you type.
struct NotebookEditorView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let courseCode: String
    let title: String

    @State private var notebook: CourseNotebook = .empty()
    @State private var draft = ""
    @State private var renamingPage = false
    @State private var renameText = ""
    @State private var loaded = false

    private var store: NotebookStore {
        NotebookStore(accessToken: session.accessToken)
    }

    private var activePage: NotebookPage? {
        notebook.pages.first { $0.id == notebook.activePageId } ?? notebook.pages.first
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            VStack(spacing: 0) {
                pageBar

                TextEditor(text: $draft)
                    .font(.body.monospaced())
                    .scrollContentBackground(.hidden)
                    .padding(12)
                    .background(LexturesTheme.cardBackground(for: colorScheme))
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                    .overlay(
                        RoundedRectangle(cornerRadius: 12, style: .continuous)
                            .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.9), lineWidth: 1)
                    )
                    .padding(16)
                    .onChange(of: draft) {
                        saveDraft()
                    }
            }
        }
        .navigationTitle(title)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Menu {
                    Button {
                        renameText = activePage?.title ?? ""
                        renamingPage = true
                    } label: {
                        Label("Rename page", systemImage: "pencil")
                    }
                    Button {
                        addPage()
                    } label: {
                        Label("New page", systemImage: "plus")
                    }
                    if notebook.pages.count > 1 {
                        Button(role: .destructive) {
                            deleteActivePage()
                        } label: {
                            Label("Delete page", systemImage: "trash")
                        }
                    }
                } label: {
                    Image(systemName: "ellipsis.circle")
                }
            }
        }
        .alert("Rename page", isPresented: $renamingPage) {
            TextField("Page title", text: $renameText)
            Button("Save") { renameActivePage() }
            Button("Cancel", role: .cancel) {}
        }
        .onAppear {
            guard !loaded else { return }
            loaded = true
            var data = store.load(courseCode: courseCode)
            if courseCode != NotebookStore.globalKey {
                data.courseTitle = title
            }
            notebook = data
            draft = activePage?.contentMd ?? ""
        }
    }

    private var pageBar: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(notebook.pages.sorted { $0.sortOrder < $1.sortOrder }) { page in
                    Button {
                        selectPage(page)
                    } label: {
                        Text(page.title.isEmpty ? "Untitled" : page.title)
                            .font(.caption.weight(page.id == activePage?.id ? .semibold : .regular))
                            .padding(.horizontal, 12)
                            .padding(.vertical, 7)
                            .background(
                                page.id == activePage?.id
                                    ? LexturesTheme.primary.opacity(0.14)
                                    : LexturesTheme.cardBackground(for: colorScheme)
                            )
                            .foregroundStyle(
                                page.id == activePage?.id
                                    ? LexturesTheme.primary
                                    : LexturesTheme.textSecondary(for: colorScheme)
                            )
                            .clipShape(Capsule())
                            .overlay(
                                Capsule().stroke(
                                    page.id == activePage?.id
                                        ? LexturesTheme.primary.opacity(0.4)
                                        : LexturesTheme.fieldBorder(for: colorScheme),
                                    lineWidth: 1
                                )
                            )
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, 16)
            .padding(.top, 12)
        }
    }

    private func selectPage(_ page: NotebookPage) {
        commitDraftToNotebook()
        notebook.activePageId = page.id
        draft = page.contentMd
        store.save(courseCode: courseCode, notebook: notebook)
    }

    private func addPage() {
        commitDraftToNotebook()
        let maxOrder = notebook.pages.map(\.sortOrder).max() ?? 0
        let page = NotebookPage.new(title: "Untitled", sortOrder: maxOrder + 1)
        notebook.pages.append(page)
        notebook.activePageId = page.id
        draft = ""
        store.save(courseCode: courseCode, notebook: notebook)
    }

    private func deleteActivePage() {
        guard let active = activePage, notebook.pages.count > 1 else { return }
        notebook.pages.removeAll { $0.id == active.id }
        notebook.activePageId = notebook.pages.first?.id
        draft = activePage?.contentMd ?? ""
        store.save(courseCode: courseCode, notebook: notebook)
    }

    private func renameActivePage() {
        guard let active = activePage else { return }
        let name = renameText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !name.isEmpty else { return }
        if let idx = notebook.pages.firstIndex(where: { $0.id == active.id }) {
            notebook.pages[idx].title = name
            store.save(courseCode: courseCode, notebook: notebook)
        }
    }

    private func commitDraftToNotebook() {
        guard let active = activePage,
              let idx = notebook.pages.firstIndex(where: { $0.id == active.id }) else { return }
        notebook.pages[idx].contentMd = draft
    }

    private func saveDraft() {
        commitDraftToNotebook()
        store.save(courseCode: courseCode, notebook: notebook)
    }
}
