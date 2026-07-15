package com.lextures.android.core.config

/**
 * Validates school codes the same way as the marketing site (`www/src/lib/school-code.ts`).
 * Special case: `local` routes the mobile app to the local API for development.
 */
object SchoolCodeLogic {
    const val SELF_LEARNER_API_BASE = "https://self.lextures.com"
    const val LOCAL_API_BASE = "http://127.0.0.1:8080"
    const val TENANT_HOST_SUFFIX = "lextures.com"

    private val pattern = Regex("^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$")

    private val reserved = setOf(
        "admin",
        "api",
        "app",
        "default",
        "demo",
        "login",
        "magic-link",
        "mfa",
        "self",
        "signup",
        "www",
    )

    fun normalize(raw: String): String = raw.trim().lowercase()

    /**
     * Returns an Android string resource name for the validation error, or null if valid.
     * `local` is always accepted (dev shortcut).
     */
    fun errorKey(code: String): String? {
        val normalized = normalize(code)
        if (normalized.isEmpty()) return "auth_getStarted_schoolCodeErrorEmpty"
        if (normalized == "local") return null
        if (normalized.length < 2) return "auth_getStarted_schoolCodeErrorLengthMin"
        if (normalized.length > 32) return "auth_getStarted_schoolCodeErrorLengthMax"
        if (!pattern.matches(normalized)) return "auth_getStarted_schoolCodeErrorFormat"
        if (normalized in reserved) return "auth_getStarted_schoolCodeErrorReserved"
        return null
    }

    fun isValid(code: String): Boolean = errorKey(code) == null

    fun apiBaseUrl(schoolCode: String): String {
        val normalized = normalize(schoolCode)
        if (normalized == "local") return LOCAL_API_BASE
        return "https://$normalized.$TENANT_HOST_SUFFIX"
    }

    fun previewHost(schoolCode: String): String {
        val normalized = normalize(schoolCode)
        if (normalized.isEmpty()) return "your-school.$TENANT_HOST_SUFFIX"
        if (normalized == "local") return "127.0.0.1:8080"
        return "$normalized.$TENANT_HOST_SUFFIX"
    }
}
