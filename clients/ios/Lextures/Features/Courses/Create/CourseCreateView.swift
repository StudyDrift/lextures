import SwiftUI

/// Full-screen create-course wizard (M11.5 / MOB.1).
struct CourseCreateView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let existingCourses: [CourseSummary]
    var onFinished: (CourseSummary) -> Void

    @State private var step: CourseCreateLogic.WizardStep = .basics
    @State private var title = ""
    @State private var description = ""
    @State private var courseMode: CourseCreateLogic.CourseMode = .traditional
    @State private var selectedTermId = ""
    @State private var selectedGradeLevel = ""
    @State private var selectedTemplateId = CourseCreateLogic.defaultTemplateId
    @State private var firstModuleTitle = ""
    @State private var competencies: [CourseCreateLogic.CompetencyDraft] = [.empty()]
    @State private var createdCourse: CourseSummary?
    @State private var terms: [OrgTerm] = []
    @State private var loadingTerms = false
    @State private var submitting = false
    @State private var errorMessage: String?
    @State private var titleError: String?
    @State private var showCancelConfirm = false
    @State private var showCanvasComingSoon = false
    @State private var draftKey = ""
    @State private var didRestoreDraft = false
    @State private var recordedStart = false

    private var isCompetency: Bool { courseMode == .competencyBased }
    private var v2Enabled: Bool { CourseCreateLogic.courseCreateV2Enabled(shell.platformFeatures) }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                VStack(spacing: 0) {
                    if step != .source {
                        progressHeader
                    }
                    ScrollView {
                        VStack(alignment: .leading, spacing: 16) {
                            if let errorMessage {
                                LMSErrorBanner(message: errorMessage)
                            }
                            switch step {
                            case .source:
                                sourceStep
                            case .basics:
                                basicsStep
                            case .syllabus:
                                syllabusStep
                            case .finish:
                                finishStep
                            }
                        }
                        .padding(16)
                    }
                    bottomBar
                }
            }
            .navigationTitle(L.text("mobile.createCourse.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) {
                        attemptDismiss()
                    }
                    .disabled(submitting)
                }
            }
            .interactiveDismissDisabled(submitting || CourseCreateLogic.shouldConfirmCancel(createdCourseCode: createdCourse?.courseCode))
            .confirmationDialog(
                L.text("mobile.createCourse.cancel.confirm"),
                isPresented: $showCancelConfirm,
                titleVisibility: .visible
            ) {
                Button(L.text("mobile.createCourse.cancel.leave"), role: .destructive) {
                    clearDraft()
                    dismiss()
                }
                Button(L.text("mobile.common.close"), role: .cancel) {}
            } message: {
                Text(L.text("mobile.createCourse.cancel.message"))
            }
            .sheet(isPresented: $showCanvasComingSoon) {
                CanvasImportComingSoonView {
                    showCanvasComingSoon = false
                    step = .source
                }
            }
            .task { await bootstrap() }
            .onChange(of: step) { _, _ in persistDraft() }
            .onChange(of: title) { _, _ in persistDraft() }
            .onChange(of: description) { _, _ in persistDraft() }
            .onChange(of: courseMode) { _, _ in persistDraft() }
            .onChange(of: selectedTermId) { _, _ in persistDraft() }
            .onChange(of: selectedGradeLevel) { _, _ in persistDraft() }
            .onChange(of: selectedTemplateId) { _, _ in persistDraft() }
            .onChange(of: firstModuleTitle) { _, _ in persistDraft() }
            .onChange(of: competencies) { _, _ in persistDraft() }
            .onChange(of: createdCourse?.courseCode) { _, _ in persistDraft() }
        }
    }

    private var progressHeader: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.format("mobile.createCourse.stepOf", step.rawValue, 3))
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            HStack(spacing: 8) {
                ForEach(CourseCreateLogic.WizardStep.progressSteps) { wizardStep in
                    let active = wizardStep <= step
                    VStack(alignment: .leading, spacing: 4) {
                        Capsule()
                            .fill(active ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.textSecondary(for: colorScheme).opacity(0.25))
                            .frame(height: 4)
                        Text(L.text(String.LocalizationValue(wizardStep.finishLabelKey(isCompetency: isCompetency))))
                            .font(.caption2)
                            .foregroundStyle(active ? LexturesTheme.textPrimary(for: colorScheme) : LexturesTheme.textSecondary(for: colorScheme))
                            .lineLimit(1)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel(L.format("mobile.createCourse.stepOf", step.rawValue, 3))
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }

    private var sourceStep: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(L.text("mobile.createCourse.source.intro"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            sourceCard(
                title: L.text("mobile.createCourse.source.scratch.title"),
                summary: L.text("mobile.createCourse.source.scratch.summary"),
                systemImage: "doc.badge.plus"
            ) {
                step = .basics
                maybeRecordStarted()
            }
            sourceCard(
                title: L.text("mobile.createCourse.source.canvas.title"),
                summary: L.text("mobile.createCourse.source.canvas.summary"),
                systemImage: "square.and.arrow.down.on.square"
            ) {
                showCanvasComingSoon = true
            }
        }
    }

    private func sourceCard(title: String, summary: String, systemImage: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack(alignment: .top, spacing: 12) {
                Image(systemName: systemImage)
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 28, height: 28)
                VStack(alignment: .leading, spacing: 4) {
                    Text(title)
                        .font(.headline)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(summary)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .multilineTextAlignment(.leading)
                }
                Spacer(minLength: 0)
            }
            .padding(14)
            .background(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .fill(LexturesTheme.cardBackground(for: colorScheme))
            )
        }
        .buttonStyle(.plain)
        .frame(minHeight: 44)
    }

    private var basicsStep: some View {
        VStack(alignment: .leading, spacing: 14) {
            fieldLabel(L.text("mobile.createCourse.field.title"))
            TextField(L.text("mobile.createCourse.field.titlePlaceholder"), text: $title)
                .textFieldStyle(.roundedBorder)
                .accessibilityLabel(L.text("mobile.createCourse.field.title"))
            if let titleError {
                Text(L.text(String.LocalizationValue(titleError)))
                    .font(.caption)
                    .foregroundStyle(.red)
            }

            fieldLabel(L.text("mobile.createCourse.field.description"))
            TextField(L.text("mobile.createCourse.field.descriptionPlaceholder"), text: $description, axis: .vertical)
                .lineLimit(3...6)
                .textFieldStyle(.roundedBorder)

            fieldLabel(L.text("mobile.createCourse.field.mode"))
            Picker(L.text("mobile.createCourse.field.mode"), selection: $courseMode) {
                Text(L.text("mobile.createCourse.mode.traditional")).tag(CourseCreateLogic.CourseMode.traditional)
                Text(L.text("mobile.createCourse.mode.competency")).tag(CourseCreateLogic.CourseMode.competencyBased)
            }
            .pickerStyle(.segmented)
            Text(L.text(isCompetency ? "mobile.createCourse.mode.competencyHint" : "mobile.createCourse.mode.traditionalHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            fieldLabel(L.text("mobile.createCourse.field.term"))
            if loadingTerms {
                ProgressView()
            } else {
                Picker(L.text("mobile.createCourse.field.term"), selection: $selectedTermId) {
                    Text(L.text("mobile.createCourse.term.none")).tag("")
                    ForEach(terms) { term in
                        Text(term.name).tag(term.id)
                    }
                }
                .pickerStyle(.menu)
            }

            fieldLabel(L.text("mobile.createCourse.field.gradeLevel"))
            Picker(L.text("mobile.createCourse.field.gradeLevel"), selection: $selectedGradeLevel) {
                Text(L.text("mobile.createCourse.gradeLevel.none")).tag("")
                ForEach(CourseCreateLogic.gradeLevels.filter { !$0.isEmpty }, id: \.self) { level in
                    Text(level).tag(level)
                }
            }
            .pickerStyle(.menu)
        }
    }

    private var syllabusStep: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(L.text("mobile.createCourse.syllabus.intro"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            templateCard(
                id: CourseCreateLogic.blankTemplateId,
                name: L.text("mobile.createCourse.template.blank"),
                summary: L.text("mobile.createCourse.template.blankSummary")
            )

            ForEach(CourseCreateLogic.starterTemplates) { tmpl in
                templateCard(
                    id: tmpl.id,
                    name: L.text(String.LocalizationValue(tmpl.nameKey)),
                    summary: L.text(String.LocalizationValue(tmpl.summaryKey))
                )
            }
        }
    }

    private func templateCard(id: String, name: String, summary: String) -> some View {
        let selected = selectedTemplateId == id
        return Button {
            selectedTemplateId = id
        } label: {
            HStack(alignment: .top, spacing: 12) {
                Image(systemName: selected ? "checkmark.circle.fill" : "circle")
                    .foregroundStyle(selected ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.textSecondary(for: colorScheme))
                VStack(alignment: .leading, spacing: 4) {
                    Text(name)
                        .font(.headline)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(summary)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .multilineTextAlignment(.leading)
                }
                Spacer(minLength: 0)
            }
            .padding(14)
            .background(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .fill(LexturesTheme.cardBackground(for: colorScheme))
            )
            .overlay(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .stroke(selected ? LexturesTheme.accent(for: colorScheme) : Color.clear, lineWidth: 2)
            )
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(selected ? .isSelected : [])
    }

    private var finishStep: some View {
        VStack(alignment: .leading, spacing: 14) {
            if isCompetency {
                if v2Enabled {
                    competencyEditor
                } else {
                    Text(L.text("mobile.createCourse.competency.handoffTitle"))
                        .font(.headline)
                    Text(L.text("mobile.createCourse.competency.handoffBody"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            } else {
                fieldLabel(L.text("mobile.createCourse.firstModule.label"))
                TextField(L.text("mobile.createCourse.firstModule.placeholder"), text: $firstModuleTitle)
                    .textFieldStyle(.roundedBorder)
                Text(L.text("mobile.createCourse.firstModule.hint"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private var competencyEditor: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(L.text("mobile.createCourse.competency.intro"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            ForEach(Array(competencies.enumerated()), id: \.element.id) { index, _ in
                competencyCard(index: index)
            }

            Button {
                competencies.append(.empty())
            } label: {
                Label(L.text("mobile.createCourse.competency.add"), systemImage: "plus.circle")
            }
            .frame(minHeight: 44)
        }
    }

    private func competencyCard(index: Int) -> some View {
        let binding = $competencies[index]
        return VStack(alignment: .leading, spacing: 10) {
            HStack {
                Button {
                    competencies[index].expanded.toggle()
                } label: {
                    HStack {
                        Image(systemName: competencies[index].expanded ? "chevron.down" : "chevron.right")
                        Text(L.format("mobile.createCourse.competency.heading", index + 1))
                            .font(.headline)
                    }
                }
                .buttonStyle(.plain)
                Spacer()
                if competencies.count > 1 {
                    Button(role: .destructive) {
                        competencies.remove(at: index)
                    } label: {
                        Image(systemName: "trash")
                    }
                    .accessibilityLabel(L.text("mobile.createCourse.competency.remove"))
                }
            }

            if competencies[index].expanded {
                TextField(L.text("mobile.createCourse.competency.titlePlaceholder"), text: binding.title)
                    .textFieldStyle(.roundedBorder)
                TextField(L.text("mobile.createCourse.competency.descriptionPlaceholder"), text: binding.description, axis: .vertical)
                    .lineLimit(2...4)
                    .textFieldStyle(.roundedBorder)

                ForEach(Array(competencies[index].subOutcomes.enumerated()), id: \.element.id) { subIndex, _ in
                    subOutcomeEditor(compIndex: index, subIndex: subIndex)
                }

                Button {
                    competencies[index].subOutcomes.append(.empty())
                } label: {
                    Text(L.text("mobile.createCourse.competency.addSubOutcome"))
                }
                .frame(minHeight: 44)
            }
        }
        .padding(14)
        .background(
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .fill(LexturesTheme.cardBackground(for: colorScheme))
        )
    }

    private func subOutcomeEditor(compIndex: Int, subIndex: Int) -> some View {
        let binding = $competencies[compIndex].subOutcomes[subIndex]
        return VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(L.format("mobile.createCourse.competency.subOutcomeHeading", subIndex + 1))
                    .font(.subheadline.weight(.semibold))
                Spacer()
                if competencies[compIndex].subOutcomes.count > 1 {
                    Button(role: .destructive) {
                        competencies[compIndex].subOutcomes.remove(at: subIndex)
                    } label: {
                        Image(systemName: "minus.circle")
                    }
                }
            }
            TextField(L.text("mobile.createCourse.competency.subOutcomeTitlePlaceholder"), text: binding.title)
                .textFieldStyle(.roundedBorder)
            TextField(L.text("mobile.createCourse.competency.subOutcomeDescriptionPlaceholder"), text: binding.description, axis: .vertical)
                .lineLimit(2...3)
                .textFieldStyle(.roundedBorder)
            TextField(L.text("mobile.createCourse.competency.assessmentTitlePlaceholder"), text: binding.assessmentTitle)
                .textFieldStyle(.roundedBorder)
            Picker(L.text("mobile.createCourse.competency.assessmentKind"), selection: binding.assessmentKind) {
                Text(L.text("mobile.createCourse.competency.assessment.quiz")).tag(CourseCreateLogic.AssessmentKind.quiz)
                Text(L.text("mobile.createCourse.competency.assessment.assignment")).tag(CourseCreateLogic.AssessmentKind.assignment)
            }
            .pickerStyle(.segmented)
        }
        .padding(10)
        .background(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .stroke(LexturesTheme.textSecondary(for: colorScheme).opacity(0.25), lineWidth: 1)
        )
    }

    private var bottomBar: some View {
        HStack(spacing: 12) {
            if step != .basics && step != .source {
                Button(L.text("mobile.createCourse.action.back")) {
                    goBack()
                }
                .disabled(submitting)
            } else if step == .basics && v2Enabled {
                Button(L.text("mobile.createCourse.action.back")) {
                    step = .source
                }
                .disabled(submitting)
            }
            Spacer()
            if step == .finish && !isCompetency {
                Button(L.text("mobile.createCourse.firstModule.skip")) {
                    Task { await finishTraditional(skipModule: true) }
                }
                .disabled(submitting)
            }
            if step != .source {
                Button {
                    Task { await primaryAction() }
                } label: {
                    if submitting {
                        ProgressView()
                            .frame(minWidth: 100)
                    } else {
                        Text(primaryButtonTitle)
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(submitting)
            }
        }
        .padding(16)
        .background(.ultraThinMaterial)
    }

    private var primaryButtonTitle: String {
        switch step {
        case .source:
            return L.text("mobile.createCourse.action.continue")
        case .basics, .syllabus:
            return L.text("mobile.createCourse.action.continue")
        case .finish:
            if isCompetency && v2Enabled {
                return L.text("mobile.createCourse.action.createCompetencies")
            }
            return L.text("mobile.createCourse.action.createOpen")
        }
    }

    private func fieldLabel(_ text: String) -> some View {
        Text(text)
            .font(.subheadline.weight(.semibold))
            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
    }

    private func attemptDismiss() {
        if CourseCreateLogic.shouldConfirmCancel(createdCourseCode: createdCourse?.courseCode) {
            showCancelConfirm = true
        } else {
            clearDraft()
            dismiss()
        }
    }

    private func goBack() {
        errorMessage = nil
        titleError = nil
        switch step {
        case .syllabus:
            if let created = createdCourse {
                title = created.title
                description = created.description
                courseMode = CourseCreateLogic.modeFromCourseType(created.courseType)
                selectedTermId = created.termId ?? selectedTermId
                selectedGradeLevel = created.gradeLevel ?? selectedGradeLevel
            }
            step = .basics
        case .finish:
            step = .syllabus
        case .basics, .source:
            break
        }
    }

    private func primaryAction() async {
        switch step {
        case .source:
            break
        case .basics:
            await submitBasics()
        case .syllabus:
            await continueFromSyllabus()
        case .finish:
            if isCompetency {
                if v2Enabled {
                    await finishCompetencyBased()
                } else {
                    await finishCompetencyHandoff()
                }
            } else {
                await finishTraditional(skipModule: false)
            }
        }
    }

    private func bootstrap() async {
        step = CourseCreateLogic.initialWizardStep(v2Enabled: v2Enabled)
        let orgId = CourseCreateLogic.resolveOrgId(accessToken: session.accessToken, courses: existingCourses)
        draftKey = CourseCreateDraftStore.storageKey(userId: session.userEmail, orgId: orgId)
        if v2Enabled, !didRestoreDraft, let draft = CourseCreateDraftStore.load(key: draftKey) {
            restore(draft)
            didRestoreDraft = true
        }
        await loadTerms()
        if step != .source {
            maybeRecordStarted()
        }
    }

    private func restore(_ draft: CourseCreateDraftStore.Draft) {
        step = CourseCreateLogic.WizardStep(rawValue: draft.step) ?? .basics
        if step == .source && !v2Enabled { step = .basics }
        title = draft.title
        description = draft.description
        courseMode = CourseCreateLogic.modeFromCourseType(draft.courseMode)
        selectedTermId = draft.selectedTermId
        selectedGradeLevel = draft.selectedGradeLevel
        selectedTemplateId = draft.selectedTemplateId
        firstModuleTitle = draft.firstModuleTitle
        competencies = draft.competencies.isEmpty ? [.empty()] : draft.competencies
        if let code = draft.createdCourseCode, !code.isEmpty {
            createdCourse = CourseSummary(
                id: code,
                courseCode: code,
                title: draft.title,
                description: draft.description,
                published: false,
                courseType: draft.courseMode,
                termId: draft.selectedTermId.isEmpty ? nil : draft.selectedTermId,
                gradeLevel: draft.selectedGradeLevel.isEmpty ? nil : draft.selectedGradeLevel
            )
        }
    }

    private func persistDraft() {
        guard v2Enabled, !draftKey.isEmpty else { return }
        let draft = CourseCreateDraftStore.Draft(
            step: step.rawValue,
            title: title,
            description: description,
            courseMode: courseMode.rawValue,
            selectedTermId: selectedTermId,
            selectedGradeLevel: selectedGradeLevel,
            selectedTemplateId: selectedTemplateId,
            firstModuleTitle: firstModuleTitle,
            createdCourseCode: createdCourse?.courseCode,
            competencies: competencies,
            createSource: nil
        )
        CourseCreateDraftStore.save(key: draftKey, draft: draft)
    }

    private func clearDraft() {
        guard !draftKey.isEmpty else { return }
        CourseCreateDraftStore.clear(key: draftKey)
    }

    private func maybeRecordStarted() {
        guard v2Enabled, !recordedStart else { return }
        CourseCreateObservability.recordStarted(mode: courseMode.rawValue, templateId: selectedTemplateId)
        recordedStart = true
    }

    private func loadTerms() async {
        guard let token = session.accessToken else { return }
        loadingTerms = true
        defer { loadingTerms = false }
        guard let orgId = CourseCreateLogic.resolveOrgId(accessToken: token, courses: existingCourses) else {
            terms = []
            return
        }
        terms = (try? await LMSAPI.fetchOrgTerms(orgId: orgId, accessToken: token)) ?? []
    }

    private func submitBasics() async {
        titleError = CourseCreateLogic.validateTitle(title)
        if titleError != nil {
            errorMessage = nil
            return
        }
        guard let token = session.accessToken else { return }
        submitting = true
        errorMessage = nil
        defer { submitting = false }
        do {
            if let existing = createdCourse,
               CourseCreateLogic.shouldUpdateExistingCourse(createdCourseCode: existing.courseCode) {
                let body = CourseCreateLogic.buildUpdateRequest(
                    course: existing,
                    title: title,
                    description: description,
                    termId: selectedTermId,
                    gradeLevel: selectedGradeLevel
                )
                createdCourse = try await LMSAPI.updateCourse(
                    courseCode: existing.courseCode,
                    body: body,
                    accessToken: token
                )
            } else {
                let body = CourseCreateLogic.buildCreateRequest(
                    title: title,
                    description: description,
                    mode: courseMode,
                    termId: selectedTermId,
                    gradeLevel: selectedGradeLevel
                )
                createdCourse = try await LMSAPI.createCourse(body: body, accessToken: token)
            }
            if v2Enabled {
                CourseCreateObservability.recordStepCompleted(step: 1)
            }
            step = .syllabus
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.createCourse.error.createFailed")
        }
    }

    private func continueFromSyllabus() async {
        guard let course = createdCourse, let token = session.accessToken else { return }
        submitting = true
        errorMessage = nil
        defer { submitting = false }
        do {
            if CourseCreateLogic.shouldPatchSyllabus(templateId: selectedTemplateId),
               let tmpl = CourseCreateLogic.template(for: selectedTemplateId) {
                let sections = CourseCreateLogic.templateSectionsToSyllabus(tmpl.sections)
                _ = try await LMSAPI.patchCourseSyllabus(
                    courseCode: course.courseCode,
                    body: PatchCourseSyllabusRequest(sections: sections, requireSyllabusAcceptance: false),
                    accessToken: token
                )
            }
            if !isCompetency {
                firstModuleTitle = CourseCreateLogic.suggestedFirstModuleTitle(
                    templateId: selectedTemplateId,
                    existing: firstModuleTitle
                )
            }
            if v2Enabled {
                CourseCreateObservability.recordStepCompleted(step: 2)
            }
            step = .finish
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.createCourse.error.syllabusFailed")
        }
    }

    private func finishTraditional(skipModule: Bool) async {
        guard let course = createdCourse, let token = session.accessToken else { return }
        submitting = true
        errorMessage = nil
        defer { submitting = false }
        do {
            if !skipModule {
                let moduleTitle = firstModuleTitle.trimmingCharacters(in: .whitespacesAndNewlines)
                if !moduleTitle.isEmpty {
                    _ = try await LMSAPI.createCourseModule(
                        courseCode: course.courseCode,
                        title: moduleTitle,
                        accessToken: token
                    )
                }
            }
            if v2Enabled {
                CourseCreateObservability.recordStepCompleted(step: 3)
                CourseCreateObservability.recordFinished(mode: courseMode.rawValue, templateId: selectedTemplateId)
            }
            clearDraft()
            await shell.refresh(accessToken: token)
            let refreshed = (try? await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)) ?? course
            onFinished(refreshed)
            dismiss()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.createCourse.error.moduleFailed")
        }
    }

    private func finishCompetencyHandoff() async {
        guard let course = createdCourse, let token = session.accessToken else { return }
        submitting = true
        errorMessage = nil
        defer { submitting = false }
        clearDraft()
        await shell.refresh(accessToken: token)
        let refreshed = (try? await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)) ?? course
        onFinished(refreshed)
        dismiss()
    }

    private func finishCompetencyBased() async {
        guard let course = createdCourse, let token = session.accessToken else { return }
        if let validation = CourseCreateLogic.validateCompetencies(competencies) {
            errorMessage = formatCompetencyError(validation)
            return
        }
        submitting = true
        errorMessage = nil
        defer { submitting = false }
        do {
            for comp in competencies {
                let module = try await LMSAPI.createCourseModule(
                    courseCode: course.courseCode,
                    title: comp.title.trimmingCharacters(in: .whitespacesAndNewlines),
                    accessToken: token
                )
                let outcome = try await LMSAPI.createCourseOutcome(
                    courseCode: course.courseCode,
                    body: CreateCourseOutcomeBody(
                        title: comp.title.trimmingCharacters(in: .whitespacesAndNewlines),
                        description: comp.description.trimmingCharacters(in: .whitespacesAndNewlines)
                    ),
                    accessToken: token
                )
                _ = try await LMSAPI.patchCourseOutcome(
                    courseCode: course.courseCode,
                    outcomeId: outcome.id,
                    body: PatchCourseOutcomeBody(moduleStructureItemId: module.id),
                    accessToken: token
                )
                for sub in comp.subOutcomes {
                    let subRow = try await LMSAPI.createCourseOutcomeSubOutcome(
                        courseCode: course.courseCode,
                        outcomeId: outcome.id,
                        body: CreateCourseOutcomeSubOutcomeBody(
                            title: sub.title.trimmingCharacters(in: .whitespacesAndNewlines),
                            description: sub.description.trimmingCharacters(in: .whitespacesAndNewlines)
                        ),
                        accessToken: token
                    )
                    let assessmentTitle = sub.assessmentTitle.trimmingCharacters(in: .whitespacesAndNewlines)
                    let item: CourseStructureItem
                    switch sub.assessmentKind {
                    case .assignment:
                        item = try await LMSAPI.createModuleAssignment(
                            courseCode: course.courseCode,
                            moduleId: module.id,
                            title: assessmentTitle,
                            accessToken: token
                        )
                    case .quiz:
                        item = try await LMSAPI.createModuleQuiz(
                            courseCode: course.courseCode,
                            moduleId: module.id,
                            title: assessmentTitle,
                            accessToken: token
                        )
                    }
                    _ = try await LMSAPI.addCourseOutcomeLink(
                        courseCode: course.courseCode,
                        outcomeId: outcome.id,
                        body: AddCourseOutcomeLinkBody(
                            structureItemId: item.id,
                            targetKind: sub.assessmentKind.rawValue,
                            measurementLevel: "summative",
                            intensityLevel: "high",
                            subOutcomeId: subRow.id
                        ),
                        accessToken: token
                    )
                }
            }
            CourseCreateObservability.recordStepCompleted(step: 3)
            CourseCreateObservability.recordFinished(mode: courseMode.rawValue, templateId: selectedTemplateId)
            clearDraft()
            await shell.refresh(accessToken: token)
            let refreshed = (try? await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)) ?? course
            onFinished(refreshed)
            dismiss()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.createCourse.error.competencyFailed")
        }
    }

    private func formatCompetencyError(_ error: CourseCreateLogic.CompetencyValidationError) -> String {
        switch error.args.count {
        case 0:
            return L.text(String.LocalizationValue(error.key))
        case 1:
            return L.format(String.LocalizationValue(error.key), error.args[0])
        case 2:
            return L.format(String.LocalizationValue(error.key), error.args[0], error.args[1])
        default:
            return L.text(String.LocalizationValue(error.key))
        }
    }
}
