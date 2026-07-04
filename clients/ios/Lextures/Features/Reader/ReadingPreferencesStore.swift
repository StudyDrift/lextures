import SwiftUI

/// Reading-surface preferences with local persistence and optional server sync (M6.3).
@MainActor
@Observable
final class ReadingPreferencesStore {
    static let shared = ReadingPreferencesStore()

    private(set) var prefs = ReaderLogic.defaultReadingPreferences()
    private(set) var loading = false
    private(set) var serverSyncEnabled = false

    private enum Keys {
        static let fontFace = "lextures.reader.fontFace"
        static let letterSpacing = "lextures.reader.letterSpacing"
        static let wordSpacing = "lextures.reader.wordSpacing"
        static let lineHeight = "lextures.reader.lineHeight"
        static let ttsSpeed = "lextures.reader.ttsSpeed"
        static let dyslexia = "lextures.reader.dyslexia"
    }

    private init() {
        loadLocal()
        syncDyslexiaToAccessibility()
    }

    func loadFromServer(accessToken: String, apiEnabled: Bool) async {
        serverSyncEnabled = apiEnabled
        guard apiEnabled else { return }
        loading = true
        defer { loading = false }
        do {
            let server = try await LMSAPI.fetchReadingPreferences(accessToken: accessToken)
            prefs = ReaderLogic.mergeReadingPreferences(local: prefs, server: server)
            persistLocal()
            syncDyslexiaToAccessibility()
        } catch {
            // Keep local prefs when offline or API unavailable.
        }
    }

    func update(_ patch: ReadingPreferencesPatch, accessToken: String?) async {
        if let fontFace = patch.fontFace { prefs.fontFace = fontFace }
        if let letterSpacing = patch.letterSpacing { prefs.letterSpacing = letterSpacing }
        if let wordSpacing = patch.wordSpacing { prefs.wordSpacing = wordSpacing }
        if let lineHeight = patch.lineHeight { prefs.lineHeight = lineHeight }
        if let ttsSpeed = patch.ttsSpeed { prefs.ttsSpeed = ttsSpeed }
        if let dyslexia = patch.dyslexiaDisplayEnabled {
            prefs.dyslexiaDisplayEnabled = dyslexia
            prefs.fontFace = ReaderLogic.fontFaceFromDyslexia(dyslexia, current: prefs.fontFace)
        }
        persistLocal()
        syncDyslexiaToAccessibility()

        guard serverSyncEnabled, let accessToken else { return }
        _ = try? await LMSAPI.patchReadingPreferences(patch, accessToken: accessToken)
    }

    var usesDyslexiaFont: Bool {
        prefs.dyslexiaDisplayEnabled || prefs.fontFace == "open-dyslexic"
    }

    // MARK: - Private

    private func loadLocal() {
        let defaults = UserDefaults.standard
        prefs.fontFace = defaults.string(forKey: Keys.fontFace) ?? prefs.fontFace
        prefs.letterSpacing = defaults.string(forKey: Keys.letterSpacing) ?? prefs.letterSpacing
        prefs.wordSpacing = defaults.string(forKey: Keys.wordSpacing) ?? prefs.wordSpacing
        prefs.lineHeight = defaults.string(forKey: Keys.lineHeight) ?? prefs.lineHeight
        if defaults.object(forKey: Keys.ttsSpeed) != nil {
            prefs.ttsSpeed = defaults.double(forKey: Keys.ttsSpeed)
        }
        prefs.dyslexiaDisplayEnabled = defaults.bool(forKey: Keys.dyslexia)
    }

    private func persistLocal() {
        let defaults = UserDefaults.standard
        defaults.set(prefs.fontFace, forKey: Keys.fontFace)
        defaults.set(prefs.letterSpacing, forKey: Keys.letterSpacing)
        defaults.set(prefs.wordSpacing, forKey: Keys.wordSpacing)
        defaults.set(prefs.lineHeight, forKey: Keys.lineHeight)
        defaults.set(prefs.ttsSpeed, forKey: Keys.ttsSpeed)
        defaults.set(prefs.dyslexiaDisplayEnabled, forKey: Keys.dyslexia)
    }

    private func syncDyslexiaToAccessibility() {
        AccessibilityPreferences.shared.dyslexiaDisplayEnabled = usesDyslexiaFont
        AccessibilityPreferences.shared.ttsSpeed = Float(prefs.ttsSpeed)
    }
}

private struct ReadingPreferencesStoreKey: EnvironmentKey {
    static let defaultValue = ReadingPreferencesStore.shared
}

extension EnvironmentValues {
    var readingPreferencesStore: ReadingPreferencesStore {
        get { self[ReadingPreferencesStoreKey.self] }
        set { self[ReadingPreferencesStoreKey.self] = newValue }
    }
}