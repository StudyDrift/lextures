package com.lextures.android.features.contenttools.sandbox

import com.lextures.android.core.lms.ContentToolSandboxLogic

/** Caps live sandbox WebViews per screen (NFR: at most 3). Pure bookkeeping; UI owns teardown. */
object SandboxWebViewPool {
    private val alive = mutableListOf<String>()
    private var maxAlive: Int = ContentToolSandboxLogic.MAX_LIVE_WEBVIEWS

    fun configure(maxAlive: Int) {
        this.maxAlive = maxAlive
    }

    fun aliveCount(): Int = alive.size

    fun aliveIds(): List<String> = alive.toList()

    /** Registers a mount. Returns instance ids that must be torn down to stay within budget. */
    fun retain(instanceId: String): List<String> {
        alive.removeAll { it == instanceId }
        alive.add(instanceId)
        val evicted = mutableListOf<String>()
        while (ContentToolSandboxLogic.poolShouldEvict(alive.size, maxAlive)) {
            val victim = alive.removeAt(0)
            if (victim != instanceId) evicted.add(victim)
        }
        return evicted
    }

    fun release(instanceId: String) {
        alive.removeAll { it == instanceId }
    }

    fun reset() {
        alive.clear()
    }
}
