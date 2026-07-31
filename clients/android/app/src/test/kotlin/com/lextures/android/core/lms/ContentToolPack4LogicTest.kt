package com.lextures.android.core.lms

import com.lextures.android.features.contenttools.ToolRegistry
import java.io.File
import kotlin.math.abs
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.double
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.longOrNull
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ContentToolPack4LogicTest {
    private val json = Json { ignoreUnknownKeys = true }

    private fun fixture(): JsonObject {
        val roots = listOf(
            File("clients/mobile/fixtures/content-tools/pack4-logic.json"),
            File("../mobile/fixtures/content-tools/pack4-logic.json"),
            File("../../../../../../mobile/fixtures/content-tools/pack4-logic.json"),
        )
        var dir = File(System.getProperty("user.dir") ?: ".")
        val candidates = roots.toMutableList()
        repeat(8) {
            candidates += File(dir, "clients/mobile/fixtures/content-tools/pack4-logic.json")
            candidates += File(dir, "mobile/fixtures/content-tools/pack4-logic.json")
            dir = dir.parentFile ?: return@repeat
        }
        val file = candidates.first { it.exists() }
        return json.parseToJsonElement(file.readText()).jsonObject
    }

    private fun policyWire(policy: ContentToolHostLogic.ConflictPolicy): String = when (policy) {
        ContentToolHostLogic.ConflictPolicy.MERGE -> "merge"
        ContentToolHostLogic.ConflictPolicy.SERVER_WINS -> "server_wins"
        ContentToolHostLogic.ConflictPolicy.CLIENT_WINS -> "client_wins"
    }

    private fun stringList(value: kotlinx.serialization.json.JsonElement): List<String> =
        value.jsonArray.map { it.jsonPrimitive.content }

    private fun optionalString(el: kotlinx.serialization.json.JsonElement?): String? =
        if (el == null || el is JsonNull) null else el.jsonPrimitive.contentOrNull

    private fun optionalLong(el: kotlinx.serialization.json.JsonElement?): Long? =
        if (el == null || el is JsonNull) null else el.jsonPrimitive.longOrNull
            ?: el.jsonPrimitive.doubleOrNull?.toLong()

    private fun optionalDouble(el: kotlinx.serialization.json.JsonElement?): Double? =
        if (el == null || el is JsonNull) null else el.jsonPrimitive.doubleOrNull

    private fun assertClose(expected: Double, actual: Double, msg: String = "") {
        assertTrue("$msg expected=$expected actual=$actual", abs(expected - actual) < 1e-9)
    }

    private fun checkpoints(root: JsonObject): List<ContentToolPack4Logic.Checkpoint> =
        root["checkpoints"]!!.jsonArray.map { item ->
            val obj = item.jsonObject
            ContentToolPack4Logic.Checkpoint(
                id = obj["id"]!!.jsonPrimitive.content,
                atSec = obj["atSec"]!!.jsonPrimitive.double,
                required = obj["required"]!!.jsonPrimitive.boolean,
                attempts = obj["attempts"]!!.jsonPrimitive.int,
            )
        }

    private fun answers(obj: JsonObject): Map<String, ContentToolPack4Logic.CheckpointAnswer> =
        obj.mapValues { (_, value) ->
            val a = value.jsonObject
            val attempts = a["attempts"]?.jsonArray.orEmpty()
            val lastCorrect = attempts.lastOrNull()?.jsonObject?.get("correct")?.jsonPrimitive?.booleanOrNull ?: false
            ContentToolPack4Logic.CheckpointAnswer(
                done = a["done"]?.jsonPrimitive?.booleanOrNull ?: false,
                attemptCount = attempts.size,
                lastCorrect = lastCorrect,
            )
        }

    @Test
    fun allowlistMatchesFixture() {
        val root = fixture()["allowlist"]!!.jsonObject
        val pack4ToolIds = root["pack4ToolIds"]!!.jsonArray.map { it.jsonPrimitive.content }.toSet()
        assertEquals(pack4ToolIds, ContentToolPack4Logic.pack4ToolIds)
        val sandboxIds = root["sandboxToolIds"]!!.jsonArray.map { it.jsonPrimitive.content }.toSet()
        assertEquals(sandboxIds, ContentToolPack4Logic.sandboxToolIds)
        for (item in root["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val allowlist = obj["allowlist"]!!.jsonArray.map { it.jsonPrimitive.content }.toSet()
            val toolId = obj["toolId"]!!.jsonPrimitive.content
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack4Logic.isClientAllowlisted(toolId, allowlist),
            )
        }
    }

    @Test
    fun allowlistedToolIdsEqualsPack4ToolIds() {
        assertEquals(ContentToolPack4Logic.pack4ToolIds, ContentToolPack4Logic.allowlistedToolIds())
        assertFalse(ContentToolPack4Logic.allowlistedToolIds().contains("code_sandbox"))
    }

    @Test
    fun conflictPolicyMatchesFixture() {
        for (item in fixture()["conflictPolicy"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val toolId = obj["toolId"]!!.jsonPrimitive.content
            val expected = obj["expected"]!!.jsonPrimitive.content
            assertEquals(expected, policyWire(ContentToolPack4Logic.conflictPolicy(toolId)))
            assertEquals(expected, policyWire(ContentToolHostLogic.conflictPolicyForTool(toolId)))
        }
    }

    @Test
    fun offlineQueueNeverQueues() {
        for (item in fixture()["offlineQueue"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val toolId = obj["toolId"]!!.jsonPrimitive.content
            val action = obj["action"]!!.jsonPrimitive.content
            assertEquals(
                "$toolId/$action",
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack4Logic.canQueueActionOffline(toolId, action),
            )
        }
    }

    @Test
    fun checkpointEngineMatchesFixture() {
        val root = fixture()["checkpointEngine"]!!.jsonObject
        assertClose(root["toleranceSec"]!!.jsonPrimitive.double, ContentToolPack4Logic.CHECKPOINT_TOLERANCE_SEC)
        val cps = checkpoints(root)

        for (item in root["findDue"]!!.jsonArray) {
            val obj = item.jsonObject
            val due = ContentToolPack4Logic.findDueCheckpoint(
                cps,
                answers(obj["answers"]!!.jsonObject),
                obj["currentTime"]!!.jsonPrimitive.double,
                obj["prompted"]!!.jsonArray.map { it.jsonPrimitive.content }.toSet(),
            )
            assertEquals(obj["name"]?.jsonPrimitive?.content, optionalString(obj["expectedId"]), due?.id)
        }

        for (item in root["clampSeek"]!!.jsonArray) {
            val obj = item.jsonObject
            val result = ContentToolPack4Logic.clampSeekTime(
                obj["preventSkip"]!!.jsonPrimitive.boolean,
                cps,
                answers(obj["answers"]!!.jsonObject),
                obj["targetSec"]!!.jsonPrimitive.double,
            )
            assertClose(obj["expectedTime"]!!.jsonPrimitive.double, result.time, obj["name"]?.jsonPrimitive?.content.orEmpty())
            assertEquals(obj["name"]?.jsonPrimitive?.content, obj["expectedClamped"]!!.jsonPrimitive.boolean, result.clamped)
        }

        for (item in root["isDone"]!!.jsonArray) {
            val obj = item.jsonObject
            val id = obj["checkpointId"]!!.jsonPrimitive.content
            val cp = cps.first { it.id == id }
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack4Logic.isCheckpointDone(answers(obj["answers"]!!.jsonObject), cp),
            )
        }

        for (item in root["mergeSegments"]!!.jsonArray) {
            val obj = item.jsonObject
            val existing = obj["existing"]!!.jsonArray.map { seg ->
                seg.jsonArray.map { it.jsonPrimitive.double }
            }
            val merged = ContentToolPack4Logic.mergeLocalSegments(
                existing,
                obj["start"]!!.jsonPrimitive.double,
                obj["end"]!!.jsonPrimitive.double,
            )
            val expected = obj["expected"]!!.jsonArray.map { seg ->
                seg.jsonArray.map { it.jsonPrimitive.double }
            }
            assertEquals(obj["name"]?.jsonPrimitive?.content, expected.size, merged.size)
            for (i in expected.indices) {
                assertClose(expected[i][0], merged[i][0])
                assertClose(expected[i][1], merged[i][1])
            }
        }

        for (item in root["resume"]!!.jsonArray) {
            val obj = item.jsonObject
            val segs = obj["watchedSegments"]!!.jsonArray.map { seg ->
                seg.jsonArray.map { it.jsonPrimitive.double }
            }
            assertClose(
                obj["expected"]!!.jsonPrimitive.double,
                ContentToolPack4Logic.resumePosition(optionalDouble(obj["furthestSec"]), segs),
                obj["name"]?.jsonPrimitive?.content.orEmpty(),
            )
        }

        val throttle = root["progressThrottle"]!!.jsonObject
        assertEquals(throttle["intervalMs"]!!.jsonPrimitive.int.toLong(), ContentToolPack4Logic.PROGRESS_THROTTLE_MS)
        for (item in throttle["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack4Logic.shouldFireProgressThrottle(
                    optionalLong(obj["lastFiredAtMs"]),
                    obj["nowMs"]!!.jsonPrimitive.longOrNull ?: obj["nowMs"]!!.jsonPrimitive.double.toLong(),
                ),
            )
        }

        for (item in root["playbackBlocked"]!!.jsonArray) {
            val obj = item.jsonObject
            val id = obj["checkpointId"]!!.jsonPrimitive.content
            val cp = cps.first { it.id == id }
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack4Logic.shouldBlockPlayback(cp, answers(obj["answers"]!!.jsonObject)),
            )
        }
    }

    @Test
    fun workedExampleMatchesFixture() {
        val root = fixture()["workedExample"]!!.jsonObject
        val allIds = root["steps"]!!.jsonArray.map { it.jsonObject["id"]!!.jsonPrimitive.content }

        for (item in root["status"]!!.jsonArray) {
            val obj = item.jsonObject
            val blanked = stringList(obj["blankedStepIds"]!!)
            val current = obj["currentStepId"]!!.jsonPrimitive.content
            val progress = obj["progress"]!!.jsonObject
            val expected = obj["expected"]!!.jsonObject
            for (stepId in allIds) {
                val status = ContentToolPack4Logic.stepStatus(
                    stepId,
                    blanked,
                    current,
                    progress,
                    allIds,
                )
                assertEquals(
                    "${obj["name"]?.jsonPrimitive?.content}/$stepId",
                    expected[stepId]!!.jsonPrimitive.content,
                    status.wire,
                )
            }
        }

        for (item in root["stepDone"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack4Logic.isStepDone(obj["progress"]),
            )
        }

        for (item in root["canCheck"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack4Logic.canCheckStep(
                    obj["draft"]!!.jsonPrimitive.content,
                    obj["readOnly"]!!.jsonPrimitive.boolean,
                    obj["busy"]!!.jsonPrimitive.boolean,
                    obj["stepDone"]!!.jsonPrimitive.boolean,
                ),
            )
        }

        for (item in root["resolveCurrent"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expected"]!!.jsonPrimitive.content,
                ContentToolPack4Logic.resolveCurrentStepId(
                    stringList(obj["blankedStepIds"]!!),
                    optionalString(obj["currentStepId"]),
                    obj["progress"]!!.jsonObject,
                ),
            )
        }
    }

    @Test
    fun parameterExplorerMatchesFixture() {
        val root = fixture()["parameterExplorer"]!!.jsonObject
        assertEquals(root["recomputeThrottleMs"]!!.jsonPrimitive.int.toLong(), ContentToolPack4Logic.RECOMPUTE_THROTTLE_MS)

        for (item in root["throttle"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack4Logic.shouldRecompute(
                    optionalLong(obj["lastAtMs"]),
                    obj["nowMs"]!!.jsonPrimitive.longOrNull ?: obj["nowMs"]!!.jsonPrimitive.double.toLong(),
                ),
            )
        }

        for (item in root["settle"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack4Logic.shouldAutosaveOnSettle(
                    obj["dragging"]!!.jsonPrimitive.boolean,
                    obj["dirty"]!!.jsonPrimitive.boolean,
                ),
            )
        }

        val defaultsRoot = root["defaults"]!!.jsonObject
        val config = JsonObject(mapOf("parameters" to defaultsRoot["parameters"]!!))
        val defaults = ContentToolPack4Logic.defaultParams(config)
        val expected = defaultsRoot["expected"]!!.jsonObject
        assertEquals(expected.size, defaults.size)
        for ((k, v) in expected) {
            val actual = defaults[k] as JsonPrimitive
            when {
                v.jsonPrimitive.booleanOrNull != null ->
                    assertEquals(k, v.jsonPrimitive.boolean, actual.booleanOrNull)
                v.jsonPrimitive.doubleOrNull != null && v.jsonPrimitive.contentOrNull?.toDoubleOrNull() != null &&
                    !v.jsonPrimitive.content.contains('"') ->
                    assertClose(v.jsonPrimitive.double, actual.doubleOrNull ?: 0.0, k)
                else ->
                    assertEquals(k, v.jsonPrimitive.content, actual.content)
            }
        }

        for (item in root["clampNumber"]!!.jsonArray) {
            val obj = item.jsonObject
            assertClose(
                obj["expected"]!!.jsonPrimitive.double,
                ContentToolPack4Logic.clampNumber(
                    obj["value"]!!.jsonPrimitive.double,
                    obj["min"]!!.jsonPrimitive.double,
                    obj["max"]!!.jsonPrimitive.double,
                    obj["step"]!!.jsonPrimitive.double,
                ),
                obj["name"]?.jsonPrimitive?.content.orEmpty(),
            )
        }
    }

    @Test
    fun unknownFieldPreservation() {
        val root = fixture()["unknownFieldPreservation"]!!.jsonObject
        val base = root["base"]!!.jsonObject
        val patch = root["patch"]!!.jsonObject
        val merged = ContentToolPack4Logic.mergePreservingUnknown(base, patch)
        val expectedKeys = stringList(root["expectedKeys"]!!).toSet()
        assertEquals(expectedKeys, merged.keys)
        assertEquals("keep-me", (merged["customClientKey"] as JsonPrimitive).content)
    }

    @Test
    fun mediaProviderReliability() {
        for (item in fixture()["mediaProvider"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expectedReliable"]!!.jsonPrimitive.boolean,
                ContentToolPack4Logic.hasReliableCheckpointTiming(
                    optionalString(obj["source"]),
                    optionalString(obj["url"]),
                    optionalString(obj["provider"]),
                ),
            )
        }
    }

    @Test
    fun registryIncludesPack4NativeTools() {
        for (toolId in ContentToolPack4Logic.pack4ToolIds) {
            assertTrue(toolId, ContentToolHostLogic.hasNativeRenderer(toolId))
        }
        assertFalse(ContentToolHostLogic.hasNativeRenderer("code_sandbox"))
        assertTrue(ToolRegistry.registeredIds().containsAll(ContentToolPack4Logic.pack4ToolIds))
        assertFalse(ToolRegistry.registeredIds().contains("code_sandbox"))
    }
}
