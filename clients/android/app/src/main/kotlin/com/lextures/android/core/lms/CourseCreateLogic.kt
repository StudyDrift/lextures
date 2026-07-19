package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.json.JSONObject
import java.util.Base64
import java.util.UUID

/** Course create wizard helpers (M11.5 / MOB.1) — permission gate, templates, validation, step state. */
object CourseCreateLogic {
    const val COURSE_CREATE_PERMISSION = "global:app:course:create"
    const val BLANK_TEMPLATE_ID = "blank"
    const val DEFAULT_FIRST_MODULE_TITLE = "Getting started"
    const val DEFAULT_TEMPLATE_ID = "higher-ed-15-week"

    enum class CourseMode(val value: String) {
        Traditional("traditional"),
        CompetencyBased("competency_based"),
        ;

        companion object {
            fun fromCourseType(courseType: String?): CourseMode =
                if (courseType == CompetencyBased.value) CompetencyBased else Traditional
        }
    }

    enum class CreateSource(val value: String) {
        Scratch("scratch"),
        Canvas("canvas"),
    }

    enum class AssessmentKind(val value: String) {
        Quiz("quiz"),
        Assignment("assignment"),
    }

    data class SubOutcomeDraft(
        val id: String = UUID.randomUUID().toString().lowercase(),
        val title: String = "",
        val description: String = "",
        val assessmentTitle: String = "",
        val assessmentKind: AssessmentKind = AssessmentKind.Quiz,
    ) {
        companion object {
            fun empty(): SubOutcomeDraft = SubOutcomeDraft()
        }
    }

    data class CompetencyDraft(
        val id: String = UUID.randomUUID().toString().lowercase(),
        val title: String = "",
        val description: String = "",
        val subOutcomes: List<SubOutcomeDraft> = listOf(SubOutcomeDraft.empty()),
        val expanded: Boolean = true,
    ) {
        companion object {
            fun empty(): CompetencyDraft = CompetencyDraft()
        }
    }

    data class CompetencyValidationError(
        val key: String,
        val args: List<String> = emptyList(),
    )

    enum class WizardStep(val number: Int) {
        Source(0),
        Basics(1),
        Syllabus(2),
        Finish(3),
        ;

        companion object {
            fun fromNumber(n: Int): WizardStep = entries.firstOrNull { it.number == n } ?: Basics
            val progressSteps: List<WizardStep> = listOf(Basics, Syllabus, Finish)
        }
    }

    data class TemplateSection(
        val heading: String,
        val markdown: String,
    )

    data class StarterTemplate(
        val id: String,
        val nameKey: String,
        val summaryKey: String,
        val suggestedFirstModuleTitle: String,
        val sections: List<TemplateSection>,
    )

