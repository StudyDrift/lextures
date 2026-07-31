package com.lextures.android.features.contenttools

import androidx.compose.runtime.Composable
import com.lextures.android.core.lms.ContentToolPack1Logic
import com.lextures.android.core.lms.ContentToolPack2Logic
import com.lextures.android.core.lms.ContentToolPack3Logic
import com.lextures.android.core.lms.ToolInstance
import com.lextures.android.core.lms.ToolStateEnvelope
import com.lextures.android.features.contenttools.tools.AskQuestionsTool
import com.lextures.android.features.contenttools.tools.ClassPulseTool
import com.lextures.android.features.contenttools.tools.DiagramHotspotTool
import com.lextures.android.features.contenttools.tools.ExplainItBackTool
import com.lextures.android.features.contenttools.tools.FlashcardsTool
import com.lextures.android.features.contenttools.tools.HighlightAnnotateTool
import com.lextures.android.features.contenttools.tools.InlineDiscussionTool
import com.lextures.android.features.contenttools.tools.InlineQuestionsTool
import com.lextures.android.features.contenttools.tools.PredictRevealTool
import com.lextures.android.features.contenttools.tools.SortSequenceTool
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
        "inline_questions" to { props -> InlineQuestionsTool(props) },
        "predict_reveal" to { props -> PredictRevealTool(props) },
        "class_pulse" to { props -> ClassPulseTool(props) },
        "flashcards" to { props -> FlashcardsTool(props) },
        "ask_questions" to { props -> AskQuestionsTool(props) },
        "explain_it_back" to { props -> ExplainItBackTool(props) },
        "inline_discussion" to { props -> InlineDiscussionTool(props) },
        "sort_sequence" to { props -> SortSequenceTool(props) },
        "highlight_annotate" to { props -> HighlightAnnotateTool(props) },
        "diagram_hotspot" to { props -> DiagramHotspotTool(props) },
    )

    fun isRegistered(toolId: String): Boolean = toolId in registeredIds()

    fun resolve(toolId: String): ContentToolRenderer? =
        if (isRegistered(toolId)) renderers[toolId] else null

    fun register(toolId: String, renderer: ContentToolRenderer) {
        renderers[toolId] = renderer
    }

    fun registeredIds(): Set<String> {
        val allowlisted =
            ContentToolPack1Logic.allowlistedToolIds() +
                ContentToolPack2Logic.allowlistedToolIds() +
                ContentToolPack3Logic.allowlistedToolIds()
        return renderers.keys.filter { id ->
            id == "noop_probe" || id in allowlisted
        }.toSet()
    }
}

fun ToolInstance.initialEnvelope(): ToolStateEnvelope =
    state ?: com.lextures.android.core.lms.emptyToolState(id)
