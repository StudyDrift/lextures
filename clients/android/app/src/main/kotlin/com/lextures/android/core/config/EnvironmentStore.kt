package com.lextures.android.core.config

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

/**
 * Persists the tenant / environment chosen on the get-started screen.
 * The resolved API base URL drives all network traffic via [AppConfiguration].
 */
class EnvironmentStore private constructor(context: Context) {
    private val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    var apiBaseUrl by mutableStateOf(prefs.getString(KEY_API_BASE_URL, null))
        private set

    var kind by mutableStateOf(prefs.getString(KEY_KIND, null)?.let { Kind.fromStorage(it) })
        private set

    var schoolCode by mutableStateOf(prefs.getString(KEY_SCHOOL_CODE, null))
        private set

    val hasSelection: Boolean
        get() = !apiBaseUrl.isNullOrBlank()

    fun selectSelfLearner() {
        persist(Kind.SelfLearner, schoolCode = null, apiBaseUrl = SchoolCodeLogic.SELF_LEARNER_API_BASE)
    }

    fun selectSchool(code: String) {
        val normalized = SchoolCodeLogic.normalize(code)
        persist(
            Kind.School,
            schoolCode = normalized,
            apiBaseUrl = SchoolCodeLogic.apiBaseUrl(normalized),
        )
    }

    /** Clears the selection so the get-started flow is shown again. */
    fun clearSelection() {
        apiBaseUrl = null
        kind = null
        schoolCode = null
        prefs.edit()
            .remove(KEY_API_BASE_URL)
            .remove(KEY_KIND)
            .remove(KEY_SCHOOL_CODE)
            .apply()
    }

    private fun persist(kind: Kind, schoolCode: String?, apiBaseUrl: String) {
        val trimmed = apiBaseUrl.trimEnd('/')
        this.kind = kind
        this.schoolCode = schoolCode
        this.apiBaseUrl = trimmed
        prefs.edit()
            .putString(KEY_API_BASE_URL, trimmed)
            .putString(KEY_KIND, kind.storageValue)
            .apply {
                if (schoolCode != null) putString(KEY_SCHOOL_CODE, schoolCode)
                else remove(KEY_SCHOOL_CODE)
            }
            .apply()
    }

    enum class Kind(val storageValue: String) {
        SelfLearner("selfLearner"),
        School("school"),
        ;

        companion object {
            fun fromStorage(value: String): Kind? =
                entries.firstOrNull { it.storageValue == value }
        }
    }

    companion object {
        private const val PREFS_NAME = "lextures_environment"
        private const val KEY_API_BASE_URL = "apiBaseURL"
        private const val KEY_KIND = "kind"
        private const val KEY_SCHOOL_CODE = "schoolCode"

        @Volatile
        private var instance: EnvironmentStore? = null

        fun get(context: Context): EnvironmentStore =
            instance ?: synchronized(this) {
                instance ?: EnvironmentStore(context.applicationContext).also { instance = it }
            }
    }
}
