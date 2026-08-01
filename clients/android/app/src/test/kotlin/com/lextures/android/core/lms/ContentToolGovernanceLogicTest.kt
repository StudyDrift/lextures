package com.lextures.android.core.lms

import java.io.File
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.longOrNull
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class ContentToolGovernanceLogicTest {
    private val json = Json { ignoreUnknownKeys = true }

    @Before
    fun setUp() {
        ContentToolsObservability.resetForTests()
    }

    private fun fixture(): JsonObject {
        return json.parseToJsonElement(resolveFixture().readText()).jsonObject
    }

    private fun resolveFixture(): File {
        var dir: File? = File(System.getProperty("user.dir") ?: ".")
        repeat(8) {
            val current = dir ?: return@repeat
            val candidates = listOf(
                File(current, "clients/mobile/fixtures/content-tools/governance-logic.json"),
                File(current, "../mobile/fixtures/content-tools/governance-logic.json"),
                File(current, "../../mobile/fixtures/content-tools/governance-logic.json"),
                File(current, "mobile/fixtures/content-tools/governance-logic.json"),
            )
            candidates.firstOrNull { it.isFile }?.let { return it.canonicalFile }
            dir = current.parentFile
        }
        error(
            "clients/mobile/fixtures/content-tools/governance-logic.json not found from ${System.getProperty("user.dir")}"
        )
    }

    private fun stringList(obj: JsonObject, key: String): List<String> =
        obj[key]?.jsonArray?.mapNotNull { it.jsonPrimitive.contentOrNull } ?: emptyList()

    private fun mountInput(raw: JsonObject): ContentToolGovernanceLogic.MountInput =
        ContentToolGovernanceLogic.MountInput(
            toolId = raw["toolId"]?.jsonPrimitive?.contentOrNull.orEmpty(),
            capabilities = stringList(raw, "capabilities"),
            sandboxMode = raw["sandboxMode"]?.jsonPrimitive?.contentOrNull,
            tombstone = raw["tombstone"]?.jsonPrimitive?.booleanOrNull ?: false,
            breakerOpen = raw["breakerOpen"]?.jsonPrimitive?.booleanOrNull ?: false,
            deprecated = raw["deprecated"]?.jsonPrimitive?.booleanOrNull ?: false,
            killed = raw["killed"]?.jsonPrimitive?.booleanOrNull ?: false,
            allowedToolIds = stringList(raw, "allowedToolIds"),
            deniedToolIds = stringList(raw, "deniedToolIds"),
            deniedCapabilities = stringList(raw, "deniedCapabilities"),
            policyFetched = raw["policyFetched"]?.jsonPrimitive?.booleanOrNull ?: false,
            policyAgeMs = raw["policyAgeMs"]?.jsonPrimitive?.longOrNull ?: 0L,
            staleWindowMs = raw["staleWindowMs"]?.jsonPrimitive?.longOrNull
                ?: ContentToolGovernanceLogic.DEFAULT_STALE_WINDOW_MS,
            unknownGovernanceState = raw["unknownGovernanceState"]?.jsonPrimitive?.booleanOrNull ?: false,
            hasCachedPolicy = raw["hasCachedPolicy"]?.jsonPrimitive?.booleanOrNull ?: false,
        )

    @Test
    fun mountDecisionMatchesFixture() {
        val cases = fixture()["mountDecision"]!!.jsonObject["cases"]!!.jsonArray
        for (case in cases) {
            val obj = case.jsonObject
            val name = obj["name"]!!.jsonPrimitive.content
            val expected = obj["expected"]!!.jsonPrimitive.content
            val actual = ContentToolGovernanceLogic.mountDecision(mountInput(obj["input"]!!.jsonObject)).wire
            assertEquals(name, expected, actual)
        }
    }

    @Test
    fun consentGatingMatchesFixture() {
        val cases = fixture()["consentGating"]!!.jsonObject["cases"]!!.jsonArray
        for (case in cases) {
            val obj = case.jsonObject
            val name = obj["name"]!!.jsonPrimitive.content
            val mode = obj["disclosureMode"]?.jsonPrimitive?.contentOrNull
            val decision = obj["decision"]?.let {
                if (it is JsonNull) null else it.jsonPrimitive.contentOrNull
            }
            val fetched = obj["consentFetched"]!!.jsonPrimitive.booleanOrNull!!
            assertEquals(
                "$name aiAllowed",
                obj["aiAllowed"]!!.jsonPrimitive.booleanOrNull,
                ContentToolGovernanceLogic.aiActionsAllowed(mode, decision, fetched),
            )
            assertEquals(
                "$name showDisclosure",
                obj["showDisclosure"]!!.jsonPrimitive.booleanOrNull,
                ContentToolGovernanceLogic.shouldShowAIDisclosure(mode, decision, fetched),
            )
        }
    }

    @Test
    fun killDerivationMatchesFixture() {
        val cases = fixture()["killDerivation"]!!.jsonObject["cases"]!!.jsonArray
        for (case in cases) {
            val obj = case.jsonObject
            val name = obj["name"]!!.jsonPrimitive.content
            val actual = ContentToolGovernanceLogic.toolIsKilled(
                toolId = obj["toolId"]!!.jsonPrimitive.content,
                capabilities = stringList(obj, "capabilities"),
                killedToolIds = stringList(obj, "killedToolIds"),
                killedCapabilities = stringList(obj, "killedCapabilities"),
                killAllAI = obj["killAllAI"]!!.jsonPrimitive.booleanOrNull!!,
            )
            assertEquals(name, obj["expected"]!!.jsonPrimitive.booleanOrNull, actual)
        }
    }

    @Test
    fun reportReachableInTwoTaps() {
        val maxTaps = fixture()["reportReachability"]!!.jsonObject["maxTaps"]!!.jsonPrimitive.intOrNull!!
        assertTrue(ContentToolGovernanceLogic.reportTapCount() <= maxTaps)
        assertTrue(ContentToolGovernanceLogic.reportReachableInTwoTaps())
    }

    @Test
    fun telemetryPayloadShape() {
        val shape = fixture()["telemetryPayloadShape"]!!.jsonObject
        for (example in shape["validExamples"]!!.jsonArray) {
            val attrs = example.jsonObject.mapValues { it.value.jsonPrimitive.content }
            assertTrue(ContentToolGovernanceLogic.telemetryAttributesAreContentFree(attrs))
        }
        for (example in shape["invalidExamples"]!!.jsonArray) {
            val attrs = example.jsonObject.mapValues { it.value.jsonPrimitive.content }
            assertFalse(ContentToolGovernanceLogic.telemetryAttributesAreContentFree(attrs))
        }
    }

    @Test
    fun filterCrisisMatchesFixture() {
        val cases = fixture()["filterCrisis"]!!.jsonObject["cases"]!!.jsonArray
        for (case in cases) {
            val obj = case.jsonObject
            val name = obj["name"]!!.jsonPrimitive.content
            val outcome = ContentToolGovernanceLogic.filterCrisisOutcome(
                ContentToolGovernanceLogic.FilterCrisisInput(
                    errorCode = obj["errorCode"]?.jsonPrimitive?.contentOrNull,
                    crisis = obj["crisis"]!!.jsonPrimitive.booleanOrNull!!,
                ),
            )
            assertEquals(name, obj["expectedKind"]!!.jsonPrimitive.content, outcome.kind.name.lowercase())
            assertEquals(name, obj["preserveDraft"]!!.jsonPrimitive.booleanOrNull, outcome.preserveDraft)
            assertEquals(name, obj["retry"]!!.jsonPrimitive.booleanOrNull, outcome.retry)
        }
    }

    @Test
    fun messageKeysMatchFixture() {
        val keys = fixture()["messageKeys"]!!.jsonObject
        for ((decision, keyEl) in keys) {
            val expected = keyEl.jsonPrimitive.content
            val mount = ContentToolGovernanceLogic.MountDecision.fromWire(decision)
            assertEquals(decision, expected, ContentToolGovernanceLogic.reasonMessageKey(mount))
        }
    }

    @Test
    fun observabilityDropsLearnerContent() {
        ContentToolsObservability.record(
            "bad_event",
            toolId = "ask_questions",
            attributes = mapOf("prompt" to "what is photosynthesis?"),
        )
        assertEquals(0, ContentToolsObservability.count("bad_event"))
        ContentToolsObservability.record(
            "tool_mount",
            toolId = "flashcards",
            attributes = mapOf("outcome" to "ok"),
        )
        assertEquals(1, ContentToolsObservability.count("tool_mount"))
    }
}
