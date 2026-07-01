import XCTest
@testable import Lextures

final class LibraryResourceLogicTests: XCTestCase {
    func testLibraryItemsFilter() {
        let items = [
            CourseStructureItem(id: "1", sortOrder: 0, kind: "library_resource", title: "Reading", parentId: nil, published: true),
            CourseStructureItem(id: "2", sortOrder: 1, kind: "quiz", title: "Quiz", parentId: nil, published: true),
        ]
        XCTAssertEqual(LibraryResourceLogic.libraryItems(from: items).count, 1)
        XCTAssertTrue(LibraryResourceLogic.hasLibraryResources(in: items))
    }

    func testResolveAccessUsesEzproxyUrl() {
        let payload = LibraryResourcePayload(
            itemId: "abc",
            resourceType: "catalog_item",
            metadata: nil,
            ezproxyUrl: "https://ezproxy.example.edu/login?url=https://publisher.example/book",
            updatedAt: nil
        )
        let state = LibraryResourceLogic.resolveAccess(payload: payload)
        guard case .ready(let url) = state else {
            return XCTFail("Expected ready state")
        }
        XCTAssertTrue(url.contains("ezproxy.example.edu"))
    }

    func testResolveAccessLegantoGatedWithoutUrl() {
        let payload = LibraryResourcePayload(
            itemId: "abc",
            resourceType: "leganto_list",
            metadata: LibraryResourceMeta(legantoListId: "list-1"),
            ezproxyUrl: nil,
            updatedAt: nil
        )
        let state = LibraryResourceLogic.resolveAccess(payload: payload)
        guard case .gated(let key) = state else {
            return XCTFail("Expected gated state")
        }
        XCTAssertEqual(key, "mobile.library.legantoGated")
    }

    func testDefaultOerProviderPrefersCommons() {
        XCTAssertEqual(
            LibraryResourceLogic.defaultOERProvider(from: ["merlot", "oer_commons"]),
            "oer_commons"
        )
    }

    func testAccessEventPath() {
        XCTAssertEqual(
            LibraryResourceLogic.accessEventPath(courseCode: "demo-101", itemId: "item-1"),
            "/api/v1/courses/demo-101/library-resources/item-1/access"
        )
    }
}