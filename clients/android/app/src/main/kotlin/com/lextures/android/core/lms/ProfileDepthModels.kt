package com.lextures.android.core.lms

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.doubleOrNull

@Serializable
data class ProfileFieldDefinition(
    val id: String,
    val key: String,
    val label: String,
    val fieldType: String,
    val selectOptions: List<String>? = null,
    val isRequired: Boolean = false,
)

@Serializable
data class ProfileFieldsResponse(
    val fields: List<ProfileFieldDefinition> = emptyList(),
    val values: Map<String, JsonElement> = emptyMap(),
)

@Serializable
data class ProfileFieldsPatch(
    val values: Map<String, JsonElement>,
)

@Serializable
data class ProfileFieldsValuesResponse(
    val values: Map<String, JsonElement> = emptyMap(),
)

@Serializable
data class StudentDemographics(
    val studentId: String? = null,
    val freeLunch: Boolean? = null,
    val reducedLunch: Boolean? = null,
    val ellStatus: Boolean? = null,
    val disabilityStatus: Boolean? = null,
    val raceEthnicityCode: String? = null,
    val homelessIndicator: Boolean? = null,
    val migrantIndicator: Boolean? = null,
    val dataSource: String? = null,
    val updatedAt: String? = null,
)

@Serializable
data class StudentDemographicsPatch(
    val freeLunch: Boolean? = null,
    val reducedLunch: Boolean? = null,
    val ellStatus: Boolean? = null,
    val disabilityStatus: Boolean? = null,
    val raceEthnicityCode: String? = null,
    val homelessIndicator: Boolean? = null,
    val migrantIndicator: Boolean? = null,
)

enum class ConsentDecision {
    @SerialName("granted") Granted,
    @SerialName("declined") Declined,
    @SerialName("withdrawn") Withdrawn,
}

@Serializable
data class ConsentStudy(
    val id: String,
    val title: String,
    val irbProtocol: String,
    val consentText: String,
    val dataUseDescription: String,
    val status: String,
)

@Serializable
data class ConsentStudiesResponse(
    val studies: List<ConsentStudy> = emptyList(),
)

@Serializable
data class ConsentHistoryEntry(
    val id: String,
    val studyId: String,
    val studyTitle: String? = null,
    val decision: ConsentDecision,
    val createdAt: String,
)

@Serializable
data class ConsentHistoryResponse(
    val history: List<ConsentHistoryEntry> = emptyList(),
)

@Serializable
data class ConsentRespondBody(
    val decision: ConsentDecision,
)

fun JsonElement?.displayText(fieldType: String): String? {
    val el = this ?: return null
    if (el is JsonNull) return null
    val prim = el as? JsonPrimitive ?: return null
    if (prim.isString) return prim.content
    prim.booleanOrNull?.let { return if (it) "true" else "false" }
    prim.doubleOrNull?.let { n ->
        return if (n % 1.0 == 0.0) n.toLong().toString() else n.toString()
    }
    return null
}