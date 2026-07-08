import Foundation

/// Learner profile helpers (LP10) — formatting, gating, and cache keys.
enum LearnerProfileLogic {
    static let facetPriority: [LearnerProfileFacetKey] = [
        "study_rhythm",
        "content_modality",
        "strengths_growth",
        "interests",
        "learning_approach",
    ]

    static func learnerProfileEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.learnerProfileEnabled && features.ffMobileLearnerProfile
    }

    static func cacheKeyProfile() -> String { "learner-profile:summary" }

    static func cacheKeyFacetEvidence(_ facetKey: LearnerProfileFacetKey) -> String {
        "learner-profile:evidence:\(facetKey)"
    }

    static func sortFacets(_ facets: [LearnerProfileFacetSummary]) -> [LearnerProfileFacetSummary] {
        let order = Dictionary(uniqueKeysWithValues: facetPriority.enumerated().map { ($1, $0) })
        return facets.sorted {
            (order[$0.facetKey] ?? 999) < (order[$1.facetKey] ?? 999)
        }
    }

    static func isPaused(_ profile: LearnerProfile) -> Bool {
        profile.status == "paused"
    }

    static func showEmptyState(_ profile: LearnerProfile) -> Bool {
        if profile.status == "insufficient_data" { return true }
        let facets = profile.facets
        return !facets.isEmpty && facets.allSatisfy { $0.state == "insufficient_data" }
    }

    enum ConfidenceLevel: String {
        case high, medium, low
    }

    static func confidenceLevel(_ score: Double) -> ConfidenceLevel {
        if score >= 0.75 { return .high }
        if score >= 0.45 { return .medium }
        return .low
    }

    static func confidenceLabelKey(_ score: Double) -> String {
        switch confidenceLevel(score) {
        case .high: return "mobile.learnerProfile.confidence.high"
        case .medium: return "mobile.learnerProfile.confidence.medium"
        case .low: return "mobile.learnerProfile.confidence.low"
        }
    }

    static func facetTitleKey(_ facetKey: LearnerProfileFacetKey) -> String {
        switch facetKey {
        case "study_rhythm": return "mobile.learnerProfile.facet.studyRhythm.title"
        case "content_modality": return "mobile.learnerProfile.facet.contentModality.title"
        case "strengths_growth": return "mobile.learnerProfile.facet.strengthsGrowth.title"
        case "interests": return "mobile.learnerProfile.facet.interests.title"
        case "learning_approach": return "mobile.learnerProfile.facet.learningApproach.title"
        default: return "mobile.learnerProfile.facet.generic.title"
        }
    }

    static func facetDescriptionKey(_ facetKey: LearnerProfileFacetKey) -> String {
        switch facetKey {
        case "study_rhythm": return "mobile.learnerProfile.facet.studyRhythm.description"
        case "content_modality": return "mobile.learnerProfile.facet.contentModality.description"
        case "strengths_growth": return "mobile.learnerProfile.facet.strengthsGrowth.description"
        case "interests": return "mobile.learnerProfile.facet.interests.description"
        case "learning_approach": return "mobile.learnerProfile.facet.learningApproach.description"
        default: return "mobile.learnerProfile.facet.generic.description"
        }
    }

    static func insightLabelKey(_ insightKey: String) -> String {
        switch insightKey {
        case "peak_study_window": return "mobile.learnerProfile.insight.peakStudyWindow"
        case "study_consistency": return "mobile.learnerProfile.insight.studyConsistency"
        case "study_streak": return "mobile.learnerProfile.insight.studyStreak"
        case "session_shape": return "mobile.learnerProfile.insight.sessionShape"
        case "modality_affinity": return "mobile.learnerProfile.insight.modalityAffinity"
        case "complexity_comfort": return "mobile.learnerProfile.insight.complexityComfort"
        case "content_pacing": return "mobile.learnerProfile.insight.contentPacing"
        case "top_strengths": return "mobile.learnerProfile.insight.topStrengths"
        case "growth_areas": return "mobile.learnerProfile.insight.growthAreas"
        case "needs_review": return "mobile.learnerProfile.insight.needsReview"
        case "persistence": return "mobile.learnerProfile.insight.persistenceLabel"
        case "help_seeking": return "mobile.learnerProfile.insight.helpSeekingLabel"
        case "consolidation": return "mobile.learnerProfile.insight.consolidationLabel"
        default:
            if insightKey.hasPrefix("topic_") {
                return "mobile.learnerProfile.insight.topic"
            }
            return "mobile.learnerProfile.insight.generic"
        }
    }

    static func sourceKindLabel(_ kind: String) -> String {
        let key = "mobile.learnerProfile.evidence.source.\(kind)"
        let translated = L.dynamicText(key)
        return translated == key ? L.dynamicText("mobile.learnerProfile.evidence.source.generic") : translated
    }

    static func totalObservationCount(_ evidence: [LearnerProfileEvidenceRow]) -> Int {
        evidence.reduce(0) { $0 + $1.observationCount }
    }

    static func uniqueCourseCount(_ evidence: [LearnerProfileEvidenceRow]) -> Int {
        Set(evidence.compactMap(\.courseId).filter { !$0.isEmpty }).count
    }

    static func derivedFromSummary(count: Int, courses: Int) -> String {
        let obs = L.plural("mobile.learnerProfile.evidence.observationCount", count: count)
        if courses <= 0 {
            return L.format("mobile.learnerProfile.evidence.derivedFromNoCourses", obs)
        }
        let coursePart = L.plural("mobile.learnerProfile.evidence.courseCount", count: courses)
        return L.format("mobile.learnerProfile.evidence.derivedFrom", obs, coursePart)
    }

    static func formatInsightValue(_ insight: LearnerProfileInsight, facetKey: LearnerProfileFacetKey) -> String {
        let value = insight.value
        switch insight.insightKey {
        case "peak_study_window":
            guard let peakWindows = value["peakWindows"],
                  case .array(let windows) = peakWindows,
                  let first = windows.first,
                  case .object(let top) = first,
                  let dowValue = top["dow"],
                  case .string(let dow) = dowValue,
                  let hourValue = top["hourBucket"],
                  case .string(let hour) = hourValue,
                  let shareValue = top["share"],
                  case .number(let share) = shareValue else {
                return L.dynamicText("mobile.learnerProfile.insight.peakUnknown")
            }
            return L.format("mobile.learnerProfile.insight.peak", dow, hour, Int((share * 100).rounded()))
        case "study_consistency":
            let score = jsonInt(value["consistencyScore"], percent: true)
            let days = jsonDoubleString(value["activeDaysPerWeek"], digits: 1) ?? "0"
            return L.format("mobile.learnerProfile.insight.consistency", score, days)
        case "study_streak":
            let current = jsonInt(value["currentStreakDays"])
            let longest = jsonInt(value["longestStreakDays"])
            return L.format("mobile.learnerProfile.insight.streak", current, longest)
        case "session_shape":
            let minutes = jsonInt(value["medianSessionMin"])
            let perWeek = jsonDoubleString(value["sessionsPerActiveWeek"], digits: 1) ?? "0"
            return L.format("mobile.learnerProfile.insight.session", minutes, perWeek)
        case "modality_affinity":
            guard let modalityRaw = value["modalityAffinity"],
                  case .object(let affinity) = modalityRaw,
                  let top = affinity.max(by: { jsonNumber($0.value) < jsonNumber($1.value) }) else {
                return L.dynamicText("mobile.learnerProfile.insight.genericUnknown")
            }
            return L.format(
                "mobile.learnerProfile.insight.modalityTop",
                top.key,
                Int((jsonNumber(top.value) * 100).rounded())
            )
        case "complexity_comfort":
            guard let bandRaw = value["complexityComfort"],
                  case .object(let band) = bandRaw,
                  let lowValue = band["low"],
                  case .string(let low) = lowValue,
                  let highValue = band["high"],
                  case .string(let high) = highValue else {
                return L.dynamicText("mobile.learnerProfile.insight.genericUnknown")
            }
            return L.format("mobile.learnerProfile.insight.comfort", low, high)
        case "content_pacing":
            let pacing = jsonString(value["pacing"]) ?? "unknown"
            return L.format("mobile.learnerProfile.insight.pacing", pacing)
        case "top_strengths":
            return conceptList(value["strengths"], key: "concept", formatKey: "mobile.learnerProfile.insight.strengthsList")
        case "growth_areas":
            return conceptList(value["growth"], keys: ["concept", "misconception"], formatKey: "mobile.learnerProfile.insight.growthList")
        case "needs_review":
            return conceptList(value["needsReview"], key: "concept", formatKey: "mobile.learnerProfile.insight.reviewList")
        case "persistence":
            let level = jsonString(value["level"]) ?? "unknown"
            let productive = if case .bool(true)? = value["productive"] { true } else { false }
            return L.format(
                "mobile.learnerProfile.insight.persistence",
                level,
                productive ? L.dynamicText("mobile.learnerProfile.insight.productiveRetakes") : ""
            )
        case "help_seeking":
            let style = jsonString(value["style"]) ?? "unknown"
            let hints = jsonDoubleString(value["hintsPerAttempt"], digits: 1) ?? "0"
            return L.format("mobile.learnerProfile.insight.helpSeeking", style, hints)
        case "consolidation":
            let level = jsonString(value["level"]) ?? "unknown"
            let actions = jsonInt(value["notebookActions"])
            return L.format("mobile.learnerProfile.insight.consolidation", level, actions)
        default:
            if insight.insightKey.hasPrefix("topic_"),
               let topicValue = value["topic"],
               case .string(let topic) = topicValue {
                let affinity = jsonInt(value["affinity"], percent: true)
                return L.format("mobile.learnerProfile.insight.interestTop", topic, affinity)
            }
            if !insight.label.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                return insight.label
            }
            return L.dynamicText("mobile.learnerProfile.insight.genericUnknown")
        }
    }

    static func rhythmChartCaption(_ summary: [String: JSONValue]) -> String? {
        guard let windowsRaw = summary["peakWindows"],
              case .array(let windows) = windowsRaw,
              !windows.isEmpty else { return nil }
        var lines: [String] = [L.text("mobile.learnerProfile.chart.rhythmCaption")]
        for window in windows.prefix(3) {
            guard case .object(let row) = window,
                  let dowValue = row["dow"],
                  case .string(let dow) = dowValue,
                  let hourValue = row["hourBucket"],
                  case .string(let hour) = hourValue,
                  let shareValue = row["share"],
                  case .number(let share) = shareValue else { continue }
            lines.append(L.format("mobile.learnerProfile.chart.rhythmRow", dow, hour, Int((share * 100).rounded())))
        }
        return lines.joined(separator: "\n")
    }

    static func modalityChartCaption(_ summary: [String: JSONValue]) -> String? {
        guard let affinityRaw = summary["modalityAffinity"],
              case .object(let affinity) = affinityRaw,
              !affinity.isEmpty else { return nil }
        var lines: [String] = [L.text("mobile.learnerProfile.chart.modalityCaption")]
        for (modality, share) in affinity.sorted(by: { jsonNumber($0.value) > jsonNumber($1.value) }).prefix(4) {
            lines.append(L.format("mobile.learnerProfile.chart.modalityRow", modality, Int((jsonNumber(share) * 100).rounded())))
        }
        return lines.joined(separator: "\n")
    }

    private static func conceptList(
        _ raw: JSONValue?,
        key: String,
        formatKey: String
    ) -> String {
        conceptList(raw, keys: [key], formatKey: formatKey)
    }

    private static func conceptList(
        _ raw: JSONValue?,
        keys: [String],
        formatKey: String
    ) -> String {
        guard let itemsRaw = raw, case .array(let items) = itemsRaw else {
            return L.dynamicText("mobile.learnerProfile.insight.genericUnknown")
        }
        let names: [String] = items.prefix(3).compactMap { item in
            guard case .object(let object) = item else { return nil }
            for key in keys {
                if let nameValue = object[key], case .string(let name) = nameValue, !name.isEmpty { return name }
            }
            return nil
        }
        guard !names.isEmpty else { return L.dynamicText("mobile.learnerProfile.insight.genericUnknown") }
        return L.format(String.LocalizationValue(stringLiteral: formatKey), names.joined(separator: ", "))
    }

    private static func jsonString(_ value: JSONValue?) -> String? {
        guard case .string(let string)? = value else { return nil }
        return string
    }

    private static func jsonNumber(_ value: JSONValue) -> Double {
        if case .number(let number) = value { return number }
        if case .string(let string) = value, let number = Double(string) { return number }
        return 0
    }

    private static func jsonInt(_ value: JSONValue?, percent: Bool = false) -> Int {
        let number = value.map(jsonNumber) ?? 0
        return Int((percent ? number * 100 : number).rounded())
    }

    private static func jsonDoubleString(_ value: JSONValue?, digits: Int) -> String? {
        guard let value else { return nil }
        let number = jsonNumber(value)
        return String(format: "%.\(digits)f", number)
    }
}