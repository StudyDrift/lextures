import SwiftUI

/// Grading agents list, create, and edit (M13.6).
struct CourseGradingAgentsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var agents: [CourseGradingAgentSummary] = []
    @State private var templates: [GraderAgentTemplateSummary] = []
    @State private var structure: [CourseStructureItem] = []
    @State private var filterQuery = ""
    @State private var loading = true
    @State private var loadError: String?
    @State private var actionError: String?
    @State private var actionSuccess: String?
    @State private var cacheLabel: String?
    @State private var selectedAgent: CourseGradingAgentSummary?
    @State private var createSheetOpen = false

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var existingItemIds: Set<String> { Set(agents.map(\.itemId)) }
    private var gradableOptions: [CourseGradingAgentsLogic.GradableOption] {
        CourseGradingAgentsLogic.gradableOptions(from: structure, excluding: existingItemIds)
    }
    private var filteredAgents: [CourseGradingAgentSummary] {
        CourseGradingAgentsLogic.filteredAgents(agents, query: filterQuery)
    }

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if loading {
                        ProgressView(L.text("mobile.courseSettings.loading"))
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

                        introCard
                        templatesCard
                        agentsCard
                    }
                }
                .padding(16)
            }
        }
        .safeAreaInset(edge: .bottom) {
            if !loading {
                Button {
                    createSheetOpen = true
                } label: {
                    Label(L.text("mobile.courseSettings.gradingAgents.createButton"), systemImage: "plus")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .padding(.horizontal, 16)
                .padding(.bottom, 8)
                .disabled(gradableOptions.isEmpty)
            }
        }
        .task(id: course.courseCode) { await reload(force: false) }
        .sheet(item: $selectedAgent) { agent in
            NavigationStack {
                GradingAgentEditorContent(
                    course: course,
                    itemId: agent.itemId,
                    itemKind: CourseGradingAgentsLogic.normalizedItemKind(agent.itemKind),
                    assignmentTitle: agent.assignmentTitle,
                    initialDraft: nil,
                    loadedConfig: true,
                    onDismiss: { selectedAgent = nil },
                    onSaved: {
                        selectedAgent = nil
                        Task { await reload(force: true) }
                    },
                    onDeleted: {
                        selectedAgent = nil
                        Task { await reload(force: true) }
                    }
                )
            }
        }
        .sheet(isPresented: $createSheetOpen) {
            CreateGradingAgentSheet(
                course: course,
                gradableOptions: gradableOptions,
                templates: templates,
                onDismiss: { createSheetOpen = false },
                onCreated: {
                    createSheetOpen = false
                    Task { await reload(force: true) }
                }
            )
        }
    }

    private var introCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.courseSettings.gradingAgents.introTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.gradingAgents.introDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    @ViewBuilder
    private var templatesCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(L.text("mobile.courseSettings.gradingAgents.templatesTitle"))
                    .font(.headline)
                if templates.isEmpty {
                    Text(L.text("mobile.courseSettings.gradingAgents.templatesEmpty"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(templates) { template in
                        HStack {
                            Text(template.name)
                                .font(.subheadline.weight(.medium))
                            Spacer()
                            if template.isBuiltin == true {
                                Text(L.text("mobile.courseSettings.gradingAgents.templateBuiltin"))
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                    }
                    Text(L.text("mobile.courseSettings.gradingAgents.templatesHint"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    private var agentsCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.gradingAgents.agentsTitle"))
                    .font(.headline)

                if agents.isEmpty {
                    Text(L.text("mobile.courseSettings.gradingAgents.empty"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    TextField(
                        L.text("mobile.courseSettings.gradingAgents.filterPlaceholder"),
                        text: $filterQuery
                    )
                    .textFieldStyle(.roundedBorder)

                    if filteredAgents.isEmpty {
                        Text(L.text("mobile.courseSettings.gradingAgents.noMatch"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    } else {
                        ForEach(filteredAgents) { agent in
                            Button {
                                selectedAgent = agent
                            } label: {
                                agentRow(agent)
                            }
                            .buttonStyle(.plain)
                            if agent.id != filteredAgents.last?.id {
                                Divider()
                            }
                        }
                    }
                }
            }
        }
    }

    private func agentRow(_ agent: CourseGradingAgentSummary) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline) {
                Text(agent.assignmentTitle)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.primary)
                if CourseGradingAgentsLogic.normalizedItemKind(agent.itemKind) == "quiz" {
                    Text(L.text("mobile.courseSettings.gradingAgents.quizBadge"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.brandTeal)
                }
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            HStack(spacing: 8) {
                statusBadge(agent.status)
                Text(
                    agent.autoGradeNew
                        ? L.text("mobile.courseSettings.gradingAgents.autoGradeOn")
                        : L.text("mobile.courseSettings.gradingAgents.autoGradeOff")
                )
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            if agent.assignmentArchived {
                Text(L.text("mobile.courseSettings.gradingAgents.archivedAssignment"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.vertical, 4)
    }

    private func statusBadge(_ status: String) -> some View {
        Text(L.text(String.LocalizationValue(CourseGradingAgentsLogic.statusLabelKey(status))))
            .font(.caption.weight(.semibold))
            .padding(.horizontal, 8)
            .padding(.vertical, 2)
            .background(statusColor(status).opacity(0.15), in: Capsule())
            .foregroundStyle(statusColor(status))
    }

    private func statusColor(_ status: String) -> Color {
        switch status {
        case "accepted": return .green
        case "archived": return .secondary
        default: return .orange
        }
    }

    private func reload(force: Bool) async {
        guard let token = session.accessToken else { return }
        if !force && !agents.isEmpty { return }
        loading = agents.isEmpty
        loadError = nil
        actionError = nil
        defer { loading = false }

        do {
            async let agentsResult = offline.cachedFetch(
                key: CourseGradingAgentsLogic.cacheKeyAgents(courseCode: course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourseGradingAgents(courseCode: course.courseCode, accessToken: token)
            }
            async let templatesResponse = LMSAPI.fetchGraderAgentTemplates(
                courseCode: course.courseCode,
                accessToken: token
            )
            async let structureResponse = LMSAPI.fetchCourseStructure(
                courseCode: course.courseCode,
                accessToken: token
            )

            let cachedAgents = try await agentsResult
            agents = cachedAgents.value.agents
            templates = try await templatesResponse.templates
            structure = try await structureResponse
            if let cached = cachedAgents.cached, cached.isStale(isOnline: isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            loadError = error.localizedDescription
        }
    }
}

private struct CreateGradingAgentSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let gradableOptions: [CourseGradingAgentsLogic.GradableOption]
    let templates: [GraderAgentTemplateSummary]
    var onDismiss: () -> Void
    var onCreated: () -> Void

    @State private var itemKind = "assignment"
    @State private var selectedItemId = ""
    @State private var selectedTemplateId = ""
    @State private var useTemplate = false
    @State private var opening = false
    @State private var errorMessage: String?
    @State private var draftTarget: GradingAgentDraftTarget?

    private var filteredOptions: [CourseGradingAgentsLogic.GradableOption] {
        gradableOptions.filter { $0.kind == itemKind }
    }

    var body: some View {
        NavigationStack {
            Form {
                if let errorMessage {
                    Section {
                        Text(errorMessage)
                            .foregroundStyle(.red)
                    }
                }
                Section(L.text("mobile.courseSettings.gradingAgents.createScopeTitle")) {
                    Picker(L.text("mobile.courseSettings.gradingAgents.itemKindLabel"), selection: $itemKind) {
                        Text(L.text("mobile.courseSettings.gradingAgents.itemKindAssignment")).tag("assignment")
                        Text(L.text("mobile.courseSettings.gradingAgents.itemKindQuiz")).tag("quiz")
                    }
                    .pickerStyle(.segmented)
                    .onChange(of: itemKind) { _, _ in
                        selectedItemId = filteredOptions.first?.id ?? ""
                    }

                    if filteredOptions.isEmpty {
                        Text(L.text("mobile.courseSettings.gradingAgents.noAvailableItems"))
                    } else {
                        Picker(L.text("mobile.courseSettings.gradingAgents.activityLabel"), selection: $selectedItemId) {
                            ForEach(filteredOptions) { option in
                                Text(option.label).tag(option.id)
                            }
                        }
                    }
                }

                if !templates.isEmpty {
                    Section(L.text("mobile.courseSettings.gradingAgents.templateOptionalTitle")) {
                        Toggle(L.text("mobile.courseSettings.gradingAgents.useTemplate"), isOn: $useTemplate)
                        if useTemplate {
                            Picker(L.text("mobile.courseSettings.gradingAgents.templateLabel"), selection: $selectedTemplateId) {
                                ForEach(templates) { template in
                                    Text(template.name).tag(template.id)
                                }
                            }
                        }
                    }
                }
            }
            .navigationTitle(L.text("mobile.courseSettings.gradingAgents.createTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) {
                        onDismiss()
                        dismiss()
                    }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.courseSettings.gradingAgents.continueButton")) {
                        Task { await continueToEditor() }
                    }
                    .disabled(opening || selectedItemId.isEmpty)
                }
            }
            .onAppear {
                selectedItemId = filteredOptions.first?.id ?? ""
                selectedTemplateId = templates.first?.id ?? ""
            }
            .sheet(item: $draftTarget) { target in
                GradingAgentDraftEditorSheet(
                    course: course,
                    target: target,
                    onDismiss: { draftTarget = nil },
                    onSaved: {
                        draftTarget = nil
                        onCreated()
                        dismiss()
                    }
                )
            }
        }
    }

    private func continueToEditor() async {
        guard let token = session.accessToken else { return }
        guard let option = filteredOptions.first(where: { $0.id == selectedItemId }) else { return }
        opening = true
        errorMessage = nil
        defer { opening = false }

        do {
            var seedDraft = CourseGradingAgentsLogic.draft(from: nil as GraderAgentConfig?)
            if useTemplate, !selectedTemplateId.isEmpty {
                let template = try await LMSAPI.fetchGraderAgentTemplate(
                    courseCode: course.courseCode,
                    templateId: selectedTemplateId,
                    accessToken: token
                )
                seedDraft = CourseGradingAgentsLogic.draft(from: template)
            }
            draftTarget = GradingAgentDraftTarget(
                itemId: option.id,
                itemKind: option.kind,
                assignmentTitle: option.label,
                seedDraft: seedDraft
            )
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

private struct GradingAgentDraftTarget: Identifiable {
    var id: String { itemId }
    var itemId: String
    var itemKind: String
    var assignmentTitle: String
    var seedDraft: CourseGradingAgentsLogic.AgentDraft
}

private struct GradingAgentDraftEditorSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.dismiss) private var dismiss

    let course: CourseSummary
    let target: GradingAgentDraftTarget
    var onDismiss: () -> Void
    var onSaved: () -> Void

    var body: some View {
        GradingAgentEditorContent(
            course: course,
            itemId: target.itemId,
            itemKind: target.itemKind,
            assignmentTitle: target.assignmentTitle,
            initialDraft: target.seedDraft,
            onDismiss: {
                onDismiss()
                dismiss()
            },
            onSaved: {
                onSaved()
                dismiss()
            },
            onDeleted: nil
        )
    }
}

private struct GradingAgentEditorContent: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let itemId: String
    let itemKind: String
    let assignmentTitle: String
    var initialDraft: CourseGradingAgentsLogic.AgentDraft?
    var loadedConfig = false
    var onDismiss: () -> Void
    var onSaved: () -> Void
    var onDeleted: (() -> Void)?

    @State private var baseline = CourseGradingAgentsLogic.draft(from: nil as GraderAgentConfig?)
    @State private var form = CourseGradingAgentsLogic.draft(from: nil as GraderAgentConfig?)
    @State private var loadingConfig = false
    @State private var loadError: String?
    @State private var actionError: String?
    @State private var actionSuccess: String?
    @State private var saving = false
    @State private var deleting = false
    @State private var showDeleteConfirm = false

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var isDirty: Bool { CourseGradingAgentsLogic.isDirty(current: form, baseline: baseline) }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        if loadingConfig {
                            ProgressView(L.text("mobile.courseSettings.loading"))
                        } else {
                            if !isOnline { OfflineBanner() }
                            if let loadError { LMSErrorBanner(message: loadError) }
                            if let actionError { LMSErrorBanner(message: actionError) }
                            if let actionSuccess {
                                LMSCard(accent: LexturesTheme.brandTeal) {
                                    Label(actionSuccess, systemImage: "checkmark.circle.fill")
                                        .font(.subheadline.weight(.semibold))
                                }
                            }

                            LMSCard {
                                VStack(alignment: .leading, spacing: 8) {
                                    Text(L.text("mobile.courseSettings.gradingAgents.scopeLabel"))
                                        .font(.headline)
                                    Text(assignmentTitle)
                                        .font(.subheadline)
                                    if itemKind == "quiz" {
                                        Text(L.text("mobile.courseSettings.gradingAgents.quizBadge"))
                                            .font(.caption)
                                            .foregroundStyle(LexturesTheme.brandTeal)
                                    }
                                }
                            }

                            LMSCard {
                                VStack(alignment: .leading, spacing: 12) {
                                    Text(L.text("mobile.courseSettings.gradingAgents.promptLabel"))
                                        .font(.headline)
                                    TextEditor(text: $form.prompt)
                                        .frame(minHeight: 160)
                                        .overlay(
                                            RoundedRectangle(cornerRadius: 8)
                                                .stroke(Color.secondary.opacity(0.25))
                                        )
                                        .accessibilityLabel(L.text("mobile.courseSettings.gradingAgents.promptLabel"))

                                    Toggle(
                                        L.text("mobile.courseSettings.gradingAgents.includeContent"),
                                        isOn: $form.includeAssignmentContent
                                    )
                                    Toggle(
                                        L.text("mobile.courseSettings.gradingAgents.includeRubric"),
                                        isOn: $form.includeRubric
                                    )
                                    Toggle(
                                        L.text("mobile.courseSettings.gradingAgents.autoGradeNew"),
                                        isOn: $form.autoGradeNew
                                    )

                                    Text(L.text("mobile.courseSettings.gradingAgents.statusLabel"))
                                        .font(.subheadline.weight(.semibold))
                                    Picker(L.text("mobile.courseSettings.gradingAgents.statusLabel"), selection: $form.status) {
                                        ForEach(CourseGradingAgentsLogic.AgentStatus.allCases) { status in
                                            Text(L.text(String.LocalizationValue(status.labelKey))).tag(status.rawValue)
                                        }
                                    }
                                    .pickerStyle(.segmented)

                                    Text(L.text("mobile.courseSettings.gradingAgents.workflowHint"))
                                        .font(.caption)
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                }
                            }

                            if onDeleted != nil {
                                Button(role: .destructive) {
                                    showDeleteConfirm = true
                                } label: {
                                    Label(L.text("mobile.courseSettings.gradingAgents.deleteButton"), systemImage: "trash")
                                        .frame(maxWidth: .infinity)
                                }
                                .buttonStyle(.bordered)
                                .disabled(deleting)
                            }
                        }
                    }
                    .padding(16)
                }

                if isDirty {
                    UnsavedChangesBanner(
                        isSaving: saving,
                        onSave: { Task { await saveChanges() } },
                        onDiscard: { form = baseline }
                    )
                }
            }
            .navigationTitle(L.text("mobile.courseSettings.gradingAgents.editTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) {
                        onDismiss()
                        dismiss()
                    }
                }
            }
            .confirmationDialog(
                L.text("mobile.courseSettings.gradingAgents.deleteConfirmTitle"),
                isPresented: $showDeleteConfirm,
                titleVisibility: .visible
            ) {
                Button(L.text("mobile.courseSettings.gradingAgents.deleteButton"), role: .destructive) {
                    Task { await deleteAgent() }
                }
                Button(L.text("mobile.common.cancel"), role: .cancel) {}
            } message: {
                Text(L.text("mobile.courseSettings.gradingAgents.deleteConfirmMessage"))
            }
            .task { await loadConfigIfNeeded() }
        }
    }

    private func loadConfigIfNeeded() async {
        if let initialDraft {
            baseline = initialDraft
            form = initialDraft
            return
        }
        guard loadedConfig else { return }
        guard let token = session.accessToken else { return }
        loadingConfig = true
        loadError = nil
        defer { loadingConfig = false }
        do {
            let config = try await LMSAPI.fetchGraderAgentConfig(
                courseCode: course.courseCode,
                itemId: itemId,
                itemKind: itemKind,
                accessToken: token
            )
            let draft = CourseGradingAgentsLogic.draft(from: config)
            baseline = draft
            form = draft
        } catch {
            loadError = error.localizedDescription
        }
    }

    private func saveChanges() async {
        guard let token = session.accessToken else { return }
        actionError = nil
        actionSuccess = nil

        if CourseGradingAgentsLogic.validateDraft(form) == .promptRequired {
            actionError = L.text("mobile.courseSettings.gradingAgents.validation.promptRequired")
            return
        }

        saving = true
        defer { saving = false }

        do {
            let body = CourseGradingAgentsLogic.buildPutBody(current: form, itemKind: itemKind)
            _ = try await offline.enqueueMutation(
                method: "PUT",
                path: CourseGradingAgentsLogic.graderAgentPath(
                    courseCode: course.courseCode,
                    itemId: itemId,
                    itemKind: itemKind
                ),
                body: body,
                label: L.text("mobile.courseSettings.gradingAgents.saveLabel"),
                accessToken: token,
                idempotencyKey: CourseGradingAgentsLogic.saveIdempotencyKey(
                    courseCode: course.courseCode,
                    itemId: itemId,
                    itemKind: itemKind
                )
            )
            let refreshed = try await LMSAPI.fetchGraderAgentConfig(
                courseCode: course.courseCode,
                itemId: itemId,
                itemKind: itemKind,
                accessToken: token
            )
            let draft = CourseGradingAgentsLogic.draft(from: refreshed)
            baseline = draft
            form = draft
            actionSuccess = L.text("mobile.courseSettings.gradingAgents.saved")
            onSaved()
        } catch {
            actionError = error.localizedDescription
        }
    }

    private func deleteAgent() async {
        guard let token = session.accessToken else { return }
        deleting = true
        defer { deleting = false }
        do {
            _ = try await offline.enqueueMutation(
                method: "DELETE",
                path: CourseGradingAgentsLogic.graderAgentPath(
                    courseCode: course.courseCode,
                    itemId: itemId,
                    itemKind: itemKind
                ),
                body: nil,
                label: L.text("mobile.courseSettings.gradingAgents.deleteLabel"),
                accessToken: token,
                idempotencyKey: CourseGradingAgentsLogic.deleteIdempotencyKey(
                    courseCode: course.courseCode,
                    itemId: itemId,
                    itemKind: itemKind
                )
            )
            onDeleted?()
        } catch {
            actionError = error.localizedDescription
        }
    }
}
