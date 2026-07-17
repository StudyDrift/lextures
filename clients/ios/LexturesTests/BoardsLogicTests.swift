import XCTest
@testable import Lextures

// Covers VC.M1–VC.M7 boards helpers in one suite.
// swiftlint:disable type_body_length

final class BoardsLogicTests: XCTestCase {
    func testCourseSummaryDecodesVisualBoardsEnabled() throws {
        let json = """
        {"id":"1","courseCode":"C1","title":"T","description":"","visualBoardsEnabled":true}
        """
        let course = try JSONDecoder().decode(CourseSummary.self, from: Data(json.utf8))
        XCTAssertEqual(course.visualBoardsEnabled, true)
        XCTAssertTrue(course.isVisualBoardsEnabled)
    }

    func testCourseSummaryVisualBoardsDefaultsOff() throws {
        let json = """
        {"id":"1","courseCode":"C1","title":"T","description":""}
        """
        let course = try JSONDecoder().decode(CourseSummary.self, from: Data(json.utf8))
        XCTAssertNil(course.visualBoardsEnabled)
        XCTAssertFalse(course.isVisualBoardsEnabled)
    }

    func testBoardDecodesOptionalFields() throws {
        let json = """
        {
          "id":"b1","courseId":"c1","title":"Wall","description":"Ideas",
          "slug":"wall","archived":false,"createdBy":"u1",
          "createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z",
          "layout":"wall","layoutLocked":true,"canPost":true,"canArrange":false,"extraUnknown":true
        }
        """
        let board = try JSONDecoder().decode(Board.self, from: Data(json.utf8))
        XCTAssertEqual(board.id, "b1")
        XCTAssertEqual(board.title, "Wall")
        XCTAssertEqual(board.description, "Ideas")
        XCTAssertFalse(board.archived)
        XCTAssertEqual(board.layout, "wall")
        XCTAssertTrue(board.layoutLocked)
        XCTAssertEqual(board.canPost, true)
        XCTAssertEqual(board.canArrange, false)
    }

    func testResolveLayoutFallsBackToStream() {
        XCTAssertEqual(BoardsLogic.resolveLayout("columns"), .columns)
        XCTAssertEqual(BoardsLogic.resolveLayout("unknown-future"), .stream)
        XCTAssertEqual(BoardsLogic.resolveLayout(nil), .stream)
    }

    func testMidpointSortIndex() {
        XCTAssertEqual(BoardsLogic.midpointSortIndex(before: nil, after: nil), 0)
        XCTAssertEqual(BoardsLogic.midpointSortIndex(before: nil, after: 10), 9)
        XCTAssertEqual(BoardsLogic.midpointSortIndex(before: 10, after: nil), 11)
        XCTAssertEqual(BoardsLogic.midpointSortIndex(before: 2, after: 4), 3)
        XCTAssertEqual(BoardsLogic.midpointSortIndex(before: 5, after: 5), 6)
    }

    func testCanArrangeRespectsLockAndOwnership() {
        let board = Board(id: "b", courseId: "c", title: "T", layoutLocked: true)
        let own = BoardPost(id: "p", boardId: "b", authorId: "u1", contentType: "text")
        let other = BoardPost(id: "p2", boardId: "b", authorId: "u2", contentType: "text")
        XCTAssertFalse(BoardsLogic.canArrangePost(post: own, board: board, currentUserId: "u1", canManage: false))
        XCTAssertTrue(BoardsLogic.canArrangePost(post: other, board: board, currentUserId: "u2", canManage: true))
        let unlocked = Board(id: "b", courseId: "c", title: "T", layoutLocked: false)
        XCTAssertTrue(BoardsLogic.canArrangePost(post: own, board: unlocked, currentUserId: "u1", canManage: false))
        XCTAssertFalse(BoardsLogic.canArrangePost(post: other, board: unlocked, currentUserId: "u1", canManage: false))
    }

    func testPostsInSectionAndTimelineBuckets() {
        let posts = [
            BoardPost(id: "1", boardId: "b", contentType: "text", sectionId: "s1", sortIndex: 2, eventDate: "2026-02-01"),
            BoardPost(id: "2", boardId: "b", contentType: "text", sectionId: "s1", sortIndex: 1, eventDate: nil),
            BoardPost(id: "3", boardId: "b", contentType: "text", sectionId: nil, sortIndex: 0, eventDate: "2026-01-01"),
        ]
        XCTAssertEqual(BoardsLogic.postsInSection(posts, sectionId: "s1").map(\.id), ["2", "1"])
        XCTAssertEqual(BoardsLogic.datedPosts(posts).map(\.id), ["3", "1"])
        XCTAssertEqual(BoardsLogic.undatedPosts(posts).map(\.id), ["2"])
    }

