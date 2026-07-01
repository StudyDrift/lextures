package com.lextures.android.core.lms

import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

/** Validation and display helpers for profile depth (M1.5). */
object ProfileDepthLogic {
    const val PREFER_NOT_TO_SAY_RACE_CODE = "unknown"

    val raceEthnicityOptions = listOf(
        "1" to "mobile.profileDepth.race.hispanic",
        "2" to "mobile.profileDepth.race.americanIndian",
        "3" to "mobile.profileDepth.race.asian",
        "4" to "mobile.profileDepth.race.black",
        "5" to "mobile.profileDepth.race.pacificIslander",
        "6" to "mobile.profileDepth.race.white",
        "7" to "mobile.profileDepth.race.twoOrMore",
        PREFER_NOT_TO_SAY_RACE_CODE to "mobile.profileDepth.preferNotToSay",
    )

    fun validateCustomFields(
        definitions: List<ProfileFieldDefinition>,
        draft: Map<String, String>,
        requiredMessage: String,
        invalidNumberMessage: String,
        invalidDateMessage: String,
        invalidSelectMessage: String,
        invalidBooleanMessage: String,
    ): Map<String, String> {
        val errors = linkedMapOf<String, String>()
        for (def in definitions) {
            val raw = draft[def.key]?.trim().orEmpty()
            if (def.isRequired && raw.isEmpty()) {
                errors[def.key] = requiredMessage
                continue
            }
            if (raw.isEmpty()) continue
            when (def.fieldType) {
                "number" -> if (raw.toDoubleOrNull() == null) errors[def.key] = invalidNumberMessage
                "date" -> if (!isValidIsoDate(raw)) errors[def.key] = invalidDateMessage
                "select" -> {
                    val options = def.selectOptions.orEmpty()
                    if (options.isNotEmpty() && raw !in options) errors[def.key] = invalidSelectMessage
                }
                "boolean" -> if (raw.lowercase() !in setOf("true", "false")) {
                    errors[def.key] = invalidBooleanMessage
                }
            }
        }
        return errors
    }

    fun encodeCustomFieldValues(
        definitions: List<ProfileFieldDefinition>,
        draft: Map<String, String>,
    ): Map<String, JsonElement> = buildJsonObject {
        for (def in definitions) {
            val raw = draft[def.key]?.trim().orEmpty()
            when {
                raw.isEmpty() -> put(def.key, JsonNull)
                def.fieldType == "number" -> {
                    val n = raw.toDoubleOrNull()
                    if (n != null) put(def.key, JsonPrimitive(n)) else put(def.key, JsonNull)
                }
                def.fieldType == "boolean" -> put(def.key, JsonPrimitive(raw.lowercase() == "true"))
                else -> put(def.key, JsonPrimitive(raw))
            }
        }
    }.toMap()

    fun draftFromValues(
        definitions: List<ProfileFieldDefinition>,
        values: Map<String, JsonElement>,
    ): Map<String, String> = buildMap {
        for (def in definitions) {
            values[def.key].displayText(def.fieldType)?.let { put(def.key, it) }
        }
    }

    fun parseTriStateBool(raw: String): Boolean? = when (raw.lowercase()) {
        "true", "yes" -> true
        "false", "no" -> false
        else -> null
    }

    fun latestConsentByStudy(history: List<ConsentHistoryEntry>): List<ConsentHistoryEntry> {
        val seen = mutableSetOf<String>()
        return buildList {
            for (entry in history) {
                if (seen.add(entry.studyId)) add(entry)
            }
        }
    }

    fun shouldShowPersonalDetails(
        demographicsEnabled: Boolean,
        fieldCount: Int,
    ): Boolean = fieldCount > 0 || demographicsEnabled

    fun shouldShowResearchStudies(
        researchConsentEnabled: Boolean,
        pendingCount: Int,
        historyCount: Int,
    ): Boolean = researchConsentEnabled && (pendingCount > 0 || historyCount > 0)

    private fun isValidIsoDate(raw: String): Boolean {
        if (raw.length != 10) return false
        val parts = raw.split("-")
        if (parts.size != 3) return false
        val y = parts[0].toIntOrNull() ?: return false
        val m = parts[1].toIntOrNull() ?: return false
        val d = parts[2].toIntOrNull() ?: return false
        return y in 1900..2100 && m in 1..12 && d in 1..31
    }
}