import SwiftUI

/// Native content page reader with sticky title, offline cache, and completion (M3.1).
struct ContentPageView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let item: CourseStructureItem
    var onProgressChanged: (() async -> Void)?

    @State private var detail: ModuleItemDetail?
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var markingComplete = false
    @State private var isComplete = false

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 0) {
                    stickyHeader

                    VStack(alignment: .leading, spacing: 14) {
                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }
                        if let cacheLabel {
                            StalenessChip(label: cacheLabel)
                        }
                        if loading {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 40)
                        } else if let markdown = detail?.markdown, !markdown.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                            LMSCard {
                                ReadAloudButton(text: markdown)
                                CourseMarkdownContentView(markdown: markdown)
                                    .lexturesReadableText()
                            }
                            .accessibilityElement(children: .contain)
                        }
                        if !loading && course.viewerIsStudent && !isComplete {
                            markDoneButton
                        }
                    }
                    .padding(16)
                }
            }
            .refreshable { await load() }
        }
        .navigationTitle(item.title)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private var stickyHeader: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(detail?.title ?? item.title)
                .font(LexturesTheme.displayFont(22))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            if isComplete {
                Label(L.text("mobile.modules.complete"), systemImage: "checkmark.circle.fill")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.primary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(LexturesTheme.sceneBackground(for: colorScheme).opacity(0.95))
    }

    private var markDoneButton: some View {
        Button {
            Task { await markComplete() }
        } label: {
            HStack {
                if markingComplete {
                    ProgressView()
                } else {
                    Text(L.text("mobile.modules.markDone"))
                }
            }
            .frame(maxWidth: .infinity)
        }
        .buttonStyle(AuthPrimaryButtonStyle())
        .disabled(markingComplete)
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.contentPage(courseCode: course.courseCode, itemId: item.id),
                accessToken: token
            ) {
                guard let page = try await LMSAPI.fetchItemDetail(
                    courseCode: course.courseCode,
                    item: item,
                    accessToken: token
                ) else {
                    throw APIError.httpStatus(404, message: L.text("mobile.modules.loadError"))
                }
                return page
            }
            detail = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
            if let progress = try? await LMSAPI.fetchModulesProgress(courseCode: course.courseCode, accessToken: token) {
                isComplete = ModuleContentLogic.isComplete(in: progress, itemId: item.id)
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.modules.loadError")
        }
    }

    private func markComplete() async {
        guard let token = session.accessToken else { return }
        markingComplete = true
        defer { markingComplete = false }
        do {
            try await offline.enqueueMutation(
                method: "POST",
                path: "/api/v1/courses/\(LMSAPI.encodePath(course.courseCode))/items/\(LMSAPI.encodePath(item.id))/complete",
        body: nil,
                label: L.text("mobile.modules.markDone"),
                accessToken: token
            )
            isComplete = true
            await onProgressChanged?()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.modules.markDoneError")
        }
    }
}
