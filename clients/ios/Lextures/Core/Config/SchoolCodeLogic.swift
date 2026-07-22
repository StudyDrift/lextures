import Foundation

/// Validates school codes the same way as the marketing site (`www/src/lib/school-code.ts`).
/// Special case: `local` routes the mobile app to the local API for development.
enum SchoolCodeLogic {
    static let homeschoolAPIBase = "https://self.lextures.com"
    static let localAPIBase = "http://127.0.0.1:8080"
    static let tenantHostSuffix = "lextures.com"

    private static let pattern = makeRegex(#"^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$"#)

    private static func makeRegex(_ pattern: String) -> NSRegularExpression {
        guard let regex = try? NSRegularExpression(pattern: pattern) else {
            preconditionFailure("Invalid school-code regex: \(pattern)")
        }
        return regex
    }

    private static let reserved: Set<String> = [
        "admin",
        "api",
        "app",
        "default",
        "demo",
        "login",
        "magic-link",
        "mfa",
        "self",
        "signup",
        "www",
    ]

    static func normalize(_ raw: String) -> String {
        raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
    }

    /// Returns a localization key for the validation error, or `nil` if valid.
    /// `local` is always accepted (dev shortcut).
    static func errorKey(for code: String) -> String? {
        let normalized = normalize(code)
        if normalized.isEmpty {
            return "auth.getStarted.schoolCodeErrorEmpty"
        }
        if normalized == "local" {
            return nil
        }
        if normalized.count < 2 {
            return "auth.getStarted.schoolCodeErrorLengthMin"
        }
        if normalized.count > 32 {
            return "auth.getStarted.schoolCodeErrorLengthMax"
        }
        let range = NSRange(normalized.startIndex..<normalized.endIndex, in: normalized)
        if pattern.firstMatch(in: normalized, options: [], range: range) == nil {
            return "auth.getStarted.schoolCodeErrorFormat"
        }
        if reserved.contains(normalized) {
            return "auth.getStarted.schoolCodeErrorReserved"
        }
        return nil
    }

    static func isValid(_ code: String) -> Bool {
        errorKey(for: code) == nil
    }

    static func apiBaseURL(schoolCode: String) -> String {
        let normalized = normalize(schoolCode)
        if normalized == "local" {
            return localAPIBase
        }
        return "https://\(normalized).\(tenantHostSuffix)"
    }

    static func previewHost(schoolCode: String) -> String {
        let normalized = normalize(schoolCode)
        if normalized.isEmpty {
            return "your-school.\(tenantHostSuffix)"
        }
        if normalized == "local" {
            return "127.0.0.1:8080"
        }
        return "\(normalized).\(tenantHostSuffix)"
    }
}
