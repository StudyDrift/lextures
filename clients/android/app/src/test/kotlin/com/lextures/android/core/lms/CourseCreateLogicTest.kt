package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseCreateLogicTest {
    @Test
    fun permissionGate() {
        assertTrue(CourseCreateLogic.canCreateCourses(listOf(CourseCreateLogic.COURSE_CREATE_PERMISSION)))
        assertFalse(CourseCreateLogic.canCreateCourses(listOf("other")))
    }

    @Test
    fun shouldShowNewCourseActionRequiresFlagPermissionAndOnline() {
        val perms = listOf(CourseCreateLogic.COURSE_CREATE_PERMISSION)
        assertFalse(
            CourseCreateLogic.shouldShowNewCourseAction(
                permissions = perms,
                features = MobilePlatformFeatures(ffMobileCreateCourse = false, ffMobileCourseCreateV2 = false),
                isOnline = true,
            ),
        )
        assertTrue(
            CourseCreateLogic.shouldShowNewCourseAction(
                permissions = perms,
                features = MobilePlatformFeatures(ffMobileCreateCourse = true),
                isOnline = true,
            ),
        )
        assertFalse(
            CourseCreateLogic.shouldShowNewCourseAction(
                permissions = perms,
                features = MobilePlatformFeatures(ffMobileCreateCourse = true),
                isOnline = false,
            ),
        )
        assertFalse(
            CourseCreateLogic.shouldShowNewCourseAction(
                permissions = emptyList(),
                features = MobilePlatformFeatures(ffMobileCreateCourse = true),
                isOnline = true,
            ),
        )
        assertTrue(
            CourseCreateLogic.shouldShowNewCourseAction(
                permissions = perms,
                features = MobilePlatformFeatures(ffMobileCreateCourse = false, ffMobileCourseCreateV2 = true),
                isOnline = true,
            ),
        )
        assertTrue(
            CourseCreateLogic.courseCreateV2Enabled(
                MobilePlatformFeatures(ffMobileCourseCreateV2 = true),
            ),
        )
        assertEquals(CourseCreateLogic.WizardStep.Source, CourseCreateLogic.initialWizardStep(true))
        assertEquals(CourseCreateLogic.WizardStep.Basics, CourseCreateLogic.initialWizardStep(false))
    }

    @Test
    fun validateCompetenciesParity() {
        assertEquals(
            "mobile.createCourse.error.competency.minOne",
            CourseCreateLogic.validateCompetencies(emptyList())?.key,
        )
        assertEquals(
            "mobile.createCourse.error.competency.titleRequired",
            CourseCreateLogic.validateCompetencies(listOf(CourseCreateLogic.CompetencyDraft.empty()))?.key,
        )
        assertEquals(
            "mobile.createCourse.error.competency.subOutcomeMinOne",
            CourseCreateLogic.validateCompetencies(
                listOf(CourseCreateLogic.CompetencyDraft(title = "Reading", subOutcomes = emptyList())),
            )?.key,
        )
        assertEquals(
            "mobile.createCourse.error.competency.subOutcomeTitleRequired",
            CourseCreateLogic.validateCompetencies(
                listOf(
                    CourseCreateLogic.CompetencyDraft(
                        title = "Reading",
                        subOutcomes = listOf(
                            CourseCreateLogic.SubOutcomeDraft(title = "", assessmentTitle = "Quiz 1"),
                        ),
                    ),
                ),
            )?.key,
        )
        assertEquals(
            "mobile.createCourse.error.competency.assessmentTitleRequired",
            CourseCreateLogic.validateCompetencies(
                listOf(
                    CourseCreateLogic.CompetencyDraft(
                        title = "Reading",
                        subOutcomes = listOf(
                            CourseCreateLogic.SubOutcomeDraft(title = "Main idea", assessmentTitle = ""),
                        ),
                    ),
                ),
            )?.key,
        )
        assertNull(
            CourseCreateLogic.validateCompetencies(
                listOf(
                    CourseCreateLogic.CompetencyDraft(
                        title = "Reading",
                        subOutcomes = listOf(
                            CourseCreateLogic.SubOutcomeDraft(title = "Main idea", assessmentTitle = "Quiz 1"),
                        ),
                    ),
                ),
            ),
        )
    }

    @Test
    fun validateTitleRequired() {
        assertNotNull(CourseCreateLogic.validateTitle("  "))
        assertNull(CourseCreateLogic.validateTitle("Biology 101"))
    }

    @Test
    fun shouldUpdateExistingCourse() {
        assertFalse(CourseCreateLogic.shouldUpdateExistingCourse(null))
        assertFalse(CourseCreateLogic.shouldUpdateExistingCourse("  "))
        assertTrue(CourseCreateLogic.shouldUpdateExistingCourse("C-ABC123"))
        assertTrue(CourseCreateLogic.shouldConfirmCancel("C-ABC123"))
    }

    @Test
    fun suggestedFirstModuleTitleFromTemplate() {
        assertEquals(
            CourseCreateLogic.DEFAULT_FIRST_MODULE_TITLE,
            CourseCreateLogic.suggestedFirstModuleTitle("blank", ""),
        )
        assertEquals(
            "Unit 1: Getting started",
            CourseCreateLogic.suggestedFirstModuleTitle("k12-semester", ""),
        )
        assertEquals(
            "My module",
            CourseCreateLogic.suggestedFirstModuleTitle("k12-semester", "My module"),
        )
    }

    @Test
    fun templateParityWithWeb() {
        val ids = CourseCreateLogic.starterTemplates.map { it.id }
        assertEquals(
            listOf(
                "k12-semester",
                "higher-ed-15-week",
                "self-paced",
                "bootcamp",
                "onboarding",
            ),
            ids,
        )
        assertEquals(5, CourseCreateLogic.starterTemplates.size)
        CourseCreateLogic.starterTemplates.forEach {
            assertTrue(it.sections.isNotEmpty())
            assertTrue(it.suggestedFirstModuleTitle.isNotEmpty())
        }
    }

    @Test
    fun shouldPatchSyllabus() {
        assertFalse(CourseCreateLogic.shouldPatchSyllabus(CourseCreateLogic.BLANK_TEMPLATE_ID))
        assertTrue(CourseCreateLogic.shouldPatchSyllabus("higher-ed-15-week"))
        assertFalse(CourseCreateLogic.shouldPatchSyllabus("unknown"))
    }

    @Test
    fun templateSectionsToSyllabusAssignsIds() {
        val tmpl = CourseCreateLogic.template("self-paced")!!
        val sections = CourseCreateLogic.templateSectionsToSyllabus(tmpl.sections)
        assertEquals(tmpl.sections.size, sections.size)
        assertEquals(sections.size, sections.map { it.id }.toSet().size)
        assertEquals(tmpl.sections.first().heading, sections.first().heading)
    }

    @Test
    fun buildCreateRequest() {
        val body = CourseCreateLogic.buildCreateRequest(
            title = "  Chem ",
            description = " Intro ",
            mode = CourseCreateLogic.CourseMode.CompetencyBased,
            termId = "  ",
            gradeLevel = "9",
        )
        assertEquals("Chem", body.title)
        assertEquals("Intro", body.description)
        assertEquals("competency_based", body.courseType)
        assertNull(body.termId)
        assertEquals("9", body.gradeLevel)
    }

    @Test
    fun buildUpdateRequestDoesNotDuplicate() {
        val course = CourseSummary(
            id = "1",
            courseCode = "C-1",
            title = "Old",
            description = "D",
            published = false,
        )
        val body = CourseCreateLogic.buildUpdateRequest(
            course = course,
            title = "New title",
            description = "New desc",
            termId = "term-1",
            gradeLevel = "5",
        )
        assertEquals("New title", body.title)
        assertEquals("New desc", body.description)
        assertEquals("term-1", body.termId)
        assertEquals("5", body.gradeLevel)
        assertEquals("fixed", body.scheduleMode)
        assertTrue(CourseCreateLogic.shouldUpdateExistingCourse(course.courseCode))
    }

    @Test
    fun modeFromCourseType() {
        assertEquals(
            CourseCreateLogic.CourseMode.CompetencyBased,
            CourseCreateLogic.CourseMode.fromCourseType("competency_based"),
        )
        assertEquals(
            CourseCreateLogic.CourseMode.Traditional,
            CourseCreateLogic.CourseMode.fromCourseType("traditional"),
        )
        assertEquals(
            CourseCreateLogic.CourseMode.Traditional,
            CourseCreateLogic.CourseMode.fromCourseType(null),
        )
    }
}
