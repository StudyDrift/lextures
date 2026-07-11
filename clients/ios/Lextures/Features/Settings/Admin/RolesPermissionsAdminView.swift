import SwiftUI

struct RolesPermissionsAdminRoute: Hashable {}

struct RolesPermissionsRoleDetailRoute: Hashable {
    var roleId: String
}

/// Roles & permissions admin: read roles, assign/remove users (M14.2).
struct RolesPermissionsAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var roles: [RoleWithPermissions] = []
    @State private var searchText = ""
    @State private var loading = true
    @State private var errorMessage: String?

    private var features: MobilePlatformFeatures { shell.platformFeatures }
    private var filteredRoles: [RoleWithPermissions] {
        RolesPermissionsAdminLogic.filterRoles(roles, query: searchText)
    }

    var body: some View {
        Group {
            if !RolesPermissionsAdminLogic.canView(features: features, permissions: shell.permissions) {
                accessDenied
            } else {
                content
            }
        }
        .navigationTitle(L.text("mobile.admin.roles.title"))
        .navigationBarTitleDisplayMode(.inline)
        .searchable(
            text: $searchText,
            prompt: Text(L.text("mobile.admin.roles.search"))
        )
        .refreshable { await load() }
        .task { await load() }
        .navigationDestination(for: RolesPermissionsRoleDetailRoute.self) { route in
            if let role = roles.first(where: { $0.id == route.roleId }) {
                RolesPermissionsRoleDetailView(role: role)
            } else {
                LMSEmptyState(
                    systemImage: "person.2.slash",
                    title: L.text("mobile.admin.roles.emptyTitle"),
                    message: L.text("mobile.admin.roles.error")
                )
            }
        }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.roles.accessDeniedTitle"),
            message: L.text("mobile.admin.roles.accessDeniedMessage")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.roles.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    webLinkCard

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && roles.isEmpty {
                        LMSSkeletonList(count: 3)
                    } else if roles.isEmpty {
                        LMSEmptyState(
                            systemImage: "person.2",
                            title: L.text("mobile.admin.roles.emptyTitle"),
                            message: L.text("mobile.admin.roles.emptyMessage")
                        )
                    } else if filteredRoles.isEmpty {
                        LMSEmptyState(
                            systemImage: "magnifyingglass",
                            title: L.text("mobile.admin.roles.emptyTitle"),
                            message: L.text("mobile.admin.roles.emptySearch")
                        )
                    } else {
                        ForEach(filteredRoles) { role in
                            NavigationLink(value: RolesPermissionsRoleDetailRoute(roleId: role.id)) {
                                roleRow(role)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
                .padding(16)
            }
        }
    }

    private var webLinkCard: some View {
        Button {
            openURL(AppConfiguration.webURL(path: RolesPermissionsAdminLogic.webSettingsPath()))
        } label: {
            LMSCard {
                HStack(spacing: 10) {
                    Image(systemName: "safari")
                        .foregroundStyle(LexturesTheme.brandTeal)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L.text("mobile.admin.roles.webTitle"))
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(L.text("mobile.admin.roles.webHint"))
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

    private func roleRow(_ role: RoleWithPermissions) -> some View {
        LMSCard {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(role.name)
                        .font(.headline)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    if let description = role.description?.trimmingCharacters(in: .whitespacesAndNewlines),
                       !description.isEmpty {
                        Text(description)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            .lineLimit(2)
                    }
                    Text(
                        L.format(
                            "mobile.admin.roles.permissionCount",
                            role.permissions.count
                        )
                    )
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer(minLength: 0)
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }

        do {
            roles = try await LMSAPI.fetchRoles(accessToken: token)
        } catch {
            errorMessage = RolesPermissionsAdminLogic.userFacingError(error)
        }
    }
}

private struct RolesPermissionsRoleDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    let role: RoleWithPermissions

    @State private var members: [RBACUserBrief] = []
    @State private var eligible: [RBACUserBrief] = []
    @State private var permissionSearch = ""
    @State private var assignSearch = ""
    @State private var loadingMembers = true
    @State private var loadingEligible = false
    @State private var errorMessage: String?
    @State private var statusMessage: String?
    @State private var busyUserId: String?
    @State private var showAssignSheet = false
    @State private var pendingAssign: RBACUserBrief?
    @State private var pendingRemove: RBACUserBrief?

    private var filteredPermissions: [RBACPermission] {
        RolesPermissionsAdminLogic.filterPermissions(role.permissions, query: permissionSearch)
    }

    private var filteredEligible: [RBACUserBrief] {
        RolesPermissionsAdminLogic.filterUsers(eligible, query: assignSearch)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.roles.readOnlyHint"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if let scope = role.scope?.trimmingCharacters(in: .whitespacesAndNewlines), !scope.isEmpty {
                        Text("\(L.text("mobile.admin.roles.scope")): \(scope)")
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }
                    if let statusMessage {
                        Text(statusMessage)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.brandTeal)
                    }

                    permissionsSection
                    membersSection
                }
                .padding(16)
            }
        }
        .navigationTitle(role.name)
        .navigationBarTitleDisplayMode(.inline)
        .searchable(
            text: $permissionSearch,
            prompt: Text(L.text("mobile.admin.roles.searchPermissions"))
        )
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button(L.text("mobile.admin.roles.assignUser")) {
                    showAssignSheet = true
                }
            }
        }
        .task { await loadMembers() }
        .sheet(isPresented: $showAssignSheet) {
            assignSheet
        }
        .confirmationDialog(
            assignConfirmMessage,
            isPresented: Binding(
                get: { pendingAssign != nil },
                set: { if !$0 { pendingAssign = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.admin.roles.assignUser")) {
                if let user = pendingAssign {
                    Task { await assign(user) }
                }
            }
        }
        .confirmationDialog(
            removeConfirmMessage,
            isPresented: Binding(
                get: { pendingRemove != nil },
                set: { if !$0 { pendingRemove = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.admin.roles.removeUser"), role: .destructive) {
                if let user = pendingRemove {
                    Task { await remove(user) }
                }
            }
        }
    }

    private var permissionsSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.admin.roles.permissionsTitle"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            if role.permissions.isEmpty {
                Text(L.text("mobile.admin.roles.emptyPermissions"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else if filteredPermissions.isEmpty {
                Text(L.text("mobile.admin.roles.emptyPermissionsSearch"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(filteredPermissions) { permission in
                    LMSCard {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(permission.permissionString)
                                .font(.caption.monospaced())
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            if !permission.description.isEmpty {
                                Text(permission.description)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
            }
        }
    }

    private var membersSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.admin.roles.members"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            if loadingMembers {
                LMSSkeletonList(count: 2)
            } else if members.isEmpty {
                Text(L.text("mobile.admin.roles.noMembers"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(members) { member in
                    memberRow(member)
                }
            }
        }
    }

    private func memberRow(_ member: RBACUserBrief) -> some View {
        let busy = busyUserId == member.id
        return LMSCard {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(RolesPermissionsAdminLogic.userDisplayLabel(member))
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(member.email)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer(minLength: 0)
                Button(role: .destructive) {
                    pendingRemove = member
                } label: {
                    Text(L.text("mobile.admin.roles.removeUser"))
                }
                .buttonStyle(.bordered)
                .disabled(busy)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var assignSheet: some View {
        NavigationStack {
            List {
                if loadingEligible {
                    ProgressView()
                } else if filteredEligible.isEmpty {
                    Text(L.text("mobile.admin.roles.noEligibleUsers"))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(filteredEligible) { user in
                        Button {
                            if RolesPermissionsAdminLogic.blocksSelfElevation(
                                role: role,
                                targetUserId: user.id,
                                currentUserId: shell.profile?.id
                            ) {
                                errorMessage = L.text("mobile.admin.roles.selfElevationBlocked")
                                showAssignSheet = false
                            } else {
                                pendingAssign = user
                                showAssignSheet = false
                            }
                        } label: {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(RolesPermissionsAdminLogic.userDisplayLabel(user))
                                Text(user.email)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                        .disabled(busyUserId == user.id)
                    }
                }
            }
            .searchable(
                text: $assignSearch,
                prompt: Text(L.text("mobile.admin.roles.searchUsers"))
            )
            .navigationTitle(L.text("mobile.admin.roles.assignUser"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) {
                        showAssignSheet = false
                    }
                }
            }
            .task(id: assignSearch) {
                await loadEligible()
            }
            .onAppear {
                Task { await loadEligible() }
            }
        }
        .presentationDetents([.medium, .large])
    }

    private var assignConfirmMessage: String {
        guard let user = pendingAssign else { return "" }
        return L.format(
            "mobile.admin.roles.assignConfirm",
            RolesPermissionsAdminLogic.userDisplayLabel(user),
            role.name
        )
    }

    private var removeConfirmMessage: String {
        guard let user = pendingRemove else { return "" }
        return L.format(
            "mobile.admin.roles.removeConfirm",
            RolesPermissionsAdminLogic.userDisplayLabel(user),
            role.name
        )
    }

    private func loadMembers() async {
        guard let token = session.accessToken else { return }
        loadingMembers = true
        errorMessage = nil
        defer { loadingMembers = false }

        do {
            members = try await LMSAPI.fetchRoleUsers(roleId: role.id, accessToken: token)
        } catch {
            errorMessage = RolesPermissionsAdminLogic.userFacingError(error)
        }
    }

    private func loadEligible() async {
        guard let token = session.accessToken else { return }
        loadingEligible = true
        defer { loadingEligible = false }

        do {
            eligible = try await LMSAPI.fetchEligibleRoleUsers(
                roleId: role.id,
                query: assignSearch,
                accessToken: token
            )
        } catch {
            errorMessage = RolesPermissionsAdminLogic.userFacingError(error)
        }
    }

    private func assign(_ user: RBACUserBrief) async {
        guard let token = session.accessToken else { return }
        pendingAssign = nil
        busyUserId = user.id
        errorMessage = nil
        statusMessage = nil
        defer { busyUserId = nil }

        do {
            try await LMSAPI.addUserToRole(roleId: role.id, userId: user.id, accessToken: token)
            members.append(user)
            statusMessage = L.text("mobile.admin.roles.assignSuccess")
        } catch {
            errorMessage = RolesPermissionsAdminLogic.userFacingError(error)
        }
    }

    private func remove(_ user: RBACUserBrief) async {
        guard let token = session.accessToken else { return }
        pendingRemove = nil
        busyUserId = user.id
        errorMessage = nil
        statusMessage = nil
        defer { busyUserId = nil }

        do {
            try await LMSAPI.removeUserFromRole(roleId: role.id, userId: user.id, accessToken: token)
            members.removeAll { $0.id == user.id }
            statusMessage = L.text("mobile.admin.roles.removeSuccess")
        } catch {
            errorMessage = RolesPermissionsAdminLogic.userFacingError(error)
        }
    }
}
