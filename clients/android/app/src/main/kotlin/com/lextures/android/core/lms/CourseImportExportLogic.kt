package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiError
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonObject

/** Course JSON backup/restore helpers (M13.10). */
object CourseImportExportLogic {
    enum class ImportMode { erase, mergeAdd, overwrite }

    /** Avoid loading very large backups fully in memory on device. */
    const val MAX_IMPORT_BYTES: Int = 50 * 1024 * 1024

    private val json = Json { ignoreUnknownKeys = true }

    fun webImportExportPath(courseCode: String): String =
        "/courses/$courseCode/settings/import-export"

    fun exportFileName(courseCode: String): String = "$courseCode-course-export.json"

    fun parseImportFileText(text: String): JsonObject {
        if (text.toByteArray(Charsets.UTF_8).size > MAX_IMPORT_BYTES) {
            throw ImportExportError.FileTooLarge
        }
        val element = runCatching { json.parseToJsonElement(text) }.getOrElse {
            throw ImportExportError.InvalidJson
        }
        val obj = runCatching { element.jsonObject }.getOrElse {
            throw ImportExportError.InvalidObject
        }
        if (obj.isEmpty()) throw ImportExportError.InvalidObject
        return obj
    }

    fun userFacingError(error: Throwable): String = when (error) {
        is ImportExportError -> error.message ?: "import-export-error"
        is ApiError.HttpStatus -> error.message ?: "import-export-error"
        else -> error.message ?: "import-export-error"
    }

    sealed class ImportExportError(message: String) : Exception(message) {
        data object InvalidJson : ImportExportError("invalid-json")
        data object InvalidObject : ImportExportError("invalid-object")
        data object FileTooLarge : ImportExportError("file-too-large")
    }
}
