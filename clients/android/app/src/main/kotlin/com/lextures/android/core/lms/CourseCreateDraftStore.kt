package com.lextures.android.core.lms

import android.content.Context
import org.json.JSONArray
import org.json.JSONObject

/** Persists in-progress create-wizard UI state across backgrounding (MOB.1 FR-10). */
class CourseCreateDraftStore(context: Context) {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    data class Draft(
        val step: Int,
        val title: String,
        val description: String,
        val courseMode: String,
        val selectedTermId: String,
        val selectedGradeLevel: String,
        val selectedTemplateId: String,
        val firstModuleTitle: String,
        val createdCourseCode: String?,
        val competencies: List<CourseCreateLogic.CompetencyDraft>,
        val createSource: String? = null,
    )

    fun storageKey(userId: String?, orgId: String?): String {
        val user = userId?.trim()?.takeIf { it.isNotEmpty() } ?: "anon"
        val org = orgId?.trim()?.takeIf { it.isNotEmpty() } ?: "org"
        return "$PREFIX$user.$org"
    }

    fun load(key: String): Draft? {
        val raw = prefs.getString(key, null) ?: return null
        return runCatching { decode(JSONObject(raw)) }.getOrNull()
    }

    fun save(key: String, draft: Draft) {
        prefs.edit().putString(key, encode(draft).toString()).apply()
    }

    fun clear(key: String) {
        prefs.edit().remove(key).apply()
    }

    private fun encode(draft: Draft): JSONObject = JSONObject().apply {
        put("step", draft.step)
        put("title", draft.title)
        put("description", draft.description)
        put("courseMode", draft.courseMode)
        put("selectedTermId", draft.selectedTermId)
        put("selectedGradeLevel", draft.selectedGradeLevel)
        put("selectedTemplateId", draft.selectedTemplateId)
        put("firstModuleTitle", draft.firstModuleTitle)
        put("createdCourseCode", draft.createdCourseCode)
        put("createSource", draft.createSource)
        put(
            "competencies",
            JSONArray().apply {
                draft.competencies.forEach { c ->
                    put(
                        JSONObject().apply {
                            put("id", c.id)
                            put("title", c.title)
                            put("description", c.description)
                            put("expanded", c.expanded)
                            put(
                                "subOutcomes",
                                JSONArray().apply {
                                    c.subOutcomes.forEach { s ->
                                        put(
                                            JSONObject().apply {
                                                put("id", s.id)
                                                put("title", s.title)
                                                put("description", s.description)
                                                put("assessmentTitle", s.assessmentTitle)
                                                put("assessmentKind", s.assessmentKind.value)
                                            },
                                        )
                                    }
                                },
                            )
                        },
                    )
                }
            },
        )
    }

    private fun decode(obj: JSONObject): Draft {
        val comps = mutableListOf<CourseCreateLogic.CompetencyDraft>()
        val arr = obj.optJSONArray("competencies")
        if (arr != null) {
            for (i in 0 until arr.length()) {
                val c = arr.getJSONObject(i)
                val subs = mutableListOf<CourseCreateLogic.SubOutcomeDraft>()
                val subArr = c.optJSONArray("subOutcomes")
                if (subArr != null) {
                    for (j in 0 until subArr.length()) {
                        val s = subArr.getJSONObject(j)
                        val kind = when (s.optString("assessmentKind")) {
                            CourseCreateLogic.AssessmentKind.Assignment.value ->
                                CourseCreateLogic.AssessmentKind.Assignment
                            else -> CourseCreateLogic.AssessmentKind.Quiz
                        }
                        subs.add(
                            CourseCreateLogic.SubOutcomeDraft(
                                id = s.optString("id").ifBlank { java.util.UUID.randomUUID().toString().lowercase() },
                                title = s.optString("title"),
                                description = s.optString("description"),
                                assessmentTitle = s.optString("assessmentTitle"),
                                assessmentKind = kind,
                            ),
                        )
                    }
                }
                comps.add(
                    CourseCreateLogic.CompetencyDraft(
                        id = c.optString("id").ifBlank { java.util.UUID.randomUUID().toString().lowercase() },
                        title = c.optString("title"),
                        description = c.optString("description"),
                        subOutcomes = if (subs.isEmpty()) listOf(CourseCreateLogic.SubOutcomeDraft.empty()) else subs,
                        expanded = c.optBoolean("expanded", true),
                    ),
                )
            }
        }
        return Draft(
            step = obj.optInt("step", 1),
            title = obj.optString("title"),
            description = obj.optString("description"),
            courseMode = obj.optString("courseMode", CourseCreateLogic.CourseMode.Traditional.value),
            selectedTermId = obj.optString("selectedTermId"),
            selectedGradeLevel = obj.optString("selectedGradeLevel"),
            selectedTemplateId = obj.optString("selectedTemplateId", CourseCreateLogic.DEFAULT_TEMPLATE_ID),
            firstModuleTitle = obj.optString("firstModuleTitle"),
            createdCourseCode = obj.optString("createdCourseCode").takeIf { it.isNotBlank() },
            competencies = if (comps.isEmpty()) listOf(CourseCreateLogic.CompetencyDraft.empty()) else comps,
            createSource = obj.optString("createSource").takeIf { it.isNotBlank() },
        )
    }

    companion object {
        private const val PREFS = "course_create_drafts"
        private const val PREFIX = "course_create_draft."
    }
}
