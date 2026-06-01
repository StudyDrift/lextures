import Foundation

/// Runtime API configuration. Debug builds read `API_BASE_URL` from `Config/Development.xcconfig` (see `clients/scripts/setup-mobile-dev.sh`).
enum AppConfiguration {
    private static let defaultAPIBase = "http://127.0.0.1:8080"

    static var apiBaseURL: URL {
        if let env = ProcessInfo.processInfo.environment["API_BASE_URL"],
           let url = URL(string: env), !env.isEmpty {
            return url
        }
        if let plist = Bundle.main.object(forInfoDictionaryKey: "API_BASE_URL") as? String,
           !plist.isEmpty,
           !plist.hasPrefix("$("),
           let url = URL(string: plist) {
            return url
        }
        return URL(string: defaultAPIBase)!
    }

    static func apiURL(path: String) -> URL {
        let base = apiBaseURL.absoluteString.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        let normalized = path.hasPrefix("/") ? path : "/\(path)"
        return URL(string: base + normalized)!
    }
}
