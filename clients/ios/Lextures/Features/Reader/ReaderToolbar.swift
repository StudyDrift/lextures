import SwiftUI

/// Unified reader toolbar: read-aloud, translate, reading prefs (M6.3).
struct ReaderToolbar: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.readingPreferencesStore) private var readingStore

    let text: String
    var courseCode: String?
    var ugcTranslation: ContentTranslationControls.Mode?
    var onContentReload: (() async -> Void)?
    var readAloudEnabled = true
    var translationEnabled = true
    var preferencesEnabled = true

    @State private var showPreferences = false

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 12) {
                if readAloudEnabled, !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    ReadAloudButton(text: text)
                }
                if preferencesEnabled {
                    Button {
                        showPreferences = true
                    } label: {
                        Label("Reading preferences", systemImage: "textformat.size")
                            .font(.caption.weight(.semibold))
                    }
                    .minimumTapTarget()
                }
            }

            if translationEnabled {
                if let ugc = ugcTranslation {
                    ContentTranslationControls(mode: ugc)
                } else if let courseCode {
                    ContentTranslationControls(mode: .courseContent(courseCode: courseCode), onReload: onContentReload)
                }
            }
        }
        .sheet(isPresented: $showPreferences) {
            ReadingPreferencesSheet()
        }
        .task {
            await readingStore.loadFromServer(
                accessToken: session.accessToken ?? "",
                apiEnabled: preferencesEnabled
            )
        }
    }
}

