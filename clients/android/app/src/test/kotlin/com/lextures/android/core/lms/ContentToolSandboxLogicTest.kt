package com.lextures.android.core.lms

import java.io.File
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
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

class ContentToolSandboxLogicTest {
    private val json = Json { ignoreUnknownKeys = true }

    private fun fixture(): JsonObject =
        json.parseToJsonElement(resolveFixture().readText()).jsonObject

    private fun resolveFixture(): File {
        var dir: File? = File(System.getProperty("user.dir") ?: ".")
        repeat(8) {
            val current = dir ?: return@repeat
            val candidates = listOf(
                File(current, "clients/mobile/fixtures/content-tools/bridge/messages.json"),
                File(current, "../mobile/fixtures/content-tools/bridge/messages.json"),
                File(current, "../../mobile/fixtures/content-tools/bridge/messages.json"),
                File(current, "mobile/fixtures/content-tools/bridge/messages.json"),
            )
            candidates.firstOrNull { it.isFile }?.let { return it.canonicalFile }
            dir = current.parentFile
        }
        error("bridge/messages.json not found from ${System.getProperty("user.dir")}")
    }

    @Test
    fun constantsMatchFixture() {
        val c = fixture()["constants"]!!.jsonObject
        assertEquals(c["bridgeVersion"]!!.jsonPrimitive.intOrNull, ContentToolSandboxLogic.BRIDGE_VERSION)
        assertEquals(c["maxMessageBytes"]!!.jsonPrimitive.intOrNull, ContentToolSandboxLogic.BRIDGE_MAX_MESSAGE_BYTES)
        assertEquals(c["maxMessagesPerSec"]!!.jsonPrimitive.intOrNull, ContentToolSandboxLogic.BRIDGE_MAX_MESSAGES_PER_SEC)
        assertEquals(c["readyTimeoutMs"]!!.jsonPrimitive.intOrNull, ContentToolSandboxLogic.READY_TIMEOUT_MS)
        assertEquals(c["minHeight"]!!.jsonPrimitive.doubleOrNull, ContentToolSandboxLogic.MIN_HEIGHT_DP)
        assertEquals(c["maxHeight"]!!.jsonPrimitive.doubleOrNull, ContentToolSandboxLogic.MAX_HEIGHT_DP)
        assertEquals(c["maxLiveWebViews"]!!.jsonPrimitive.intOrNull, ContentToolSandboxLogic.MAX_LIVE_WEBVIEWS)
    }

