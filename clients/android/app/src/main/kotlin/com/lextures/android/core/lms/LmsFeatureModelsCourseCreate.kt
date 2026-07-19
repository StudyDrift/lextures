package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class CreateCourseRequest(
    val title: String,
    val description: String,
    val courseType: String? = null,
    val termId: String? = null,
    val gradeLevel: String? = null,
)

@Serializable
data class OrgTerm(
    val id: String,
    val orgId: String? = null,
    val name: String,
    val termType: String? = null,
    val startDate: String? = null,
    val endDate: String? = null,
    val status: String? = null,
)

@Serializable
data class OrgTermsResponse(
    val terms: List<OrgTerm>? = null,
)

@Serializable
data class PatchCourseSyllabusRequest(
    val sections: List<SyllabusSection>,
    val requireSyllabusAcceptance: Boolean,
)

@Serializable
data class CreateCourseModuleRequest(
    val title: String,
)

@Serializable
data class CreateModuleItemRequest(
    val title: String,
)
