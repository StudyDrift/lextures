package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PlatformCoursesAdminLogicTest {
    @Test
    fun toggleFilterSelectsAndClears() {
        assertEquals(CoursesListFilter.Draft, PlatformCoursesAdminLogic.toggleFilter(null, CoursesListFilter.Draft))
        assertNull(PlatformCoursesAdminLogic.toggleFilter(CoursesListFilter.Draft, CoursesListFilter.Draft))
    }

    @Test
    fun valueMapsStatsKeys() {
        val stats = CoursesDashboardStats(
            createdLast7Days = 1,
            activeCourses = 2,
            draftCourses = 3,
            totalCourses = 10,
            archivedCourses = 4,
        )
        assertEquals(1, PlatformCoursesAdminLogic.value(CoursesListFilter.Created7d, stats))
        assertEquals(2, PlatformCoursesAdminLogic.value(CoursesListFilter.Active, stats))
        assertEquals(3, PlatformCoursesAdminLogic.value(CoursesListFilter.Draft, stats))
        assertEquals(10, PlatformCoursesAdminLogic.value(CoursesListFilter.Total, stats))
        assertEquals(4, PlatformCoursesAdminLogic.value(CoursesListFilter.Archived, stats))
    }

    @Test
    fun apiFilterValuesMatchServer() {
        assertEquals("created_7d", CoursesListFilter.Created7d.apiValue)
        assertEquals("active", CoursesListFilter.Active.apiValue)
        assertEquals("draft", CoursesListFilter.Draft.apiValue)
        assertEquals("total", CoursesListFilter.Total.apiValue)
        assertEquals("archived", CoursesListFilter.Archived.apiValue)
    }
}

class PeopleAdminMetricsLogicTest {
    @Test
    fun toggleFilterSelectsAndClears() {
        assertEquals(PeopleListFilter.Active, PeopleAdminLogic.toggleFilter(null, PeopleListFilter.Active))
        assertNull(PeopleAdminLogic.toggleFilter(PeopleListFilter.Active, PeopleListFilter.Active))
    }

    @Test
    fun valueMapsStatsKeys() {
        val stats = PeopleDashboardStats(
            signupsLast7Days = 1,
            activeAccounts = 2,
            totalAccounts = 5,
            recentlyActive30Days = 3,
            suspendedAccounts = 4,
        )
        assertEquals(1, PeopleAdminLogic.value(PeopleListFilter.Signups7d, stats))
        assertEquals(2, PeopleAdminLogic.value(PeopleListFilter.Active, stats))
        assertEquals(3, PeopleAdminLogic.value(PeopleListFilter.Recent30d, stats))
        assertEquals(5, PeopleAdminLogic.value(PeopleListFilter.Total, stats))
        assertEquals(4, PeopleAdminLogic.value(PeopleListFilter.Suspended, stats))
    }
}
