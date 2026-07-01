package com.lextures.android.core.search

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import com.lextures.android.core.navigation.ShellTab
import com.lextures.android.core.routing.DeepLinkDestination
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner

@RunWith(RobolectricTestRunner::class)
class SearchTests {
    private lateinit var context: Context

    @Before
    fun setUp() {
        context = ApplicationProvider.getApplicationContext()
        SearchRecentsStore.clearAll(context)
    }

    @Test
    fun shouldQueryRequiresTwoCharacters() {
        assertFalse(SearchQueryEngine.shouldQuery("a"))
        assertTrue(SearchQueryEngine.shouldQuery("ab"))
    }

    @Test
    fun actionMatcherFindsCalendar() {
        val actions = listOf(
            SearchListItem(
                id = "action:calendar",
                group = SearchResultGroup.Action,
                title = "Open Calendar",
                subtitle = "Your schedule",
                path = "/calendar",
                haystack = "calendar schedule open calendar goto jump action",
            ),
        )
        val matches = SearchActionRegistry.matchActions("calendar", actions)
        assertEquals(1, matches.size)
        assertEquals("Open Calendar", matches.first().title)
    }

    @Test
    fun recentsCapAtTen() {
        repeat(12) { index ->
            SearchRecentsStore.recordSearch(context, "query-$index")
        }
        assertEquals(SearchQueryEngine.MAX_RECENTS, SearchRecentsStore.recentSearches(context).size)
        assertEquals("query-11", SearchRecentsStore.recentSearches(context).first())
    }

    @Test
    fun pathNavigatorMapsCalendar() {
        val target = SearchPathNavigator.resolve("/calendar")
        assertEquals(SearchNavigationTarget.ShellTabTarget(ShellTab.Calendar), target)
    }

    @Test
    fun pathNavigatorMapsCourseContent() {
        val target = SearchPathNavigator.resolve("/courses/demo/assignments/item-1")
        assertTrue(target is SearchNavigationTarget.DeepLinkTarget)
        val destination = (target as SearchNavigationTarget.DeepLinkTarget).destination
        assertTrue(destination is DeepLinkDestination.Course)
        val course = destination as DeepLinkDestination.Course
        assertEquals("demo", course.code)
        assertEquals(com.lextures.android.core.routing.CourseDeepLinkSection.Modules, course.section)
        assertEquals("item-1", course.itemId)
    }
}