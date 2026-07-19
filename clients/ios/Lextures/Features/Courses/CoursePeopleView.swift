import SwiftUI

/// Staff course roster: search, filters, add, message, state, and remove (M11.4 / MOB.4).
struct CoursePeopleSection: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    @State private var enrollments: [CourseEnrollment] = []
    @State private var sections: [CourseSection] = []
    @State private var permissions: [String] = []
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var successMessage: String?
    @State private var loading = true
    @State private var searchText = ""
    @State private var roleFilter: CoursePeopleRoleFilter = .all
    @State private var sectionFilter = ""
    @State private var selectedEnrollment: CourseEnrollment?
    @State private var removeTarget: CourseEnrollment?
    @State private var showAddSheet = false
    @State private var composeMode = false
    @State private var messageSubject = ""
    @State private var messageBody = ""
    @State private var actionBusy = false

    private var canRemove: Bool {
        CoursePeopleLogic.canUpdateEnrollments(courseCode: course.courseCode, permissions: permissions)
    }

    private var canAdd: Bool {
        CoursePeopleLogic.canAddEnrollments(
            courseCode: course.courseCode,
            permissions: permissions,
            features: shell.platformFeatures,
            isOnline: NetworkMonitor.shared.isOnline
        )
    }

    private var filteredEnrollments: [CourseEnrollment] {
        CoursePeopleLogic.filter(
            enrollments: enrollments,
            search: searchText,
            roleFilter: roleFilter,
            sectionId: sectionFilter.isEmpty ? nil : sectionFilter
        )
    }

    private var groupedSections: [CoursePeopleGroup] {
        CoursePeopleLogic.groupedSections(from: filteredEnrollments)
    }

    var body: some View {
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
            if let successMessage {
                Label(successMessage, systemImage: "checkmark.circle.fill")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.brandTeal)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(12)
                    .background(LexturesTheme.brandTeal.opacity(0.12))
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            }

            if loading && enrollments.isEmpty {
                LMSSkeletonList(count: 4)
            } else {
                if canAdd {
                    Button {
                        showAddSheet = true
                        successMessage = nil
                        errorMessage = nil
                    } label: {
                        Label(L.text("mobile.people.add.button"), systemImage: "person.badge.plus")
                            .font(.subheadline.weight(.semibold))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 11)
                    }
                    .background(LexturesTheme.accent(for: colorScheme))
                    .foregroundStyle(colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                    .buttonStyle(.plain)
                    .accessibilityLabel(L.text("mobile.people.add.button"))
                }

                searchField
                roleFilterChips
                if course.isSectionsEnabled && !sections.isEmpty {
                    sectionFilterChips
                }

                if filteredEnrollments.isEmpty {
                    LMSEmptyState(
                        systemImage: "person.3",
                        title: enrollments.isEmpty
                            ? L.text("mobile.people.empty")
                            : L.text("mobile.people.noResults"),
                        message: enrollments.isEmpty
                            ? (canAdd
                                ? L.text("mobile.people.emptyHintAdd")
                                : L.text("mobile.people.emptyHint"))
                            : L.text("mobile.people.noResultsHint")
                    )
                } else {
                    ForEach(groupedSections) { group in
                        groupSection(group)
                    }
                }
            }
        }
        .task { await load() }
        .sheet(item: $selectedEnrollment) { enrollment in
            enrollmentSheet(enrollment)
        }
        .sheet(isPresented: $showAddSheet) {
            CoursePeopleAddSheet(courseCode: course.courseCode) { summary in
                showAddSheet = false
                successMessage = addSuccessMessage(summary)
                Task { await load() }
            }
        }
        .confirmationDialog(
            L.text("mobile.people.remove.confirmTitle"),
            isPresented: Binding(
                get: { removeTarget != nil },
                set: { if !$0 { removeTarget = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.people.remove.confirm"), role: .destructive) {
                if let target = removeTarget {
                    Task { await removeEnrollment(target) }
                }
            }
            Button(L.text("mobile.people.remove.cancel"), role: .cancel) {
                removeTarget = nil
            }
        } message: {
            if let target = removeTarget {
                Text(L.format("mobile.people.remove.confirmMessage", CoursePeopleLogic.displayName(target)))
            }
        }
    }

    private var searchField: some View {
        HStack(spacing: 8) {
            Image(systemName: "magnifyingglass")
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            TextField(L.text("mobile.people.search"), text: $searchText)
                .textInputAutocapitalization(.words)
                .autocorrectionDisabled()
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 12, style: .continuous)
                .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
        )
    }

    private var roleFilterChips: some View {
        LMSSegmentedChips(
            options: CoursePeopleRoleFilter.allCases,
            selection: $roleFilter,
            label: roleFilterLabel
        )
    }

    private var sectionFilterChips: some View {
        let options = [""] + sections.map(\.id)
        return LMSSegmentedChips(
            options: options,
            selection: $sectionFilter,
            label: sectionFilterLabel
        )
    }

    private func roleFilterLabel(_ filter: CoursePeopleRoleFilter) -> String {
        switch filter {
        case .all: return L.text("mobile.people.filter.allRoles")
        case .staff: return L.text("mobile.people.filter.staff")
        case .students: return L.text("mobile.people.filter.students")
        }
    }

    private func sectionFilterLabel(_ sectionId: String) -> String {
        if sectionId.isEmpty { return L.text("mobile.people.filter.allSections") }
        return sections.first(where: { $0.id == sectionId })?.displayName ?? sectionId
    }

    private func groupTitle(_ kind: CoursePeopleGroupKind) -> String {
        switch kind {
        case .teachers: return L.text("mobile.people.role.teachers")
        case .tas: return L.text("mobile.people.role.tas")
        case .students: return L.text("mobile.people.role.students")
        case .other: return L.text("mobile.people.role.other")
        }
    }

    private func groupSection(_ group: CoursePeopleGroup) -> some View {
        LMSCard {
            Text(groupTitle(group.kind))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            ForEach(Array(group.enrollments.enumerated()), id: \.element.id) { index, enrollment in
                if index > 0 { Divider() }
                rosterRow(enrollment)
            }
        }
    }

    private func rosterRow(_ enrollment: CourseEnrollment) -> some View {
        Button {
            selectedEnrollment = enrollment
            composeMode = false
            messageSubject = ""
            messageBody = ""
            successMessage = nil
        } label: {
            HStack(spacing: 12) {
                ProfileAvatarView(
                    avatarUrl: enrollment.avatarUrl,
                    initials: CoursePeopleLogic.initials(enrollment),
                    size: 40
                )

                VStack(alignment: .leading, spacing: 3) {
                    HStack(spacing: 6) {
                        Text(CoursePeopleLogic.displayName(enrollment))
                            .font(.subheadline.weight(.medium))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        if enrollment.invitationPending == true {
                            invitedBadge
                        } else if shell.platformFeatures.ffEnrollmentStateMachine {
                            stateBadge(enrollment.state)
                        }
                    }
                    Text(CoursePeopleLogic.roleLabel(enrollment))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if let section = CoursePeopleLogic.sectionLabel(enrollment) {
                        Text(L.format("mobile.people.section", section))
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }

                Spacer(minLength: 0)

                Image(systemName: "chevron.right")
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
            }
            .padding(.vertical, 4)
        }
        .buttonStyle(.plain)
        .accessibilityLabel(
            "\(CoursePeopleLogic.displayName(enrollment)), \(CoursePeopleLogic.roleLabel(enrollment))"
        )
    }

    private var invitedBadge: some View {
        Text(L.text("mobile.people.invited"))
            .font(.caption2.weight(.semibold))
            .foregroundStyle(LexturesTheme.amber)
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(LexturesTheme.amber.opacity(0.14))
            .clipShape(Capsule())
            .accessibilityLabel(L.text("mobile.people.invited"))
    }

    private func stateBadge(_ state: String?) -> some View {
        let inactive = CoursePeopleLogic.isInactiveState(state)
        return Text(L.dynamicText(CoursePeopleLogic.stateLabelKey(state)))
            .font(.caption2.weight(.semibold))
            .foregroundStyle(inactive ? LexturesTheme.textSecondary(for: colorScheme) : LexturesTheme.brandTeal)
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(
                (inactive ? LexturesTheme.textSecondary(for: colorScheme) : LexturesTheme.brandTeal)
                    .opacity(0.14)
            )
            .clipShape(Capsule())
            .accessibilityLabel(L.dynamicText(CoursePeopleLogic.stateLabelKey(state)))
    }

    private func enrollmentSheet(_ enrollment: CourseEnrollment) -> some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        HStack(spacing: 14) {
                            ProfileAvatarView(
                                avatarUrl: enrollment.avatarUrl,
                                initials: CoursePeopleLogic.initials(enrollment),
                                size: 64
                            )
                            VStack(alignment: .leading, spacing: 4) {
                                Text(CoursePeopleLogic.displayName(enrollment))
                                    .font(LexturesTheme.displayFont(20, weight: .bold))
                                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                Text(CoursePeopleLogic.roleLabel(enrollment))
                                    .font(.subheadline)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                if enrollment.invitationPending == true {
                                    invitedBadge
                                }
                            }
                        }

                        detailRow(L.text("mobile.people.detail.role"), CoursePeopleLogic.roleLabel(enrollment))
                        if let section = CoursePeopleLogic.sectionLabel(enrollment) {
                            detailRow(L.text("mobile.people.detail.section"), section)
                        }
                        if let lastAccess = enrollment.lastCourseAccessAt, !lastAccess.isEmpty {
                            detailRow(
                                L.text("mobile.people.detail.lastAccess"),
                                LMSDates.relative(lastAccess)
                            )
                        }
                        if enrollment.invitationPending == true {
                            detailRow(
                                L.text("mobile.people.detail.state"),
                                L.text("mobile.people.invited")
                            )
                        } else if shell.platformFeatures.ffEnrollmentStateMachine {
                            detailRow(
                                L.text("mobile.people.detail.state"),
                                L.dynamicText(CoursePeopleLogic.stateLabelKey(enrollment.state))
                            )
                        } else if let state = enrollment.state, !state.isEmpty {
                            detailRow(L.text("mobile.people.detail.state"), state.capitalized)
                        }

                        if composeMode {
                            composeFields
                        } else {
                            actionButtons(enrollment)
                        }
                    }
                    .padding(16)
                }
            }
            .navigationTitle(L.text("mobile.people.detail.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button(L.text("mobile.people.detail.done")) {
                        selectedEnrollment = nil
                    }
                }
            }
        }
        .presentationDetents([.medium, .large])
    }

    private func detailRow(_ label: String, _ value: String) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Text(value)
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func actionButtons(_ enrollment: CourseEnrollment) -> some View {
        let canChangeState = CoursePeopleLogic.canChangeEnrollmentState(
            enrollment: enrollment,
            courseCode: course.courseCode,
            permissions: permissions,
            features: shell.platformFeatures,
            isOnline: NetworkMonitor.shared.isOnline
        )
        return VStack(spacing: 10) {
            Button {
                composeMode = true
            } label: {
                Text(L.text("mobile.people.message"))
                    .font(.subheadline.weight(.semibold))
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 11)
            }
            .background(LexturesTheme.accent(for: colorScheme))
            .foregroundStyle(colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .buttonStyle(.plain)
            .disabled(!NetworkMonitor.shared.isOnline)
            .opacity(NetworkMonitor.shared.isOnline ? 1 : 0.55)

            if !NetworkMonitor.shared.isOnline {
                Text(L.text("mobile.people.message.offline"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }

            if canChangeState {
                Button {
                    Task { await toggleEnrollmentState(enrollment) }
                } label: {
                    if actionBusy {
                        ProgressView().controlSize(.small).frame(maxWidth: .infinity)
                    } else {
                        Text(
                            CoursePeopleLogic.isInactiveState(enrollment.state)
                                ? L.text("mobile.people.state.reactivate")
                                : L.text("mobile.people.state.deactivate")
                        )
                        .font(.subheadline.weight(.semibold))
                        .frame(maxWidth: .infinity)
                    }
                }
                .padding(.vertical, 11)
                .overlay(
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
                )
                .buttonStyle(.plain)
                .disabled(actionBusy || !NetworkMonitor.shared.isOnline)
            }

            if canRemove {
                Button(role: .destructive) {
                    removeTarget = enrollment
                } label: {
                    Text(L.text("mobile.people.remove"))
                        .font(.subheadline.weight(.semibold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 11)
                }
                .overlay(
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .stroke(LexturesTheme.error.opacity(0.35), lineWidth: 1)
                )
                .buttonStyle(.plain)
                .disabled(!NetworkMonitor.shared.isOnline)
                .opacity(NetworkMonitor.shared.isOnline ? 1 : 0.55)

                if !NetworkMonitor.shared.isOnline {
                    Text(L.text("mobile.people.remove.offline"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
        .padding(.top, 8)
    }

    private var composeFields: some View {
        VStack(spacing: 10) {
            AuthTextField(
                title: L.text("mobile.people.message.subject"),
                text: $messageSubject,
                placeholder: L.text("mobile.people.message.subject"),
                autocapitalization: .sentences
            )
            DictationField(
                title: L.text("mobile.people.message.body"),
                text: $messageBody,
                placeholder: L.text("mobile.people.message.body")
            )
            Button {
                if let enrollment = selectedEnrollment {
                    Task { await sendMessage(enrollment) }
                }
            } label: {
                if actionBusy {
                    ProgressView().controlSize(.small).frame(maxWidth: .infinity)
                } else {
                    Text(L.text("mobile.people.message.send"))
                        .font(.subheadline.weight(.semibold))
                        .frame(maxWidth: .infinity)
                }
            }
            .padding(.vertical, 11)
            .background(LexturesTheme.accent(for: colorScheme))
            .foregroundStyle(colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .buttonStyle(.plain)
            .disabled(actionBusy || messageBody.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
    }

    private func load() async {
        guard course.viewerIsStaff, let token = session.accessToken else {
            loading = false
            return
        }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            async let permissionsTask = try? LMSAPI.fetchMyPermissions(accessToken: token)
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.courseEnrollments(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourseEnrollments(courseCode: course.courseCode, accessToken: token)
            }
            enrollments = result.value
            permissions = await permissionsTask ?? permissions
            if course.isSectionsEnabled {
                sections = (try? await LMSAPI.fetchCourseSections(
                    courseCode: course.courseCode,
                    accessToken: token
                )) ?? []
            }
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.people.loadError")
        }
    }

    private func sendMessage(_ enrollment: CourseEnrollment) async {
        guard let token = session.accessToken, NetworkMonitor.shared.isOnline else { return }
        actionBusy = true
        errorMessage = nil
        defer { actionBusy = false }
        do {
            _ = try await LMSAPI.sendEnrollmentMessage(
                courseCode: course.courseCode,
                enrollmentId: enrollment.id,
                body: EnrollmentMessageBody(
                    subject: messageSubject.trimmingCharacters(in: .whitespacesAndNewlines),
                    body: messageBody
                ),
                accessToken: token
            )
            selectedEnrollment = nil
            composeMode = false
            successMessage = L.text("mobile.people.message.success")
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.people.message.error")
        }
    }

    private func removeEnrollment(_ enrollment: CourseEnrollment) async {
        guard let token = session.accessToken, NetworkMonitor.shared.isOnline else { return }
        actionBusy = true
        errorMessage = nil
        removeTarget = nil
        defer { actionBusy = false }
        do {
            try await LMSAPI.removeCourseEnrollment(
                courseCode: course.courseCode,
                enrollmentId: enrollment.id,
                accessToken: token
            )
            enrollments.removeAll { $0.id == enrollment.id }
            if selectedEnrollment?.id == enrollment.id {
                selectedEnrollment = nil
            }
            CoursePeopleObservability.recordRemoved(role: enrollment.role)
            successMessage = L.text("mobile.people.remove.success")
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.people.remove.error")
        }
    }

    private func toggleEnrollmentState(_ enrollment: CourseEnrollment) async {
        guard let token = session.accessToken, NetworkMonitor.shared.isOnline else { return }
        let nextState = CoursePeopleLogic.deactivateState(for: enrollment.state)
        actionBusy = true
        errorMessage = nil
        defer { actionBusy = false }
        do {
            let updated = try await LMSAPI.patchEnrollmentState(
                courseCode: course.courseCode,
                enrollmentId: enrollment.id,
                body: PatchEnrollmentStateRequest(state: nextState, reason: nil),
                accessToken: token
            )
            let newState = updated.state ?? nextState
            if let index = enrollments.firstIndex(where: { $0.id == enrollment.id }) {
                enrollments[index].state = newState
            }
            if selectedEnrollment?.id == enrollment.id {
                selectedEnrollment?.state = newState
            }
            CoursePeopleObservability.recordStateChanged(role: enrollment.role, state: newState)
            successMessage = L.text(
                CoursePeopleLogic.isInactiveState(newState)
                    ? "mobile.people.state.deactivateSuccess"
                    : "mobile.people.state.reactivateSuccess"
            )
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.people.state.error")
        }
    }

    private func addSuccessMessage(_ summary: CoursePeopleAddResultSummary) -> String {
        var parts: [String] = []
        if summary.didAdd {
            parts.append(L.format("mobile.people.add.success.added", summary.added.count as Int))
        }
        if !summary.alreadyEnrolled.isEmpty {
            parts.append(L.format("mobile.people.add.success.alreadyEnrolled", summary.alreadyEnrolled.count as Int))
        }
        if !summary.notFound.isEmpty {
            parts.append(L.format("mobile.people.add.success.notFound", summary.notFound.count as Int))
        }
        return parts.isEmpty ? L.text("mobile.people.add.success") : parts.joined(separator: " ")
    }
}

private struct CoursePeopleAddSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme

    let courseCode: String
    let onAdded: (CoursePeopleAddResultSummary) -> Void

    @State private var emailsText = ""
    @State private var selectedRole = "student"
    @State private var busy = false
    @State private var errorMessage: String?

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
                    Text(L.text("mobile.people.add.emailsHint"))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    TextEditor(text: $emailsText)
                        .frame(minHeight: 100)
                        .textInputAutocapitalization(.never)
                        .keyboardType(.emailAddress)
                        .autocorrectionDisabled()
                        .accessibilityLabel(L.text("mobile.people.add.emails"))
                } header: {
                    Text(L.text("mobile.people.add.emails"))
                }
                Section {
                    Picker(L.text("mobile.people.add.role"), selection: $selectedRole) {
                        ForEach(CoursePeopleLogic.assignableRoles) { role in
                            Text(L.dynamicText(role.labelKey)).tag(role.value)
                        }
                    }
                    .accessibilityLabel(L.text("mobile.people.add.role"))
                } header: {
                    Text(L.text("mobile.people.add.role"))
                } footer: {
                    Text(L.text("mobile.people.add.existingAccountsOnly"))
                }
            }
            .navigationTitle(L.text("mobile.people.add.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(busy ? L.text("mobile.people.add.submitting") : L.text("mobile.people.add.submit")) {
                        Task { await submit() }
                    }
                    .disabled(busy || emailsText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
        }
        .presentationDetents([.medium, .large])
    }

    private func submit() async {
        guard let token = session.accessToken, NetworkMonitor.shared.isOnline else {
            errorMessage = L.text("mobile.people.add.error.offline")
            return
        }
        let validation = CoursePeopleLogic.validateEmailsForAdd(emailsText)
        switch validation {
        case .emailsRequired, .invalidEmail:
            if let key = validation.errorKey {
                errorMessage = L.dynamicText(key)
            }
            return
        case .ok(let emails):
            busy = true
            errorMessage = nil
            defer { busy = false }
            do {
                let request = CoursePeopleLogic.buildAddRequest(emails: emails, courseRole: selectedRole)
                let response = try await LMSAPI.addCourseEnrollments(
                    courseCode: courseCode,
                    body: request,
                    accessToken: token
                )
                let summary = CoursePeopleLogic.summarizeAddResponse(response)
                CoursePeopleObservability.recordAdded(
                    role: CoursePeopleLogic.normalizeCourseRole(selectedRole),
                    addedCount: summary.added.count,
                    alreadyCount: summary.alreadyEnrolled.count,
                    notFoundCount: summary.notFound.count
                )
                if !summary.didAdd && summary.hasConflicts {
                    if !summary.alreadyEnrolled.isEmpty {
                        errorMessage = L.text("mobile.people.add.error.alreadyEnrolled")
                    } else {
                        errorMessage = L.text("mobile.people.add.error.notFound")
                    }
                    return
                }
                onAdded(summary)
            } catch {
                errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.people.add.error.generic")
            }
        }
    }
}