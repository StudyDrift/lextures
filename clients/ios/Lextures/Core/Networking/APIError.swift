import Foundation

enum APIError: LocalizedError {
    case invalidResponse
    case httpStatus(Int, message: String?)
    case decoding(Error)
    case transport(Error)

    var errorDescription: String? {
        switch self {
        case .invalidResponse:
            return "Unexpected response from the server."
        case let .httpStatus(code, message):
            if let message, !message.isEmpty {
                return message
            }
            return "Request failed (HTTP \(code))."
        case let .decoding(error):
            return error.localizedDescription
        case let .transport(error):
            return error.localizedDescription
        }
    }
}

/// Mirrors web `readApiErrorMessage` for common API error shapes.
func parseAPIErrorMessage(from data: Data) -> String? {
    guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
        return nil
    }
    if let error = json["error"] as? [String: Any],
       let message = error["message"] as? String,
       !message.isEmpty {
        return message
    }
    if let message = json["message"] as? String, !message.isEmpty {
        return message
    }
    if let error = json["error"] as? String, !error.isEmpty {
        return error
    }
    if let detail = json["detail"] as? String, !detail.isEmpty {
        return detail
    }
    return nil
}
