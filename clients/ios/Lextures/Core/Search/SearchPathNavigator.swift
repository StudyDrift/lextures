import Foundation

enum SearchNavigationTarget: Equatable {
    case shellTab(ShellTab)
    case deepLink(DeepLinkDestination)
    case more(MoreDestination)
}

/// Maps web-style search result paths to native shell navigation.
enum SearchPathNavigator {
    static func resolve(_ rawPath: String) -> SearchNavigationTarget? {
        let trimmed = rawPath.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return nil }

        if trimmed.hasPrefix("lextures://shell/") {
            let segment = String(trimmed.dropFirst("lextures://shell/".count)).lowercased()
            switch segment {
            case "teach": return .shellTab(.teach)
            case "children": return .shellTab(.children)
            case "calendar": return .shellTab(.calendar)
            default: break
            }
        }

        let path = trimmed.split(separator: "?").first.map(String.init) ?? trimmed
        let segments = path.split(separator: "/").map(String.init).filter { !$0.isEmpty }

        if segments.isEmpty || (segments.count == 1 && segments[0].isEmpty) {
            return .shellTab(.home)
        }

        switch segments[0].lowercased() {
        case "courses":
            if segments.count == 1 { return .shellTab(.courses) }
            return .deepLink(DeepLinkRouter.resolve(path))
        case "inbox":
            return .deepLink(.inbox)
        case "notebooks":
            return .shellTab(.notebooks)
        case "calendar":
            return .shellTab(.calendar)
        case "parent":
            return .shellTab(.children)
        case "todos":
            return .more(.planner)
        case "portfolios":
            return .more(.portfolio)
        case "catalog":
            return .more(.catalog)
        case "paths":
            return .more(.paths)
        case "library":
            return .more(.library)
        case "reading":
            return .more(.reading)
        case "credentials":
            return .more(.credentials)
        case "advising":
            return .more(.advising)
        case "settings":
            return .shellTab(.profile)
        default:
            let deep = DeepLinkRouter.resolve(path)
            if case .home = deep, segments.first?.lowercased() != "courses" {
                return nil
            }
            return .deepLink(deep)
        }
    }
}