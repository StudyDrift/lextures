import SwiftUI

/// User detail for people admin: profile, actions, role assignment (M14.3).
struct UserDetailAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    let userId: String

    @State private var report: PersonReport?
    @State private var roles: [RoleWithPermissions] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var statusMessage: String?
    @State private var busy = false
    @State private var showAssignRoleSheet = false
    @State private var pendingSuspend = false
    @State private var pendingReactivate = false
    @State private var pendingResendInvite = false
    @State private var pendingAssignRole: RoleWithPermissions?

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }
                    if let statusMessage {
                        Text(statusMessage)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.brandTeal)
                    }

                    if loading && report == nil {
                        LMSSkeletonList(count: 2)
                    } else if let report {
                        profileCard(report)
                        if !PeopleAdminLogic.isErased(email: report.email) {
                            actionsSection(report)
                        }
                        roleSection(report)
                        enrollmentsSection(report)
                    }
                }
                .padding(16)
            }
        }
        .navigationTitle(report.map { PeopleAdminLogic.personDisplayName($0) } ?? L.text("mobile.admin.people.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
        .confirmationDialog(
            L.text("mobile.admin.people.suspendConfirm"),
            isPresented: $pendingSuspend,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.admin.people.suspend"), role: .destructive) {
                Task { await patchActive(false) }
            }
        } message: {
            if let report {
                Text("\(PeopleAdminLogic.personDisplayName(report))\n\(report.email)")
            }
        }
        .confirmationDialog(
            L.text("mobile.admin.people.reactivateConfirm"),
            isPresented: $pendingReactivate,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.admin.people.reactivate")) {
                Task { await patchActive(true) }
            }
        }
        .confirmationDialog(
            L.text("mobile.admin.people.resendInviteConfirm"),
            isPresented: $pendingResendInvite,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.admin.people.resendInvite")) {
                Task { await resendInvite() }
            }
        } message: {
            if let report {
                Text(report.email)
            }
        }
        .confirmationDialog(
            L.text("mobile.admin.people.assignRole"),
            isPresented: Binding(
                get: { pendingAssignRole != nil },
                set: { if !$0 { pendingAssignRole = nil } }
            ),
            titleVisibility: .visible
        ) {
            if let role = pendingAssignRole, let report {
                Button(L.text("mobile.admin.people.assignRole")) {
                    Task { await assignRole(role, report: report) }
                }
            }
        } message: {
            if let role = pendingAssignRole, let report {
                Text(
                    L.format(
                        "mobile.admin.roles.assignConfirm",
                        PeopleAdminLogic.personDisplayName(report),
                        role.name
                    )
                )
            }
        }
        .sheet(isPresented: $showAssignRoleSheet) {
            assignRoleSheet
        }
    }

    private func profileCard(_ report: PersonReport) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(PeopleAdminLogic.personDisplayName(report))
                    .font(.title3.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(report.email)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                Divider()

                detailRow(L.text("mobile.admin.people.detail.org"), report.orgName)
                detailRow(L.text("mobile.admin.people.detail.role"), report.role.isEmpty ? L.text("mobile.emDash") : report.role)
                detailRow(
                    L.text("mobile.admin.people.detail.status"),
                    PeopleAdminLogic.statusLabel(active: report.active)
                )
                detailRow(
                    L.text("mobile.admin.people.detail.joined"),
                    LMSDates.shortDateTime(report.createdAt)
                )
                detailRow(
                    L.text("mobile.admin.people.detail.lastActivity"),
                    report.lastActivityAt.map { LMSDates.shortDateTime($0) } ?? L.text("mobile.emDash")
                )
                detailRow(
                    L.text("mobile.admin.people.detail.enrollments"),
                    String(report.enrollmentCount)
                )
            }
        }
    }

    private func detailRow(_ label: String, _ value: String) -> some View {
        HStack(alignment: .top) {
            Text(label)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .frame(width: 110, alignment: .leading)
            Text(value)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Spacer(minLength: 0)
        }
    }

    private func actionsSection(_ report: PersonReport) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.admin.people.actions"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            if report.active {
                Button {
                    if PeopleAdminLogic.blocksSelfSuspend(
                        targetUserId: report.id,
                        currentUserId: shell.profile?.id
                    ) {
                        errorMessage = L.text("mobile.admin.people.selfSuspendBlocked")
                    } else {
                        pendingSuspend = true
                    }
                } label: {
                    Label(L.text("mobile.admin.people.suspend"), systemImage: "person.fill.xmark")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered)
                .tint(LexturesTheme.brandCoral)
                .disabled(busy)
            } else {
                Button {
                    pendingReactivate = true
                } label: {
                    Label(L.text("mobile.admin.people.reactivate"), systemImage: "person.fill.checkmark")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.brandTeal)
                .disabled(busy)
            }

            Button {
                pendingResendInvite = true
            } label: {
                Label(L.text("mobile.admin.people.resendInvite"), systemImage: "envelope.arrow.triangle.branch")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.bordered)
            .disabled(busy)
        }
    }

    private func roleSection(_ report: PersonReport) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.admin.roles.title"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            Button {
                showAssignRoleSheet = true
            } label: {
                Label(L.text("mobile.admin.people.assignRole"), systemImage: "person.badge.plus")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.bordered)
        }
    }

    private var assignRoleSheet: some View {
        NavigationStack {
            List(roles) { role in
                Button {
                    pendingAssignRole = role
                    showAssignRoleSheet = false
                } label: {
                    HStack {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(role.name)
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            if let description = role.description?.trimmingCharacters(in: .whitespacesAndNewlines),
                               !description.isEmpty {
                                Text(description)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                        Spacer()
                        if let report, PeopleAdminLogic.roleMatchesReport(role, report: report) {
                            Image(systemName: "checkmark")
                                .foregroundStyle(LexturesTheme.brandTeal)
                        }
                    }
                }
            }
            .navigationTitle(L.text("mobile.admin.people.assignRoleTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) { showAssignRoleSheet = false }
                }
            }
        }
        .presentationDetents([.medium, .large])
    }

    private func enrollmentsSection(_ report: PersonReport) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.admin.people.detail.enrollmentsTitle"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            if report.enrollments.isEmpty {
                Text(L.text("mobile.admin.people.detail.noEnrollments"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(report.enrollments) { enrollment in
                    LMSCard {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(enrollment.courseTitle)
                                .font(.subheadline.weight(.medium))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Text(enrollment.courseCode)
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            Text(
                                "\(enrollment.role) · \(enrollment.state) · \(LMSDates.shortDateTime(enrollment.enrolledAt))"
                            )
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
            }
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }

        do {
            async let reportTask = LMSAPI.fetchPersonReport(userId: userId, accessToken: token)
            async let rolesTask = LMSAPI.fetchRoles(accessToken: token)
            report = try await reportTask
            roles = try await rolesTask
        } catch {
            errorMessage = PeopleAdminLogic.userFacingError(error)
        }
    }

    private func patchActive(_ active: Bool) async {
        guard let token = session.accessToken else { return }
        busy = true
        errorMessage = nil
        statusMessage = nil
        defer { busy = false }

        do {
            _ = try await LMSAPI.patchPerson(userId: userId, active: active, accessToken: token)
            statusMessage = active
                ? L.text("mobile.admin.people.reactivateSuccess")
                : L.text("mobile.admin.people.suspendSuccess")
            await load()
        } catch {
            errorMessage = PeopleAdminLogic.userFacingError(error)
        }
    }

    private func resendInvite() async {
        guard let report else { return }
        busy = true
        errorMessage = nil
        statusMessage = nil
        defer { busy = false }

        do {
            try await LMSAPI.resendPersonInvite(email: report.email)
            statusMessage = L.text("mobile.admin.people.resendInviteSuccess")
        } catch {
            errorMessage = PeopleAdminLogic.userFacingError(error)
        }
    }

    private func assignRole(_ role: RoleWithPermissions, report: PersonReport) async {
        guard let token = session.accessToken else { return }
        if RolesPermissionsAdminLogic.blocksSelfElevation(
            role: role,
            targetUserId: report.id,
            currentUserId: shell.profile?.id
        ) {
            errorMessage = L.text("mobile.admin.roles.selfElevationBlocked")
            pendingAssignRole = nil
            return
        }
        busy = true
        errorMessage = nil
        statusMessage = nil
        defer {
            busy = false
            pendingAssignRole = nil
        }

        do {
            try await LMSAPI.addUserToRole(
                roleId: role.id,
                userId: report.id,
                accessToken: token
            )
            statusMessage = L.text("mobile.admin.roles.assignSuccess")
            await load()
        } catch {
            errorMessage = PeopleAdminLogic.userFacingError(error)
        }
    }
}
