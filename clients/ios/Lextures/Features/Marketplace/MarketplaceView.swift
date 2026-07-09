import SwiftUI

enum MarketplaceRoute: Hashable {
    case course(String)
}

/// Authenticated marketplace storefront browse (MKT6).
struct MarketplaceView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var query = ""
    @State private var category = ""
    @State private var level: MarketplaceLevelFilter = .any
    @State private var price: MarketplacePriceFilter = .any
    @State private var sort: MarketplaceSortMode = .popular
    @State private var categories: [MarketplaceCategory] = []
    @State private var courses: [MarketplaceCourse] = []
    @State private var nextCursor = ""
    @State private var loading = false
    @State private var loadingMore = false
    @State private var errorMessage: String?
    @State private var hasSearched = false

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            VStack(alignment: .leading, spacing: 12) {
                filterChips

                if let errorMessage {
                    LMSErrorBanner(message: errorMessage)
                }

                if loading && courses.isEmpty {
                    LMSSkeletonList(count: 4)
                } else if courses.isEmpty {
                    LMSEmptyState(
                        systemImage: hasSearched ? "magnifyingglass" : "bag",
                        title: L.text("mobile.marketplace.emptyTitle"),
                        message: hasSearched
                            ? L.text("mobile.marketplace.emptyMessage")
                            : L.text("mobile.marketplace.prompt")
                    )
                } else {
                    ScrollView {
                        LazyVStack(spacing: 12) {
                            ForEach(courses) { course in
                                NavigationLink(value: MarketplaceRoute.course(course.slug)) {
                                    courseCard(course)
                                }
                                .buttonStyle(.plain)
                            }
                            if !nextCursor.isEmpty {
                                Button {
                                    Task { await loadMore() }
                                } label: {
                                    Text(loadingMore
                                        ? L.text("mobile.marketplace.loadingMore")
                                        : L.text("mobile.marketplace.loadMore"))
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
            .padding(16)
        }
        .navigationTitle(L.text("mobile.marketplace.title"))
        .navigationBarTitleDisplayMode(.inline)
        .searchable(text: $query, prompt: L.text("mobile.marketplace.search"))
        .onSubmit(of: .search) { Task { await searchCourses(reset: true) } }
        .onChange(of: category) { _, _ in Task { await searchCourses(reset: true) } }
        .onChange(of: level) { _, _ in Task { await searchCourses(reset: true) } }
        .onChange(of: price) { _, _ in Task { await searchCourses(reset: true) } }
        .onChange(of: sort) { _, _ in Task { await searchCourses(reset: true) } }
        .navigationDestination(for: MarketplaceRoute.self) { route in
            switch route {
            case .course(let slug):
                MarketplaceDetailView(slug: slug)
            }
        }
        .task {
            await loadCategories()
            await searchCourses(reset: true)
        }
    }

    @ViewBuilder
    private var filterChips: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                Menu {
                    Button(L.text("mobile.marketplace.filter.categoryAny")) { category = "" }
                    ForEach(categories) { item in
                        Button(item.category) { category = item.category }
                    }
                } label: {
                    filterChip(
                        category.isEmpty ? L.text("mobile.marketplace.filter.categoryAny") : category,
                        active: !category.isEmpty
                    )
                }

                Menu {
                    ForEach(MarketplaceLevelFilter.allCases) { item in
                        Button(L.text(String.LocalizationValue(item.labelKey))) { level = item }
                    }
                } label: {
                    filterChip(L.text(String.LocalizationValue(level.labelKey)), active: level != .any)
                }

                Menu {
                    ForEach(MarketplacePriceFilter.allCases) { item in
                        Button(L.text(String.LocalizationValue(item.labelKey))) { price = item }
                    }
                } label: {
                    filterChip(L.text(String.LocalizationValue(price.labelKey)), active: price != .any)
                }

                Menu {
                    ForEach(MarketplaceSortMode.allCases) { item in
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

    private func courseCard(_ course: MarketplaceCourse) -> some View {
        let freeLabel = L.text("mobile.marketplace.free")
        let ownedLabel = L.text("mobile.marketplace.owned")
        let priceLabel = MarketplaceLogic.formatPrice(
            cents: course.priceCents,
            currency: course.priceCurrency,
            freeLabel: freeLabel
        )
        let a11y = MarketplaceLogic.cardAccessibleName(
            title: course.title,
            priceLabel: priceLabel,
            owned: course.owned,
            ownedLabel: ownedLabel
        )
        return LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                CourseHeroImage(urlString: course.heroImageUrl, fallbackKey: course.courseCode, height: 120)
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))

                HStack(spacing: 6) {
                    if let category = course.category, !category.isEmpty {
                        Text(category)
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    if let level = course.level, !level.isEmpty {
                        Text(level.capitalized)
                            .font(.caption2.weight(.semibold))
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Capsule().fill(LexturesTheme.cardBackground(for: colorScheme)))
                    }
                    if course.owned {
                        Text(ownedLabel)
                            .font(.caption2.weight(.semibold))
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Capsule().fill(LexturesTheme.accent(for: colorScheme).opacity(0.15)))
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
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
                    Spacer()
                    Text(priceLabel)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .accessibilityElement(children: .combine)
            .accessibilityLabel(a11y)
        }
    }

    private func loadCategories() async {
        guard let token = session.accessToken else { return }
        do {
            categories = try await LMSAPI.fetchMarketplaceCategories(accessToken: token)
        } catch {
            categories = []
        }
    }

    private func searchCourses(reset: Bool) async {
        guard let token = session.accessToken else { return }
        if reset {
            loading = true
            nextCursor = ""
        }
        errorMessage = nil
        hasSearched = true
        defer { if reset { loading = false } }

        do {
            let cacheKey = MarketplaceLogic.cacheKey(
                query: query,
                category: category,
                level: level,
                price: price,
                sort: sort
            )
            let response = try await OfflineService.shared.cachedFetch(
                key: OfflineCacheKey.marketplaceCourses(key: cacheKey),
                accessToken: token
            ) {
                try await LMSAPI.fetchMarketplaceCourses(
                    query: query,
                    category: category,
                    level: level.queryValue ?? "",
                    sort: sort.rawValue,
                    priceMax: price.priceMax,
                    freeOnly: price.freeOnly,
                    accessToken: token
                )
            }.value
            var page = response.courses ?? []
            if price == .paid {
                page = page.filter { MarketplaceLogic.isPaid(priceCents: $0.priceCents) }
            }
            courses = page
            nextCursor = response.nextCursor ?? ""
        } catch {
            errorMessage = L.text("mobile.marketplace.error")
            if reset { courses = [] }
        }
    }

    private func loadMore() async {
        guard let token = session.accessToken, !nextCursor.isEmpty else { return }
        loadingMore = true
        defer { loadingMore = false }
        do {
            let response = try await LMSAPI.fetchMarketplaceCourses(
                query: query,
                category: category,
                level: level.queryValue ?? "",
                sort: sort.rawValue,
                priceMax: price.priceMax,
                freeOnly: price.freeOnly,
                cursor: nextCursor,
                accessToken: token
            )
            var page = response.courses ?? []
            if price == .paid {
                page = page.filter { MarketplaceLogic.isPaid(priceCents: $0.priceCents) }
            }
            courses.append(contentsOf: page)
            nextCursor = response.nextCursor ?? ""
        } catch {
            errorMessage = L.text("mobile.marketplace.error")
        }
    }
}
