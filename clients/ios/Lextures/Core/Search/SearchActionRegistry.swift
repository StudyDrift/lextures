import Foundation

/// Client-side command palette rows from the M0.5 destination registry.
enum SearchActionRegistry {
    static func buildActions(
        context: MobileRoleContext,
        platform: MobilePlatformFeatures
    ) -> [SearchListItem] {
        var items: [SearchListItem] = []

        for tab in MobileDestinations.shellTabs(context: context) {
            let path = shellTabPath(tab)
            items.append(
                SearchListItem(
                    id: "action:tab:\(tab.rawValue)",
                    group: .action,
                    title: L.format("mobile.search.action.openTab", tab.label),
                    subtitle: L.text("mobile.search.action.jumpTo"),
                    path: path,
                    haystack: "open \(tab.label.lowercased()) \(tab.rawValue) goto jump action".lowercased()
                )
            )
        }

        for destination in MobileDestinations.moreDestinations(context: context, platform: platform) {
            let path = moreDestinationPath(destination)
            items.append(
                SearchListItem(
                    id: "action:more:\(destination.rawValue)",
                    group: .action,
                    title: L.format("mobile.search.action.openDestination", destination.label),
                    subtitle: L.text("mobile.search.action.moreHub"),
                    path: path,
                    haystack: "open \(destination.label.lowercased()) \(destination.rawValue) goto jump action more".lowercased()
                )
            )
        }

        items.append(
            SearchListItem(
                id: "action:calendar",
                group: .action,
                title: L.text("mobile.search.action.openCalendar"),
                subtitle: L.text("mobile.search.action.calendarSubtitle"),
                path: "/calendar",
                haystack: "calendar schedule open calendar goto jump action"
            )
        )

        if context == .teaching {
            items.append(
                SearchListItem(
                    id: "action:attendance",
                    group: .action,
                    title: L.text("mobile.search.action.takeAttendance"),
                    subtitle: L.text("mobile.search.action.attendanceSubtitle"),
                    path: "/courses",
                    haystack: "take attendance roll call goto jump action teach"
                )
            )
        }

        if context == .learning {
            items.append(
                SearchListItem(
                    id: "action:grades",
                    group: .action,
                    title: L.text("mobile.search.action.myGrades"),
                    subtitle: L.text("mobile.search.action.gradesSubtitle"),
                    path: "/courses",
                    haystack: "my grades scores transcript goto jump action"
                )
            )
        }

        return items
    }

    static func matchActions(query: String, actions: [SearchListItem], limit: Int = 5) -> [SearchListItem] {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !trimmed.isEmpty else { return [] }
        let tokens = trimmed.split(whereSeparator: \.isWhitespace).map(String.init)
        return actions
            .filter { item in
                let hay = item.haystack
                return tokens.allSatisfy { hay.contains($0) }
            }
            .prefix(limit)
            .map { $0 }
    }

    private static func shellTabPath(_ tab: ShellTab) -> String {
        switch tab {
        case .home: return "/"
        case .courses: return "/courses"
        case .notebooks: return "/notebooks"
        case .inbox: return "/inbox"
        case .profile: return "/settings/account"
        case .teach: return "lextures://shell/teach"
        case .children: return "/parent"
        case .calendar: return "/calendar"
        }
    }

    private static func moreDestinationPath(_ destination: MoreDestination) -> String {
        switch destination {
        case .calendar: return "/calendar"
        case .planner: return "/todos"
        case .catalog: return "/catalog"
        case .paths: return "/paths"
        case .library: return "/library"
        case .reading: return "/reading"
        case .portfolio: return "/portfolios"
        case .credentials: return "/credentials"
        case .advising: return "/advising"
        case .settings: return "/settings/account"
        case .askAi: return "/ask-ai"
        }
    }
}