import SwiftUI

struct CoursesListView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @State private var courses: [CourseSummary] = []
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = false
    @State private var loadedOnce = false
    @State private var searchText = ""
    @State private var deepLinkedCourse: CourseSummary?
    @State private var deepLinkSection: CourseDeepLinkSection?
    @State private var deepLinkItemId: String?
    @Bindable private var realtime = RealtimeManager.shared

    private var filteredCourses: [CourseSummary] {
        let query = searchText.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !query.isEmpty else { return courses }
        return courses.filter {
            $0.displayTitle.lowercased().contains(query)
                || $0.courseCode.lowercased().contains(query)
                || $0.description.lowercased().contains(query)
        }
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 12) {
                        if !NetworkMonitor.shared.isOnline {
                            OfflineBanner()
                        }
                        if let cacheLabel {
                            StalenessChip(label: cacheLabel)
                        }
                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }

                        if loading && courses.isEmpty {
                            LMSSkeletonList(count: 5)
                        } else if filteredCourses.isEmpty {
                            LMSEmptyState(
                                systemImage: "book",
                                title: searchText.isEmpty ? "No courses yet" : "No matching courses",
                                message: searchText.isEmpty
                                    ? "Courses you enroll in will show up here."
                                    : "Try different keywords, or clear search."
                            )
                        } else {
                            ForEach(filteredCourses) { course in
                                NavigationLink(value: course) {
                                    CourseRowCard(course: course)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                    .padding(16)
                }
                .refreshable { await load(force: true) }
            }
            .navigationTitle(L.text("mobile.courses.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                if shell.iaRedesignEnabled && shell.universalSearchEnabled {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button { shell.showUniversalSearch = true } label: {
                            Image(systemName: "magnifyingglass")
                        }
                        .accessibilityLabel(L.text("mobile.ia.search"))
                    }
                }
            }
            .searchable(text: $searchText, prompt: "Search courses")
            .navigationDestination(for: CourseSummary.self) { course in
                CourseDetailView(
                    course: course,
                    initialSection: mapSection(deepLinkSection),
                    initialItemId: deepLinkItemId
                )
            }
            .navigationDestination(item: $deepLinkedCourse) { course in
                CourseDetailView(
                    course: course,
                    initialSection: mapSection(deepLinkSection),
                    initialItemId: deepLinkItemId
                )
            }
            .task { await load() }
            .onChange(of: realtime.coursesRevision) { _, _ in
                Task { await load(force: true) }
            }
            .onChange(of: realtime.enrollmentsRevision) { _, _ in
                Task { await load(force: true) }
            }
            .onChange(of: shell.pendingDeepLink) { _, link in
                guard let link else { return }
                if case let .course(code, section, itemId) = link {
                    deepLinkSection = section
                    deepLinkItemId = itemId
                    Task { await openCourse(code: code) }
                    shell.pendingDeepLink = nil
                }
            }
        }
    }

    private func mapSection(_ section: CourseDeepLinkSection?) -> CourseWorkspaceSection? {
        guard let section else { return nil }
        return CourseWorkspaceSection.from(deepLink: section)
    }

    private func openCourse(code: String) async {
        guard let token = session.accessToken else { return }
        do {
            let course = try await LMSAPI.fetchCourse(courseCode: code, accessToken: token)
            deepLinkedCourse = course
        } catch {
            shell.openDeepLink(.home)
        }
    }

    private func load(force: Bool = false) async {
        guard let token = session.accessToken else { return }
        if loadedOnce && !force { return }
        loading = true
        errorMessage = nil
        defer {
            loading = false
            loadedOnce = true
        }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.courses(),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourses(accessToken: token)
            }
            courses = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load courses."
        }
    }
}

struct CourseRowCard: View {
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    var body: some View {
        LMSCard {
            HStack(alignment: .top, spacing: 14) {
                LMSCoverTile(key: course.courseCode, systemImage: "book.fill", size: 52)

                VStack(alignment: .leading, spacing: 4) {
                    Text(course.displayTitle)
                        .font(LexturesTheme.displayFont(16))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(course.courseCode.uppercased())
                        .font(.caption2.weight(.semibold))
                        .tracking(0.8)
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    if !course.description.isEmpty {
                        Text(course.description)
                            .font(.caption)
                            .lineLimit(2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }

                Spacer(minLength: 0)

                Image(systemName: "chevron.right")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
                    .padding(.top, 18)
            }
        }
    }
}
