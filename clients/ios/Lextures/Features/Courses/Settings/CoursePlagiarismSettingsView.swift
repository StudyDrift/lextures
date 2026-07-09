import SwiftUI

/// Plagiarism and AI-authorship settings (M13.7).
struct CoursePlagiarismSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var baseline = CoursePlagiarismLogic.draft(from: nil as CoursePlagiarismSettings?)
    @State private var form = CoursePlagiarismLogic.draft(from: nil as CoursePlagiarismSettings?)
    @State private var loading = true
    @State private var loadError: String?
    @State private var actionError: String?
    @State private var actionSuccess: String?
    @State private var cacheLabel: String?
    @State private var saving = false

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var isDirty: Bool { CoursePlagiarismLogic.isDirty(current: form, baseline: baseline) }

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

                        settingsCard
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

    private var settingsCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 16) {
                VStack(alignment: .leading, spacing: 8) {
                    Text(L.text("mobile.courseSettings.plagiarism.introTitle"))
                        .font(.headline)
                    Text(L.text("mobile.courseSettings.plagiarism.introDescription"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                Toggle(
                    L.text("mobile.courseSettings.plagiarism.enableLabel"),
                    isOn: $form.checksEnabled
                )

                VStack(alignment: .leading, spacing: 6) {
                    Text(L.text("mobile.courseSettings.plagiarism.providerLabel"))
                        .font(.subheadline.weight(.semibold))
                    Picker(L.text("mobile.courseSettings.plagiarism.providerLabel"), selection: $form.provider) {
                        ForEach(CoursePlagiarismLogic.providerOptions) { option in
                            Text(L.text(String.LocalizationValue(option.labelKey))).tag(option.value)
                        }
                    }
                    .pickerStyle(.menu)
                }

                VStack(alignment: .leading, spacing: 6) {
                    Text(L.text("mobile.courseSettings.plagiarism.thresholdLabel"))
                        .font(.subheadline.weight(.semibold))
                    TextField(
                        L.text("mobile.courseSettings.plagiarism.thresholdLabel"),
                        text: $form.thresholdPct
                    )
                    .textFieldStyle(.roundedBorder)
                    .keyboardType(.decimalPad)
                    .accessibilityLabel(L.text("mobile.courseSettings.plagiarism.thresholdLabel"))
                    Text(L.text("mobile.courseSettings.plagiarism.thresholdHint"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                Text(L.text("mobile.courseSettings.plagiarism.privacyNote"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                Text(L.text("mobile.courseSettings.plagiarism.signalNote"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func discardChanges() {
        form = baseline
        actionError = nil
        actionSuccess = nil
    }

    private func reload(force: Bool) async {
        guard let token = session.accessToken else { return }
        if !force && !loading && loadError == nil && !isDirty { return }
        loading = baseline == CoursePlagiarismLogic.draft(from: nil as CoursePlagiarismSettings?) && loadError == nil
        loadError = nil
        defer { loading = false }

        do {
            let result = try await offline.cachedFetch(
                key: CoursePlagiarismLogic.cacheKey(courseCode: course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCoursePlagiarismSettings(
                    courseCode: course.courseCode,
                    accessToken: token
                )
            }
            let loaded = CoursePlagiarismLogic.draft(from: result.value)
            baseline = loaded
            if !isDirty || force {
                form = loaded
            }
            if let cached = result.cached, cached.isStale(isOnline: isOnline) {
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

        if CoursePlagiarismLogic.validateDraft(form) == .thresholdInvalid {
            actionError = L.text("mobile.courseSettings.plagiarism.validation.thresholdInvalid")
            return
        }

        saving = true
        defer { saving = false }

        do {
            let body = CoursePlagiarismLogic.buildPatchBody(current: form)
            _ = try await offline.enqueueMutation(
                method: "PATCH",
                path: CoursePlagiarismLogic.patchPath(courseCode: course.courseCode),
                body: body,
                label: L.text("mobile.courseSettings.plagiarism.saveLabel"),
                accessToken: token,
                idempotencyKey: CoursePlagiarismLogic.saveIdempotencyKey(courseCode: course.courseCode)
            )
            let refreshed = try await LMSAPI.fetchCoursePlagiarismSettings(
                courseCode: course.courseCode,
                accessToken: token
            )
            let loaded = CoursePlagiarismLogic.draft(from: refreshed)
            baseline = loaded
            form = loaded
            actionSuccess = L.text("mobile.courseSettings.plagiarism.saved")
        } catch {
            actionError = error.localizedDescription
        }
    }
}
