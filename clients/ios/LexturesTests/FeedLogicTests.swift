import XCTest
@testable import Lextures

final class FeedLogicTests: XCTestCase {
    func testOrderedMessagesPutsPinnedFirstThenFlattensReplies() {
        let reply = makeMessage(id: "r1", createdAt: "2024-01-01T00:01:00Z")
        let root = makeMessage(id: "root", createdAt: "2024-01-01T00:00:00Z", replies: [reply])
        let pinned = makeMessage(id: "pinned", createdAt: "2024-01-01T00:02:00Z", pinnedAt: "2024-01-01T00:02:30Z")
        let ordered = FeedLogic.orderedMessages([root, pinned])
        XCTAssertEqual(ordered.map(\.id), ["pinned", "root", "r1"])
    }

    func testCanEditAndDeleteRequireAuthorMatch() {
        let message = makeMessage(id: "m1", authorUserId: "u1")
        XCTAssertTrue(FeedLogic.canEdit(message, viewerId: "u1"))
        XCTAssertFalse(FeedLogic.canEdit(message, viewerId: "u2"))
        XCTAssertFalse(FeedLogic.canEdit(message, viewerId: nil))
        XCTAssertTrue(FeedLogic.canDelete(message, viewerId: "u1"))
    }

    func testCanPinRequiresStaffAndRootMessage() {
        XCTAssertTrue(FeedLogic.canPin(viewerIsStaff: true, isReply: false))
        XCTAssertFalse(FeedLogic.canPin(viewerIsStaff: true, isReply: true))
        XCTAssertFalse(FeedLogic.canPin(viewerIsStaff: false, isReply: false))
    }

    func testExtractImagePathSplitsMarkdownFromText() {
        let (text, path) = FeedLogic.extractImagePath(from: "hello\n\n![image](/api/v1/x/content)")
        XCTAssertEqual(text, "hello")
        XCTAssertEqual(path, "/api/v1/x/content")
    }

    func testExtractImagePathReturnsNilWhenNoMarkdown() {
        let (text, path) = FeedLogic.extractImagePath(from: "just text")
        XCTAssertEqual(text, "just text")
        XCTAssertNil(path)
    }

    private func makeMessage(
        id: String,
        authorUserId: String = "u1",
        createdAt: String = "2024-01-01T00:00:00Z",
        pinnedAt: String? = nil,
        replies: [FeedMessage] = []
    ) -> FeedMessage {
        FeedMessage(
            id: id,
            channelId: "c1",
            authorUserId: authorUserId,
            authorEmail: "user@example.com",
            authorDisplayName: nil,
            parentMessageId: nil,
            body: "body",
            mentionsEveryone: false,
            mentionUserIds: [],
            pinnedAt: pinnedAt,
            createdAt: createdAt,
            editedAt: nil,
            likeCount: 0,
            viewerHasLiked: false,
            replies: replies
        )
    }
}