    /** Port of web `COURSE_CREATE_STARTER_TEMPLATES` (ids + content parity). */
    val starterTemplates: List<StarterTemplate> = listOf(
        StarterTemplate(
            id = "k12-semester",
            nameKey = "mobile_createCourse_template_k12_name",
            summaryKey = "mobile_createCourse_template_k12_summary",
            suggestedFirstModuleTitle = "Unit 1: Getting started",
            sections = listOf(
                TemplateSection(
                    "Course overview",
                    "Briefly describe what students will learn this term and how day-to-day class time is structured.\n\n- **Big ideas**:\n- **Major projects or exams**:\n",
                ),
                TemplateSection(
                    "Materials & technology",
                    "List required texts, supplies, and any accounts or apps (including this LMS).\n\n| Item | Notes |\n|------|-------|\n| | |\n",
                ),
                TemplateSection(
                    "Grading",
                    "Explain how the gradebook categories work and how families can check progress.\n\n- **Formative vs summative**:\n- **Late work**:\n- **Retakes or revisions**:\n",
                ),
                TemplateSection(
                    "Classroom expectations",
                    "Norms for participation, discussion, academic honesty, and communication.\n\n1. **Respect** — listen and assume good intent.\n2. **Readiness** — arrive with materials.\n3. **Integrity** — cite sources; complete your own work.\n",
                ),
                TemplateSection(
                    "Contact & support",
                    "Best way to reach you, typical response time, and how to request extra help or accommodations.\n\n- **Email**:\n- **Office hours**:\n- **School resources**:\n",
                ),
            ),
        ),
        StarterTemplate(
            id = "higher-ed-15-week",
            nameKey = "mobile_createCourse_template_higherEd_name",
            summaryKey = "mobile_createCourse_template_higherEd_summary",
            suggestedFirstModuleTitle = "Week 1: Introduction & syllabus",
            sections = listOf(
                TemplateSection(
                    "Instructor & meeting times",
                    "**Instructor:**\n\n**Email:**\n\n**Office hours:**\n\n**Lecture / lab / discussion:**\n\n**Course site:** This LMS\n",
                ),
                TemplateSection(
                    "Course description",
                    "Paste the catalog description, then add a short paragraph on themes and prerequisites.\n\n**Prerequisites:**\n\n**Credit hours:**\n",
                ),
                TemplateSection(
                    "Learning outcomes",
                    "By the end of the term, students will be able to:\n\n1. \n2. \n3. \n",
                ),
                TemplateSection(
                    "Schedule at a glance",
                    "Outline major units or themes by week. Adjust dates to match your term.\n\n| Week | Topics | Due |\n|------|--------|-----|\n| 1 | | |\n| 2 | | |\n| … | | |\n",
                ),
                TemplateSection(
                    "Assessment & grading",
                    "Summarize weights (they should match your gradebook assignment groups).\n\n- **Exams:**\n- **Assignments:**\n- **Participation:**\n\n**Curve / grading scale:**\n",
                ),
                TemplateSection(
                    "Policies",
                    "Attendance, late work, academic integrity, accessibility, and technology expectations. Link or summarize institutional policies as required.\n",
                ),
            ),
        ),
        StarterTemplate(
            id = "self-paced",
            nameKey = "mobile_createCourse_template_selfPaced_name",
            summaryKey = "mobile_createCourse_template_selfPaced_summary",
            suggestedFirstModuleTitle = "Start here",
            sections = listOf(
                TemplateSection(
                    "How this course works",
                    "Explain that learners move at their own speed and where modules, due dates (if any), and checkpoints live.\n\n- **Estimated time:**\n- **Recommended pace:**\n- **Hard deadlines (if any):**\n",
                ),
                TemplateSection(
                    "Learning goals",
                    "What someone should be able to do after finishing all modules.\n\n1. \n2. \n3. \n",
                ),
                TemplateSection(
                    "Getting help",
                    "Where to ask questions (feed, inbox, discussion), expected response times, and links to FAQs or community norms.\n",
                ),
                TemplateSection(
                    "Completion criteria",
                    "Define what “done” means: required items, minimum scores, portfolio review, or certificate triggers.\n",
                ),
            ),
        ),
        StarterTemplate(
            id = "bootcamp",
            nameKey = "mobile_createCourse_template_bootcamp_name",
            summaryKey = "mobile_createCourse_template_bootcamp_summary",
            suggestedFirstModuleTitle = "Day 1: Orientation",
            sections = listOf(
                TemplateSection(
                    "Program overview",
                    "Length of program, daily schedule blocks, and how cohorts or teams are organized.\n\n- **Start / end dates:**\n- **Live vs async:**\n- **Capstone or demo day:**\n",
                ),
                TemplateSection(
                    "Projects & deliverables",
                    "List major builds, presentations, or assessments with rough timing.\n\n| Milestone | Description | Target |\n|-----------|-------------|--------|\n| | | |\n",
                ),
                TemplateSection(
                    "Tools & environment",
                    "Required installs, repos, API keys (never commit secrets), and how to verify your setup.\n\n```bash\n# Example: clone and install\n```\n",
                ),
                TemplateSection(
                    "Code of conduct & academic integrity",
                    "Collaboration rules, AI use policy if applicable, harassment-free space, and escalation paths.\n",
                ),
            ),
        ),
        StarterTemplate(
            id = "onboarding",
            nameKey = "mobile_createCourse_template_onboarding_name",
            summaryKey = "mobile_createCourse_template_onboarding_summary",
            suggestedFirstModuleTitle = "Week one checklist",
            sections = listOf(
                TemplateSection(
                    "Welcome",
                    "Warm intro to the program, who to ping first, and what success looks like in the first 30 days.\n",
                ),
                TemplateSection(
                    "Role & expectations",
                    "Responsibilities, stakeholders, working agreements, and how performance is reviewed.\n",
                ),
                TemplateSection(
                    "Systems & access",
                    "Accounts, security basics, where docs live, and how to request access.\n\n- [ ] Email / SSO\n- [ ] Chat\n- [ ] Ticketing\n",
                ),
                TemplateSection(
                    "First week checklist",
                    "Concrete tasks with owners and links.\n\n1. Complete profile in this LMS\n2. Read handbook section …\n3. Schedule intro 1:1s\n",
                ),
            ),
        ),
    )

    val gradeLevels: List<String> = CourseSettingsLogic.gradeLevels

    fun createCourseEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileCreateCourse || features.ffMobileCourseCreateV2

    fun courseCreateV2Enabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileCourseCreateV2

    fun canvasImportEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileCanvasImport

    fun shouldShowCanvasImportSource(
        permissions: List<String>,
        features: MobilePlatformFeatures,
        isOnline: Boolean,
    ): Boolean = CanvasImportLogic.shouldShowCanvasImportEntry(
        permissions = permissions,
        features = features,
        isOnline = isOnline,
    )

    fun canCreateCourses(permissions: List<String>): Boolean =
        permissions.contains(COURSE_CREATE_PERMISSION)

    fun shouldShowNewCourseAction(
        permissions: List<String>,
        features: MobilePlatformFeatures,
        isOnline: Boolean,
    ): Boolean {
        if (!createCourseEnabled(features)) return false
        if (!canCreateCourses(permissions)) return false
        return isOnline
    }

    fun initialWizardStep(v2Enabled: Boolean): WizardStep =
        if (v2Enabled) WizardStep.Source else WizardStep.Basics

