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
        default:
            EmptyView()
        }
    }
}
