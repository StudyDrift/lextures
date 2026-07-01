import Foundation

enum SearchResultGroup: String, Codable, CaseIterable, Equatable {
    case recentSearch
    case recentDestination
    case action
    case course
    case content
    case person

    var label: String {
        switch self {
        case .recentSearch: return L.text("mobile.search.group.recentSearches")
        case .recentDestination: return L.text("mobile.search.group.recentDestinations")
        case .action: return L.text("mobile.search.group.actions")
        case .course: return L.text("mobile.search.group.courses")
        case .content: return L.text("mobile.search.group.content")
        case .person: return L.text("mobile.search.group.people")
        }
    }
}

struct SearchListItem: Identifiable, Equatable, Codable, Hashable {
    var id: String
    var group: SearchResultGroup
    var title: String
    var subtitle: String
    var path: String
    var haystack: String

    init(
        id: String,
        group: SearchResultGroup,
        title: String,
        subtitle: String,
        path: String,
        haystack: String? = nil
    ) {
        self.id = id
        self.group = group
        self.title = title
        self.subtitle = subtitle
        self.path = path
        self.haystack = haystack ?? "\(title) \(subtitle) \(group.rawValue)".lowercased()
    }
}

struct SearchResultSection: Identifiable, Equatable {
    var id: String { group.rawValue }
    var group: SearchResultGroup
    var items: [SearchListItem]
}

enum SearchQueryEngine {
    static let debounceMilliseconds = 280
    static let minQueryLength = 2
    static let maxRecents = 10

    static func shouldQuery(_ query: String) -> Bool {
        query.trimmingCharacters(in: .whitespacesAndNewlines).count >= minQueryLength
    }
}