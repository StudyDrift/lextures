import Foundation

// MARK: - Account settings (editable profile)

/// GET/PATCH `/api/v1/settings/account` — the server-backed, editable profile.
struct AccountProfile: Codable {
    var email: String
    var displayName: String?
    var firstName: String?
    var lastName: String?
    var avatarUrl: String?
    var phoneNumber: String?

    enum CodingKeys: String, CodingKey {
        case email, displayName, firstName, lastName, avatarUrl, phoneNumber
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        email = try container.decodeIfPresent(String.self, forKey: .email) ?? ""
        displayName = try container.decodeIfPresent(String.self, forKey: .displayName)
        firstName = try container.decodeIfPresent(String.self, forKey: .firstName)
        lastName = try container.decodeIfPresent(String.self, forKey: .lastName)
        avatarUrl = try container.decodeIfPresent(String.self, forKey: .avatarUrl)
        phoneNumber = try container.decodeIfPresent(String.self, forKey: .phoneNumber)
    }
}

/// Body for PATCH `/api/v1/settings/account`. Only the editable fields are sent.
struct AccountProfilePatch: Encodable {
    var firstName: String?
    var lastName: String?
    var avatarUrl: String?
    var phoneNumber: String?
}

extension AccountProfile {
    /// First/last name for forms — falls back to splitting `displayName` (parity with web).
    var resolvedNameFields: (firstName: String, lastName: String) {
        let first = firstName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let last = lastName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !first.isEmpty || !last.isEmpty {
            return (first, last)
        }
        let display = displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !display.isEmpty else { return ("", "") }
        let parts = display.split(whereSeparator: \.isWhitespace).map(String.init).filter { !$0.isEmpty }
        guard !parts.isEmpty else { return ("", "") }
        if parts.count == 1 { return (parts[0], "") }
        return (parts[0], parts.dropFirst().joined(separator: " "))
    }

    /// Display label for profile headers — combined name, then displayName, then email.
    var resolvedDisplayName: String {
        let fields = resolvedNameFields
        let combined = [fields.firstName, fields.lastName]
            .filter { !$0.isEmpty }
            .joined(separator: " ")
        if !combined.isEmpty { return combined }
        if let display = displayName?.trimmingCharacters(in: .whitespacesAndNewlines), !display.isEmpty {
            return display
        }
        return email
    }

    /// Two-letter initials derived from the resolved display name.
    var resolvedInitials: String {
        let parts = resolvedDisplayName
            .split(whereSeparator: \.isWhitespace)
            .map(String.init)
            .filter { !$0.isEmpty }
        if parts.count >= 2,
           let first = parts.first?.first,
           let last = parts.last?.first {
            return String([first, last]).uppercased()
        }
        if let only = parts.first?.first {
            return String(only).uppercased()
        }
        return String(email.prefix(1)).uppercased()
    }
}

// MARK: - My accommodations

/// GET `/api/v1/me/accommodations` — the student's currently active supports.
struct MyAccommodationsResponse: Decodable {
    var accommodations: [MyAccommodation]
}

struct MyAccommodation: Decodable, Identifiable {
    var courseCode: String?
    var hasExtendedTime: Bool
    var hasExtraAttempts: Bool
    var hintsAlwaysAvailable: Bool
    var reducedDistractionRecommended: Bool
    var speechToTextEnabled: Bool
    var ttsEnabled: Bool
    var dyslexiaDisplayEnabled: Bool
    var highContrastEnabled: Bool
    var reducedMotionEnabled: Bool
    var separateSetting: Bool
    var effectiveFrom: String?
    var effectiveUntil: String?

    /// Stable identity for SwiftUI lists (one row per course scope).
    var id: String { courseCode ?? "__all__" }

    enum CodingKeys: String, CodingKey {
        case courseCode
        case hasExtendedTime
        case hasExtraAttempts
        case hintsAlwaysAvailable
        case reducedDistractionRecommended
        case speechToTextEnabled
        case ttsEnabled
        case dyslexiaDisplayEnabled
        case highContrastEnabled
        case reducedMotionEnabled
        case separateSetting
        case effectiveFrom
        case effectiveUntil
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        courseCode = try container.decodeIfPresent(String.self, forKey: .courseCode)
        hasExtendedTime = try container.decodeIfPresent(Bool.self, forKey: .hasExtendedTime) ?? false
        hasExtraAttempts = try container.decodeIfPresent(Bool.self, forKey: .hasExtraAttempts) ?? false
        hintsAlwaysAvailable = try container.decodeIfPresent(Bool.self, forKey: .hintsAlwaysAvailable) ?? false
        reducedDistractionRecommended = try container.decodeIfPresent(Bool.self, forKey: .reducedDistractionRecommended) ?? false
        speechToTextEnabled = try container.decodeIfPresent(Bool.self, forKey: .speechToTextEnabled) ?? false
        ttsEnabled = try container.decodeIfPresent(Bool.self, forKey: .ttsEnabled) ?? false
        dyslexiaDisplayEnabled = try container.decodeIfPresent(Bool.self, forKey: .dyslexiaDisplayEnabled) ?? false
        highContrastEnabled = try container.decodeIfPresent(Bool.self, forKey: .highContrastEnabled) ?? false
        reducedMotionEnabled = try container.decodeIfPresent(Bool.self, forKey: .reducedMotionEnabled) ?? false
        separateSetting = try container.decodeIfPresent(Bool.self, forKey: .separateSetting) ?? false
        effectiveFrom = try container.decodeIfPresent(String.self, forKey: .effectiveFrom)
        effectiveUntil = try container.decodeIfPresent(String.self, forKey: .effectiveUntil)
    }

    /// True when this entry carries no active supports (defensive — server filters these out).
    var isEmpty: Bool {
        !hasExtendedTime && !hasExtraAttempts && !hintsAlwaysAvailable
            && !reducedDistractionRecommended && !speechToTextEnabled && !ttsEnabled
            && !dyslexiaDisplayEnabled && !highContrastEnabled && !reducedMotionEnabled
            && !separateSetting
    }
}