    func testClusterPinsGroupsNearby() {
        let posts = [
            BoardPost(id: "a", boardId: "b", contentType: "text", lat: 10, lng: 10),
            BoardPost(id: "b", boardId: "b", contentType: "text", lat: 10.1, lng: 10.1),
            BoardPost(id: "c", boardId: "b", contentType: "text", lat: 80, lng: 80),
        ]
        let clusters = BoardsLogic.clusterPins(posts: posts, zoom: 1)
        XCTAssertGreaterThanOrEqual(clusters.count, 1)
        XCTAssertEqual(clusters.flatMap(\.postIds).sorted(), ["a", "b", "c"])
    }

    func testBoardPostDecodesAttachmentAndBody() throws {
        let json = """
        {
          "id":"p1","boardId":"b1","authorId":"u1","contentType":"image","title":"Shot",
          "body":{"text":"hello","html":"<p>hello</p>"},
          "attachment":{
            "id":"a1","url":"/files/a1","fileName":"shot.jpg","mimeType":"image/jpeg",
            "sizeBytes":1200,"altText":"whiteboard","scanStatus":"clean"
          },
          "sortIndex":1,"createdAt":"2026-01-02T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"
        }
        """
        let post = try JSONDecoder().decode(BoardPost.self, from: Data(json.utf8))
        XCTAssertEqual(post.contentType, "image")
        XCTAssertEqual(post.body?.text, "hello")
        XCTAssertEqual(post.attachment?.scanStatus, "clean")
        XCTAssertEqual(post.attachment?.altText, "whiteboard")
    }

    func testSortedBoardsExcludesArchivedByDefault() {
        let boards = [
            Board(id: "1", courseId: "c", title: "Old", updatedAt: "2026-01-01T00:00:00Z"),
            Board(id: "2", courseId: "c", title: "New", updatedAt: "2026-01-03T00:00:00Z"),
            Board(id: "3", courseId: "c", title: "Archived", archived: true, updatedAt: "2026-01-04T00:00:00Z"),
        ]
        let visible = BoardsLogic.sortedBoards(boards, includeArchived: false)
        XCTAssertEqual(visible.map(\.id), ["2", "1"])
        let all = BoardsLogic.sortedBoards(boards, includeArchived: true)
        XCTAssertEqual(all.map(\.id), ["3", "2", "1"])
    }

    func testSortedPostsNewestFirst() {
        let posts = [
            BoardPost(id: "1", boardId: "b", contentType: "text", createdAt: "2026-01-01T00:00:00Z"),
            BoardPost(id: "2", boardId: "b", contentType: "text", createdAt: "2026-01-03T00:00:00Z"),
        ]
        XCTAssertEqual(BoardsLogic.sortedPosts(posts).map(\.id), ["2", "1"])
    }

    func testCanCreateBoardsUsesItemCreatePermission() {
        XCTAssertTrue(
            BoardsLogic.canCreateBoards(
                courseCode: "C1",
                permissions: ["course:C1:item:create"]
            )
        )
        XCTAssertFalse(
            BoardsLogic.canCreateBoards(
                courseCode: "C1",
                permissions: ["course:C1:item:read"]
            )
        )
    }

    func testCanPostPrefersServerFlag() {
        let board = Board(id: "b", courseId: "c", title: "T", canPost: true)
        XCTAssertTrue(
            BoardsLogic.canPost(
                board: board,
                courseCode: "C1",
                permissions: ["course:C1:item:read"]
            )
        )
        let locked = Board(id: "b", courseId: "c", title: "T", canPost: false)
        XCTAssertFalse(
            BoardsLogic.canPost(
                board: locked,
                courseCode: "C1",
                permissions: ["course:C1:item:create"]
            )
        )
    }

    func testCanEditOrDeletePostAuthorOrManager() {
        let post = BoardPost(id: "p", boardId: "b", authorId: "u1", contentType: "text")
        XCTAssertTrue(BoardsLogic.canEditOrDeletePost(post: post, currentUserId: "u1", canManage: false))
        XCTAssertFalse(BoardsLogic.canEditOrDeletePost(post: post, currentUserId: "u2", canManage: false))
        XCTAssertTrue(BoardsLogic.canEditOrDeletePost(post: post, currentUserId: "u2", canManage: true))
    }

