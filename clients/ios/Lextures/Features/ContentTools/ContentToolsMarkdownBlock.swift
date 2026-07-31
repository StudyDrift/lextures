import SwiftUI

extension View {
    /// Injects Content Tool page context for descendant `lex-tool` fences.
    func contentToolsPageHost(
        courseCode: String,
        itemId: String,
        contentToolsEnabled: Bool
    ) -> some View {
        ContentToolsPageHostModifier(
            courseCode: courseCode,
            itemId: itemId,
            contentToolsEnabled: contentToolsEnabled,
            content: self
        )
    }
}

private struct ContentToolsPageHostModifier<Content: View>: View {
    @Environment(AppShellModel.self) private var shell
    let courseCode: String
    let itemId: String
    let contentToolsEnabled: Bool
    let content: Content

    var body: some View {
        ContentToolsPageProvider(
            context: ContentToolsPageContext(
                courseCode: courseCode,
                itemId: itemId,
                contentToolsEnabled: contentToolsEnabled,
                mobileContentToolsEnabled: shell.platformFeatures.ffMobileContentTools,
                mobileContentToolsSandboxEnabled: shell.platformFeatures.ffMobileContentToolsSandbox
            )
        ) {
            content
        }
    }
}
