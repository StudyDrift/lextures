import SwiftUI

/// Opens a course e-reserve / library module item with access logging (M3.6).
struct LibraryResourceView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let course: CourseSummary
    let item: CourseStructureItem
    var nativeEnabled: Bool = true

    @State private var payload: LibraryResourcePayload?
    @State private var accessState: LibraryAccessState?
    @State private var openURLString: String?
    @State private var accessEventError: String?
    @State private var loadError: String?
    @State private var loading = true

    var body: some View {
        Group {
            if let openURLString {
                WebItemView(title: displayTitle, urlString: openURLString, provider: L.text("mobile.library.providerLabel"))
            } else if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                detailBody
            }
        }
        .navigationTitle(displayTitle)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private var displayTitle: String {
        payload?.metadata?.title?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
            ?? item.title
    }

    @ViewBuilder
    private var detailBody: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    if let loadError {
                        LMSErrorBanner(message: loadError)
                    }
                    if let accessEventError {
                        LMSErrorBanner(message: accessEventError)
                    }
                    headerCard
                    accessCard
                }
                .padding(16)
            }
        }
    }

    private var headerCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(displayTitle)
                    .font(.headline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                if let payload {
                    Text(LibraryResourceLogic.resourceTypeLabel(payload.resourceType))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    metadataLines(payload.metadata)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private var accessCard: some View {
        switch accessState {
        case .ready(let url):
            LMSCard {
                VStack(alignment: .leading, spacing: 12) {
                    Text(L.text("mobile.library.readyToOpen"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Button(L.text("mobile.library.openResource")) {
                        openURLString = url
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(LexturesTheme.accent(for: colorScheme))
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        case .gated(let messageKey):
            LMSCard {
                VStack(alignment: .leading, spacing: 12) {
                    Label(L.text("mobile.library.accessRestricted"), systemImage: "lock.fill")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(localizedMessage(messageKey))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if nativeEnabled {
                        openOnWebButton
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        case .requiresWeb(let path):
            LMSCard {
                VStack(alignment: .leading, spacing: 12) {
                    Text(L.text("mobile.library.largerScreenHint"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Button(L.text("mobile.library.openOnWeb")) {
                        openURL(AppConfiguration.apiURL(path: path))
                    }
                    .buttonStyle(.bordered)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        case nil:
            EmptyView()
        }
    }

    private var openOnWebButton: some View {
        Button(L.text("mobile.library.openOnWeb")) {
            let path = LibraryResourceLogic.webModulePath(courseCode: course.courseCode, itemId: item.id)
            openURL(AppConfiguration.apiURL(path: path))
        }
        .buttonStyle(.bordered)
    }

    @ViewBuilder
    private func metadataLines(_ meta: LibraryResourceMeta?) -> some View {
        if let author = meta?.author?.nilIfEmpty {
            Text(author).font(.caption).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        if let isbn = meta?.isbn?.nilIfEmpty {
            Text("ISBN \(isbn)").font(.caption).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        if let issn = meta?.issn?.nilIfEmpty {
            Text("ISSN \(issn)").font(.caption).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        defer { loading = false }
        do {
            guard let row = try await LMSAPI.fetchModuleLibraryResource(
                courseCode: course.courseCode,
                itemId: item.id,
                accessToken: token
            ) else {
                loadError = L.text("mobile.library.notFound")
                return
            }
            payload = row
            let state = LibraryResourceLogic.resolveAccess(payload: row)
            accessState = state
            if case .ready(let url) = state {
                await recordAccess(token: token)
                openURLString = url
            }
        } catch {
            loadError = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.library.loadError")
        }
    }

    private func recordAccess(token: String) async {
        do {
            try await offline.enqueueMutation(
                method: "POST",
                path: LibraryResourceLogic.accessEventPath(courseCode: course.courseCode, itemId: item.id),
                body: nil as String?,
                label: L.text("mobile.library.accessEventLabel"),
                accessToken: token
            )
        } catch {
            accessEventError = L.text("mobile.library.accessEventFailed")
        }
    }

    private func localizedMessage(_ key: String) -> String {
        let localized = String(
            localized: String.LocalizationValue(stringLiteral: key),
            locale: LocalePreferences.effectiveLocaleValue()
        )
        return localized == key ? key : localized
    }
}

private extension String {
    var nilIfEmpty: String? {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}