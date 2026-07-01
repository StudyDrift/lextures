import Foundation

/// Typed accessors for localized UI strings (keys align with web i18n).
enum L {
    static func text(_ key: String.LocalizationValue) -> String {
        String(localized: key, locale: LocalePreferences.effectiveLocaleValue())
    }

    static func format(_ key: String.LocalizationValue, _ args: CVarArg...) -> String {
        String(format: text(key), locale: LocalePreferences.effectiveLocaleValue(), arguments: args)
    }

    static func plural(_ key: String.LocalizationValue, count: Int) -> String {
        String(format: String(localized: key, locale: LocalePreferences.effectiveLocaleValue()), locale: LocalePreferences.effectiveLocaleValue(), count)
    }

    /// Resolve a runtime string table key (e.g. profile-field metadata).
    static func dynamicText(_ key: String) -> String {
        String(
            localized: String.LocalizationValue(stringLiteral: key),
            locale: LocalePreferences.effectiveLocaleValue()
        )
    }
}
