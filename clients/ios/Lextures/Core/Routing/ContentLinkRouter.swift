import SwiftUI
import UIKit

/// Routes in-app markdown / content-page links to native screens or the web app (IC07).
enum ContentLinkRouter {
    static func openURLAction(shell: AppShellModel) -> OpenURLAction {
        OpenURLAction { url in
            handle(url: url, shell: shell)
        }
    }

    @discardableResult
    static func handle(url: URL, shell: AppShellModel) -> OpenURLAction.Result {
        guard let path = normalizedPath(for: url) else {
            return .systemAction
        }

        if path.hasPrefix("/settings/")
            || path.hasPrefix("/courses")
            || path.hasPrefix("/me/")
            || path == "/inbox"
            || path == "/review" {
            let destination = DeepLinkRouter.resolve(path)
            Task { @MainActor in
                shell.openDeepLink(destination)
            }
            return .handled
        }

        if path.hasPrefix("/privacy")
            || path.hasPrefix("/security")
            || path.hasPrefix("/accessibility") {
            UIApplication.shared.open(AppConfiguration.webURL(path: path))
            return .handled
        }

        return .systemAction
    }

    private static func normalizedPath(for url: URL) -> String? {
        if url.scheme?.lowercased() == "lextures" {
            var path = url.path
            if path.isEmpty { path = "/" }
            return path
        }
        if url.host == nil, url.path.hasPrefix("/") {
            return url.path
        }
        guard let host = url.host?.lowercased() else { return nil }
        if host == "lextures.com" || host.hasSuffix(".lextures.com") || host == "localhost" {
            var path = url.path
            if !path.hasPrefix("/") { path = "/\(path)" }
            return path
        }
        return nil
    }
}