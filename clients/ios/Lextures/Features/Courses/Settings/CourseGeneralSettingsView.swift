import PhotosUI
import SwiftUI

/// General course settings form with unsaved-changes banner and offline-queued saves (M13.1).
struct CourseGeneralSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    var onCourseUpdated: (CourseSummary) -> Void

    @State private var serverCourse: CourseSummary
    @State private var form = CourseGeneralFormState()
    @State private var structureItems: [CourseStructureItem] = []
    @State private var cacheLabel: String?
    @State private var loadError: String?
    @State private var loading = true
    @State private var saveStatus: CourseSettingsLogic.SaveStatus = .idle
    @State private var validationError: CourseSettingsLogic.ValidationError?
    @State private var showHeroEditor = false
    @State private var showTimezonePicker = false
    @State private var showContentPagePicker = false

    init(course: CourseSummary, onCourseUpdated: @escaping (CourseSummary) -> Void) {
        self.course = course
        self.onCourseUpdated = onCourseUpdated
        _serverCourse = State(initialValue: course)
    }

    private var isDirty: Bool {
        CourseSettingsLogic.isGeneralFormDirty(form: form, course: serverCourse)
    }

    private var contentPages: [CourseStructureItem] {
        CourseSettingsLogic.contentPages(from: structureItems)
    }

    var body: some View {
        ZStack(alignment: .bottom) {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if !NetworkMonitor.shared.isOnline {
                        OfflineBanner()
                    }
                    if let loadError {
                        LMSErrorBanner(message: loadError)
                    }
                    if let cacheLabel {
                        StalenessChip(label: cacheLabel)
                    }
                    if loading && serverCourse.title.isEmpty {
                        LMSSkeletonList(count: 4)
                    } else {
                        basicInfoCard
                        timezoneCard
                        publishingCard
                        courseHomeCard
                        scheduleCard
                        heroCard
                        themeCard
                    }
                }
                .padding(16)
                .padding(.bottom, isDirty ? 88 : 16)
            }
            .refreshable { await load(force: true) }

            if isDirty {
                UnsavedChangesBanner(
                    isSaving: {
                        if case .saving = saveStatus { return true }
                        return false
                    }(),
                    onSave: { Task { await saveChanges() } },
                    onDiscard: discardChanges
                )
            }
        }
        .sheet(isPresented: $showHeroEditor) {
            CourseHeroImageEditor(
                course: serverCourse,
                onSaved: { updated in
                    applyUpdatedCourse(updated)
                }
            )
        }
        .sheet(isPresented: $showTimezonePicker) {
            TimezonePickerSheet(selection: $form.courseTimezone)
        }
        .sheet(isPresented: $showContentPagePicker) {
            ContentPagePickerSheet(
                pages: contentPages,
                selection: $form.courseHomeContentItemId
            )
        }
        .task { await load(force: false) }
    }

    private var basicInfoCard: some View {
        LMSCard {
            Text(L.text("mobile.courseSettings.basicInfo"))
                .font(LexturesTheme.displayFont(17))

            labeledField(L.text("mobile.courseSettings.title")) {
                TextField(L.text("mobile.courseSettings.titlePlaceholder"), text: $form.title)
                    .textFieldStyle(.roundedBorder)
            }
            if let titleError = validationError?.title {
                Text(titleError).font(.caption).foregroundStyle(.red)
            }

            labeledField(L.text("mobile.courseSettings.description")) {
                TextField(L.text("mobile.courseSettings.descriptionPlaceholder"), text: $form.description, axis: .vertical)
                    .lineLimit(3...6)
                    .textFieldStyle(.roundedBorder)
            }

            labeledField(L.text("mobile.courseSettings.gradeLevel")) {
                Picker("", selection: $form.gradeLevel) {
                    ForEach(CourseSettingsLogic.gradeLevels, id: \.self) { level in
                        Text(CourseSettingsLogic.gradeLevelLabel(level)).tag(level)
                    }
                }
                .pickerStyle(.menu)
            }
        }
    }

    private var timezoneCard: some View {
        LMSCard {
            Text(L.text("mobile.courseSettings.timezone"))
                .font(LexturesTheme.displayFont(17))
            Button {
                showTimezonePicker = true
            } label: {
                HStack {
                    Text(form.courseTimezone)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Spacer()
                    Image(systemName: "chevron.right")
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            .buttonStyle(.plain)
        }
    }

    private var publishingCard: some View {
        LMSCard {
            Toggle(isOn: $form.published) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(L.text("mobile.courseSettings.published"))
                        .font(.subheadline.weight(.semibold))
                    Text(L.text("mobile.courseSettings.publishedHint"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    private var courseHomeCard: some View {
        LMSCard {
            Text(L.text("mobile.courseSettings.courseHome"))
                .font(LexturesTheme.displayFont(17))

            Picker("", selection: $form.courseHomeLanding) {
                Text(L.text("mobile.courseSettings.home.dashboard")).tag(CourseSettingsLogic.CourseHomeLanding.data)
                Text(L.text("mobile.courseSettings.home.calendar")).tag(CourseSettingsLogic.CourseHomeLanding.calendar)
                Text(L.text("mobile.courseSettings.home.contentPage")).tag(CourseSettingsLogic.CourseHomeLanding.content_page)
            }
            .pickerStyle(.segmented)
            .onChange(of: form.courseHomeLanding) { _, landing in
                if landing != .content_page {
                    form.courseHomeContentItemId = ""
                }
            }

            if form.courseHomeLanding == .content_page {
                Button {
                    showContentPagePicker = true
                } label: {
                    HStack {
                        Text(selectedContentPageTitle)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Spacer()
                        Image(systemName: "chevron.right")
                    }
                }
                .buttonStyle(.plain)
                if let homeError = validationError?.courseHome {
                    Text(homeError).font(.caption).foregroundStyle(.red)
                }
            }
        }
    }

    private var selectedContentPageTitle: String {
        let id = form.courseHomeContentItemId
        if id.isEmpty { return L.text("mobile.courseSettings.chooseContentPage") }
        return contentPages.first(where: { $0.id == id })?.title ?? id
    }

    private var scheduleCard: some View {
        LMSCard {
            Text(L.text("mobile.courseSettings.schedule"))
                .font(LexturesTheme.displayFont(17))

            Picker("", selection: $form.scheduleMode) {
                Text(L.text("mobile.courseSettings.schedule.fixed")).tag(CourseSettingsLogic.ScheduleMode.fixed)
                Text(L.text("mobile.courseSettings.schedule.relative")).tag(CourseSettingsLogic.ScheduleMode.relative)
            }
            .pickerStyle(.segmented)

            if form.scheduleMode == .fixed {
                dateField(L.text("mobile.courseSettings.startsAt"), text: $form.startsAt)
                dateField(L.text("mobile.courseSettings.endsAt"), text: $form.endsAt)
                dateField(L.text("mobile.courseSettings.visibleFrom"), text: $form.visibleFrom)
                dateField(L.text("mobile.courseSettings.hiddenAt"), text: $form.hiddenAt)
            } else {
                relativeDurationField(
                    title: L.text("mobile.courseSettings.relativeEnd"),
                    amount: $form.relEndAmount,
                    unit: $form.relEndUnit
                )
                relativeDurationField(
                    title: L.text("mobile.courseSettings.relativeHidden"),
                    amount: $form.relHiddenAmount,
                    unit: $form.relHiddenUnit
                )
            }
        }
    }

    private var heroCard: some View {
        LMSCard {
            Text(L.text("mobile.courseSettings.heroImage"))
                .font(LexturesTheme.displayFont(17))

            if let urlString = serverCourse.heroImageUrl, let url = URL(string: urlString) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image.resizable().scaledToFill()
                    default:
                        Rectangle().fill(LexturesTheme.cardBackground(for: colorScheme))
                    }
                }
                .frame(height: 120)
                .clipShape(RoundedRectangle(cornerRadius: 12))
            }

            Button(L.text("mobile.courseSettings.editHeroImage")) {
                showHeroEditor = true
            }
            .buttonStyle(.bordered)
        }
    }

    private var themeCard: some View {
        LMSCard {
            Text(L.text("mobile.courseSettings.readingTheme"))
                .font(LexturesTheme.displayFont(17))

            LazyVGrid(columns: [GridItem(.adaptive(minimum: 88), spacing: 8)], spacing: 8) {
                ForEach(CourseSettingsLogic.markdownThemePresets.filter { $0 != "custom" }, id: \.self) { preset in
                    Button {
                        form.markdownThemePreset = preset
                    } label: {
                        Text(preset.capitalized)
                            .font(.caption.weight(.semibold))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 10)
                            .background(
                                form.markdownThemePreset == preset
                                    ? LexturesTheme.brandTeal.opacity(0.18)
                                    : LexturesTheme.cardBackground(for: colorScheme),
                                in: RoundedRectangle(cornerRadius: 8)
                            )
                    }
                    .buttonStyle(.plain)
                }
            }

            if form.markdownThemePreset == "custom" {
                customThemeFields
            }

            Button(L.text("mobile.courseSettings.customizeTheme")) {
                form.markdownThemePreset = "custom"
            }
            .font(.caption)
        }
    }

    private var customThemeFields: some View {
        VStack(alignment: .leading, spacing: 8) {
            colorField(L.text("mobile.courseSettings.theme.heading"), value: binding(\.headingColor))
            colorField(L.text("mobile.courseSettings.theme.body"), value: binding(\.bodyColor))
            colorField(L.text("mobile.courseSettings.theme.link"), value: binding(\.linkColor))
            Picker(L.text("mobile.courseSettings.theme.width"), selection: binding(\.articleWidth)) {
                ForEach(CourseSettingsLogic.articleWidths, id: \.self) { width in
                    Text(width.capitalized).tag(width as String?)
                }
            }
            Picker(L.text("mobile.courseSettings.theme.font"), selection: binding(\.fontFamily)) {
                ForEach(CourseSettingsLogic.fontFamilies, id: \.self) { font in
                    Text(font.capitalized).tag(font as String?)
                }
            }
        }
    }

    private func labeledField<Content: View>(_ label: String, @ViewBuilder content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(label).font(.subheadline.weight(.semibold))
            content()
        }
    }

    private func dateField(_ label: String, text: Binding<String>) -> some View {
        labeledField(label) {
            TextField("YYYY-MM-DDTHH:mm", text: text)
                .textFieldStyle(.roundedBorder)
                .keyboardType(.numbersAndPunctuation)
        }
    }

    private func relativeDurationField(
        title: String,
        amount: Binding<String>,
        unit: Binding<CourseSettingsLogic.RelativeDurationUnit>
    ) -> some View {
        labeledField(title) {
            HStack {
                TextField("1", text: amount)
                    .textFieldStyle(.roundedBorder)
                    .keyboardType(.numberPad)
                Picker("", selection: unit) {
                    ForEach(CourseSettingsLogic.RelativeDurationUnit.allCases, id: \.self) { value in
                        Text(value.rawValue).tag(value)
                    }
                }
                .pickerStyle(.menu)
            }
        }
    }

    private func colorField(_ label: String, value: Binding<String>) -> some View {
        labeledField(label) {
            TextField("#000000", text: value)
                .textFieldStyle(.roundedBorder)
                .textInputAutocapitalization(.never)
        }
    }

    private func binding(_ keyPath: WritableKeyPath<MarkdownThemeCustom, String?>) -> Binding<String> {
        Binding(
            get: { form.customDraft[keyPath: keyPath] ?? "" },
            set: { form.customDraft[keyPath: keyPath] = $0.nilIfEmpty }
        )
    }

    private func discardChanges() {
        form = CourseSettingsLogic.applyCourseToForm(serverCourse)
        validationError = nil
        saveStatus = .idle
    }

    private func applyUpdatedCourse(_ updated: CourseSummary) {
        serverCourse = updated
        form = CourseSettingsLogic.applyCourseToForm(updated)
        onCourseUpdated(updated)
    }

    private func load(force: Bool) async {
        guard let token = session.accessToken else { return }
        loading = true
        loadError = nil
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: CourseSettingsLogic.cacheKeySettings(courseCode: course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)
            }
            applyUpdatedCourse(result.value)
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
            structureItems = (try? await LMSAPI.fetchCourseStructure(courseCode: course.courseCode, accessToken: token)) ?? []
        } catch {
            loadError = error.localizedDescription
        }
    }

    private func saveChanges() async {
        validationError = CourseSettingsLogic.validateGeneralForm(
            title: form.title,
            courseHomeLanding: form.courseHomeLanding,
            courseHomeContentItemId: form.courseHomeContentItemId
        )
        if validationError != nil { return }

        guard let token = session.accessToken else { return }
        saveStatus = .saving
        defer {
            if case .saving = saveStatus { saveStatus = .idle }
        }

        do {
            var latest = serverCourse
            if CourseSettingsLogic.courseNeedsUpdate(form: form, course: serverCourse) {
                let body = CourseSettingsLogic.buildCourseUpdateRequest(form: form)
                _ = try await offline.enqueueMutation(
                    method: "PUT",
                    path: "/api/v1/courses/\(course.courseCode)",
                    body: body,
                    label: L.text("mobile.courseSettings.saveLabel"),
                    accessToken: token,
                    idempotencyKey: "course-settings:\(course.courseCode):general"
                )
                latest = try await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)
            }
            if CourseSettingsLogic.themeNeedsUpdate(form: form, course: serverCourse) {
                let patch = CourseSettingsLogic.buildMarkdownThemePatch(form: form)
                _ = try await offline.enqueueMutation(
                    method: "PATCH",
                    path: "/api/v1/courses/\(course.courseCode)/markdown-theme",
                    body: patch,
                    label: L.text("mobile.courseSettings.saveThemeLabel"),
                    accessToken: token,
                    idempotencyKey: "course-settings:\(course.courseCode):theme"
                )
                latest = try await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)
            }
            applyUpdatedCourse(latest)
            saveStatus = .saved
        } catch {
            saveStatus = .error(error.localizedDescription)
        }
    }
}

