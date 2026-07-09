import SwiftUI

/// Course locale variants, glossary, and coverage (M13.9).
struct CourseTranslationsSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var locales: [TranslationCoverage] = []
    @State private var trackedLocales: [String] = []
    @State private var loading = true
    @State private var loadError: String?
    @State private var actionError: String?
    @State private var actionSuccess: String?
    @State private var cacheLabel: String?
    @State private var selectedLocale: String?
    @State private var showAddLocale = false

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }

    var body: some View {
        Group {
            if let selectedLocale {
                CourseTranslationLocaleDetailView(
                    course: course,
                    targetLocale: selectedLocale,
                    onBack: { self.selectedLocale = nil; Task { await reload(force: true) } }
                )
            } else {
                localesList
            }
        }
        .task(id: course.courseCode) { await reload(force: false) }
        .confirmationDialog(
            L.text("mobile.courseSettings.translations.addLocaleTitle"),
            isPresented: $showAddLocale,
            titleVisibility: .visible
        ) {
            ForEach(CourseTranslationsLogic.availableLocalesToAdd(existing: locales), id: \.tag) { option in
                Button(L.text(String.LocalizationValue(option.labelKey))) {
                    Task { await addLocale(option.tag) }
                }
            }
            Button(L.text("mobile.courseSettings.translations.cancel"), role: .cancel) {}
        } message: {
            Text(L.text("mobile.courseSettings.translations.addLocaleConfirm"))
        }
    }

    private var localesList: some View {
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
                        if locales.isEmpty {
                            emptyLocalesCard
                        } else {
                            localesCard
                        }

                        Button {
                            showAddLocale = true
                        } label: {
                            Label(
                                L.text("mobile.courseSettings.translations.addLocale"),
                                systemImage: "plus.circle.fill"
                            )
                            .font(.subheadline.weight(.semibold))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 12)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(LexturesTheme.brandTeal)
                        .disabled(CourseTranslationsLogic.availableLocalesToAdd(existing: locales).isEmpty)
                        .accessibilityLabel(L.text("mobile.courseSettings.translations.addLocale"))
                    }
                }
                .padding(16)
            }
        }
    }

    private var introCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.courseSettings.translations.introTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.translations.introDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private var emptyLocalesCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.courseSettings.translations.emptyTitle"))
                    .font(.subheadline.weight(.semibold))
                Text(L.text("mobile.courseSettings.translations.emptyMessage"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private var localesCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(L.text("mobile.courseSettings.translations.localesTitle"))
                    .font(.headline)

                ForEach(locales) { locale in
                    Button {
                        selectedLocale = locale.targetLocale
                    } label: {
                        HStack(spacing: 12) {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(CourseTranslationsLogic.localeDisplayName(for: locale.targetLocale))
                                    .font(.subheadline.weight(.medium))
                                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                Text(locale.targetLocale)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                Text(CourseTranslationsLogic.formatCoverageLabel(
                                    translated: locale.translatedItems,
                                    total: locale.totalItems,
                                    locale: locale.targetLocale
                                ))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            Spacer(minLength: 8)
                            Text(CourseTranslationsLogic.formatCoveragePercentOnly(percent: locale.percent))
                                .font(.subheadline.weight(.bold))
                                .foregroundStyle(LexturesTheme.brandTeal)
                            Image(systemName: "chevron.right")
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.vertical, 6)
                        .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel(CourseTranslationsLogic.formatCoverageLabel(
                        translated: locale.translatedItems,
                        total: locale.totalItems,
                        locale: locale.targetLocale
                    ))
                    .accessibilityHint(L.text("mobile.courseSettings.translations.openLocaleHint"))

                    if locale.id != locales.last?.id {
                        Divider()
                    }
                }
            }
        }
    }

    private func addLocale(_ tag: String) async {
        actionError = nil
        actionSuccess = nil
        trackedLocales = CourseTranslationsLogic.trackLocale(tag, into: trackedLocales)
        CourseTranslationsLogic.saveTrackedLocales(trackedLocales, courseCode: course.courseCode)
        locales = CourseTranslationsLogic.mergeLocales(server: locales, tracked: trackedLocales)
        selectedLocale = tag
        actionSuccess = L.format(
            "mobile.courseSettings.translations.localeAdded",
            CourseTranslationsLogic.localeDisplayName(for: tag)
        )
    }

    private func reload(force: Bool) async {
        guard let token = session.accessToken else { return }
        if !force && !loading && loadError == nil && !locales.isEmpty { return }
        loading = locales.isEmpty && loadError == nil
        loadError = nil
        defer { loading = false }

        trackedLocales = CourseTranslationsLogic.mergeTracked(
            cached: CourseTranslationsLogic.loadTrackedLocales(courseCode: course.courseCode),
            current: trackedLocales
        )

        do {
            let result = try await offline.cachedFetch(
                key: CourseTranslationsLogic.cacheKeyLocales(courseCode: course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchTranslationLocales(
                    courseCode: course.courseCode,
                    accessToken: token
                )
            }
            let serverLocales = result.value
            var enriched = serverLocales
            if isOnline {
                for tag in trackedLocales where !serverLocales.contains(where: {
                    CourseTranslationsLogic.normalizeLocaleTag($0.targetLocale) == tag
                }) {
                    if let cov = try? await LMSAPI.fetchTranslationCoverage(
                        courseCode: course.courseCode,
                        targetLocale: tag,
                        accessToken: token
                    ) {
                        enriched.append(cov)
                    }
                }
            }
            locales = CourseTranslationsLogic.mergeLocales(
                server: enriched,
                tracked: trackedLocales
            )
            if let cached = result.cached, cached.isStale(isOnline: isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            loadError = error.localizedDescription
        }
    }
}

// MARK: - Locale detail

private struct CourseTranslationLocaleDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let targetLocale: String
    var onBack: () -> Void

    @State private var coverage: TranslationCoverage?
    @State private var items: [CourseTranslationListItem] = []
    @State private var glossary: [CourseGlossaryEntry] = []
    @State private var itemQuery = ""
    @State private var glossaryQuery = ""
    @State private var itemPage = 0
    @State private var glossaryPage = 0
    @State private var loading = true
    @State private var loadError: String?
    @State private var actionError: String?
    @State private var actionSuccess: String?
    @State private var cacheLabel: String?
    @State private var glossaryDraft: CourseTranslationsLogic.GlossaryDraft?
    @State private var savingGlossary = false

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var filteredItems: [CourseTranslationListItem] {
        CourseTranslationsLogic.filterItems(items, query: itemQuery)
    }
    private var filteredGlossary: [CourseGlossaryEntry] {
        CourseTranslationsLogic.filterGlossary(glossary, query: glossaryQuery)
    }
    private var visibleItems: [CourseTranslationListItem] {
        CourseTranslationsLogic.paginatedItems(filteredItems, page: itemPage)
    }
    private var visibleGlossary: [CourseGlossaryEntry] {
        CourseTranslationsLogic.paginatedGlossary(filteredGlossary, page: glossaryPage)
    }

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Button {
                    onBack()
                } label: {
                    Label(
                        L.text("mobile.courseSettings.translations.backToLocales"),
                        systemImage: "chevron.left"
                    )
                    .font(.subheadline.weight(.semibold))
                }
                .buttonStyle(.plain)
                Spacer()
            }
            .padding(.horizontal, 16)
            .padding(.top, 12)

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

                        headerCard
                        if let coverage {
                            coverageCard(coverage)
                        }
                        itemsCard
                        glossaryCard
                    }
                }
                .padding(16)
            }
        }
        .task(id: "\(course.courseCode):\(targetLocale)") { await reload(force: true) }
        .sheet(item: Binding(
            get: { glossaryDraft.map { GlossarySheetItem(draft: $0) } },
            set: { glossaryDraft = $0?.draft }
        )) { item in
            NavigationStack {
                GlossaryEditorSheet(
                    draft: item.draft,
                    targetLocale: targetLocale,
                    isSaving: savingGlossary,
                    onCancel: { glossaryDraft = nil },
                    onSave: { updated in
                        Task { await saveGlossary(updated) }
                    }
                )
            }
            .presentationDetents([.medium])
        }
    }

    private var headerCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 6) {
                Text(CourseTranslationsLogic.localeDisplayName(for: targetLocale))
                    .font(.headline)
                Text(targetLocale)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(L.text("mobile.courseSettings.translations.localeDetailHint"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .environment(\.layoutDirection, CourseTranslationsLogic.isRTLLocale(targetLocale)
                ? .rightToLeft
                : .leftToRight)
        }
    }

    private func coverageCard(_ coverage: TranslationCoverage) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.translations.coverageTitle"))
                    .font(.headline)
                HStack(spacing: 16) {
                    ZStack {
                        Circle()
                            .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 6)
                        Circle()
                            .trim(from: 0, to: CGFloat(coverage.percent) / 100)
                            .stroke(LexturesTheme.brandTeal, style: StrokeStyle(lineWidth: 6, lineCap: .round))
                            .rotationEffect(.degrees(-90))
                        Text("\(coverage.percentInt)%")
                            .font(.title3.weight(.bold))
                    }
                    .frame(width: 72, height: 72)
                    .accessibilityLabel(CourseTranslationsLogic.formatCoverageLabel(
                        translated: coverage.translatedItems,
                        total: coverage.totalItems,
                        locale: coverage.targetLocale
                    ))

                    VStack(alignment: .leading, spacing: 4) {
                        Text(CourseTranslationsLogic.formatCoverageLabel(
                            translated: coverage.translatedItems,
                            total: coverage.totalItems,
                            locale: coverage.targetLocale
                        ))
                        .font(.subheadline.weight(.semibold))
                        Text(L.format(
                            "mobile.courseSettings.translations.unpublishedCount",
                            CourseTranslationsLogic.unpublishedCount(items)
                        ))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
            }
        }
    }

    private var itemsCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(L.text("mobile.courseSettings.translations.itemsTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.translations.itemsReadOnlyHint"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                TextField(
                    L.text("mobile.courseSettings.translations.searchItems"),
                    text: $itemQuery
                )
                .textFieldStyle(.roundedBorder)
                .accessibilityLabel(L.text("mobile.courseSettings.translations.searchItems"))
                .onChange(of: itemQuery) { _, _ in itemPage = 0 }

                if visibleItems.isEmpty {
                    Text(L.text("mobile.courseSettings.translations.itemsEmpty"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(visibleItems) { item in
                        HStack(alignment: .top, spacing: 10) {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(item.title.isEmpty
                                    ? L.text("mobile.courseSettings.translations.untitled")
                                    : item.title)
                                    .font(.subheadline.weight(.medium))
                                    .multilineTextAlignment(.leading)
                                Text(L.text(String.LocalizationValue(
                                    CourseTranslationsLogic.statusLabelKey(for: item)
                                )))
                                .font(.caption)
                                .foregroundStyle(statusColor(for: item))
                            }
                            Spacer(minLength: 8)
                        }
                        .padding(.vertical, 4)
                        if item.id != visibleItems.last?.id {
                            Divider()
                        }
                    }
                    if CourseTranslationsLogic.hasMoreItemPages(items: filteredItems, page: itemPage) {
                        Button(L.text("mobile.courseSettings.translations.loadMore")) {
                            itemPage += 1
                        }
                        .font(.subheadline.weight(.semibold))
                    }
                }
            }
        }
    }

    private var glossaryCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                HStack {
                    Text(L.text("mobile.courseSettings.translations.glossaryTitle"))
                        .font(.headline)
                    Spacer()
                    Button {
                        glossaryDraft = CourseTranslationsLogic.GlossaryDraft()
                    } label: {
                        Label(
                            L.text("mobile.courseSettings.translations.addTerm"),
                            systemImage: "plus"
                        )
                        .font(.subheadline.weight(.semibold))
                    }
                    .buttonStyle(.bordered)
                    .accessibilityLabel(L.text("mobile.courseSettings.translations.addTerm"))
                }

                TextField(
                    L.text("mobile.courseSettings.translations.searchGlossary"),
                    text: $glossaryQuery
                )
                .textFieldStyle(.roundedBorder)
                .accessibilityLabel(L.text("mobile.courseSettings.translations.searchGlossary"))
                .onChange(of: glossaryQuery) { _, _ in glossaryPage = 0 }

                if visibleGlossary.isEmpty {
                    Text(L.text("mobile.courseSettings.translations.glossaryEmpty"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(visibleGlossary) { entry in
                        Button {
                            glossaryDraft = CourseTranslationsLogic.draft(from: entry)
                        } label: {
                            HStack(alignment: .top, spacing: 10) {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(entry.sourceTerm)
                                        .font(.subheadline.weight(.medium))
                                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                    Text(entry.targetTerm)
                                        .font(.caption)
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                        .environment(\.layoutDirection,
                                            CourseTranslationsLogic.isRTLLocale(targetLocale)
                                                ? .rightToLeft
                                                : .leftToRight)
                                }
                                Spacer(minLength: 8)
                                Image(systemName: "pencil")
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.vertical, 4)
                        }
                        .buttonStyle(.plain)
                        .accessibilityLabel(
                            L.format(
                                "mobile.courseSettings.translations.glossaryTermA11y",
                                entry.sourceTerm,
                                entry.targetTerm
                            )
                        )
                        if entry.id != visibleGlossary.last?.id {
                            Divider()
                        }
                    }
                    if CourseTranslationsLogic.hasMoreGlossaryPages(
                        entries: filteredGlossary,
                        page: glossaryPage
                    ) {
                        Button(L.text("mobile.courseSettings.translations.loadMore")) {
                            glossaryPage += 1
                        }
                        .font(.subheadline.weight(.semibold))
                    }
                }
            }
        }
    }

    private func statusColor(for item: CourseTranslationListItem) -> Color {
        if item.hasPublished == true { return LexturesTheme.brandTeal }
        if item.hasDraft == true || item.isDraft == true { return .orange }
        return LexturesTheme.textSecondary(for: colorScheme)
    }

    private func reload(force: Bool) async {
        guard let token = session.accessToken else { return }
        loading = coverage == nil && loadError == nil
        loadError = nil
        defer { loading = false }

        do {
            let listResult = try await offline.cachedFetch(
                key: CourseTranslationsLogic.cacheKeyLocaleDetail(
                    courseCode: course.courseCode,
                    locale: targetLocale
                ),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourseTranslations(
                    courseCode: course.courseCode,
                    targetLocale: targetLocale,
                    accessToken: token
                )
            }
            let glossaryResult = try await offline.cachedFetch(
                key: CourseTranslationsLogic.cacheKeyGlossary(
                    courseCode: course.courseCode,
                    locale: targetLocale
                ),
                accessToken: token
            ) {
                let entries = try await LMSAPI.fetchCourseGlossary(
                    courseCode: course.courseCode,
                    targetLocale: targetLocale,
                    accessToken: token
                )
                return CourseGlossaryListResponse(entries: entries)
            }
            items = listResult.value.items
            coverage = listResult.value.coverage
            glossary = glossaryResult.value.entries
            itemPage = 0
            glossaryPage = 0
            if let cached = listResult.cached, cached.isStale(isOnline: isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            loadError = error.localizedDescription
        }
    }

    private func saveGlossary(_ draft: CourseTranslationsLogic.GlossaryDraft) async {
        guard let token = session.accessToken else { return }
        actionError = nil
        actionSuccess = nil

        switch CourseTranslationsLogic.validateGlossaryDraft(draft) {
        case .sourceRequired:
            actionError = L.text("mobile.courseSettings.translations.validation.sourceRequired")
            return
        case .targetRequired:
            actionError = L.text("mobile.courseSettings.translations.validation.targetRequired")
            return
        case .ok:
            break
        }

        savingGlossary = true
        defer { savingGlossary = false }

        do {
            let body = CourseTranslationsLogic.buildGlossaryBody(
                draft: draft,
                targetLocale: targetLocale
            )
            _ = try await offline.enqueueMutation(
                method: "POST",
                path: CourseTranslationsLogic.glossaryPath(courseCode: course.courseCode),
                body: body,
                label: L.text("mobile.courseSettings.translations.glossarySaveLabel"),
                accessToken: token,
                idempotencyKey: CourseTranslationsLogic.glossaryIdempotencyKey(
                    courseCode: course.courseCode,
                    locale: targetLocale,
                    sourceTerm: body.sourceTerm
                )
            )
            // Optimistic local update; refetch when online.
            let optimistic = CourseGlossaryEntry(
                id: draft.id ?? UUID().uuidString,
                sourceTerm: body.sourceTerm,
                targetTerm: body.targetTerm,
                sourceLocale: body.sourceLocale,
                targetLocale: body.targetLocale
            )
            glossary = CourseTranslationsLogic.upsertGlossaryEntry(optimistic, into: glossary)
            if isOnline, let refreshed = try? await LMSAPI.fetchCourseGlossary(
                courseCode: course.courseCode,
                targetLocale: targetLocale,
                accessToken: token
            ) {
                glossary = refreshed
            }
            glossaryDraft = nil
            actionSuccess = L.text("mobile.courseSettings.translations.glossarySaved")
        } catch {
            actionError = error.localizedDescription
        }
    }
}

