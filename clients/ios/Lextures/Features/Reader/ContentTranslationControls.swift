import SwiftUI

/// Inline translate + course locale picker for content surfaces (M6.3).
struct ContentTranslationControls: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    enum Mode {
        case ugc(contentType: String, contentId: String, text: String, targetLang: String)
        case courseContent(courseCode: String)
    }

    let mode: Mode
    var onReload: (() async -> Void)?

    @State private var translateState: TranslateUIState = .idle
    @State private var translatedText: String?
    @State private var sourceLang: String?
    @State private var showOriginal = false
    @State private var locales: [TranslationCoverageLocale] = []
    @State private var selectedLocale = ""
    @State private var savingLocale = false

    private enum TranslateUIState {
        case idle, loading, done, error
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            switch mode {
            case .ugc(let contentType, let contentId, let text, let targetLang):
                ugcControls(contentType: contentType, contentId: contentId, text: text, targetLang: targetLang)
            case .courseContent(let courseCode):
                courseLocalePicker(courseCode: courseCode)
            }
        }
        .task {
            if case .courseContent(let courseCode) = mode {
                await loadLocales(courseCode: courseCode)
            }
        }
    }

    @ViewBuilder
    private func ugcControls(contentType: String, contentId: String, text: String, targetLang: String) -> some View {
        if translateState == .done, let translatedText, !showOriginal {
            Text(translatedText)
                .font(.subheadline)
                .lineSpacing(4)
                .padding(10)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(LexturesTheme.sceneBackground(for: colorScheme))
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
            Label("Machine translated", systemImage: "globe")
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            if let sourceLang {
                Text("From \(ReaderLogic.localeLabel(sourceLang))")
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }

        HStack(spacing: 10) {
            if translateState == .idle || translateState == .error {
                Button("Translate") {
                    Task { await translateUGC(contentType: contentType, contentId: contentId, text: text, targetLang: targetLang) }
                }
                .font(.caption.weight(.semibold))
                .minimumTapTarget()
            } else if translateState == .loading {
                ProgressView()
                    .controlSize(.small)
                Text("Translating…")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else if translateState == .done {
                Button(showOriginal ? "Show translation" : "Show original") {
                    showOriginal.toggle()
                }
                .font(.caption.weight(.semibold))
                .minimumTapTarget()
            }
        }
    }

    @ViewBuilder
    private func courseLocalePicker(courseCode: String) -> some View {
        if !locales.isEmpty {
            HStack(spacing: 8) {
                Image(systemName: "globe")
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                Picker("Content language", selection: $selectedLocale) {
                    Text("Original").tag("")
                    ForEach(locales, id: \.targetLocale) { locale in
                        Text("\(ReaderLogic.localeLabel(locale.targetLocale)) (\(Int(locale.percent))%)")
                            .tag(locale.targetLocale)
                    }
                }
                .pickerStyle(.menu)
                .disabled(savingLocale)
                .onChange(of: selectedLocale) { _, newValue in
                    Task { await saveLocale(courseCode: courseCode, locale: newValue.isEmpty ? nil : newValue) }
                }
            }
            .font(.caption.weight(.semibold))
            .accessibilityLabel("Course content language")
        }
    }

    private func translateUGC(
        contentType: String,
        contentId: String,
        text: String,
        targetLang: String
    ) async {
        guard let token = session.accessToken, !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
        translateState = .loading
        do {
            let result = try await LMSAPI.translateContent(
                contentType: contentType,
                contentId: contentId,
                targetLang: targetLang,
                text: text,
                accessToken: token
            )
            translatedText = result.translated
            sourceLang = result.sourceLang
            showOriginal = false
            translateState = .done
        } catch {
            translateState = .error
        }
    }

    private func loadLocales(courseCode: String) async {
        guard let token = session.accessToken else { return }
        locales = (try? await LMSAPI.fetchTranslationCoverage(courseCode: courseCode, accessToken: token))?
            .filter { $0.percent > 0 } ?? []
    }

    private func saveLocale(courseCode: String, locale: String?) async {
        guard let token = session.accessToken else { return }
        savingLocale = true
        defer { savingLocale = false }
        do {
            try await LMSAPI.patchMyContentLocale(courseCode: courseCode, locale: locale, accessToken: token)
            await onReload?()
        } catch {
            // Keep picker value; parent may show error via reload failure.
        }
    }
}