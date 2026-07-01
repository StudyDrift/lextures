import XCTest
@testable import Lextures

final class MasteryLogicTests: XCTestCase {
    func testLevelThresholds() {
        XCTAssertEqual(MasteryLogic.level(score: 0.9, assessed: true), .mastered)
        XCTAssertEqual(MasteryLogic.level(score: 0.8, assessed: true), .mastered)
        XCTAssertEqual(MasteryLogic.level(score: 0.7, assessed: true), .developing)
        XCTAssertEqual(MasteryLogic.level(score: 0.6, assessed: true), .developing)
        XCTAssertEqual(MasteryLogic.level(score: 0.5, assessed: true), .beginning)
        XCTAssertEqual(MasteryLogic.level(score: 0.4, assessed: true), .beginning)
        XCTAssertEqual(MasteryLogic.level(score: 0.1, assessed: true), .atRisk)
        XCTAssertEqual(MasteryLogic.level(score: 0.9, assessed: false), .notAssessed)
        XCTAssertEqual(MasteryLogic.level(score: nil, assessed: true), .notAssessed)
    }

    func testRowsJoinsConceptsAndCellsAndSortsUnassessedFirst() {
        let row = StudentMasteryRow(
            enrollmentId: "e1",
            userId: "u1",
            concepts: [
                MasteryConcept(id: "c1", name: "Fractions"),
                MasteryConcept(id: "c2", name: "Decimals"),
                MasteryConcept(id: "c3", name: "Ratios"),
            ],
            cells: [
                MasteryCell(conceptId: "c1", masteryScore: 0.9, assessed: true, updatedAt: nil),
                MasteryCell(conceptId: "c2", masteryScore: 0.3, assessed: true, updatedAt: nil),
            ]
        )
        let rows = MasteryLogic.rows(from: row)
        XCTAssertEqual(rows.count, 3)
        // Not-assessed concept (c3) sorts first, then lowest-scoring assessed (c2), then c1.
        XCTAssertEqual(rows.map(\.id), ["c3", "c2", "c1"])
        XCTAssertEqual(rows[0].level, .notAssessed)
        XCTAssertEqual(rows[1].level, .atRisk)
        XCTAssertEqual(rows[2].level, .mastered)
    }

    func testSummaryCountsMasteredAndAtRisk() {
        let rows = [
            MasteryConceptRow(id: "1", name: "A", score: 0.9, assessed: true, level: .mastered),
            MasteryConceptRow(id: "2", name: "B", score: 0.85, assessed: true, level: .mastered),
            MasteryConceptRow(id: "3", name: "C", score: 0.1, assessed: true, level: .atRisk),
            MasteryConceptRow(id: "4", name: "D", score: nil, assessed: false, level: .notAssessed),
        ]
        let summary = MasteryLogic.summary(rows)
        XCTAssertEqual(summary.mastered, 2)
        XCTAssertEqual(summary.atRisk, 1)
        XCTAssertEqual(summary.total, 4)
    }

    func testReleasedReportCardsFiltersAndSortsDescending() {
        let cards = [
            ReportCardSummary(
                id: "1", studentId: "s", courseId: "c", gradingPeriod: "Q1",
                status: "draft", finalGradePct: nil, letterGrade: nil, comment: nil,
                pdfUrl: nil, generatedAt: nil, releasedAt: nil, createdAt: nil
            ),
            ReportCardSummary(
                id: "2", studentId: "s", courseId: "c", gradingPeriod: "Q2",
                status: "released", finalGradePct: nil, letterGrade: nil, comment: nil,
                pdfUrl: nil, generatedAt: nil, releasedAt: nil, createdAt: nil
            ),
            ReportCardSummary(
                id: "3", studentId: "s", courseId: "c", gradingPeriod: "Q1",
                status: "released", finalGradePct: nil, letterGrade: nil, comment: nil,
                pdfUrl: nil, generatedAt: nil, releasedAt: nil, createdAt: nil
            ),
        ]
        let released = MasteryLogic.releasedReportCards(cards)
        XCTAssertEqual(released.map(\.id), ["2", "3"])
    }
}
