import SwiftUI

struct PeopleAdminRoute: Hashable {}

struct UserDetailAdminRoute: Hashable {
    var userId: String
}

/// People admin: search, invite, and open user detail (M14.3).
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
    @State private var showInviteSheet = false

    private var features: MobilePlatformFeatures { shell.platformFeatures }

    var body: some View {
        Group {
            if !PeopleAdminLogic.canView(features: features, permissions: shell.permissions) {
                accessDenied
            } else {
                content
            }
        }
        .navigationTitle(L.text("mobile.admin.people.title"))
        .navigationBarTitleDisplayMode(.inline)
        .sheet(isPresented: $showInviteSheet) {
            PeopleInviteSheet(
                onInvited: { email in
                    searchText = email
                    submittedQuery = email
                    page = 1
                    Task { await search() }
                }
            )
        }
        .navigationDestination(for: UserDetailAdminRoute.self) { route in
            UserDetailAdminView(userId: route.userId)
        }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.people.accessDeniedTitle"),
            message: L.text("mobile.admin.people.accessDeniedMessage")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.people.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    webLinkCard

                    searchForm

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && results == nil {
                        LMSSkeletonList(count: 3)
                    } else if let results {
                        resultsSection(results)
                    } else if !PeopleAdminLogic.shouldSearch(submittedQuery) {
                        LMSEmptyState(
                            systemImage: "person.2",
                            title: L.text("mobile.admin.people.emptyTitle"),
                            message: L.text("mobile.admin.people.emptyMessage")
                        )
                    }
                }
                .padding(16)
            }
        }
    }

    private var webLinkCard: some View {
        Button {
            openURL(AppConfiguration.webURL(path: PeopleAdminLogic.webSettingsPath()))
        } label: {
            LMSCard {
                HStack(spacing: 10) {
                    Image(systemName: "safari")
                        .foregroundStyle(LexturesTheme.brandTeal)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L.text("mobile.admin.people.webTitle"))
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(L.text("mobile.admin.people.webHint"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer(minLength: 0)
                    Image(systemName: "arrow.up.right")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
        .buttonStyle(.plain)
    }

    private var searchForm: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                TextField(L.text("mobile.admin.people.search"), text: $searchText)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .submitLabel(.search)
                    .onSubmit { submitSearch() }

                Button(L.text("mobile.admin.people.search")) {
                    submitSearch()
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.brandTeal)
            }

            Button {
                showInviteSheet = true
            } label: {
                Label(L.text("mobile.admin.people.invite"), systemImage: "envelope.badge")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.bordered)
        }
    }

    private func resultsSection(_ data: PaginatedPeople) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            if data.items.isEmpty {
                LMSEmptyState(
                    systemImage: "magnifyingglass",
                    title: L.text("mobile.admin.people.emptyTitle"),
                    message: L.text("mobile.admin.people.emptySearch")
                )
            } else {
                Text(
                    L.format(
                        "mobile.admin.people.resultsCount",
                        Int(data.total)
                    )
                )
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                ForEach(data.items) { person in
                    NavigationLink(value: UserDetailAdminRoute(userId: person.id)) {
                        personRow(person)
                    }
                    .buttonStyle(.plain)
                }

                if data.totalPages > 1 {
                    paginationControls(data)
                }
            }
        }
    }

    private func personRow(_ person: PersonRow) -> some View {
        LMSCard {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(PeopleAdminLogic.personDisplayName(person))
                        .font(.headline)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(person.email)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    HStack(spacing: 8) {
                        Text(person.orgName)
                        Text("·")
                        Text(PeopleAdminLogic.statusLabel(active: person.active))
                    }
                    .font(.caption2)
                    .foregroundStyle(
                        person.active
                            ? LexturesTheme.textSecondary(for: colorScheme)
                            : LexturesTheme.brandCoral
                    )
                }
                Spacer(minLength: 0)
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func paginationControls(_ data: PaginatedPeople) -> some View {
        HStack {
            Button(L.text("mobile.admin.people.previous")) {
                page = max(1, page - 1)
                Task { await search() }
            }
            .disabled(page <= 1 || loading)

            Spacer()

            Text(
                L.format(
                    "mobile.admin.people.pageOf",
                    page,
                    data.totalPages
                )
            )
            .font(.caption)
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            Spacer()

            Button(L.text("mobile.admin.people.next")) {
                page = min(data.totalPages, page + 1)
                Task { await search() }
            }
            .disabled(page >= data.totalPages || loading)
        }
    }

    private func submitSearch() {
        submittedQuery = PeopleAdminLogic.normalizedSearchQuery(searchText)
        page = 1
        Task { await search() }
    }

    private func search() async {
        guard let token = session.accessToken else { return }
        guard PeopleAdminLogic.shouldSearch(submittedQuery) else {
            results = nil
            return
        }
        loading = true
        errorMessage = nil
        defer { loading = false }

        do {
            results = try await LMSAPI.searchPeople(
                query: submittedQuery,
                page: page,
                perPage: PeopleAdminLogic.defaultPerPage,
                accessToken: token
            )
        } catch {
            errorMessage = PeopleAdminLogic.userFacingError(error)
            results = nil
        }
    }
}

private struct PeopleInviteSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme

    @State private var email = ""
    @State private var firstName = ""
    @State private var lastName = ""
    @State private var busy = false
    @State private var errorMessage: String?

    let onInvited: (String) -> Void

    var body: some View {
        NavigationStack {
            Form {
                if let errorMessage {
                    Section {
                        Text(errorMessage)
                            .foregroundStyle(.red)
                            .font(.caption)
                    }
                }
                Section {
                    TextField(L.text("mobile.admin.people.inviteEmail"), text: $email)
                        .textInputAutocapitalization(.never)
                        .keyboardType(.emailAddress)
                        .autocorrectionDisabled()
                    TextField(L.text("mobile.admin.people.inviteFirstName"), text: $firstName)
                    TextField(L.text("mobile.admin.people.inviteLastName"), text: $lastName)
                }
            }
            .navigationTitle(L.text("mobile.admin.people.inviteTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(busy ? L.text("mobile.admin.people.loading") : L.text("mobile.admin.people.inviteSend")) {
                        Task { await invite() }
                    }
                    .disabled(busy || email.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
        }
        .presentationDetents([.medium])
    }

    private func invite() async {
        guard let token = session.accessToken else { return }
        busy = true
        errorMessage = nil
        defer { busy = false }

        let request = PeopleAdminLogic.invitePersonRequest(
            email: email,
            firstName: firstName,
            lastName: lastName
        )
        do {
            _ = try await LMSAPI.invitePerson(request, accessToken: token)
            onInvited(request.email)
            dismiss()
        } catch {
            errorMessage = PeopleAdminLogic.userFacingError(error)
        }
    }
}
