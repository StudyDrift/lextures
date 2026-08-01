package com.lextures.android.core.lms

import java.io.File
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ContentToolPack1LogicTest {
    private val json = Json { ignoreUnknownKeys = true }

    private fun fixture(): JsonObject {
        val roots = listOf(
            File("clients/mobile/fixtures/content-tools/pack1-logic.json"),
            File("../mobile/fixtures/content-tools/pack1-logic.json"),
            File("../../../../../../mobile/fixtures/content-tools/pack1-logic.json"),
        )
        var dir = File(System.getProperty("user.dir") ?: ".")
        val candidates = roots.toMutableList()
        repeat(8) {
            candidates += File(dir, "clients/mobile/fixtures/content-tools/pack1-logic.json")
            candidates += File(dir, "mobile/fixtures/content-tools/pack1-logic.json")
            dir = dir.parentFile ?: return@repeat
        }
        val file = candidates.first { it.exists() }
        return json.parseToJsonElement(file.readText()).jsonObject
    }

    private fun answersMap(raw: JsonObject): Map<String, kotlinx.serialization.json.JsonElement> =
        raw.mapValues { (_, value) ->
            val obj = value.jsonObject
            val attempts = obj["attempts"]?.jsonArray ?: JsonArray(emptyList())
            buildJsonObject {
                put(
                    "attempts",
                    JsonArray(List(attempts.size) { buildJsonObject { put("correct", JsonPrimitive(false)) } }),
                )
            }
        }

    @Test
    fun allowlistMatchesFixture() {
        val cases = fixture()["allowlist"]!!.jsonObject["cases"]!!.jsonArray
        for (item in cases) {
            val obj = item.jsonObject
            val allowlist = obj["allowlist"]!!.jsonArray.map { it.jsonPrimitive.content }.toSet()
            val toolId = obj["toolId"]!!.jsonPrimitive.content
            val expected = obj["expected"]!!.jsonPrimitive.content.toBooleanStrict()
            assertEquals(obj["name"]?.jsonPrimitive?.content, expected, ContentToolPack1Logic.isClientAllowlisted(toolId, allowlist))
        }
    }

    @Test
    fun offlineQueueRulesMatchFixture() {
        val root = fixture()["offlineQueue"]!!.jsonObject
        for (item in root["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                "${obj["toolId"]!!.jsonPrimitive.content}/${obj["action"]!!.jsonPrimitive.content}",
                obj["expected"]!!.jsonPrimitive.content.toBooleanStrict(),
                ContentToolPack1Logic.canQueueActionOffline(
                    obj["toolId"]!!.jsonPrimitive.content,
                    obj["action"]!!.jsonPrimitive.content,
                ),
            )
        }
        val order = root["order"]!!.jsonObject
        val pending = order["input"]!!.jsonArray.map {
            val o = it.jsonObject
            ContentToolPack1Logic.PendingAction(
                instanceId = o["instanceId"]!!.jsonPrimitive.content,
                toolId = "flashcards",
                action = "rate",
                sequence = o["sequence"]!!.jsonPrimitive.int.toLong(),
                payloadJSON = "{}",
            )
        }
        val ordered = ContentToolPack1Logic.orderPendingActions(pending)
        assertEquals(
            order["expectedInstanceOrder"]!!.jsonArray.map { it.jsonPrimitive.content },
            ordered.map { it.instanceId },
        )
        assertEquals(
            order["expectedSequenceOrder"]!!.jsonArray.map { it.jsonPrimitive.int.toLong() },
            ordered.map { it.sequence },
        )
    }

    @Test
    fun attemptGatingMatchesFixture() {
        for (item in fixture()["attempts"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val maxAttempts = when (val raw = obj["maxAttempts"]) {
                null, JsonNull -> null
                else -> raw.jsonPrimitive.int
            }
            assertEquals(
                obj["name"]!!.jsonPrimitive.content,
                obj["expected"]!!.jsonPrimitive.content.toBooleanStrict(),
                ContentToolPack1Logic.canSubmit(
                    answers = answersMap(obj["answers"]!!.jsonObject),
                    questionId = obj["questionId"]!!.jsonPrimitive.content,
                    maxAttempts = maxAttempts,
                    readOnly = obj["readOnly"]!!.jsonPrimitive.content.toBooleanStrict(),
                ),
            )
        }
    }

    @Test
    fun sequentialUnlockMatchesFixture() {
        val root = fixture()["sequential"]!!.jsonObject
        val questions = root["questions"]!!.jsonArray.map { it.jsonPrimitive.content }
        for (item in root["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["name"]!!.jsonPrimitive.content,
                obj["expected"]!!.jsonPrimitive.content.toBooleanStrict(),
                ContentToolPack1Logic.isSequentiallyUnlocked(
                    questions = questions,
                    answers = answersMap(obj["answers"]!!.jsonObject),
                    questionId = obj["questionId"]!!.jsonPrimitive.content,
                    sequential = obj["sequential"]!!.jsonPrimitive.content.toBooleanStrict(),
                ),
            )
        }
    }

    @Test
    fun questionsAtATimePagingHelpers() {
        assertEquals(null, ContentToolPack1Logic.parseQuestionsAtATime(JsonPrimitive("all")))
        assertEquals(null, ContentToolPack1Logic.parseQuestionsAtATime(null))
        assertEquals(1, ContentToolPack1Logic.parseQuestionsAtATime(JsonPrimitive(1)))
        assertEquals(2, ContentToolPack1Logic.parseQuestionsAtATime(JsonPrimitive(2)))
        assertEquals(null, ContentToolPack1Logic.parseQuestionsAtATime(JsonPrimitive(9)))

        assertEquals(0 until 3, ContentToolPack1Logic.pageWindow(3, null, 0))
        assertEquals(0 until 1, ContentToolPack1Logic.pageWindow(3, 1, 0))
        assertEquals(1 until 2, ContentToolPack1Logic.pageWindow(3, 1, 1))
        assertEquals(0 until 2, ContentToolPack1Logic.pageWindow(3, 2, 0))
        assertEquals(2 until 3, ContentToolPack1Logic.pageWindow(3, 2, 1))
        assertEquals(1, ContentToolPack1Logic.initialPageIndex(3, 1, 2))
        assertEquals(0, ContentToolPack1Logic.initialPageIndex(3, 2, 1))
    }

    @Test
    fun predictRevealGatingMatchesFixture() {
        for (item in fixture()["predictReveal"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val committedAt = obj["committedAt"]
            val state = if (committedAt == null || committedAt is JsonNull || committedAt.jsonPrimitive.contentOrNull.isNullOrEmpty()) {
                buildJsonObject {}
            } else {
                buildJsonObject { put("committedAt", JsonPrimitive(committedAt.jsonPrimitive.content)) }
            }
            val committed = ContentToolPack1Logic.isCommitted(state)
            val hasReveal = obj["hasReveal"]!!.jsonPrimitive.content.toBooleanStrict()
            assertEquals(obj["canShow"]!!.jsonPrimitive.content.toBooleanStrict(), ContentToolPack1Logic.canShowReveal(committed, hasReveal))
            assertEquals(obj["canEdit"]!!.jsonPrimitive.content.toBooleanStrict(), ContentToolPack1Logic.canEditPrediction(committed, false))
        }
    }

    @Test
    fun classPulsePollMatchesFixture() {
        val root = fixture()["classPulsePoll"]!!.jsonObject
        assertEquals(root["baseMs"]!!.jsonPrimitive.int, ContentToolPack1Logic.CLASS_PULSE_POLL_INTERVAL_MS)
        for (item in root["visibility"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["expected"]!!.jsonPrimitive.content.toBooleanStrict(),
                ContentToolPack1Logic.shouldPollAggregate(
                    obj["visible"]!!.jsonPrimitive.content.toBooleanStrict(),
                    obj["hasVoted"]!!.jsonPrimitive.content.toBooleanStrict(),
                ),
            )
        }
        for (item in root["backoff"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["expected"]!!.jsonPrimitive.int,
                ContentToolPack1Logic.nextPollDelayMs(obj["failures"]!!.jsonPrimitive.int),
            )
        }
    }

    @Test
    fun flashcardsRatingsAndReviewKeys() {
        val root = fixture()["flashcards"]!!.jsonObject
        for (rating in root["validRatings"]!!.jsonArray.map { it.jsonPrimitive.content }) {
            assertTrue(ContentToolPack1Logic.isValidRating(rating))
        }
        for (rating in root["invalidRatings"]!!.jsonArray.map { it.jsonPrimitive.content }) {
            // "AGAIN " keeps trailing space — ratings must match exactly after lowercasing.
            assertFalse(rating, ContentToolPack1Logic.isValidRating(rating))
        }
        assertEquals(
            root["reviewCacheKeys"]!!.jsonArray.map { it.jsonPrimitive.content },
            ContentToolPack1Logic.reviewCacheKeysToInvalidate(),
        )
        assertFalse(ContentToolPack1Logic.shouldDoubleCountReviewSubmit("flashcards"))
    }

    @Test
    fun conflictPolicyAndUnknownPreservation() {
        for (item in fixture()["conflictPolicy"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val toolId = obj["toolId"]!!.jsonPrimitive.content
            val expected = obj["expected"]!!.jsonPrimitive.content
            assertEquals(expected, ContentToolPack1Logic.conflictPolicy(toolId).name.lowercase().let {
                when (it) {
                    "merge" -> "merge"
                    "server_wins" -> "server_wins"
                    else -> it
                }
            }.let {
                when (ContentToolPack1Logic.conflictPolicy(toolId)) {
                    ContentToolHostLogic.ConflictPolicy.MERGE -> "merge"
                    ContentToolHostLogic.ConflictPolicy.SERVER_WINS -> "server_wins"
                    ContentToolHostLogic.ConflictPolicy.CLIENT_WINS -> "client_wins"
                }
            })
            assertEquals(
                when (expected) {
                    "merge" -> ContentToolHostLogic.ConflictPolicy.MERGE
                    else -> ContentToolHostLogic.ConflictPolicy.SERVER_WINS
                },
                ContentToolHostLogic.conflictPolicyForTool(toolId),
            )
        }
        val preserve = fixture()["unknownFieldPreservation"]!!.jsonObject
        val base = mapOf(
            "v" to JsonPrimitive(1),
            "futureField" to JsonPrimitive("keep-me"),
            "drafts" to buildJsonObject { put("q1", JsonPrimitive("a")) },
        )
        val merged = ContentToolPack1Logic.mergePreservingUnknown(
            base,
            mapOf("drafts" to buildJsonObject { put("q1", JsonPrimitive("b")) }),
        )
        assertEquals(
            preserve["expectedKeys"]!!.jsonArray.map { it.jsonPrimitive.content }.toSet(),
            merged.keys,
        )
        assertEquals("b", merged["drafts"]!!.jsonObject["q1"]!!.jsonPrimitive.content)
    }

    @Test
    fun registeredNativeIncludesPack1() {
        val ids = ContentToolHostLogic.registeredNativeToolIds()
        assertTrue(ids.contains("noop_probe"))
        for (toolId in ContentToolPack1Logic.pack1ToolIds) {
            assertTrue(toolId, ids.contains(toolId))
        }
        assertFalse(ContentToolHostLogic.shouldShowUnsupportedPlaceholder("inline_questions", 1))
        // Pack-2 registers ask_questions when allowlisted (CT.M6).
        assertFalse(ContentToolHostLogic.shouldShowUnsupportedPlaceholder("ask_questions", 1))
    }
}
