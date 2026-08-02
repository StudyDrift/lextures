import SwiftUI
import UIKit

/// Routes in-app markdown / content-page links via `LinkOpener` (MB.1 / IC07).
/// Main-actor isolated to match `LinkOpener` / `AppShellModel` (presentation + deep links).
@MainActor
enum ContentLinkRouter {
    static func openURLAction(shell: AppShellModel) -> OpenURLAction {
        LinkOpener.openURLAction(shell: shell, source: "content")
    }

    @discardableResult
    static func handle(url: URL, shell: AppShellModel) -> OpenURLAction.Result {
        _ = LinkOpener.open(url, shell: shell, source: "content")
        return .handled
    }
}
