import Foundation
import Observation
import SwiftUI

/// Device locale with optional in-app override (matches web `lextures.locale` storage key).
@MainActor
@Observable
final class LocalePreferences {
    static let shared = LocalePreferences()
    static let storageKey = "lextures.locale"

    /// BCP 47 tags offered in the language picker (endonyms for fixed languages).
    static let localeOptions: [(tag: String, label: String)] = [
        ("system", "System default"),
        ("en", "English"),
        ("es", "Español"),
        ("fr", "Français"),
        ("ar", "العربية"),
        ("en-XA", "Pseudo (en-XA)"),
    ]

    private static let rtlLocales: Set<String> = ["ar", "he", "fa", "ur", "ps"]

    /// Stored override tag, or `system` to follow the device locale.
    var localeTag: String {
        didSet {
            UserDefaults.standard.set(localeTag, forKey: Self.storageKey)
        }
    }

    init() {
        localeTag = UserDefaults.standard.string(forKey: Self.storageKey) ?? "system"
    }

    var usesSystemLocale: Bool {
        localeTag == "system" || localeTag.isEmpty
    }

    /// Effective BCP 47 tag sent to the API and used for formatting.
    var effectiveTag: String {
        if usesSystemLocale {
            return Locale.current.identifier
        }
        return localeTag
    }

    var effectiveLocale: Locale {
        Locale(identifier: effectiveTag)
    }

    var isRTL: Bool {
        Self.isRTLLocale(effectiveTag)
    }

    var layoutDirection: LayoutDirection {
        isRTL ? .rightToLeft : .leftToRight
    }

    var acceptLanguageHeader: String {
        effectiveTag.replacingOccurrences(of: "_", with: "-")
    }

    /// Nonisolated Accept-Language for URLSession (reads UserDefaults directly).
    nonisolated static func acceptLanguageHeaderValue() -> String {
        let stored = UserDefaults.standard.string(forKey: storageKey) ?? "system"
        if stored != "system", !stored.isEmpty {
            return stored.replacingOccurrences(of: "_", with: "-")
        }
        if let preferred = UserDefaults.standard.array(forKey: "AppleLanguages")?.first as? String {
            return preferred.replacingOccurrences(of: "_", with: "-")
        }
        return "en"
    }

    /// Nonisolated effective locale for formatters and networking.
    nonisolated static func effectiveLocaleValue() -> Locale {
        Locale(identifier: acceptLanguageHeaderValue())
    }

    /// Maps a tag to a bundled translation language (en/es/fr); RTL tags keep their layout tag.
    nonisolated static func resolveResourceLanguage(_ tag: String) -> String {
        let primary = tag.split(separator: "-").first?.lowercased() ?? "en"
        if primary == "es" || primary == "fr" || primary == "ar" {
            return String(primary)
        }
        if tag == "en-XA" { return "en-XA" }
        return "en"
    }

    nonisolated static func isRTLLocale(_ tag: String) -> Bool {
        let primary = tag.split(separator: "-").first?.lowercased() ?? ""
        return rtlLocales.contains(primary)
    }

    func applyStoredTag(_ tag: String?) {
        guard let tag, !tag.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
        localeTag = tag.trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
