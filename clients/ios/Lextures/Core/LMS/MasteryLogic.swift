import Foundation

enum MasteryLevel: String, Hashable {
    case mastered
    case developing
    case beginning
    case atRisk
    case notAssessed

    var labelKey: String {
        switch self {
        case .mastered: return "mobile.mastery.levelMastered"
        case .developing: return "mobile.mastery.levelDeveloping"
        case .beginning: return "mobile.mastery.levelBeginning"
        case .atRisk: return "mobile.mastery.levelAtRisk"
        case .notAssessed: return "mobile.mastery.levelNotAssessed"
        }
    }
}

struct MasteryConceptRow: Identifiable, Hashable {
    var id: String
    var name: String
    var score: Double?
    var assessed: Bool
    var level: MasteryLevel
}

enum MasteryLogic {
    /// Matches web `masteryLabel()` thresholds (0.8 / 0.6 / 0.4).
    static func level(score: Double?, assessed: Bool) -> MasteryLevel {
        guard assessed, let score else { return .notAssessed }
        if score >= 0.8 { return .mastered }
        if score >= 0.6 { return .developing }
        if score >= 0.4 { return .beginning }
        return .atRisk
    }

    static func rows(from row: StudentMasteryRow) -> [MasteryConceptRow] {
        let cellsByConcept = Dictionary(uniqueKeysWithValues: row.cells.map { ($0.conceptId, $0) })
        return row.concepts.map { concept in
            let cell = cellsByConcept[concept.id]
            return MasteryConceptRow(
                id: concept.id,
                name: concept.name,
                score: cell?.masteryScore,
                assessed: cell?.assessed ?? false,
                level: level(score: cell?.masteryScore, assessed: cell?.assessed ?? false)
            )
        }
        .sorted { lhs, rhs in
            if lhs.assessed != rhs.assessed { return !lhs.assessed && rhs.assessed }
            return (lhs.score ?? 0) < (rhs.score ?? 0)
        }
    }

    static func summary(_ rows: [MasteryConceptRow]) -> (mastered: Int, atRisk: Int, total: Int) {
        let assessed = rows.filter(\.assessed)
        return (
            mastered: assessed.filter { $0.level == .mastered }.count,
            atRisk: assessed.filter { $0.level == .atRisk }.count,
            total: rows.count
        )
    }

    static func cacheKeyMastery(courseCode: String, enrollmentId: String) -> String {
        "mastery:\(courseCode):\(enrollmentId)"
    }

    static func cacheKeyMyReportCards() -> String {
        "mastery:my-report-cards"
    }

    static func releasedReportCards(_ cards: [ReportCardSummary]) -> [ReportCardSummary] {
        cards.filter { $0.status == "released" }
            .sorted { $0.gradingPeriod > $1.gradingPeriod }
    }
}
