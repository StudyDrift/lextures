package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class SettingsAccommodationsTest {
    private val json = kotlinx.serialization.json.Json { ignoreUnknownKeys = true }

    @Test
    fun decodesAccountProfile() {
        val profile = json.decodeFromString<AccountProfile>(
            """
            {"email":"ada@example.com","displayName":"Ada Lovelace","firstName":"Ada",
            "lastName":"Lovelace","avatarUrl":"https://img/a.png","phoneNumber":"+1 555 0100",
            "uiTheme":"dark","locale":"en"}
            """.trimIndent(),
        )
        assertEquals("ada@example.com", profile.email)
        assertEquals("Ada", profile.firstName)
        assertEquals("Lovelace", profile.lastName)
        assertEquals("https://img/a.png", profile.avatarUrl)
        assertEquals("+1 555 0100", profile.phoneNumber)
    }

    @Test
    fun accountProfileToleratesMissingFields() {
        val profile = json.decodeFromString<AccountProfile>("""{"email":"only@example.com"}""")
        assertEquals("only@example.com", profile.email)
        assertNull(profile.firstName)
        assertNull(profile.phoneNumber)
    }

    @Test
    fun resolvesNameFieldsFromDisplayName() {
        val profile = AccountProfile(email = "ada@example.com", displayName = "Ada Lovelace")
        val (first, last) = nameFieldsFromProfile(profile)
        assertEquals("Ada", first)
        assertEquals("Lovelace", last)
        assertEquals("Ada Lovelace", profile.resolvedDisplayName())
        assertEquals("AL", profile.resolvedInitials())
    }

    @Test
    fun resolvedDisplayNameFallsBackToEmail() {
        val profile = AccountProfile(email = "solo@example.com")
        assertEquals("solo@example.com", profile.resolvedDisplayName())
        assertEquals("S", profile.resolvedInitials())
    }

    @Test
    fun patchEncodesEditableFields() {
        val body = json.encodeToString(
            AccountProfilePatch.serializer(),
            AccountProfilePatch(firstName = "Grace", lastName = "Hopper", avatarUrl = "", phoneNumber = "555"),
        )
        assertTrue(body.contains("\"firstName\":\"Grace\""))
        assertTrue(body.contains("\"lastName\":\"Hopper\""))
        assertTrue(body.contains("\"avatarUrl\":\"\""))
        assertTrue(body.contains("\"phoneNumber\":\"555\""))
    }

    @Test
    fun decodesMyAccommodations() {
        val response = json.decodeFromString<MyAccommodationsResponse>(
            """
            {"accommodations":[
              {"courseCode":"MATH101","hasExtendedTime":true,"ttsEnabled":true,
               "hintsAlwaysAvailable":true,"effectiveFrom":"2026-01-01","effectiveUntil":"2026-12-31"},
              {"hasExtendedTime":false}
            ]}
            """.trimIndent(),
        )
        assertEquals(2, response.accommodations.size)

        val first = response.accommodations[0]
        assertEquals("MATH101", first.courseCode)
        assertTrue(first.hasExtendedTime)
        assertTrue(first.ttsEnabled)
        assertFalse(first.isEmpty)
        assertEquals("2026-01-01", first.effectiveFrom)

        val second = response.accommodations[1]
        assertNull(second.courseCode)
        assertTrue(second.isEmpty)
    }
}
