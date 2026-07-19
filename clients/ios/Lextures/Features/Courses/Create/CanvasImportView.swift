import SwiftUI

/// Full-screen Canvas course import wizard (MOB.2) — credentials → select → importing.
struct CanvasImportView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let existingCourses: [CourseSummary]
    var onFinished: (CourseSummary) -> Void
    var onBackToSource: (() -> Void)?

    @State private var step: CanvasImportLogic.ImportStep = .credentials
    @State private var canvasBaseUrl = ""
    @State private var canvasToken = ""
    @State private var courses: [CanvasCourseListItem] = []
    @State private var selectedCourseId: Int?
    @State private var include = CanvasImportLogic.Include.all
    @State private var targetMode: CanvasImportLogic.TargetMode = .newCourse
    @State private var importMode: CourseImportExportLogic.ImportMode = .erase
    @State private var existingCourseCode = ""
    @State private var enableGradeSync = false
    @State private var nameFilter = ""
    @State private var hideUnpublished = false
    @State private var progressLog: [String] = []
    @State private var busy = false
    @State private var importComplete = false
    @State private var cancelled = false
    @State private var errorMessage: String?
    @State private var cancelRequested = false
    @State private var importedCourse: CourseSummary?

    private var filteredCourses: [CanvasCourseListItem] {
        var list = CanvasImportLogic.filterCourses(courses, query: nameFilter)
        if hideUnpublished {
            list = list.filter { !CanvasImportLogic.isUnpublished($0.workflowState) }
        }
        return list
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                VStack(spacing: 0) {
                    stepHeader
                    ScrollView {
                        VStack(alignment: .leading, spacing: 16) {
                            if let errorMessage {
                                LMSErrorBanner(message: errorMessage)
                            }
                            switch step {
                            case .credentials:
                                credentialsStep
                            case .select:
                                selectStep
                            case .importing:
                                importingStep
                            }
                        }
                        .padding(16)
                    }
                    bottomBar
                }
            }
            .navigationTitle(L.text("mobile.canvasImport.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) {
                        if busy && !importComplete { return }
                        if let onBackToSource {
                            onBackToSource()
                        } else {
                            dismiss()
                        }
                    }
                    .disabled(busy && !importComplete)
                }
            }
            .interactiveDismissDisabled(busy && !importComplete)
        }
    }

    private var stepHeader: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.format("mobile.canvasImport.stepOf", step.rawValue + 1, 3))
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            HStack(spacing: 8) {
                ForEach(CanvasImportLogic.ImportStep.allCases) { wizardStep in
                    let active = wizardStep <= step
                    VStack(alignment: .leading, spacing: 4) {
                        Capsule()
                            .fill(active ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.textSecondary(for: colorScheme).opacity(0.25))
                            .frame(height: 4)
                        Text(L.text(String.LocalizationValue(wizardStep.labelKey)))
                            .font(.caption2)
                            .foregroundStyle(active ? LexturesTheme.textPrimary(for: colorScheme) : LexturesTheme.textSecondary(for: colorScheme))
                            .lineLimit(1)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel(L.format("mobile.canvasImport.stepOf", step.rawValue + 1, 3))
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }

    private var credentialsStep: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text(L.text("mobile.canvasImport.credentials.intro"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Text(L.text("mobile.canvasImport.credentials.tokenNotStored"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .accessibilityLabel(L.text("mobile.canvasImport.credentials.tokenNotStored"))

            fieldLabel("mobile.canvasImport.field.baseUrl")
            TextField(
                L.text("mobile.canvasImport.field.baseUrlPlaceholder"),
                text: $canvasBaseUrl
            )
            .textInputAutocapitalization(.never)
            .autocorrectionDisabled()
            .keyboardType(.URL)
            .textContentType(.URL)

            fieldLabel("mobile.canvasImport.field.token")
            SecureField(
                L.text("mobile.canvasImport.field.tokenPlaceholder"),
                text: $canvasToken
            )
            .textContentType(.password)
            .autocorrectionDisabled()
            .textInputAutocapitalization(.never)
        }
    }

    private var selectStep: some View {
        VStack(alignment: .leading, spacing: 14) {
            if courses.isEmpty {
                LMSEmptyState(
                    systemImage: "tray",
                    title: L.text("mobile.canvasImport.empty.title"),
                    message: L.text("mobile.canvasImport.empty.body")
                )
            } else {
                TextField(L.text("mobile.canvasImport.filter.placeholder"), text: $nameFilter)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                Toggle(isOn: $hideUnpublished) {
                    Text(L.text("mobile.canvasImport.filter.hideUnpublished"))
                }

                ForEach(filteredCourses) { course in
                    courseRow(course)
                }

                Divider().padding(.vertical, 4)

                Text(L.text("mobile.canvasImport.include.heading"))
                    .font(.headline)
                Text(L.text("mobile.canvasImport.include.piiNotice"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                ForEach(CanvasImportLogic.IncludeCategory.allCases) { category in
                    Toggle(isOn: includeBinding(category)) {
                        Text(L.text(String.LocalizationValue(category.labelKey)))
                    }
                }

                Divider().padding(.vertical, 4)

                Text(L.text("mobile.canvasImport.target.heading"))
                    .font(.headline)
                Picker(L.text("mobile.canvasImport.target.heading"), selection: $targetMode) {
                    ForEach(CanvasImportLogic.TargetMode.allCases) { mode in
                        Text(L.text(String.LocalizationValue(mode.titleKey))).tag(mode)
                    }
                }
                .pickerStyle(.segmented)
                .onChange(of: targetMode) { _, newValue in
                    importMode = CanvasImportLogic.defaultImportMode(for: newValue)
                }

                if targetMode == .existingCourse {
                    if existingCourses.isEmpty {
                        Text(L.text("mobile.canvasImport.target.noExisting"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    } else {
                        Picker(L.text("mobile.canvasImport.target.existingPicker"), selection: $existingCourseCode) {
                            Text(L.text("mobile.canvasImport.target.existingPicker")).tag("")
                            ForEach(existingCourses, id: \.courseCode) { course in
                                Text(course.title).tag(course.courseCode)
                            }
                        }
                        Picker(L.text("mobile.canvasImport.mode.heading"), selection: $importMode) {
                            ForEach(CourseImportExportLogic.ImportMode.allCases) { mode in
                                Text(L.text(String.LocalizationValue(CourseImportExportLogic.importModeTitleKey(mode))))
                                    .tag(mode)
                            }
                        }
                    }
                }

                Toggle(isOn: $enableGradeSync) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L.text("mobile.canvasImport.gradeSync.title"))
                        Text(L.text("mobile.canvasImport.gradeSync.summary"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
            }
        }
    }

    private var importingStep: some View {
        VStack(alignment: .leading, spacing: 14) {
            if importComplete {
                Label(L.text("mobile.canvasImport.success.title"), systemImage: "checkmark.circle.fill")
                    .font(.headline)
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                Text(L.text("mobile.canvasImport.success.body"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else if cancelled {
                Label(L.text("mobile.canvasImport.cancelled.title"), systemImage: "xmark.circle")
                    .font(.headline)
                Text(L.text("mobile.canvasImport.cancelled.body"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ProgressView()
                    .accessibilityLabel(L.text("mobile.canvasImport.progress.live"))
                Text(L.text("mobile.canvasImport.progress.live"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }

            VStack(alignment: .leading, spacing: 6) {
                ForEach(Array(progressLog.enumerated()), id: \.offset) { _, line in
                    Text(line)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel(progressLog.last ?? L.text("mobile.canvasImport.progress.live"))
            .accessibilityAddTraits(.updatesFrequently)
        }
    }

    @ViewBuilder
    private var bottomBar: some View {
        HStack(spacing: 12) {
            switch step {
            case .credentials:
                Button(L.text("mobile.canvasImport.action.connect")) {
                    Task { await connect() }
                }
                .buttonStyle(.borderedProminent)
                .disabled(busy)
            case .select:
                Button(L.text("mobile.canvasImport.action.back")) {
                    step = .credentials
                    errorMessage = nil
                }
                .disabled(busy)
                Spacer()
                Button(L.text("mobile.canvasImport.action.import")) {
                    Task { await startImport() }
                }
                .buttonStyle(.borderedProminent)
                .disabled(busy || selectedCourseId == nil || (targetMode == .existingCourse && existingCourseCode.isEmpty))
            case .importing:
                if busy && !importComplete {
                    Button(L.text("mobile.canvasImport.action.cancel"), role: .destructive) {
                        cancelRequested = true
                        cancelled = true
                        busy = false
                        CanvasImportObservability.recordCancelled()
                    }
                } else if importComplete {
                    Button(L.text("mobile.canvasImport.action.openCourse")) {
                        if let importedCourse {
                            onFinished(importedCourse)
                            dismiss()
                        }
                    }
                    .buttonStyle(.borderedProminent)
                } else {
                    Button(L.text("mobile.canvasImport.action.back")) {
                        step = .select
                        cancelled = false
                        progressLog = []
                        errorMessage = nil
                    }
                }
            }
        }
        .padding(16)
        .background(LexturesTheme.sceneBackground(for: colorScheme).opacity(0.95))
    }

    private func fieldLabel(_ key: String) -> some View {
        Text(L.text(String.LocalizationValue(key)))
            .font(.subheadline.weight(.semibold))
            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
    }

    private func courseRow(_ course: CanvasCourseListItem) -> some View {
        let selected = selectedCourseId == course.id
        return Button {
            selectedCourseId = course.id
        } label: {
            HStack(alignment: .top, spacing: 12) {
                Image(systemName: selected ? "checkmark.circle.fill" : "circle")
                    .foregroundStyle(selected ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.textSecondary(for: colorScheme))
                VStack(alignment: .leading, spacing: 2) {
                    Text(course.name)
                        .font(.headline)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    let subtitle = [course.courseCode, course.termName, course.workflowState]
                        .compactMap { $0?.trimmingCharacters(in: .whitespacesAndNewlines) }
                        .filter { !$0.isEmpty }
                        .joined(separator: " · ")
                    if !subtitle.isEmpty {
                        Text(subtitle)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
                Spacer(minLength: 0)
            }
            .padding(12)
            .background(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .fill(LexturesTheme.cardBackground(for: colorScheme))
            )
        }
        .buttonStyle(.plain)
    }

    private func includeBinding(_ category: CanvasImportLogic.IncludeCategory) -> Binding<Bool> {
        Binding(
            get: { include.value(for: category) },
            set: { include.set(category, $0) }
        )
    }

    @MainActor
    private func connect() async {
        errorMessage = nil
        if let key = CanvasImportLogic.validateCredentials(baseURL: canvasBaseUrl, accessToken: canvasToken) {
            errorMessage = L.text(String.LocalizationValue(key))
            return
        }
        guard let token = session.accessToken else {
            errorMessage = L.text("mobile.canvasImport.error.session")
            return
        }
        guard NetworkMonitor.shared.isOnline else {
            errorMessage = L.text("mobile.canvasImport.error.offline")
            return
        }
        busy = true
        defer { busy = false }
        do {
            // Token stays in @State memory only — never written to Keychain/UserDefaults.
            let listed = try await LMSAPI.fetchCanvasCourses(
                canvasBaseUrl: canvasBaseUrl,
                accessToken: canvasToken,
                sessionAccessToken: token
            )
            courses = listed
            selectedCourseId = listed.first?.id
            CanvasImportObservability.recordListed(courseCount: listed.count)
            step = .select
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.canvasImport.error.listFailed")
        }
    }

    @MainActor
    private func startImport() async {
        errorMessage = nil
        cancelRequested = false
        cancelled = false
        importComplete = false
        progressLog = []
        guard let selectedId = selectedCourseId else {
            errorMessage = L.text("mobile.canvasImport.error.selectCourse")
            return
        }
        guard let selected = courses.first(where: { $0.id == selectedId }) else {
            errorMessage = L.text("mobile.canvasImport.error.selectCourse")
            return
        }
        guard let sessionToken = session.accessToken else {
            errorMessage = L.text("mobile.canvasImport.error.session")
            return
        }
        guard NetworkMonitor.shared.isOnline else {
            errorMessage = L.text("mobile.canvasImport.error.offline")
            return
        }

        busy = true
        step = .importing
        CanvasImportObservability.recordStarted(include: include)

        do {
            let targetCode: String
            if targetMode == .newCourse {
                appendProgress(L.text("mobile.canvasImport.progress.creatingShell"))
                let body = CreateCourseRequest(
                    title: selected.name,
                    description: "",
                    courseType: CourseCreateLogic.CourseMode.traditional.rawValue,
                    termId: nil,
                    gradeLevel: nil
                )
                let created = try await LMSAPI.createCourse(body: body, accessToken: sessionToken)
                targetCode = created.courseCode
            } else {
                targetCode = existingCourseCode
            }

            if cancelRequested {
                throw CanvasImportLogic.CanvasImportError.cancelled
            }

            let request = PostCourseImportCanvasRequest(
                mode: importMode,
                canvasBaseUrl: CanvasImportLogic.normalizeBaseURL(canvasBaseUrl),
                canvasCourseId: String(selected.id),
                accessToken: canvasToken.trimmingCharacters(in: .whitespacesAndNewlines),
                include: include,
                canvasGradeSyncEnabled: enableGradeSync
            )
            // Clear local token reference after the request body is built — request holds the only copy for the HTTP call.
            // The SecureField binding still holds in-memory UI state until dismiss; never persisted.
            try await LMSAPI.postCourseImportCanvas(
                courseCode: targetCode,
                body: request,
                sessionAccessToken: sessionToken,
                onProgress: { message in
                    appendProgress(message)
                    CanvasImportObservability.recordProgress()
                },
                isCancelled: { cancelRequested }
            )

            let summary = try await LMSAPI.fetchCourse(courseCode: targetCode, accessToken: sessionToken)
            importedCourse = summary
            importComplete = true
            busy = false
            CanvasImportObservability.recordSucceeded(include: include)
            appendProgress(L.text("mobile.canvasImport.progress.done"))
            // Auto-open after a short beat for success recognition.
            try? await Task.sleep(for: .milliseconds(600))
            onFinished(summary)
            dismiss()
        } catch {
            busy = false
            if CanvasImportLogic.isCancelledError(error) || cancelRequested {
                cancelled = true
                appendProgress(CanvasImportLogic.cancelledMessage)
                CanvasImportObservability.recordCancelled()
            } else {
                CanvasImportObservability.recordFailed()
                errorMessage = (error as? LocalizedError)?.errorDescription
                    ?? L.text("mobile.canvasImport.error.importFailed")
            }
        }
    }

    private func appendProgress(_ message: String) {
        let trimmed = message.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        progressLog.append(trimmed)
    }
}
