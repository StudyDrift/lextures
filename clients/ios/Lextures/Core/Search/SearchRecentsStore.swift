import Foundation

/// Local recent searches and destinations (FR-5).
enum SearchRecentsStore {
    private static let searchesKey = "mobile_search_recent_queries"
    private static let destinationsKey = "mobile_search_recent_destinations"

    static func recentSearches() -> [String] {
        UserDefaults.standard.stringArray(forKey: searchesKey) ?? []
    }

    static func recentDestinations() -> [SearchListItem] {
        guard let data = UserDefaults.standard.data(forKey: destinationsKey) else { return [] }
        return (try? JSONDecoder().decode([SearchListItem].self, from: data)) ?? []
    }

    static func recordSearch(_ query: String) {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        var items = recentSearches().filter { $0.caseInsensitiveCompare(trimmed) != .orderedSame }
        items.insert(trimmed, at: 0)
        if items.count > SearchQueryEngine.maxRecents {
            items = Array(items.prefix(SearchQueryEngine.maxRecents))
        }
        UserDefaults.standard.set(items, forKey: searchesKey)
    }

    static func recordDestination(_ item: SearchListItem) {
        guard item.group != .recentSearch, item.group != .recentDestination else { return }
        var items = recentDestinations().filter { $0.id != item.id }
        items.insert(item, at: 0)
        if items.count > SearchQueryEngine.maxRecents {
            items = Array(items.prefix(SearchQueryEngine.maxRecents))
        }
        if let data = try? JSONEncoder().encode(items) {
            UserDefaults.standard.set(data, forKey: destinationsKey)
        }
    }

    static func clearAll() {
        UserDefaults.standard.removeObject(forKey: searchesKey)
        UserDefaults.standard.removeObject(forKey: destinationsKey)
    }
}