private struct TimezonePickerSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Binding var selection: String
    @State private var query = ""

    private var filtered: [String] {
        let q = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let all = CourseSettingsLogic.timezoneOptions()
        guard !q.isEmpty else { return all.prefix(100).map { $0 } }
        return all.filter { $0.lowercased().contains(q) }.prefix(100).map { $0 }
    }

    var body: some View {
        NavigationStack {
            List(filtered, id: \.self) { tz in
                Button(tz) { selection = tz; dismiss() }
            }
            .searchable(text: $query, prompt: Text(L.text("mobile.courseSettings.searchTimezone")))
            .navigationTitle(L.text("mobile.courseSettings.timezone"))
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) { dismiss() }
                }
            }
        }
    }
}

private struct ContentPagePickerSheet: View {
    @Environment(\.dismiss) private var dismiss
    let pages: [CourseStructureItem]
    @Binding var selection: String
    @State private var query = ""

    private var filtered: [CourseStructureItem] {
        let q = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !q.isEmpty else { return pages }
        return pages.filter { $0.title.lowercased().contains(q) }
    }

    var body: some View {
        NavigationStack {
            List(filtered) { page in
                Button(page.title) { selection = page.id; dismiss() }
            }
            .searchable(text: $query, prompt: Text(L.text("mobile.courseSettings.searchPages")))
            .navigationTitle(L.text("mobile.courseSettings.chooseContentPage"))
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) { dismiss() }
                }
            }
        }
    }
}

private extension String {
    var nilIfEmpty: String? {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}
