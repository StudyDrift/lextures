package com.lextures.android.core.lms

import kotlinx.serialization.json.JsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ProfileDepthLogicTest {
    private val json = kotlinx.serialization.json.Json { ignoreUnknownKeys = true }
    @Test
    fun validateCustomFields_flagsRequiredBlank() {
        val defs = listOf(
            ProfileFieldDefinition(
                id = "1",
                key = "student_id",
                label = "Student ID",
                fieldType = "text",
                isRequired = true,
            ),
        )
        val errors = ProfileDepthLogic.validateCustomFields(
            definitions = defs,
            draft = emptyMap(),
            requiredMessage = "required",
            invalidNumberMessage = "number",
            invalidDateMessage = "date",
            invalidSelectMessage = "select",
            invalidBooleanMessage = "boolean",
        )
        assertEquals("required", errors["student_id"])
    }

    @Test
    fun validateCustomFields_rejectsInvalidSelect() {
        val defs = listOf(
            ProfileFieldDefinition(
                id = "1",
                key = "dept",
                label = "Department",
                fieldType = "select",
                selectOptions = listOf("Math", "Science"),
                isRequired = true,
            ),
        )
        val errors = ProfileDepthLogic.validateCustomFields(
            definitions = defs,
            draft = mapOf("dept" to "History"),
            requiredMessage = "required",
            invalidNumberMessage = "number",
            invalidDateMessage = "date",
            invalidSelectMessage = "select",
            invalidBooleanMessage = "boolean",
        )
        assertEquals("select", errors["dept"])
    }

    @Test
    fun encodeCustomFieldValues() {
        val defs = listOf(
            ProfileFieldDefinition(id = "1", key = "active", label = "Active", fieldType = "boolean"),
            ProfileFieldDefinition(id = "2", key = "note", label = "Note", fieldType = "text"),
        )
        val encoded = ProfileDepthLogic.encodeCustomFieldValues(
            definitions = defs,
            draft = mapOf("active" to "true", "note" to "hello"),
        )
        assertEquals(JsonPrimitive(true), encoded["active"])
        assertEquals(JsonPrimitive("hello"), encoded["note"])
    }

    @Test
    fun latestConsentByStudy_keepsFirstPerStudy() {
        val history = listOf(
            ConsentHistoryEntry("a", "s1", "A", ConsentDecision.Granted, "2026-01-02"),
            ConsentHistoryEntry("b", "s1", "A", ConsentDecision.Withdrawn, "2026-01-01"),
            ConsentHistoryEntry("c", "s2", "B", ConsentDecision.Declined, "2026-01-03"),
        )
        val latest = ProfileDepthLogic.latestConsentByStudy(history)
        assertEquals(2, latest.size)
        assertEquals(ConsentDecision.Granted, latest[0].decision)
        assertEquals("s2", latest[1].studyId)
    }

    @Test
    fun shouldShowPersonalDetails() {
        assertTrue(ProfileDepthLogic.shouldShowPersonalDetails(demographicsEnabled = false, fieldCount = 1))
        assertTrue(ProfileDepthLogic.shouldShowPersonalDetails(demographicsEnabled = true, fieldCount = 0))
        assertFalse(ProfileDepthLogic.shouldShowPersonalDetails(demographicsEnabled = false, fieldCount = 0))
    }

    @Test
    fun shouldShowResearchStudies() {
        assertTrue(ProfileDepthLogic.shouldShowResearchStudies(true, 1, 0))
        assertTrue(ProfileDepthLogic.shouldShowResearchStudies(true, 0, 1))
        assertFalse(ProfileDepthLogic.shouldShowResearchStudies(false, 1, 1))
        assertFalse(ProfileDepthLogic.shouldShowResearchStudies(true, 0, 0))
    }

    @Test
    fun decodesProfileFieldsResponse() {
        val response = json.decodeFromString<ProfileFieldsResponse>(
            """
            {"fields":[{"id":"f1","key":"student_id","label":"Student ID","fieldType":"text","isRequired":true}],
            "values":{"student_id":"123"}}
            """.trimIndent(),
        )
        assertEquals(1, response.fields.size)
        assertEquals("123", response.values["student_id"].displayText("text"))
    }

}