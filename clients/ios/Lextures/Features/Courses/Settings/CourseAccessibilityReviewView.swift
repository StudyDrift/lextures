import SwiftUI

/// Alt-text coverage review and inline fixes (M13.8).
struct CourseAccessibilityReviewView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var data: CourseAccessibilityInfo?
    @State private var listPage = 0
    @State private var loading = true
    @State private var loadError: String?
    @State private var actionError: String?
    @State private var actionSuccess: String?
    @State private var cacheLabel: String?
    @State private var selectedItem: UncoveredAccessibilityItem?

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var coverage: AltTextCoverage? { data?.altTextCoverage }
    private var uncoveredItems: [UncoveredAccessibilityItem] { coverage?.uncoveredItems ?? [] }
    private var visibleItems: [UncoveredAccessibilityItem] {
        CourseAccessibilityReviewLogic.paginatedUncoveredItems(uncoveredItems, page: listPage)
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
                        if let coverage {
                            coverageCard(coverage)
                            if coverage.uncoveredItems.isEmpty {
                                emptyStateCard
                            } else {
                                gapsCard
                            }
                        }
                    }
                }
                .padding(16)
            }
        }
        .task(id: course.courseCode) { await reload(force: false) }
        .sheet(item: $selectedItem) { item in
            NavigationStack {
                AltTextItemFixerView(
                    course: course,
                    item: item,
                    onDismiss: { selectedItem = nil },
                    onSaved: {
                        selectedItem = nil
                        actionSuccess = L.text("mobile.courseSettings.accessibility.saved")
                        Task { await reload(force: true) }
                    }
                )
            }
        }
    }

    private var introCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.courseSettings.accessibility.introTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.accessibility.introDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func coverageCard(_ coverage: AltTextCoverage) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.accessibility.coverageTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.accessibility.coverageDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                HStack(spacing: 16) {
                    ZStack {
                        Circle()
                            .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 6)
                        Circle()
                            .trim(from: 0, to: CGFloat(coverage.percent) / 100)
                            .stroke(LexturesTheme.brandTeal, style: StrokeStyle(lineWidth: 6, lineCap: .round))
                            .rotationEffect(.degrees(-90))
                        Text("\(coverage.percent)%")
                            .font(.title3.weight(.bold))
                    }
                    .frame(width: 72, height: 72)
                    .accessibilityLabel(CourseAccessibilityReviewLogic.formatCoverageLabel(
                        withAlt: coverage.withAlt,
                        total: coverage.total
                    ))

                    Text(CourseAccessibilityReviewLogic.formatCoverageLabel(
                        withAlt: coverage.withAlt,
                        total: coverage.total
                    ))
                    .font(.title3.weight(.semibold))
                }

                if data?.hardBlockSave == true {
                    Text(L.text("mobile.courseSettings.accessibility.hardBlockNote"))
                        .font(.caption)
                        .foregroundStyle(.orange)
                }
            }
        }
    }

    private var emptyStateCard: some View {
        LMSCard(accent: LexturesTheme.brandTeal) {
            Label(L.text("mobile.courseSettings.accessibility.emptyState"), systemImage: "checkmark.seal.fill")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.primary)
        }
    }

    private var gapsCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(L.text("mobile.courseSettings.accessibility.gapsTitle"))
                    .font(.headline)

                ForEach(visibleItems) { item in
                    Button {
                        selectedItem = item
                    } label: {
                        HStack(alignment: .top, spacing: 10) {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(item.title.isEmpty
                                    ? L.text("mobile.courseSettings.accessibility.untitled")
                                    : item.title)
                                    .font(.subheadline.weight(.medium))
                                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                    .multilineTextAlignment(.leading)
                                Text(L.text(String.LocalizationValue(
                                    CourseAccessibilityReviewLogic.kindLabelKey(for: item.kind)
                                )))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                Text(CourseAccessibilityReviewLogic.itemMissingLabel(
                                    missing: item.missing,
                                    total: item.total
                                ))
                                .font(.caption)
                                .foregroundStyle(.orange)
                            }
                            Spacer(minLength: 8)
                            Image(systemName: "chevron.right")
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.vertical, 4)
                    }
                    .buttonStyle(.plain)
                    .accessibilityHint(L.text("mobile.courseSettings.accessibility.openFixerHint"))
                }

                if CourseAccessibilityReviewLogic.hasMorePages(items: uncoveredItems, page: listPage) {
                    Button(L.text("mobile.courseSettings.accessibility.loadMore")) {
                        listPage += 1
                    }
                    .font(.subheadline.weight(.semibold))
                }
            }
        }
    }

    private func reload(force: Bool) async {
        guard let token = session.accessToken else { return }
        if !force && !loading && loadError == nil { return }
        loading = data == nil && loadError == nil
        loadError = nil
        defer { loading = false }

        do {
            let result = try await offline.cachedFetch(
                key: CourseAccessibilityReviewLogic.cacheKey(courseCode: course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourseAccessibility(
                    courseCode: course.courseCode,
                    accessToken: token
                )
            }
            data = result.value
            listPage = 0
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

private struct AltTextItemFixerView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let course: CourseSummary
    let item: UncoveredAccessibilityItem
    var onDismiss: () -> Void
    var onSaved: () -> Void

    @State private var markdown = ""
    @State private var missingImages: [CourseAccessibilityReviewLogic.MarkdownImageRef] = []
    @State private var drafts: [Int: CourseAccessibilityReviewLogic.ImageAltDraft] = [:]
    @State private var loading = true
    @State private var loadError: String?
    @State private var actionError: String?
    @State private var saving = false
    @State private var suggestingIndex: Int?
    @State private var aiUnavailable = false

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var supportsEdit: Bool { CourseAccessibilityReviewLogic.supportsInlineEdit(kind: item.kind) }
    private var pendingUpdates: [(imageIndex: Int, alt: String, decorative: Bool)] {
        CourseAccessibilityReviewLogic.pendingUpdates(images: missingImages, drafts: drafts)
    }
    private var isDirty: Bool { !pendingUpdates.isEmpty }

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if loading {
                        ProgressView(L.text("mobile.courseSettings.loading"))
                    } else if !supportsEdit {
                        linkOutCard
                    } else {
                        if !isOnline { OfflineBanner() }
                        if let loadError { LMSErrorBanner(message: loadError) }
                        if let actionError { LMSErrorBanner(message: actionError) }
                        if aiUnavailable {
                            Text(L.text("mobile.courseSettings.accessibility.aiUnavailable"))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }

                        if missingImages.isEmpty {
                            LMSCard(accent: LexturesTheme.brandTeal) {
                                Label(
                                    L.text("mobile.courseSettings.accessibility.itemComplete"),
                                    systemImage: "checkmark.circle.fill"
                                )
                                .font(.subheadline.weight(.semibold))
                            }
                        } else {
                            ForEach(missingImages) { image in
                                imageEditorCard(image)
                            }
                        }
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
        .navigationTitle(item.title.isEmpty
            ? L.text("mobile.courseSettings.accessibility.untitled")
            : item.title)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button(L.text("mobile.common.cancel")) { onDismiss() }
            }
        }
        .task { await loadItem() }
    }

    private var linkOutCard: some View {
        LMSEmptyState(
            systemImage: "desktopcomputer",
            title: L.text("mobile.courseSettings.accessibility.linkOutTitle"),
            message: L.text("mobile.courseSettings.accessibility.linkOutMessage")
        )
    }

    private func imageEditorCard(_ image: CourseAccessibilityReviewLogic.MarkdownImageRef) -> some View {
        let draft = drafts[image.globalIndex] ?? CourseAccessibilityReviewLogic.ImageAltDraft(
            alt: image.alt,
            decorative: image.decorative
        )
        let bindingAlt = Binding<String>(
            get: { drafts[image.globalIndex]?.alt ?? draft.alt },
            set: { newValue in
                var next = drafts[image.globalIndex] ?? draft
                next.alt = newValue
                drafts[image.globalIndex] = next
            }
        )
        let bindingDecorative = Binding<Bool>(
            get: { drafts[image.globalIndex]?.decorative ?? draft.decorative },
            set: { newValue in
                var next = drafts[image.globalIndex] ?? draft
                next.decorative = newValue
                drafts[image.globalIndex] = next
            }
        )

        return LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                AsyncImage(url: URL(string: image.src)) { phase in
                    switch phase {
                    case .success(let loaded):
                        loaded
                            .resizable()
                            .scaledToFit()
                            .frame(maxHeight: 160)
                            .clipShape(RoundedRectangle(cornerRadius: 8))
                    case .failure:
                        Image(systemName: "photo")
                            .font(.largeTitle)
                            .frame(maxWidth: .infinity, minHeight: 80)
                    default:
                        ProgressView()
                            .frame(maxWidth: .infinity, minHeight: 80)
                    }
                }
                .accessibilityHidden(true)

                Text(L.format("mobile.courseSettings.accessibility.imageLine", image.line))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                Toggle(L.text("mobile.courseSettings.accessibility.decorativeLabel"), isOn: bindingDecorative)

                TextField(
                    L.text("mobile.courseSettings.accessibility.altTextLabel"),
                    text: bindingAlt,
                    axis: .vertical
                )
                .textFieldStyle(.roundedBorder)
                .lineLimit(2 ... 5)
                .disabled(bindingDecorative.wrappedValue)
                .accessibilityLabel(L.text("mobile.courseSettings.accessibility.altTextLabel"))

                Button {
                    Task { await suggestAltText(for: image) }
                } label: {
                    Label(
                        L.text("mobile.courseSettings.accessibility.suggestButton"),
                        systemImage: "sparkles"
                    )
                }
                .disabled(bindingDecorative.wrappedValue || suggestingIndex == image.globalIndex)
            }
        }
    }

    private func discardChanges() {
        drafts = CourseAccessibilityReviewLogic.drafts(from: missingImages)
        actionError = nil
    }

    private func loadItem() async {
        guard let token = session.accessToken else { return }
        guard supportsEdit else {
            loading = false
            return
        }
        loading = true
        loadError = nil
        defer { loading = false }

        do {
            let structureItem = CourseStructureItem(
                id: item.itemId,
                sortOrder: 0,
                kind: item.kind,
                title: item.title,
                parentId: nil,
                published: true,
                dueAt: nil,
                pointsWorth: nil,
                pointsPossible: nil,
                archived: nil,
                updatedAt: nil
            )
            guard let detail = try await LMSAPI.fetchItemDetail(
                courseCode: course.courseCode,
                item: structureItem,
                accessToken: token
            ) else {
                loadError = L.text("mobile.courseSettings.accessibility.loadItemError")
                return
            }
            let loadedMarkdown = detail.markdown ?? ""
            markdown = loadedMarkdown
            missingImages = CourseAccessibilityReviewLogic.missingImages(loadedMarkdown)
            drafts = CourseAccessibilityReviewLogic.drafts(from: missingImages)
        } catch {
            loadError = error.localizedDescription
        }
    }

    private func suggestAltText(for image: CourseAccessibilityReviewLogic.MarkdownImageRef) async {
        guard let token = session.accessToken else { return }
        suggestingIndex = image.globalIndex
        defer { suggestingIndex = nil }
        do {
            let suggestion = try await LMSAPI.suggestAltText(
                courseCode: course.courseCode,
                imageUrl: image.src,
                language: "",
                accessToken: token
            )
            guard !suggestion.suggestion.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
            var next = drafts[image.globalIndex] ?? CourseAccessibilityReviewLogic.ImageAltDraft(alt: "", decorative: false)
            next.alt = suggestion.suggestion
            next.decorative = false
            drafts[image.globalIndex] = next
            aiUnavailable = false
        } catch {
            aiUnavailable = true
        }
    }

    private func saveChanges() async {
        guard let token = session.accessToken else { return }
        let updates = pendingUpdates
        guard !updates.isEmpty else { return }
        guard let path = CourseAccessibilityReviewLogic.markdownPatchPath(
            courseCode: course.courseCode,
            itemId: item.itemId,
            kind: item.kind
        ) else { return }
        guard let updatedMarkdown = CourseAccessibilityReviewLogic.applyAltTextUpdates(
            in: markdown,
            updates: updates
        ) else {
            actionError = L.text("mobile.courseSettings.accessibility.saveError")
            return
        }

        saving = true
        actionError = nil
        defer { saving = false }

        do {
            _ = try await offline.enqueueMutation(
                method: "PATCH",
                path: path,
                body: PatchItemMarkdownBody(markdown: updatedMarkdown),
                label: L.text("mobile.courseSettings.accessibility.saveLabel"),
                accessToken: token,
                idempotencyKey: CourseAccessibilityReviewLogic.saveMarkdownIdempotencyKey(
                    courseCode: course.courseCode,
                    itemId: item.itemId,
                    kind: item.kind
                )
            )
            markdown = updatedMarkdown
            missingImages = CourseAccessibilityReviewLogic.missingImages(updatedMarkdown)
            drafts = CourseAccessibilityReviewLogic.drafts(from: missingImages)
            if missingImages.isEmpty {
                onSaved()
            }
        } catch {
            actionError = error.localizedDescription
        }
    }
}
