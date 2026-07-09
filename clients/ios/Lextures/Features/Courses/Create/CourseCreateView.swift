import SwiftUI

/// Full-screen create-course wizard (M11.5).
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
    @State private var createdCourse: CourseSummary?
    @State private var terms: [OrgTerm] = []
    @State private var loadingTerms = false
    @State private var submitting = false
    @State private var errorMessage: String?
    @State private var titleError: String?
    @State private var showCancelConfirm = false

    private var isCompetency: Bool { courseMode == .competencyBased }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                VStack(spacing: 0) {
                    progressHeader
                    ScrollView {
                        VStack(alignment: .leading, spacing: 16) {
                            if let errorMessage {
                                LMSErrorBanner(message: errorMessage)
                            }
                            switch step {
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
                    dismiss()
                }
                Button(L.text("mobile.common.close"), role: .cancel) {}
            } message: {
                Text(L.text("mobile.createCourse.cancel.message"))
            }
            .task { await loadTerms() }
        }
    }

    private var progressHeader: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.format("mobile.createCourse.stepOf", step.rawValue, 3))
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            HStack(spacing: 8) {
                ForEach(CourseCreateLogic.WizardStep.allCases) { wizardStep in
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
                Text(L.text("mobile.createCourse.competency.handoffTitle"))
                    .font(.headline)
                Text(L.text("mobile.createCourse.competency.handoffBody"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
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

    private var bottomBar: some View {
        HStack(spacing: 12) {
            if step != .basics {
                Button(L.text("mobile.createCourse.action.back")) {
                    goBack()
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
        .padding(16)
        .background(.ultraThinMaterial)
    }

    private var primaryButtonTitle: String {
        switch step {
        case .basics, .syllabus:
            return L.text("mobile.createCourse.action.continue")
        case .finish:
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
        case .basics:
            break
        }
    }

    private func primaryAction() async {
        switch step {
        case .basics:
            await submitBasics()
        case .syllabus:
            await continueFromSyllabus()
        case .finish:
            if isCompetency {
                await finishCompetencyHandoff()
            } else {
                await finishTraditional(skipModule: false)
            }
        }
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
        await shell.refresh(accessToken: token)
        let refreshed = (try? await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)) ?? course
        onFinished(refreshed)
        dismiss()
    }
}
