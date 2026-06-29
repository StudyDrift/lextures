import XCTest
@testable import Lextures

@MainActor
final class I18nTests: XCTestCase {
    func testRTLLocaleDetection() {
        XCTAssertTrue(LocalePreferences.isRTLLocale("ar"))
        XCTAssertTrue(LocalePreferences.isRTLLocale("ar-SA"))
        XCTAssertTrue(LocalePreferences.isRTLLocale("he-IL"))
        XCTAssertFalse(LocalePreferences.isRTLLocale("en"))
        XCTAssertFalse(LocalePreferences.isRTLLocale("es-MX"))
    }

    func testResolveResourceLanguage() {
        XCTAssertEqual(LocalePreferences.resolveResourceLanguage("es"), "es")
        XCTAssertEqual(LocalePreferences.resolveResourceLanguage("fr-CA"), "fr")
        XCTAssertEqual(LocalePreferences.resolveResourceLanguage("ar"), "ar")
        XCTAssertEqual(LocalePreferences.resolveResourceLanguage("en-XA"), "en-XA")
        XCTAssertEqual(LocalePreferences.resolveResourceLanguage("de"), "en")
    }

    func testDateFormattingParsesIsoTimestamps() {
        let parsed = DateFormatting.parse("2026-06-15T23:59:59Z")
        XCTAssertNotNil(parsed)
        let formatted = DateFormatting.formatAbsoluteShort("2026-06-15T23:59:59Z", timeZone: TimeZone(secondsFromGMT: 0)!)
        XCTAssertFalse(formatted.isEmpty)
        XCTAssertNotEqual(formatted, L.text("mobile.emDash"))
    }

    func testDateFormattingDueUsesTimezone() {
        let utc = TimeZone(secondsFromGMT: 0)!
        let pacific = TimeZone(identifier: "America/Los_Angeles")!
        let iso = "2026-06-16T07:59:59Z"
        let utcLabel = DateFormatting.formatDue(iso, timeZone: utc)
        let pacificLabel = DateFormatting.formatDue(iso, timeZone: pacific)
        XCTAssertNotEqual(utcLabel, pacificLabel)
    }

    func testLocalizedAuthLoginTitleIsNonEmpty() {
        XCTAssertFalse(L.text("auth.login.title").isEmpty)
    }
}
