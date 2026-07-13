package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.util.Date

class OrgStructureAdminLogicTest {
    @Test
    fun adminSettingsEnabled() {
        val off = MobilePlatformFeatures()
        assertFalse(OrgStructureAdminLogic.adminSettingsEnabled(off))
        val on = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertTrue(OrgStructureAdminLogic.adminSettingsEnabled(on))
    }

    @Test
    fun canManageOrganizations() {
        assertFalse(OrgStructureAdminLogic.canManageOrganizations(emptyList()))
        assertTrue(
            OrgStructureAdminLogic.canManageOrganizations(
                listOf(OrgStructureAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
    }

    @Test
    fun canManageOrgUnitsAndTerms() {
        assertFalse(OrgStructureAdminLogic.canManageOrgUnitsAndTerms(emptyList()))
        assertTrue(
            OrgStructureAdminLogic.canManageOrgUnitsAndTerms(
                listOf(OrgStructureAdminLogic.ORG_UNITS_ADMIN_PERMISSION),
            ),
        )
    }

    @Test
    fun shouldShowEntry() {
        val features = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertFalse(OrgStructureAdminLogic.shouldShowEntry(features, emptyList()))
        assertTrue(
            OrgStructureAdminLogic.shouldShowEntry(
                features,
                listOf(OrgStructureAdminLogic.ORG_UNITS_ADMIN_PERMISSION),
            ),
        )
    }

    @Test
    fun createTermRequest() {
        val request = OrgStructureAdminLogic.createTermRequest(
            name = "  Spring 2026 ",
            termType = "",
            startDate = "2026-01-10",
            endDate = "2026-05-15",
        )
        assertEquals("Spring 2026", request.name)
        assertEquals(OrgStructureAdminLogic.DEFAULT_TERM_TYPE, request.termType)
    }

    @Test
    fun dateRangeValidation() {
        assertTrue(OrgStructureAdminLogic.isValidDateRange("2026-01-01", "2026-06-01"))
        assertFalse(OrgStructureAdminLogic.isValidDateRange("2026-06-01", "2026-01-01"))
    }

    @Test
    fun isoDateRoundTrip() {
        val date = Date(1_704_067_200_000L) // 2024-01-01 UTC
        val iso = OrgStructureAdminLogic.isoDateString(date)
        assertEquals(iso, OrgStructureAdminLogic.isoDateString(OrgStructureAdminLogic.dateFromIso(iso)!!))
    }
}
