import Foundation

enum LibraryAccessState: Equatable {
    case ready(url: String)
    case gated(messageKey: String)
    case requiresWeb(path: String)
}

enum LibraryBrowseTab: String, CaseIterable, Identifiable {
    case library
    case oer

    var id: String { rawValue }
}

enum LibraryResourceLogic {
    static func libraryItems(from items: [CourseStructureItem]) -> [CourseStructureItem] {
        items.filter { $0.kind == "library_resource" }
    }

    static func hasLibraryResources(in items: [CourseStructureItem]) -> Bool {
        !libraryItems(from: items).isEmpty
    }

    static func accessEventPath(courseCode: String, itemId: String) -> String {
        "/api/v1/courses/\(LMSAPI.encodePath(courseCode))/library-resources/\(LMSAPI.encodePath(itemId))/access"
    }

    static func resourceDetailPath(courseCode: String, itemId: String) -> String {
        "/api/v1/courses/\(LMSAPI.encodePath(courseCode))/library-resources/\(LMSAPI.encodePath(itemId))"
    }

    static func webModulePath(courseCode: String, itemId: String) -> String {
        "/courses/\(LMSAPI.encodePath(courseCode))/modules/library-resource/\(LMSAPI.encodePath(itemId))"
    }

    static func resolveAccess(payload: LibraryResourcePayload) -> LibraryAccessState {
        if let url = normalizedURL(payload.ezproxyUrl) {
            return .ready(url: url)
        }
        if let url = normalizedURL(payload.metadata?.ezproxyUrl) {
            return .ready(url: url)
        }
        if payload.resourceType == "leganto_list" {
            return .gated(messageKey: "mobile.library.legantoGated")
        }
        if payload.resourceType == "catalog_item" {
            return .gated(messageKey: "mobile.library.catalogGated")
        }
        return .gated(messageKey: "mobile.library.noAccess")
    }

    static func resourceTypeLabel(_ resourceType: String) -> String {
        switch resourceType {
        case "leganto_list": return L.text("mobile.library.type.leganto")
        case "catalog_item": return L.text("mobile.library.type.catalog")
        default: return L.text("mobile.library.type.generic")
        }
    }

    static func subtitle(for item: CourseStructureItem, payload: LibraryResourcePayload?) -> String {
        if let payload {
            return resourceTypeLabel(payload.resourceType)
        }
        return L.text("mobile.library.type.generic")
    }

    static func parseLibrarySearchResults(from data: Data) throws -> [LibraryCatalogResult] {
        let response = try JSONDecoder().decode(LibrarySearchResponse.self, from: data)
        return response.results
    }

    static func parseOERSearchResults(from data: Data) throws -> OERSearchResponse {
        try JSONDecoder().decode(OERSearchResponse.self, from: data)
    }

    static func defaultOERProvider(from providers: [String]) -> String? {
        let preferred = ["oer_commons", "openstax", "merlot"]
        for id in preferred where providers.contains(id) {
            return id
        }
        return providers.first
    }

    static func oerProviderLabel(_ provider: String) -> String {
        switch provider {
        case "oer_commons": return "OER Commons"
        case "openstax": return "OpenStax"
        case "merlot": return "MERLOT"
        default: return provider
        }
    }

    private static func normalizedURL(_ raw: String?) -> String? {
        guard let raw else { return nil }
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}