    @Test
    fun validationMatchesFixture() {
        for (case in fixture()["validation"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = case.jsonObject
            val name = obj["name"]!!.jsonPrimitive.content
            val direction = obj["direction"]!!.jsonPrimitive.content
            val accept = obj["accept"]!!.jsonPrimitive.booleanOrNull == true
            val msgEl = obj["msg"]
            val actual = when (direction) {
                "fromTool" -> ContentToolSandboxLogic.isBridgeFromTool(msgEl)
                "toTool" -> ContentToolSandboxLogic.isBridgeToTool(msgEl)
                else -> error("bad direction")
            }
            assertEquals(name, accept, actual)
        }
    }

    @Test
    fun rateLimitMatchesFixture() {
        val rate = fixture()["rateLimit"]!!.jsonObject
        val max = rate["maxPerSec"]!!.jsonPrimitive.intOrNull!!
        for (case in rate["cases"]!!.jsonArray) {
            val obj = case.jsonObject
            val limiter = ContentToolSandboxLogic.BridgeRateLimiter(max)
            val stamps = obj["timestampsMs"]!!.jsonArray.map { it.jsonPrimitive.longOrNullCompatible() }
            val expected = obj["expectedAllow"]!!.jsonArray.map { it.jsonPrimitive.booleanOrNull == true }
            val actual = stamps.map { limiter.allow(it) }
            assertEquals(obj["name"]!!.jsonPrimitive.content, expected, actual)
        }
    }

    @Test
    fun sizeGuardRejectsOversize() {
        for (case in fixture()["sizeGuard"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = case.jsonObject
            val approx = obj["approxBytes"]!!.jsonPrimitive.intOrNull!!
            val reject = obj["reject"]!!.jsonPrimitive.booleanOrNull == true
            val payload = "x".repeat(approx)
            val raw = """{"t":"announce","v":1,"message":"$payload"}"""
            val limiter = ContentToolSandboxLogic.BridgeRateLimiter(1000)
            val reason = ContentToolSandboxLogic.rejectIngress(raw, limiter, nowMs = 1)
            if (reject) {
                assertEquals(obj["name"]!!.jsonPrimitive.content, ContentToolSandboxLogic.RejectionReason.OVERSIZED, reason)
            } else {
                assertNull(obj["name"]!!.jsonPrimitive.content, reason)
            }
        }
    }

    @Test
    fun heightClampMatchesFixture() {
        for (case in fixture()["heightClamp"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = case.jsonObject
            val input = obj["input"]!!.jsonPrimitive.doubleOrNull!!
            val expected = obj["expected"]!!.jsonPrimitive.doubleOrNull!!
            assertEquals(expected, ContentToolSandboxLogic.clampHeight(input), 0.0)
        }
    }

    @Test
    fun opaqueParticipantIdMatchesFixture() {
        for (case in fixture()["opaqueParticipantId"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = case.jsonObject
            val instanceId = obj["instanceId"]!!.jsonPrimitive.content
            val hintEl = obj["enrollmentHint"]
            val hint = if (hintEl == null || hintEl is JsonNull) null else hintEl.jsonPrimitive.content
            val expected = obj["expected"]!!.jsonPrimitive.content
            assertEquals(expected, ContentToolSandboxLogic.opaqueParticipantId(instanceId, hint))
        }
    }

    @Test
    fun resolutionMatchesFixture() {
        for (case in fixture()["resolution"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = case.jsonObject
            val registered = obj["registered"]!!.jsonArray.map { it.jsonPrimitive.content }.toSet()
            val sandboxModeEl = obj["sandboxMode"]
            val sandboxMode = if (sandboxModeEl == null || sandboxModeEl is JsonNull) {
                null
            } else {
                sandboxModeEl.jsonPrimitive.content
            }
            val path = ContentToolSandboxLogic.resolveRenderPath(
                toolId = obj["toolId"]!!.jsonPrimitive.content,
                contract = obj["contract"]!!.jsonPrimitive.intOrNull!!,
                sandboxMode = sandboxMode,
                sandboxEnabled = obj["sandboxEnabled"]!!.jsonPrimitive.booleanOrNull == true,
                registered = registered,
                tombstone = obj["tombstone"]!!.jsonPrimitive.booleanOrNull == true,
                breakerOpen = obj["breakerOpen"]!!.jsonPrimitive.booleanOrNull == true,
                deprecated = obj["deprecated"]!!.jsonPrimitive.booleanOrNull == true,
                killed = obj["killed"]!!.jsonPrimitive.booleanOrNull == true,
            )
            assertEquals(obj["name"]!!.jsonPrimitive.content, obj["expected"]!!.jsonPrimitive.content, path.wire())
        }
    }

    @Test
    fun contractRangeMatchesFixture() {
        val root = fixture()["contractRange"]!!.jsonObject
        for (case in root["cases"]!!.jsonArray) {
            val obj = case.jsonObject
            val ok = obj["ok"]!!.jsonPrimitive.booleanOrNull == true
            assertEquals(ok, ContentToolSandboxLogic.contractInSupportedRange(obj["contract"]!!.jsonPrimitive.content))
        }
    }

    @Test
    fun poolEvictsBeyondMax() {
        assertFalse(ContentToolSandboxLogic.poolShouldEvict(3, 3))
        assertTrue(ContentToolSandboxLogic.poolShouldEvict(4, 3))
    }

    private fun JsonPrimitive.longOrNullCompatible(): Long =
        contentOrNull?.toLongOrNull() ?: intOrNull?.toLong() ?: error("not a long")
}
