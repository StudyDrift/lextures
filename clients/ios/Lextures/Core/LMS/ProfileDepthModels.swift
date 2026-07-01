import Foundation

// MARK: - Custom profile fields

struct ProfileFieldDefinition: Decodable, Identifiable, Hashable {
    var id: String
    var key: String
    var label: String
    var fieldType: String
    var selectOptions: [String]?
    var isRequired: Bool
}

struct ProfileFieldsResponse: Decodable {
    var fields: [ProfileFieldDefinition]
    var values: [String: JSONValue]
}

struct ProfileFieldsPatch: Encodable {
    var values: [String: JSONValue]
}

struct ProfileFieldsValuesResponse: Decodable {
    var values: [String: JSONValue]
}

// MARK: - Demographics (self-reported)

struct StudentDemographics: Codable, Equatable {
    var studentId: String?
    var freeLunch: Bool?
    var reducedLunch: Bool?
    var ellStatus: Bool?
    var disabilityStatus: Bool?
    var raceEthnicityCode: String?
    var homelessIndicator: Bool?
    var migrantIndicator: Bool?
    var dataSource: String?
    var updatedAt: String?
}

struct StudentDemographicsPatch: Encodable {
    var freeLunch: Bool?
    var reducedLunch: Bool?
    var ellStatus: Bool?
    var disabilityStatus: Bool?
    var raceEthnicityCode: String?
    var homelessIndicator: Bool?
    var migrantIndicator: Bool?
}

// MARK: - Research consent

enum ConsentDecision: String, Codable, CaseIterable {
    case granted, declined, withdrawn
}

struct ConsentStudy: Decodable, Identifiable, Hashable {
    var id: String
    var title: String
    var irbProtocol: String
    var consentText: String
    var dataUseDescription: String
    var status: String
}

struct ConsentStudiesResponse: Decodable {
    var studies: [ConsentStudy]
}

struct ConsentHistoryEntry: Decodable, Identifiable, Hashable {
    var id: String
    var studyId: String
    var studyTitle: String?
    var decision: ConsentDecision
    var createdAt: String

    var identifiableId: String { studyId }
}

struct ConsentHistoryResponse: Decodable {
    var history: [ConsentHistoryEntry]
}

struct ConsentRespondBody: Encodable {
    var decision: ConsentDecision
}

struct ConsentRecordResponse: Decodable {
    var record: ConsentRecord?
}

struct ConsentRecord: Decodable {
    var id: String
    var studyId: String
    var decision: ConsentDecision
}

/// Flexible JSON value for custom field payloads and review question payloads.
enum JSONValue: Codable, Hashable {
    case string(String)
    case bool(Bool)
    case number(Double)
    case object([String: JSONValue])
    case array([JSONValue])
    case null

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if container.decodeNil() {
            self = .null
        } else if let boolValue = try? container.decode(Bool.self) {
            self = .bool(boolValue)
        } else if let numberValue = try? container.decode(Double.self) {
            self = .number(numberValue)
        } else if let stringValue = try? container.decode(String.self) {
            self = .string(stringValue)
        } else if let objectValue = try? container.decode([String: JSONValue].self) {
            self = .object(objectValue)
        } else if let arrayValue = try? container.decode([JSONValue].self) {
            self = .array(arrayValue)
        } else {
            self = .null
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case .string(let stringValue): try container.encode(stringValue)
        case .bool(let boolValue): try container.encode(boolValue)
        case .number(let numberValue): try container.encode(numberValue)
        case .object(let objectValue): try container.encode(objectValue)
        case .array(let arrayValue): try container.encode(arrayValue)
        case .null: try container.encodeNil()
        }
    }

    var isEmpty: Bool {
        switch self {
        case .null: return true
        case .string(let stringValue): return stringValue.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        case .array(let arrayValue): return arrayValue.isEmpty
        case .object(let objectValue): return objectValue.isEmpty
        case .bool, .number: return false
        }
    }
}