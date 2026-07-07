import SwiftUI

/// Course tools, caption policy, and consortium sharing (M13.2).
struct CourseFeaturesSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    var onCourseUpdated: (CourseSummary) -> Void

    @State private var serverCourse: CourseSummary
    @State private var consortiumShareable = false
    @State private var query = ""
    @State private var loading = true
    @State private var consortiumLoading = false
    @State private var loadError: String?
    @State private var actionError: String?
    @State private var actionSuccess: String?
    @State private var cacheLabel: String?
    @State private var savingTool: CourseFeaturesLogic.Tool?
    @State private var savingCaption = false
    @State private var savingConsortium = false
    @State private var pendingTools: Set<CourseFeaturesLogic.Tool> = []
    @State private var pendingDisableTool: CourseFeaturesLogic.Tool?

    init(course: CourseSummary, onCourseUpdated: @escaping (CourseSummary) -> Void) {
        self.course = course
        self.onCourseUpdated = onCourseUpdated
        _serverCourse = State(initialValue: course)
    }

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }

    private var visibleTools: [CourseFeaturesLogic.ToolRow] {
        CourseFeaturesLogic.filterTools(CourseFeaturesLogic.allToolRows, query: query)
    }

    private var showCaptions: Bool {
        CourseFeaturesLogic.videoCaptionsSectionEnabled(shell.platformFeatures)
    }

    private var showConsortium: Bool {
        CourseFeaturesLogic.consortiumSectionEnabled(shell.platformFeatures)
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                if loading {
                    ProgressView(L.text("mobile.courseSettings.loading"))
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

                    toolsSection

                    if showCaptions {
                        captionPolicySection
                    }

                    if showConsortium {
                        consortiumSection
                    }
                }
            }
            .padding(16)
        }
        .task(id: course.courseCode) {
            await reload(force: false)
            if showConsortium {
                await loadConsortium()
            }
        }
        .confirmationDialog(
            L.text("mobile.courseSettings.features.disableConfirmTitle"),
            isPresented: Binding(
                get: { pendingDisableTool != nil },
                set: { if !$0 { pendingDisableTool = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.courseSettings.features.disableConfirmAction"), role: .destructive) {
                if let tool = pendingDisableTool {
                    Task { await persistToggle(tool: tool, enabled: false) }
                }
                pendingDisableTool = nil
            }
            Button(L.text("mobile.courseSettings.features.cancel"), role: .cancel) {
                pendingDisableTool = nil
            }
        } message: {
            if let tool = pendingDisableTool {
                Text(L.format("mobile.courseSettings.features.disableConfirmMessage", L.text(String.LocalizationValue(tool.labelKey))))
            }
        }
    }

    private var toolsSection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.features.toolsTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.features.toolsDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                TextField(L.text("mobile.courseSettings.features.searchPlaceholder"), text: $query)
                    .textFieldStyle(.roundedBorder)

                if visibleTools.isEmpty {
                    Text(L.format("mobile.courseSettings.features.noToolsMatch", query))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                } else {
                    ForEach(visibleTools) { row in
                        toolRow(row.tool)
                        if row.id != visibleTools.last?.id {
                            Divider()
                        }
                    }
                }
            }
        }
    }

    private func toolRow(_ tool: CourseFeaturesLogic.Tool) -> some View {
        let enabled = CourseFeaturesLogic.isEnabled(tool, course: serverCourse)
        let isSaving = savingTool == tool
        let isPending = pendingTools.contains(tool)

        return HStack(alignment: .top, spacing: 12) {
            VStack(alignment: .leading, spacing: 4) {
                HStack(spacing: 6) {
                    Text(L.text(String.LocalizationValue(tool.labelKey)))
                        .font(.subheadline.weight(.semibold))
                    if isPending {
                        Text(L.text("mobile.courseSettings.features.pending"))
                            .font(.caption2.weight(.semibold))
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(LexturesTheme.brandTeal.opacity(0.15), in: Capsule())
                    }
                }
                Text(L.text(String.LocalizationValue(tool.descriptionKey)))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            Spacer(minLength: 8)
            if isSaving {
                ProgressView()
                    .controlSize(.small)
            } else {
                Toggle(
                    "",
                    isOn: Binding(
                        get: { enabled },
                        set: { newValue in
                            Task { await handleToggle(tool: tool, enabled: newValue) }
                        }
                    )
                )
                .labelsHidden()
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(L.text(String.LocalizationValue(tool.labelKey)))
        .accessibilityValue(enabled ? L.text("mobile.courseSettings.features.enabled") : L.text("mobile.courseSettings.features.disabled"))
    }

    private var captionPolicySection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.features.captionsTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.features.captionsDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                HStack {
                    Text(L.text("mobile.courseSettings.features.captionsMandatory"))
                        .font(.subheadline)
                    Spacer()
                    if savingCaption {
                        ProgressView().controlSize(.small)
                    } else {
                        Toggle(
                            "",
                            isOn: Binding(
                                get: { serverCourse.requireCaptions == true },
                                set: { newValue in Task { await persistCaptionPolicy(enabled: newValue) } }
                            )
                        )
                        .labelsHidden()
                    }
                }
            }
        }
    }

    private var consortiumSection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.features.consortiumTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.features.consortiumDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if consortiumLoading {
                    ProgressView(L.text("mobile.courseSettings.features.consortiumLoading"))
                } else {
                    HStack {
                        Text(L.text("mobile.courseSettings.features.consortiumAllow"))
                            .font(.subheadline)
                        Spacer()
                        if savingConsortium {
                            ProgressView().controlSize(.small)
                        } else {
                            Toggle(
                                "",
                                isOn: Binding(
                                    get: { consortiumShareable },
                                    set: { newValue in Task { await persistConsortium(enabled: newValue) } }
                                )
                            )
                            .labelsHidden()
                        }
                    }
                }
            }
        }
    }

    private func handleToggle(tool: CourseFeaturesLogic.Tool, enabled: Bool) async {
        let currentlyEnabled = CourseFeaturesLogic.isEnabled(tool, course: serverCourse)
        if !enabled && CourseFeaturesLogic.shouldConfirmDisable(tool, currentlyEnabled: currentlyEnabled) {
            pendingDisableTool = tool
            return
        }
        await persistToggle(tool: tool, enabled: enabled)
    }

    private func persistToggle(tool: CourseFeaturesLogic.Tool, enabled: Bool) async {
        guard let token = session.accessToken else { return }
        let previous = serverCourse
        let optimistic = CourseFeaturesLogic.applyToggle(course: serverCourse, tool: tool, enabled: enabled)
        serverCourse = optimistic
        onCourseUpdated(optimistic)
        savingTool = tool
        actionError = nil
        actionSuccess = nil

        do {
            let patch = CourseFeaturesLogic.buildFeaturesPatch(from: optimistic)
            let item = try await offline.enqueueMutation(
                method: "PATCH",
                path: "/api/v1/courses/\(course.courseCode)/features",
                body: patch,
                label: L.text("mobile.courseSettings.features.saveLabel"),
                accessToken: token,
                idempotencyKey: CourseFeaturesLogic.toggleIdempotencyKey(courseCode: course.courseCode, tool: tool)
            )
            if item.status != .synced {
                pendingTools.insert(tool)
            } else {
                pendingTools.remove(tool)
                let refreshed = try await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)
                serverCourse = refreshed
                onCourseUpdated(refreshed)
                actionSuccess = L.text("mobile.courseSettings.features.saved")
            }
        } catch {
            serverCourse = previous
            onCourseUpdated(previous)
            actionError = CourseFeaturesLogic.userFacingError(error)
        }
        savingTool = nil
    }

    private func persistCaptionPolicy(enabled: Bool) async {
        guard let token = session.accessToken else { return }
        let previous = serverCourse
        var optimistic = serverCourse
        optimistic.requireCaptions = enabled
        serverCourse = optimistic
        onCourseUpdated(optimistic)
        savingCaption = true
        actionError = nil

        do {
            _ = try await offline.enqueueMutation(
                method: "PATCH",
                path: "/api/v1/courses/\(course.courseCode)/caption-policy",
                body: CourseCaptionPolicyPatch(requireCaptions: enabled),
                label: L.text("mobile.courseSettings.features.captionSaveLabel"),
                accessToken: token,
                idempotencyKey: CourseFeaturesLogic.captionPolicyIdempotencyKey(courseCode: course.courseCode)
            )
            actionSuccess = L.text("mobile.courseSettings.features.captionSaved")
        } catch {
            serverCourse = previous
            onCourseUpdated(previous)
            actionError = CourseFeaturesLogic.userFacingError(error)
        }
        savingCaption = false
    }

    private func persistConsortium(enabled: Bool) async {
        guard let token = session.accessToken else { return }
        let previous = consortiumShareable
        consortiumShareable = enabled
        savingConsortium = true
        actionError = nil

        do {
            _ = try await offline.enqueueMutation(
                method: "PATCH",
                path: "/api/v1/courses/\(course.courseCode)/consortium-settings",
                body: CourseConsortiumSettingsPatch(consortiumShareable: enabled),
                label: L.text("mobile.courseSettings.features.consortiumSaveLabel"),
                accessToken: token,
                idempotencyKey: CourseFeaturesLogic.consortiumIdempotencyKey(courseCode: course.courseCode)
            )
            actionSuccess = L.text("mobile.courseSettings.features.consortiumSaved")
        } catch {
            consortiumShareable = previous
            actionError = CourseFeaturesLogic.userFacingError(error)
        }
        savingConsortium = false
    }

    private func reload(force: Bool) async {
        guard let token = session.accessToken else {
            loading = false
            return
        }
        loading = true
        loadError = nil
        do {
            let result = try await offline.cachedFetch(
                key: CourseFeaturesLogic.cacheKeyFeatures(courseCode: course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)
            }
            serverCourse = result.value
            onCourseUpdated(result.value)
            if let cached = result.cached, cached.isStale(isOnline: isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            loadError = CourseFeaturesLogic.userFacingError(error)
        }
        loading = false
    }

    private func loadConsortium() async {
        guard let token = session.accessToken else { return }
        consortiumLoading = true
        do {
            let result = try await offline.cachedFetch(
                key: CourseFeaturesLogic.cacheKeyConsortium(courseCode: course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourseConsortiumSettings(courseCode: course.courseCode, accessToken: token)
                    ?? CourseConsortiumSettings(consortiumShareable: false)
            }
            consortiumShareable = result.value.consortiumShareable
        } catch {
            loadError = CourseFeaturesLogic.userFacingError(error)
        }
        consortiumLoading = false
    }
}
