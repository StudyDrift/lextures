import SwiftUI

/// Full-screen universal search + command palette (M0.6).
struct UniversalSearchView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    var courseScope: String? = nil

    @State private var query = ""
    @State private var scopedToCourse = false
    @State private var sections: [SearchResultSection] = []
    @State private var loading = false
    @State private var errorMessage: String?
    @State private var isOffline = false
    @State private var searchTask: Task<Void, Never>?
    private var trimmedQuery: String {
        query.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var effectiveScope: String? {
        guard scopedToCourse, let courseScope else { return nil }
        return courseScope
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                VStack(spacing: 0) {
                    if isOffline {
                        offlineBanner
                    }
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                            .padding(.horizontal, 16)
                            .padding(.top, 8)
                    }

                    if loading && sections.isEmpty {
                        LMSSkeletonList(count: 5)
                            .padding(16)
                    } else if sections.isEmpty && !trimmedQuery.isEmpty && !loading {
                        noResultsState
                    } else {
                        resultsList
                    }
                }
            }
            .navigationTitle(L.text("mobile.search.title"))
            .navigationBarTitleDisplayMode(.inline)
            .searchable(text: $query, placement: .navigationBarDrawer(displayMode: .always), prompt: L.text("mobile.search.prompt"))
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.ia.close")) { dismiss() }
                }
                if courseScope != nil {
                    ToolbarItem(placement: .primaryAction) {
                        scopeToggle
                    }
                }
            }
            .onAppear {
                reloadEmptyState()
            }
            .onChange(of: query) { _, newValue in
                scheduleSearch(newValue)
            }
            .onChange(of: scopedToCourse) { _, _ in
                scheduleSearch(query)
            }
        }
    }

    private var scopeToggle: some View {
        Button {
            scopedToCourse.toggle()
        } label: {
            Text(L.text("mobile.search.inThisCourse"))
                .font(.caption.weight(.semibold))
                .padding(.horizontal, 10)
                .padding(.vertical, 6)
                .background(
                    scopedToCourse
                        ? AnyShapeStyle(LexturesTheme.accent(for: colorScheme))
                        : AnyShapeStyle(LexturesTheme.cardBackground(for: colorScheme))
                )
                .foregroundStyle(
                    scopedToCourse
                        ? (colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
                        : LexturesTheme.textSecondary(for: colorScheme)
                )
                .clipShape(Capsule())
                .overlay(
                    Capsule().stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: scopedToCourse ? 0 : 1)
                )
        }
        .buttonStyle(.plain)
        .accessibilityLabel(L.text("mobile.search.inThisCourse"))
        .accessibilityAddTraits(scopedToCourse ? [.isSelected] : [])
    }

    private var offlineBanner: some View {
        Label(L.text("mobile.search.offlineNotice"), systemImage: "wifi.slash")
            .font(.subheadline)
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(12)
            .background(LexturesTheme.cardBackground(for: colorScheme))
            .padding(.horizontal, 16)
            .padding(.top, 8)
    }

    private var noResultsState: some View {
        LMSEmptyState(
            systemImage: "magnifyingglass",
            title: L.text("mobile.search.noResultsTitle"),
            message: L.format("mobile.search.noResultsMessage", trimmedQuery)
        )
    }

    private var resultsList: some View {
        List {
            ForEach(sections) { section in
                Section {
                    ForEach(section.items) { item in
                        Button {
                            select(item)
                        } label: {
                            SearchResultRow(item: item)
                        }
                        .buttonStyle(.plain)
                    }
                } header: {
                    Text(section.group.label)
                        .font(.subheadline.weight(.semibold))
                        .accessibilityAddTraits(.isHeader)
                }
            }
        }
        .listStyle(.plain)
        .scrollContentBackground(.hidden)
    }

    private func reloadEmptyState() {
        isOffline = !NetworkMonitor.shared.isOnline
        guard trimmedQuery.isEmpty else { return }
        sections = buildRecentsSections()
    }

    private func buildRecentsSections() -> [SearchResultSection] {
        var out: [SearchResultSection] = []
        let searches = SearchRecentsStore.recentSearches()
        if !searches.isEmpty {
            out.append(
                SearchResultSection(
                    group: .recentSearch,
                    items: searches.map { term in
                        SearchListItem(
                            id: "recent-search:\(term)",
                            group: .recentSearch,
                            title: term,
                            subtitle: L.text("mobile.search.recentSearchSubtitle"),
                            path: "",
                            haystack: term.lowercased()
                        )
                    }
                )
            )
        }
        let destinations = SearchRecentsStore.recentDestinations()
        if !destinations.isEmpty {
            out.append(SearchResultSection(group: .recentDestination, items: destinations))
        }
        return out
    }

    private func scheduleSearch(_ value: String) {
        searchTask?.cancel()
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            loading = false
            errorMessage = nil
            reloadEmptyState()
            return
        }
        searchTask = Task {
            try? await Task.sleep(nanoseconds: UInt64(SearchQueryEngine.debounceMilliseconds) * 1_000_000)
            guard !Task.isCancelled else { return }
            await performSearch(trimmed)
        }
    }

    @MainActor
    private func performSearch(_ trimmed: String) async {
        isOffline = !NetworkMonitor.shared.isOnline
        if isOffline {
            sections = buildRecentsSections()
            errorMessage = nil
            loading = false
            return
        }

        guard SearchQueryEngine.shouldQuery(trimmed) else {
            sections = buildRecentsSections()
            loading = false
            return
        }

        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil

        do {
            async let queryResponse = LMSAPI.fetchSearchQuery(
                query: trimmed,
                scope: effectiveScope,
                accessToken: token
            )
            let actions = SearchActionRegistry.buildActions(
                context: shell.activeRoleContext,
                platform: shell.platformFeatures
            )
            let matchedActions = SearchActionRegistry.matchActions(query: trimmed, actions: actions)
            let response = try await queryResponse
            var built = mapServerGroups(response.groups)
            if !matchedActions.isEmpty {
                built.insert(SearchResultSection(group: .action, items: matchedActions), at: 0)
            }
            sections = built
        } catch {
            if !Task.isCancelled {
                errorMessage = L.text("mobile.search.error")
                sections = buildRecentsSections()
            }
        }
        loading = false
    }

    private func mapServerGroups(_ groups: [SearchQueryGroup]) -> [SearchResultSection] {
        let groupMap: [String: SearchResultGroup] = [
            "course": .course,
            "content": .content,
            "person": .person,
        ]
        return groups.compactMap { group in
            guard let mapped = groupMap[group.type], !group.items.isEmpty else { return nil }
            let items = group.items.map { row in
                SearchListItem(
                    id: row.id,
                    group: mapped,
                    title: row.title,
                    subtitle: row.subtitle,
                    path: row.path
                )
            }
            return SearchResultSection(group: mapped, items: items)
        }
    }

    private func select(_ item: SearchListItem) {
        if item.group == .recentSearch {
            query = item.title
            return
        }
        if !item.path.isEmpty {
            SearchRecentsStore.recordDestination(item)
            if !trimmedQuery.isEmpty {
                SearchRecentsStore.recordSearch(trimmedQuery)
            }
            shell.navigateFromSearch(path: item.path)
        }
        dismiss()
    }
}

private struct SearchResultRow: View {
    @Environment(\.colorScheme) private var colorScheme
    let item: SearchListItem

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: iconName)
                .font(.body.weight(.semibold))
                .foregroundStyle(iconColor)
                .frame(width: 32, height: 32)
                .background(iconColor.opacity(0.14))
                .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))

            VStack(alignment: .leading, spacing: 2) {
                Text(item.title)
                    .font(.body.weight(.medium))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .lineLimit(1)
                if !item.subtitle.isEmpty {
                    Text(item.subtitle)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .lineLimit(2)
                }
            }
            Spacer(minLength: 0)
        }
        .frame(minHeight: 44)
        .accessibilityElement(children: .combine)
    }

    private var iconName: String {
        switch item.group {
        case .action, .recentDestination: return "bolt.fill"
        case .course: return "books.vertical.fill"
        case .content: return "doc.text.fill"
        case .person: return "person.fill"
        case .recentSearch: return "clock.arrow.circlepath"
        }
    }

    private var iconColor: Color {
        item.group == .action ? LexturesTheme.coral : LexturesTheme.accent(for: colorScheme)
    }
}