package com.lextures.android.features.contenttools

import androidx.compose.runtime.Composable
import com.lextures.android.core.lms.ToolInstance
import com.lextures.android.core.lms.ToolStateEnvelope
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject

/** Props passed to a native Content Tool renderer (CT.M3 runtime contract v1). */
data class ContentToolRendererProps(
    val instanceId: String,
    val toolId: String,
    val config: JsonElement,
    val state: JsonElement,
    val status: String,
    val readOnly: Boolean,
    val save: (Map<String, JsonElement>) -> Unit,
    val submit: (Map<String, JsonElement>) -> Unit,
    val runAction: suspend (name: String, input: JsonObject) -> JsonElement?,
    val announce: (String, Boolean) -> Unit,
)

typealias ContentToolRenderer = @Composable (ContentToolRendererProps) -> Unit

object ToolRegistry {
    private val renderers: MutableMap<String, ContentToolRenderer> = mutableMapOf(
        "noop_probe" to { props -> NoopProbeRenderer(props) },
    )

    fun isRegistered(toolId: String): Boolean = toolId in renderers

    fun resolve(toolId: String): ContentToolRenderer? = renderers[toolId]

    fun register(toolId: String, renderer: ContentToolRenderer) {
        renderers[toolId] = renderer
    }

    fun registeredIds(): Set<String> = renderers.keys.toSet()
}

fun ToolInstance.initialEnvelope(): ToolStateEnvelope =
    state ?: com.lextures.android.core.lms.emptyToolState(id)
