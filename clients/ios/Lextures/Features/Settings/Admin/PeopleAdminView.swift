import SwiftUI

struct PeopleAdminRoute: Hashable {}

struct UserDetailAdminRoute: Hashable {
    var userId: String
}

struct PeopleAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var searchText = ""
    @State private var submittedQuery = ""
    @State private var page = 1
    @State private var results: PaginatedPeople?
    @State private var loading = false
    @State private var errorMessage: String?
    @State private var stats: PeopleDashboardStats?
    @State private var statsLoading = true
    @State private var statsError: String?
    @State private var selectedFilter: PeopleListFilter?
    @State private var filterPage = 1
    @State private var filterResults: PaginatedPeople?
    @State private var filterLoading = false
    @State private var filterError: String?
    @State private var showInviteSheet = false

    private var features: MobilePlatformFeatures { shell.platformFeatures }

    var body: some View {
        Group {
            if !PeopleAdminLogic.canView(features: features, permissions: shell.permissions) {
                LMSEmptyState(systemImage: "lock.fill", title: L.text("mobile.admin.people.accessDeniedTitle"), message: L.text("mobile.admin.people.accessDeniedMessage")).padding(16)
            } else { content }
        }
        .navigationTitle(L.text("mobile.admin.people.title"))
        .navigationBarTitleDisplayMode(.inline)
        .sheet(isPresented: $showInviteSheet) {
            PeopleInviteSheet(onInvited: { email in
                searchText = email; submittedQuery = email; page = 1
                selectedFilter = nil; filterResults = nil
                Task { await loadStats(); await search() }
            })
        }
        .navigationDestination(for: UserDetailAdminRoute.self) { UserDetailAdminView(userId: $0.userId) }
        .task { await loadStats() }
        .refreshable {
            await loadStats()
            if selectedFilter != nil { await loadFilter() }
            if PeopleAdminLogic.shouldSearch(submittedQuery) { await search() }
        }
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.people.description")).font(.subheadline).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    webLink
                    if let statsError { LMSErrorBanner(message: statsError) }
                    AdminMetricCardsGrid(
                        definitions: PeopleAdminLogic.metricDefinitions,
                        selected: selectedFilter,
                        loading: statsLoading,
                        value: { def in stats.map { PeopleAdminLogic.value(for: def.filter, in: $0) } },
                        title: { L.text($0.titleKey) },
                        hint: { $0.hintKey.map { L.text($0) } },
                        systemImage: { $0.systemImage },
                        onSelect: { filter in
                            selectedFilter = PeopleAdminLogic.toggleFilter(current: selectedFilter, tapped: filter)
                            filterPage = 1; filterResults = nil; filterError = nil
                            if selectedFilter != nil { Task { await loadFilter() } }
                        },
                        hintLine: L.text("mobile.admin.people.metric.hint")
                    )
                    if let filter = selectedFilter, let metric = PeopleAdminLogic.metric(for: filter) {
                        filterPanel(metric)
                    }
                    searchForm
                    if let errorMessage { LMSErrorBanner(message: errorMessage) }
                    if loading && results == nil { LMSSkeletonList(count: 3) }
                    else if let results { peopleList(results, isFilter: false) }
                    else if !PeopleAdminLogic.shouldSearch(submittedQuery) {
                        LMSEmptyState(systemImage: "person.2", title: L.text("mobile.admin.people.emptyTitle"), message: L.text("mobile.admin.people.emptyMessage"))
                    }
                }
                .padding(16)
            }
        }
    }

    private func filterPanel(_ metric: PeopleAdminLogic.MetricDefinition) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(L.text(metric.tableTitleKey)).font(LexturesTheme.displayFont(17)).foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(L.text(metric.tableDescriptionKey)).font(.caption).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if let filterResults, !filterLoading {
                        Text(L.format("mobile.admin.people.resultsCount", Int(filterResults.total))).font(.caption.weight(.semibold))
                    }
                }
                Spacer()
                Button(L.text("mobile.admin.metric.close")) { selectedFilter = nil; filterResults = nil; filterError = nil }
                    .font(.subheadline.weight(.semibold))
            }
            if let filterError { LMSErrorBanner(message: filterError) }
            if filterLoading && filterResults == nil { LMSSkeletonList(count: 3) }
            else if let filterResults { peopleList(filterResults, isFilter: true) }
        }
        .padding(14)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
    }

    private var webLink: some View {
        Button { openURL(AppConfiguration.webURL(path: PeopleAdminLogic.webSettingsPath())) } label: {
            LMSCard {
                HStack {
                    Image(systemName: "safari").foregroundStyle(LexturesTheme.brandTeal)
                    VStack(alignment: .leading) {
                        Text(L.text("mobile.admin.people.webTitle")).font(.subheadline.weight(.semibold))
                        Text(L.text("mobile.admin.people.webHint")).font(.caption).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer()
                    Image(systemName: "arrow.up.right").font(.caption)
                }
            }
        }.buttonStyle(.plain)
    }

    private var searchForm: some View {
        VStack(alignment: .leading, spacing: 12) {
            LMSSectionHeader(title: L.text("mobile.admin.people.searchSection"), systemImage: "magnifyingglass")
            HStack {
                TextField(L.text("mobile.admin.people.search"), text: $searchText)
                    .textInputAutocapitalization(.never).autocorrectionDisabled().submitLabel(.search)
                    .onSubmit { submittedQuery = PeopleAdminLogic.normalizedSearchQuery(searchText); page = 1; Task { await search() } }
                Button(L.text("mobile.admin.people.search")) {
                    submittedQuery = PeopleAdminLogic.normalizedSearchQuery(searchText); page = 1; Task { await search() }
                }.buttonStyle(.borderedProminent).tint(LexturesTheme.brandTeal)
            }
            Button { showInviteSheet = true } label: {
                Label(L.text("mobile.admin.people.invite"), systemImage: "envelope.badge").frame(maxWidth: .infinity)
            }.buttonStyle(.bordered)
        }
    }

    private func peopleList(_ data: PaginatedPeople, isFilter: Bool) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            if data.items.isEmpty {
                LMSEmptyState(systemImage: "magnifyingglass",
                    title: isFilter ? L.text("mobile.admin.people.metric.emptyTitle") : L.text("mobile.admin.people.emptyTitle"),
                    message: isFilter ? L.text("mobile.admin.people.metric.emptyMessage") : L.text("mobile.admin.people.emptySearch"))
            } else {
                if !isFilter {
                    Text(L.format("mobile.admin.people.resultsCount", Int(data.total))).font(.caption).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                ForEach(data.items) { person in
                    NavigationLink(value: UserDetailAdminRoute(userId: person.id)) {
                        LMSCard {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(PeopleAdminLogic.personDisplayName(person)).font(.headline)
                                Text(person.email).font(.caption).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                Text("\(person.orgName) · \(PeopleAdminLogic.statusLabel(active: person.active))")
                                    .font(.caption2)
                                    .foregroundStyle(person.active ? LexturesTheme.textSecondary(for: colorScheme) : LexturesTheme.brandCoral)
                            }
                        }
                    }.buttonStyle(.plain)
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
        do { stats = try await LMSAPI.fetchPeopleStats(accessToken: token) }
        catch { statsError = PeopleAdminLogic.userFacingError(error) }
    }

    private func loadFilter() async {
        guard let token = session.accessToken, let selectedFilter else { return }
        filterLoading = true; filterError = nil; defer { filterLoading = false }
        do {
            filterResults = try await LMSAPI.searchPeople(filter: selectedFilter, page: filterPage, perPage: PeopleAdminLogic.defaultPerPage, accessToken: token)
        } catch { filterError = PeopleAdminLogic.userFacingError(error); filterResults = nil }
    }

    private func search() async {
        guard let token = session.accessToken else { return }
        guard PeopleAdminLogic.shouldSearch(submittedQuery) else { results = nil; return }
        loading = true; errorMessage = nil; defer { loading = false }
        do {
            results = try await LMSAPI.searchPeople(query: submittedQuery, page: page, perPage: PeopleAdminLogic.defaultPerPage, accessToken: token)
        } catch { errorMessage = PeopleAdminLogic.userFacingError(error); results = nil }
    }
}

private struct PeopleInviteSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var email = ""; @State private var firstName = ""; @State private var lastName = ""
    @State private var busy = false; @State private var errorMessage: String?
    let onInvited: (String) -> Void

    var body: some View {
        NavigationStack {
            Form {
                if let errorMessage { Section { Text(errorMessage).foregroundStyle(.red).font(.caption) } }
                Section {
                    TextField(L.text("mobile.admin.people.inviteEmail"), text: $email).textInputAutocapitalization(.never).keyboardType(.emailAddress).autocorrectionDisabled()
                    TextField(L.text("mobile.admin.people.inviteFirstName"), text: $firstName)
                    TextField(L.text("mobile.admin.people.inviteLastName"), text: $lastName)
                }
            }
            .navigationTitle(L.text("mobile.admin.people.inviteTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button(L.text("mobile.common.cancel")) { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button(busy ? L.text("mobile.admin.people.loading") : L.text("mobile.admin.people.inviteSend")) { Task { await invite() } }
                        .disabled(busy || email.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
        }
        .presentationDetents([.medium])
    }

    private func invite() async {
        guard let token = session.accessToken else { return }
        busy = true; errorMessage = nil; defer { busy = false }
        let request = PeopleAdminLogic.invitePersonRequest(email: email, firstName: firstName, lastName: lastName)
        do { _ = try await LMSAPI.invitePerson(request, accessToken: token); onInvited(request.email); dismiss() }
        catch { errorMessage = PeopleAdminLogic.userFacingError(error) }
    }
}
