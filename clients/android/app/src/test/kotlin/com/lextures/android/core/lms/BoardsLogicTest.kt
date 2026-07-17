package com.lextures.android.core.lms

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class BoardsLogicTest {
    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    @Test
    fun courseSummaryDecodesVisualBoardsEnabled() {
        val course = json.decodeFromString<CourseSummary>(
            """{"id":"1","courseCode":"C1","title":"T","description":"","visualBoardsEnabled":true}""",
        )
        assertEquals(true, course.visualBoardsEnabled)
        assertTrue(course.isVisualBoardsEnabled)
    }

    @Test
    fun courseSummaryVisualBoardsDefaultsOff() {
        val course = json.decodeFromString<CourseSummary>(
            """{"id":"1","courseCode":"C1","title":"T","description":""}""",
        )
        assertEquals(null, course.visualBoardsEnabled)
        assertFalse(course.isVisualBoardsEnabled)
    }

    @Test
    fun boardDecodesOptionalFields() {
        val board = json.decodeFromString<Board>(
            """
            {
              "id":"b1","courseId":"c1","title":"Wall","description":"Ideas",
              "slug":"wall","archived":false,"createdBy":"u1",
              "createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z",
              "layout":"wall","layoutLocked":true,"canPost":true,"canArrange":false,"extraUnknown":true
            }
            """.trimIndent(),
        )
        assertEquals("b1", board.id)
        assertEquals("Wall", board.title)
        assertEquals("Ideas", board.description)
        assertFalse(board.archived)
        assertEquals("wall", board.layout)
        assertTrue(board.layoutLocked)
        assertEquals(true, board.canPost)
        assertEquals(false, board.canArrange)
    }

    @Test
    fun resolveLayoutFallsBackToStream() {
        assertEquals(BoardLayout.Columns, BoardsLogic.resolveLayout("columns"))
        assertEquals(BoardLayout.Stream, BoardsLogic.resolveLayout("unknown-future"))
        assertEquals(BoardLayout.Stream, BoardsLogic.resolveLayout(null))
    }

    @Test
    fun midpointSortIndex() {
        assertEquals(0.0, BoardsLogic.midpointSortIndex(null, null), 0.0)
        assertEquals(9.0, BoardsLogic.midpointSortIndex(null, 10.0), 0.0)
        assertEquals(11.0, BoardsLogic.midpointSortIndex(10.0, null), 0.0)
        assertEquals(3.0, BoardsLogic.midpointSortIndex(2.0, 4.0), 0.0)
        assertEquals(6.0, BoardsLogic.midpointSortIndex(5.0, 5.0), 0.0)
    }

    @Test
    fun canArrangeRespectsLockAndOwnership() {
        val locked = Board(id = "b", courseId = "c", title = "T", layoutLocked = true)
        val own = BoardPost(id = "p", boardId = "b", authorId = "u1", contentType = "text")
        val other = BoardPost(id = "p2", boardId = "b", authorId = "u2", contentType = "text")
        assertFalse(BoardsLogic.canArrangePost(own, locked, "u1", canManage = false))
        assertTrue(BoardsLogic.canArrangePost(other, locked, "u2", canManage = true))
        val unlocked = Board(id = "b", courseId = "c", title = "T", layoutLocked = false)
        assertTrue(BoardsLogic.canArrangePost(own, unlocked, "u1", canManage = false))
        assertFalse(BoardsLogic.canArrangePost(other, unlocked, "u1", canManage = false))
    }

    @Test
    fun postsInSectionAndTimelineBuckets() {
        val posts = listOf(
            BoardPost(id = "1", boardId = "b", contentType = "text", sectionId = "s1", sortIndex = 2.0, eventDate = "2026-02-01"),
            BoardPost(id = "2", boardId = "b", contentType = "text", sectionId = "s1", sortIndex = 1.0),
            BoardPost(id = "3", boardId = "b", contentType = "text", sortIndex = 0.0, eventDate = "2026-01-01"),
        )
        assertEquals(listOf("2", "1"), BoardsLogic.postsInSection(posts, "s1").map { it.id })
        assertEquals(listOf("3", "1"), BoardsLogic.datedPosts(posts).map { it.id })
        assertEquals(listOf("2"), BoardsLogic.undatedPosts(posts).map { it.id })
    }

    @Test
    fun clusterPinsGroupsNearby() {
        val posts = listOf(
            BoardPost(id = "a", boardId = "b", contentType = "text", lat = 10.0, lng = 10.0),
            BoardPost(id = "b", boardId = "b", contentType = "text", lat = 10.1, lng = 10.1),
            BoardPost(id = "c", boardId = "b", contentType = "text", lat = 80.0, lng = 80.0),
        )
        val clusters = BoardsLogic.clusterPins(posts, zoom = 1.0)
        assertTrue(clusters.isNotEmpty())
        assertEquals(listOf("a", "b", "c"), clusters.flatMap { it.postIds }.sorted())
    }

    @Test
    fun boardPostDecodesAttachmentAndBody() {
        val post = json.decodeFromString<BoardPost>(
            """
            {
              "id":"p1","boardId":"b1","authorId":"u1","contentType":"image","title":"Shot",
              "body":{"text":"hello","html":"<p>hello</p>"},
              "attachment":{
                "id":"a1","url":"/files/a1","fileName":"shot.jpg","mimeType":"image/jpeg",
                "sizeBytes":1200,"altText":"whiteboard","scanStatus":"clean"
              },
              "sortIndex":1,"createdAt":"2026-01-02T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"
            }
            """.trimIndent(),
        )
        assertEquals("image", post.contentType)
        assertEquals("hello", post.body?.text)
        assertEquals("clean", post.attachment?.scanStatus)
        assertEquals("whiteboard", post.attachment?.altText)
    }

    @Test
    fun sortedBoardsExcludesArchivedByDefault() {
        val boards = listOf(
            Board(id = "1", courseId = "c", title = "Old", updatedAt = "2026-01-01T00:00:00Z"),
            Board(id = "2", courseId = "c", title = "New", updatedAt = "2026-01-03T00:00:00Z"),
            Board(id = "3", courseId = "c", title = "Archived", archived = true, updatedAt = "2026-01-04T00:00:00Z"),
        )
        assertEquals(listOf("2", "1"), BoardsLogic.sortedBoards(boards, includeArchived = false).map { it.id })
        assertEquals(listOf("3", "2", "1"), BoardsLogic.sortedBoards(boards, includeArchived = true).map { it.id })
    }

    @Test
    fun sortedPostsNewestFirst() {
        val posts = listOf(
            BoardPost(id = "1", boardId = "b", contentType = "text", createdAt = "2026-01-01T00:00:00Z"),
            BoardPost(id = "2", boardId = "b", contentType = "text", createdAt = "2026-01-03T00:00:00Z"),
        )
        assertEquals(listOf("2", "1"), BoardsLogic.sortedPosts(posts).map { it.id })
    }

    @Test
    fun canCreateBoardsUsesItemCreatePermission() {
        assertTrue(BoardsLogic.canCreateBoards("C1", listOf("course:C1:item:create")))
        assertFalse(BoardsLogic.canCreateBoards("C1", listOf("course:C1:item:read")))
    }

    @Test
    fun canPostPrefersServerFlag() {
        assertTrue(
            BoardsLogic.canPost(
                Board(id = "b", courseId = "c", title = "T", canPost = true),
                "C1",
                listOf("course:C1:item:read"),
            ),
        )
        assertFalse(
            BoardsLogic.canPost(
                Board(id = "b", courseId = "c", title = "T", canPost = false),
                "C1",
                listOf("course:C1:item:create"),
            ),
        )
    }

    @Test
    fun canEditOrDeletePostAuthorOrManager() {
        val post = BoardPost(id = "p", boardId = "b", authorId = "u1", contentType = "text")
        assertTrue(BoardsLogic.canEditOrDeletePost(post, "u1", canManage = false))
        assertFalse(BoardsLogic.canEditOrDeletePost(post, "u2", canManage = false))
        assertTrue(BoardsLogic.canEditOrDeletePost(post, "u2", canManage = true))
    }

    @Test
    fun attachmentMediaUrlRespectsScanStatus() {
        assertNotNull(BoardsLogic.attachmentMediaUrl(BoardAttachment(id = "a", url = "/x", scanStatus = "clean")))
        assertNull(BoardsLogic.attachmentMediaUrl(BoardAttachment(id = "a", url = "/x", scanStatus = "blocked")))
        assertNull(BoardsLogic.attachmentMediaUrl(BoardAttachment(id = "a", url = "/x", scanStatus = "pending")))
    }

    @Test
    fun validateComposeMissingContent() {
        assertEquals(
            BoardComposeValidation.MissingText,
            BoardsLogic.validateCompose(BoardContentType.Text, " ", "", false, "", false),
        )
        assertEquals(
            BoardComposeValidation.MissingAltText,
            BoardsLogic.validateCompose(BoardContentType.Image, "", "", true, "", false),
        )
        assertEquals(
            BoardComposeValidation.Ok,
            BoardsLogic.validateCompose(BoardContentType.Link, "", "https://example.com", false, "", false),
        )
    }

    @Test
    fun videoEmbedFromYouTubeAndVimeo() {
        val yt = BoardsLogic.videoEmbedFromUrl("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
        assertEquals("youtube", yt?.provider)
        assertEquals("dQw4w9WgXcQ", yt?.id)
        assertEquals("dQw4w9WgXcQ", BoardsLogic.videoEmbedFromUrl("https://youtu.be/dQw4w9WgXcQ")?.id)
        val vimeo = BoardsLogic.videoEmbedFromUrl("https://vimeo.com/123456")
        assertEquals("vimeo", vimeo?.provider)
        assertEquals("123456", vimeo?.id)
        assertNull(BoardsLogic.videoEmbedFromUrl("https://example.com/video"))
    }

    @Test
    fun isKnownContentTypeAndUnsupported() {
        assertTrue(BoardsLogic.isKnownContentType("text"))
        assertTrue(BoardsLogic.isKnownContentType("drawing"))
        assertFalse(BoardsLogic.isKnownContentType("hologram"))
    }

    @Test
    fun visualBoardsToolToggleAndPatch() {
        var course = CourseSummary(id = "1", courseCode = "C1", title = "T")
        assertFalse(CourseFeaturesLogic.isEnabled(CourseFeaturesLogic.Tool.visualBoards, course))
        course = CourseFeaturesLogic.applyToggle(course, CourseFeaturesLogic.Tool.visualBoards, true)
        assertTrue(CourseFeaturesLogic.isEnabled(CourseFeaturesLogic.Tool.visualBoards, course))
        assertTrue(CourseFeaturesLogic.buildFeaturesPatch(course).visualBoardsEnabled)
    }

    // VC.M5 engagement

    @Test
    fun applyReactionResultToggle() {
        val post = BoardPost(
            id = "p",
            boardId = "b",
            contentType = "text",
            reactionCount = 2,
            myReaction = BoardMyReaction(kind = "like"),
        )
        val off = BoardsLogic.applyReactionResult(
            post,
            BoardReactionResult(active = false, reactionCount = 1),
        )
        assertNull(off.myReaction)
        assertEquals(1, off.reactionCount)

        val on = BoardsLogic.applyReactionResult(
            post,
            BoardReactionResult(
                active = true,
                reactionCount = 3,
                myReaction = BoardMyReaction(kind = "like"),
            ),
        )
        assertEquals("like", on.myReaction?.kind)
        assertEquals(3, on.reactionCount)
    }

    @Test
    fun optimisticToggleIdempotent() {
        val base = BoardPost(id = "p", boardId = "b", contentType = "text", reactionCount = 1)
        val liked = BoardsLogic.optimisticToggleReaction(base, "like")
        assertNotNull(liked.myReaction)
        assertEquals(2, liked.reactionCount)
        val unliked = BoardsLogic.optimisticToggleReaction(liked, "like")
        assertNull(unliked.myReaction)
        assertEquals(1, unliked.reactionCount)
    }

    @Test
    fun starAverageScoreAndSort() {
        val posts = listOf(
            BoardPost(id = "a", boardId = "b", contentType = "text", reactionCount = 3, avgStars = 4.0),
            BoardPost(id = "b", boardId = "b", contentType = "text", reactionCount = 10, avgStars = 2.0),
            BoardPost(id = "c", boardId = "b", contentType = "text", reactionCount = 1, avgStars = 5.0),
        )
        assertEquals(
            4.0 * 1000 + 3,
            BoardsLogic.boardPostReactionScore(posts[0], BoardReactionMode.Star),
            0.001,
        )
        val sorted = BoardsLogic.sortedPosts(posts, BoardSortMode.MostReacted, BoardReactionMode.Star)
        assertEquals(listOf("c", "a", "b"), sorted.map { it.id })
    }

    @Test
    fun visibleGradeIsServerValueOnly() {
        assertEquals(88.0, BoardsLogic.visibleGrade(BoardPost(id = "p", boardId = "b", contentType = "text", grade = 88.0)))
        assertNull(BoardsLogic.visibleGrade(BoardPost(id = "p2", boardId = "b", contentType = "text")))
    }

    @Test
    fun nestCommentsAndVisibility() {
        val comments = listOf(
            BoardComment(id = "1", postId = "p", body = BoardPostBody(text = "root")),
            BoardComment(id = "2", postId = "p", parentId = "1", body = BoardPostBody(text = "reply")),
            BoardComment(id = "3", postId = "p", body = BoardPostBody(text = "hidden"), hidden = true),
        )
        val nested = BoardsLogic.nestComments(comments)
        assertEquals(2, nested.size)
        assertEquals(listOf("2"), nested.first().children.map { it.id })
        assertEquals(listOf("1", "2"), BoardsLogic.visibleComments(comments, canManageBoard = false).map { it.id })
        assertEquals(3, BoardsLogic.visibleComments(comments, canManageBoard = true).size)
    }

    @Test
    fun canInteractDefaultsTrueAndRespectsFlags() {
        assertTrue(BoardsLogic.canInteract(null))
        assertTrue(BoardsLogic.canInteract(Board(id = "b", courseId = "c", title = "T")))
        assertFalse(BoardsLogic.canInteract(Board(id = "b", courseId = "c", title = "T", canInteract = false)))
        assertFalse(
            BoardsLogic.canInteract(
                Board(
                    id = "b",
                    courseId = "c",
                    title = "T",
                    canInteract = true,
                    capabilities = BoardCapabilities(canInteract = false),
                ),
            ),
        )
    }

    @Test
    fun boardPostDecodesEngagementAggregates() {
        val post = json.decodeFromString<BoardPost>(
            """
            {
              "id":"p1","boardId":"b1","contentType":"text","title":"",
              "reactionCount":3,"myReaction":{"kind":"star","value":4},
              "avgStars":4.0,"commentCount":2,"grade":91,
              "sortIndex":0,"createdAt":"2026-01-02T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"
            }
            """.trimIndent(),
        )
        assertEquals(3, post.reactionCount)
        assertEquals("star", post.myReaction?.kind)
        assertEquals(4.0, post.myReaction?.value)
        assertEquals(4.0, post.avgStars)
        assertEquals(2, post.commentCount)
        assertEquals(91.0, post.grade)
    }

    // VC.M4 realtime

    @Test
    fun parseBoardChangedEvent() {
        val parsed = BoardRealtimeLogic.parseBoardChangedEvent(
            """{"type":"board.changed","reason":"post.created","postId":"p1"}""",
        )
        assertEquals("post.created", parsed?.reason)
        assertEquals("p1", parsed?.postId)
        assertNull(BoardRealtimeLogic.parseBoardChangedEvent("""{"type":"other"}"""))
        assertNull(BoardRealtimeLogic.parseBoardChangedEvent("not-json"))
        // Binary-looking garbage is a safe no-op
        assertNull(BoardRealtimeLogic.parseBoardChangedEvent("\u0000\u0001\u0002"))
    }

    @Test
    fun parseBoardLockedOrFrozenError() {
        assertTrue(
            BoardRealtimeLogic.isBoardLockedOrFrozenError("""{"error":"board_locked_or_frozen"}"""),
        )
        assertFalse(BoardRealtimeLogic.isBoardLockedOrFrozenError("""{"type":"board.changed"}"""))
    }

    @Test
    fun permanentWsRefusalAndRetryStop() {
        assertTrue(BoardRealtimeLogic.isPermanentWsRefusal(404))
        assertTrue(BoardRealtimeLogic.isPermanentWsRefusal(403))
        assertFalse(BoardRealtimeLogic.isPermanentWsRefusal(500))
        assertFalse(BoardRealtimeLogic.isPermanentWsRefusal(null))
        assertTrue(BoardRealtimeLogic.shouldStopRetrying(1, 404))
        assertFalse(BoardRealtimeLogic.shouldStopRetrying(2, null))
        assertTrue(
            BoardRealtimeLogic.shouldStopRetrying(
                BoardRealtimeLogic.MAX_TRANSIENT_FAILURES_BEFORE_OFFLINE,
                null,
            ),
        )
    }

    @Test
    fun coalesceRefetchPlan() {
        val single = BoardRealtimeLogic.coalesceRefetchPlan(
            listOf(
                BoardChangedEvent("post.updated", "p1"),
                BoardChangedEvent("post.updated", "p1"),
            ),
        )
        assertFalse(single.full)
        assertEquals("p1", single.postId)

        val multi = BoardRealtimeLogic.coalesceRefetchPlan(
            listOf(
                BoardChangedEvent("post.created", "a"),
                BoardChangedEvent("post.created", "b"),
            ),
        )
        assertTrue(multi.full)
        assertNull(multi.postId)
        assertEquals(2, multi.createdCount)

        val general = BoardRealtimeLogic.coalesceRefetchPlan(
            listOf(BoardChangedEvent("section.created")),
        )
        assertTrue(general.full)
    }

    // VC.M6 access / attribution

    @Test
    fun canManagePrefersCapabilities() {
        val caps = Board(
            id = "b",
            courseId = "c",
            title = "T",
            capabilities = BoardCapabilities(canManage = true),
        )
        assertTrue(BoardsLogic.canManageBoard(caps, "c", emptyList()))
        val denied = Board(
            id = "b",
            courseId = "c",
            title = "T",
            capabilities = BoardCapabilities(canManage = false),
        )
        assertFalse(BoardsLogic.canManageBoard(denied, "c", emptyList()))
    }

    @Test
    fun canArrangeDisabledByCapabilitiesEvenForOwner() {
        val board = Board(
            id = "b",
            courseId = "c",
            title = "T",
            capabilities = BoardCapabilities(canArrange = false),
        )
        val own = BoardPost(id = "p", boardId = "b", authorId = "u1", contentType = "text")
        assertFalse(BoardsLogic.canArrangePost(own, board, "u1", canManage = false))
        assertTrue(BoardsLogic.canArrangePost(own, board, "u1", canManage = true))
    }

    @Test
    fun attributionLabelNeverInventsAuthor() {
        assertNull(BoardsLogic.attributionLabel(null, null))
        assertNull(BoardsLogic.attributionLabel("  ", ""))
        assertEquals("u1", BoardsLogic.attributionLabel("u1", null))
        assertEquals("Guest", BoardsLogic.attributionLabel("u1", "Guest"))
        assertNull(BoardsLogic.attributionLabel(BoardPost(id = "p", boardId = "b", contentType = "text")))
    }

    @Test
    fun boardDecodesAccessFields() {
        val board = json.decodeFromString<Board>(
            """
            {
              "id":"b1","courseId":"c1","title":"Wall","description":"",
              "visibility":"invite","visibilityTarget":null,"attribution":"anon_to_peers",
              "canPost":false,"canInteract":true,"canArrange":false,
              "externalSharingAllowed":true,"minorModerationFloor":false,
              "capabilities":{"canView":true,"canPost":false,"canInteract":true,"canArrange":false,"canManage":true},
              "createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"
            }
            """.trimIndent(),
        )
        assertEquals("invite", board.visibility)
        assertEquals("anon_to_peers", board.attribution)
        assertEquals(true, board.externalSharingAllowed)
        assertEquals(true, board.capabilities?.canManage)
        assertTrue(BoardsLogic.externalSharingAllowed(board))
        assertEquals(
            BoardVisibility.inCourse + listOf(BoardVisibility.Link, BoardVisibility.Public),
            BoardsLogic.visibilityOptions(board),
        )
    }

    @Test
    fun externalSharingBlockedForMinors() {
        val board = Board(
            id = "b",
            courseId = "c",
            title = "T",
            externalSharingAllowed = true,
            minorModerationFloor = true,
        )
        assertFalse(BoardsLogic.externalSharingAllowed(board))
        assertEquals(BoardVisibility.inCourse, BoardsLogic.visibilityOptions(board))
    }

    @Test
    fun boardDecodesModerationFields() {
        val board = json.decodeFromString<Board>(
            """
            {
              "id":"b1","courseId":"c1","title":"Wall",
              "moderationMode":"approval","filterAction":"block","locked":true,
              "frozenUntil":"2099-01-01T00:00:00Z",
              "createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"
            }
            """.trimIndent(),
        )
        assertEquals("approval", board.moderationMode)
        assertEquals("block", board.filterAction)
        assertTrue(board.locked)
        assertTrue(BoardsLogic.isBoardLocked(board))
        assertTrue(BoardsLogic.isBoardFrozen(board))
        assertFalse(BoardsLogic.canWritePosts(board, canManage = false))
        assertTrue(BoardsLogic.canWritePosts(board, canManage = true))
    }

    @Test
    fun postSafetyStates() {
        assertEquals(
            BoardPostSafetyState.PendingApproval,
            BoardsLogic.postSafetyState(BoardPost(id = "p", boardId = "b", contentType = "text", status = "pending")),
        )
        assertEquals(
            BoardPostSafetyState.Removed,
            BoardsLogic.postSafetyState(BoardPost(id = "p", boardId = "b", contentType = "text", hidden = true)),
        )
        assertEquals(
            BoardPostSafetyState.FileBlocked,
            BoardsLogic.postSafetyState(
                BoardPost(
                    id = "p",
                    boardId = "b",
                    contentType = "image",
                    attachment = BoardAttachment(id = "a", scanStatus = "blocked"),
                ),
            ),
        )
    }

    @Test
    fun reportDebounceAndOrgFloor() {
        BoardsLogic.resetReportedTargetsForTests()
        assertFalse(BoardsLogic.hasReported(postId = "p1"))
        BoardsLogic.markReported(postId = "p1")
        assertTrue(BoardsLogic.hasReported(postId = "p1"))
        val floor = Board(id = "b", courseId = "c", title = "T", minorModerationFloor = true)
        assertTrue(BoardsLogic.moderationControlsLockedByOrgFloor(floor))
        assertTrue(BoardsLogic.isFilterBlockMessage("This content could not be posted."))
        assertTrue(BoardsLogic.isLockOrFreezeMessage("This board is locked"))
        BoardsLogic.resetReportedTargetsForTests()
    }

    @Test
    fun classifyBoardLinkError() {
        assertEquals(
            BoardLinkAccessState.NeedsPassword,
            BoardsLogic.classifyBoardLinkError(401, "Incorrect password"),
        )
        assertEquals(BoardLinkAccessState.Denied, BoardsLogic.classifyBoardLinkError(404, null))
        assertEquals(BoardLinkAccessState.Denied, BoardsLogic.classifyBoardLinkError(403, "forbidden"))
    }

    @Test
    fun shareUrlBuildsFromToken() {
        val share = BoardShare(id = "s1", boardId = "b1", capability = "view", token = "abc")
        val url = BoardsLogic.shareUrl(share)
        assertTrue(url != null && url.endsWith("/board-links/abc"))
    }
}
