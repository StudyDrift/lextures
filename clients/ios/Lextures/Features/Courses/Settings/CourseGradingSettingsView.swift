import SwiftUI

/// Grading scale, weighted groups, and item mapping (M13.4).
struct CourseGradingSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var baseline = CourseGradingLogic.FormBaseline(
        gradingScale: "letter_plus_minus",
        groups: CourseGradingLogic.defaultGroups(),
        schemeType: "points",
        bands: CourseGradingLogic.defaultBands(),
        passMinPct: "60",
        completeMinPct: "50"
    )
    @State private var form = CourseGradingLogic.FormBaseline(
        gradingScale: "letter_plus_minus",
        groups: CourseGradingLogic.defaultGroups(),
        schemeType: "points",
        bands: CourseGradingLogic.defaultBands(),
        passMinPct: "60",
        completeMinPct: "50"
    )
    @State private var structure: [CourseStructureItem] = []
    @State private var loading = true
    @State private var loadError: String?
    @State private var actionError: String?
    @State private var actionSuccess: String?
    @State private var cacheLabel: String?
    @State private var saving = false
    @State private var itemPatchingId: String?
    @State private var pendingItemIds: Set<String> = []

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var weightTotal: Double { CourseGradingLogic.weightTotal(form.groups) }
    private var isDirty: Bool {
        CourseGradingLogic.isSettingsDirty(current: form, baseline: baseline)
            || CourseGradingLogic.isSchemeDirty(current: form, baseline: baseline)
    }
    private var gradableRows: [CourseGradingLogic.GradableRow] {
        CourseGradingLogic.gradableRows(from: structure)
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

                        gradingScaleCard
                        schemeCard
                        groupsCard
                        mappingCard
                    }
                }
                .padding(16)
            }

            if isDirty {
                UnsavedChangesBanner(
                    isSaving: saving,
                    onSave: { Task { await saveChanges() } },
                    onDiscard: discardChanges
                )
            }
        }
        .task(id: course.courseCode) { await reload(force: false) }
    }

    private var gradingScaleCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.grading.scaleTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.grading.scaleDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                ForEach(CourseGradingLogic.gradingScaleOptions) { option in
                    Button {
                        form.gradingScale = option.id
                    } label: {
                        HStack(alignment: .top, spacing: 10) {
                            Image(systemName: form.gradingScale == option.id ? "largecircle.fill.circle" : "circle")
                                .foregroundStyle(form.gradingScale == option.id ? LexturesTheme.brandTeal : LexturesTheme.textSecondary(for: colorScheme))
                            VStack(alignment: .leading, spacing: 2) {
                                Text(L.text(String.LocalizationValue(option.labelKey)))
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                Text(L.text(String.LocalizationValue(option.descriptionKey)))
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            Spacer(minLength: 0)
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(10)
                        .background(
                            form.gradingScale == option.id
                                ? LexturesTheme.brandTeal.opacity(0.1)
                                : Color.clear,
                            in: RoundedRectangle(cornerRadius: 10)
                        )
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private var schemeCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.grading.schemeTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.grading.schemeDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                Picker(L.text("mobile.courseSettings.grading.schemeDisplayAs"), selection: $form.schemeType) {
                    ForEach(CourseGradingLogic.schemeDisplayTypes) { type in
                        Text(L.text(String.LocalizationValue(type.labelKey))).tag(type.id)
                    }
                }
                .pickerStyle(.menu)

                if form.schemeType == "letter" || form.schemeType == "gpa" {
                    Text(L.text("mobile.courseSettings.grading.bandsTitle"))
                        .font(.subheadline.weight(.semibold))
                    ForEach($form.bands, id: \.clientKey) { $band in
                        HStack(spacing: 8) {
                            TextField(L.text("mobile.courseSettings.grading.bandLabel"), text: $band.label)
                                .textFieldStyle(.roundedBorder)
                            TextField(L.text("mobile.courseSettings.grading.bandMinPct"), text: $band.minPct)
                                .textFieldStyle(.roundedBorder)
                                .keyboardType(.decimalPad)
                            if form.schemeType == "gpa" {
                                TextField("GPA", text: $band.gpa)
                                    .textFieldStyle(.roundedBorder)
                                    .keyboardType(.decimalPad)
                                    .frame(width: 56)
                            }
                            if form.bands.count > 1 {
                                Button(role: .destructive) {
                                    form.bands.removeAll { $0.clientKey == band.clientKey }
                                } label: {
                                    Image(systemName: "trash")
                                }
                            }
                        }
                    }
                    Button(L.text("mobile.courseSettings.grading.addBand")) {
                        form.bands.append(
                            .init(clientKey: CourseGradingLogic.newClientKey(), label: "", minPct: "0", gpa: "")
                        )
                    }
                    .font(.subheadline.weight(.semibold))
                } else if form.schemeType == "pass_fail" {
                    labeledField(L.text("mobile.courseSettings.grading.passMinPct")) {
                        TextField("60", text: $form.passMinPct)
                            .textFieldStyle(.roundedBorder)
                            .keyboardType(.decimalPad)
                    }
                } else if form.schemeType == "complete_incomplete" {
                    labeledField(L.text("mobile.courseSettings.grading.completeMinPct")) {
                        TextField("50", text: $form.completeMinPct)
                            .textFieldStyle(.roundedBorder)
                            .keyboardType(.decimalPad)
                    }
                }
            }
        }
    }

    private var groupsCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.grading.groupsTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.grading.groupsDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                ForEach($form.groups, id: \.clientKey) { $group in
                    VStack(alignment: .leading, spacing: 8) {
                        TextField(L.text("mobile.courseSettings.grading.groupName"), text: $group.name)
                            .textFieldStyle(.roundedBorder)
                        HStack {
                            Text(L.text("mobile.courseSettings.grading.groupWeight"))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            TextField("0", text: $group.weightPercent)
                                .textFieldStyle(.roundedBorder)
                                .keyboardType(.decimalPad)
                            if form.groups.count > 1 {
                                Button(role: .destructive) {
                                    form.groups.removeAll { $0.clientKey == group.clientKey }
                                } label: {
                                    Label(L.text("mobile.courseSettings.grading.removeGroup"), systemImage: "trash")
                                        .font(.caption)
                                }
                            }
                        }
                        if group.clientKey != form.groups.last?.clientKey {
                            Divider()
                        }
                    }
                }

                Button(L.text("mobile.courseSettings.grading.addGroup")) {
                    form.groups.append(
                        .init(
                            clientKey: CourseGradingLogic.newClientKey(),
                            id: nil,
                            name: "",
                            sortOrder: form.groups.count,
                            weightPercent: "0"
                        )
                    )
                }
                .font(.subheadline.weight(.semibold))

                Text(CourseGradingLogic.weightTotalLabel(total: weightTotal))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(
                        CourseGradingLogic.hasWeightWarning(weightTotal)
                            ? Color.orange
                            : LexturesTheme.brandTeal
                    )
                    .accessibilityLabel(CourseGradingLogic.weightTotalLabel(total: weightTotal))
                if CourseGradingLogic.hasWeightWarning(weightTotal) {
                    Text(L.text("mobile.courseSettings.grading.weightWarning"))
                        .font(.caption)
                        .foregroundStyle(Color.orange)
                }
            }
        }
    }

    private var mappingCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.grading.mappingTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.grading.mappingDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if gradableRows.isEmpty {
                    Text(L.text("mobile.courseSettings.grading.mappingEmpty"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(gradableRows) { row in
                        mappingRow(row)
                        if row.id != gradableRows.last?.id {
                            Divider()
                        }
                    }
                }
            }
        }
    }

    private func mappingRow(_ row: CourseGradingLogic.GradableRow) -> some View {
        let selectedGroupId = structure.first(where: { $0.id == row.item.id })?.assignmentGroupId ?? ""
        let namedGroups = CourseGradingLogic.namedGroupsWithIds(form.groups)
        let isPatching = itemPatchingId == row.item.id
        let isPending = pendingItemIds.contains(row.item.id)

        return VStack(alignment: .leading, spacing: 6) {
            if !row.moduleTitle.isEmpty {
                Text(row.moduleTitle)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(row.item.title)
                        .font(.subheadline.weight(.semibold))
                    Text(L.text(String.LocalizationValue(CourseGradingLogic.kindLabelKey(for: row.item.kind))))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer(minLength: 8)
                if isPatching {
                    ProgressView().controlSize(.small)
                } else if isPending {
                    Text(L.text("mobile.courseSettings.grading.pending"))
                        .font(.caption2.weight(.semibold))
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(LexturesTheme.brandTeal.opacity(0.15), in: Capsule())
                }
            }

            Picker(L.text("mobile.courseSettings.grading.mappingGroup"), selection: Binding(
                get: { selectedGroupId },
                set: { newValue in
                    Task { await updateItemGroup(itemId: row.item.id, groupId: newValue.nilIfEmpty) }
                }
            )) {
                Text(L.text("mobile.courseSettings.grading.mappingNone")).tag("")
                ForEach(namedGroups, id: \.clientKey) { group in
                    Text(group.name).tag(group.id ?? "")
                }
            }
            .pickerStyle(.menu)
            .disabled(isPatching || namedGroups.isEmpty)
        }
    }

    private func labeledField<Content: View>(_ label: String, @ViewBuilder content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            content()
        }
    }

    private func discardChanges() {
        form = baseline
        actionError = nil
        actionSuccess = nil
    }

    private func reload(force: Bool) async {
        guard let token = session.accessToken else { return }
        loading = true
        loadError = nil
        defer { loading = false }
        do {
            let settingsResult = try await offline.cachedFetch(
                key: CourseGradingLogic.cacheKeyGrading(courseCode: course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourseGradingSettings(courseCode: course.courseCode, accessToken: token)
            }
            let scheme = try? await LMSAPI.fetchCourseGradingScheme(courseCode: course.courseCode, accessToken: token)
            let loaded = CourseGradingLogic.baseline(settings: settingsResult.value, scheme: scheme)
            baseline = loaded
            form = loaded
            structure = (try? await LMSAPI.fetchCourseStructure(courseCode: course.courseCode, accessToken: token)) ?? []
            if let cached = settingsResult.cached, cached.isStale(isOnline: isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            loadError = error.localizedDescription
        }
    }

    private func saveChanges() async {
        guard let token = session.accessToken else { return }
        actionError = nil
        actionSuccess = nil

        let settingsDirty = CourseGradingLogic.isSettingsDirty(current: form, baseline: baseline)
        let schemeDirty = CourseGradingLogic.isSchemeDirty(current: form, baseline: baseline)

        if settingsDirty, let error = CourseGradingLogic.validateGroups(form.groups) {
            switch error {
            case .groupsNeedNames:
                actionError = L.text("mobile.courseSettings.grading.validation.groupsNeedNames")
            default:
                break
            }
            return
        }
        if schemeDirty, let error = CourseGradingLogic.validateScheme(form: form) {
            switch error {
            case .bandsInvalid(let message), .schemeInvalid(let message):
                actionError = message
            default:
                break
            }
            return
        }

        saving = true
        defer { saving = false }

        do {
            if settingsDirty {
                let body = CourseGradingLogic.buildPutSettingsBody(form: form)
                _ = try await offline.enqueueMutation(
                    method: "PUT",
                    path: "/api/v1/courses/\(course.courseCode)/grading",
                    body: body,
                    label: L.text("mobile.courseSettings.grading.saveSettingsLabel"),
                    accessToken: token,
                    idempotencyKey: CourseGradingLogic.settingsIdempotencyKey(courseCode: course.courseCode)
                )
            }
            if schemeDirty {
                let body = CourseGradingLogic.buildPutSchemeBody(form: form)
                _ = try await offline.enqueueMutation(
                    method: "PUT",
                    path: "/api/v1/courses/\(course.courseCode)/grading-scheme",
                    body: body,
                    label: L.text("mobile.courseSettings.grading.saveSchemeLabel"),
                    accessToken: token,
                    idempotencyKey: CourseGradingLogic.schemeIdempotencyKey(courseCode: course.courseCode)
                )
            }

            let settings = try await LMSAPI.fetchCourseGradingSettings(courseCode: course.courseCode, accessToken: token)
            let scheme = try? await LMSAPI.fetchCourseGradingScheme(courseCode: course.courseCode, accessToken: token)
            let loaded = CourseGradingLogic.baseline(settings: settings, scheme: scheme)
            baseline = loaded
            form = loaded
            structure = (try? await LMSAPI.fetchCourseStructure(courseCode: course.courseCode, accessToken: token)) ?? structure
            if settingsDirty && schemeDirty {
                actionSuccess = L.text("mobile.courseSettings.grading.savedBoth")
            } else if schemeDirty {
                actionSuccess = L.text("mobile.courseSettings.grading.savedScheme")
            } else {
                actionSuccess = L.text("mobile.courseSettings.grading.savedSettings")
            }
        } catch {
            actionError = error.localizedDescription
        }
    }

    private func updateItemGroup(itemId: String, groupId: String?) async {
        guard let token = session.accessToken else { return }
        itemPatchingId = itemId
        actionError = nil
        let previous = structure
        structure = structure.map { item in
            guard item.id == itemId else { return item }
            var updated = item
            updated.assignmentGroupId = groupId
            return updated
        }
        defer { itemPatchingId = nil }

        do {
            let item = try await offline.enqueueMutation(
                method: "PATCH",
                path: "/api/v1/courses/\(course.courseCode)/structure/items/\(itemId)/assignment-group",
                body: PatchItemAssignmentGroupBody(assignmentGroupId: groupId),
                label: L.text("mobile.courseSettings.grading.mappingSaveLabel"),
                accessToken: token,
                idempotencyKey: CourseGradingLogic.itemMappingIdempotencyKey(courseCode: course.courseCode, itemId: itemId)
            )
            if item.status != .synced {
                pendingItemIds.insert(itemId)
            } else {
                pendingItemIds.remove(itemId)
                structure = (try? await LMSAPI.fetchCourseStructure(courseCode: course.courseCode, accessToken: token)) ?? structure
            }
        } catch {
            structure = previous
            actionError = error.localizedDescription
        }
    }
}

private extension String {
    var nilIfEmpty: String? {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}