    func testAttachmentMediaURLRespectsScanStatus() {
        let clean = BoardAttachment(id: "a", url: "/x", scanStatus: "clean")
        XCTAssertNotNil(BoardsLogic.attachmentMediaURL(clean))
        let blocked = BoardAttachment(id: "a", url: "/x", scanStatus: "blocked")
        XCTAssertNil(BoardsLogic.attachmentMediaURL(blocked))
        let pending = BoardAttachment(id: "a", url: "/x", scanStatus: "pending")
        XCTAssertNil(BoardsLogic.attachmentMediaURL(pending))
    }

    func testValidateComposeMissingContent() {
        XCTAssertEqual(
            BoardsLogic.validateCompose(contentType: .text, text: " ", linkUrl: "", hasFile: false, altText: "", hasAudio: false),
            .missingText
        )
        XCTAssertEqual(
            BoardsLogic.validateCompose(contentType: .image, text: "", linkUrl: "", hasFile: true, altText: "", hasAudio: false),
            .missingAltText
        )
        XCTAssertEqual(
            BoardsLogic.validateCompose(contentType: .link, text: "", linkUrl: "https://example.com", hasFile: false, altText: "", hasAudio: false),
            .ok
        )
    }

    func testVideoEmbedFromYouTubeAndVimeo() {
        let yt = BoardsLogic.videoEmbedFromUrl("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
        XCTAssertEqual(yt?.provider, "youtube")
        XCTAssertEqual(yt?.id, "dQw4w9WgXcQ")
        let short = BoardsLogic.videoEmbedFromUrl("https://youtu.be/dQw4w9WgXcQ")
        XCTAssertEqual(short?.id, "dQw4w9WgXcQ")
        let vimeo = BoardsLogic.videoEmbedFromUrl("https://vimeo.com/123456")
        XCTAssertEqual(vimeo?.provider, "vimeo")
        XCTAssertEqual(vimeo?.id, "123456")
        XCTAssertNil(BoardsLogic.videoEmbedFromUrl("https://example.com/video"))
    }

    func testIsKnownContentTypeAndUnsupported() {
        XCTAssertTrue(BoardsLogic.isKnownContentType("text"))
        XCTAssertTrue(BoardsLogic.isKnownContentType("drawing"))
        XCTAssertFalse(BoardsLogic.isKnownContentType("hologram"))
    }

    func testVisualBoardsToolToggleAndPatch() {
        var course = CourseSummary(id: "1", courseCode: "C1", title: "T", description: "")
        XCTAssertFalse(CourseFeaturesLogic.isEnabled(.visualBoards, course: course))
        course = CourseFeaturesLogic.applyToggle(course: course, tool: .visualBoards, enabled: true)
        XCTAssertTrue(CourseFeaturesLogic.isEnabled(.visualBoards, course: course))
        let patch = CourseFeaturesLogic.buildFeaturesPatch(from: course)
        XCTAssertTrue(patch.visualBoardsEnabled)
    }

    // MARK: - VC.M5 engagement

    func testApplyReactionResultToggle() {
        let post = BoardPost(
            id: "p",
            boardId: "b",
            contentType: "text",
            reactionCount: 2,
            myReaction: BoardMyReaction(kind: "like")
        )
        let off = BoardsLogic.applyReactionResult(
            post,
            result: BoardReactionResult(active: false, reactionCount: 1)
        )
        XCTAssertNil(off.myReaction)
        XCTAssertEqual(off.reactionCount, 1)

        let on = BoardsLogic.applyReactionResult(
            post,
            result: BoardReactionResult(
                active: true,
                reactionCount: 3,
                myReaction: BoardMyReaction(kind: "like")
            )
        )
        XCTAssertEqual(on.myReaction?.kind, "like")
        XCTAssertEqual(on.reactionCount, 3)
    }

    func testOptimisticToggleIdempotent() {
        let base = BoardPost(id: "p", boardId: "b", contentType: "text", reactionCount: 1)
        let liked = BoardsLogic.optimisticToggleReaction(base, kind: "like")
        XCTAssertNotNil(liked.myReaction)
        XCTAssertEqual(liked.reactionCount, 2)
        let unliked = BoardsLogic.optimisticToggleReaction(liked, kind: "like")
        XCTAssertNil(unliked.myReaction)
        XCTAssertEqual(unliked.reactionCount, 1)
    }

    func testStarAverageScoreAndSort() {
        let posts = [
            BoardPost(id: "a", boardId: "b", contentType: "text", reactionCount: 3, avgStars: 4.0),
            BoardPost(id: "b", boardId: "b", contentType: "text", reactionCount: 10, avgStars: 2.0),
            BoardPost(id: "c", boardId: "b", contentType: "text", reactionCount: 1, avgStars: 5.0),
        ]
        XCTAssertEqual(
            BoardsLogic.boardPostReactionScore(posts[0], mode: .star),
            4.0 * 1000 + 3,
            accuracy: 0.001
        )
        let sorted = BoardsLogic.sortedPosts(posts, mode: .mostReacted, reactionMode: .star)
        XCTAssertEqual(sorted.map(\.id), ["c", "a", "b"])
    }

    func testVisibleGradeIsServerValueOnly() {
        let own = BoardPost(id: "p", boardId: "b", contentType: "text", grade: 88)
        let peer = BoardPost(id: "p2", boardId: "b", contentType: "text", grade: nil)
        XCTAssertEqual(BoardsLogic.visibleGrade(for: own), 88)
        XCTAssertNil(BoardsLogic.visibleGrade(for: peer))
    }

    func testNestCommentsAndVisibility() {
        let comments = [
            BoardComment(id: "1", postId: "p", body: BoardPostBody(text: "root")),
            BoardComment(id: "2", postId: "p", parentId: "1", body: BoardPostBody(text: "reply")),
            BoardComment(id: "3", postId: "p", body: BoardPostBody(text: "hidden"), hidden: true),
        ]
        let nested = BoardsLogic.nestComments(comments)
        XCTAssertEqual(nested.count, 2)
        XCTAssertEqual(nested.first?.children.map(\.id), ["2"])
        XCTAssertEqual(BoardsLogic.visibleComments(comments, canManageBoard: false).map(\.id), ["1", "2"])
        XCTAssertEqual(BoardsLogic.visibleComments(comments, canManageBoard: true).count, 3)
    }

    func testCanInteractDefaultsTrueAndRespectsFlags() {
        XCTAssertTrue(BoardsLogic.canInteract(board: nil))
        XCTAssertTrue(BoardsLogic.canInteract(board: Board(id: "b", courseId: "c", title: "T")))
        XCTAssertFalse(
            BoardsLogic.canInteract(board: Board(id: "b", courseId: "c", title: "T", canInteract: false))
        )
        XCTAssertFalse(
            BoardsLogic.canInteract(
                board: Board(
                    id: "b",
                    courseId: "c",
                    title: "T",
                    canInteract: true,
                    capabilities: BoardCapabilities(canInteract: false)
                )
            )
        )
    }

    func testBoardPostDecodesEngagementAggregates() throws {
        let json = """
        {
          "id":"p1","boardId":"b1","contentType":"text","title":"",
          "reactionCount":3,"myReaction":{"kind":"star","value":4},
          "avgStars":4.0,"commentCount":2,"grade":91,
          "sortIndex":0,"createdAt":"2026-01-02T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"
        }
        """
        let post = try JSONDecoder().decode(BoardPost.self, from: Data(json.utf8))
        XCTAssertEqual(post.reactionCount, 3)
        XCTAssertEqual(post.myReaction?.kind, "star")
        XCTAssertEqual(post.myReaction?.value, 4)
        XCTAssertEqual(post.avgStars, 4)
        XCTAssertEqual(post.commentCount, 2)
        XCTAssertEqual(post.grade, 91)
    }

    // MARK: - VC.M4 realtime

    func testParseBoardChangedEvent() {
        let parsed = BoardRealtimeLogic.parseBoardChangedEvent(
            from: #"{"type":"board.changed","reason":"post.created","postId":"p1"}"#
        )
        XCTAssertEqual(parsed?.reason, "post.created")
        XCTAssertEqual(parsed?.postId, "p1")
        XCTAssertNil(BoardRealtimeLogic.parseBoardChangedEvent(from: #"{"type":"other"}"#))
        XCTAssertNil(BoardRealtimeLogic.parseBoardChangedEvent(from: "not-json"))
        // Binary / non-UTF8-looking garbage is a safe no-op
        XCTAssertNil(BoardRealtimeLogic.parseBoardChangedEvent(from: Data([0x00, 0x01, 0x02, 0xFF])))
    }

    func testParseBoardLockedOrFrozenError() {
        XCTAssertTrue(
            BoardRealtimeLogic.isBoardLockedOrFrozenError(from: #"{"error":"board_locked_or_frozen"}"#)
        )
        XCTAssertFalse(BoardRealtimeLogic.isBoardLockedOrFrozenError(from: #"{"type":"board.changed"}"#))
    }

    func testPermanentWsRefusalAndRetryStop() {
        XCTAssertTrue(BoardRealtimeLogic.isPermanentWsRefusal(404))
        XCTAssertTrue(BoardRealtimeLogic.isPermanentWsRefusal(403))
        XCTAssertFalse(BoardRealtimeLogic.isPermanentWsRefusal(500))
        XCTAssertFalse(BoardRealtimeLogic.isPermanentWsRefusal(nil))
        XCTAssertTrue(BoardRealtimeLogic.shouldStopRetrying(consecutiveFailures: 1, lastHttpStatus: 404))
        XCTAssertFalse(BoardRealtimeLogic.shouldStopRetrying(consecutiveFailures: 2, lastHttpStatus: nil))
        XCTAssertTrue(
            BoardRealtimeLogic.shouldStopRetrying(
                consecutiveFailures: BoardRealtimeLogic.maxTransientFailuresBeforeOffline,
                lastHttpStatus: nil
            )
        )
    }

    func testCoalesceRefetchPlan() {
        let single = BoardRealtimeLogic.coalesceRefetchPlan(events: [
            BoardChangedEvent(reason: "post.updated", postId: "p1"),
            BoardChangedEvent(reason: "post.updated", postId: "p1"),
        ])
        XCTAssertFalse(single.full)
        XCTAssertEqual(single.postId, "p1")

        let multi = BoardRealtimeLogic.coalesceRefetchPlan(events: [
            BoardChangedEvent(reason: "post.created", postId: "a"),
            BoardChangedEvent(reason: "post.created", postId: "b"),
        ])
        XCTAssertTrue(multi.full)
        XCTAssertNil(multi.postId)
        XCTAssertEqual(multi.createdCount, 2)

        let general = BoardRealtimeLogic.coalesceRefetchPlan(events: [
            BoardChangedEvent(reason: "section.created", postId: nil),
        ])
        XCTAssertTrue(general.full)
    }

    // MARK: - VC.M6 access / attribution

    func testCanManagePrefersCapabilities() {
        let caps = Board(
            id: "b",
            courseId: "c",
            title: "T",
            capabilities: BoardCapabilities(canManage: true)
        )
        XCTAssertTrue(BoardsLogic.canManageBoard(board: caps, courseCode: "c", permissions: []))
        let denied = Board(
            id: "b",
            courseId: "c",
            title: "T",
            capabilities: BoardCapabilities(canManage: false)
        )
        XCTAssertFalse(BoardsLogic.canManageBoard(board: denied, courseCode: "c", permissions: []))
    }

    func testCanArrangeDisabledByCapabilitiesEvenForOwner() {
        let board = Board(
            id: "b",
            courseId: "c",
            title: "T",
            capabilities: BoardCapabilities(canArrange: false)
        )
        let own = BoardPost(id: "p", boardId: "b", authorId: "u1", contentType: "text")
        XCTAssertFalse(
            BoardsLogic.canArrangePost(post: own, board: board, currentUserId: "u1", canManage: false)
        )
        XCTAssertTrue(
            BoardsLogic.canArrangePost(post: own, board: board, currentUserId: "u1", canManage: true)
        )
    }

    func testAttributionLabelNeverInventsAuthor() {
        XCTAssertNil(BoardsLogic.attributionLabel(authorId: nil, guestDisplayName: nil))
        XCTAssertNil(BoardsLogic.attributionLabel(authorId: "  ", guestDisplayName: ""))
        XCTAssertEqual(
            BoardsLogic.attributionLabel(authorId: "u1", guestDisplayName: nil),
            "u1"
        )
        XCTAssertEqual(
            BoardsLogic.attributionLabel(authorId: "u1", guestDisplayName: "Guest"),
            "Guest"
        )
        let post = BoardPost(id: "p", boardId: "b", contentType: "text")
        XCTAssertNil(BoardsLogic.attributionLabel(for: post))
    }

    func testBoardDecodesAccessFields() throws {
        let json = """
        {
          "id":"b1","courseId":"c1","title":"Wall","description":"",
          "visibility":"invite","visibilityTarget":null,"attribution":"anon_to_peers",
          "canPost":false,"canInteract":true,"canArrange":false,
          "externalSharingAllowed":true,"minorModerationFloor":false,
          "capabilities":{"canView":true,"canPost":false,"canInteract":true,"canArrange":false,"canManage":true},
          "createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"
        }
        """
        let board = try JSONDecoder().decode(Board.self, from: Data(json.utf8))
        XCTAssertEqual(board.visibility, "invite")
        XCTAssertEqual(board.attribution, "anon_to_peers")
        XCTAssertEqual(board.externalSharingAllowed, true)
        XCTAssertEqual(board.capabilities?.canManage, true)
        XCTAssertTrue(BoardsLogic.externalSharingAllowed(board: board))
        XCTAssertEqual(
            BoardsLogic.visibilityOptions(for: board),
            BoardVisibility.inCourse + [.link, .public]
        )
    }

    func testExternalSharingBlockedForMinors() {
        let board = Board(
            id: "b",
            courseId: "c",
            title: "T",
            externalSharingAllowed: true,
            minorModerationFloor: true
        )
        XCTAssertFalse(BoardsLogic.externalSharingAllowed(board: board))
        XCTAssertEqual(BoardsLogic.visibilityOptions(for: board), BoardVisibility.inCourse)
    }

    func testBoardDecodesModerationFields() throws {
        let json = """
        {
          "id":"b1","courseId":"c1","title":"Wall",
          "moderationMode":"approval","filterAction":"block","locked":true,
          "frozenUntil":"2099-01-01T00:00:00Z",
          "createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"
        }
        """
        let board = try JSONDecoder().decode(Board.self, from: Data(json.utf8))
        XCTAssertEqual(board.moderationMode, "approval")
        XCTAssertEqual(board.filterAction, "block")
        XCTAssertTrue(board.locked)
        XCTAssertTrue(BoardsLogic.isBoardLocked(board))
        XCTAssertTrue(BoardsLogic.isBoardFrozen(board))
        XCTAssertFalse(BoardsLogic.canWritePosts(board: board, canManage: false))
        XCTAssertTrue(BoardsLogic.canWritePosts(board: board, canManage: true))
    }

    func testPostSafetyStates() {
        let pending = BoardPost(id: "p", boardId: "b", contentType: "text", status: "pending")
        XCTAssertEqual(BoardsLogic.postSafetyState(pending), .pendingApproval)
        let removed = BoardPost(id: "p", boardId: "b", contentType: "text", hidden: true)
        XCTAssertEqual(BoardsLogic.postSafetyState(removed), .removed)
        let blocked = BoardPost(
            id: "p",
            boardId: "b",
            contentType: "image",
            attachment: BoardAttachment(id: "a", scanStatus: "blocked")
        )
        XCTAssertEqual(BoardsLogic.postSafetyState(blocked), .fileBlocked)
    }

    func testReportDebounceAndOrgFloor() {
        BoardsLogic.resetReportedTargetsForTests()
        XCTAssertFalse(BoardsLogic.hasReported(postId: "p1"))
        BoardsLogic.markReported(postId: "p1")
        XCTAssertTrue(BoardsLogic.hasReported(postId: "p1"))
        let floor = Board(
            id: "b",
            courseId: "c",
            title: "T",
            minorModerationFloor: true
        )
        XCTAssertTrue(BoardsLogic.moderationControlsLockedByOrgFloor(floor))
        XCTAssertTrue(BoardsLogic.isFilterBlockMessage("This content could not be posted."))
        XCTAssertTrue(BoardsLogic.isLockOrFreezeMessage("This board is locked"))
        BoardsLogic.resetReportedTargetsForTests()
    }

    func testClassifyBoardLinkError() {
        XCTAssertEqual(
            BoardsLogic.classifyBoardLinkError(status: 401, message: "Incorrect password"),
            .needsPassword
        )
        XCTAssertEqual(
            BoardsLogic.classifyBoardLinkError(status: 404, message: nil),
            .denied
        )
        XCTAssertEqual(
            BoardsLogic.classifyBoardLinkError(status: 403, message: "forbidden"),
            .denied
        )
    }

    func testBoardLinkDeepLinkResolves() {
        let dest = DeepLinkRouter.resolve("https://lextures.com/board-links/tok-abc")
        guard case let .boardLink(token) = dest else {
            return XCTFail("expected boardLink")
        }
        XCTAssertEqual(token, "tok-abc")
        XCTAssertEqual(DeepLinkRouter.resolve("/board-links/"), .home)
    }

    func testShareURLBuildsFromToken() {
        let share = BoardShare(id: "s1", boardId: "b1", capability: "view", token: "abc")
        XCTAssertEqual(BoardsLogic.shareURL(for: share)?.path, "/board-links/abc")
    }
}

// swiftlint:enable type_body_length
