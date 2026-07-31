package com.lextures.android.core.lms

import java.io.File
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ContentToolHostLogicTest {
    private val json = Json { ignoreUnknownKeys = true }

    private fun fixture(): JsonObject {
        return json.parseToJsonElement(resolveFixture().readText()).jsonObject
    }

    private fun resolveFixture(): File {
        var dir: File? = File(System.getProperty("user.dir") ?: ".")
        repeat(8) {
            val current = dir ?: return@repeat
            val candidates = listOf(
                File(current, "clients/mobile/fixtures/content-tools/host-logic.json"),
                File(current, "../mobile/fixtures/content-tools/host-logic.json"),
                File(current, "../../mobile/fixtures/content-tools/host-logic.json"),
                File(current, "mobile/fixtures/content-tools/host-logic.json"),
            )
            candidates.firstOrNull { it.isFile }?.let { return it.canonicalFile }
            dir = current.parentFile
        }
        error(
            "clients/mobile/fixtures/content-tools/host-logic.json not found from ${System.getProperty("user.dir")}"
        )
    }

    @Test
    fun clampDebounceMatchesFixture() {
        val cases = fixture()["debounce"]!!.jsonObject["cases"]!!.jsonArray
        for (case in cases) {
            val obj = case.jsonObject
            val inputEl = obj["input"]
            val expected = obj["expected"]!!.jsonPrimitive.intOrNull!!
            val actual = when {
                inputEl == null || inputEl is JsonNull -> ContentToolHostLogic.clampDebounceMs(null as Int?)
                inputEl.jsonPrimitive.intOrNull != null ->
                    ContentToolHostLogic.clampDebounceMs(inputEl.jsonPrimitive.intOrNull)
                else -> ContentToolHostLogic.clampDebounceMs(inputEl.jsonPrimitive.doubleOrNull)
            }
            assertEquals(expected, actual)
        }
    }

    @Test
    fun conflictPolicyMatchesFixture() {
        val cases = fixture()["conflictPolicy"]!!.jsonObject["cases"]!!.jsonArray
        for (case in cases) {
            val obj = case.jsonObject
            val policy = ContentToolHostLogic.ConflictPolicy.from(obj["policy"]!!.jsonPrimitive.content)
            val client = obj["client"]!!.jsonObject.mapValues { it.value }
            val server = obj["server"]!!.jsonObject.mapValues { it.value }
            val expected = obj["expected"]!!.jsonObject
            val resolved = ContentToolHostLogic.resolveConflictState(policy, client, server)
            assertEquals(expected, JsonObject(resolved))
        }
    }

    @Test
    fun readOnlyPrecedenceMatchesFixture() {
        val cases = fixture()["readOnlyPrecedence"]!!.jsonArray
        for (case in cases) {
            val obj = case.jsonObject
            val input = obj["input"]!!.jsonObject
            val expectedRaw = obj["expected"]
            val reason = ContentToolHostLogic.readOnlyReason(
                ContentToolHostLogic.ReadOnlyInput(
                    tombstone = input["tombstone"]!!.jsonPrimitive.content.toBooleanStrict(),
                    breakerOpen = input["breakerOpen"]!!.jsonPrimitive.content.toBooleanStrict(),
                    status = input["status"]!!.jsonPrimitive.content,
                    pastDue = input["pastDue"]!!.jsonPrimitive.content.toBooleanStrict(),
                    respectsDueDate = input["respectsDueDate"]!!.jsonPrimitive.content.toBooleanStrict(),
                    observer = input["observer"]!!.jsonPrimitive.content.toBooleanStrict(),
                ),
            )
            if (expectedRaw == null || expectedRaw is JsonNull) {
                assertNull(obj["name"]!!.jsonPrimitive.content, reason)
            } else {
                val expected = when (expectedRaw.jsonPrimitive.content) {
                    "tombstone" -> ContentToolHostLogic.ReadOnlyReason.TOMBSTONE
                    "breaker" -> ContentToolHostLogic.ReadOnlyReason.BREAKER
                    "archived" -> ContentToolHostLogic.ReadOnlyReason.ARCHIVED
                    "past_due" -> ContentToolHostLogic.ReadOnlyReason.PAST_DUE
                    "observer" -> ContentToolHostLogic.ReadOnlyReason.OBSERVER
                    else -> error("unknown reason")
                }
                assertEquals(obj["name"]!!.jsonPrimitive.content, expected, reason)
            }
        }
    }

    @Test
    fun contractGatingMatchesFixture() {
        val contract = fixture()["contract"]!!.jsonObject
        val supported = contract["supportedVersion"]!!.jsonPrimitive.intOrNull!!
        for (case in contract["cases"]!!.jsonArray) {
            val obj = case.jsonObject
            val value = obj["contract"]!!.jsonPrimitive.intOrNull!!
            val ok = obj["supported"]!!.jsonPrimitive.content.toBooleanStrict()
            assertEquals(ok, ContentToolHostLogic.contractSupported(value, supported))
        }
    }

    @Test
    fun fenceMappingMatchesFixture() {
        val mapping = fixture()["fenceMapping"]!!.jsonObject
        val instances = mapping["instances"]!!.jsonArray.map {
            val o = it.jsonObject
            ToolInstance(
                id = o["id"]!!.jsonPrimitive.content,
                toolId = o["toolId"]!!.jsonPrimitive.content,
            )
        }
        val map = ContentToolHostLogic.instanceMap(instances)
        for (case in mapping["cases"]!!.jsonArray) {
            val obj = case.jsonObject
            val id = obj["instanceId"]!!.jsonPrimitive.content
            val found = obj["found"]!!.jsonPrimitive.content.toBooleanStrict()
            assertEquals(found, ContentToolHostLogic.resolveInstance(map, id) != null)
            assertEquals(found, ContentToolHostLogic.shouldMountFence(map[id]))
        }
    }

    @Test
    fun renderGateMatchesFixture() {
        for (case in fixture()["renderGate"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = case.jsonObject
            val mode = ContentToolHostLogic.fenceRenderMode(
                mobileContentToolsEnabled = obj["mobileContentToolsEnabled"]!!.jsonPrimitive.content.toBooleanStrict(),
                contentToolsEnabled = obj["contentToolsEnabled"]!!.jsonPrimitive.content.toBooleanStrict(),
            )
            val expected = when (obj["expected"]!!.jsonPrimitive.content) {
                "legacy_placeholder" -> ContentToolHostLogic.FenceRenderMode.LEGACY_PLACEHOLDER
                "hidden" -> ContentToolHostLogic.FenceRenderMode.HIDDEN
                "host" -> ContentToolHostLogic.FenceRenderMode.HOST
                else -> error("bad expected")
            }
            assertEquals(obj["name"]!!.jsonPrimitive.content, expected, mode)
        }
    }

    @Test
    fun highlightAnnotateUsesMergePolicy() {
        assertEquals(
            ContentToolHostLogic.ConflictPolicy.MERGE,
            ContentToolHostLogic.conflictPolicyForTool("highlight_annotate"),
        )
        assertEquals(
            ContentToolHostLogic.ConflictPolicy.SERVER_WINS,
            ContentToolHostLogic.conflictPolicyForTool("noop_probe"),
        )
    }

    @Test
    fun actionsAreNotQueuedOffline() {
        assertFalse(ContentToolHostLogic.canQueueActionOffline())
        assertTrue(ContentToolHostLogic.canQueueStateWriteOffline())
    }

    @Test
    fun outboxOrdersPerInstance() {
        val items = listOf(
            "b" to 2L,
            "a" to 10L,
            "b" to 1L,
            "a" to 1L,
        )
        assertEquals(
            listOf("a" to 1L, "a" to 10L, "b" to 1L, "b" to 2L),
            ContentToolHostLogic.orderOutboxByInstance(items),
        )
    }

    @Test
    fun unsupportedPlaceholderForUnknownToolOrContract() {
        assertTrue(ContentToolHostLogic.shouldShowUnsupportedPlaceholder("inline_questions", 1))
        assertTrue(ContentToolHostLogic.shouldShowUnsupportedPlaceholder("noop_probe", 2))
        assertFalse(ContentToolHostLogic.shouldShowUnsupportedPlaceholder("noop_probe", 1))
    }

    @Test
    fun mergeStatePatchKeepsPriorKeys() {
        val base = buildJsonObject { put("a", JsonPrimitive("1")) }
        val merged = ContentToolHostLogic.mergeStatePatch(
            base,
            mapOf("b" to JsonPrimitive("2")),
        )
        assertEquals("1", merged["a"]!!.jsonPrimitive.content)
        assertEquals("2", merged["b"]!!.jsonPrimitive.content)
    }

    @Test
    fun shouldFetchRequiresFlagsAndContext() {
        assertTrue(
            ContentToolHostLogic.shouldFetchInstances(
                mobileContentToolsEnabled = true,
                contentToolsEnabled = true,
                courseCode = "CS101",
                itemId = "item-1",
            ),
        )
        assertFalse(
            ContentToolHostLogic.shouldFetchInstances(
                mobileContentToolsEnabled = true,
                contentToolsEnabled = true,
                courseCode = "CS101",
                itemId = null,
            ),
        )
        assertFalse(
            ContentToolHostLogic.shouldFetchInstances(
                mobileContentToolsEnabled = false,
                contentToolsEnabled = true,
                courseCode = "CS101",
                itemId = "item-1",
            ),
        )
    }

    @Test
    fun webActivityPathAnchorsInstance() {
        assertEquals(
            "/courses/CS101/modules/items/item-1#lex-tool-abc",
            ContentToolHostLogic.webActivityPath("CS101", "item-1", "abc"),
        )
    }
}
