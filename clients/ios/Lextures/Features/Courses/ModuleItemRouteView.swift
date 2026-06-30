import SwiftUI

/// Placeholder for item types routed to future mobile epics (M3.2, M3.3, M4.1, M5.1).
struct ModuleItemPlaceholderView: View {
    @Environment(\.colorScheme) private var colorScheme

    let item: CourseStructureItem
    let messageKey: String

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            LMSEmptyState(
                systemImage: ItemKind.icon(for: item.kind),
                title: item.title,
                message: localizedMessage(messageKey)
            )
            .padding(24)
        }
        .navigationTitle(item.title)
        .navigationBarTitleDisplayMode(.inline)
    }

    private func localizedMessage(_ key: String) -> String {
        let localized = String(
            localized: String.LocalizationValue(stringLiteral: key),
            locale: LocalePreferences.effectiveLocaleValue()
        )
        return localized == key ? key : localized
    }
}

/// Routes a structure item to its native destination (M3.1).
struct ModuleItemRouteView: View {
    let course: CourseSummary
    let item: CourseStructureItem
    var onProgressChanged: (() async -> Void)?

    var body: some View {
        switch ModuleContentLogic.destination(for: item.kind) {
        case .contentPage:
            ContentPageView(course: course, item: item, onProgressChanged: onProgressChanged)
        case .quiz:
            ModuleItemPlaceholderView(item: item, messageKey: "mobile.modules.placeholder.quiz")
        case .assignment:
            ItemDetailView(course: course, item: item)
        case .externalLink, .webContent:
            WebItemLoader(course: course, item: item)
        case .interactive:
            ModuleItemPlaceholderView(item: item, messageKey: "mobile.modules.placeholder.interactive")
        case .file:
            ModuleItemPlaceholderView(item: item, messageKey: "mobile.modules.placeholder.file")
        case .unsupported:
            ModuleItemPlaceholderView(item: item, messageKey: "mobile.modules.placeholder.unsupported")
        }
    }
}

/// Loads URL for external/textbook items then opens WebItemView.
private struct WebItemLoader: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let item: CourseStructureItem

    @State private var url: String?
    @State private var provider: String?
    @State private var loading = true
    @State private var errorMessage: String?

    var body: some View {
        Group {
            if let url {
                WebItemView(title: item.title, urlString: url, provider: provider)
            } else if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                ZStack {
                    LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                    LMSEmptyState(
                        systemImage: "link",
                        title: item.title,
                        message: errorMessage ?? L.text("mobile.modules.loadError")
                    )
                }
            }
        }
        .navigationTitle(item.title)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        defer { loading = false }
        do {
            let detail = try await LMSAPI.fetchItemDetail(courseCode: course.courseCode, item: item, accessToken: token)
            if let link = detail?.url, !link.isEmpty {
                url = link
                provider = detail?.provider
            } else {
                errorMessage = L.text("mobile.modules.noLink")
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
        }
    }
}
