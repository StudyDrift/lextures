import XCTest
@testable import Lextures

final class BoardsAdvancedLogicTests: XCTestCase {
    override func setUp() {
        super.setUp()
        BoardsAdvancedObservability.resetForTests()
    }

    func testAdvancedGatingRequiresFlagAndCourse() {
        var features = MobilePlatformFeatures()
        features.ffMobileBoardsAdvanced = false
        XCTAssertFalse(BoardsAdvancedLogic.isAdvancedEnabled(courseEnabled: true, features: features))
        features.ffMobileBoardsAdvanced = true
        XCTAssertTrue(BoardsAdvancedLogic.isAdvancedEnabled(courseEnabled: true, features: features))
        XCTAssertFalse(BoardsAdvancedLogic.isAdvancedEnabled(courseEnabled: false, features: features))
        XCTAssertTrue(BoardsAdvancedLogic.canUseTemplates(courseEnabled: true, features: features, canCreate: true))
        XCTAssertFalse(BoardsAdvancedLogic.canUseTemplates(courseEnabled: true, features: features, canCreate: false))
        XCTAssertTrue(BoardsAdvancedLogic.canExportOrPresent(courseEnabled: true, features: features, canManage: true))
        XCTAssertFalse(BoardsAdvancedLogic.canViewBoardAnalytics(courseEnabled: true, features: features, canManage: false))
    }

    func testFilterTemplatesByScopeAndQuery() {
        let templates = [
            BoardTemplate(id: "1", scope: "builtin", title: "KWL Chart", description: "Know Want Learned", tags: ["kwl"]),
            BoardTemplate(id: "2", scope: "course", title: "Exit ticket", description: "Quick check", tags: []),
            BoardTemplate(id: "3", scope: "org", title: "Brainstorm", description: "Ideas wall", tags: ["ideas"]),
        ]
        XCTAssertEqual(BoardsAdvancedLogic.filterTemplates(templates, scope: .builtin, query: "").map(\.id), ["1"])
        XCTAssertEqual(BoardsAdvancedLogic.filterTemplates(templates, scope: nil, query: "exit").map(\.id), ["2"])
        XCTAssertEqual(BoardsAdvancedLogic.filterTemplates(templates, scope: .org, query: "ideas").map(\.id), ["3"])
    }

    func testPollDelayBackoffAndTerminal() {
        XCTAssertEqual(BoardsAdvancedLogic.pollDelaySeconds(attempt: 0), 0.5, accuracy: 0.001)
        XCTAssertEqual(BoardsAdvancedLogic.pollDelaySeconds(attempt: 1), 1.0, accuracy: 0.001)
        XCTAssertEqual(BoardsAdvancedLogic.pollDelaySeconds(attempt: 10), 8.0, accuracy: 0.001)
        XCTAssertTrue(BoardsAdvancedLogic.isExportTerminal("done"))
        XCTAssertTrue(BoardsAdvancedLogic.isExportTerminal("failed"))
        XCTAssertFalse(BoardsAdvancedLogic.isExportTerminal("running"))
        XCTAssertTrue(BoardsAdvancedLogic.isCopyTerminal("completed"))
        XCTAssertEqual(BoardsAdvancedLogic.exportFileExtension(format: .image), "png")
    }

    func testOrderedPostsForPresent() {
        let sections = [
            BoardSection(id: "s2", boardId: "b", title: "B", sortIndex: 2),
            BoardSection(id: "s1", boardId: "b", title: "A", sortIndex: 1),
        ]
        let posts = [
            BoardPost(id: "p3", boardId: "b", contentType: "text", sectionId: "s2", sortIndex: 0, title: "Later"),
            BoardPost(id: "p1", boardId: "b", contentType: "text", sectionId: "s1", sortIndex: 2, title: "Second"),
            BoardPost(id: "p2", boardId: "b", contentType: "text", sectionId: "s1", sortIndex: 1, title: "First"),
            BoardPost(id: "p4", boardId: "b", contentType: "text", sectionId: nil, sortIndex: 0, title: "Unsectioned"),
        ]
        XCTAssertEqual(
            BoardsAdvancedLogic.orderedPostsForPresent(posts: posts, sections: sections).map(\.id),
            ["p2", "p1", "p3", "p4"]
        )
    }

    func testBoardTemplateDecodeIgnoresUnknown() throws {
        let json = """
        {"id":"t1","scope":"builtin","title":"KWL","description":"d","tags":["a"],"definition":{"layout":"columns"},"extra":true}
        """
        let template = try JSONDecoder().decode(BoardTemplate.self, from: Data(json.utf8))
        XCTAssertEqual(template.id, "t1")
        XCTAssertEqual(template.title, "KWL")
        XCTAssertEqual(template.tags, ["a"])
    }

    func testGovernanceGating() {
        var features = MobilePlatformFeatures()
        features.ffMobileAdminConsole = true
        features.ffMobileBoardsAdvanced = true
        XCTAssertTrue(BoardsGovernanceAdminLogic.canView(features: features, permissions: ["global:app:rbac:manage"]))
        XCTAssertFalse(BoardsGovernanceAdminLogic.canView(features: features, permissions: []))
        features.ffMobileBoardsAdvanced = false
        XCTAssertFalse(BoardsGovernanceAdminLogic.canView(features: features, permissions: ["global:app:rbac:manage"]))
    }

    func testObservabilityCounters() {
        BoardsAdvancedObservability.record("board_template_used", attributes: ["scope": "builtin"])
        BoardsAdvancedObservability.record("board_exported", attributes: ["format": "pdf"])
        BoardsAdvancedObservability.record("board_presented")
        XCTAssertEqual(BoardsAdvancedObservability.count(for: "board_template_used"), 1)
        XCTAssertEqual(BoardsAdvancedObservability.count(for: "board_exported"), 1)
        XCTAssertEqual(BoardsAdvancedObservability.count(for: "board_presented"), 1)
    }

    func testParseBoardCapDraft() {
        XCTAssertEqual(BoardsAdvancedLogic.parseBoardCapDraft("12"), 12)
        XCTAssertNil(BoardsAdvancedLogic.parseBoardCapDraft(""))
        XCTAssertNil(BoardsAdvancedLogic.parseBoardCapDraft("abc"))
    }
}
