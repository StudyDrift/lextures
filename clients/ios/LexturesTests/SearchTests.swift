import XCTest
@testable import Lextures

final class SearchTests: XCTestCase {
    func testShouldQueryRequiresTwoCharacters() {
        XCTAssertFalse(SearchQueryEngine.shouldQuery("a"))
        XCTAssertTrue(SearchQueryEngine.shouldQuery("ab"))
    }

    func testActionMatcherFindsCalendar() {
        let actions = [
            SearchListItem(
                id: "action:calendar",
                group: .action,
                title: "Open Calendar",
                subtitle: "Your schedule",
                path: "/calendar",
                haystack: "calendar schedule open calendar goto jump action"
            ),
        ]
        let matches = SearchActionRegistry.matchActions(query: "calendar", actions: actions)
        XCTAssertEqual(matches.count, 1)
        XCTAssertEqual(matches.first?.title, "Open Calendar")
    }

    func testRecentsCapAtTen() {
        let defaults = UserDefaults.standard
        defaults.removeObject(forKey: "mobile_search_recent_queries")
        defer { defaults.removeObject(forKey: "mobile_search_recent_queries") }

        for index in 0..<12 {
            SearchRecentsStore.recordSearch("query-\(index)")
        }
        XCTAssertEqual(SearchRecentsStore.recentSearches().count, SearchQueryEngine.maxRecents)
        XCTAssertEqual(SearchRecentsStore.recentSearches().first, "query-11")
    }

    func testPathNavigatorMapsCalendar() {
        let target = SearchPathNavigator.resolve("/calendar")
        XCTAssertEqual(target, .shellTab(.calendar))
    }

    func testPathNavigatorMapsCourseContent() {
        let target = SearchPathNavigator.resolve("/courses/demo/assignments/item-1")
        guard case .deepLink(let destination) = target else {
            return XCTFail("expected deep link")
        }
        guard case .course(let code, let section, let itemId) = destination else {
            return XCTFail("expected course deep link")
        }
        XCTAssertEqual(code, "demo")
        XCTAssertEqual(section, .modules)
        XCTAssertEqual(itemId, "item-1")
    }
}