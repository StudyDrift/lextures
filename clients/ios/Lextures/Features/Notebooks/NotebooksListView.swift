import SwiftUI

/// My Notebooks: the global notebook plus one notebook per enrolled course (device-local).
struct NotebooksListView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    @State private var courses: [CourseSummary] = []
    @State private var savedNotebooks: [String: CourseNotebook] = [:]
    @State private var loadedOnce = false

    private var store: NotebookStore {
        NotebookStore(accessToken: session.accessToken)
    }

    /// Course-code → display title for notebook rows (enrolled courses plus any saved strays).
    private var courseRows: [(code: String, title: String, notebook: CourseNotebook?)] {
        var rows: [(String, String, CourseNotebook?)] = []
        var seen = Set<String>()
        for course in courses where course.notebookEnabled != false {
            rows.append((course.courseCode, course.displayTitle, savedNotebooks[course.courseCode]))
            seen.insert(course.courseCode)
        }
        for (code, notebook) in savedNotebooks where !seen.contains(code) {
            rows.append((code, notebook.courseTitle ?? code, notebook))
        }
        return rows
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 12) {
                        LMSSectionHeader(title: "Global notebook", systemImage: "globe")
                        NavigationLink(
                            value: NotebookRoute(courseCode: NotebookStore.globalKey, title: NotebookStore.globalTitle)
                        ) {
                            notebookCard(
                                title: NotebookStore.globalTitle,
                                subtitle: "Notes that follow you across courses",
                                notebook: store.load(courseCode: NotebookStore.globalKey)
                            )
                        }
                        .buttonStyle(.plain)

                        LMSSectionHeader(title: "Course notebooks", systemImage: "note.text")
                        if courseRows.isEmpty {
                            LMSEmptyState(
                                systemImage: "note.text",
                                title: "No course notebooks",
                                message: "Enroll in a course to start a notebook for it."
                            )
                        } else {
                            ForEach(courseRows, id: \.code) { row in
                                NavigationLink(value: NotebookRoute(courseCode: row.code, title: row.title)) {
                                    notebookCard(
                                        title: row.title,
                                        subtitle: row.code,
                                        notebook: row.notebook
                                    )
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                    .padding(16)
                }
                .refreshable { await load(force: true) }
            }
            .navigationTitle("Notebooks")
            .navigationBarTitleDisplayMode(.inline)
            .navigationDestination(for: NotebookRoute.self) { route in
                NotebookEditorView(courseCode: route.courseCode, title: route.title)
            }
            .task { await load() }
            .onAppear { savedNotebooks = store.listCourseNotebooks() }
        }
    }

    private func notebookCard(title: String, subtitle: String, notebook: CourseNotebook?) -> some View {
        LMSCard {
            HStack(alignment: .top, spacing: 14) {
                LMSCoverTile(key: subtitle, systemImage: "square.and.pencil", size: 48)

                VStack(alignment: .leading, spacing: 4) {
                    Text(title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(subtitle)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if let notebook, !notebook.previewText.isEmpty {
                        Text(notebook.previewText)
                            .font(.caption)
                            .lineLimit(2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        if let updated = LMSDates.parse(notebook.updatedAt) {
                            Text("Updated \(updated.formatted(date: .abbreviated, time: .shortened))")
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    } else {
                        Text("No notes yet")
                            .font(.caption)
                            .italic()
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }

                Spacer(minLength: 0)

                Image(systemName: "chevron.right")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .padding(.top, 14)
            }
        }
    }

    private func load(force: Bool = false) async {
        savedNotebooks = store.listCourseNotebooks()
        guard let token = session.accessToken else { return }
        if loadedOnce && !force { return }
        loadedOnce = true
        courses = (try? await LMSAPI.fetchCourses(accessToken: token)) ?? courses
    }
}

struct NotebookRoute: Hashable {
    let courseCode: String
    let title: String
}
