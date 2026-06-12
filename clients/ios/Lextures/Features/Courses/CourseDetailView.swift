import SwiftUI

/// Course home: gradient hero + segmented sections
/// (Overview · Modules · Grades · Attendance · Grading by role).
struct CourseDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    enum Section: String, CaseIterable {
        case overview = "Overview"
        case modules = "Modules"
        case grades = "Grades"
        case attendance = "Attendance"
        case grading = "Grading"
    }

    @State private var section: Section = .modules
    @State private var items: [CourseStructureItem] = []
    @State private var hasAttendanceSessions = false
    @State private var errorMessage: String?
    @State private var loading = false

    private var sections: [Section] {
        var out: [Section] = [.overview, .modules]
        if course.viewerIsStudent { out.append(.grades) }
        if course.viewerIsStaff || hasAttendanceSessions { out.append(.attendance) }
        if course.viewerIsStaff { out.append(.grading) }
        return out
    }

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

                    LMSSegmentedChips(
                        options: sections,
                        selection: $section,
                        label: \.rawValue
                    )

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    switch section {
                    case .overview:
                        CourseSyllabusSection(course: course)
                    case .modules:
                        modulesSection
                    case .grades:
                        CourseGradesSection(course: course)
                    case .attendance:
                        CourseAttendanceSection(course: course)
                    case .grading:
                        GradingBacklogSection(course: course)
                    }
                }
                .padding(16)
            }
            .refreshable { await load() }
        }
        .navigationTitle(course.displayTitle)
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(for: CourseStructureItem.self) { item in
            ItemDetailView(course: course, item: item)
        }
        .navigationDestination(for: AttendanceSession.self) { attendanceSession in
            AttendanceSessionDetailView(course: course, attendanceSession: attendanceSession)
        }
        .navigationDestination(for: GradingBacklogItem.self) { backlogItem in
            SubmissionsListView(course: course, backlogItem: backlogItem)
        }
        .task { await load() }
    }

    /// Gradient cover banner — matches the course's tile color across the app.
    private var header: some View {
        ZStack(alignment: .topTrailing) {
            Circle()
                .fill(.white.opacity(0.08))
                .frame(width: 140, height: 140)
                .offset(x: 44, y: -52)

            VStack(alignment: .leading, spacing: 7) {
                Text(course.courseCode.uppercased())
                    .font(.caption2.weight(.semibold))
                    .tracking(1.2)
                    .foregroundStyle(.white.opacity(0.8))
                Text(course.title)
                    .font(LexturesTheme.displayFont(22))
                    .foregroundStyle(.white)
                if !course.description.isEmpty {
                    Text(course.description)
                        .font(.footnote)
                        .foregroundStyle(.white.opacity(0.85))
                        .lineLimit(3)
                }
                HStack(spacing: 6) {
                    if let starts = LMSDates.parse(course.startsAt) {
                        heroChip(starts.formatted(date: .abbreviated, time: .omitted), icon: "calendar")
                    }
                    ForEach(roleBadges, id: \.self) { role in
                        heroChip(role, icon: "person.fill")
                    }
                }
                .padding(.top, 4)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(20)
        }
        .background(LexturesTheme.coverGradient(for: course.courseCode))
        .clipShape(RoundedRectangle(cornerRadius: 24, style: .continuous))
        .shadow(color: LexturesTheme.cardShadow(for: colorScheme), radius: 14, y: 7)
    }

    private var roleBadges: [String] {
        (course.viewerEnrollmentRoles ?? []).map { role in
            role.count <= 2 ? role.uppercased() : role.capitalized
        }
    }

    private func heroChip(_ text: String, icon: String) -> some View {
        Label(text, systemImage: icon)
            .font(.caption.weight(.medium))
            .foregroundStyle(.white)
            .padding(.horizontal, 9)
            .padding(.vertical, 4)
            .background(.white.opacity(0.16))
            .clipShape(Capsule())
    }

    // MARK: Modules

    @ViewBuilder
    private var modulesSection: some View {
        if loading && items.isEmpty {
            LMSSkeletonList(count: 3)
        } else if moduleGroups.isEmpty {
            LMSEmptyState(
                systemImage: "square.stack.3d.up",
                title: "No content yet",
                message: "Modules and assignments will appear here once published."
            )
        } else {
            ForEach(Array(moduleGroups.enumerated()), id: \.element.id) { index, group in
                moduleCard(group, number: index + 1)
            }
        }
    }

    private func moduleCard(_ group: ModuleGroup, number: Int) -> some View {
        LMSCard {
            HStack(spacing: 10) {
                Text("\(number)")
                    .font(LexturesTheme.displayFont(14, weight: .bold))
                    .foregroundStyle(.white)
                    .frame(width: 26, height: 26)
                    .background(LexturesTheme.coverGradient(for: course.courseCode))
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                Text(group.title)
                    .font(LexturesTheme.displayFont(17))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            }

            if group.items.isEmpty {
                Text("Nothing in this module yet")
                    .font(.caption)
                    .italic()
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(Array(group.items.enumerated()), id: \.element.id) { index, item in
                    if index > 0 {
                        Divider()
                    }
                    if ItemKind.isOpenable(item.kind) {
                        NavigationLink(value: item) {
                            itemRow(item, openable: true)
                        }
                        .buttonStyle(.plain)
                    } else {
                        itemRow(item, openable: false)
                    }
                }
            }
        }
    }

    private func itemRow(_ item: CourseStructureItem, openable: Bool) -> some View {
        HStack(spacing: 12) {
            Image(systemName: ItemKind.icon(for: item.kind))
                .font(.footnote.weight(.semibold))
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                .frame(width: 32, height: 32)
                .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.16 : 0.13))
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))

            VStack(alignment: .leading, spacing: 3) {
                Text(item.title)
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                HStack(spacing: 6) {
                    Text(ItemKind.label(for: item.kind))
                        .font(.caption2.weight(.medium))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if let due = LMSDates.parse(item.dueAt) {
                        Text("·")
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        Text("Due \(due.formatted(date: .abbreviated, time: .shortened))")
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(LexturesTheme.coral)
                    }
                }
            }

            Spacer(minLength: 0)

            if let points = item.pointsWorth ?? item.pointsPossible {
                Text("\(points.formatted()) pts")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.amber)
                    .padding(.horizontal, 7)
                    .padding(.vertical, 3)
                    .background(LexturesTheme.amber.opacity(0.13))
                    .clipShape(Capsule())
            }
            if openable {
                Image(systemName: "chevron.right")
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
            }
        }
        .padding(.vertical, 4)
        .contentShape(Rectangle())
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            async let sessionsTask = (try? LMSAPI.fetchAttendanceSessions(
                courseCode: course.courseCode,
                accessToken: token
            )) ?? []
            items = try await LMSAPI.fetchCourseStructure(courseCode: course.courseCode, accessToken: token)
            hasAttendanceSessions = await !sessionsTask.isEmpty
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load course content."
        }
    }
}
