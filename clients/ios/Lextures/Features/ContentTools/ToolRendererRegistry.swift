import SwiftUI

/// Props passed to a native Content Tool renderer (CT.M3 runtime contract v1).
struct ContentToolRendererProps {
    var instanceId: String
    var toolId: String
    var config: JSONValue
    var state: JSONValue
    var status: String
    var readOnly: Bool
    var save: ([String: JSONValue]) -> Void
    var submit: ([String: JSONValue]) -> Void
    var runAction: (String, JSONValue) async throws -> JSONValue?
    var announce: (String, Bool) -> Void
}

enum ToolRendererRegistry {
    static func isRegistered(_ toolId: String) -> Bool {
        registeredIds().contains(toolId)
    }

    static func registeredIds() -> Set<String> {
        var ids: Set<String> = ["noop_probe"]
        ids.formUnion(ContentToolPack1Logic.allowlistedToolIds())
        ids.formUnion(ContentToolPack2Logic.allowlistedToolIds())
        ids.formUnion(ContentToolPack3Logic.allowlistedToolIds())
        ids.formUnion(ContentToolPack4Logic.allowlistedToolIds())
        return ids
    }

    @ViewBuilder
    static func view(for toolId: String, props: ContentToolRendererProps) -> some View {
        switch toolId {
        case "noop_probe":
            NoopProbeRendererView(props: props)
        case "inline_questions":
            InlineQuestionsToolView(props: props)
        case "predict_reveal":
            PredictRevealToolView(props: props)
        case "class_pulse":
            ClassPulseToolView(props: props)
        case "flashcards":
            FlashcardsToolView(props: props)
        case "ask_questions":
            AskQuestionsToolView(props: props)
        case "explain_it_back":
            ExplainItBackToolView(props: props)
        case "inline_discussion":
            InlineDiscussionToolView(props: props)
        case "sort_sequence":
            SortSequenceToolView(props: props)
        case "highlight_annotate":
            HighlightAnnotateToolView(props: props)
        case "diagram_hotspot":
            DiagramHotspotToolView(props: props)
        case "media_checkpoints":
            MediaCheckpointsToolView(props: props)
        case "worked_example":
            WorkedExampleToolView(props: props)
        case "parameter_explorer":
            ParameterExplorerToolView(props: props)
        default:
            EmptyView()
        }
    }
}
