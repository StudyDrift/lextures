import SwiftUI

/// Course structure (modules and items) for one course.
struct CourseDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    @State private var items: [CourseStructureItem] = []
    @State private var errorMessage: String?
    @State private var loading = false

    private struct ModuleGroup: Identifiable {
        let id: String
        let title: String
        let items: [CourseStructureItem]
    }

    private var moduleGroups: [ModuleGroup] {
        let modules = items.filter(\.isModule).sorted { $0.sortOrder < $1.sortOrder }
        let children = Dictionary(grouping: items.filter { !$0.isModule && $0.parentId != nil }) { $0.parentId! }
        var groups: [ModuleGroup] = modules.map { module in
            ModuleGroup(
                id: module.id,
                title: module.title,
                items: (children[module.id] ?? []).sorted { $0.sortOrder < $1.sortOrder }
            )
        }
        let orphans = items
            .filter { !$0.isModule && $0.parentId == nil && $0.kind != "heading" }
            .sorted { $0.sortOrder < $1.sortOrder }
        if !orphans.isEmpty {
            groups.append(ModuleGroup(id: "__orphans__", title: "Other items", items: orphans))
        }
        return groups
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    header

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && items.isEmpty {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 40)
                    } else if moduleGroups.isEmpty {
                        LMSEmptyState(
                            systemImage: "square.stack.3d.up",
                            title: "No content yet",
                            message: "Modules and assignments will appear here once published."
                        )
                    } else {
                        ForEach(moduleGroups) { group in
                            moduleCard(group)
                        }
                    }
                }
                .padding(16)
            }
            .refreshable { await load() }
        }
        .navigationTitle(course.displayTitle)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private var header: some View {
        LMSCard {
            Text(course.title)
                .font(.headline)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(course.courseCode)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            if !course.description.isEmpty {
                Text(course.description)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            if let starts = LMSDates.parse(course.startsAt) {
                Label(starts.formatted(date: .abbreviated, time: .omitted), systemImage: "calendar")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func moduleCard(_ group: ModuleGroup) -> some View {
        LMSCard {
            Text(group.title)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            if group.items.isEmpty {
                Text("Empty module")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(group.items) { item in
                    HStack(spacing: 10) {
                        Image(systemName: icon(for: item.kind))
                            .font(.subheadline)
                            .frame(width: 22)
                            .foregroundStyle(LexturesTheme.primary)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(item.title)
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            if let due = LMSDates.parse(item.dueAt) {
                                Text("Due \(due.formatted(date: .abbreviated, time: .shortened))")
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                        Spacer(minLength: 0)
                        if let points = item.pointsWorth ?? item.pointsPossible {
                            Text("\(points.formatted()) pts")
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }
                    .padding(.vertical, 2)
                }
            }
        }
    }

    private func icon(for kind: String) -> String {
        switch kind {
        case "assignment": return "doc.text"
        case "quiz": return "checkmark.circle"
        case "content_page": return "doc.richtext"
        case "external_link": return "link"
        case "survey": return "list.clipboard"
        case "lti_link": return "puzzlepiece.extension"
        case "h5p": return "play.rectangle"
        case "vibe_activity": return "sparkles"
        case "library_resource", "textbook_resource": return "books.vertical"
        default: return "square.stack.3d.up"
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            items = try await LMSAPI.fetchCourseStructure(courseCode: course.courseCode, accessToken: token)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load course content."
        }
    }
}
