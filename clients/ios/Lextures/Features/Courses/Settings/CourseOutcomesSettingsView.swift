import SwiftUI

/// Learning outcomes CRUD, item mapping, and class progress (M13.5).
struct CourseOutcomesSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var outcomes: [CourseOutcome] = []
    @State private var enrolledLearners = 0
    @State private var structure: [CourseStructureItem] = []
    @State private var drafts: [String: CourseOutcomesLogic.OutcomeDraft] = [:]
    @State private var loading = true
    @State private var loadError: String?
    @State private var actionError: String?
    @State private var actionSuccess: String?
    @State private var cacheLabel: String?
    @State private var saving = false
    @State private var creating = false
    @State private var newTitle = ""
    @State private var newDescription = ""
    @State private var deleteOutcomeId: String?
    @State private var showAddAnother = false

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var gradableOptions: [CourseOutcomesLogic.GradableOption] {
        CourseOutcomesLogic.gradableOptions(from: structure)
    }
    private var isDirty: Bool {
        CourseOutcomesLogic.isDirty(drafts: drafts, outcomes: outcomes)
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

                        if !outcomes.isEmpty {
                            Text(L.format("mobile.courseSettings.outcomes.listCount", String(outcomes.count)))
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }

                        ForEach(outcomes) { outcome in
                            OutcomeCardView(
                                course: course,
                                outcome: outcome,
                                enrolledLearners: enrolledLearners,
                                gradableOptions: gradableOptions,
                                draft: binding(for: outcome.id),
                                onDelete: { deleteOutcomeId = outcome.id },
                                onLinksChanged: { Task { await reload(force: true) } }
                            )
                        }

                        if outcomes.isEmpty {
                            createOutcomeForm(compact: false)
                        } else {
                            addAnotherSection
                        }
                    }
                }
                .padding(16)
            }

            if isDirty {
                UnsavedChangesBanner(
                    isSaving: saving,
                    onSave: { Task { await saveDrafts() } },
                    onDiscard: discardDrafts
                )
            }
        }
        .task(id: course.courseCode) { await reload(force: false) }
        .confirmationDialog(
            L.text("mobile.courseSettings.outcomes.deleteConfirmTitle"),
            isPresented: Binding(
                get: { deleteOutcomeId != nil },
                set: { if !$0 { deleteOutcomeId = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.courseSettings.outcomes.deleteButton"), role: .destructive) {
                if let id = deleteOutcomeId {
                    Task { await deleteOutcome(id) }
                }
            }
            Button(L.text("mobile.common.cancel"), role: .cancel) {
                deleteOutcomeId = nil
            }
        } message: {
            Text(L.text("mobile.courseSettings.outcomes.deleteConfirmMessage"))
        }
    }

    private var introCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.courseSettings.outcomes.introTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.outcomes.introDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    @ViewBuilder
    private var addAnotherSection: some View {
        if showAddAnother {
            createOutcomeForm(compact: true)
        } else {
            Button {
                showAddAnother = true
            } label: {
                Label(L.text("mobile.courseSettings.outcomes.addAnother"), systemImage: "plus")
                    .font(.subheadline.weight(.semibold))
            }
            .buttonStyle(.bordered)
        }
    }

    private func createOutcomeForm(compact: Bool) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                if !compact {
                    Text(L.text("mobile.courseSettings.outcomes.emptyTitle"))
                        .font(.headline)
                    Text(L.text("mobile.courseSettings.outcomes.emptyDescription"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                TextField(L.text("mobile.courseSettings.outcomes.titlePlaceholder"), text: $newTitle)
                    .textFieldStyle(.roundedBorder)
                TextField(L.text("mobile.courseSettings.outcomes.descriptionPlaceholder"), text: $newDescription, axis: .vertical)
                    .textFieldStyle(.roundedBorder)
                    .lineLimit(2 ... 4)
                Button {
                    Task { await createOutcome() }
                } label: {
                    if creating {
                        ProgressView()
                    } else {
                        Label(L.text("mobile.courseSettings.outcomes.createButton"), systemImage: "plus")
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(creating || newTitle.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
    }

    private func binding(for outcomeId: String) -> Binding<CourseOutcomesLogic.OutcomeDraft> {
        Binding(
            get: {
                drafts[outcomeId] ?? CourseOutcomesLogic.OutcomeDraft(title: "", description: "")
            },
            set: { drafts[outcomeId] = $0 }
        )
    }

    private func discardDrafts() {
        drafts = CourseOutcomesLogic.drafts(from: outcomes)
        actionError = nil
    }

    private func reload(force: Bool) async {
        guard let token = session.accessToken else { return }
        if !force && !outcomes.isEmpty { return }
        loading = outcomes.isEmpty
        loadError = nil
        defer { loading = false }

        do {
            let outcomesResult = try await offline.cachedFetch(
                key: CourseOutcomesLogic.cacheKeyOutcomes(courseCode: course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourseOutcomes(courseCode: course.courseCode, accessToken: token)
            }
            outcomes = outcomesResult.value.outcomes
            enrolledLearners = outcomesResult.value.enrolledLearners
            drafts = CourseOutcomesLogic.drafts(from: outcomes)
            structure = (try? await LMSAPI.fetchCourseStructure(courseCode: course.courseCode, accessToken: token)) ?? []
            if let cached = outcomesResult.cached, cached.isStale(isOnline: isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            loadError = error.localizedDescription
        }
    }

    private func saveDrafts() async {
        guard let token = session.accessToken else { return }
        actionError = nil
        actionSuccess = nil

        if let error = CourseOutcomesLogic.validateDrafts(drafts: drafts, outcomes: outcomes) {
            switch error {
            case .titleRequired:
                actionError = L.text("mobile.courseSettings.outcomes.validation.titleRequired")
            default:
                break
            }
            return
        }

        let dirtyIds = CourseOutcomesLogic.dirtyOutcomeIds(drafts: drafts, outcomes: outcomes)
        guard !dirtyIds.isEmpty else { return }

        saving = true
        defer { saving = false }

        do {
            for id in dirtyIds {
                guard let draft = drafts[id] else { continue }
                let body = CourseOutcomesLogic.buildPatchBody(draft: draft)
                _ = try await offline.enqueueMutation(
                    method: "PATCH",
                    path: "/api/v1/courses/\(course.courseCode)/outcomes/\(id)",
                    body: body,
                    label: L.text("mobile.courseSettings.outcomes.saveLabel"),
                    accessToken: token,
                    idempotencyKey: "\(CourseOutcomesLogic.saveIdempotencyKey(courseCode: course.courseCode)):\(id)"
                )
            }
            let response = try await LMSAPI.fetchCourseOutcomes(courseCode: course.courseCode, accessToken: token)
            outcomes = response.outcomes
            enrolledLearners = response.enrolledLearners
            drafts = CourseOutcomesLogic.drafts(from: outcomes)
            actionSuccess = L.text("mobile.courseSettings.outcomes.saved")
        } catch {
            actionError = error.localizedDescription
        }
    }

    private func createOutcome() async {
        guard let token = session.accessToken else { return }
        if let error = CourseOutcomesLogic.validateCreateTitle(newTitle) {
            switch error {
            case .titleRequired:
                actionError = L.text("mobile.courseSettings.outcomes.validation.titleRequired")
            default:
                break
            }
            return
        }

        creating = true
        actionError = nil
        defer { creating = false }

        do {
            let body = CourseOutcomesLogic.buildCreateBody(title: newTitle, description: newDescription)
            _ = try await offline.enqueueMutation(
                method: "POST",
                path: "/api/v1/courses/\(course.courseCode)/outcomes",
                body: body,
                label: L.text("mobile.courseSettings.outcomes.createButton"),
                accessToken: token,
                idempotencyKey: CourseOutcomesLogic.createOutcomeIdempotencyKey(courseCode: course.courseCode)
            )
            let response = try await LMSAPI.fetchCourseOutcomes(courseCode: course.courseCode, accessToken: token)
            outcomes = response.outcomes
            enrolledLearners = response.enrolledLearners
            drafts = CourseOutcomesLogic.drafts(from: outcomes)
            newTitle = ""
            newDescription = ""
            showAddAnother = false
            actionSuccess = L.text("mobile.courseSettings.outcomes.created")
        } catch {
            actionError = error.localizedDescription
        }
    }

    private func deleteOutcome(_ outcomeId: String) async {
        guard let token = session.accessToken else { return }
        actionError = nil
        deleteOutcomeId = nil

        do {
            _ = try await offline.enqueueMutation(
                method: "DELETE",
                path: "/api/v1/courses/\(course.courseCode)/outcomes/\(outcomeId)",
                body: nil,
                label: L.text("mobile.courseSettings.outcomes.deleteOutcomeLabel"),
                accessToken: token,
                idempotencyKey: CourseOutcomesLogic.deleteOutcomeIdempotencyKey(
                    courseCode: course.courseCode,
                    outcomeId: outcomeId
                )
            )
            outcomes.removeAll { $0.id == outcomeId }
            drafts.removeValue(forKey: outcomeId)
        } catch {
            actionError = error.localizedDescription
            await reload(force: true)
        }
    }
}

private struct OutcomeCardView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let outcome: CourseOutcome
    let enrolledLearners: Int
    let gradableOptions: [CourseOutcomesLogic.GradableOption]
    @Binding var draft: CourseOutcomesLogic.OutcomeDraft
    let onDelete: () -> Void
    let onLinksChanged: () -> Void

    @State private var itemId = ""
    @State private var quizScopeWhole = true
    @State private var questionId = ""
    @State private var quizQuestions: [CourseOutcomesLogic.QuizQuestionOption] = []
    @State private var loadingQuiz = false
    @State private var addingLink = false
    @State private var localError: String?
    @State private var measurementLevel = CourseOutcomesLogic.defaultMeasurement.rawValue
    @State private var intensityLevel = CourseOutcomesLogic.defaultIntensity.rawValue

    private var selectedGradable: CourseOutcomesLogic.GradableOption? {
        gradableOptions.first { $0.id == itemId }
    }

    var body: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 8) {
                        TextField(L.text("mobile.courseSettings.outcomes.titleLabel"), text: $draft.title)
                            .textFieldStyle(.roundedBorder)
                            .font(.headline)
                        TextField(L.text("mobile.courseSettings.outcomes.descriptionLabel"), text: $draft.description, axis: .vertical)
                            .textFieldStyle(.roundedBorder)
                            .lineLimit(2 ... 4)
                    }
                    Button(role: .destructive, action: onDelete) {
                        Image(systemName: "trash")
                    }
                    .accessibilityLabel(L.text("mobile.courseSettings.outcomes.deleteButton"))
                }

                classProgressSection
                linksSection
                addLinkForm
            }
        }
        .task(id: itemId) { await loadQuizQuestionsIfNeeded() }
    }

    private var classProgressSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(L.text("mobile.courseSettings.outcomes.classProgressTitle"))
                .font(.subheadline.weight(.semibold))
            if let rollup = CourseOutcomesLogic.rollupPercentLabel(outcome.rollupAvgScorePercent) {
                Text(L.format("mobile.courseSettings.outcomes.classProgressRollup", rollup))
                    .font(.subheadline)
                ProgressView(value: min(100, max(0, outcome.rollupAvgScorePercent ?? 0)) / 100)
                    .accessibilityLabel(L.text("mobile.courseSettings.outcomes.classProgressTitle"))
            } else {
                Text(L.format("mobile.courseSettings.outcomes.classProgressEmpty", String(enrolledLearners)))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private var linksSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.courseSettings.outcomes.linksTitle"))
                .font(.subheadline.weight(.semibold))
            Text(L.text("mobile.courseSettings.outcomes.linksDescription"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            if outcome.links.isEmpty {
                Text(L.text("mobile.courseSettings.outcomes.linksEmpty"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(outcome.links) { link in
                    HStack(alignment: .top) {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(CourseOutcomesLogic.linkSummary(for: link))
                                .font(.subheadline.weight(.medium))
                            Text(CourseOutcomesLogic.progressLabel(for: link))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        Spacer(minLength: 0)
                        Button {
                            Task { await removeLink(link.id) }
                        } label: {
                            Image(systemName: "trash")
                        }
                        .accessibilityLabel(L.text("mobile.courseSettings.outcomes.removeLink"))
                    }
                    .padding(10)
                    .background(LexturesTheme.cardBackground(for: colorScheme), in: RoundedRectangle(cornerRadius: 10))
                }
            }
        }
    }

    private var addLinkForm: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.courseSettings.outcomes.addLinkTitle"))
                .font(.subheadline.weight(.semibold))
            if let localError {
                Text(localError)
                    .font(.caption)
                    .foregroundStyle(.red)
            }

            Picker(L.text("mobile.courseSettings.outcomes.selectItem"), selection: $itemId) {
                Text(L.text("mobile.courseSettings.outcomes.selectItemPlaceholder")).tag("")
                ForEach(gradableOptions) { option in
                    let prefix = option.kind == "quiz"
                        ? L.text("mobile.courseSettings.outcomes.kind.quiz")
                        : L.text("mobile.courseSettings.outcomes.kind.assignment")
                    Text("\(prefix): \(option.label)").tag(option.id)
                }
            }
            .pickerStyle(.menu)
            .onChange(of: itemId) { _, _ in
                quizScopeWhole = true
                questionId = ""
            }

            if selectedGradable?.kind == "quiz" {
                Picker(L.text("mobile.courseSettings.outcomes.selectQuestion"), selection: $quizScopeWhole) {
                    Text(L.text("mobile.courseSettings.outcomes.quizScopeWhole")).tag(true)
                    Text(L.text("mobile.courseSettings.outcomes.quizScopeQuestion")).tag(false)
                }
                .pickerStyle(.segmented)

                if !quizScopeWhole {
                    if loadingQuiz {
                        ProgressView()
                    } else {
                        Picker(L.text("mobile.courseSettings.outcomes.selectQuestion"), selection: $questionId) {
                            ForEach(quizQuestions) { question in
                                Text(question.prompt).tag(question.id)
                            }
                        }
                        .pickerStyle(.menu)
                    }
                }
            }

            Picker(L.text("mobile.courseSettings.outcomes.measurementLevel"), selection: $measurementLevel) {
                ForEach(CourseOutcomesLogic.MeasurementLevelId.allCases) { level in
                    Text(L.text(String.LocalizationValue(level.labelKey))).tag(level.rawValue)
                }
            }
            .pickerStyle(.menu)

            Picker(L.text("mobile.courseSettings.outcomes.intensityLevel"), selection: $intensityLevel) {
                ForEach(CourseOutcomesLogic.IntensityLevelId.allCases) { level in
                    Text(L.text(String.LocalizationValue(level.labelKey))).tag(level.rawValue)
                }
            }
            .pickerStyle(.menu)

            Button {
                Task { await addLink() }
            } label: {
                if addingLink {
                    ProgressView()
                } else {
                    Label(L.text("mobile.courseSettings.outcomes.addLinkButton"), systemImage: "link")
                }
            }
            .buttonStyle(.bordered)
            .disabled(addingLink || itemId.isEmpty)
        }
    }

    private func loadQuizQuestionsIfNeeded() async {
        guard let token = session.accessToken,
              let selected = selectedGradable,
              selected.kind == "quiz",
              !itemId.isEmpty
        else {
            quizQuestions = []
            questionId = ""
            return
        }

        loadingQuiz = true
        defer { loadingQuiz = false }

        do {
            let quiz = try await LMSAPI.fetchModuleQuiz(
                courseCode: course.courseCode,
                itemId: itemId,
                attemptId: nil,
                accessToken: token
            )
            quizQuestions = CourseOutcomesLogic.questionOptions(from: quiz.questions)
            if !quizQuestions.contains(where: { $0.id == questionId }) {
                questionId = quizQuestions.first?.id ?? ""
            }
        } catch {
            quizQuestions = []
            questionId = ""
        }
    }

    private func addLink() async {
        guard let token = session.accessToken else { return }
        localError = nil

        guard let selected = selectedGradable else {
            localError = L.text("mobile.courseSettings.outcomes.validation.selectItem")
            return
        }

        let targetKind = CourseOutcomesLogic.targetKind(
            gradableKind: selected.kind,
            quizScopeWhole: quizScopeWhole
        )
        if targetKind == "quiz_question" && questionId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            localError = L.text("mobile.courseSettings.outcomes.validation.selectQuestion")
            return
        }

        addingLink = true
        defer { addingLink = false }

        do {
            let body = CourseOutcomesLogic.buildAddLinkBody(
                structureItemId: itemId,
                targetKind: targetKind,
                quizQuestionId: targetKind == "quiz_question" ? questionId : nil,
                measurementLevel: measurementLevel,
                intensityLevel: intensityLevel
            )
            _ = try await offline.enqueueMutation(
                method: "POST",
                path: "/api/v1/courses/\(course.courseCode)/outcomes/\(outcome.id)/links",
                body: body,
                label: L.text("mobile.courseSettings.outcomes.createLinkLabel"),
                accessToken: token,
                idempotencyKey: CourseOutcomesLogic.addLinkIdempotencyKey(
                    courseCode: course.courseCode,
                    outcomeId: outcome.id,
                    itemId: itemId
                )
            )
            itemId = ""
            quizScopeWhole = true
            questionId = ""
            measurementLevel = CourseOutcomesLogic.defaultMeasurement.rawValue
            intensityLevel = CourseOutcomesLogic.defaultIntensity.rawValue
            onLinksChanged()
        } catch {
            localError = error.localizedDescription
        }
    }

    private func removeLink(_ linkId: String) async {
        guard let token = session.accessToken else { return }
        localError = nil

        do {
            _ = try await offline.enqueueMutation(
                method: "DELETE",
                path: "/api/v1/courses/\(course.courseCode)/outcomes/\(outcome.id)/links/\(linkId)",
                body: nil,
                label: L.text("mobile.courseSettings.outcomes.deleteLinkLabel"),
                accessToken: token,
                idempotencyKey: CourseOutcomesLogic.deleteLinkIdempotencyKey(
                    courseCode: course.courseCode,
                    outcomeId: outcome.id,
                    linkId: linkId
                )
            )
            onLinksChanged()
        } catch {
            localError = error.localizedDescription
        }
    }
}