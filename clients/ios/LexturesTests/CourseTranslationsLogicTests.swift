import XCTest
@testable import Lextures

final class CourseTranslationsLogicTests: XCTestCase {
    func testFeatureGate() {
        var off = MobilePlatformFeatures()
        off.translationMemoryEnabled = false
        XCTAssertFalse(CourseTranslationsLogic.isFeatureEnabled(off))
        off.translationMemoryEnabled = true
        XCTAssertTrue(CourseTranslationsLogic.isFeatureEnabled(off))
    }

    func testCoveragePercentEmptyTotalIs100() {
        XCTAssertEqual(CourseTranslationsLogic.coveragePercent(translated: 0, total: 0), 100)
    }

    func testCoveragePercentRoundsCorrectly() {
        XCTAssertEqual(CourseTranslationsLogic.coveragePercent(translated: 1, total: 3), 33)
        XCTAssertEqual(CourseTranslationsLogic.coveragePercent(translated: 2, total: 2), 100)
    }

    func testMergeLocalesIncludesTracked() {
        let server = [
            TranslationCoverage(targetLocale: "es", totalItems: 10, translatedItems: 4, percent: 40),
        ]
        let merged = CourseTranslationsLogic.mergeLocales(server: server, tracked: ["fr", "es"])
        XCTAssertEqual(merged.map(\.targetLocale).sorted(), ["es", "fr"])
        let fr = merged.first { $0.targetLocale == "fr" }
        XCTAssertEqual(fr?.translatedItems, 0)
        XCTAssertEqual(fr?.totalItems, 10)
    }

    func testAvailableLocalesExcludesExisting() {
        let existing = [
            TranslationCoverage(targetLocale: "es", totalItems: 1, translatedItems: 0, percent: 0),
        ]
        let available = CourseTranslationsLogic.availableLocalesToAdd(existing: existing)
        XCTAssertFalse(available.contains { $0.tag == "es" })
        XCTAssertTrue(available.contains { $0.tag == "fr" })
    }

    func testGlossaryValidation() {
        XCTAssertEqual(
            CourseTranslationsLogic.validateGlossaryDraft(.init(sourceTerm: "", targetTerm: "x")),
            .sourceRequired
        )
        XCTAssertEqual(
            CourseTranslationsLogic.validateGlossaryDraft(.init(sourceTerm: "x", targetTerm: "  ")),
            .targetRequired
        )
        XCTAssertEqual(
            CourseTranslationsLogic.validateGlossaryDraft(.init(sourceTerm: "a", targetTerm: "b")),
            .ok
        )
    }

    func testGlossaryDiffDetectsChanges() {
        let existing = CourseGlossaryEntry(id: "1", sourceTerm: "term", targetTerm: "término")
        XCTAssertFalse(CourseTranslationsLogic.glossaryDiff(
            sourceTerm: "term",
            targetTerm: "término",
            existing: existing
        ))
        XCTAssertTrue(CourseTranslationsLogic.glossaryDiff(
            sourceTerm: "term",
            targetTerm: "palabra",
            existing: existing
        ))
        XCTAssertTrue(CourseTranslationsLogic.glossaryDiff(
            sourceTerm: "new",
            targetTerm: "nuevo",
            existing: nil
        ))
    }

    func testFilterAndPaginateGlossary() {
        let entries = (0 ..< 25).map {
            CourseGlossaryEntry(id: "\($0)", sourceTerm: "term-\($0)", targetTerm: "t-\($0)")
        }
        XCTAssertEqual(CourseTranslationsLogic.paginatedGlossary(entries, page: 0).count, 20)
        XCTAssertTrue(CourseTranslationsLogic.hasMoreGlossaryPages(entries: entries, page: 0))
        XCTAssertFalse(CourseTranslationsLogic.hasMoreGlossaryPages(entries: entries, page: 1))
        let filtered = CourseTranslationsLogic.filterGlossary(entries, query: "term-1")
        XCTAssertTrue(filtered.allSatisfy { $0.sourceTerm.contains("term-1") })
    }

    func testUpsertGlossaryEntryReplacesBySourceTerm() {
        let existing = [
            CourseGlossaryEntry(id: "1", sourceTerm: "Alpha", targetTerm: "A1"),
            CourseGlossaryEntry(id: "2", sourceTerm: "Beta", targetTerm: "B1"),
        ]
        let updated = CourseTranslationsLogic.upsertGlossaryEntry(
            CourseGlossaryEntry(id: "3", sourceTerm: "alpha", targetTerm: "A2"),
            into: existing
        )
        XCTAssertEqual(updated.count, 2)
        XCTAssertEqual(updated.first { $0.sourceTerm.lowercased() == "alpha" }?.targetTerm, "A2")
    }

    func testLocaleTagValidation() {
        XCTAssertTrue(CourseTranslationsLogic.isValidLocaleTag("es"))
        XCTAssertTrue(CourseTranslationsLogic.isValidLocaleTag("es-MX"))
        XCTAssertTrue(CourseTranslationsLogic.isValidLocaleTag("xx")) // shape only; server enforces allow-list
        XCTAssertFalse(CourseTranslationsLogic.isValidLocaleTag("ES"))
        XCTAssertFalse(CourseTranslationsLogic.isValidLocaleTag("e"))
        XCTAssertFalse(CourseTranslationsLogic.isValidLocaleTag(""))
    }

    func testIsRTLLocale() {
        XCTAssertTrue(CourseTranslationsLogic.isRTLLocale("ar"))
        XCTAssertTrue(CourseTranslationsLogic.isRTLLocale("he-IL"))
        XCTAssertFalse(CourseTranslationsLogic.isRTLLocale("es"))
    }

    func testStatusLabelKey() {
        let published = CourseTranslationListItem(
            itemId: "1", itemType: "content_page", title: "A", body: "", hasPublished: true
        )
        let draft = CourseTranslationListItem(
            itemId: "2", itemType: "content_page", title: "B", body: "", hasDraft: true
        )
        let missing = CourseTranslationListItem(
            itemId: "3", itemType: "content_page", title: "C", body: ""
        )
        XCTAssertEqual(
            CourseTranslationsLogic.statusLabelKey(for: published),
            "mobile.courseSettings.translations.status.published"
        )
        XCTAssertEqual(
            CourseTranslationsLogic.statusLabelKey(for: draft),
            "mobile.courseSettings.translations.status.draft"
        )
        XCTAssertEqual(
            CourseTranslationsLogic.statusLabelKey(for: missing),
            "mobile.courseSettings.translations.status.missing"
        )
    }

    func testTrackLocaleDedupes() {
        let once = CourseTranslationsLogic.trackLocale("es", into: [])
        let twice = CourseTranslationsLogic.trackLocale("es", into: once)
        XCTAssertEqual(once, ["es"])
        XCTAssertEqual(twice, ["es"])
    }
}
