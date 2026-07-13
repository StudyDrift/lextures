package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PeopleAdminLogicTest {
    @Test
    fun adminSettingsEnabledRequiresFlag() {
        val off = MobilePlatformFeatures()
        assertFalse(PeopleAdminLogic.adminSettingsEnabled(off))
        val on = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertTrue(PeopleAdminLogic.adminSettingsEnabled(on))
    }

    @Test
    fun canManagePeopleRequiresRbacManage() {
        assertFalse(PeopleAdminLogic.canManagePeople(emptyList()))
        assertTrue(
            PeopleAdminLogic.canManagePeople(listOf(PeopleAdminLogic.RBAC_MANAGE_PERMISSION)),
        )
    }

    @Test
    fun shouldShowEntryRequiresFlagAndPermission() {
        val features = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertFalse(PeopleAdminLogic.shouldShowEntry(features, emptyList()))
        assertTrue(
            PeopleAdminLogic.shouldShowEntry(
                features,
                listOf(PeopleAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
    }

    @Test
    fun personDisplayNamePrefersDisplayName() {
        val row = PersonRow(
            id = "1",
            email = "a@example.com",
            displayName = "Alex Admin",
            orgId = "org",
            orgName = "Org",
            role = "teacher",
            active = true,
            createdAt = "2026-01-01T00:00:00Z",
        )
        assertEquals("Alex Admin", PeopleAdminLogic.personDisplayName(row))
    }

    @Test
    fun blocksSelfSuspend() {
        assertTrue(PeopleAdminLogic.blocksSelfSuspend("me", "me"))
        assertFalse(PeopleAdminLogic.blocksSelfSuspend("other", "me"))
    }

    @Test
    fun isErasedDetectsErasedEmail() {
        assertTrue(PeopleAdminLogic.isErased("user@erased.invalid"))
        assertFalse(PeopleAdminLogic.isErased("user@example.com"))
    }

    @Test
    fun invitePersonRequestTrimsFields() {
        val request = PeopleAdminLogic.invitePersonRequest(
            email = "  teacher@school.edu ",
            firstName = " Pat ",
            lastName = " ",
        )
        assertEquals("teacher@school.edu", request.email)
        assertEquals("Pat", request.firstName)
        assertNull(request.lastName)
    }

    @Test
    fun patchPersonRequestPayload() {
        assertFalse(PeopleAdminLogic.patchPersonRequest(false).active)
        assertTrue(PeopleAdminLogic.patchPersonRequest(true).active)
    }
}
