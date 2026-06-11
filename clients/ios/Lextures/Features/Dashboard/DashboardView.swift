import SwiftUI

struct DueItem: Identifiable, Hashable {
    var id: String { "\(courseCode)/\(item.id)" }
    let courseCode: String
    let courseTitle: String
    let item: CourseStructureItem
    let dueDate: Date
}

@MainActor
@Observable
final class DashboardModel {
    var courses: [CourseSummary] = []
    var dueThisWeek: [DueItem] = []
    var errorMessage: String?
    var loading = false
    private var loadedOnce = false

    func load(accessToken: String?, force: Bool = false) async {
        guard let accessToken else { return }
        if loadedOnce && !force { return }
        loading = true
        errorMessage = nil
        defer {
            loading = false
            loadedOnce = true
        }

        do {
            let list = try await LMSAPI.fetchCourses(accessToken: accessToken)
            // The list GET omits viewer roles; enrich from the single-course GET.
            let enriched = await withTaskGroup(of: CourseSummary.self) { group in
                for course in list {
                    group.addTask {
                        (try? await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: accessToken)) ?? course
                    }
                }
                var out: [CourseSummary] = []
                for await course in group { out.append(course) }
                return out
            }
            // Preserve catalog order from the list response.
            let order = Dictionary(uniqueKeysWithValues: list.enumerated().map { ($1.id, $0) })
            courses = enriched.sorted { (order[$0.id] ?? 0) < (order[$1.id] ?? 0) }

            let studentCourses = courses.filter(\.viewerIsStudent)
            dueThisWeek = await loadDueThisWeek(for: studentCourses, accessToken: accessToken)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load your dashboard."
        }
    }

    private func loadDueThisWeek(for studentCourses: [CourseSummary], accessToken: String) async -> [DueItem] {
        let (weekStart, weekEnd) = DashboardModel.currentWeek()
        var out: [DueItem] = []
        await withTaskGroup(of: [DueItem].self) { group in
            for course in studentCourses {
                group.addTask {
                    let items = (try? await LMSAPI.fetchCourseStructure(
                        courseCode: course.courseCode,
                        accessToken: accessToken
                    )) ?? []
                    return items.compactMap { item in
                        guard item.isGradable, let due = LMSDates.parse(item.dueAt) else { return nil }
                        guard due >= weekStart, due <= weekEnd else { return nil }
                        return DueItem(
                            courseCode: course.courseCode,
                            courseTitle: course.displayTitle,
                            item: item,
                            dueDate: due
                        )
                    }
                }
            }
            for await items in group { out.append(contentsOf: items) }
        }
        return out.sorted { $0.dueDate < $1.dueDate }
    }

    /// Monday 00:00 through Sunday 23:59 of the current week (parity with web dashboard).
    static func currentWeek(now: Date = Date()) -> (Date, Date) {
        var calendar = Calendar.current
        calendar.firstWeekday = 2 // Monday
        let start = calendar.dateInterval(of: .weekOfYear, for: now)?.start ?? now
        let end = calendar.date(byAdding: DateComponents(day: 7, second: -1), to: start) ?? now
        return (start, end)
    }
}

struct DashboardView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @State private var model = DashboardModel()
    @Binding var unreadInbox: Int

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        greeting

                        if let error = model.errorMessage {
                            LMSErrorBanner(message: error)
                        }

                        if model.loading && model.courses.isEmpty {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 40)
                        } else {
                            statsRow
                            dueThisWeekSection
                            coursesSection
                        }
                    }
                    .padding(16)
                }
                .refreshable {
                    await model.load(accessToken: session.accessToken, force: true)
                }
            }
            .navigationTitle("Dashboard")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Menu {
                        if let email = session.userEmail {
                            Text(email)
                        }
                        Button("Sign out", role: .destructive) {
                            session.signOut()
                        }
                    } label: {
                        Image(systemName: "person.crop.circle")
                    }
                }
            }
            .navigationDestination(for: CourseSummary.self) { course in
                CourseDetailView(course: course)
            }
            .task {
                await model.load(accessToken: session.accessToken)
            }
        }
    }

    private var greeting: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(greetingText)
                .font(.title2.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            if let email = session.userEmail {
                Text(email)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private var greetingText: String {
        let hour = Calendar.current.component(.hour, from: Date())
        switch hour {
        case ..<12: return "Good morning"
        case ..<17: return "Good afternoon"
        default: return "Good evening"
        }
    }

    private var statsRow: some View {
        HStack(spacing: 12) {
            statCard(value: "\(model.courses.count)", label: "Courses", systemImage: "book")
            statCard(value: "\(model.dueThisWeek.count)", label: "Due this week", systemImage: "clipboard")
            statCard(value: "\(unreadInbox)", label: "Unread", systemImage: "tray")
        }
    }

    private func statCard(value: String, label: String, systemImage: String) -> some View {
        LMSCard {
            Image(systemName: systemImage)
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.primary)
            Text(value)
                .font(.title2.weight(.bold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(label)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    @ViewBuilder
    private var dueThisWeekSection: some View {
        LMSSectionHeader(title: "Due this week", systemImage: "calendar")
        if model.dueThisWeek.isEmpty {
            LMSCard {
                Text("Nothing due this week. Enjoy the breathing room!")
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        } else {
            ForEach(model.dueThisWeek) { due in
                LMSCard {
                    Text(due.item.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    HStack {
                        Text(due.courseTitle)
                        Spacer()
                        Text(due.dueDate.formatted(date: .abbreviated, time: .shortened))
                    }
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    @ViewBuilder
    private var coursesSection: some View {
        LMSSectionHeader(title: "Your courses", systemImage: "book")
        if model.courses.isEmpty {
            LMSEmptyState(
                systemImage: "book",
                title: "No courses yet",
                message: "Courses you enroll in will show up here."
            )
        } else {
            ForEach(model.courses.prefix(5)) { course in
                NavigationLink(value: course) {
                    LMSCard {
                        Text(course.displayTitle)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(course.courseCode)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
                .buttonStyle(.plain)
            }
        }
    }
}

#Preview {
    DashboardView(unreadInbox: .constant(2))
        .environment(AuthSession())
}
