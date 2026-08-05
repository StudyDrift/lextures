package com.lextures.android.core.lms

import com.lextures.android.core.navigation.CourseWorkspaceContext
import com.lextures.android.core.navigation.CourseWorkspaceSection
import com.lextures.android.core.navigation.MobileDestinations
import com.lextures.android.core.navigation.MobileRoleContext
import com.lextures.android.core.offline.OfflineCacheKey
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.double
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

class CourseChecklistLogicTest {
    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    private fun fixtureText(): String {
        val candidates = mutableListOf(
            File("clients/mobile/fixtures/checklist/logic-parity.json"),
            File("../mobile/fixtures/checklist/logic-parity.json"),
            File("../../../../../../mobile/fixtures/checklist/logic-parity.json"),
        )
        var dir = File(System.getProperty("user.dir") ?: ".")
        repeat(10) {
            candidates += File(dir, "clients/mobile/fixtures/checklist/logic-parity.json")
            candidates += File(dir, "mobile/fixtures/checklist/logic-parity.json")
            dir = dir.parentFile ?: return@repeat
        }
        val file = candidates.firstOrNull { it.isFile }
            ?: error("logic-parity.json not found from ${System.getProperty("user.dir")}")
        return file.readText()
    }

    private fun fixtureRoot(): JsonObject = json.parseToJsonElement(fixtureText()).jsonObject

    private fun table(): Map<String, String> {
        // Prefer packages JSON (unit tests have no Android assets).
        val candidates = mutableListOf(
            File("clients/packages/checklist-targets.json"),
            File("../packages/checklist-targets.json"),
            File("app/src/main/assets/checklist-targets.json"),
            File("src/main/assets/checklist-targets.json"),
        )
        var dir = File(System.getProperty("user.dir") ?: ".")
        repeat(10) {
            candidates += File(dir, "clients/packages/checklist-targets.json")
            candidates += File(dir, "packages/checklist-targets.json")
            dir = dir.parentFile ?: return@repeat
        }
        val file = candidates.firstOrNull { it.isFile }
            ?: return emptyMap()
        return CourseChecklistLogic.loadTargetTable(file.readText())
    }

    @Test
    fun badgeMatchesFixture() {
        val cases = fixtureRoot()["badge"]!!.jsonArray
        for (el in cases) {
            val item = el.jsonObject
            val count = item["outstandingEssential"]!!.jsonPrimitive.int
            val badge = CourseChecklistLogic.badgePresentation(count)
            assertEquals(item["visible"]!!.jsonPrimitive.boolean, badge.visible)
            assertEquals(item["text"]!!.jsonPrimitive.content, badge.text)
            val contains = item["accessibilityContains"]?.jsonPrimitive?.content.orEmpty()
            if (contains.isNotEmpty()) {
                assertEquals(contains, badge.accessibilityLabel)
            }
        }
    }

    @Test
    fun statusMatchesFixture() {
        val cases = fixtureRoot()["status"]!!.jsonArray
        for (el in cases) {
            val item = el.jsonObject
            val raw = item["raw"]!!.jsonPrimitive.content
            val normalized = when (CourseChecklistLogic.normalizeStatus(raw)) {
                ChecklistStatus.Done -> "done"
                ChecklistStatus.Todo -> "todo"
                ChecklistStatus.InProgress -> "in_progress"
                ChecklistStatus.NotApplicable -> "not_applicable"
                ChecklistStatus.Unknown -> "unknown"
            }
            assertEquals(item["normalized"]!!.jsonPrimitive.content, normalized)
            assertEquals(item["outstanding"]!!.jsonPrimitive.boolean, CourseChecklistLogic.isOutstanding(raw))
            assertEquals(
                item["accessibilityValue"]!!.jsonPrimitive.content,
                CourseChecklistLogic.accessibilityStatusValue(raw),
            )
        }
    }

    @Test
    fun progressMatchesFixture() {
        val cases = fixtureRoot()["progress"]!!.jsonArray
        for (el in cases) {
            val item = el.jsonObject
            val done = item["done"]!!.jsonPrimitive.int
            val total = item["total"]!!.jsonPrimitive.int
            assertEquals(
                item["fraction"]!!.jsonPrimitive.double,
                CourseChecklistLogic.progressFraction(done, total),
                0.0001,
            )
            assertEquals(
                item["label"]!!.jsonPrimitive.content,
                CourseChecklistLogic.progressLabel(done, total),
            )
        }
    }

