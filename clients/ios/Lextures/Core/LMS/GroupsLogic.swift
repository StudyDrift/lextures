import Foundation

struct GroupMemberRow: Identifiable, Equatable {
    let id: String
    let displayName: String
    let email: String
}

enum GroupSpaceTab: String, CaseIterable, Identifiable {
    case members
    case discussion
    case files
    case docs

    var id: String { rawValue }

    var labelKey: String {
        switch self {
        case .members: return "mobile.groups.tab.members"
        case .discussion: return "mobile.groups.tab.discussion"
        case .files: return "mobile.groups.tab.files"
        case .docs: return "mobile.groups.tab.docs"
        }
    }
}

enum GroupsLogic {
    static func sortedGroups(_ groups: [GroupPublic]) -> [GroupPublic] {
        groups.sorted { lhs, rhs in
            if lhs.sortOrder != rhs.sortOrder { return lhs.sortOrder < rhs.sortOrder }
            return lhs.name.localizedCaseInsensitiveCompare(rhs.name) == .orderedAscending
        }
    }

    static func courseCollabDocs(_ docs: [CollabDoc]) -> [CollabDoc] {
        docs.filter { $0.groupId == nil || $0.groupId?.isEmpty == true }
    }

    static func groupCollabDocs(_ docs: [CollabDoc], groupId: String) -> [CollabDoc] {
        docs.filter { $0.groupId == groupId }
    }

    static func collabDocWebPath(courseCode: String, docId: String) -> String {
        "/courses/\(LMSAPI.encodePath(courseCode))/collab-docs/\(LMSAPI.encodePath(docId))"
    }

    static func memberRows(
        roster: [FeedRosterPerson],
        messageAuthorIds: Set<String>
    ) -> [GroupMemberRow] {
        let rosterById = Dictionary(uniqueKeysWithValues: roster.map { ($0.userId, $0) })
        var seen = Set<String>()
        var rows: [GroupMemberRow] = []

        for person in roster where messageAuthorIds.contains(person.userId) {
            guard seen.insert(person.userId).inserted else { continue }
            rows.append(
                GroupMemberRow(
                    id: person.userId,
                    displayName: person.displayName?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
                        ? person.displayName!
                        : person.email,
                    email: person.email
                )
            )
        }

        for authorId in messageAuthorIds where !seen.contains(authorId) {
            if let person = rosterById[authorId] {
                rows.append(
                    GroupMemberRow(
                        id: person.userId,
                        displayName: person.displayName?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
                            ? person.displayName!
                            : person.email,
                        email: person.email
                    )
                )
            }
        }

        return rows.sorted { $0.displayName.localizedCaseInsensitiveCompare($1.displayName) == .orderedAscending }
    }

    static func collectMessageAuthorIds(from messages: [FeedMessage]) -> Set<String> {
        var ids = Set<String>()
        func walk(_ message: FeedMessage) {
            ids.insert(message.authorUserId)
            message.replies.forEach(walk)
        }
        messages.forEach(walk)
        return ids
    }

    static func displayInitials(_ label: String) -> String {
        let trimmed = label.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return "?" }
        let parts = trimmed.split(separator: " ").filter { !$0.isEmpty }
        if parts.count >= 2 {
            let first = parts[0].prefix(1)
            let second = parts[1].prefix(1)
            return "\(first)\(second)".uppercased()
        }
        return String(trimmed.prefix(2)).uppercased()
    }

    static func avatarHue(seed: String) -> Double {
        var hash = 0
        for scalar in seed.lowercased().unicodeScalars {
            hash = (hash &* 31 &+ Int(scalar.value)) | 0
        }
        return Double(abs(hash) % 360)
    }
}