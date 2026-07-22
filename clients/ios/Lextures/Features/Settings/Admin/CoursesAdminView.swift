import SwiftUI

struct CoursesAdminRoute: Hashable {}

struct CoursesAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var searchText = ""
    @State private var submittedQuery = ""
    @State private var page = 1
    @State private var results: PaginatedPlatformCourses?
    @State private var loading = false
    @State private var errorMessage: String?
    @State private var stats: CoursesDashboardStats?
    @State private var statsLoading = true
    @State private var statsError: String?
    @State private var selectedFilter: CoursesListFilter?
    @State private var filterPage = 1
    @State private var filterResults: PaginatedPlatformCourses?
    @State private var filterLoading = false
    @State private var filterError: String?
    @State private var openingCourseId: String?

    private var features: MobilePlatformFeatures { shell.platformFeatures }

    var body: some View {
        Group {
            if !PlatformCoursesAdminLogic.canView(features: features, permissions: shell.permissions) {
                LMSEmptyState(systemImage: "lock.fill", title: L.text("mobile.admin.courses.accessDeniedTitle"), message: L.text("mobile.admin.courses.accessDeniedMessage")).padding(16)
            } else { content }
        }
        .navigationTitle(L.text("mobile.admin.courses.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await loadStats() }
        .refreshable {
            await loadStats()
            if selectedFilter != nil { await loadFilter() }
            if PlatformCoursesAdminLogic.shouldSearch(submittedQuery) { await search() }
        }
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.courses.description")).font(.subheadline).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Button { openURL(AppConfiguration.webURL(path: PlatformCoursesAdminLogic.webSettingsPath())) } label: {
                        LMSCard {
                            HStack {
                                Image(systemName: "safari").foregroundStyle(LexturesTheme.brandTeal)
                                VStack(alignment: .leading) {
                                    Text(L.text("mobile.admin.courses.webTitle")).font(.subheadline.weight(.semibold))
                                    Text(L.text("mobile.admin.courses.webHint")).font(.caption).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                }
                                Spacer()
                                Image(systemName: "arrow.up.right").font(.caption)
                            }
                        }
                    }.buttonStyle(.plain)
                    if let statsError { LMSErrorBanner(message: statsError) }
                    AdminMetricCardsGrid(
                        definitions: PlatformCoursesAdminLogic.metricDefinitions,
                        selected: selectedFilter,
                        loading: statsLoading,
                        value: { def in stats.map { PlatformCoursesAdminLogic.value(for: def.filter, in: $0) } },
                        title: { L.text($0.titleKey) },
                        hint: { $0.hintKey.map { L.text($0) } },
                        systemImage: { $0.systemImage },
                        onSelect: { filter in
                            selectedFilter = PlatformCoursesAdminLogic.toggleFilter(current: selectedFilter, tapped: filter)
                            filterPage = 1; filterResults = nil; filterError = nil
                            if selectedFilter != nil { Task { await loadFilter() } }
                        },
                        hintLine: L.text("mobile.admin.courses.metric.hint")
                    )
                    if let filter = selectedFilter, let metric = PlatformCoursesAdminLogic.metric(for: filter) {
                        VStack(alignment: .leading, spacing: 12) {
                            HStack(alignment: .top) {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(L.text(metric.tableTitleKey)).font(LexturesTheme.displayFont(17))
                                    Text(L.text(metric.tableDescriptionKey)).font(.caption).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    if let filterResults, !filterLoading {
                                        Text(L.format("mobile.admin.courses.resultsCount", Int(filterResults.total))).font(.caption.weight(.semibold))
                                    }
                                }
                                Spacer()
                                Button(L.text("mobile.admin.metric.close")) { selectedFilter = nil; filterResults = nil; filterError = nil }
                                    .font(.subheadline.weight(.semibold))
                            }
                            if let filterError { LMSErrorBanner(message: filterError) }
                            if filterLoading && filterResults == nil { LMSSkeletonList(count: 3) }
                            else if let filterResults { courseList(filterResults, isFilter: true) }
                        }
                        .padding(14)
                        .background(LexturesTheme.cardBackground(for: colorScheme))
                        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
                    }
                    LMSSectionHeader(title: L.text("mobile.admin.courses.searchSection"), systemImage: "magnifyingglass")
                    HStack {
                        TextField(L.text("mobile.admin.courses.search"), text: $searchText)
                            .textInputAutocapitalization(.never).autocorrectionDisabled().submitLabel(.search)
                            .onSubmit { submittedQuery = PlatformCoursesAdminLogic.normalizedSearchQuery(searchText); page = 1; Task { await search() } }
                        Button(L.text("mobile.admin.courses.searchAction")) {
                            submittedQuery = PlatformCoursesAdminLogic.normalizedSearchQuery(searchText); page = 1; Task { await search() }
                        }.buttonStyle(.borderedProminent).tint(LexturesTheme.brandTeal)
                    }
                    if let errorMessage { LMSErrorBanner(message: errorMessage) }
                    if loading && results == nil { LMSSkeletonList(count: 3) }
                    else if let results { courseList(results, isFilter: false) }
                    else if !PlatformCoursesAdminLogic.shouldSearch(submittedQuery) {
                        LMSEmptyState(systemImage: "books.vertical", title: L.text("mobile.admin.courses.emptyTitle"), message: L.text("mobile.admin.courses.emptyMessage"))
                    }
                }
                .padding(16)
            }
        }
    }

    private func courseList(_ data: PaginatedPlatformCourses, isFilter: Bool) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            if data.items.isEmpty {
                LMSEmptyState(systemImage: "magnifyingglass",
                    title: isFilter ? L.text("mobile.admin.courses.metric.emptyTitle") : L.text("mobile.admin.courses.emptyTitle"),
                    message: isFilter ? L.text("mobile.admin.courses.metric.emptyMessage") : L.text("mobile.admin.courses.emptySearch"))
            } else {
                if !isFilter {
                    Text(L.format("mobile.admin.courses.resultsCount", Int(data.total))).font(.caption).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                ForEach(data.items) { course in
                    Button { Task { await openCourse(course) } } label: {
                        LMSCard {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(course.title).font(.headline)
                                Text(course.courseCode).font(.caption.weight(.semibold)).foregroundStyle(LexturesTheme.accent(for: colorScheme))
                                Text("\(course.orgName) · \(PlatformCoursesAdminLogic.statusLabel(course.status)) · \(L.format("mobile.admin.courses.enrollments", Int(course.enrollmentCount)))")
                                    .font(.caption2).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                    }.buttonStyle(.plain).disabled(openingCourseId == course.id)
                }
                if data.totalPages > 1 {
                    HStack {
                        Button(L.text("mobile.admin.people.previous")) {
                            if isFilter { filterPage = max(1, filterPage - 1); Task { await loadFilter() } }
                            else { page = max(1, page - 1); Task { await search() } }
                        }.disabled((isFilter ? filterPage : page) <= 1)
                        Spacer()
                        Text(L.format("mobile.admin.people.pageOf", isFilter ? filterPage : page, data.totalPages)).font(.caption)
                        Spacer()
                        Button(L.text("mobile.admin.people.next")) {
                            if isFilter { filterPage = min(data.totalPages, filterPage + 1); Task { await loadFilter() } }
                            else { page = min(data.totalPages, page + 1); Task { await search() } }
                        }.disabled((isFilter ? filterPage : page) >= data.totalPages)
                    }
                }
            }
        }
    }

    private func loadStats() async {
        guard let token = session.accessToken else { return }
        statsLoading = true; statsError = nil; defer { statsLoading = false }
        do { stats = try await LMSAPI.fetchCoursesStats(accessToken: token) }
        catch { statsError = PlatformCoursesAdminLogic.userFacingError(error) }
    }

    private func loadFilter() async {
        guard let token = session.accessToken, let selectedFilter else { return }
        filterLoading = true; filterError = nil; defer { filterLoading = false }
        do {
            filterResults = try await LMSAPI.searchPlatformCourses(filter: selectedFilter, page: filterPage, perPage: PlatformCoursesAdminLogic.defaultPerPage, accessToken: token)
        } catch { filterError = PlatformCoursesAdminLogic.userFacingError(error); filterResults = nil }
    }

    private func search() async {
        guard let token = session.accessToken else { return }
        guard PlatformCoursesAdminLogic.shouldSearch(submittedQuery) else { results = nil; return }
        loading = true; errorMessage = nil; defer { loading = false }
        do {
            results = try await LMSAPI.searchPlatformCourses(query: submittedQuery, page: page, perPage: PlatformCoursesAdminLogic.defaultPerPage, accessToken: token)
        } catch { errorMessage = PlatformCoursesAdminLogic.userFacingError(error); results = nil }
    }

    private func openCourse(_ course: PlatformCourseRow) async {
        guard let token = session.accessToken else { return }
        openingCourseId = course.id
        defer { openingCourseId = nil }
        do {
            let report = try await LMSAPI.ensurePlatformCourseAdminAccess(courseId: course.id, accessToken: token)
            openURL(AppConfiguration.webURL(path: PlatformCoursesAdminLogic.courseWebPath(courseCode: report.courseCode)))
        } catch {
            openURL(AppConfiguration.webURL(path: PlatformCoursesAdminLogic.courseWebPath(courseCode: course.courseCode)))
        }
    }
}
