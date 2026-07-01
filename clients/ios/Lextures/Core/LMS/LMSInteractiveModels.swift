import Foundation

// MARK: - Interactive content (M3.3)

struct ModuleH5PPayload: Decodable {
    var packageId: String
    var itemId: String?
    var title: String
    var contentType: String?
    var extractStatus: String
    var assetsBaseUrl: String?
    var downloadUrl: String?
}

struct ModuleScormSco: Decodable {
    var id: String
    var identifier: String?
    var title: String?
    var launchHref: String?
}

struct ModuleScormPayload: Decodable {
    var packageId: String
    var itemId: String?
    var title: String
    var packageType: String?
    var extractStatus: String
    var assetsBaseUrl: String?
    var downloadUrl: String?
    var scos: [ModuleScormSco]
}

struct ScormLaunchResponse: Decodable {
    var registrationId: String
    var launchUrl: String?
    var renderUrl: String
    var initialCmi: [String: String]?

    enum CodingKeys: String, CodingKey {
        case registrationId, launchUrl, renderUrl, initialCmi
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        registrationId = try container.decodeIfPresent(String.self, forKey: .registrationId) ?? ""
        launchUrl = try container.decodeIfPresent(String.self, forKey: .launchUrl)
        renderUrl = try container.decodeIfPresent(String.self, forKey: .renderUrl) ?? ""
        initialCmi = try container.decodeIfPresent([String: String].self, forKey: .initialCmi)
    }
}

struct ModuleLtiLinkPayload: Decodable {
    var itemId: String
    var title: String
    var externalToolId: String?
    var externalToolName: String?
    var resourceLinkId: String?
    var lineItemUrl: String?
}

struct LtiEmbedTicketResponse: Decodable {
    var ticket: String
}

struct ModuleVibeActivityPayload: Decodable {
    var id: String
    var title: String
    var html: String?
    var published: Bool?
    var archived: Bool?
}

/// Human label + tint key for an attendance status string.
enum AttendanceStatusInfo {
    static func label(_ status: String) -> String {
        switch status {
        case "present": return "Present"
        case "absent": return "Absent"
        case "tardy": return "Tardy"
        case "excused": return "Excused"
        case "not_recorded": return "Not recorded"
        default: return status.replacingOccurrences(of: "_", with: " ").capitalized
        }
    }
}