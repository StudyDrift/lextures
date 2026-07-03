import SwiftUI

enum CatalogRoute: Hashable {
    case course(String)
}

/// Browse and search public courses and learning paths (M9.1).
struct CatalogView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var tab: CatalogBrowseTab = .courses
    @State private var query = ""
    @State private var category = ""
    @State private var level: CatalogLevelFilter = .any
    @State private var price: CatalogPriceFilter = .any
    @State private var sort: CatalogSortMode = .popular
    @State private var categories: [CatalogCategory] = []
    @State private var courses: [PublicCatalogCourse] = []
    @State private var nextCursor = ""
    @State private var loading = false
    @State private var loadingMore = false
    @State private var errorMessage: String?
    @State private var hasSearched = false

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            VStack(alignment: .leading, spacing: 12) {
                if shell.platformFeatures.ffLearningPaths {
                    LMSSegmentedChips(
                        options: CatalogBrowseTab.allCases,
                        selection: $tab,
                        label: { L.text(String.LocalizationValue($0.labelKey)) }
                    )
                }

                if tab == .courses {
                    coursesBody
                } else {
                    PathsCatalogView(embedded: true)
                }
            }
            .padding(16)
        }
        .navigationTitle(L.text("mobile.catalog.title"))
        .navigationBarTitleDisplayMode(.inline)
        .searchable(text: $query, prompt: L.text("mobile.catalog.search"))
        .onSubmit(of: .search) { Task { await searchCourses(reset: true) } }
        .onChange(of: tab) { _, newTab in
            if newTab == .courses { Task { await searchCourses(reset: true) } }
        }
        .onChange(of: category) { _, _ in Task { await searchCourses(reset: true) } }
        .onChange(of: level) { _, _ in Task { await searchCourses(reset: true) } }
        .onChange(of: price) { _, _ in Task { await searchCourses(reset: true) } }
        .onChange(of: sort) { _, _ in Task { await searchCourses(reset: true) } }
        .navigationDestination(for: CatalogRoute.self) { route in
            switch route {
            case .course(let slug):
                CourseLandingView(slug: slug)
            }
        }
        .task {
            await loadCategories()
            await searchCourses(reset: true)
        }
    }

    @ViewBuilder
    private var coursesBody: some View {
        filterChips

        if let errorMessage {
            LMSErrorBanner(message: errorMessage)
        }

        if loading && courses.isEmpty {
            LMSSkeletonList(count: 4)
        } else if courses.isEmpty {
            LMSEmptyState(
                systemImage: hasSearched ? "magnifyingglass" : "books.vertical",
                title: L.text("mobile.catalog.emptyTitle"),
                message: hasSearched
                    ? L.text("mobile.catalog.emptyMessage")
                    : L.text("mobile.catalog.prompt")
            )
        } else {
            ScrollView {
                LazyVStack(spacing: 12) {
                    ForEach(courses) { course in
                        NavigationLink(value: CatalogRoute.course(course.slug)) {
                            courseCard(course)
                        }
                        .buttonStyle(.plain)
                    }
                    if !nextCursor.isEmpty {
                        Button {
                            Task { await loadMore() }
                        } label: {
                            Text(loadingMore ? L.text("mobile.catalog.loadingMore") : L.text("mobile.catalog.loadMore"))
                                .font(.subheadline.weight(.semibold))
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 12)
                        }
                        .disabled(loadingMore)
                    }
                }
            }
        }
    }

    @ViewBuilder
    private var filterChips: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                Menu {
                    Button(L.text("mobile.catalog.filter.categoryAny")) { category = "" }
                    ForEach(categories) { item in
                        Button(item.category) { category = item.category }
                    }
                } label: {
                    filterChip(
                        category.isEmpty ? L.text("mobile.catalog.filter.categoryAny") : category,
                        active: !category.isEmpty
                    )
                }

                Menu {
                    ForEach(CatalogLevelFilter.allCases) { item in
                        Button(L.text(String.LocalizationValue(item.labelKey))) { level = item }
                    }
                } label: {
                    filterChip(L.text(String.LocalizationValue(level.labelKey)), active: level != .any)
                }

                Menu {
                    ForEach(CatalogPriceFilter.allCases) { item in
                        Button(L.text(String.LocalizationValue(item.labelKey))) { price = item }
                    }
                } label: {
                    filterChip(L.text(String.LocalizationValue(price.labelKey)), active: price != .any)
                }

                Menu {
                    ForEach(CatalogSortMode.allCases) { item in
                        Button(L.text(String.LocalizationValue(item.labelKey))) { sort = item }
                    }
                } label: {
                    filterChip(L.text(String.LocalizationValue(sort.labelKey)), active: sort != .popular)
                }
            }
        }
    }

    private func filterChip(_ title: String, active: Bool) -> some View {
        Text(title)
            .font(.caption.weight(.semibold))
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(
                Capsule().fill(
                    active
                        ? LexturesTheme.accent(for: colorScheme).opacity(0.15)
                        : LexturesTheme.cardBackground(for: colorScheme)
                )
            )
            .foregroundStyle(
                active
                    ? LexturesTheme.accent(for: colorScheme)
                    : LexturesTheme.textSecondary(for: colorScheme)
            )
    }

    private func courseCard(_ course: PublicCatalogCourse) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                CourseHeroImage(urlString: course.heroImageUrl, fallbackKey: course.courseCode, height: 120)
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))

                HStack(spacing: 6) {
                    if let category = course.category, !category.isEmpty {
                        Text(category)
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    if let level = course.difficultyLevel, !level.isEmpty {
                        Text(level.capitalized)
                            .font(.caption2.weight(.semibold))
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Capsule().fill(LexturesTheme.cardBackground(for: colorScheme)))
                    }
                }

                Text(course.title)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .lineLimit(2)

                if let instructor = course.instructorName, !instructor.isEmpty {
                    Text(instructor)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                HStack {
                    Text(CatalogLogic.ratingLabel(average: course.averageRating, count: course.ratingCount))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Spacer()
                    Text(CatalogLogic.formatPrice(cents: course.priceCents))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func loadCategories() async {
        do {
            categories = try await LMSAPI.fetchPublicCatalogCategories(accessToken: session.accessToken)
        } catch {
            categories = []
        }
    }

    private func searchCourses(reset: Bool) async {
        guard tab == .courses else { return }
        if reset {
            loading = true
            nextCursor = ""
        }
        errorMessage = nil
        hasSearched = true
        defer { if reset { loading = false } }

        do {
            let cacheKey = CatalogLogic.cacheKey(
                query: query,
                category: category,
                level: level,
                price: price,
                sort: sort
            )
            let response: PublicCatalogSearchResponse
            if let token = session.accessToken {
                response = try await OfflineService.shared.cachedFetch(
                    key: OfflineCacheKey.catalogCourses(key: cacheKey),
                    accessToken: token
                ) {
                    try await LMSAPI.fetchPublicCatalogCourses(
                        query: query,
                        category: category,
                        level: level.queryValue ?? "",
                        sort: sort.rawValue,
                        priceMax: price.priceMax,
                        accessToken: token
                    )
                }.value
            } else {
                response = try await LMSAPI.fetchPublicCatalogCourses(
                    query: query,
                    category: category,
                    level: level.queryValue ?? "",
                    sort: sort.rawValue,
                    priceMax: price.priceMax,
                    accessToken: nil
                )
            }
            var page = response.courses ?? []
            if price == .paid {
                page = page.filter { CatalogLogic.isPaid(priceCents: $0.priceCents) }
            }
            courses = page
            nextCursor = response.nextCursor ?? ""
        } catch {
            errorMessage = L.text("mobile.catalog.error")
            if reset { courses = [] }
        }
    }

    private func loadMore() async {
        guard !nextCursor.isEmpty else { return }
        loadingMore = true
        defer { loadingMore = false }
        do {
            let response = try await LMSAPI.fetchPublicCatalogCourses(
                query: query,
                category: category,
                level: level.queryValue ?? "",
                sort: sort.rawValue,
                priceMax: price.priceMax,
                cursor: nextCursor,
                accessToken: session.accessToken
            )
            var page = response.courses ?? []
            if price == .paid {
                page = page.filter { CatalogLogic.isPaid(priceCents: $0.priceCents) }
            }
            courses.append(contentsOf: page)
            nextCursor = response.nextCursor ?? ""
        } catch {
            errorMessage = L.text("mobile.catalog.error")
        }
    }
}