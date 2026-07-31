package com.lextures.android.core.lms

import com.lextures.android.features.contenttools.ContentToolDraftStore
import java.io.File
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.int
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ContentToolPack2LogicTest {
    private val json = Json { ignoreUnknownKeys = true }

    private fun fixture(): JsonObject {
        val roots = listOf(
            File("clients/mobile/fixtures/content-tools/pack2-logic.json"),
            File("../mobile/fixtures/content-tools/pack2-logic.json"),
            File("../../../../../../mobile/fixtures/content-tools/pack2-logic.json"),
        )
        var dir = File(System.getProperty("user.dir") ?: ".")
        val candidates = roots.toMutableList()
        repeat(8) {
            candidates += File(dir, "clients/mobile/fixtures/content-tools/pack2-logic.json")
            candidates += File(dir, "mobile/fixtures/content-tools/pack2-logic.json")
            dir = dir.parentFile ?: return@repeat
        }
        val file = candidates.first { it.exists() }
        return json.parseToJsonElement(file.readText()).jsonObject
    }

    @Test
    fun allowlistMatchesFixture() {
        val cases = fixture()["allowlist"]!!.jsonObject["cases"]!!.jsonArray
        for (item in cases) {
            val obj = item.jsonObject
            val allowlist = obj["allowlist"]!!.jsonArray.map { it.jsonPrimitive.content }.toSet()
            val toolId = obj["toolId"]!!.jsonPrimitive.content
            val expected = obj["expected"]!!.jsonPrimitive.boolean
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                expected,
                ContentToolPack2Logic.isClientAllowlisted(toolId, allowlist),
            )
        }
    }

    @Test
    fun offlineQueueNeverQueues() {
        for (item in fixture()["offlineQueue"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                "${obj["toolId"]!!.jsonPrimitive.content}/${obj["action"]!!.jsonPrimitive.content}",
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack2Logic.canQueueActionOffline(
                    obj["toolId"]!!.jsonPrimitive.content,
                    obj["action"]!!.jsonPrimitive.content,
                ),
            )
        }
    }

    @Test
    fun draftLifecycleMatchesFixture() {
        val root = fixture()["draftLifecycle"]!!.jsonObject
        for (item in root["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val event = ContentToolPack2Logic.draftEventAfterAction(
                success = obj["success"]!!.jsonPrimitive.boolean,
                preserveInput = obj["preserveInput"]!!.jsonPrimitive.boolean,
            )
            val expected = obj["expected"]!!.jsonPrimitive.content
            val actual = when (event) {
                ContentToolPack2Logic.DraftEvent.CLEAR_ON_SUCCESS -> "clearOnSuccess"
                ContentToolPack2Logic.DraftEvent.RETAIN_ON_FAILURE -> "retainOnFailure"
                ContentToolPack2Logic.DraftEvent.SAVE -> "save"
                ContentToolPack2Logic.DraftEvent.RESTORE -> "restore"
            }
            assertEquals(obj["name"]?.jsonPrimitive?.content, expected, actual)
        }
        for (item in root["keys"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["expected"]!!.jsonPrimitive.content,
                ContentToolPack2Logic.draftStorageKey(
                    obj["instanceId"]!!.jsonPrimitive.content,
                    obj["slot"]!!.jsonPrimitive.content,
                ),
            )
        }
    }

    @Test
    fun consentGatingMatchesFixture() {
        for (item in fixture()["consentGating"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val mode = obj["disclosureMode"]?.jsonPrimitive?.contentOrNull
            val decision = obj["decision"].let {
                if (it == null || it is JsonNull) null else it.jsonPrimitive.contentOrNull
            }
            val fetched = obj["consentFetched"]!!.jsonPrimitive.boolean
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["composerAllowed"]!!.jsonPrimitive.boolean,
                ContentToolPack2Logic.composerAIAllowed(mode, decision, fetched),
            )
            assertEquals(
                obj["name"]?.jsonPrimitive?.content,
                obj["showDisclosure"]!!.jsonPrimitive.boolean,
                ContentToolPack2Logic.shouldShowAIDisclosure(mode, decision, fetched),
            )
        }
    }

    @Test
    fun errorClassificationMatchesFixture() {
        for (item in fixture()["errorClassification"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val code = obj["code"]?.jsonPrimitive?.contentOrNull
            assertEquals(obj["expected"]!!.jsonPrimitive.content, ContentToolPack2Logic.classifyAIError(code).code)
            assertEquals(
                obj["messageKey"]!!.jsonPrimitive.content,
                ContentToolPack2Logic.plainLanguageMessageKey(code),
            )
        }
    }

    @Test
    fun lengthGuidanceAndPagination() {
        for (item in fixture()["lengthGuidance"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack2Logic.lengthGuidanceOK(
                    obj["text"]!!.jsonPrimitive.content,
                    obj["minWords"]!!.jsonPrimitive.int,
                    obj["maxWords"]!!.jsonPrimitive.int,
                ),
            )
        }
        for (item in fixture()["pagination"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val totalEl = obj["total"]
            val total = if (totalEl == null || totalEl is JsonNull) null else totalEl.jsonPrimitive.intOrNull
            val next = ContentToolPack2Logic.nextPage(
                obj["currentPage"]!!.jsonPrimitive.int,
                obj["pageSize"]!!.jsonPrimitive.int,
                total,
            )
            val expectedEl = obj["expectedNext"]
            val expected = if (expectedEl == null || expectedEl is JsonNull) null else expectedEl.jsonPrimitive.intOrNull
            assertEquals(expected, next)
        }
    }

    @Test
    fun discussionControlsAndTombstone() {
        for (item in fixture()["discussionControls"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val controls = ContentToolPack2Logic.discussionControls(
                isOwn = obj["isOwn"]!!.jsonPrimitive.boolean,
                canEditFlag = obj["canEditFlag"]!!.jsonPrimitive.boolean,
                canDeleteFlag = obj["canDeleteFlag"]!!.jsonPrimitive.boolean,
                allowReplies = obj["allowReplies"]!!.jsonPrimitive.boolean,
                viewerCanEndorse = obj["viewerCanEndorse"]!!.jsonPrimitive.boolean,
                viewerCanModerate = obj["viewerCanModerate"]!!.jsonPrimitive.boolean,
                readOnly = obj["readOnly"]!!.jsonPrimitive.boolean,
                removed = obj["removed"]!!.jsonPrimitive.boolean,
            )
            val expected = obj["expected"]!!.jsonObject
            assertEquals(obj["name"]?.jsonPrimitive?.content, expected["canEdit"]!!.jsonPrimitive.boolean, controls.canEdit)
            assertEquals(expected["canDelete"]!!.jsonPrimitive.boolean, controls.canDelete)
            assertEquals(expected["canEndorse"]!!.jsonPrimitive.boolean, controls.canEndorse)
            assertEquals(expected["canModerate"]!!.jsonPrimitive.boolean, controls.canModerate)
            assertEquals(expected["canUpvote"]!!.jsonPrimitive.boolean, controls.canUpvote)
            assertEquals(expected["canReport"]!!.jsonPrimitive.boolean, controls.canReport)
            assertEquals(expected["canReply"]!!.jsonPrimitive.boolean, controls.canReply)
        }
        for (item in fixture()["tombstone"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val mod = obj["moderationState"].let {
                if (it == null || it is JsonNull) null else it.jsonPrimitive.contentOrNull
            }
            assertEquals(
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack2Logic.shouldRenderTombstone(
                    obj["removed"]!!.jsonPrimitive.boolean,
                    obj["tombstone"]!!.jsonPrimitive.boolean,
                    mod,
                ),
            )
        }
    }

    private fun policyWire(policy: ContentToolHostLogic.ConflictPolicy): String = when (policy) {
        ContentToolHostLogic.ConflictPolicy.MERGE -> "merge"
        ContentToolHostLogic.ConflictPolicy.SERVER_WINS -> "server_wins"
        ContentToolHostLogic.ConflictPolicy.CLIENT_WINS -> "client_wins"
    }

    @Test
    fun conflictPolicyComposerAndRegistry() {
        for (item in fixture()["conflictPolicy"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            val toolId = obj["toolId"]!!.jsonPrimitive.content
            val expected = obj["expected"]!!.jsonPrimitive.content
            assertEquals(expected, policyWire(ContentToolPack2Logic.conflictPolicy(toolId)))
            assertEquals(expected, policyWire(ContentToolHostLogic.conflictPolicyForTool(toolId)))
        }
        for (item in fixture()["composerSend"]!!.jsonObject["cases"]!!.jsonArray) {
            val obj = item.jsonObject
            assertEquals(
                obj["expected"]!!.jsonPrimitive.boolean,
                ContentToolPack2Logic.composerSendEnabled(
                    text = obj["text"]!!.jsonPrimitive.content,
                    readOnly = obj["readOnly"]!!.jsonPrimitive.boolean,
                    online = obj["online"]!!.jsonPrimitive.boolean,
                    busy = obj["busy"]!!.jsonPrimitive.boolean,
                    consentAllowed = obj["consentAllowed"]!!.jsonPrimitive.boolean,
                ),
            )
        }
        val ids = ContentToolHostLogic.registeredNativeToolIds()
        assertTrue(ids.contains("noop_probe"))
        for (toolId in ContentToolPack2Logic.pack2ToolIds) {
            assertTrue(toolId, ids.contains(toolId))
            assertFalse(ContentToolHostLogic.shouldShowUnsupportedPlaceholder(toolId, 1))
        }
    }

    @Test
    fun draftStoreRoundTrip() {
        val store = ContentToolDraftStore.inMemory()
        val key = ContentToolPack2Logic.draftStorageKey("test-inst", "composer")
        store.clear(key)
        assertEquals("", store.load(key))
        store.save(key, "half typed")
        assertEquals("half typed", store.load(key))
        store.clear(key)
        assertEquals("", store.load(key))
    }
}
