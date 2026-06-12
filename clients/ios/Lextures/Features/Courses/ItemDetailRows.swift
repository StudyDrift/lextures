import Foundation

/// Detail-preview rows for assignment, quiz, and external-link activities.
enum ItemDetailRows {
    static func rows(
        for item: CourseStructureItem,
        detail: ModuleItemDetail?,
        pointsValue: Int?
    ) -> [(String, String)] {
        var rows: [(String, String)] = []
        let due = LMSDates.parse(detail?.dueAt ?? item.dueAt)
        if let due {
            rows.append(("Due date", due.formatted(date: .abbreviated, time: .shortened)))
        }

        switch item.kind {
        case "quiz":
            if let detail {
                rows.append(("Unlimited attempts", yesNo(detail.unlimitedAttempts ?? false)))
                rows.append(("One question at a time", yesNo(detail.oneQuestionAtATime ?? false)))
                rows.append(("Course lockdown feature", lockdownLabel(detail.lockdownMode)))
                rows.append(("Delivery mode", titlecase(detail.adaptiveDeliveryMode ?? "standard")))
                if detail.unlimitedAttempts != true {
                    rows.append(("Max attempts", "\(detail.maxAttempts ?? 1)"))
                }
                rows.append(("Grade uses", gradePolicyLabel(detail.gradeAttemptPolicy)))
                if let limit = detail.timeLimitMinutes {
                    rows.append(("Time limit", "\(limit) min"))
                }
                if let passing = detail.passingScorePercent {
                    rows.append(("Passing score", "\(passing)%"))
                }
                if let points = pointsValue {
                    rows.append(("Points", "\(points)"))
                }
                rows.append(("Shuffle questions", yesNo(detail.shuffleQuestions ?? false)))
            }
        case "assignment":
            if let points = pointsValue {
                rows.append(("Points", "\(points)"))
            }
            if let detail {
                let types = [
                    (detail.submissionAllowText ?? false) ? "Text" : nil,
                    (detail.submissionAllowFileUpload ?? false) ? "File upload" : nil,
                    (detail.submissionAllowUrl ?? false) ? "URL" : nil,
                ].compactMap(\.self)
                if !types.isEmpty {
                    rows.append(("Submission types", types.joined(separator: ", ")))
                }
                if let policy = detail.lateSubmissionPolicy {
                    rows.append(("Late submissions", lateLabel(policy, penalty: detail.latePenaltyPercent)))
                }
                if let from = LMSDates.parse(detail.availableFrom) {
                    rows.append(("Available from", from.formatted(date: .abbreviated, time: .shortened)))
                }
                if let until = LMSDates.parse(detail.availableUntil) {
                    rows.append(("Available until", until.formatted(date: .abbreviated, time: .shortened)))
                }
            }
        case "external_link":
            if let provider = detail?.provider, !provider.isEmpty {
                rows.append(("Provider", titlecase(provider)))
            }
        default:
            if let points = pointsValue {
                rows.append(("Points", "\(points)"))
            }
        }

        if let updated = LMSDates.parse(detail?.updatedAt) {
            rows.append(("Updated", updated.formatted(date: .abbreviated, time: .omitted)))
        }
        return rows
    }

    private static func yesNo(_ value: Bool) -> String { value ? "Yes" : "No" }

    private static func lockdownLabel(_ mode: String?) -> String {
        guard let mode, !mode.isEmpty, mode != "off", mode != "none" else { return "Off" }
        return titlecase(mode)
    }

    private static func gradePolicyLabel(_ policy: String?) -> String {
        switch policy {
        case "highest": return "Highest attempt"
        case "latest": return "Latest attempt"
        case "first": return "First attempt"
        case "average": return "Average of attempts"
        default: return titlecase(policy ?? "Latest attempt")
        }
    }

    private static func lateLabel(_ policy: String, penalty: Int?) -> String {
        switch policy {
        case "allow": return "Allowed"
        case "block", "reject": return "Not allowed"
        case "penalty":
            if let penalty { return "Allowed, −\(penalty)%" }
            return "Allowed with penalty"
        default: return titlecase(policy)
        }
    }

    private static func titlecase(_ raw: String) -> String {
        let cleaned = raw.replacingOccurrences(of: "_", with: " ")
        guard let first = cleaned.first else { return cleaned }
        return first.uppercased() + cleaned.dropFirst()
    }
}
