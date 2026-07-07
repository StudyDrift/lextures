import SwiftUI

/// Course sections list, roster assignment, due-date overrides (M13.3).
struct CourseSectionsSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    var onCourseUpdated: (CourseSummary) -> Void

    @State private var serverCourse: CourseSummary
    @State private var sections: [CourseSection] = []
    @State private var enrollments: [CourseEnrollment] = []
    @State private var assignments: [CourseStructureItem] = []
    @State private var permissions: [String] = []
    @State private var loading = true
    @State private var loadError: String?
    @State private var actionError: String?
    @State private var actionSuccess: String?
    @State private var cacheLabel: String?
    @State private var busy = false

    @State private var newSectionCode = ""
    @State private var newSectionName = ""

    @State private var overrideSectionId = ""
    @State private var overrideItemId = ""
    @State private var overrideDue = ""

    @State private var selectedSection: CourseSection?
    @State private var editSectionCode = ""
    @State private var editSectionName = ""
    @State private var pendingArchiveSection: CourseSection?
    @State private var pendingMoveEnrollment: CourseEnrollment?
    @State private var moveTargetSectionId = ""

    init(course: CourseSummary, onCourseUpdated: @escaping (CourseSummary) -> Void) {
        self.course = course
        self.onCourseUpdated = onCourseUpdated
        _serverCourse = State(initialValue: course)
    }

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var showEditors: Bool { CourseSectionsLogic.shouldShowEditors(sectionsEnabled: serverCourse.isSectionsEnabled) }
    private var canAssignStudents: Bool {
        CourseSectionsLogic.canAssignStudents(courseCode: course.courseCode, permissions: permissions)
    }
    private var activeSections: [CourseSection] { CourseSectionsLogic.activeSections(sections) }
    private var mutationsDisabledReason: String? { CourseSectionsLogic.mutationsDisabledReason(isOnline: isOnline) }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                if loading {
                    ProgressView(L.text("mobile.courseSettings.loading"))
                } else if !showEditors {
                    disabledGate
                } else {
                    if !isOnline { OfflineBanner() }
                    if let cacheLabel { StalenessChip(label: cacheLabel) }
                    if let loadError { LMSErrorBanner(message: loadError) }
                    if let actionError { LMSErrorBanner(message: actionError) }
                    if let actionSuccess {
                        LMSCard(accent: LexturesTheme.brandTeal) {
                            Label(actionSuccess, systemImage: "checkmark.circle.fill")
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(LexturesTheme.primary)
                        }
                    }
                    if let reason = mutationsDisabledReason {
                        Text(reason)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }

                    sectionsListCard
                    createSectionCard
                    overrideCard

                    CourseCrossListingView(
                        course: serverCourse,
                        sections: sections,
                        permissions: permissions,
                        onReload: { await reload(force: true) }
                    )
                }
            }
            .padding(16)
        }
        .task(id: course.courseCode) {
            permissions = (try? await LMSAPI.fetchMyPermissions(accessToken: session.accessToken ?? "")) ?? []
            await reload(force: false)
        }
        .sheet(item: $selectedSection) { section in
            sectionDetailSheet(section)
        }
        .confirmationDialog(
            L.text("mobile.courseSettings.sections.archiveConfirmTitle"),
            isPresented: Binding(
                get: { pendingArchiveSection != nil },
                set: { if !$0 { pendingArchiveSection = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.courseSettings.sections.archive"), role: .destructive) {
                if let section = pendingArchiveSection {
                    Task { await archiveSection(section) }
                }
                pendingArchiveSection = nil
            }
            Button(L.text("mobile.courseSettings.sections.cancel"), role: .cancel) {
                pendingArchiveSection = nil
            }
        } message: {
            if let section = pendingArchiveSection {
                Text(L.format("mobile.courseSettings.sections.archiveConfirmMessage", section.displayLabel))
            }
        }
        .confirmationDialog(
            L.text("mobile.courseSettings.sections.moveStudentTitle"),
            isPresented: Binding(
                get: { pendingMoveEnrollment != nil },
                set: { if !$0 { pendingMoveEnrollment = nil; moveTargetSectionId = "" } }
            ),
            titleVisibility: .visible
        ) {
            ForEach(activeSections) { section in
                Button(section.displayLabel) {
                    moveTargetSectionId = section.id
                    if let enrollment = pendingMoveEnrollment {
                        Task { await moveEnrollment(enrollment, to: section.id) }
                    }
                    pendingMoveEnrollment = nil
                }
            }
            Button(L.text("mobile.courseSettings.sections.cancel"), role: .cancel) {
                pendingMoveEnrollment = nil
            }
        }
    }

    private var disabledGate: some View {
        LMSEmptyState(
            systemImage: "square.grid.2x2",
            title: L.text("mobile.courseSettings.sections.disabledTitle"),
            message: L.text("mobile.courseSettings.sections.disabledMessage")
        )
    }

    private var sectionsListCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.sections.listTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.sections.listDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if sections.isEmpty {
                    Text(L.text("mobile.courseSettings.sections.empty"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(sections) { section in
                        Button {
                            selectedSection = section
                            editSectionCode = section.sectionCode
                            editSectionName = section.name ?? ""
                        } label: {
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(section.displayLabel)
                                        .font(.subheadline.weight(.medium))
                                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                    Text(L.format(
                                        "mobile.courseSettings.sections.rosterCount",
                                        "\(CourseSectionsLogic.rosterCount(sectionId: section.id, enrollments: enrollments))"
                                    ))
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    if !section.isActive {
                                        Text(section.status ?? "archived")
                                            .font(.caption2.weight(.semibold))
                                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    }
                                }
                                Spacer()
                                Image(systemName: "chevron.right")
                                    .font(.caption2)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            .padding(.vertical, 4)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
    }

    private var createSectionCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(L.text("mobile.courseSettings.sections.createTitle"))
                    .font(.subheadline.weight(.semibold))

                AuthTextField(
                    title: L.text("mobile.courseSettings.sections.sectionCode"),
                    text: $newSectionCode,
                    placeholder: "001"
                )
                AuthTextField(
                    title: L.text("mobile.courseSettings.sections.sectionNameOptional"),
                    text: $newSectionName,
                    placeholder: L.text("mobile.courseSettings.sections.sectionNamePlaceholder")
                )

                Button {
                    Task { await createSection() }
                } label: {
                    if busy {
                        ProgressView().controlSize(.small).frame(maxWidth: .infinity)
                    } else {
                        Text(L.text("mobile.courseSettings.sections.createButton"))
                            .font(.subheadline.weight(.semibold))
                            .frame(maxWidth: .infinity)
                    }
                }
                .padding(.vertical, 10)
                .background(LexturesTheme.accent(for: colorScheme))
                .foregroundStyle(colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
                .clipShape(RoundedRectangle(cornerRadius: 10))
                .buttonStyle(.plain)
                .disabled(busy || newSectionCode.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
    }

    private var overrideCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(L.text("mobile.courseSettings.sections.overrideTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.sections.overrideDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                Picker(L.text("mobile.courseSettings.sections.overrideSection"), selection: $overrideSectionId) {
                    Text(L.text("mobile.courseSettings.sections.selectPlaceholder")).tag("")
                    ForEach(activeSections) { section in
                        Text(section.displayLabel).tag(section.id)
                    }
                }
                .pickerStyle(.menu)

                Picker(L.text("mobile.courseSettings.sections.overrideAssignment"), selection: $overrideItemId) {
                    Text(L.text("mobile.courseSettings.sections.selectPlaceholder")).tag("")
                    ForEach(assignments) { item in
                        Text(item.title).tag(item.id)
                    }
                }
                .pickerStyle(.menu)

                AuthTextField(
                    title: L.text("mobile.courseSettings.sections.overrideDueLocal"),
                    text: $overrideDue,
                    placeholder: "yyyy-MM-dd'T'HH:mm"
                )

                Button {
                    Task { await saveOverride() }
                } label: {
                    Text(L.text("mobile.courseSettings.sections.overrideSave"))
                        .font(.subheadline.weight(.semibold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                }
                .background(LexturesTheme.accent(for: colorScheme))
                .foregroundStyle(colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
                .clipShape(RoundedRectangle(cornerRadius: 10))
                .buttonStyle(.plain)
                .disabled(busy || overrideSectionId.isEmpty || overrideItemId.isEmpty)
            }
        }
    }

    private func sectionDetailSheet(_ section: CourseSection) -> some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    AuthTextField(
                        title: L.text("mobile.courseSettings.sections.sectionCode"),
                        text: $editSectionCode,
                        placeholder: "001"
                    )
                    AuthTextField(
                        title: L.text("mobile.courseSettings.sections.sectionNameOptional"),
                        text: $editSectionName,
                        placeholder: L.text("mobile.courseSettings.sections.sectionNamePlaceholder")
                    )

                    if section.isActive {
                        Button {
                            Task { await renameSection(section) }
                        } label: {
                            Text(L.text("mobile.courseSettings.sections.saveRename"))
                                .font(.subheadline.weight(.semibold))
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 10)
                        }
                        .background(LexturesTheme.accent(for: colorScheme))
                        .foregroundStyle(colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                        .buttonStyle(.plain)
                        .disabled(busy || !isOnline)

                        Button(role: .destructive) {
                            pendingArchiveSection = section
                        } label: {
                            Text(L.text("mobile.courseSettings.sections.archive"))
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 10)
                        }
                        .buttonStyle(.plain)
                        .disabled(busy || !isOnline)
                    }

                    LMSCard {
                        Text(L.text("mobile.courseSettings.sections.sectionRosterTitle"))
                            .font(.headline)
                        let roster = enrollments.filter {
                            $0.sectionId == section.id && CoursePeopleLogic.isStudentRole($0.role)
                        }
                        if roster.isEmpty {
                            Text(L.text("mobile.courseSettings.sections.sectionRosterEmpty"))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        } else {
                            ForEach(roster) { enrollment in
                                HStack {
                                    VStack(alignment: .leading) {
                                        Text(CoursePeopleLogic.displayName(enrollment))
                                            .font(.subheadline)
                                        Text(CoursePeopleLogic.roleLabel(enrollment))
                                            .font(.caption)
                                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    }
                                    Spacer()
                                    if canAssignStudents && section.isActive {
                                        Button(L.text("mobile.courseSettings.sections.moveStudent")) {
                                            pendingMoveEnrollment = enrollment
                                        }
                                        .font(.caption.weight(.semibold))
                                        .disabled(!isOnline)
                                    }
                                }
                                .padding(.vertical, 4)
                            }
                        }
                    }
                }
                .padding(16)
            }
            .navigationTitle(section.displayLabel)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button(L.text("mobile.courseSettings.sections.done")) {
                        selectedSection = nil
                    }
                }
            }
        }
        .presentationDetents([.medium, .large])
    }

    private func reload(force: Bool) async {
        guard showEditors, let token = session.accessToken else {
            loading = false
            return
        }
        loading = sections.isEmpty
        loadError = nil
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: CourseSectionsLogic.cacheKeySections(courseCode: course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourseSectionsPayload(
                    courseCode: course.courseCode,
                    accessToken: token
                )
            }
            sections = result.value.sections
            enrollments = result.value.enrollments
            assignments = result.value.assignments
            if let cached = result.cached, cached.isStale(isOnline: isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
            let refreshed = try? await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)
            if let refreshed {
                serverCourse = refreshed
                onCourseUpdated(refreshed)
            }
        } catch {
            loadError = CourseSectionsLogic.userFacingError(error)
        }
    }

    private func createSection() async {
        guard let token = session.accessToken else { return }
        let code = newSectionCode.trimmingCharacters(in: .whitespacesAndNewlines)
        if let validation = CourseSectionsLogic.validateCreateSection(sectionCode: code) {
            actionError = validation
            return
        }
        busy = true
        actionError = nil
        actionSuccess = nil
        defer { busy = false }
        do {
            let body = CreateCourseSectionBody(
                sectionCode: code,
                name: newSectionName.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
            )
            _ = try await offline.enqueueMutation(
                method: "POST",
                path: "/api/v1/courses/\(course.courseCode)/sections",
                body: body,
                label: L.text("mobile.courseSettings.sections.createLabel"),
                accessToken: token,
                idempotencyKey: CourseSectionsLogic.createSectionIdempotencyKey(
                    courseCode: course.courseCode,
                    sectionCode: code
                )
            )
            newSectionCode = ""
            newSectionName = ""
            actionSuccess = L.text("mobile.courseSettings.sections.createSuccess")
            await reload(force: true)
        } catch {
            actionError = CourseSectionsLogic.userFacingError(error)
        }
    }

    private func renameSection(_ section: CourseSection) async {
        guard let token = session.accessToken, isOnline else { return }
        busy = true
        actionError = nil
        defer { busy = false }
        do {
            _ = try await offline.enqueueMutation(
                method: "PATCH",
                path: "/api/v1/courses/\(course.courseCode)/sections/\(section.id)",
                body: PatchCourseSectionBody(
                    sectionCode: editSectionCode.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
                    name: editSectionName.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
                    status: nil
                ),
                label: L.text("mobile.courseSettings.sections.renameLabel"),
                accessToken: token,
                idempotencyKey: CourseSectionsLogic.patchSectionIdempotencyKey(
                    courseCode: course.courseCode,
                    sectionId: section.id
                )
            )
            actionSuccess = L.text("mobile.courseSettings.sections.renameSuccess")
            await reload(force: true)
        } catch {
            actionError = CourseSectionsLogic.userFacingError(error)
        }
    }

    private func archiveSection(_ section: CourseSection) async {
        guard let token = session.accessToken, isOnline else { return }
        busy = true
        actionError = nil
        defer { busy = false }
        do {
            _ = try await offline.enqueueMutation(
                method: "DELETE",
                path: "/api/v1/courses/\(course.courseCode)/sections/\(section.id)",
                body: nil,
                label: L.text("mobile.courseSettings.sections.archiveLabel"),
                accessToken: token,
                idempotencyKey: CourseSectionsLogic.archiveSectionIdempotencyKey(
                    courseCode: course.courseCode,
                    sectionId: section.id
                )
            )
            if selectedSection?.id == section.id { selectedSection = nil }
            actionSuccess = L.text("mobile.courseSettings.sections.archiveSuccess")
            await reload(force: true)
        } catch {
            actionError = CourseSectionsLogic.userFacingError(error)
        }
    }

    private func saveOverride() async {
        guard let token = session.accessToken else { return }
        guard let body = CourseSectionsLogic.buildOverrideBody(dueAtLocal: overrideDue) else {
            actionError = L.text("mobile.courseSettings.sections.overrideInvalidDate")
            return
        }
        busy = true
        actionError = nil
        defer { busy = false }
        do {
            _ = try await offline.enqueueMutation(
                method: "PUT",
                path: "/api/v1/sections/\(overrideSectionId)/overrides/\(overrideItemId)",
                body: body,
                label: L.text("mobile.courseSettings.sections.overrideLabel"),
                accessToken: token,
                idempotencyKey: CourseSectionsLogic.overrideIdempotencyKey(
                    sectionId: overrideSectionId,
                    itemId: overrideItemId
                )
            )
            overrideDue = ""
            actionSuccess = L.text("mobile.courseSettings.sections.overrideSuccess")
        } catch {
            actionError = CourseSectionsLogic.userFacingError(error)
        }
    }

    private func moveEnrollment(_ enrollment: CourseEnrollment, to sectionId: String) async {
        guard let token = session.accessToken, isOnline else { return }
        busy = true
        actionError = nil
        defer { busy = false }
        do {
            _ = try await offline.enqueueMutation(
                method: "PATCH",
                path: "/api/v1/enrollments/\(enrollment.id)/section",
                body: EnrollmentSectionPatchBody(sectionId: sectionId),
                label: L.text("mobile.courseSettings.sections.moveStudentLabel"),
                accessToken: token,
                idempotencyKey: CourseSectionsLogic.enrollmentSectionIdempotencyKey(
                    enrollmentId: enrollment.id,
                    sectionId: sectionId
                )
            )
            actionSuccess = L.text("mobile.courseSettings.sections.moveStudentSuccess")
            await reload(force: true)
        } catch {
            actionError = CourseSectionsLogic.userFacingError(error)
        }
    }
}

private extension String {
    var nilIfEmpty: String? {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}
