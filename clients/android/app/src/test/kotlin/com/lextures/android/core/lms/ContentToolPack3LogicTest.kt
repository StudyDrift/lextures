package com.lextures.android.core.lms

import java.io.File
import kotlin.math.abs
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.double
import kotlinx.serialization.json.int
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ContentToolPack3LogicTest {
    private val json = Json { ignoreUnknownKeys = true }

    private fun fixture(): JsonObject {
        val roots = listOf(
            File("clients/mobile/fixtures/content-tools/pack3-logic.json"),
            File("../mobile/fixtures/content-tools/pack3-logic.json"),
            File("../../../../../../mobile/fixtures/content-tools/pack3-logic.json"),
        )
        var dir = File(System.getProperty("user.dir") ?: ".")
        val candidates = roots.toMutableList()
        repeat(8) {
            candidates += File(dir, "clients/mobile/fixtures/content-tools/pack3-logic.json")
            candidates += File(dir, "mobile/fixtures/content-tools/pack3-logic.json")
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

    private fun optionalInt(el: kotlinx.serialization.json.JsonElement?): Int? =
        if (el == null || el is JsonNull) null else el.jsonPrimitive.intOrNull

    private fun categorizePlacement(obj: JsonObject): Map<String, String?> =
        obj.mapValues { (_, v) ->
            when (v) {
                is JsonNull -> null
                else -> v.jsonPrimitive.contentOrNull
            }
        }

    private fun placementHit(obj: JsonObject): ContentToolPack3Logic.PlacementHit =
        when (obj["type"]!!.jsonPrimitive.content) {
            "item" -> ContentToolPack3Logic.PlacementHit.Item(obj["id"]!!.jsonPrimitive.content)
            "bucket" -> ContentToolPack3Logic.PlacementHit.Bucket(obj["id"]!!.jsonPrimitive.content)
            "tray" -> ContentToolPack3Logic.PlacementHit.Tray
            "position" -> ContentToolPack3Logic.PlacementHit.Position(obj["index"]!!.jsonPrimitive.int)
            else -> error("unknown hit type ${obj["type"]}")
        }

    private fun assertPlacementMapsEqual(actual: Map<String, String?>, expected: Map<String, String?>, msg: String) {
        assertEquals(msg, expected.keys, actual.keys)
        for (key in expected.keys) {
            assertEquals("$msg key=$key", expected[key], actual[key])
        }
    }

    private fun assertClose(expected: Double, actual: Double, msg: String = "") {
        assertTrue("$msg expected=$expected actual=$actual", abs(expected - actual) < 1e-9)
    }

    @Test
    fun allowlistMatchesFixture() {
        val root = fixture()["allowlist"]!!.jsonObject
        val pack3ToolIds = root["pack3ToolIds"]!!.jsonArray.map { it.jsonPrimitive.content }.toSet()
        assertEquals(pack3ToolIds, ContentToolPack3Logic.pack3ToolIds)
        for (item in root["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val allowlist = obj["allowlist"]!!.jsonArray.map { it.jsonPrimitive.content }.toSet()
            val toolId = obj["toolId"]!!.jsonPrimitive.content
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack3Logic.isClientAllowlisted(toolId, allowlist),
            )
        }
    }

    @Test
    fun allowlistedToolIdsEqualsPack3ToolIds() {
        assertEquals(ContentToolPack3Logic.pack3ToolIds, ContentToolPack3Logic.allowlistedToolIds())
    }

    @Test
    fun conflictPolicyMatchesFixture() {
        for (item in fixture()["conflictPolicy"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val toolId = obj["toolId"]!!.jsonPrimitive.content
            val expected = obj["expected"]!!.jsonPrimitive.content
            assertEquals(expected, policyWire(ContentToolPack3Logic.conflictPolicy(toolId)))
            assertEquals(expected, policyWire(ContentToolHostLogic.conflictPolicyForTool(toolId)))
        }
    }

    @Test
    fun offlineQueueNeverQueues() {
        for (item in fixture()["offlineQueue"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                "${obj["toolId"]!!.jsonPrimitive.content}/${obj["action"]!!.jsonPrimitive.content}",
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack3Logic.canQueueActionOffline(
                    obj["toolId"]!!.jsonPrimitive.content,
                    obj["action"]!!.jsonPrimitive.content,
                ),
            )
        }
    }

    @Test
    fun attemptsMatchesFixture() {
        val root = fixture()["attempts"]!!.jsonObject
        for (item in root["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val rawEl = obj["raw"]
            val raw = if (rawEl == null || rawEl is JsonNull) null else rawEl
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                optionalInt(obj["expectedMax"]),
                ContentToolPack3Logic.parseAttemptsConfig(raw),
            )
        }
        for (item in root["canCheck"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack3Logic.canCheck(
                    attemptsUsed = obj["attemptsUsed"]!!.jsonPrimitive.int,
                    maxAttempts = optionalInt(obj["maxAttempts"]),
                    readOnly = obj["readOnly"]!!.jsonPrimitive.boolean,
                ),
            )
        }
    }

    @Test
    fun sortReorderMatchesFixture() {
        for (item in fixture()["sortReorder"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val result = ContentToolPack3Logic.moveInOrder(
                order = stringList(obj["order"]!!),
                itemId = obj["itemId"]!!.jsonPrimitive.content,
                direction = obj["direction"]!!.jsonPrimitive.int,
                lockedItemIds = stringList(obj["locked"]!!),
            )
            assertEquals(obj["name"]?.jsonPrimitive?.content, stringList(obj["expected"]!!), result)
        }
    }

    @Test
    fun sortTapAssignCategorizeMatchesFixture() {
        for (item in fixture()["sortTapAssign"]!!.jsonObject["categorize"]!!.jsonArray) {
            val obj = item.jsonObject
            val state = ContentToolPack3Logic.EngineState(
                grabbedId = optionalString(obj["grabbedId"]),
                placement = ContentToolPack3Logic.Placement.Categorize(
                    categorizePlacement(obj["placement"]!!.jsonObject),
                ),
            )
            val next = ContentToolPack3Logic.tapItemOrTarget(
                state,
                ContentToolPack3Logic.PlacementMode.CATEGORIZE,
                emptyList(),
                placementHit(obj["hit"]!!.jsonObject),
            )
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                optionalString(obj["expectedGrabbed"]),
                next.grabbedId,
            )
            val actual = (next.placement as ContentToolPack3Logic.Placement.Categorize).map
            assertPlacementMapsEqual(
                actual,
                categorizePlacement(obj["expectedPlacement"]!!.jsonObject),
                obj["name"]?.jsonPrimitive?.content ?: "",
            )
        }
    }

    @Test
    fun sortTapAssignOrderMatchesFixture() {
        for (item in fixture()["sortTapAssign"]!!.jsonObject["order"]!!.jsonArray) {
            val obj = item.jsonObject
            val state = ContentToolPack3Logic.EngineState(
                grabbedId = optionalString(obj["grabbedId"]),
                placement = ContentToolPack3Logic.Placement.Order(stringList(obj["placement"]!!)),
            )
            val next = ContentToolPack3Logic.tapItemOrTarget(
                state,
                ContentToolPack3Logic.PlacementMode.ORDER,
                emptyList(),
                placementHit(obj["hit"]!!.jsonObject),
            )
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                optionalString(obj["expectedGrabbed"]),
                next.grabbedId,
            )
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                stringList(obj["expectedPlacement"]!!),
                (next.placement as ContentToolPack3Logic.Placement.Order).ids,
            )
        }
    }

    @Test
    fun dragInterruptMatchesFixture() {
        for (item in fixture()["dragInterrupt"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val settled = ContentToolPack3Logic.EngineState(
                grabbedId = optionalString(obj["inFlightGrabbed"]),
                target = ContentToolPack3Logic.PlacementTarget.Tray,
                placement = ContentToolPack3Logic.Placement.Categorize(
                    categorizePlacement(obj["settledPlacement"]!!.jsonObject),
                ),
            )
            val restored = ContentToolPack3Logic.restoreAfterDragInterrupt(settled)
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                optionalString(obj["expectedGrabbed"]),
                restored.grabbedId,
            )
            assertPlacementMapsEqual(
                (restored.placement as ContentToolPack3Logic.Placement.Categorize).map,
                categorizePlacement(obj["expectedPlacement"]!!.jsonObject),
                obj["name"]?.jsonPrimitive?.content ?: "",
            )
        }
    }

    @Test
    fun allPlacedMatchesFixture() {
        for (item in fixture()["allPlaced"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val mode = when (obj["mode"]!!.jsonPrimitive.content) {
                "categorize" -> ContentToolPack3Logic.PlacementMode.CATEGORIZE
                "order" -> ContentToolPack3Logic.PlacementMode.ORDER
                else -> error("unknown mode")
            }
            val itemIds = stringList(obj["itemIds"]!!)
            val placement = when (mode) {
                ContentToolPack3Logic.PlacementMode.CATEGORIZE ->
                    ContentToolPack3Logic.Placement.Categorize(categorizePlacement(obj["placement"]!!.jsonObject))
                ContentToolPack3Logic.PlacementMode.ORDER ->
                    ContentToolPack3Logic.Placement.Order(stringList(obj["placement"]!!))
            }
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack3Logic.allPlaced(mode, itemIds, placement),
            )
        }
    }

    @Test
    fun anchorsBuildResolveSegmentMatchFixture() {
        val root = fixture()["anchors"]!!.jsonObject
        assertEquals(root["contextLen"]!!.jsonPrimitive.int, ContentToolPack3Logic.CONTEXT_LEN)

        for (item in root["build"]!!.jsonArray) {
            val obj = item.jsonObject
            val built = ContentToolPack3Logic.buildQuoteAnchor(
                passage = obj["passage"]!!.jsonPrimitive.content,
                start = obj["start"]!!.jsonPrimitive.int,
                end = obj["end"]!!.jsonPrimitive.int,
            )
            assertNotNull(obj["name"]?.jsonPrimitive?.content, built)
            assertEquals(obj["expectedQuote"]!!.jsonPrimitive.content, built!!.first)
            assertEquals(obj["expectedPrefix"]!!.jsonPrimitive.content, built.second.prefix)
            assertEquals(obj["expectedSuffix"]!!.jsonPrimitive.content, built.second.suffix)
            assertEquals(obj["expectedOffset"]!!.jsonPrimitive.int, built.second.approxOffset)
        }

        for (item in root["resolve"]!!.jsonArray) {
            val obj = item.jsonObject
            val anchorObj = obj["anchor"]!!.jsonObject
            val anchor = ContentToolPack3Logic.QuoteAnchor(
                prefix = anchorObj["prefix"]!!.jsonPrimitive.content,
                suffix = anchorObj["suffix"]!!.jsonPrimitive.content,
                approxOffset = anchorObj["approxOffset"]!!.jsonPrimitive.int,
            )
            val resolved = ContentToolPack3Logic.resolveQuoteAnchor(
                passage = obj["passage"]!!.jsonPrimitive.content,
                quote = obj["quote"]!!.jsonPrimitive.content,
                anchor = anchor,
            )
            val expectedStart = optionalInt(obj["expectedStart"])
            val expectedEnd = optionalInt(obj["expectedEnd"])
            if (expectedStart == null && expectedEnd == null) {
                assertNull(obj["name"]?.jsonPrimitive?.content, resolved)
            } else {
                assertNotNull(obj["name"]?.jsonPrimitive?.content, resolved)
                assertEquals(expectedStart, resolved!!.start)
                assertEquals(expectedEnd, resolved.end)
            }
        }

        for (item in root["segment"]!!.jsonArray) {
            val obj = item.jsonObject
            val units = ContentToolPack3Logic.segmentPassage(
                obj["passage"]!!.jsonPrimitive.content,
                obj["granularity"]!!.jsonPrimitive.content,
            )
            val expected = obj["expected"]!!.jsonArray
            assertEquals(obj["name"]?.jsonPrimitive?.content, expected.size, units.size)
            for ((unit, expEl) in units.zip(expected)) {
                val exp = expEl.jsonObject
                assertEquals(exp["index"]!!.jsonPrimitive.int, unit.index)
                assertEquals(exp["text"]!!.jsonPrimitive.content, unit.text)
                assertEquals(exp["start"]!!.jsonPrimitive.int, unit.start)
                assertEquals(exp["end"]!!.jsonPrimitive.int, unit.end)
            }
        }
    }

    @Test
    fun geometryMatchesFixture() {
        val root = fixture()["geometry"]!!.jsonObject

        for (item in root["clamp01"]!!.jsonArray) {
            val obj = item.jsonObject
            assertClose(
                obj["expected"]!!.jsonPrimitive.double,
                ContentToolPack3Logic.clamp01(obj["input"]!!.jsonPrimitive.double),
            )
        }

        for (item in root["pointInShape"]!!.jsonArray) {
            val obj = item.jsonObject
            val shape = ContentToolPack3Logic.parseShape(obj["shape"])!!
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack3Logic.pointInShape(
                    obj["x"]!!.jsonPrimitive.double,
                    obj["y"]!!.jsonPrimitive.double,
                    shape,
                ),
            )
        }

        for (item in root["hitTest"]!!.jsonArray) {
            val obj = item.jsonObject
            val regions = obj["regions"]!!.jsonArray.map { regionEl ->
                val region = regionEl.jsonObject
                ContentToolPack3Logic.DiagramRegion(
                    id = region["id"]!!.jsonPrimitive.content,
                    label = region["label"]!!.jsonPrimitive.content,
                    description = region["description"]!!.jsonPrimitive.content,
                    shape = ContentToolPack3Logic.parseShape(region["shape"])!!,
                )
            }
            val hit = ContentToolPack3Logic.hitTestRegions(
                regions,
                obj["x"]!!.jsonPrimitive.double,
                obj["y"]!!.jsonPrimitive.double,
            )
            assertEquals(obj["name"]?.jsonPrimitive?.content, obj["expectedId"]!!.jsonPrimitive.content, hit?.id)
        }

        for (item in root["hitTargetExpansion"]!!.jsonArray) {
            val obj = item.jsonObject
            val shape = ContentToolPack3Logic.parseShape(obj["shape"])!!
            val original = shape
            val point = obj["pointInsideExpanded"]!!.jsonArray
            val contains = ContentToolPack3Logic.pointInExpandedHitTarget(
                x = point[0].jsonPrimitive.double,
                y = point[1].jsonPrimitive.double,
                shape = shape,
                imageDisplayWidthPt = obj["imageDisplayWidthPt"]!!.jsonPrimitive.double,
                imageDisplayHeightPt = obj["imageDisplayHeightPt"]!!.jsonPrimitive.double,
                minTargetPt = obj["minTargetPt"]!!.jsonPrimitive.double,
            )
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["expectedContains"]!!.jsonPrimitive.boolean,
                contains,
            )
            if (obj["storedShapeUnchanged"]!!.jsonPrimitive.boolean) {
                assertEquals(obj["name"]?.jsonPrimitive?.content, original, shape)
            }
        }

        for (item in root["pointerToNormalized"]!!.jsonArray) {
            val obj = item.jsonObject
            val point = ContentToolPack3Logic.pointerToNormalized(
                clientX = obj["clientX"]!!.jsonPrimitive.double,
                clientY = obj["clientY"]!!.jsonPrimitive.double,
                viewWidth = obj["viewWidth"]!!.jsonPrimitive.double,
                viewHeight = obj["viewHeight"]!!.jsonPrimitive.double,
                naturalWidth = obj["naturalWidth"]!!.jsonPrimitive.double,
                naturalHeight = obj["naturalHeight"]!!.jsonPrimitive.double,
                zoom = obj["zoom"]!!.jsonPrimitive.double,
                panX = obj["panX"]!!.jsonPrimitive.double,
                panY = obj["panY"]!!.jsonPrimitive.double,
            )
            assertNotNull(obj["name"]?.jsonPrimitive?.content, point)
            assertClose(obj["expectedX"]!!.jsonPrimitive.double, point!!.first, obj["name"]?.jsonPrimitive?.content ?: "")
            assertClose(obj["expectedY"]!!.jsonPrimitive.double, point.second, obj["name"]?.jsonPrimitive?.content ?: "")
        }
    }

    @Test
    fun checkResultMatchesFixture() {
        for (item in fixture()["checkResult"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val parsed = ContentToolPack3Logic.parseCheckResult(obj["result"])
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                optionalString(obj["expectedError"]),
                parsed.error?.code,
            )
            val expectedScore = obj["expectedScore"]
            if (expectedScore == null || expectedScore is JsonNull) {
                assertNull(obj["name"]?.jsonPrimitive?.content, parsed.scorePct)
            } else {
                assertClose(
                    expectedScore.jsonPrimitive.double,
                    parsed.scorePct!!,
                    obj["name"]?.jsonPrimitive?.content ?: "",
                )
            }
            assertEquals(optionalInt(obj["expectedAttemptsRemaining"]), parsed.attemptsRemaining)
        }
    }

    @Test
    fun registeredNativeIncludesPack3() {
        val ids = ContentToolHostLogic.registeredNativeToolIds()
        assertTrue(ids.contains("noop_probe"))
        for (toolId in ContentToolPack3Logic.pack3ToolIds) {
            assertTrue(toolId, ids.contains(toolId))
            assertFalse(ContentToolHostLogic.shouldShowUnsupportedPlaceholder(toolId, 1))
        }
    }
}