    fun validateTitle(title: String): String? =
        if (title.trim().isEmpty()) "mobile.createCourse.error.titleRequired" else null

    /** Parity with web `validateCompetencies` (localized message keys + format args). */
    fun validateCompetencies(competencies: List<CompetencyDraft>): CompetencyValidationError? {
        if (competencies.isEmpty()) {
            return CompetencyValidationError("mobile.createCourse.error.competency.minOne")
        }
        competencies.forEachIndexed { i, c ->
            val title = c.title.trim()
            if (title.isEmpty()) {
                return CompetencyValidationError(
                    "mobile.createCourse.error.competency.titleRequired",
                    listOf("${i + 1}"),
                )
            }
            if (c.subOutcomes.isEmpty()) {
                return CompetencyValidationError(
                    "mobile.createCourse.error.competency.subOutcomeMinOne",
                    listOf(title),
                )
            }
            c.subOutcomes.forEachIndexed { j, s ->
                val subTitle = s.title.trim()
                if (subTitle.isEmpty()) {
                    return CompetencyValidationError(
                        "mobile.createCourse.error.competency.subOutcomeTitleRequired",
                        listOf(title, "${j + 1}"),
                    )
                }
                if (s.assessmentTitle.trim().isEmpty()) {
                    return CompetencyValidationError(
                        "mobile.createCourse.error.competency.assessmentTitleRequired",
                        listOf(subTitle),
                    )
                }
            }
        }
        return null
    }

    fun template(id: String): StarterTemplate? = starterTemplates.firstOrNull { it.id == id }

    fun suggestedFirstModuleTitle(templateId: String, existing: String): String {
        val trimmed = existing.trim()
        if (trimmed.isNotEmpty()) return trimmed
        if (templateId == BLANK_TEMPLATE_ID) return DEFAULT_FIRST_MODULE_TITLE
        return template(templateId)?.suggestedFirstModuleTitle ?: DEFAULT_FIRST_MODULE_TITLE
    }

    fun templateSectionsToSyllabus(sections: List<TemplateSection>): List<SyllabusSection> =
        sections.map {
            SyllabusSection(
                id = UUID.randomUUID().toString().lowercase(),
                heading = it.heading,
                markdown = it.markdown,
            )
        }

    fun shouldPatchSyllabus(templateId: String): Boolean =
        templateId != BLANK_TEMPLATE_ID && template(templateId) != null

    fun shouldUpdateExistingCourse(createdCourseCode: String?): Boolean =
        !createdCourseCode.isNullOrBlank()

    fun shouldConfirmCancel(createdCourseCode: String?): Boolean =
        shouldUpdateExistingCourse(createdCourseCode)

    fun buildCreateRequest(
        title: String,
        description: String,
        mode: CourseMode,
        termId: String?,
        gradeLevel: String?,
    ): CreateCourseRequest {
        val term = termId?.trim()?.takeIf { it.isNotEmpty() }
        val grade = gradeLevel?.trim()?.takeIf { it.isNotEmpty() }
        return CreateCourseRequest(
            title = title.trim(),
            description = description.trim(),
            courseType = mode.value,
            termId = term,
            gradeLevel = grade,
        )
    }

    fun buildUpdateRequest(
        course: CourseSummary,
        title: String,
        description: String,
        termId: String?,
        gradeLevel: String?,
    ): CourseUpdateRequest {
        val mode = if (course.scheduleMode == "relative") "relative" else "fixed"
        val term = termId?.trim()?.takeIf { it.isNotEmpty() }
        val grade = gradeLevel?.trim()?.takeIf { it.isNotEmpty() }
        return CourseUpdateRequest(
            title = title.trim(),
            description = description.trim(),
            published = course.published ?: false,
            startsAt = course.startsAt,
            endsAt = course.endsAt,
            visibleFrom = course.visibleFrom,
            hiddenAt = course.hiddenAt,
            scheduleMode = mode,
            relativeEndAfter = course.relativeEndAfter,
            relativeHiddenAfter = course.relativeHiddenAfter,
            courseHomeLanding = course.courseHomeLanding ?: "data",
            courseHomeContentItemId = course.courseHomeContentItemId,
            courseTimezone = course.courseTimezone,
            gradeLevel = grade,
            termId = term,
        )
    }

    fun orgIdFromAccessToken(jwt: String): String? {
        val segments = jwt.split(".")
        if (segments.size < 2) return null
        return try {
            var base64 = segments[1].replace('-', '+').replace('_', '/')
            while (base64.length % 4 != 0) base64 += "="
            val json = String(Base64.getDecoder().decode(base64))
            val obj = JSONObject(json)
            obj.optString("org_id").takeIf { it.isNotBlank() }
                ?: obj.optString("orgId").takeIf { it.isNotBlank() }
        } catch (_: Exception) {
            null
        }
    }

    fun resolveOrgId(accessToken: String?, courses: List<CourseSummary>): String? {
        accessToken?.let { orgIdFromAccessToken(it) }?.let { return it }
        return courses.mapNotNull { it.orgId }.firstOrNull { it.isNotBlank() }
    }
}
