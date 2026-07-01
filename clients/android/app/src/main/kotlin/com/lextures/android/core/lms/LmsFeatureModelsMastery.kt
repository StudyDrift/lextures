package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class MasteryConcept(
    val id: String,
    val name: String,
)

@Serializable
data class MasteryCell(
    val conceptId: String,
    val masteryScore: Double? = null,
    val assessed: Boolean = false,
    val updatedAt: String? = null,
)

@Serializable
data class StudentMasteryRow(
    val enrollmentId: String,
    val userId: String,
    val concepts: List<MasteryConcept> = emptyList(),
    val cells: List<MasteryCell> = emptyList(),
)

@Serializable
data class ReportCardSummary(
    val id: String,
    val studentId: String,
    val courseId: String,
    val gradingPeriod: String,
    val status: String,
    val finalGradePct: Double? = null,
    val letterGrade: String? = null,
    val comment: String? = null,
    val pdfUrl: String? = null,
    val generatedAt: String? = null,
    val releasedAt: String? = null,
    val createdAt: String? = null,
)

@Serializable
data class MyReportCardsResponse(
    val reportCards: List<ReportCardSummary> = emptyList(),
)
