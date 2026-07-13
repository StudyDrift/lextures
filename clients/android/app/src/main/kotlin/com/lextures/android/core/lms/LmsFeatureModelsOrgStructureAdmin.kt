package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

// MARK: - Org structure admin (M14.4)

@Serializable
data class AdminOrgRow(
    val id: String,
    val slug: String,
    val name: String,
    val status: String,
    val dataRegion: String? = null,
    val userCount: Long? = null,
    val courseCount: Long? = null,
    val createdAt: String? = null,
)

@Serializable
data class AdminOrgsListResponse(
    val organizations: List<AdminOrgRow>? = null,
)

@Serializable
data class OrgUnitTreeNode(
    val id: String,
    val name: String,
    val unitType: String,
    val status: String,
    val childCourseCount: Long? = null,
    val children: List<OrgUnitTreeNode>? = null,
)

@Serializable
data class OrgUnitTreeResponse(
    val tree: List<OrgUnitTreeNode>? = null,
)

@Serializable
data class CreateAcademicTermRequest(
    val name: String,
    val termType: String,
    val startDate: String,
    val endDate: String,
)

@Serializable
data class PatchAcademicTermRequest(
    val name: String? = null,
    val termType: String? = null,
    val startDate: String? = null,
    val endDate: String? = null,
    val status: String? = null,
)

@Serializable
data class PatchOrgUnitRequest(
    val name: String? = null,
)
