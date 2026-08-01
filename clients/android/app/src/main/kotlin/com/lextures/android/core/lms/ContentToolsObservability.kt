package com.lextures.android.core.lms

/** CT.M9 FR-17/18 — content-free mobile content-tools counters. */
object ContentToolsObservability {
    private val counters = mutableMapOf<String, Int>()
    private val lock = Any()
    const val PLATFORM = "android"

    fun record(event: String, toolId: String? = null, attributes: Map<String, String> = emptyMap()) {
        val attrs = attributes.toMutableMap()
        attrs["platform"] = PLATFORM
        if (!toolId.isNullOrBlank()) attrs["tool_id"] = toolId
        attrs["event"] = event
        if (!ContentToolGovernanceLogic.telemetryAttributesAreContentFree(attrs)) {
            // Drop payloads that would violate FR-18 rather than emit learner content.
            return
        }
        synchronized(lock) {
            val key = if (attrs.isEmpty()) {
                event
            } else {
                event + "|" + attrs.keys.sorted().joinToString(",") { "$it=${attrs[it]}" }
            }
            counters[key] = (counters[key] ?: 0) + 1
        }
    }

    fun count(event: String): Int = synchronized(lock) {
        counters.filter { it.key == event || it.key.startsWith("$event|") }.values.sum()
    }

    fun resetForTests() {
        synchronized(lock) { counters.clear() }
    }
}
