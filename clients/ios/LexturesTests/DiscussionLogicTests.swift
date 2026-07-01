import XCTest
@testable import Lextures

final class DiscussionLogicTests: XCTestCase {
    func testNestPostsOrdersChildrenAfterParents() {
        let posts = [
            makePost(id: "root", parent: nil),
            makePost(id: "c1", parent: "root"),
            makePost(id: "c2", parent: "c1"),
            makePost(id: "root2", parent: nil),
        ]
        let nested = DiscussionLogic.nestPosts(posts)
        XCTAssertEqual(nested.map(\.post.id), ["root", "c1", "c2", "root2"])
        XCTAssertEqual(nested.map(\.depth), [0, 1, 2, 0])
    }

    func testSortThreadsPinsFirst() {
        let threads = [
            makeThread(id: "a", pinned: false, updatedAt: "2099-01-02T00:00:00Z"),
            makeThread(id: "b", pinned: true, updatedAt: "2020-01-01T00:00:00Z"),
        ]
        XCTAssertEqual(DiscussionLogic.sortThreads(threads).map(\.id), ["b", "a"])
    }

    func testPlainTextFromTipTapDoc() throws {
        let data = try DiscussionLogic.encodeBody(text: "Hello\nWorld")
        XCTAssertEqual(DiscussionLogic.plainText(from: data), "Hello\nWorld")
    }

    func testAuthorLabelUsesYouForViewer() {
        XCTAssertEqual(DiscussionLogic.authorLabel(authorId: "u1", viewerId: "u1"), L.text("mobile.discussions.authorYou"))
        XCTAssertEqual(DiscussionLogic.authorLabel(authorId: "abcdefgh123", viewerId: "other"), "abcdefgh")
    }

    private func makePost(id: String, parent: String?) -> DiscussionPost {
        DiscussionPost(
            id: id,
            threadId: "t1",
            parentPostId: parent,
            authorId: "u1",
            bodyJSON: DiscussionLogic.emptyBodyJSON(),
            upvoteCount: 0,
            viewerUpvoted: false,
            createdAt: "2024-01-01T00:00:00Z",
            updatedAt: "2024-01-01T00:00:00Z"
        )
    }

    private func makeThread(id: String, pinned: Bool, updatedAt: String) -> DiscussionThreadSummary {
        DiscussionThreadSummary(
            id: id,
            forumId: "f1",
            authorId: "u1",
            title: id,
            isPinned: pinned,
            isLocked: false,
            requirePostFirst: false,
            assignmentStructureItemId: nil,
            createdAt: updatedAt,
            updatedAt: updatedAt,
            replyCount: 0
        )
    }
}
