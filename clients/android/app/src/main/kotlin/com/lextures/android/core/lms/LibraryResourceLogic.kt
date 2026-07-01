package com.lextures.android.core.lms

/** Access resolution for HE library / e-reserve resources (M3.6). */
sealed class LibraryAccessState {
    data class Ready(val url: String) : LibraryAccessState()
    data class Gated(val messageKey: String) : LibraryAccessState()
    data class RequiresWeb(val path: String) : LibraryAccessState()
}

enum class LibraryBrowseTab {
    Library,
    Oer,
}

object LibraryResourceLogic {
    fun libraryItems(items: List<CourseStructureItem>): List<CourseStructureItem> =
        items.filter { it.kind == "library_resource" }

    fun hasLibraryResources(items: List<CourseStructureItem>): Boolean =
        libraryItems(items).isNotEmpty()

    fun accessEventPath(courseCode: String, itemId: String): String =
        "/api/v1/courses/${encodePath(courseCode)}/library-resources/${encodePath(itemId)}/access"

    fun resourceDetailPath(courseCode: String, itemId: String): String =
        "/api/v1/courses/${encodePath(courseCode)}/library-resources/${encodePath(itemId)}"

    fun webModulePath(courseCode: String, itemId: String): String =
        "/courses/${encodePath(courseCode)}/modules/library-resource/${encodePath(itemId)}"

    fun resolveAccess(payload: LibraryResourcePayload): LibraryAccessState {
        normalizedUrl(payload.ezproxyUrl)?.let { return LibraryAccessState.Ready(it) }
        normalizedUrl(payload.metadata?.ezproxyUrl)?.let { return LibraryAccessState.Ready(it) }
        return when (payload.resourceType) {
            "leganto_list" -> LibraryAccessState.Gated("mobile.library.legantoGated")
            "catalog_item" -> LibraryAccessState.Gated("mobile.library.catalogGated")
            else -> LibraryAccessState.Gated("mobile.library.noAccess")
        }
    }

    fun resourceTypeLabel(resourceType: String): String = when (resourceType) {
        "leganto_list" -> "mobile.library.type.leganto"
        "catalog_item" -> "mobile.library.type.catalog"
        else -> "mobile.library.type.generic"
    }

    fun defaultOerProvider(providers: List<String>): String? {
        val preferred = listOf("oer_commons", "openstax", "merlot")
        return preferred.firstOrNull { it in providers } ?: providers.firstOrNull()
    }

    fun oerProviderLabel(provider: String): String = when (provider) {
        "oer_commons" -> "OER Commons"
        "openstax" -> "OpenStax"
        "merlot" -> "MERLOT"
        else -> provider
    }

    private fun normalizedUrl(raw: String?): String? {
        val trimmed = raw?.trim().orEmpty()
        return trimmed.ifEmpty { null }
    }

    private fun encodePath(value: String): String =
        java.net.URLEncoder.encode(value, "UTF-8").replace("+", "%20")
}