    @Test
    fun targetResolutionMatchesFixture() {
        val table = table()
        assertTrue("target table should load", table.isNotEmpty())
        val cases = fixtureRoot()["targets"]!!.jsonArray
        for (el in cases) {
            val item = el.jsonObject
            val name = item["name"]?.jsonPrimitive?.content ?: "?"
            val courseCode = item["courseCode"]!!.jsonPrimitive.content
            val targetEl = item["target"]
            val target = if (targetEl == null || targetEl is JsonNull) {
                null
            } else {
                val t = targetEl.jsonObject
                ChecklistNavTarget(
                    route = t["route"]!!.jsonPrimitive.content,
                    anchor = t["anchor"]?.jsonPrimitive?.contentOrNull,
                    entityKey = t["entityKey"]?.jsonPrimitive?.contentOrNull,
                )
            }
            val resolved = CourseChecklistLogic.resolveTarget(target, courseCode, table)
            val kind = when (resolved.kind) {
                CourseChecklistLogic.TargetKind.Native -> "native"
                CourseChecklistLogic.TargetKind.Web -> "web"
                CourseChecklistLogic.TargetKind.Unresolved -> "unresolved"
            }
            assertEquals("$name kind", item["expectedKind"]!!.jsonPrimitive.content, kind)
            item["expectedSection"]?.jsonPrimitive?.content?.let { expectedSection ->
                val actual = resolved.workspaceSection?.deepLinkSegment
                    ?: resolved.workspaceSection?.name?.lowercase()
                assertEquals("$name section", expectedSection, actual)
            }
            item["webPathContains"]?.jsonPrimitive?.content?.let { contains ->
                assertTrue(
                    "$name webPath ${resolved.webPath}",
                    resolved.webPath?.contains(contains) == true,
                )
            }
        }
    }

    @Test
    fun visibilityMatchesFixture() {
        val cases = fixtureRoot()["visibility"]!!.jsonArray
        for (el in cases) {
            val item = el.jsonObject
            val role = when (item["roleContext"]!!.jsonPrimitive.content) {
                "teaching" -> MobileRoleContext.Teaching
                "learning" -> MobileRoleContext.Learning
                else -> MobileRoleContext.Parent
            }
            assertEquals(
                item["show"]!!.jsonPrimitive.boolean,
                CourseChecklistLogic.shouldShowWorkspaceSection(
                    item["viewerIsStaff"]!!.jsonPrimitive.boolean,
                    role,
                ),
            )
        }
    }

    @Test
    fun presentationParity() {
        val root = fixtureRoot()["presentation"]!!.jsonObject
        val checklist = json.decodeFromString<CourseChecklist>(root["checklist"]!!.toString())
        val presented = CourseChecklistLogic.presentChecklist(checklist, table())
        val expected = root["expected"]!!.jsonArray
        assertEquals(expected.size, presented.size)
        for (i in expected.indices) {
            val item = expected[i].jsonObject
            val p = presented[i]
            assertEquals(item["id"]!!.jsonPrimitive.content, p.id)
            assertEquals(item["title"]!!.jsonPrimitive.content, p.title)
            assertEquals(item["status"]!!.jsonPrimitive.content, p.status)
            assertEquals(item["accessibilityValue"]!!.jsonPrimitive.content, p.accessibilityValue)
            assertEquals(item["isDone"]!!.jsonPrimitive.boolean, p.isDone)
            assertEquals(item["isOutstanding"]!!.jsonPrimitive.boolean, p.isOutstanding)
            assertEquals(item["targetKind"]!!.jsonPrimitive.content, p.targetKind)
        }
    }

    @Test
    fun storageNeverPersistsChecklist() {
        val prefix = fixtureRoot()["storage"]!!.jsonObject["forbiddenKeyPrefix"]!!.jsonPrimitive.content
        assertEquals(prefix, CourseChecklistLogic.OFFLINE_CACHE_KEY_PREFIX)
        assertFalse(OfflineCacheKey.courses().startsWith(prefix))
        CourseChecklistSummaryStore.clearAll()
        CourseChecklistSummaryStore.put(
            "C-test",
            CourseChecklistSummary(outstandingEssential = 2, outstandingTotal = 3, done = 1, total = 4),
        )
        assertEquals(2, CourseChecklistSummaryStore.outstandingEssential("C-test"))
        CourseChecklistSummaryStore.clearAll()
    }

    @Test
    fun workspaceInsertsChecklistAfterOverviewForStaffTeaching() {
        val course = CourseSummary(
            id = "1",
            courseCode = "demo",
            title = "Demo",
            description = "",
            viewerEnrollmentRoles = listOf("teacher"),
        )
        val sections = MobileDestinations.courseWorkspaceSections(
            CourseWorkspaceContext(course = course, roleContext = MobileRoleContext.Teaching),
        )
        assertEquals(CourseWorkspaceSection.Overview, sections.first())
        assertEquals(CourseWorkspaceSection.Checklist, sections[1])
        val learning = MobileDestinations.courseWorkspaceSections(
            CourseWorkspaceContext(course = course, roleContext = MobileRoleContext.Learning),
        )
        assertFalse(learning.contains(CourseWorkspaceSection.Checklist))
    }
}