// MARK: - Glossary sheet

private struct GlossarySheetItem: Identifiable {
    var draft: CourseTranslationsLogic.GlossaryDraft
    var id: String { draft.id ?? "new" }
}

private struct GlossaryEditorSheet: View {
    @Environment(\.colorScheme) private var colorScheme

    let draft: CourseTranslationsLogic.GlossaryDraft
    let targetLocale: String
    let isSaving: Bool
    var onCancel: () -> Void
    var onSave: (CourseTranslationsLogic.GlossaryDraft) -> Void

    @State private var sourceTerm: String = ""
    @State private var targetTerm: String = ""

    var body: some View {
        Form {
            Section {
                TextField(
                    L.text("mobile.courseSettings.translations.sourceTerm"),
                    text: $sourceTerm
                )
                .accessibilityLabel(L.text("mobile.courseSettings.translations.sourceTerm"))
                TextField(
                    L.text("mobile.courseSettings.translations.targetTerm"),
                    text: $targetTerm
                )
                .accessibilityLabel(L.text("mobile.courseSettings.translations.targetTerm"))
                .environment(\.layoutDirection, CourseTranslationsLogic.isRTLLocale(targetLocale)
                    ? .rightToLeft
                    : .leftToRight)
            } footer: {
                Text(L.text("mobile.courseSettings.translations.glossaryEditorHint"))
                    .font(.caption)
            }
        }
        .navigationTitle(
            draft.isEditing
                ? L.text("mobile.courseSettings.translations.editTerm")
                : L.text("mobile.courseSettings.translations.addTerm")
        )
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button(L.text("mobile.courseSettings.translations.cancel"), action: onCancel)
            }
            ToolbarItem(placement: .confirmationAction) {
                if isSaving {
                    ProgressView()
                } else {
                    Button(L.text("mobile.courseSettings.translations.saveTerm")) {
                        onSave(CourseTranslationsLogic.GlossaryDraft(
                            id: draft.id,
                            sourceTerm: sourceTerm,
                            targetTerm: targetTerm
                        ))
                    }
                    .disabled(
                        sourceTerm.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                            || targetTerm.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    )
                }
            }
        }
        .onAppear {
            sourceTerm = draft.sourceTerm
            targetTerm = draft.targetTerm
        }
    }
}

