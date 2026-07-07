import SwiftUI

/// District blueprint settings: link children, push updates, sync history (M13.11).
struct CourseBlueprintSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    var onCourseUpdated: (CourseSummary) -> Void

    @State private var serverCourse: CourseSummary
    @State private var permissions: [String] = []
    @State private var permissionsLoaded = false
    @State private var children: [BlueprintChildRow] = []
    @State private var syncLogs: [BlueprintSyncLogRow] = []
    @State private var isBlueprintDraft = false
    @State private var childCodeDraft = ""
    @State private var pushResult: BlueprintPushResult?
    @State private var loading = true
    @State private var loadError: String?
    @State private var cacheLabel: String?
    @State private var actionError: String?
    @State private var actionSuccess: String?
    @State private var busy = false
    @State private var savingDesignation = false
    @State private var showPushConfirm = false
    @State private var pendingUnlinkCode: String?

    init(course: CourseSummary, onCourseUpdated: @escaping (CourseSummary) -> Void) {
        self.course = course
        self.onCourseUpdated = onCourseUpdated
        _serverCourse = State(initialValue: course)
        _isBlueprintDraft = State(initialValue: course.isBlueprint ?? false)
    }

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }

    private var canOrgBlueprint: Bool {
        CourseBlueprintLogic.canManageBlueprint(course: serverCourse, permissions: permissions)
    }

    private var isDesignationDirty: Bool {
        isBlueprintDraft != (serverCourse.isBlueprint ?? false)
    }

    private var pushDisabledReason: String? {
        CourseBlueprintLogic.pushDisabledReason(isOnline: isOnline, childCount: children.count)
    }

    private var mutationsDisabledReason: String? {
        CourseBlueprintLogic.mutationsDisabledReason(isOnline: isOnline)
    }

    var body: some View {
        ZStack(alignment: .bottom) {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if !permissionsLoaded || loading {
                        ProgressView(L.text("mobile.courseSettings.loading"))
                    } else if !canOrgBlueprint {
                        accessDenied
                    } else {
                        if !isOnline {
                            OfflineBanner()
                        }
                        if let cacheLabel {
                            StalenessChip(label: cacheLabel)
                        }
                        if let loadError {
                            LMSErrorBanner(message: loadError)
                        }
                        if let actionError {
                            LMSErrorBanner(message: actionError)
                        }
                        if let actionSuccess {
                            LMSCard(accent: LexturesTheme.brandTeal) {
                                Label(actionSuccess, systemImage: "checkmark.circle.fill")
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(LexturesTheme.primary)
                            }
                        }

                        overviewSection

                        if serverCourse.isBlueprint == true {
                            childrenSection
                            pushSection
                            historySection
                        }
                    }
                }
                .padding(16)
                .padding(.bottom, isDesignationDirty ? 72 : 0)
            }

            if isDesignationDirty {
                UnsavedChangesBanner(
                    isSaving: savingDesignation,
                    onSave: { Task { await saveDesignation() } },
                    onDiscard: {
                        isBlueprintDraft = serverCourse.isBlueprint ?? false
                        actionError = nil
                    }
                )
            }
        }
        .task(id: course.courseCode) {
            await loadPermissions()
            await reload(force: false)
        }
        .confirmationDialog(
            L.text("mobile.courseSettings.blueprint.pushConfirmTitle"),
            isPresented: $showPushConfirm,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.courseSettings.blueprint.pushButton")) {
                Task { await performPush() }
            }
            Button(L.text("mobile.courseSettings.blueprint.cancel"), role: .cancel) {}
        } message: {
            Text(L.text("mobile.courseSettings.blueprint.pushConfirmMessage"))
        }
        .confirmationDialog(
            L.text("mobile.courseSettings.blueprint.unlinkConfirmTitle"),
            isPresented: Binding(
                get: { pendingUnlinkCode != nil },
                set: { if !$0 { pendingUnlinkCode = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.courseSettings.blueprint.unlink"), role: .destructive) {
                if let code = pendingUnlinkCode {
                    Task { await unlinkChild(code) }
                }
                pendingUnlinkCode = nil
            }
            Button(L.text("mobile.courseSettings.blueprint.cancel"), role: .cancel) {
                pendingUnlinkCode = nil
            }
        } message: {
            if let code = pendingUnlinkCode {
                Text(L.format("mobile.courseSettings.blueprint.unlinkConfirmMessage", code))
            }
        }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.courseSettings.section.blueprint"),
            message: L.text("mobile.courseSettings.blueprint.accessDeniedMessage")
        )
    }

    private var overviewSection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.section.blueprint"))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(L.text("mobile.courseSettings.blueprint.description"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                switch CourseBlueprintLogic.blueprintRole(for: serverCourse) {
                case .child(let parentCode):
                    childStatusBanner(parentCode: parentCode)
                case .master:
                    Label(
                        L.text("mobile.courseSettings.blueprint.statusBlueprintMaster"),
                        systemImage: "doc.on.doc.fill"
                    )
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.brandTeal)
                case .none:
                    Text(L.text("mobile.courseSettings.blueprint.notBlueprintInfo"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(L.text("mobile.courseSettings.blueprint.enableDesignation"))
                            .font(.subheadline.weight(.semibold))
                        Text(
                            serverCourse.blueprintParentCourseCode?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
                                ? L.text("mobile.courseSettings.blueprint.enableDesignationDisabledHint")
                                : L.text("mobile.courseSettings.blueprint.enableDesignationHint")
                        )
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer(minLength: 12)
                    Toggle(
                        "",
                        isOn: $isBlueprintDraft
                    )
                    .labelsHidden()
                    .disabled(
                        busy
                            || savingDesignation
                            || serverCourse.blueprintParentCourseCode?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
                    )
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func childStatusBanner(parentCode: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(L.format("mobile.courseSettings.blueprint.childLinkedBanner", parentCode))
                .font(.caption.weight(.semibold))
            Text(L.format(
                "mobile.courseSettings.blueprint.lastSync",
                CourseBlueprintLogic.formatSyncAt(serverCourse.blueprintLastSyncAt)
            ))
            .font(.caption)
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.brandTeal.opacity(0.12), in: RoundedRectangle(cornerRadius: 10))
    }

    private var childrenSection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.blueprint.linkedChildrenTitle"))
                    .font(.subheadline.weight(.semibold))
                Text(L.text("mobile.courseSettings.blueprint.linkedChildrenDescription"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if let mutationsDisabledReason {
                    Text(mutationsDisabledReason)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                HStack(spacing: 8) {
                    TextField(
                        L.text("mobile.courseSettings.blueprint.childCourseCodePlaceholder"),
                        text: $childCodeDraft
                    )
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .textFieldStyle(.roundedBorder)

                    Button(L.text("mobile.courseSettings.blueprint.linkAndSync")) {
                        Task { await linkChild() }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(busy || mutationsDisabledReason != nil || childCodeDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }

                if children.isEmpty {
                    Text(L.text("mobile.courseSettings.blueprint.noChildren"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(children) { child in
                        HStack(alignment: .top) {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(child.courseCode)
                                    .font(.subheadline.weight(.semibold).monospaced())
                                Text(child.title)
                                    .font(.caption)
                                Text(L.format(
                                    "mobile.courseSettings.blueprint.lastSync",
                                    CourseBlueprintLogic.formatSyncAt(child.lastSyncAt)
                                ))
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            Spacer()
                            Button(L.text("mobile.courseSettings.blueprint.unlink")) {
                                pendingUnlinkCode = child.courseCode
                            }
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.red)
                            .disabled(busy || mutationsDisabledReason != nil)
                        }
                        .padding(.vertical, 6)
                        Divider()
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var pushSection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.blueprint.pushTitle"))
                    .font(.subheadline.weight(.semibold))
                Text(L.text("mobile.courseSettings.blueprint.pushDescription"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if let pushDisabledReason {
                    Text(pushDisabledReason)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                Button {
                    showPushConfirm = true
                } label: {
                    Text(busy ? L.text("mobile.courseSettings.blueprint.pushWorking") : L.text("mobile.courseSettings.blueprint.pushButton"))
                }
                .buttonStyle(.borderedProminent)
                .tint(.green)
                .disabled(busy || pushDisabledReason != nil)

                if let pushResult {
                    VStack(alignment: .leading, spacing: 6) {
                        Text(CourseBlueprintLogic.pushResultSummary(
                            success: pushResult.childrenSuccess,
                            total: pushResult.childrenTotal,
                            errors: pushResult.childrenError
                        ))
                        .font(.caption.weight(.semibold))
                        ForEach(Array(pushResult.detail.enumerated()), id: \.offset) { _, row in
                            Text("\(row.courseCode ?? "—"): \(row.ok == true ? "ok" : (row.error ?? "error"))")
                                .font(.caption2.monospaced())
                        }
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var historySection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.blueprint.syncHistoryTitle"))
                    .font(.subheadline.weight(.semibold))

                if syncLogs.isEmpty {
                    Text(L.text("mobile.courseSettings.blueprint.noSyncHistory"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(syncLogs) { log in
                        VStack(alignment: .leading, spacing: 2) {
                            Text(CourseBlueprintLogic.formatSyncAt(log.triggeredAt))
                                .font(.caption.weight(.semibold))
                            Text(CourseBlueprintLogic.syncHistorySummary(
                                success: log.childrenSuccess,
                                total: log.childrenTotal,
                                errors: log.childrenError
                            ))
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        .padding(.vertical, 4)
                        Divider()
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func loadPermissions() async {
        defer { permissionsLoaded = true }
        guard let token = session.accessToken else {
            permissions = shell.permissions
            return
        }
        permissions = (try? await LMSAPI.fetchMyPermissions(accessToken: token)) ?? shell.permissions
    }

    private func reload(force: Bool) async {
        guard let token = session.accessToken else { return }
        loading = true
        loadError = nil
        defer { loading = false }

        do {
            let courseResult = try await offline.cachedFetch(
                key: CourseSettingsLogic.cacheKeySettings(courseCode: course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)
            }
            serverCourse = courseResult.value
            isBlueprintDraft = serverCourse.isBlueprint ?? false
            onCourseUpdated(serverCourse)
            if let cached = courseResult.cached, cached.isStale(isOnline: isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }

            guard CourseBlueprintLogic.shouldLoadBlueprintDetails(course: serverCourse, canManage: canOrgBlueprint) else {
                children = []
                syncLogs = []
                return
            }

            let payloadResult = try await offline.cachedFetch(
                key: CourseBlueprintLogic.cacheKeyBlueprintData(courseCode: course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchBlueprintPayload(courseCode: course.courseCode, accessToken: token)
            }
            children = payloadResult.value.children
            syncLogs = payloadResult.value.syncLogs
            if payloadResult.cached?.isStale(isOnline: isOnline) == true, cacheLabel == nil {
                cacheLabel = payloadResult.cached?.lastUpdatedLabel
            }
        } catch {
            loadError = CourseBlueprintLogic.userFacingError(error)
        }
    }

    private func saveDesignation() async {
        guard let token = session.accessToken else { return }
        savingDesignation = true
        actionError = nil
        defer { savingDesignation = false }

        do {
            let updated = try await LMSAPI.patchCourseBlueprint(
                courseCode: course.courseCode,
                isBlueprint: isBlueprintDraft,
                accessToken: token
            )
            serverCourse = updated
            onCourseUpdated(updated)
            actionSuccess = isBlueprintDraft
                ? L.text("mobile.courseSettings.blueprint.designationEnabled")
                : L.text("mobile.courseSettings.blueprint.designationDisabled")
            await reload(force: true)
        } catch {
            actionError = CourseBlueprintLogic.userFacingError(error)
        }
    }

    private func linkChild() async {
        let code = childCodeDraft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !code.isEmpty, let token = session.accessToken else { return }
        busy = true
        actionError = nil
        actionSuccess = nil
        defer { busy = false }

        do {
            try await LMSAPI.postBlueprintChildLink(
                courseCode: course.courseCode,
                childCourseCode: code,
                accessToken: token
            )
            childCodeDraft = ""
            actionSuccess = L.text("mobile.courseSettings.blueprint.linkSuccess")
            await reload(force: true)
        } catch {
            actionError = CourseBlueprintLogic.userFacingError(error)
        }
    }

    private func unlinkChild(_ childCode: String) async {
        guard let token = session.accessToken else { return }
        busy = true
        actionError = nil
        actionSuccess = nil
        defer { busy = false }

        do {
            try await LMSAPI.deleteBlueprintChildLink(
                courseCode: course.courseCode,
                childCourseCode: childCode,
                accessToken: token
            )
            actionSuccess = L.text("mobile.courseSettings.blueprint.unlinkSuccess")
            await reload(force: true)
        } catch {
            actionError = CourseBlueprintLogic.userFacingError(error)
        }
    }

    private func performPush() async {
        guard let token = session.accessToken else { return }
        busy = true
        actionError = nil
        actionSuccess = nil
        pushResult = nil
        defer { busy = false }

        do {
            let result = try await LMSAPI.postBlueprintPush(
                courseCode: course.courseCode,
                accessToken: token
            )
            pushResult = result
            actionSuccess = L.text("mobile.courseSettings.blueprint.pushSuccess")
            await reload(force: true)
        } catch {
            actionError = CourseBlueprintLogic.userFacingError(error)
        }
    }
}
