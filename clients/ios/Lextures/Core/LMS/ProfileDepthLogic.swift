import Foundation

/// Validation and display helpers for profile depth (M1.5).
enum ProfileDepthLogic {
    static let preferNotToSayRaceCode = "unknown"

    static let raceEthnicityOptions: [(code: String, labelKey: String)] = [
        ("1", "mobile.profileDepth.race.hispanic"),
        ("2", "mobile.profileDepth.race.americanIndian"),
        ("3", "mobile.profileDepth.race.asian"),
        ("4", "mobile.profileDepth.race.black"),
        ("5", "mobile.profileDepth.race.pacificIslander"),
        ("6", "mobile.profileDepth.race.white"),
        ("7", "mobile.profileDepth.race.twoOrMore"),
        (preferNotToSayRaceCode, "mobile.profileDepth.preferNotToSay"),
    ]

    static let demographicsFieldKeys: [String] = [
        "raceEthnicityCode",
        "freeLunch",
        "reducedLunch",
        "ellStatus",
        "disabilityStatus",
        "homelessIndicator",
        "migrantIndicator",
    ]

    static func demographicsLabelKey(for key: String) -> String {
        "mobile.profileDepth.demographics.\(key)"
    }

    static func validateCustomFields(
        definitions: [ProfileFieldDefinition],
        draft: [String: String]
    ) -> [String: String] {
        var errors: [String: String] = [:]
        for def in definitions {
            let raw = draft[def.key]?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            if def.isRequired && raw.isEmpty {
                errors[def.key] = L.text("mobile.profileDepth.requiredField")
                continue
            }
            guard !raw.isEmpty else { continue }
            switch def.fieldType {
            case "number":
                if Double(raw) == nil {
                    errors[def.key] = L.text("mobile.profileDepth.invalidNumber")
                }
            case "date":
                if !isValidISODate(raw) {
                    errors[def.key] = L.text("mobile.profileDepth.invalidDate")
                }
            case "select":
                let options = def.selectOptions ?? []
                if !options.isEmpty && !options.contains(raw) {
                    errors[def.key] = L.text("mobile.profileDepth.invalidSelect")
                }
            case "boolean":
                if !["true", "false"].contains(raw.lowercased()) {
                    errors[def.key] = L.text("mobile.profileDepth.invalidBoolean")
                }
            default:
                break
            }
        }
        return errors
    }

    static func encodeCustomFieldValues(
        definitions: [ProfileFieldDefinition],
        draft: [String: String]
    ) -> [String: JSONValue] {
        var out: [String: JSONValue] = [:]
        for def in definitions {
            let raw = draft[def.key]?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            if raw.isEmpty {
                out[def.key] = .null
                continue
            }
            switch def.fieldType {
            case "number":
                if let numberValue = Double(raw) { out[def.key] = .number(numberValue) } else { out[def.key] = .null }
            case "boolean":
                out[def.key] = .bool(raw.lowercased() == "true")
            default:
                out[def.key] = .string(raw)
            }
        }
        return out
    }

    static func displayValue(for def: ProfileFieldDefinition, value: JSONValue?) -> String {
        guard let value, !value.isEmpty else { return L.text("mobile.emDash") }
        switch (def.fieldType, value) {
        case ("boolean", .bool(let boolValue)):
            return boolValue ? L.text("mobile.profileDepth.yes") : L.text("mobile.profileDepth.no")
        case (_, .string(let stringValue)):
            return stringValue
        case (_, .number(let numberValue)):
            if numberValue.rounded(.towardZero) == numberValue { return String(Int(numberValue)) }
            return String(numberValue)
        case (_, .bool(let boolValue)):
            return boolValue ? L.text("mobile.profileDepth.yes") : L.text("mobile.profileDepth.no")
        default:
            return L.text("mobile.emDash")
        }
    }

    static func draftFromValues(
        definitions: [ProfileFieldDefinition],
        values: [String: JSONValue]
    ) -> [String: String] {
        var draft: [String: String] = [:]
        for def in definitions {
            guard let value = values[def.key] else { continue }
            switch value {
            case .string(let stringValue): draft[def.key] = stringValue
            case .bool(let boolValue): draft[def.key] = boolValue ? "true" : "false"
            case .number(let numberValue):
                if numberValue.rounded(.towardZero) == numberValue { draft[def.key] = String(Int(numberValue)) }
                else { draft[def.key] = String(numberValue) }
            case .null: break
            }
        }
        return draft
    }

    static func triStateBoolLabel(_ value: Bool?) -> String {
        guard let value else { return L.text("mobile.profileDepth.preferNotToSay") }
        return value ? L.text("mobile.profileDepth.yes") : L.text("mobile.profileDepth.no")
    }

    static func parseTriStateBool(_ raw: String) -> Bool? {
        switch raw.lowercased() {
        case "true", "yes": return true
        case "false", "no": return false
        default: return nil
        }
    }

    static func consentDecisionLabel(_ decision: ConsentDecision) -> String {
        switch decision {
        case .granted: return L.text("mobile.profileDepth.consent.enrolled")
        case .declined: return L.text("mobile.profileDepth.consent.declined")
        case .withdrawn: return L.text("mobile.profileDepth.consent.withdrawn")
        }
    }

    static func latestConsentByStudy(_ history: [ConsentHistoryEntry]) -> [ConsentHistoryEntry] {
        var seen = Set<String>()
        var out: [ConsentHistoryEntry] = []
        for entry in history {
            if seen.contains(entry.studyId) { continue }
            seen.insert(entry.studyId)
            out.append(entry)
        }
        return out
    }

    static func shouldShowPersonalDetails(
        customFieldsEnabled: Bool,
        demographicsEnabled: Bool,
        fieldCount: Int
    ) -> Bool {
        if fieldCount > 0 { return true }
        return demographicsEnabled
    }

    static func shouldShowResearchStudies(
        researchConsentEnabled: Bool,
        pendingCount: Int,
        historyCount: Int
    ) -> Bool {
        guard researchConsentEnabled else { return false }
        return pendingCount > 0 || historyCount > 0
    }

    private static func isValidISODate(_ raw: String) -> Bool {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count == 10 else { return false }
        let parts = trimmed.split(separator: "-")
        guard parts.count == 3,
              let year = Int(parts[0]), let month = Int(parts[1]), let day = Int(parts[2]) else { return false }
        var comps = DateComponents()
        comps.year = year
        comps.month = month
        comps.day = day
        return Calendar.current.date(from: comps) != nil
    }
}